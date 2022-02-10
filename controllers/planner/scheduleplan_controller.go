/*
Copyright 2021 Ciena Corporation..

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

import (
	"context"

	plannerv1alpha1 "github.com/ciena/outbound/pkg/apis/scheduleplanner/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SchedulePlanReconciler reconciles a SchedulePlan object.
type SchedulePlanReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// nolint:lll
// +kubebuilder:rbac:groups=planner.ciena.io,resources=scheduleplans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=planner.ciena.io,resources=scheduleplans/status,verbs=get;update;patch

// Reconcile evaluates updates to scheduleplan
func (r *SchedulePlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("scheduleplan", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulePlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// nolint:wrapcheck
	return ctrl.NewControllerManagedBy(mgr).
		For(&plannerv1alpha1.SchedulePlan{}).
		Complete(r)
}
