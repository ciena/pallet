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
