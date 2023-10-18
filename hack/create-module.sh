#!/bin/bash

set -eo pipefail

# module config
CHANNEL="${CHANNEL:-fast}"
DEFAULT_NAME=$(cat sec-scanners-config.yaml | grep module-name | sed 's/module-name: //g')
NAME="${NAME:-$DEFAULT_NAME}"
DEFAULT_RELEASE=$(cat sec-scanners-config.yaml | grep rc-tag | sed 's/rc-tag: //g')
RELEASE="${RELEASE:-$DEFAULT_RELEASE}"

CREATE_MODULE_EXTRA_ARGS="${CREATE_MODULE_EXTRA_ARGS:-}"

## generate manifest
printf "Generate manifest to the warden-manifest.yaml file\n"
helm template --namespace kyma-system warden charts/warden > warden-manifest.yaml

## generate module-config.yaml template
printf "Generate the module-config.yaml from template\n"
cat module-config-template.yaml |
    sed "s/{{.Name}}/kyma-project.io\/module\/${NAME}/g" |
        sed "s/{{.Channel}}/${CHANNEL}/g" |
            sed "s/{{.Version}}/${RELEASE}/g" > module-config.yaml

## create module
printf "Create module\n"
kyma-dev alpha create module --path . --output=moduletemplate.yaml \
    --module-config-file=module-config.yaml \
    --registry ${REGISTRY_ADDRESS} ${CREATE_MODULE_EXTRA_ARGS}
