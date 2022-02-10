package planner

import (
	"context"

	"github.com/ciena/outbound/internal/pkg/podpredicates"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Implements the PodSetHandle interface.
type podSetHandlerImpl struct {
	planner *PodSetPlanner
	pods    []*v1.Pod
}

var _ podpredicates.PodSetHandle = &podSetHandlerImpl{}

func newPodSetHandler(planner *PodSetPlanner, pods []*v1.Pod) *podSetHandlerImpl {

	return &podSetHandlerImpl{planner: planner, pods: pods}
}

// ClientSet returns the k8s clientset.
// nolint:ireturn
func (ps *podSetHandlerImpl) ClientSet() kubernetes.Interface {

	return ps.planner.clientset
}

// List returns all the pods for the podset assigned to the node.
func (ps *podSetHandlerImpl) List(_ context.Context, node *v1.Node) ([]*v1.Pod, error) {

	pods := []*v1.Pod{}

	for _, pod := range ps.pods {
		if name, err := ps.planner.GetNodeName(pod); err == nil && name == node.Name {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}
