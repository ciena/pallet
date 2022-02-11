package podpredicates

import (
	"context"
	stdlog "log"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/ciena/outbound/pkg/parallelize"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

// Predicate defines an interface to implement predicate handlers.
type Predicate interface {
	Name() string
}

// PredicateHandle defines a handle used while initializing the predicate.
type PredicateHandle interface {
	CallTimeout() time.Duration
	Log(prefix string) logr.Logger
}

// FilterPredicate defines an interface to implement filter predicate handlers.
type FilterPredicate interface {
	Predicate
	Filter(ctx context.Context, hdl PodSetHandle, pod *v1.Pod, node *v1.Node) *framework.Status
}

// ScorePredicate defines an interface to implement scoring a node assignment for the pod.
type ScorePredicate interface {
	Predicate
	Score(ctx context.Context, hdl PodSetHandle, pod *v1.Pod, node *v1.Node) (int64, *framework.Status)
}

type predicateOption struct {
	outOfTreeRegistry Registry
	callTimeout       time.Duration
	parallelism       int
	log               logr.Logger
}

// PredicateHandler is a framework used to run through all the predicates.
type PredicateHandler struct {
	parallelizer     *parallelize.Parallelizer
	predicateMap     map[string]Predicate
	registry         Registry
	filterPredicates []FilterPredicate
	scorePredicates  []ScorePredicate
	predicateOption
}

type extensionPoint struct {
	slicePtr interface{}
}

// PredicateFactory is the initialization routine used to create a new predicate instance.
type PredicateFactory func(handle PredicateHandle) (Predicate, error)

// Option is the option used while creating the predicate handler.
type Option func(o *predicateOption)

const (
	defaultCallTimeout = time.Second * 15
)

type predicateHandleImpl struct {
	callTimeout time.Duration
	log         logr.Logger
}

func (p *predicateHandleImpl) CallTimeout() time.Duration {
	return p.callTimeout
}

func (p *predicateHandleImpl) Log(prefix string) logr.Logger {
	return p.log.WithName(prefix)
}

// WithCallTimeout is the grpc timeout to be used while accessing remotes with predicates.
func WithCallTimeout(timeout time.Duration) Option {
	return func(o *predicateOption) {
		o.callTimeout = timeout
	}
}

// WithOutOfTreeRegistry is used to extend the built-in registry with a custom set.
func WithOutOfTreeRegistry(registry Registry) Option {
	return func(o *predicateOption) {
		o.outOfTreeRegistry = registry
	}
}

// WithParallelism defines the parallelism factor used while running the filter predicates.
func WithParallelism(p int) Option {
	return func(o *predicateOption) {
		o.parallelism = p
	}
}

// WithLogger defines the logger to be used with predicate handler.
func WithLogger(log logr.Logger) Option {
	return func(o *predicateOption) {
		o.log = log
	}
}

func stdrLogger() logr.Logger {
	stdr.SetVerbosity(1)

	return stdr.New(stdlog.New(os.Stdout, "pred-handler", stdlog.LstdFlags))
}

// New is used to create a predicateHandler instance.
func New(opts ...Option) (*PredicateHandler, error) {
	popt := predicateOption{
		callTimeout: defaultCallTimeout,
		log:         stdrLogger(),
	}

	for _, opt := range opts {
		opt(&popt)
	}

	registry := NewInTreeRegistry()

	if err := registry.Merge(popt.outOfTreeRegistry); err != nil {
		return nil, err
	}

	predicateMap := make(map[string]Predicate, len(registry))

	predHandle := &predicateHandleImpl{
		callTimeout: popt.callTimeout,
		log:         popt.log,
	}

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
		updatePredicateList(e.slicePtr, predicateMap)
	}

	return predicateHandler, nil
}

func (p *PredicateHandler) getExtensionPoints() []extensionPoint {
	return []extensionPoint{
		{&p.filterPredicates},
		{&p.scorePredicates},
	}
}

// RunFilterPredicates is used to run all filter predicates to determine
// if the pod can be scheduled on the given node.
func (p *PredicateHandler) RunFilterPredicates(ctx context.Context,
	handle PodSetHandle,
	pod *v1.Pod,
	node *v1.Node,
) *framework.Status {
	for _, filterPred := range p.filterPredicates {
		st := filterPred.Filter(ctx, handle, pod, node)
		if !st.IsSuccess() {
			return st
		}
	}

	return framework.NewStatus(framework.Success)
}

// FindNodesThatPassFilters is used to find eligible nodes for a pod that pass all filters.
func (p *PredicateHandler) FindNodesThatPassFilters(parentCtx context.Context,
	podsetHandle PodSetHandle,
	pod *v1.Pod,
	eligibleNodes []*v1.Node,
) []*v1.Node {
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
	p.log.V(1).Info("filtered-eligible-nodes", "pod", pod.Name, "nodes", len(filteredNodes))

	return filteredNodes
}

// RunScorePredicates is used to run all score predicates to determine
// the best node for the pod.
func (p *PredicateHandler) RunScorePredicates(ctx context.Context,
	handle PodSetHandle,
	pod *v1.Pod,
	nodes []*v1.Node,
) framework.NodeScoreList {
	var nodeScoreList framework.NodeScoreList

	var updateLock sync.Mutex

	scoreNodes := func(index int) {
		var sum int64

		node := nodes[index]

		for _, scorePred := range p.scorePredicates {
			score, st := scorePred.Score(ctx, handle, pod, node)
			if !st.IsSuccess() {
				continue
			}

			sum += score
		}

		updateLock.Lock()
		nodeScoreList = append(nodeScoreList, framework.NodeScore{
			Name:  node.Name,
			Score: sum,
		})
		updateLock.Unlock()
	}

	p.parallelizer.Until(ctx, len(nodes), scoreNodes)

	return nodeScoreList
}

func updatePredicateList(predicateList interface{}, predicateMap map[string]Predicate) {
	predicates := reflect.ValueOf(predicateList).Elem()
	predicateType := predicates.Type().Elem()

	for _, pred := range predicateMap {
		if !reflect.TypeOf(pred).Implements(predicateType) {
			continue
		}

		newPredicates := reflect.Append(predicates, reflect.ValueOf(pred))

		predicates.Set(newPredicates)
	}
}
