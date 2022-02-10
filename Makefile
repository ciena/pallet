# Copyright 2021 Ciena Corporation.
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Capture version informaton from version file
VERSION=$(shell head -1 ./VERSION)

# Image URL to use all building/pushing image targets
DOCKER_REGISTRY ?= dockerhub.com
DOCKER_TAG ?= $(VERSION)

DOCKER_MANAGER_REPOSITORY ?= podset-planner-controller
DOCKER_SCHEDULER_REPOSITORY ?= podset-planner-scheduler
DOCKER_PLANNER_REPOSITORY ?= podset-planner

ifeq ($(DOCKER_REGISTRY),)
MANAGER_IMG ?= $(DOCKER_MANAGER_REPOSITORY):$(DOCKER_TAG)
SCHEDULER_IMG ?= $(DOCKER_SCHEDULER_REPOSITORY):$(DOCKER_TAG)
PLANNER_IMG ?= $(DOCKER_PLANNER_REPOSITORY):$(DOCKER_TAG)
else
MANAGER_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_MANAGER_REPOSITORY):$(DOCKER_TAG)
SCHEDULER_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_SCHEDULER_REPOSITORY):$(DOCKER_TAG)
PLANNER_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_PLANNER_REPOSITORY):$(DOCKER_TAG)
endif

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Setting up the version information for the build
GO_VERSION=$(shell go version 2>/dev/null | cut -d\  -f 3)
GO_ARCH=$(shell go env GOHOSTARCH)
GO_OS=$(shell go env GOHOSTOS)
BUILD_DATE=$(shell date -u "+%Y-%m-%dT%H:%M:%S%Z")
VCS_REF=$(shell git rev-parse HEAD)
ifeq ($(shell git ls-files --others --modified --deleted --exclude-standard | wc -l | tr -d ' '),0)
VCS_DIRTY=false
else
VCS_DIRTY=true
endif
ifeq ($(shell uname -s | tr '[:upper:]' '[:lower:]'),darwin)
VCS_COMMIT_DATE=$(shell date -j -u -f "%Y-%m-%d %H:%M:%S %z" "$(shell git show -s --format=%ci HEAD)" "+%Y-%m-%dT%H:%M:%S%Z")
else
VCS_COMMIT_DATE=$(shell date -u -d "$(shell git show -s --format=%ci HEAD)" "+%Y-%m-%dT%H:%M:%S%Z")
endif
GIT_TRACKING=$(shell git status -b --porcelain=2 | grep branch\.upstream | awk '{print $$3}' | cut -d/ -f1)
ifeq ($(GIT_TRACKING),)
GIT_TRACKING=origin
endif
# Remove any auth information from URL
VCS_URL=$(shell git remote get-url $(GIT_TRACKING) | sed -e 's/\/\/[-_:@a-zA-Z0-9]*[:@]/\/\//g')

VERSION_LDFLAGS=\
-X github.com/ciena/outbound/controllers/planner.version="$(VERSION)" \
-X github.com/ciena/outbound/controllers/planner.vcsURL="$(VCS_URL)" \
-X github.com/ciena/outbound/controllers/planner.vcsRef="$(VCS_REF)" \
-X github.com/ciena/outbound/controllers/planner.vcsCommitDate="$(VCS_COMMIT_DATE)" \
-X github.com/ciena/outbound/controllers/planner.vcsDirty="$(VCS_DIRTY)" \
-X github.com/ciena/outbound/controllers/planner.goVersion="$(GO_VERSION)" \
-X github.com/ciena/outbound/controllers/planner.os="$(GO_OS)" \
-X github.com/ciena/outbound/controllers/planner.arch="$(GO_ARCH)" \
-X github.com/ciena/outbound/controllers/planner.buildDate="$(BUILD_DATE)"

DOCKER_BUILD_ARGS=\
--build-arg org_label_schema_version="$(VERSION)" \
--build-arg org_label_schema_vcs_url="$(VCS_URL)" \
--build-arg org_label_schema_vcs_ref="$(VCS_REF)" \
--build-arg org_label_schema_vcs_commit_date="$(VCS_COMMIT_DATE)" \
--build-arg org_label_schema_vcs_dirty="$(VCS_DIRTY)" \
--build-arg org_label_schema_build_date="$(BUILD_DATE)"

