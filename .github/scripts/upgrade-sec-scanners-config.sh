#!/bin/sh

IMG_VERSION=${IMG_VERSION?"Define IMG_VERSION env"}

yq -i ".bdba[0] = \"europe-docker.pkg.dev/kyma-project/prod/warden/operator:${IMG_VERSION}\"" sec-scanners-config.yaml
yq -i ".bdba[1] = \"europe-docker.pkg.dev/kyma-project/prod/warden/admission:${IMG_VERSION}\"" sec-scanners-config.yaml
yq -i "del(.rc-tag)" sec-scanners-config.yaml