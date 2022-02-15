# [![pallet](./assets/pallet_64x64_r_bg.png)](https://github.com/ciena/pallet)

---

Pallet - Enables delayed scheduling and schedule planning for operator
specified sets of pods.

## Summary

This project introduces into the Kubernetes environment the ability to
identify a set of pods and delay their scheduling until a trigger is fired.
Additionally, while the set of pods is delayed a scheduling plan can be
created such that the node selection considers all pods in the set of pods
when making selections, thus plans that optimize across the whole set of pods
can be leveraged.

## `PodSet`

The term `PodSet` in this project references the set of pods as identified
by the use of label selectors. This reference is not a formal Kubernetes
resource type.

## Custom Resources

The following are the set of custom resources introduced by this project.

### `ScheduleTrigger`

A `ScheduleTrigger` identifies, via label selection, a set of pods (`PodSet`)
and a trigger state for the associated pods. The trigger state may have the
following values:

- **Planning** - indicates that the pods in the set will not be assigned to
  a node and the pod will remain in the `Pending` state.

- **Schedule** - when the state is set from `Planning` to `Schedule` a
  schedule plan will be created for all `Pending` pods in the trigger's PodSet
  and Pods will be assigned to nodes based on this schedule plan.

The `state` value of a schedule trigger can be manually managed and updated
via the Kubernetes API or CLI. In addition to manual management, the schedule
trigger controller also recognizes a well known label of the format
`planner.ciena.io/quiet-time: <time>`, where `<time>` is a string
representation of the time the controller will wait since the last pod was
created that selects into the trigger's pod set before automatically changing
the state from `Planning` to `Schedule`. The format for time should match a
value that is parsable by Golang's time.Duration implementation.

### `SchedulePlan`

A `SchedulePlan` represents the Pod to Node association for a set of pods
based on the evaluation by a planner.

## `Planner`

A `Planner` is a Kubernetes service and implementation that implements the
planner gRPC API as defined in [planner.proto](./api/planner.proto). A planner
is associated with a schedule trigger by adding the well know label of the
form `planner.ciena.io/<pod-set-label>: enabled`, where `<pod-set-label>` is
replaced with the label associated with the set of pods to plan. There is a
default planner that will be used when a planner specific to a PodSet cannot
be found.

## Adhering to the Plan

The schedule plan for a pod set is created when the trigger's state is set to
`Schedule`. This is done to prevent churn and throw-away processing if a new
plan was generated each time a pod was added to a pod set who's trigger is in
the `Planning` state.

When the trigger is change to `Schedule` then each affected pod will be
assigned to a node as it is passed to the scheduler. When using the schedule
plan a check is first made to verify the the planned node is still valid for
the pod and if not the plan is recalculated. If the planned node is still
valid the pod is assigned to that node.

If a pod is created and matched to a trigger that is already in the `Schedule`
state then the plan is recalculated and the pod immediately assigned to the
planned node.