LDFLAGS=-ldflags "$(VERSION_LDFLAGS)"

.DEFAULT_GOAL:=help

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./pkg/apis/scheduleplanner/..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./pkg/apis/scheduleplanner/..."

.PHONY: generate-plugin-args
generate-plugin-args: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./internal/pkg/scheduler/config.go"

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run status code analysis against code.
	golangci-lint run -c .golangci.yaml --sort-results --timeout 15m ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: build-manager build-planner build-scheduler

.PHONY: build-%
build-%: deps-% fmt vet
	@echo "Building $*"
	go build -o bin/$* $(LDFLAGS) ./cmd/$*

deps-manager: manifests generate

deps-planner: protos

deps-scheduler: generate-plugin-args

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run $(LDFLAGS) ./cmd/controller

.PHONY: docker-build
docker-build: docker-build-manager docker-build-scheduler docker-build-planner ## Build all docker images.

docker-build-manager: ## Build docker image for the podset-planner-controller
	docker build $(DOCKER_BUILD_FLAGS) -t $(MANAGER_IMG) -f build/Dockerfile.manager $(DOCKER_BUILD_ARGS) .

docker-build-scheduler: ## Build docker image for the podset-planner-scheduler
	docker build $(DOCKER_BUILD_FLAGS) -t $(SCHEDULER_IMG) -f build/Dockerfile.scheduler $(DOCKER_BUILD_ARGS) .

docker-build-planner: ## Build docker image for the podset-planner
	docker build $(DOCKER_BUILD_FLAGS) -t $(PLANNER_IMG) -f build/Dockerfile.planner $(DOCKER_BUILD_ARGS) .

.PHONY: docker-push
docker-push: docker-push-manager docker-push-scheduler docker-push-planner ## Push all docker images.

.PHONY: docker-push-manager
docker-push-manager: ## Push the docker image for the podset-planner-controller
	docker push $(MANAGER_IMG)

.PHONY: docker-push-scheduler
docker-push-scheduler: ## Push the docker image for the podset-planner-scheduler
	docker push $(SCHEDULER_IMG)

.PHONY: docker-push-planner
docker-push-planner: ## Push the docker image for the podset-planner
	docker push $(PLANNER_IMG)

ifeq (,$(shell which protoc 2>/dev/null))
	$(warn Please install protobuf compiler : https://grpc.io/docs/protoc-installation)
endif

pkg/apis/planner/planner.pb.go: api/planner.proto
ifeq (,$(shell which protoc 2>/dev/null))
	$(error Please install protobuf compiler: https://grpc.io/docs/protoc-installation)
endif
	protoc -I api/third_party \
		-I=./api \
		--go_out=plugins=grpc:$$(dirname $@) \
		./api/planner.proto

.PHONY: protos ## Generate Go code from the protocol buffer definitions
protos: pkg/apis/planner/planner.pb.go

.PHONY: clean
clean: ## Delete build and/or temporary artifacts
	rm -rf ./bin *.out

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy deploy-manager deploy-scheduler
deploy: deploy-manager deploy-scheduler deploy-planner

deploy-manager:manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(MANAGER_IMG)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

deploy-scheduler:
	sed -e "s;IMAGE_SPEC;$(SCHEDULER_IMG);g" ./deploy/podset-planner-scheduler.yaml | kubectl apply -f -

deploy-planner:
	sed -e "s;IMAGE_SPEC;$(PLANNER_IMG);g" ./deploy/podset-planner.yaml | kubectl apply -f -

.PHONY: undeploy undeploy-manager undeploy-scheduler undeploy-planner
undeploy: undeploy-scheduler undeploy-planner undeploy-manager

undeploy-manager: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

undeploy-scheduler:
	kubectl delete  --ignore-not-found=$(ignore-not-found) -f ./deploy/podset-planner-scheduler.yaml

undeploy-planner:
	kubectl delete  --ignore-not-found=$(ignore-not-found) -f ./deploy/podset-planner.yaml

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
