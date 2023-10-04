#!/bin/bash

NAME=$(cat sec-scanners-config.yaml | grep module-name | sed 's/module-name: //g')
CHANNEL="fast"
RELEASE=$(cat sec-scanners-config.yaml | grep rc-tag | sed 's/rc-tag: //g')

# generate manifest
helm template warden charts/warden > warden-manifest.yaml

# generate module-config.yaml template
cat module-config-template.yaml |
    sed "s/{{.Name}}/${NAME}/g" |
        sed "s/{{.Channel}}/${CHANNEL}/g" |
            sed "s/{{.Version}}/${RELEASE}/g" > module-config.yaml
