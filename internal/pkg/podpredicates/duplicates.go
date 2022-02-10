package podpredicates

import (
	"context"
	"reflect"
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	podutil "sigs.k8s.io/descheduler/pkg/descheduler/pod"
)

const (
	podIsADuplicateName = "PodIsADuplicate"
)

type podIsADuplicate struct {
	handle PredicateHandle
}

var (
	_ Predicate       = &podIsADuplicate{}
	_ FilterPredicate = &podIsADuplicate{}
)

//nolint:unparam
func newPodIsADuplicatePredicate(handle PredicateHandle) (*podIsADuplicate, error) {

	return &podIsADuplicate{handle: handle}, nil
}

func (p *podIsADuplicate) Name() string {

	return podIsADuplicateName
}

// Filter for podIsADuplicate checks if the given pod is a duplicate of another pod on the same node
// A pod is said to be a duplicate of other if both of them are from same creator, kind and are within the same
// namespace, and have at least one container with the same image.
func (p *podIsADuplicate) Filter(
	parentCtx context.Context,
	podsetHandle PodSetHandle,
	pod *v1.Pod,
	node *v1.Node,
) *framework.Status {

	ctx, cancel := context.WithTimeout(parentCtx, p.handle.CallTimeout())
	podsetPods, err := podsetHandle.List(ctx, node)

	cancel()

	if err != nil {
		return framework.AsStatus(err)
	}

	ctx, cancel = context.WithTimeout(parentCtx, p.handle.CallTimeout())
	defer cancel()

	pods, err := podutil.ListPodsOnANode(ctx, podsetHandle.ClientSet(), node)
	if err != nil {
		return framework.AsStatus(err)
	}

	pods = mergePods(pods, podsetPods)

	givenPodContainerKeys := getPodContainerKeys(pod)
	for _, p := range pods {
		// skip terminating pods
		if p.GetDeletionTimestamp() != nil {
			continue
		}

		podContainerKeys := getPodContainerKeys(p)
		if reflect.DeepEqual(givenPodContainerKeys, podContainerKeys) {
			// given pod is a duplicate of another pod on this node
			return framework.NewStatus(framework.Unschedulable)
		}
	}

	return framework.NewStatus(framework.Success)
}

func getPodContainerKeys(pod *v1.Pod) []string {
	ownerRefList := podutil.OwnerRef(pod)
	podContainerKeys := make([]string, 0, len(ownerRefList)*len(pod.Spec.Containers))

	for i := range ownerRefList {
		ownerRef := &ownerRefList[i]

		for j := range pod.Spec.Containers {

			container := &pod.Spec.Containers[j]

			// Namespace/Kind/Name should be unique for the cluster.
			// We also consider the image, as 2 pods could have the same owner but serve different purposes
			// So any non-unique Namespace/Kind/Name/Image pattern is a duplicate pod.
			s := strings.Join([]string{pod.ObjectMeta.Namespace, ownerRef.Kind, ownerRef.Name, container.Image}, "/")
			podContainerKeys = append(podContainerKeys, s)
		}
	}

	sort.Strings(podContainerKeys)

	return podContainerKeys
}
