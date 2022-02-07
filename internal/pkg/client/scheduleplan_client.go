/*
Copyright 2021 Ciena Corporation..

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

package client

import (
	"context"
	plannerv1alpha1 "github.com/ciena/outbound/pkg/apis/scheduleplanner/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	options "sigs.k8s.io/controller-runtime/pkg/client"
)

type SchedulePlannerClient struct {
	Client client.Client
	Log    logr.Logger
}

func NewPlannerClient(c client.Client, log logr.Logger) *SchedulePlannerClient {
	return &SchedulePlannerClient{Client: c, Log: log}
}

func (c *SchedulePlannerClient) List(ctx context.Context, namespace string, labels map[string]string) (*plannerv1alpha1.SchedulePlanList, error) {
	var planList plannerv1alpha1.SchedulePlanList

	err := c.Client.List(ctx, &planList,
		options.InNamespace(namespace),
		options.MatchingLabels(labels))

	if err != nil {
		c.Log.Error(err, "error-listing-planner", "namespace", namespace, "labels", labels)
		return nil, err
	}

	return &planList, nil
}

func (c *SchedulePlannerClient) Get(ctx context.Context, namespace, podset string) (*plannerv1alpha1.SchedulePlan, error) {
	var plan plannerv1alpha1.SchedulePlan

	name := getName(namespace, podset)

	err := c.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &plan)
	if err != nil {
		return nil, err
	}

	return &plan, nil
}

func (c *SchedulePlannerClient) CreateOrUpdate(ctx context.Context, namespace, podset string,
	assignments map[string]string) error {

	planRef, err := c.Get(ctx, namespace, podset)
	if err != nil {
		var newPlan []plannerv1alpha1.PlanSpec

		for pod, node := range assignments {
			newPlan = append(newPlan,
				plannerv1alpha1.PlanSpec{Pod: pod, Node: node})
		}

		name := getName(namespace, podset)
		plan := plannerv1alpha1.SchedulePlan{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
				Labels: map[string]string{
					"planner.ciena.io/pod-set": podset,
				},
			},
			Spec: plannerv1alpha1.SchedulePlanSpec{
				Plan: newPlan,
			},
		}

		err = c.Client.Create(ctx, &plan)
		if err != nil {
			c.Log.Error(err, "error-creating-schedule-plan", "podset", podset)
			return err
		}

		c.Log.V(1).Info("plan-create-success", "name", name, "podset", podset)

		return nil
	}

	//update the plan spec if pod and node assignment has changed
	var newPlan []plannerv1alpha1.PlanSpec
	var currentPlan []plannerv1alpha1.PlanSpec

	presenceMap := make(map[plannerv1alpha1.PlanSpec]struct{}, len(planRef.Spec.Plan))

	for _, planSpec := range planRef.Spec.Plan {
		presenceMap[planSpec] = struct{}{}
	}

	for pod, node := range assignments {
		planSpec := plannerv1alpha1.PlanSpec{Pod: pod, Node: node}

		if _, ok := presenceMap[planSpec]; !ok {
			newPlan = append(newPlan, planSpec)
		} else {
			currentPlan = append(currentPlan, planSpec)
		}
	}

	if len(newPlan) == 0 {
		c.Log.V(1).Info("no-changes-in-planner-assignments", "podset", podset, "namespace", namespace)

		return nil
	}

	planRef.Spec.Plan = append(currentPlan, newPlan...)

	err = c.Client.Update(ctx, planRef)
	if err != nil {
		c.Log.Error(err, "plan-crud-update-error", "name", planRef.Name,
			"podset", podset, "namespace", namespace)

		return err
	}

	c.Log.V(1).Info("plan-update-success", "name", planRef.Name, "podset", podset)

	return nil
}

func (c *SchedulePlannerClient) Delete(ctx context.Context, podName, namespace, podset string) (bool, error) {
	planRef, err := c.Get(ctx, namespace, podset)
	if err != nil {
		c.Log.Error(err, "no-planner-instance-found", "pod", podName, "podset", podset, "namespace", namespace)

		return false, err
	}

	var newPlan []plannerv1alpha1.PlanSpec
	var found bool = false

	for _, spec := range planRef.Spec.Plan {
		if spec.Pod == podName {
			found = true
		} else {
			newPlan = append(newPlan, spec)
		}
	}

	// nothing to update
	if !found {

		return false, nil
	}

	if len(newPlan) == 0 {
		//delete the plan
		plan := &plannerv1alpha1.SchedulePlan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      planRef.Name,
				Namespace: planRef.Namespace,
			},
		}

		err = c.Client.Delete(ctx, plan)
		if err != nil {
			c.Log.Error(err, "error-deleting-plan", "plan", planRef.Name, "podset", podset)

			return true, err
		}

		return false, nil
	} else {
		planRef.Spec.Plan = newPlan

		err = c.Client.Update(ctx, planRef)
		if err != nil {
			c.Log.Error(err, "error-updating-plan-spec", "pod", podName, "podset", podset)

			return true, err
		}

		return false, nil
	}
}

func (c *SchedulePlannerClient) CheckIfPodPresent(ctx context.Context, namespace, podset, podName string) (bool, *plannerv1alpha1.PlanSpec, error) {
	plan, err := c.Get(ctx, namespace, podset)
	if err != nil {
		return false, nil, err
	}

	for _, spec := range plan.Spec.Plan {

		if spec.Pod == podName {
			return true, &spec, nil
		}
	}

	return false, nil, nil
}
