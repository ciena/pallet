package podpredicates

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	descheduler_nodeutil "sigs.k8s.io/descheduler/pkg/descheduler/node"
	descheduler_utils "sigs.k8s.io/descheduler/pkg/utils"
)

// Resource is a collection of compute resource.
type Resource struct {
	MilliCPU         int64
	Memory           int64
	EphemeralStorage int64
	// We store allowedPodNumber (which is Node.Status.Allocatable.Pods().Value())
	// explicitly as int, to avoid conversions and improve performance.
	AllowedPodNumber int
	// ScalarResources
	ScalarResources map[v1.ResourceName]int64
}

// NewResource creates a Resource from ResourceList.
func NewResource(rl v1.ResourceList) *Resource {
	r := &Resource{}
	r.Add(rl)

	return r
}

// Add adds ResourceList into Resource.
func (r *Resource) Add(rl v1.ResourceList) {
	if r == nil {
		return
	}

	for rName := range rl {
		rQuant := rl[rName]

		//nolint:exhaustive
		switch rName {
		case v1.ResourceCPU:
			r.MilliCPU += rQuant.MilliValue()
		case v1.ResourceMemory:
			r.Memory += rQuant.Value()
		case v1.ResourcePods:
			r.AllowedPodNumber += int(rQuant.Value())
		case v1.ResourceEphemeralStorage:
			// if utilfeature.DefaultFeatureGate.Enabled(LocalStorageCapacityIsolation) {
			// 	// if the local storage capacity isolation feature gate is disabled, pods request 0 disk.
			// 	r.EphemeralStorage += rQuant.Value()
			// }
		default:
			if IsScalarResourceName(rName) {
				r.AddScalar(rName, rQuant.Value())
			}
		}
	}
}

// ResourceList returns a resource list of this resource.
func (r *Resource) ResourceList() v1.ResourceList {
	result := v1.ResourceList{
		v1.ResourceCPU:              *resource.NewMilliQuantity(r.MilliCPU, resource.DecimalSI),
		v1.ResourceMemory:           *resource.NewQuantity(r.Memory, resource.BinarySI),
		v1.ResourcePods:             *resource.NewQuantity(int64(r.AllowedPodNumber), resource.BinarySI),
		v1.ResourceEphemeralStorage: *resource.NewQuantity(r.EphemeralStorage, resource.BinarySI),
	}
	for rName, rQuant := range r.ScalarResources {

		if IsHugePageResourceName(rName) {
			result[rName] = *resource.NewQuantity(rQuant, resource.BinarySI)
		} else {
			result[rName] = *resource.NewQuantity(rQuant, resource.DecimalSI)
		}
	}

	return result
}

// Clone returns a copy of this resource.
func (r *Resource) Clone() *Resource {
	res := &Resource{
		MilliCPU:         r.MilliCPU,
		Memory:           r.Memory,
		AllowedPodNumber: r.AllowedPodNumber,
		EphemeralStorage: r.EphemeralStorage,
	}
	if r.ScalarResources != nil {
		res.ScalarResources = make(map[v1.ResourceName]int64)
		for k, v := range r.ScalarResources {
			res.ScalarResources[k] = v
		}
	}

	return res
}

// AddScalar adds a resource by a scalar value of this resource.
func (r *Resource) AddScalar(name v1.ResourceName, quantity int64) {
	r.SetScalar(name, r.ScalarResources[name]+quantity)
}

// SetScalar sets a resource by a scalar value of this resource.
func (r *Resource) SetScalar(name v1.ResourceName, quantity int64) {
	// Lazily allocate scalar resource map.
	if r.ScalarResources == nil {
		r.ScalarResources = map[v1.ResourceName]int64{}
	}

	r.ScalarResources[name] = quantity
}

