#!/bin/sh

CHART_VERSION=${CHART_VERSION?"Define CHART_VERSION env"}

yq -i ".appVersion = \"${CHART_VERSION}\"" charts/warden/Chart.yaml
yq -i ".version = \"${CHART_VERSION}\"" charts/warden/Chart.yaml 
