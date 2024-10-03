#!/bin/sh

CHART_VERSION=${CHART_VERSION?"Define CHART_VERSION env"}

for c in $(find charts/warden -name Chart.yaml);
do
    yq -i ".appVersion = \"${CHART_VERSION}\"" $c
    yq -i ".version = \"${CHART_VERSION}\"" $c
done