// SetMaxResource compares with ResourceList and takes max value for each Resource.
func (r *Resource) SetMaxResource(rl v1.ResourceList) {
	if r == nil {
		return
	}

	for rName := range rl {
		rQuantity := rl[rName]

		//nolint:exhaustive
		switch rName {
		case v1.ResourceMemory:
			if mem := rQuantity.Value(); mem > r.Memory {
				r.Memory = mem
			}
		case v1.ResourceCPU:
			if cpu := rQuantity.MilliValue(); cpu > r.MilliCPU {
				r.MilliCPU = cpu
			}
		case v1.ResourceEphemeralStorage:
			if ephemeralStorage := rQuantity.Value(); ephemeralStorage > r.EphemeralStorage {
				r.EphemeralStorage = ephemeralStorage
			}
		default:
			if IsScalarResourceName(rName) {
				value := rQuantity.Value()
				if value > r.ScalarResources[rName] {
					r.SetScalar(rName, value)
				}
			}
		}
	}
}

func computePodResourceRequest(pod *v1.Pod) *Resource {
	result := &Resource{}
	for i := range pod.Spec.Containers {
		result.Add(pod.Spec.Containers[i].Resources.Requests)
	}

	// take max_resource(sum_pod, any_init_container)
	for i := range pod.Spec.InitContainers {
		result.SetMaxResource(pod.Spec.InitContainers[i].Resources.Requests)
	}

	// If Overhead is being utilized, add to the total requests for the pod
	if pod.Spec.Overhead != nil && utilfeature.DefaultFeatureGate.Enabled(PodOverhead) {
		result.Add(pod.Spec.Overhead)
	}

	return result
}

type podFitsNode struct {
	handle PredicateHandle
}

const (
	podFitsNodeName = "PodFitsNode"
)

var (
	_ Predicate       = &podFitsNode{}
	_ FilterPredicate = &podFitsNode{}
)

//nolint:unparam
func newPodFitsNodePredicate(
	handle PredicateHandle) (*podFitsNode, error) {

	return &podFitsNode{handle: handle}, nil
}

func (p *podFitsNode) Name() string {

	return podFitsNodeName
}

// Filter checks if pod fits node resource/taints.
func (p *podFitsNode) Filter(_ context.Context,
	_ PodSetHandle,
	pod *v1.Pod,
	node *v1.Node) *framework.Status {

	// check for node taint toleration
	if ok := descheduler_utils.TolerationsTolerateTaintsWithFilter(
		pod.Spec.Tolerations, node.Spec.Taints,
		func(taint *v1.Taint) bool {
			return taint.Effect == v1.TaintEffectNoSchedule
		}); !ok {

		return framework.NewStatus(framework.Unschedulable)
	}

	// check for node label and affinity
	if ok := descheduler_nodeutil.PodFitsCurrentNode(pod, node); !ok {
		return framework.NewStatus(framework.Unschedulable)
	}

	// check if it fits node resource
	allocatable := NewResource(node.Status.Allocatable)
	podRequest := computePodResourceRequest(pod)
	reasons := []string{}

	if podRequest.MilliCPU == 0 &&
		podRequest.Memory == 0 &&
		podRequest.EphemeralStorage == 0 &&
		len(podRequest.ScalarResources) == 0 {

		return framework.NewStatus(framework.Success)
	}

	if allocatable.MilliCPU < podRequest.MilliCPU {
		reasons = append(reasons, fmt.Sprintf("Insufficient cpu. Allocatable %d, requested %d for pod %s",
			allocatable.MilliCPU, podRequest.MilliCPU, pod.Name))
	}

	if allocatable.Memory < podRequest.Memory {
		reasons = append(reasons, fmt.Sprintf("Insufficient memory. Allocatable %d, requested %d for pod %s",
			allocatable.Memory, podRequest.Memory, pod.Name))
	}

	if allocatable.EphemeralStorage < podRequest.EphemeralStorage {
		reasons = append(reasons, fmt.Sprintf("Insufficient ephemeral-storage. Allocatable %d, requested %d for pod %s",
			allocatable.EphemeralStorage, podRequest.EphemeralStorage, pod.Name))
	}

	for rName, rQuant := range podRequest.ScalarResources {
		if allocatable.ScalarResources[rName] < rQuant {
			reasons = append(reasons, fmt.Sprintf("Insufficient scalar resource %v for pod %s. Allocatable %d, requested %d",
				rName, pod.Name, allocatable.ScalarResources[rName], rQuant))
		}
	}

	if len(reasons) > 0 {

		return framework.NewStatus(framework.Error, reasons...)
	}

	return framework.NewStatus(framework.Success)
}
