# [![pallet](./assets/pallet_64x64_r_bg.png)](https://github.com/ciena/pallet)

---

Pallet - Enables delayed scheduling and schedule planning for operator
specified sets of pods.

## Summary

This project introduces a custom resource definition for a `PodSet` that
allows an operator to specify a set of pods via a label match. A `PodSet`
includes an attribute that until set will prevent a pod in the set to be
scheduled to a node. Pods in the set that are already scheduled are not
effected.

When a pod in a set is attempted to be scheduled, an optional planner can
be invoked to select a node onto which the pod should be scheduled. When
a pod set is enabled to be scheduled, and if a plan is available, the pods
in the set are placed on a node based on the associated plan. If no plan
for a set is available pods are scheduled via the default mechanism.
