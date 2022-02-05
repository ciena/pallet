package planner

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ktypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"math/rand"
	"sync"
	"time"
)

type PodSetPlanner struct {
	options      PlannerOptions
	clientset    *kubernetes.Clientset
	log          logr.Logger
	quit         chan struct{}
	nodeLister   listersv1.NodeLister
	podToNodeMap map[ktypes.NamespacedName]string
	sync.Mutex
}

type PlannerOptions struct {
}

type PodPlannerInfo struct {
	EligibleNodes []string
}

func NewPlanner(options PlannerOptions,
	clientset *kubernetes.Clientset,
	log logr.Logger) *PodSetPlanner {

	planner := &PodSetPlanner{
		options:      options,
		clientset:    clientset,
		log:          log,
		quit:         make(chan struct{}),
		podToNodeMap: make(map[ktypes.NamespacedName]string),
	}

	var addFunc func(interface{})

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

	deleteFunc := func(obj interface{}) {
		pod, ok := obj.(*v1.Pod)
		if !ok {
			return
		}

		planner.handlePodDelete(pod)
	}

	planner.nodeLister = initInformers(
		clientset,
		log,
		planner.quit,
		addFunc,
		updateFunc,
		deleteFunc)

	return planner
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

func getEligibleNodeNames(nodeLister listersv1.NodeLister) ([]string, error) {
	eligibleNodeList, err := getEligibleNodes(nodeLister)
	if err != nil {
		return nil, err
	}

	eligibleNodes := make([]string, len(eligibleNodeList))

	for i, eligibleNode := range eligibleNodeList {
		eligibleNodes[i] = eligibleNode.Name
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
		p.handlePodDelete(newPod)
	}
}

func (p *PodSetPlanner) handlePodDelete(pod *v1.Pod) {
	p.Lock()
	defer p.Unlock()
	delete(p.podToNodeMap, ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace})
}

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

	return nil, fmt.Errorf("Cound not find node lister instance for node %s", node)
}

func (p *PodSetPlanner) Stop() {
	close(p.quit)
}

func (p *PodSetPlanner) BuildPlan(podList []*v1.Pod,
	schedulingMap map[ktypes.NamespacedName]PodPlannerInfo) (map[string]string, error) {

	var failedPodList []*v1.Pod

	podNames := make([]string, len(podList))

	for i, pod := range podList {
		podNames[i] = pod.Name
	}

	p.log.V(1).Info("scheduler-planner", "pod-list", podNames)
	assignmentMap := make(map[string]string)

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

		eligibleNodes := schedulingMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}].EligibleNodes
		selectedNode, err := p.findFit(pod, eligibleNodes)
		if err != nil {
			p.log.Error(err, "nodes-not-found", "pod", pod.Name)
			failedPodList = append(failedPodList, pod)
			continue
		}

		p.log.V(1).Info("pod-node-assignment", "pod", pod.Name, "node", selectedNode.Name)

		assignmentMap[pod.Name] = selectedNode.Name
	}

	// check if there are failed pods and requeue them for assignment through planner again
	if len(failedPodList) > 0 {
		p.log.V(1).Info("build-plan", "pods-planning-failed", len(failedPodList))

		for _, pod := range failedPodList {
			p.log.V(1).Info("build-plan", "failed-to-build-plan-for-pod", pod.Name)
		}
	}

	return assignmentMap, nil
}

func (p *PodSetPlanner) BuildSchedulePlan(namespace, podSet string, eligibleNodes []string) (map[string]string, error) {
	p.log.V(1).Info("build-schedule-plan", "podset", podSet, "namespace", namespace)

	pods, err := p.clientset.CoreV1().Pods(namespace).List(context.Background(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("planner.ciena.io/pod-set=%s", podSet)},
	)
	if err != nil {
		p.log.Error(err, "build-schedule-plan-list-pods-error", "podset", podSet)
		return nil, err
	}

	podList := make([]*v1.Pod, len(pods.Items))
	schedulingMap := make(map[ktypes.NamespacedName]PodPlannerInfo)

	for i := range pods.Items {
		podList[i] = &pods.Items[i]
		schedulingMap[ktypes.NamespacedName{Name: pods.Items[i].Name, Namespace: pods.Items[i].Namespace}] = PodPlannerInfo{
			EligibleNodes: eligibleNodes,
		}
	}

	return p.BuildPlan(podList, schedulingMap)
}

// called with constraintpolicymutex held
func (p *PodSetPlanner) getPodNode(pod *v1.Pod) (string, error) {
	if node, ok := p.podToNodeMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}]; !ok {
		return "", fmt.Errorf("Cannot find pod %s node", pod.Name)
	} else {
		return node, nil
	}
}

// called with constraintpolicymutex held
func (p *PodSetPlanner) setPodNode(pod *v1.Pod, nodeName string) {
	p.podToNodeMap[ktypes.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}] = nodeName
}

// Fast version. look up internal cache.
// look up for the node in the lister cache if pod host ip is not set
func (p *PodSetPlanner) GetNodeName(pod *v1.Pod) (string, error) {
	if pod.Status.HostIP == "" {
		return p.getPodNode(pod)
	}

	nodes, err := getEligibleNodes(p.nodeLister)
	if err != nil {
		return "", err
	}

	for _, node := range nodes {

		for _, nodeAddr := range node.Status.Addresses {

			if nodeAddr.Address == pod.Status.HostIP {

				return node.Name, nil
			}
		}
	}

	return "", fmt.Errorf("Pod ip %s not found in node lister cache", pod.Status.HostIP)
}

func (p *PodSetPlanner) findFit(pod *v1.Pod, eligibleNodes []string) (*v1.Node, error) {
	if len(eligibleNodes) == 0 {
		p.log.V(1).Info("no-eligible-nodes-found", "pod", pod.Name)

		return nil, fmt.Errorf("no-eligible-nodes-found-for-pod-%s", pod.Name)
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
