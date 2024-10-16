#!/bin/bash

get_all_and_fail() {
	kubectl get all --all-namespaces
	exit 1
}

echo "waiting for deployment"
kubectl wait -n kyma-system --for=condition=Available --timeout=1m deployment warden-operator || get_all_and_fail

echo "waiting for operator"
kubectl wait -n kyma-system --for=condition=Ready --timeout=1m pod --selector "app.kubernetes.io/component"="warden-operator" || get_all_and_fail

echo "waiting for admission"
kubectl wait -n kyma-system --for=condition=Ready --timeout=1m pod --selector "app.kubernetes.io/component"="warden-admission" || get_all_and_fail
