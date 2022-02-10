package planner

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ciena/outbound/internal/pkg/client"
	"github.com/ciena/outbound/internal/pkg/podpredicates"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// PodSetPlanner stores the info related to podset planning.
type PodSetPlanner struct {
	options          PlannerOptions
	clientset        *kubernetes.Clientset
	log              logr.Logger
	quit             chan struct{}
	nodeLister       listersv1.NodeLister
	podToNodeMap     map[ktypes.NamespacedName]string
	updateQueue      workqueue.RateLimitingInterface
	predicateHandler *podpredicates.PredicateHandler
	plannerClient    *client.SchedulePlannerClient
	sync.Mutex
}

// PlannerOptions is the configurable set of options for creating planner.
type PlannerOptions struct {
	CallTimeout        time.Duration
	Parallelism        int
	UpdateWorkerPeriod time.Duration
}

type podPlannerInfo struct {
	EligibleNodes []string
}

type workWrapper struct {
	work func() error
}

// NewPlanner is used to instantate a podset planner.
func NewPlanner(options PlannerOptions,
	clientset *kubernetes.Clientset,
	plannerClient *client.SchedulePlannerClient,
	log logr.Logger) (*PodSetPlanner, error) {

	planner := &PodSetPlanner{
		options:      options,
		clientset:    clientset,
		log:          log,
		quit:         make(chan struct{}),
		podToNodeMap: make(map[ktypes.NamespacedName]string),
		updateQueue: workqueue.NewRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter()),
		plannerClient: plannerClient,
	}

	predicateHandler, err := podpredicates.New(
		podpredicates.WithCallTimeout(options.CallTimeout),
		podpredicates.WithParallelism(options.Parallelism),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating predicate handler: %w", err)
	}

	planner.predicateHandler = predicateHandler

	var addFunc func(interface{})

	var deleteFunc func(obj interface{})

	updateFunc := func(oldObj, newObj interface{}) {
		oldPod, ok := oldObj.(*v1.Pod)
		if !ok {
			return
		}

		newPod, ok := newObj.(*v1.Pod)
		if !ok {
			return
		}

		planner.handlePodUpdate(oldPod, newPod)
	}

	planner.nodeLister = initInformers(
		clientset,
		log,
		planner.quit,
		addFunc,
		updateFunc,
		deleteFunc)

	go planner.listenForUpdateEvents()

	return planner, nil
}

func getEligibleNodes(nodeLister listersv1.NodeLister) ([]*v1.Node, error) {
	nodes, err := nodeLister.List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("unable to list nodes: %w", err)
	}

	var eligibleNodes, preferNoScheduleNodes []*v1.Node

	for _, node := range nodes {
		preferNoSchedule, noSchedule := false, false

		for i := range node.Spec.Taints {
			if node.Spec.Taints[i].Effect == v1.TaintEffectPreferNoSchedule {
				preferNoSchedule = true
			} else if node.Spec.Taints[i].Effect == v1.TaintEffectNoSchedule ||
				node.Spec.Taints[i].Effect == v1.TaintEffectNoExecute {
				noSchedule = true
			}
		}

		if !noSchedule {
			eligibleNodes = append(eligibleNodes, node)
		} else if preferNoSchedule {
			preferNoScheduleNodes = append(preferNoScheduleNodes, node)
		}
	}

	if len(eligibleNodes) == 0 {
		return preferNoScheduleNodes, nil
	}

	return eligibleNodes, nil
}

func initInformers(clientset *kubernetes.Clientset,
	log logr.Logger,
	quit chan struct{},
	add func(interface{}),
	update func(interface{}, interface{}),
	delete func(interface{})) listersv1.NodeLister {
	factory := informers.NewSharedInformerFactory(clientset, 0)
	nodeInformer := factory.Core().V1().Nodes()

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node, ok := obj.(*v1.Node)
			if !ok {
				log.V(1).Info("this-is-not-a-node")

				return
			}

			log.V(1).Info("new-node-added", "node", node.GetName())
		},
	})

	podInformer := factory.Core().V1().Pods()
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    add,
		UpdateFunc: update,
		DeleteFunc: delete,
	})

	factory.Start(quit)

	return nodeInformer.Lister()
}

func (p *PodSetPlanner) handlePodUpdate(oldPod *v1.Pod, newPod *v1.Pod) {
	p.Lock()
	defer p.Unlock()

	if oldPod.Status.Phase != v1.PodRunning && newPod.Status.Phase == v1.PodRunning {
		delete(p.podToNodeMap, ktypes.NamespacedName{Name: newPod.Name, Namespace: newPod.Namespace})
	} else if oldPod.GetDeletionTimestamp() == nil && newPod.GetDeletionTimestamp() != nil {
		p.handlePodDeleteWithLock(newPod)
	}
}

func (p *PodSetPlanner) handlePodDeleteWithLock(pod *v1.Pod) {
	delete(p.podToNodeMap, ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace})

	p.removePodFromPlanner(pod)
}

