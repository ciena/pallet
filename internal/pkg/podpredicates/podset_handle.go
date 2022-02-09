package podpredicates

import (
	"context"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
)

type PodSetHandle interface {
	// ClientSet returns a k8s clientset
	ClientSet() clientset.Interface

	// List returns a list of pods for the podset on the node
	List(context.Context, *v1.Node) ([]*v1.Pod, error)
}
