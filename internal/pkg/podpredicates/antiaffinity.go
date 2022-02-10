package podpredicates

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	podutil "sigs.k8s.io/descheduler/pkg/descheduler/pod"
	"sigs.k8s.io/descheduler/pkg/utils"
)

type podAntiAffinity struct {
	handle PredicateHandle
}

const (
	podAntiAffinityName = "PodAntiAffinity"
)

var (
	_ Predicate       = &podAntiAffinity{}
	_ FilterPredicate = &podAntiAffinity{}
)

//nolint:unparam
func newPodAntiAffinityPredicate(handle PredicateHandle) (*podAntiAffinity, error) {
	return &podAntiAffinity{
		handle: handle,
	}, nil
}

func (p *podAntiAffinity) Name() string {
	return podAntiAffinityName
}

// check if a given pod has anti-affinity with an existing pod on the given node.
func (p *podAntiAffinity) Filter(parentCtx context.Context,
	podsetHandle PodSetHandle,
	pod *v1.Pod, node *v1.Node,
) *framework.Status {
	ctx, cancel := context.WithTimeout(parentCtx, p.handle.CallTimeout())
	defer cancel()

	pods, err := podutil.ListPodsOnANode(ctx, podsetHandle.ClientSet(), node)
	if err != nil {
		return framework.AsStatus(err)
	}

	return p.checkPodsWithAntiAffinityExist(parentCtx, podsetHandle, pod, pods, node)
}

func (p *podAntiAffinity) checkPodsWithAntiAffinityExist(parentCtx context.Context,
	podsetHandle PodSetHandle,
	pod *v1.Pod,
	pods []*v1.Pod,
	node *v1.Node,
) *framework.Status {
	affinity := pod.Spec.Affinity

	ctx, cancel := context.WithTimeout(parentCtx, p.handle.CallTimeout())
	defer cancel()

	podsetPods, err := podsetHandle.List(ctx, node)
	if err != nil {
		return framework.AsStatus(err)
	}

	// merge the podset pods with the pods for the node before checking anti-affinity
	pods = mergePods(pods, podsetPods)

	if affinity != nil && affinity.PodAntiAffinity != nil {
		affinityTerms := getPodAntiAffinityTerms(affinity.PodAntiAffinity)

		for index := range affinityTerms {
			term := &affinityTerms[index]
			namespaces := utils.GetNamespacesFromPodAffinityTerm(pod, term)

			selector, err := metav1.LabelSelectorAsSelector(term.LabelSelector)
			if err != nil {
				return framework.NewStatus(framework.Unschedulable)
			}

			for _, existingPod := range pods {
				if existingPod.Name != pod.Name &&
					utils.PodMatchesTermsNamespaceAndSelector(existingPod,
						namespaces,
						selector) {
					return framework.NewStatus(framework.Unschedulable)
				}
			}
		}
	}

	return framework.NewStatus(framework.Success)
}

// getPodAntiAffinityTerms gets the antiaffinity terms for the given pod.
func getPodAntiAffinityTerms(podAntiAffinity *v1.PodAntiAffinity) []v1.PodAffinityTerm {
	if podAntiAffinity == nil {
		return []v1.PodAffinityTerm{}
	}

	return podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
}
