#!/bin/bash

set -eo pipefail

# requirements
KYMA=${KYMA?"Define KYMA env"}
HELM=${HELM?"Define HELM env"}
MODULE_REGISTRY=${MODULE_REGISTRY?"Define MODULE_REGISTRY env"}

# module config
CHANNEL="${CHANNEL:-fast}"
DEFAULT_NAME=$(cat sec-scanners-config.yaml | grep module-name | sed 's/module-name: //g')
NAME="${NAME:-$DEFAULT_NAME}"
DEFAULT_RELEASE=$(cat sec-scanners-config.yaml | grep rc-tag | sed 's/rc-tag: //g')

if [[ -n "$MODULE_SHA" ]]; then
    DEFAULT_RELEASE="$DEFAULT_RELEASE-$MODULE_SHA"
fi

RELEASE="${RELEASE:-$DEFAULT_RELEASE}"

CREATE_MODULE_EXTRA_ARGS="${CREATE_MODULE_EXTRA_ARGS:-}"

## generate manifest
printf "Generate manifest to the warden-manifest.yaml file\n"
${HELM} template --namespace kyma-system warden charts/warden > warden-manifest.yaml

## generate module-config.yaml template
printf "Generate the module-config.yaml from template\n"
cat module-config-template.yaml |
    sed "s/{{.Name}}/kyma-project.io\/module\/${NAME}/g" |
        sed "s/{{.Channel}}/${CHANNEL}/g" |
            sed "s/{{.Version}}/${RELEASE}/g" > module-config.yaml

## create module
printf "Create module\n"
${KYMA} alpha create module --path . --output=moduletemplate.yaml \
    --module-config-file=module-config.yaml \
    --registry ${MODULE_REGISTRY} ${CREATE_MODULE_EXTRA_ARGS}
