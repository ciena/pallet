package client

import (
	plannerv1alpha1 "github.com/ciena/outbound/pkg/apis/scheduleplanner/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctlrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	localSchemeBuilder = runtime.SchemeBuilder{
		plannerv1alpha1.AddToScheme,
	}

	Scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(localSchemeBuilder.AddToScheme(Scheme))
}

func NewSchedulePlannerClient(config *rest.Config, log logr.Logger) (*SchedulePlannerClient, error) {

	genericClient, err := ctlrclient.New(config,
		ctlrclient.Options{
			Scheme: Scheme,
		})

	if err != nil {
		return nil, err
	}

	return NewPlannerClient(genericClient, log), nil
}

func NewScheduleTriggerClient(config *rest.Config, log logr.Logger) (*ScheduleTriggerClient, error) {
	genericClient, err := ctlrclient.New(config,
		ctlrclient.Options{
			Scheme: Scheme,
		})

	if err != nil {
		return nil, err
	}

	return NewTriggerClient(genericClient, log), nil
}