func (p *PodSetPlanner) removePodFromPlanner(pod *v1.Pod) {
	podset := ""

	for k, v := range pod.Labels {
		if k == "planner.ciena.io/pod-set" {
			podset = v
		}
	}

	if podset == "" {
		return
	}

	name, namespace := pod.Name, pod.Namespace

	doUpdate := func() error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		updateFailed, err := p.plannerClient.Delete(ctx, name,
			namespace, podset)

		if err == nil {
			p.log.V(1).Info("planner-delete",
				"pod", name, "namespace", namespace,
				"podset", podset)

			return nil
		}

		if !updateFailed {
			p.log.Error(err, "error-deleting-planner",
				"pod", name, "namespace", namespace, "podset", podset)

			return nil
		}

		p.log.Error(err, "planner-update-will-be-retried",
			"pod", name, "namespace", namespace, "podset", podset)

		//nolint:wrapcheck
		return err
	}

	if err := doUpdate(); err == nil {
		return
	}

	// retry update with rate limiter
	p.updateQueue.AddRateLimited(&workWrapper{work: doUpdate})
}

// FindNodeLister finds an node reference from the lister cache.
func (p *PodSetPlanner) FindNodeLister(node string) (*v1.Node, error) {
	nodes, err := getEligibleNodes(p.nodeLister)
	if err != nil {
		return nil, err
	}

	for _, n := range nodes {
		if n.Name == node {
			return n, nil
		}
	}

	return nil, fmt.Errorf("cound not find node lister instance for node %s: %w", node, ErrNotFound)
}

// Stop is used to stop planner.
func (p *PodSetPlanner) Stop() {
	close(p.quit)
}

func (p *PodSetPlanner) processUpdate(item interface{}) {
	forgetItem := true

	defer func() {
		if forgetItem {
			p.updateQueue.Forget(item)
		}
	}()

	workItem, ok := item.(*workWrapper)
	if !ok {
		return
	}

	if err := workItem.work(); err != nil {
		forgetItem = false

		p.log.V(1).Info("planner-update-work-failed", "numrequeues", p.updateQueue.NumRequeues(item))

		p.updateQueue.AddRateLimited(item)
	}
}

func (p *PodSetPlanner) processUpdateEvents() {
	item, quit := p.updateQueue.Get()
	if quit {
		return
	}

	defer p.updateQueue.Done(item)

	p.processUpdate(item)
}

func (p *PodSetPlanner) updateWorker() {
	p.processUpdateEvents()
}

func (p *PodSetPlanner) listenForUpdateEvents() {
	defer p.updateQueue.ShutDown()

	go wait.Until(p.updateWorker, p.options.UpdateWorkerPeriod, p.quit)

	<-p.quit
}

func (p *PodSetPlanner) getEligibleNodesForPod(parentCtx context.Context,
	podSetHandler *podSetHandlerImpl,
	pod *v1.Pod,
	allEligibleNodes []*v1.Node) ([]string, error) {

	filteredNodes := p.predicateHandler.FindNodesThatPassFilters(parentCtx,
		podSetHandler,
		pod,
		allEligibleNodes)
	if len(filteredNodes) == 0 {
		return nil, ErrNoNodesFound
	}

	nodeNames := make([]string, len(filteredNodes))

	for i := range filteredNodes {
		nodeNames[i] = filteredNodes[i].Name
	}

	return nodeNames, nil
}

func (p *PodSetPlanner) getEligibleNodes(parentCtx context.Context,
	podSetHandler *podSetHandlerImpl,
	pod *v1.Pod,
	allEligibleNodes []*v1.Node,
	schedulingMap map[ktypes.NamespacedName]*podPlannerInfo) ([]string, error) {

	plannerInfo, ok := schedulingMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}]
	if ok {
		return plannerInfo.EligibleNodes, nil
	}

	//get and load eligible nodes for pod
	eligibleNodes, err := p.getEligibleNodesForPod(parentCtx, podSetHandler, pod, allEligibleNodes)
	if err != nil {
		return nil, err
	}

	plannerInfo = &podPlannerInfo{EligibleNodes: eligibleNodes}
	schedulingMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}] = plannerInfo

	return eligibleNodes, nil
}

