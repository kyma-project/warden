
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.27.1

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

.PHONY: verify
verify: manifests generate fmt vet envtest ## Verifies formatting and run unit tests
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile $(TEST_COVER_OUT)

compile:
	go build -a -o bin/admission ./cmd/admission/main.go
	go build -a -o bin/operator ./cmd/operator/main.go

clean:
	rm bin/admission
	rm bin/operator

run-integration-tests:##Compile and run integration tests
	( cd ./tests && go test -tags integration -count=1 ./  )

##@ Deployment
## Operator
OPERATOR_NAME = warden-operator

build-operator:
	docker build -t $(OPERATOR_NAME) -f ./docker/operator/Dockerfile .

tag-operator:
	$(eval HASH_TAG=$(shell docker images $(OPERATOR_NAME):latest --quiet))
	docker tag $(OPERATOR_NAME) $(OPERATOR_NAME):$(HASH_TAG)

install-operator-k3d: ##Build local operator and then update the deployment with local image
install-operator-k3d: build-operator tag-operator
	$(eval HASH_TAG=$(shell docker images $(OPERATOR_NAME):latest --quiet))
	k3d image import $(OPERATOR_NAME):$(HASH_TAG) -c kyma
	kubectl set image deployment warden-operator -n default operator=$(OPERATOR_NAME):$(HASH_TAG)

## Admission
ADMISSION_NAME = warden-admission

build-admission:
	docker build -t $(ADMISSION_NAME) -f ./docker/admission/Dockerfile .

tag-admission: build-admission
	docker images $(ADMISSION_NAME):latest --quiet
	$(eval HASH_TAG = $(shell docker images $(ADMISSION_NAME):latest --quiet))
	docker tag $(ADMISSION_NAME) $(ADMISSION_NAME):$(HASH_TAG)

install-admission-k3d: ##Build local admission and then update the deployment with local image
install-admission-k3d: build-admission tag-admission
	$(eval HASH_TAG=$(shell docker images $(ADMISSION_NAME):latest --quiet))
	k3d image import $(ADMISSION_NAME):$(HASH_TAG) -c kyma
	kubectl set image deployment warden-admission -n default admission=$(ADMISSION_NAME):$(HASH_TAG)

install: ##Install helm chart with admission and log level set to debug
	helm upgrade --install --wait --set global.config.data.logging.level=debug --set admission.enabled=true  warden ./charts/warden/

install-local: ##Install helm chart with locally build images
install-local: build-admission tag-admission build-operator tag-operator
	$(eval ADMISSION_HASH=$(shell docker images $(ADMISSION_NAME):latest --quiet))
	k3d image import $(ADMISSION_NAME):$(ADMISSION_HASH) -c kyma

	$(eval OPERATOR_HASH=$(shell docker images $(OPERATOR_NAME):latest --quiet))
	k3d image import $(OPERATOR_NAME):$(OPERATOR_HASH) -c kyma

	helm upgrade --install --wait --set global.config.data.logging.level=debug --set admission.enabled=true \
  --set global.admission.image=$(ADMISSION_NAME):$(ADMISSION_HASH) \
  --set global.operator.image=$(OPERATOR_NAME):$(OPERATOR_HASH) \
		warden ./charts/warden/

uninstall:##Uninstall helm chart
	helm uninstall warden --wait

##@ Module

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

.PHONY: render-manifest
render-manifest: helm ## renders warden-manifest.yaml
	${HELM} template --namespace kyma-system warden charts/warden --set admission.enabled=true > warden-manifest.yaml

.PHONY: render-manifest-for-values
render-manifest-for-values: helm ## renders warden-manifest.yaml for values.yaml file
	${HELM} template --namespace kyma-system warden charts/warden --values values.yaml > warden.yaml

.PHONY: module-config
module-config:
	yq ".channel = \"${CHANNEL}\" | .version = \"${MODULE_VERSION}\""\
    	module-config-template.yaml > module-config.yaml

##@ CI
.PHONY: configure-git-origin
configure-git-origin:
#	test-infra does not include origin remote in the .git directory.
#	the CLI is looking for the origin url in the .git dir so first we need to be sure it's not empty
	@git remote | grep '^origin$$' -q || \
		git remote add origin https://github.com/kyma-project/warden

# deprecated - no longer called on prow ?
.PHONY: k3d-integration-test
k3d-integration-test: 
	@IMG_VERSION="main" IMG_DIRECTORY="prod" make replace-chart-images run-on-k3d verify-status run-integration-tests

.PHONY: verify-on-cluster
verify-on-cluster:
	@echo "this target requires IMG_VERSION and IMG_DIRECTORY envs"
	@IMG_VERSION=${IMG_VERSION} IMG_DIRECTORY=${IMG_DIRECTORY} make replace-chart-images run-on-cluster verify-status run-integration-tests

.PHONY: create-k3d
create-k3d: ## Create k3d
	${KYMA} provision k3d --ci -p 6080:8080@loadbalancer -p 6433:8433@loadbalancer
	kubectl create namespace kyma-system

.PHONY: run-on-k3d
run-on-k3d: kyma create-k3d configure-git-origin render-manifest 
	kubectl apply -f warden-manifest.yaml

.PHONY: run-on-cluster
run-on-cluster: configure-git-origin render-manifest
	kubectl create namespace kyma-system
	kubectl apply -f warden-manifest.yaml

.PHONY: verify-status
verify-status:
	@./hack/verify_warden_status.sh

.PHONY: replace-chart-images
replace-chart-images:
	yq -i ".global.operator.image = \"europe-docker.pkg.dev/kyma-project/${IMG_DIRECTORY}/warden/operator:${IMG_VERSION}\"" charts/warden/values.yaml
	yq -i ".global.admission.image = \"europe-docker.pkg.dev/kyma-project/${IMG_DIRECTORY}/warden/admission:${IMG_VERSION}\"" charts/warden/values.yaml
	@echo "==== Local Changes ===="
	yq '.global.operator.image' charts/warden/values.yaml
	yq '.global.admission.image' charts/warden/values.yaml
	@echo "==== End of Local Changes ===="

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
CONTROLLER_TOOLS_VERSION ?= v0.14.0
HELM_VERSION ?= v3.13.1

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen $(CONTROLLER_GEN)
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test "$(shell ${LOCALBIN}/controller-gen --version)" = "Version: ${CONTROLLER_TOOLS_VERSION}" || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

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
