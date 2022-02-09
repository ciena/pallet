package podpredicates

type Registry map[string]PredicateFactory

func (r Registry) Add(name string, factory PredicateFactory) error {
	if _, ok := r[name]; ok {
		return ErrAlreadyExists
	}

	r[name] = factory

	return nil
}

func (r Registry) Merge(new Registry) error {
	for name, factory := range new {
		if err := r.Add(name, factory); err != nil {

			return err
		}
	}

	return nil
}

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
