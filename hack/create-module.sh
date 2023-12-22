#!/bin/bash

set -eo pipefail

# require envs
KYMA=${KYMA?"Define KYMA env"}
HELM=${HELM?"Define HELM env"}

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
${HELM} template --namespace kyma-system warden charts/warden --set admission.enabled=true > warden-manifest.yaml

## generate module-config.yaml template
printf "Generate the module-config.yaml from template\n"
cat module-config-template.yaml |
    sed "s/{{.Name}}/kyma-project.io\/module\/${NAME}/g" |
        sed "s/{{.Channel}}/${CHANNEL}/g" |
            sed "s/{{.Version}}/${RELEASE}/g" > module-config.yaml
