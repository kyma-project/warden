#!/bin/bash

set -eo pipefail

# colors :)
BLUE_COLOR=$(tput setaf 2)
NORMAL_COLOR=$(tput sgr0)

# k3d config
K3D_CLUSTER_NAME="kyma"
K3D_REGISTRY_PORT=5001
K3D_REGISTRY_NAME="${K3D_CLUSTER_NAME}-registry"

# module config
CHANNEL="fast"
NAME=$(cat sec-scanners-config.yaml | grep module-name | sed 's/module-name: //g')
RELEASE=$(cat sec-scanners-config.yaml | grep rc-tag | sed 's/rc-tag: //g')

## generate manifest
printf "${BLUE_COLOR}[ 1 ]${NORMAL_COLOR} Generate manifest to the warden-manifest.yaml file\n"
helm template --namespace kyma-system warden charts/warden > warden-manifest.yaml

## generate module-config.yaml template
printf "${BLUE_COLOR}[ 2 ]${NORMAL_COLOR} Generate the module-config.yaml from template\n"
cat module-config-template.yaml |
    sed "s/{{.Name}}/kyma-project.io\/module\/${NAME}/g" |
        sed "s/{{.Channel}}/${CHANNEL}/g" |
            sed "s/{{.Version}}/${RELEASE}/g" > module-config.yaml

## create k3d cluster and registry
printf "${BLUE_COLOR}[ 3 ]${NORMAL_COLOR} Create k3d cluster and registry\n"
kyma-dev provision k3d --registry-port ${K3D_REGISTRY_PORT} --name ${K3D_CLUSTER_NAME} --ci
kubectl create namespace kyma-system

## create module
printf "\n${BLUE_COLOR}[ 4 ]${NORMAL_COLOR} Create module\n"
kyma-dev alpha create module --path . --output=moduletemplate.yaml \
		--module-config-file=module-config.yaml \
        --registry localhost:${K3D_REGISTRY_PORT} --insecure


## fix moduletemplate (to able pulling artifacts by the k8s internally)
printf "\n${BLUE_COLOR}[ 5 ]${NORMAL_COLOR} Fix moduletemplate\n"
cat moduletemplate.yaml \
	| sed -e "s/remote/control-plane/g" \
		-e "s/${K3D_REGISTRY_PORT}/5000/g" \
	      	-e "s/localhost/k3d-${K3D_REGISTRY_NAME}.localhost/g" \
				> moduletemplate-k3d.yaml

## deploy LM
printf "\n${BLUE_COLOR}[ 6 ]${NORMAL_COLOR} Deploy LM\n"
kyma-dev alpha deploy --ci --force-conflicts

## apply moduletemplate
printf "\n${BLUE_COLOR}[ 7 ]${NORMAL_COLOR} Apply moduletemplate\n"
kubectl apply -f moduletemplate-k3d.yaml

## enable warden module
printf "\n${BLUE_COLOR}[ 8 ]${NORMAL_COLOR} Enable warden module\n"
kyma-dev alpha enable module warden -c fast

## verify
printf "\n${BLUE_COLOR}[ 9 ]${NORMAL_COLOR} Verify\n"
printf "to verify use 'kubectl get kyma -A -w'"
