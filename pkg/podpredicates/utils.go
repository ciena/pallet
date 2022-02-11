package podpredicates

import (
	v1 "k8s.io/api/core/v1"
	ktypes "k8s.io/apimachinery/pkg/types"
)

func mergePods(set1 []*v1.Pod, set2 []*v1.Pod) []*v1.Pod {
	res := make([]*v1.Pod, 0, len(set1)+len(set2))

	presenceMap := make(map[ktypes.NamespacedName]struct{})

	for _, pod := range set1 {
		res = append(res, pod)
		presenceMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}] = struct{}{}
	}

	for _, pod := range set2 {
		if _, ok := presenceMap[ktypes.NamespacedName{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		}]; ok {
			continue
		}

		res = append(res, pod)
	}

	return res
}
