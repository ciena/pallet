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
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	plannerv1alpha1 "github.com/ciena/outbound/pkg/apis/scheduleplanner/v1alpha1"
	"sync"
)

type ScheduleTriggerCallback func(ctx context.Context, trigger *plannerv1alpha1.ScheduleTrigger)

// ScheduleTriggerReconciler reconciles a ScheduleTrigger object
type ScheduleTriggerReconciler struct {
	client.Client
	Log                logr.Logger
	Scheme             *runtime.Scheme
	scheduleTriggerMap map[types.NamespacedName]*triggerReference
	triggerCallbacks   []ScheduleTriggerCallback
	mutex              sync.Mutex
}

type triggerReference struct {
	timer     *time.Timer
	duration  time.Duration
	timerQuit chan struct{}
}

// transition the trigger to schedule state
func (r *ScheduleTriggerReconciler) transitionToSchedule(parentCtx context.Context,
	triggerRef *plannerv1alpha1.ScheduleTrigger) {
	//lookup the trigger
	var trigger plannerv1alpha1.ScheduleTrigger

	ctx, cancel := context.WithCancel(parentCtx)

	err := r.Client.Get(ctx,
		types.NamespacedName{Name: triggerRef.Name, Namespace: triggerRef.Namespace},
		&trigger)

	if err != nil {
		cancel()
		r.Log.Error(err, "error-transitioning-trigger-to-schedule", "trigger", trigger.Name, "namespace", trigger.Namespace)
		return
	}

	cancel()

	if trigger.Spec.State == "Schedule" {
		return
	}

	ctx, cancel = context.WithCancel(parentCtx)
	defer cancel()

	trigger.Spec.State = "Schedule"
	err = r.Client.Update(ctx, &trigger)

	if err != nil {
		r.Log.Error(err, "error-updating-trigger", "trigger", trigger.Name, "namespace", trigger.Namespace)
		return
	}
}

func (r *ScheduleTriggerReconciler) ScheduleTimerReset(trigger *plannerv1alpha1.ScheduleTrigger) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ref, ok := r.scheduleTriggerMap[types.NamespacedName{Name: trigger.Name, Namespace: trigger.Namespace}]
	if !ok {

		//no timer to reset
		return
	}

	ref.timer.Reset(ref.duration)
}

func (r *ScheduleTriggerReconciler) quietTimer(ref *triggerReference,
	trigger *plannerv1alpha1.ScheduleTrigger) {

	defer ref.timer.Stop()

	for {
		select {
		case <-ref.timerQuit:
			r.Log.V(1).Info("quiet-timer-stop", "trigger", trigger.Name, "namespace", trigger.Namespace)

			return

		case <-ref.timer.C:
			r.Log.V(1).Info("quiet-timer-schedule", "trigger", trigger.Name, "namespace", trigger.Namespace)

			ctx, cancel := context.WithCancel(context.Background())

			r.transitionToSchedule(ctx, trigger)

			cancel()
		}
	}
}

func (r *ScheduleTriggerReconciler) deleteTimer(key types.NamespacedName) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ref, ok := r.scheduleTriggerMap[key]
	if !ok {

		return
	}

	ref.timerQuit <- struct{}{}
	delete(r.scheduleTriggerMap, key)
}

func (r *ScheduleTriggerReconciler) checkAndAllocateScheduleTimer(trigger *plannerv1alpha1.ScheduleTrigger) {
	var durationStr string
	var duration time.Duration

	for k, v := range trigger.Labels {
		if k == "planner.ciena.io/quiet-time" {
			durationStr = v
			break
		}
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := types.NamespacedName{Name: trigger.Name, Namespace: trigger.Namespace}

	if durationStr == "" {
		//stop existing timers
		ref, ok := r.scheduleTriggerMap[key]
		if ok {
			r.Log.V(1).Info("quiet-timer-stop", "trigger", trigger.Name, "namespace", trigger.Namespace)

			ref.timerQuit <- struct{}{}
			delete(r.scheduleTriggerMap, key)
		}

		return
	}

	if d, err := time.ParseDuration(durationStr); err != nil {
		r.Log.Error(err, "error-parsing-duration-string-for-trigger", "duration", durationStr)

		return
	} else {
		duration = d
	}

	if ref, ok := r.scheduleTriggerMap[key]; !ok {
		timer := time.NewTimer(duration)

		ref = &triggerReference{timer: timer, duration: duration, timerQuit: make(chan struct{})}
		r.scheduleTriggerMap[key] = ref

		r.Log.V(1).Info("quiet-timer-start", "trigger", trigger.Name, "duration", duration)
		go r.quietTimer(ref, trigger)
	} else {
		if ref.duration != duration {
			r.Log.V(1).Info("quiet-timer-reset", "duration", duration)

			ref.duration = duration
			ref.timer.Reset(duration)
		}
	}
}

func (r *ScheduleTriggerReconciler) RegisterScheduleTrigger(cb ScheduleTriggerCallback) {
	if cb != nil {
		r.triggerCallbacks = append(r.triggerCallbacks, cb)
	}
}

func (r *ScheduleTriggerReconciler) scheduleTrigger(parentCtx context.Context, trigger *plannerv1alpha1.ScheduleTrigger) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	for _, cb := range r.triggerCallbacks {
		if cb != nil {
			r.Log.V(1).Info("schedule-trigger-invoke", "trigger", trigger.Name)
			cb(ctx, trigger)
		}
	}
}

// +kubebuilder:rbac:groups=planner.ciena.io,resources=scheduletriggers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=planner.ciena.io,resources=scheduletriggers/status,verbs=get;update;patch

func (r *ScheduleTriggerReconciler) Reconcile(parentCtx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.V(1).Info("schedule-trigger", "trigger", req.NamespacedName.Name, "namespace", req.NamespacedName.Namespace)

	// lookup the trigger being reconcile
	var trigger plannerv1alpha1.ScheduleTrigger

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	if err := r.Client.Get(ctx, req.NamespacedName, &trigger); err != nil {

		// delete timer for the trigger
		r.deleteTimer(req.NamespacedName)

		return ctrl.Result{}, nil
	}

	if trigger.Spec.State == "Schedule" {
		r.scheduleTrigger(ctx, &trigger)
	}

	r.checkAndAllocateScheduleTimer(&trigger)
	return ctrl.Result{}, nil
}

func (r *ScheduleTriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.scheduleTriggerMap = make(map[types.NamespacedName]*triggerReference)

	return ctrl.NewControllerManagedBy(mgr).
		For(&plannerv1alpha1.ScheduleTrigger{}).
		Complete(r)
}
