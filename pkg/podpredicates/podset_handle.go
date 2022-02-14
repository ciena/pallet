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
	"context"

	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

// PodSetHandle defines the interface for a podset used with predicate handler.
type PodSetHandle interface {
	// ClientSet returns a k8s clientset
	ClientSet() clientset.Interface

	// List returns a list of pods for the podset on the node
	List(context.Context, *v1.Node) ([]*v1.Pod, error)
}
