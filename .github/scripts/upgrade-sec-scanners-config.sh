#!/bin/sh

IMG_VERSION=${IMG_VERSION?"Define IMG_VERSION env"}

yq -i ".protecode[] |= sub(\":main\", \":${IMG_VERSION}\")" sec-scanners-config.yaml
yq -i "del(.rc-tag)" sec-scanners-config.yaml