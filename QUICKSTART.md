# Quick start guide

The purpose of this document is to do a short walkthrough of building,
deploying, and using the podset planner resources as a way to help demonstrate
the capability.

# Local Kubernetes cluster

This quick start guide use Kubernetes in Docker (KinD) to create a Kubernetes
cluster for the walkthrough. If you choose to use your own cluster you will
have to adjust the commands to your environment.

Find instructions to install KinD at `https://kind.sigs.k8s.io/docs/user/quick-start/#installation`.

## Create Kubernetes cluster

The following script creates a Kubernetes cluster with a single control plane
node and 3 compute nodes. Additionally, this script creates a local docker
repository and attaches it to the Kubernetes cluster. This script it based on
a script on the KinD website:
`https://kind.sigs.k8s.io/docs/user/local-registry/`.

```bash
#!/bin/sh
set -o errexit

# create registry container unless it already exists
reg_name='kind-registry'
reg_port='5000'
if [ "$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)" != 'true' ]; then
  docker run \
    -d --restart=always -p "127.0.0.1:${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

# create a cluster with the local registry enabled in containerd
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
- role: worker
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:5000"]
EOF

# connect the registry to the cluster network if not already connected
if [ "$(docker inspect -f='{{json .NetworkSettings.Networks.kind}}' "${reg_name}")" = 'null' ]; then
  docker network connect "kind" "${reg_name}"
fi

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
```

# Build and deploy

## Docker registry

Define the docker registry to use with your Kubernetes cluster

```bash
export DOCKER_REGISTRY=localhost:5000
```

## Build and deploy podset planner components

The custom resource definitions (CRDs) are the definitions of the resource
type introduced by the podset planner. The controllers implement the behaviors
for those CRDs.

```bash
make docker-build docker-push install deploy
```

## Checkpoint

At this point in the walkthrough you should be able to see an output similar
to the following when querying Kubernetes for crds, services, and pods.

```bash
$ kubectl get --all-namespaces crd,svc,pod | grep -v kube-system
NAME                                                                              CREATED AT
customresourcedefinition.apiextensions.k8s.io/scheduleplans.planner.ciena.io      2022-02-15T15:51:53Z
customresourcedefinition.apiextensions.k8s.io/scheduletriggers.planner.ciena.io   2022-02-15T15:51:53Z

NAMESPACE        NAME                                                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                  AGE
default          service/kubernetes                                   ClusterIP   10.96.0.1      <none>        443/TCP                  3m40s
planner-system   service/planner-controller-manager-metrics-service   ClusterIP   10.96.163.91   <none>        7443/TCP                 27s
planner-system   service/podset-planner                               ClusterIP   10.96.136.74   <none>        7309/TCP                 26s

NAMESPACE            NAME                                              READY   STATUS    RESTARTS   AGE
local-path-storage   pod/local-path-provisioner-547f784dff-zs72k       1/1     Running   0          3m25s
planner-system       pod/planner-controller-manager-7ddcccdf54-9h8bt   2/2     Running   0          26s
planner-system       pod/podset-planner-6ffc4b5599-bqh2w               1/1     Running   0          26s
planner-system       pod/podset-planner-scheduler-774f687f49-hkvlc     1/1     Running   0          26s
```

# Deploy initial trigger policies for podset scheduler to schedule pods when trigger is active

The following command will create a `ScheduleTrigger` that is attached to
planner-podset via label. It also starts a quiet timer of 2 mins
to transition the trigger from Planning to Schedule state. The scheduler will
only schedule pods for triggers associated with the podset identified by pod label
that are in Schedule or active state.

```bash
kubectl apply -f examples/wt-scheduletrigger.yaml
```

# Deploy workloads with schedulerName podset-planner-scheduler and labeled with planner-podset

```bash
kubectl apply -f examples/wt-deploy.yaml
```

Because no active triggers can be found for the podset, `hello-server` and
`hello-client` pod will be stuck in `Pending` status.

```bash
$ kubectl get po
NAME                            READY   STATUS    RESTARTS   AGE
hello-client-5bb8d986bb-5pxdd   0/1     Pending   0          15s
hello-server-7b5bbf4454-j9x45   0/1     Pending   0          15s
```

# Wait for the schedule trigger to transition to Schedule state from Planning

This will typically happen after the configured trigger timer of 2 mins and retry backoff
associated with default scheduler to invoke the podset-planner-scheduler plugin to
schedule the podset.

```bash
$ kubectl get scheduletriggers/customtrigger -o wide
NAME            STATE
customtrigger   Schedule
```

Pods will now be scheduled to a node and created. This could take up to a
minute more after the trigger configuration as the default scheduler has a
hard-coded retry of unschedulable pods of about 60 seconds.

```bash
$ kubectl get pods -o wide
NAME                            READY   STATUS    RESTARTS   AGE    IP           NODE           NOMINATED NODE   READINESS GATES
hello-client-5bb8d986bb-5pxdd   1/1     Running   0          3m7s   10.244.1.3   kind-worker3   <none>           <none>
hello-server-7b5bbf4454-j9x45   1/1     Running   0          3m7s   10.244.2.3   kind-worker    <none>           <none>
```

Verify the `SchedulePlan` resource is created by the podset planner associated
with the pod identified by the label: planner.ciena.io/pod-set:
planner-podset. The planner-podset is a service that creates the pod
assignments to the nodes to which they are scheduled.

```bash
$ kubectl get sp -o wide
NAME                       PLAN
planner-podset-d4cd7dc68   [{"node":"kind-worker","pod":"hello-server-7b5bbf4454-j9x45"},{"node":"kind-worker3","pod":"hello-client-5bb8d986bb-5pxdd"}]
```

The plan has the node assignments that were planned for the podset. This has
to match the scheduler assignments for the podset. This can be verified by
using the script planner_verify.py under examples directory which should
return success.

```bash
$ examples/planner_verify.py
Planner assignment verification success

```
# Delete the podset deployment.

This should cause the SchedulePlan resource instance to be deleted since all
pods belonging to the podset are deleted.

```bash
$ kubectl delete -f examples/wt-deploy.yaml
$ kubectl get sp -o wide
No resources found in default namespace.
```