func (p *PodSetPlanner) BuildPlan(parentCtx context.Context,
	podSetHandler *podSetHandlerImpl,
	podList []*v1.Pod,
	schedulingMap map[ktypes.NamespacedName]*podPlannerInfo) (map[string]string, error) {
	var failedPodList []*v1.Pod

	podNames := make([]string, len(podList))

	for i, pod := range podList {
		podNames[i] = pod.Name
	}

	p.log.V(1).Info("scheduler-planner", "pod-list", podNames)

	assignmentMap := make(map[string]string)

	allEligibleNodes, err := getEligibleNodes(p.nodeLister)
	if err != nil {
		return nil, err
	}

	p.Lock()
	defer p.Unlock()

	for _, pod := range podList {
		if pod.Status.Phase == v1.PodFailed || pod.DeletionTimestamp != nil {

			continue
		}

		// check if the pod is already assigned a node
		if node, err := p.GetNodeName(pod); err == nil {
			assignmentMap[pod.Name] = node

			continue
		}

		eligibleNodes, err := p.getEligibleNodes(parentCtx, podSetHandler,
			pod, allEligibleNodes, schedulingMap)

		if err != nil {
			p.log.Error(err, "eligible-nodes-not-found", "pod", pod.Name)
			failedPodList = append(failedPodList, pod)

			continue
		}

		selectedNode, err := p.findFit(parentCtx, pod, eligibleNodes)
		if err != nil {
			p.log.Error(err, "nodes-not-found", "pod", pod.Name)
			failedPodList = append(failedPodList, pod)

			continue
		}

		p.log.V(1).Info("pod-node-assignment", "pod", pod.Name, "node", selectedNode.Name)

		assignmentMap[pod.Name] = selectedNode.Name
	}

	if len(failedPodList) > 0 {
		p.log.V(1).Info("build-plan", "pods-planning-failed", len(failedPodList))

		for _, pod := range failedPodList {
			p.log.V(1).Info("build-plan", "failed-to-build-plan-for-pod", pod.Name)
		}
	}

	return assignmentMap, nil
}

// BuildSchedulePlan builds a schedule plan for the podset and returns an
// assignment map containing pod to node assignments.
func (p *PodSetPlanner) BuildSchedulePlan(parentCtx context.Context,
	namespace, podSet string,
	scheduledPod string,
	eligibleNodes []string) (map[string]string, error) {

	p.log.V(1).Info("build-schedule-plan", "podset", podSet, "namespace", namespace)

	ctx, cancel := context.WithTimeout(parentCtx, p.options.CallTimeout)
	defer cancel()

	pods, err := p.clientset.CoreV1().Pods(namespace).List(ctx,
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("planner.ciena.io/pod-set=%s", podSet),
		},
	)
	if err != nil {
		p.log.Error(err, "build-schedule-plan-list-pods-error", "podset", podSet)

		return nil, err
	}

	podList := make([]*v1.Pod, len(pods.Items))
	schedulingMap := make(map[ktypes.NamespacedName]*podPlannerInfo)

	indexOfScheduledPod := 0

	for index := range pods.Items {
		podList[index] = &pods.Items[index]

		// the eligibleNodes is for the pod getting scheduled
		if podList[index].Name == scheduledPod {
			indexOfScheduledPod = index
			schedulingMap[ktypes.NamespacedName{
				Name:      pods.Items[index].Name,
				Namespace: pods.Items[index].Namespace,
			}] = &podPlannerInfo{
				EligibleNodes: eligibleNodes,
			}
		}
	}

	// move the scheduled pod to the top
	if indexOfScheduledPod > 0 {
		podList[0], podList[indexOfScheduledPod] = podList[indexOfScheduledPod], podList[0]
	}

	podSetHandler := newPodSetHandler(p, podList)

	return p.BuildPlan(parentCtx, podSetHandler, podList, schedulingMap)
}

// called with constraintpolicymutex held.
func (p *PodSetPlanner) getPodNode(pod *v1.Pod) (string, error) {
	node, ok := p.podToNodeMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}]
	if !ok {
		return "", fmt.Errorf("cannot find pod %s node: %w", pod.Name, ErrPodNotAssigned)
	}

	return node, nil
}

// called with constraintpolicymutex held.
func (p *PodSetPlanner) setPodNode(pod *v1.Pod, nodeName string) {
	p.podToNodeMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}] = nodeName
}

// Fast version. look up internal cache.
// look up for the node in the lister cache if pod host ip is not set.
// GetNodeName returns the nodename for the given pod.
func (p *PodSetPlanner) GetNodeName(pod *v1.Pod) (string, error) {
	if pod.Status.HostIP == "" {
		return p.getPodNode(pod)
	}

	nodes, err := getEligibleNodes(p.nodeLister)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {
		for i := range node.Status.Addresses {

			if node.Status.Addresses[i].Address == pod.Status.HostIP {
				return node.Name, nil
			}
		}
	}

	return "", fmt.Errorf("pod ip %s not found in node lister cache: %w", pod.Status.HostIP, ErrNotFound)
}

func (p *PodSetPlanner) findFit(_ context.Context, pod *v1.Pod, eligibleNodes []string) (*v1.Node, error) {
	if len(eligibleNodes) == 0 {
		p.log.V(1).Info("no-eligible-nodes-found", "pod", pod.Name)

		return nil, fmt.Errorf("no-eligible-nodes-found-for-pod-%s: %w", pod.Name, ErrNoNodesFound)
	}

	selectedNode := eligibleNodes[rand.Intn(len(eligibleNodes))]

	nodeInstance, err := p.FindNodeLister(selectedNode)
	if err != nil {
		p.log.V(1).Info("node-instance-not-found-in-lister-cache")

		return nil, err
	}

	p.setPodNode(pod, nodeInstance.Name)
	p.log.V(1).Info("found-matching", "node", nodeInstance.Name, "pod", pod.Name)

	return nodeInstance, nil
}

// nolint:gochecknoinits
func init() {
	rand.Seed(time.Now().Unix())
}
