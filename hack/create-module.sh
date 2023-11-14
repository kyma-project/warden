#!/bin/bash

set -eo pipefail

# require envs
KYMA=${KYMA?"Define KYMA env"}
HELM=${HELM?"Define HELM env"}
MODULE_REGISTRY=${MODULE_REGISTRY?"Define MODULE_REGISTRY env"}

# optional envs
SEC_SCANNERS_CONFIG=${SEC_SCANNERS_CONFIG:-}

CHANNEL="${CHANNEL:-fast}"

DEFAULT_NAME=$(cat sec-scanners-config.yaml | grep module-name | sed 's/module-name: //g')
NAME="${NAME:-$DEFAULT_NAME}"

DEFAULT_RELEASE=$(cat sec-scanners-config.yaml | grep rc-tag | sed 's/rc-tag: //g')
RELEASE_SUFFIX="${RELEASE_SUFFIX:-}"
RELEASE="${RELEASE:-$DEFAULT_RELEASE}"
if [[ -n "${RELEASE_SUFFIX}" ]]; then
    RELEASE="$RELEASE-${RELEASE_SUFFIX}"
fi

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
    --sec-scanners-config="$SEC_SCANNERS_CONFIG" \
    --module-config-file=module-config.yaml \
    --registry ${MODULE_REGISTRY} ${CREATE_MODULE_EXTRA_ARGS} --module-archive-version-overwrite
