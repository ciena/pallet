package podpredicates

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/ciena/outbound/internal/pkg/parallelize"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

//Predicate defines an interface to implement predicates.
type Predicate interface {
	Name() string
}

//PredicateHandle defines a handle used while initializing the predicate.
type PredicateHandle interface {
	CallTimeout() time.Duration
}

//FilterPredicate defines an interface to implement filter predicate.
type FilterPredicate interface {
	Predicate
	Filter(ctx context.Context, podsetHandle PodSetHandle, pod *v1.Pod, node *v1.Node) *framework.Status
}

type predicateOption struct {
	outOfTreeRegistry Registry
	callTimeout       time.Duration
	parallelism       int
}

//PredicateHandler is a framework used to run through all the predicates.
type PredicateHandler struct {
	parallelizer     *parallelize.Parallelizer
	predicateMap     map[string]Predicate
	registry         Registry
	filterPredicates []FilterPredicate
	predicateOption
}

type extensionPoint struct {
	slicePtr interface{}
}

//PredicateFactory is the initialization routine used to create a new predicate instance.
type PredicateFactory func(handle PredicateHandle) (Predicate, error)

//Option is the option used while creating the predicate handler.
type Option func(o *predicateOption)

const (
	defaultCallTimeout = time.Second * 15
)

type predicateHandleImpl struct {
	callTimeout time.Duration
}

func (p *predicateHandleImpl) CallTimeout() time.Duration {
	return p.callTimeout
}

//WithCallTimeout is the grpc timeout to be used while accessing remotes with predicates.
func WithCallTimeout(timeout time.Duration) Option {
	return func(o *predicateOption) {
		o.callTimeout = timeout
	}
}

//WithOutOfTreeRegistry is used to register a custom predicate.
func WithOutOfTreeRegistry(registry Registry) Option {
	return func(o *predicateOption) {
		o.outOfTreeRegistry = registry
	}
}

//WithParallelism defines the parallelism factor used while running the filter predicates.
func WithParallelism(p int) Option {
	return func(o *predicateOption) {
		o.parallelism = p
	}
}

//New is used to create a predicateHandler instance.
func New(opts ...Option) (*PredicateHandler, error) {

	popt := predicateOption{
		callTimeout: defaultCallTimeout,
	}

	for _, opt := range opts {
		opt(&popt)
	}

	registry := NewInTreeRegistry()

	if err := registry.Merge(popt.outOfTreeRegistry); err != nil {

		return nil, err
	}

	predicateMap := make(map[string]Predicate, len(registry))

	predHandle := &predicateHandleImpl{callTimeout: popt.callTimeout}

	for name, factory := range registry {

		if pred, err := factory(predHandle); err == nil {
			predicateMap[name] = pred
		}
	}

	predicateHandler := &PredicateHandler{
		predicateMap:    predicateMap,
		registry:        registry,
		parallelizer:    parallelize.NewParallelizer(popt.parallelism),
		predicateOption: popt,
	}

	for _, e := range predicateHandler.getExtensionPoints() {

		if err := updatePredicateList(e.slicePtr, predicateMap); err != nil {
			return nil, err
		}
	}

	return predicateHandler, nil
}

func (p *PredicateHandler) getExtensionPoints() []extensionPoint {
	return []extensionPoint{
		{&p.filterPredicates},
	}
}

func (p *PredicateHandler) RunFilterPredicates(ctx context.Context,
	handle PodSetHandle,
	pod *v1.Pod,
	node *v1.Node) *framework.Status {

	for _, filterPred := range p.filterPredicates {
		st := filterPred.Filter(ctx, handle, pod, node)
		if !st.IsSuccess() {

			return st
		}
	}

	return framework.NewStatus(framework.Success)
}

//FindNodesThatPassFilters is used to find an eligible nodeset from all eligible nodes
//that pass filter predicate for a given pod
func (p *PredicateHandler) FindNodesThatPassFilters(parentCtx context.Context,
	podsetHandle PodSetHandle,
	pod *v1.Pod,
	eligibleNodes []*v1.Node) []*v1.Node {

	var updateLock sync.Mutex

	filteredNodes := []*v1.Node{}

	ctx, cancel := context.WithTimeout(parentCtx, p.callTimeout)
	defer cancel()

	// run the filter for each node and save on success
	// filter is run for all nodes in parallel
	check := func(index int) {
		node := eligibleNodes[index]

		status := p.RunFilterPredicates(ctx, podsetHandle, pod, node)

		if status.IsSuccess() {
			updateLock.Lock()
			filteredNodes = append(filteredNodes, node)
			updateLock.Unlock()
		}
	}

	p.parallelizer.Until(ctx, len(eligibleNodes), check)

	return filteredNodes
}

func updatePredicateList(predicateList interface{}, predicateMap map[string]Predicate) error {

	predicates := reflect.ValueOf(predicateList).Elem()
	predicateType := predicates.Type().Elem()

	for name, pred := range predicateMap {

		if !reflect.TypeOf(pred).Implements(predicateType) {

			return fmt.Errorf("%w: predicate %s does not extend %s predicate",
				ErrNoPredicate, name, predicateType.Name())
		}

		newPredicates := reflect.Append(predicates, reflect.ValueOf(pred))

		predicates.Set(newPredicates)
	}

	return nil
}
