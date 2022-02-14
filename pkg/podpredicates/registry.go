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

// Registry contains a map of predicates to their initialization function.
type Registry map[string]PredicateFactory

// Add is used to add a predicate with its init function to the registry.
func (r Registry) Add(name string, factory PredicateFactory) error {
	if _, ok := r[name]; ok {
		return ErrAlreadyExists
	}

	r[name] = factory

	return nil
}

// Merge is used to merge new registry into the current registry.
func (r Registry) Merge(newr Registry) error {
	for name, factory := range newr {
		if err := r.Add(name, factory); err != nil {
			return err
		}
	}

	return nil
}

// NewInTreeRegistry is used to instantiate a built-in registry.
func NewInTreeRegistry() Registry {
	return Registry{
		podAntiAffinityName: func(handle PredicateHandle) (Predicate, error) {
			return newPodAntiAffinityPredicate(handle)
		},
		podIsADuplicateName: func(handle PredicateHandle) (Predicate, error) {
			return newPodIsADuplicatePredicate(handle)
		},
		podFitsNodeName: func(handle PredicateHandle) (Predicate, error) {
			return newPodFitsNodePredicate(handle)
		},
	}
}
