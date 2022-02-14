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
