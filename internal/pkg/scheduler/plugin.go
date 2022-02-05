/*
Copyright 2021 Ciena Corporation.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scheduler

import (
	"context"
	"fmt"

	"github.com/ciena/outbound/internal/pkg/client"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	// Name of the scheduling plugin.
	Name              = "Planner"
	preFilterStateKey = "PreFilter" + Name
)

// PodSetPlanner instance state for the the policy scheduling
// plugin.
type PodSetPlanner struct {
	handle         framework.Handle
	log            logr.Logger
	options        *PodSetPlannerOptions
	plannerService *PlannerService
	plannerClient  *client.SchedulePlannerClient
	triggerClient  *client.ScheduleTriggerClient
}

type preFilterState struct {
	node string
}

var (
	_ framework.PreFilterPlugin  = &PodSetPlanner{}
	_ framework.ScorePlugin      = &PodSetPlanner{}
	_ framework.PostFilterPlugin = &PodSetPlanner{}
)

// New create a new framework plugin intance.
// nolint:ireturn
func New(
	obj runtime.Object, handle framework.Handle) (framework.Plugin, error,
) {
	var log logr.Logger

	var config *PodSetPlannerOptions

	defaultConfig := DefaultPodSetPlannerConfig()

	if obj != nil {
		//nolint: forcetypeassert
		pluginConfig := obj.(*PodSetPlannerArgs)
		config = parsePluginConfig(pluginConfig, defaultConfig)
	} else {
		config = defaultConfig
	}

	if config.Debug {
		zapLog, err := zap.NewDevelopment()
		if err != nil {
			return nil, err
		}

		log = zapr.NewLogger(zapLog)
	} else {
		zapLog, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}

		log = zapr.NewLogger(zapLog)
	}

	pluginLogger := log.WithName("scheduling-plugin")

	plannerClient, err := client.NewSchedulePlannerClient(handle.KubeConfig(),
		pluginLogger.WithName("planner-client"))
	if err != nil {
		pluginLogger.Error(err, "error-initializing-planner-client")

		return nil, err
	}

	triggerClient, err := client.NewScheduleTriggerClient(handle.KubeConfig(),
		pluginLogger.WithName("trigger-client"))
	if err != nil {
		pluginLogger.Error(err, "error-initializing-trigger-client")

		return nil, err
	}

	plannerService := NewPlannerService(plannerClient, handle,
		pluginLogger.WithName("planner-service"),
		config.CallTimeout)

	schedulePlanner := &PodSetPlanner{
		handle:         handle,
		log:            pluginLogger,
		options:        config,
		plannerService: plannerService,
		plannerClient:  plannerClient,
		triggerClient:  triggerClient,
	}

	pluginLogger.V(1).Info("podset-scheduling-plugin-initialized")

	return schedulePlanner, nil
}

func getPreFilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	state, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to read cycle state: %w", err)
	}

	if state == nil {
		return nil, ErrNilAssignmentState
	}

	assignmentState, ok := state.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v convert to node assignment state error: %w",
			state, ErrInvalidAssignmentState)
	}

	return assignmentState, nil
}

// Clone isn't needed for our state data.
// nolint:ireturn
func (s *preFilterState) Clone() framework.StateData {
	return s
}

func (c *PodSetPlanner) createPreFilterState(
	ctx context.Context,
	pod *v1.Pod) (*preFilterState, *framework.Status) {
	allNodes, err := c.handle.SnapshotSharedLister().NodeInfos().List()
	if err != nil {
		return nil, framework.AsStatus(err)
	}

	eligibleNodes := make([]*v1.Node, len(allNodes))

	for i, nodeInfo := range allNodes {
		eligibleNodes[i] = nodeInfo.Node()
	}

	if len(eligibleNodes) == 0 {
		c.log.V(1).Info("pre-filter-no-nodes-eligible")

		return nil, framework.NewStatus(framework.Unschedulable)
	}

	node := eligibleNodes[0]

	return &preFilterState{node: node.Name}, framework.NewStatus(framework.Success)
}

// Name returns the name of the scheduler.
func (c *PodSetPlanner) Name() string {
	return Name
}

// PreFilter pre-filters the pods to be placed.
func (c *PodSetPlanner) PreFilter(
	parentCtx context.Context,
	state *framework.CycleState,
	pod *v1.Pod) *framework.Status {
	c.log.V(1).Info("prefilter", "pod", pod.Name)

	//nolint: gomnd
	ctx, cancel := context.WithTimeout(parentCtx, c.options.CallTimeout*2)
	assignmentState, status := c.createPreFilterState(ctx, pod)

	cancel()

	if !status.IsSuccess() {

		return status
	}

	c.log.V(1).Info("prefilter-state-assignment", "pod", pod.Name, "node", assignmentState.node)
	state.Write(preFilterStateKey, assignmentState)

	return status
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
// nolint:ireturn
func (c *PodSetPlanner) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Score scores the eligible nodes.
func (c *PodSetPlanner) Score(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	c.log.V(1).Info("score", "pod", pod.Name, "node", nodeName)

	assignmentState, err := getPreFilterState(state)
	if err != nil {
		return framework.MinNodeScore, framework.NewStatus(framework.Success)
	}

	if assignmentState.node != nodeName {
		return 1, framework.NewStatus(framework.Success)
	}

	c.log.V(1).Info("set-score", "score", framework.MaxNodeScore, "pod", pod.Name, "node", nodeName)

	return framework.MaxNodeScore, framework.NewStatus(framework.Success)
}

// ScoreExtensions calcuates scores for the extensions.
// nolint:ireturn
func (c *PodSetPlanner) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// PostFilter is called when no node can be assigned to the pod.
func (c *PodSetPlanner) PostFilter(ctx context.Context,
	state *framework.CycleState, pod *v1.Pod,
	filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	c.log.V(1).Info("post-filter", "pod", pod.Name)

	assignmentState, err := getPreFilterState(state)
	if err != nil {
		return nil, framework.AsStatus(err)
	}

	c.log.V(1).Info("post-filter", "nominated-node", assignmentState.node, "pod", pod.Name)

	return &framework.PostFilterResult{NominatedNodeName: assignmentState.node}, framework.NewStatus(framework.Success)
}

func (p *PodSetPlanner) findFit(parentCtx context.Context, pod *v1.Pod, eligibleNodes []string) (string, error) {
	podset := p.getPodSet(pod)
	if podset == "" {
		return "", ErrNoPodSetFound
	}

	ctx, cancel := context.WithTimeout(parentCtx, p.options.CallTimeout)

	trigger, err := p.triggerClient.Get(ctx, pod.Namespace, podset)
	cancel()

	if err != nil {
		p.log.V(1).Info("no-trigger-found", "pod", pod.Name, "podset", podset, "namespace", pod.Namespace)
		return "", err
	}

	// pod not assignable if trigger is not active
	if trigger.Spec.State != "Schedule" {
		p.log.V(1).Info("trigger-not-active", trigger.Name, "state", trigger.Spec.State, "podset", podset)

		return "", ErrPodNotAssignable
	}

	ctx, cancel = context.WithTimeout(parentCtx, p.options.CallTimeout)

	planStatus, planSpec, _ := p.plannerClient.CheckIfPodPresent(ctx,
		pod.Namespace, podset, pod.Name)
	cancel()

	// pod is in the planspec.
	// check if the node is in the eligible filtered list
	// if not, we hit the planner again to reallocate pod
	// the new eligible set

	if planStatus {

		for _, node := range eligibleNodes {

			if node == planSpec.Node {

				return planSpec.Node, nil
			}
		}

	}

	// get to the planner to place the pod
	planners, err := p.plannerService.Lookup(parentCtx, pod.Namespace, podset, eligibleNodes)
	if err != nil {
		p.log.V(1).Info("could-not-find-planner", "pod", pod.Name, "podset", podset)

		return "", err
	}

	p.log.V(1).Info("found-planner", "pod", pod.Name, "podset", podset, "planners", len(planners))
	assignments, err := planners.Invoke(parentCtx, trigger)
	if err != nil {
		return "", err
	}

	// planner assignment success. validate if this pod is in the assignment map
	selectedNode, ok := assignments[pod.Name]
	if !ok {
		return "", ErrNotFound
	}

	err = p.plannerService.CreateOrUpdate(parentCtx, pod, podset, selectedNode)
	if err != nil {
		return "", err
	}

	p.log.V(1).Info("scheduler-assignment", "pod", pod.Name, "node", selectedNode)

	return selectedNode, nil
}

func (p *PodSetPlanner) getPodSet(pod *v1.Pod) string {
	podset := ""

	for k, v := range pod.Labels {
		if k == "planner.ciena.io/pod-set" {
			podset = v
			break
		}
	}

	return podset
}
