/*
Copyright 2022 Ciena Corporation..

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

package planner

import (
	"context"

	"github.com/ciena/outbound/pkg/podpredicates"
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
