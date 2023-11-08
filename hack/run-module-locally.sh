#!/bin/bash

# script creates k3d cluster with lifecycle-manager,
# installs self prepared moduletemplate on it and enables warden module

set -eo pipefail


# colors :)
EXTRA_COLOR=$(tput setaf 2)
NORMAL_COLOR=$(tput sgr0)

# k3d config
K3D_CLUSTER_NAME="kyma"
K3D_REGISTRY_PORT=5001
K3D_REGISTRY_NAME="${K3D_CLUSTER_NAME}-registry"
K3D_REGISTRY_ADDRESS="localhost:${K3D_REGISTRY_PORT}"

# programs
export KYMA="bin/kyma-unstable"
export HELM="${HELM:-$(which helm)}"


## create k3d cluster and registry
printf "${EXTRA_COLOR}[ 1 ]${NORMAL_COLOR} Create k3d cluster and registry\n"
${KYMA} provision k3d --registry-port ${K3D_REGISTRY_PORT} --name ${K3D_CLUSTER_NAME} --ci
kubectl create namespace kyma-system

printf "\n${EXTRA_COLOR}[ 2 ]${NORMAL_COLOR} Create module template\n"
export MODULE_REGISTRY=$K3D_REGISTRY_ADDRESS
export CREATE_MODULE_EXTRA_ARGS="--insecure"
./hack/create-module.sh


## fix moduletemplate (to able pulling artifacts by the k8s internally)
printf "\n${EXTRA_COLOR}[ 3 ]${NORMAL_COLOR} Fix moduletemplate\n"
cat moduletemplate.yaml \
	| sed -e "s/remote/control-plane/g" \
		-e "s/${K3D_REGISTRY_PORT}/5000/g" \
	      	-e "s/localhost/k3d-${K3D_REGISTRY_NAME}.localhost/g" \
				> moduletemplate-k3d.yaml

## deploy LM
printf "\n${EXTRA_COLOR}[ 4 ]${NORMAL_COLOR} Deploy LM\n"
${KYMA} alpha deploy --ci --force-conflicts

## apply moduletemplate
printf "\n${EXTRA_COLOR}[ 5 ]${NORMAL_COLOR} Apply moduletemplate\n"
kubectl apply -f moduletemplate-k3d.yaml

## enable warden module
printf "\n${EXTRA_COLOR}[ 6 ]${NORMAL_COLOR} Enable warden module\n"
${KYMA} alpha enable module warden -c fast

## verify
printf "\n${EXTRA_COLOR}[ 7 ]${NORMAL_COLOR} Verify\n"
printf "to verify use 'kubectl get kyma -A -w'"
