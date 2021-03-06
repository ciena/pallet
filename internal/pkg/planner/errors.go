/*
Copyright 2022 Ciena Corporation.

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

import "errors"

var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("not-found")

	// ErrNoNodesFound returned when no elible nodes are found for pod.
	ErrNoNodesFound = errors.New("no eligible nodes found")

	// ErrPodNotAssigned returned when a pod has net yet been assigned to
	// a node.
	ErrPodNotAssigned = errors.New("pod-not-assigned")

	// ErrEmptyPriorityList returned when no node scores are found to select a
	// node for the pod.
	ErrEmptyPriorityList = errors.New("empty-priority-list")
)
