package scheduler

import (
	"context"
	"fmt"
	"github.com/ciena/outbound/internal/pkg/client"
	planner "github.com/ciena/outbound/pkg/apis/planner"
	plannerv1alpha1 "github.com/ciena/outbound/pkg/apis/scheduleplanner/v1alpha1"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"math/rand"
	"sort"
	"time"
)

type Planner struct {
	Service       v1.Service
	Namespace     string
	Podset        string
	ScheduledPod  string
	EligibleNodes []string
	Log           logr.Logger
	CallTimeout   time.Duration
	DialOptions   []grpc.DialOption
}

type PlannerList []*Planner

type PlannerService struct {
	clnt        *client.SchedulePlannerClient
	handle      framework.Handle
	log         logr.Logger
	callTimeout time.Duration
}

type planReward struct {
	reward      int
	assignments map[string]string
}

func NewPlannerService(clnt *client.SchedulePlannerClient, handle framework.Handle,
	log logr.Logger, callTimeout time.Duration) *PlannerService {

	return &PlannerService{clnt: clnt, handle: handle, log: log, callTimeout: callTimeout}
}

func (s *PlannerService) CreateOrUpdate(parentCtx context.Context, pod *v1.Pod,
	podset string,
	assignments map[string]string) error {

	ctx, cancel := context.WithTimeout(parentCtx, s.callTimeout)
	defer cancel()

	err := s.clnt.CreateOrUpdate(ctx,
		pod.Namespace,
		podset,
		assignments)

	if err != nil {
		s.log.Error(err, "create-update-plan-failure", "pod", pod.Name)
		return err
	}

	return nil
}

func (s *PlannerService) Lookup(parentCtx context.Context,
	namespace, podset, scheduledPod string, eligibleNodes []string) (PlannerList, error) {

	podSetLabel := fmt.Sprintf("planner.ciena.io/%s", podset)
	defaultPodSetLabel := "planner.ciena.io/default"

	labels := fmt.Sprintf("%s,%s", podSetLabel, defaultPodSetLabel)

	ctx, cancel := context.WithTimeout(parentCtx, s.callTimeout)
	defer cancel()

	svcs, err := s.handle.ClientSet().CoreV1().Services("").List(ctx, metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		return nil, err
	}

	if len(svcs.Items) == 0 {
		return nil, ErrNoPlannersFound
	}

	var planners PlannerList
	var defaultPlanners PlannerList

	for _, svc := range svcs.Items {
		for k, v := range svc.Labels {
			switch {
			case k == podSetLabel && v == "enabled":

				planners = append(planners, &Planner{
					Service:       svc,
					Namespace:     namespace,
					Podset:        podset,
					ScheduledPod:  scheduledPod,
					EligibleNodes: eligibleNodes,
					Log:           s.log.WithName("podset-planner"),
					CallTimeout:   s.callTimeout,
					DialOptions: []grpc.DialOption{
						grpc.WithInsecure(),
					},
				})

			case k == defaultPodSetLabel && v == "enabled":

				defaultPlanners = append(defaultPlanners, &Planner{
					Service:       svc,
					Namespace:     namespace,
					Podset:        podset,
					ScheduledPod:  scheduledPod,
					EligibleNodes: eligibleNodes,
					Log:           s.log.WithName("default-podset-planner"),
					CallTimeout:   s.callTimeout,
					DialOptions: []grpc.DialOption{
						grpc.WithInsecure(),
					},
				})
			}
		}
	}

	if len(planners) == 0 {

		if len(defaultPlanners) == 0 {
			return nil, ErrNoPlannersFound
		}

		return defaultPlanners, nil
	}

	return planners, nil
}

func (planners PlannerList) Invoke(parentCtx context.Context,
	trigger *plannerv1alpha1.ScheduleTrigger) (map[string]string, error) {

	var planRewards []*planReward
	var lastError error

	for _, plan := range planners {
		assignments, err := plan.BuildSchedulePlan(parentCtx)
		if err != nil {
			st := status.Convert(err)
			lastError = fmt.Errorf("%v", st.Message())
			continue
		}

		planRewards = append(planRewards, computePlanReward(trigger, assignments))
	}

	if len(planRewards) == 0 {
		if lastError != nil {
			return nil, lastError
		}

		return nil, ErrBuildingPlan
	}

	if len(planRewards) == 1 {
		return planRewards[0].assignments, nil
	}

	sort.SliceStable(planRewards, func(i, j int) bool {
		return planRewards[i].reward > planRewards[j].reward
	})

	return planRewards[0].assignments, nil
}

func computePlanReward(trigger *plannerv1alpha1.ScheduleTrigger, assignments map[string]string) *planReward {

	return &planReward{assignments: assignments, reward: rand.Intn(100) + 1}
}

func (p *Planner) BuildSchedulePlan(parentCtx context.Context) (map[string]string, error) {
	p.Log.V(1).Info("build-schedule-plan", "namespace", p.Namespace,
		"podset", p.Podset,
		"scheduledPod", p.ScheduledPod)

	dns := fmt.Sprintf("%s.%s.svc.cluster.local:7309", p.Service.Name, p.Service.Namespace)

	dctx, dcancel := context.WithTimeout(parentCtx, p.CallTimeout)
	defer dcancel()

	conn, err := grpc.DialContext(dctx, dns, p.DialOptions...)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	client := planner.NewSchedulePlannerClient(conn)
	ctx, cancel := context.WithTimeout(parentCtx, p.CallTimeout)
	defer cancel()

	req := &planner.SchedulePlanRequest{Namespace: p.Namespace,
		PodSet:        p.Podset,
		ScheduledPod:  p.ScheduledPod,
		EligibleNodes: p.EligibleNodes,
	}

	p.Log.V(1).Info("build-schedule-plan-request", "podset", p.Podset,
		"namespace", p.Namespace,
		"scheduledPod", p.ScheduledPod,
		"nodes", p.EligibleNodes)

	resp, err := client.BuildSchedulePlan(ctx, req)
	if err != nil {
		p.Log.Error(err, "build-schedule-plan-request-error")
		return nil, err
	}

	if resp.Assignments != nil {
		p.Log.V(1).Info("build-schedule-plan-request", "assignments", resp.Assignments)
	}

	return resp.Assignments, nil
}
