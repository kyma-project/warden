
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.25.0

# TEST_COVER_OUT determines path for the output file with coverage 
TEST_COVER_OUT ?= $(shell pwd)/cover.out

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Operating system architecture
OS_ARCH ?= $(shell uname -m)

# Operating system type
OS_TYPE ?= $(shell uname)

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
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile $(TEST_COVER_OUT)

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/operator/main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross

##@ Module

.PHONY: module-build
module-build: kyma helm ## create moduletemplate and push manifest artifacts
	@KYMA=${KYMA} HELM=${HELM} MODULE_SHA=${MODULE_SHA} ./hack/create-module.sh

##@ CI

.PHONY: ci-module-build
ci-module-build: configure-git-origin module-build
	@echo "=======MODULE TEMPLATE======="
	@cat moduletemplate.yaml
	@echo "============================="

.PHONY: configure-git-origin
configure-git-origin:
#	test-infra does not include origin remote in the .git directory.
#	the CLI is looking for the origin url in the .git dir so first we need to be sure it's not empty
	@git remote | grep '^origin$$' -q || \
		git remote add origin https://github.com/kyma-project/serverless-manager

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

#TODO: clean this, we don't have custom CRD
#.PHONY: install
#install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
#	$(KUSTOMIZE) build config/crd | kubectl apply -f -
#
#.PHONY: uninstall
#uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
#	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

#.PHONY: deploy
#deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
#	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
#	$(KUSTOMIZE) build config/default | kubectl apply -f -
#
#.PHONY: undeploy
#undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
#	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.5
CONTROLLER_TOOLS_VERSION ?= v0.9.2
HELM_VERSION ?= v3.13.1

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# $(call os_error, os-type, os-architecture)
define os_error
$(error Error: unsuported platform OS_TYPE:$2, OS_ARCH:$3; to mitigate this problem set variable $1 with absolute path to the binary compatible with your operating system and architecture)
endef

KYMA_FILE_NAME ?= $(shell ./hack/get_kyma_file_name.sh ${OS_TYPE} ${OS_ARCH})
KYMA_STABILITY ?= unstable
KYMA ?= $(LOCALBIN)/kyma-$(KYMA_STABILITY)
kyma: $(LOCALBIN) $(KYMA) ## Download kyma-cli locally if necessary.
$(KYMA):
	## Detect if operating system
	$(if $(KYMA_FILE_NAME),,$(call os_error, "KYMA" ${OS_TYPE}, ${OS_ARCH}))
	test -f $@ || curl -s -Lo $(KYMA) https://storage.googleapis.com/kyma-cli-$(KYMA_STABILITY)/$(KYMA_FILE_NAME)
	chmod 0100 $(KYMA)

HELM_FILE_NAME ?= $(shell ./hack/get_helm_file_name.sh ${HELM_VERSION} ${OS_TYPE} ${OS_ARCH})
HELM ?= $(LOCALBIN)/helm
helm: $(LOCALBIN) $(HELM) ## Download helm locally if necessary.
$(HELM):
	## Detect if operating system
	$(if $(HELM_FILE_NAME),,$(call os_error, "HELM" ${OS_TYPE}, ${OS_ARCH}))
	curl -Ss https://get.helm.sh/${HELM_FILE_NAME} > $(LOCALBIN)/helm.tar.gz
	tar zxf $(LOCALBIN)/helm.tar.gz -C $(LOCALBIN) --strip-components=1 $(shell tar tzf ala2.tar.gz | grep helm)
	rm $(LOCALBIN)/helm.tar.gz

## Operator

OPERATOR_NAME = warden-operator

build-operator:
	docker build -t $(OPERATOR_NAME) -f ./docker/operator/Dockerfile .

install-operator-k3d: build-operator
	$(eval HASH_TAG=$(shell docker images $(OPERATOR_NAME):latest --quiet))
	docker tag $(OPERATOR_NAME) $(OPERATOR_NAME):$(HASH_TAG)

	k3d image import $(OPERATOR_NAME):$(HASH_TAG) -c kyma
	kubectl set image deployment warden-operator -n default operator=$(OPERATOR_NAME):$(HASH_TAG)

## Admission

ADMISSION_NAME = warden-admission

build-admission:
	docker build -t $(ADMISSION_NAME) -f ./docker/admission/Dockerfile .

install-admission-k3d: build-admission
	$(eval HASH_TAG=$(shell docker images $(ADMISSION_NAME):latest --quiet))
	docker tag $(ADMISSION_NAME) $(ADMISSION_NAME):$(HASH_TAG)

	k3d image import $(ADMISSION_NAME):$(HASH_TAG) -c kyma
	kubectl set image deployment warden-admission -n default admission=$(ADMISSION_NAME):$(HASH_TAG)

## Install

install:
	 helm upgrade --install --wait --set global.config.data.logging.level=debug warden ./charts/warden/
uninstall:
	helm uninstall warden --wait

compile:
	go build -a -o bin/admission ./cmd/admission/main.go
	go build -a -o bin/operator ./cmd/operator/main.go

clean:
	rm bin/admission
	rm bin/operator

run-integration-tests:
	( cd ./tests && go test -tags integration -count=1 ./  )

unit-test:
	go test ./...
