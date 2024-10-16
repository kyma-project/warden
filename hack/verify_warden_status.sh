#!/bin/bash

get_all_and_fail() {
	kubectl get all --all-namespaces
	exit 1
}

echo "waiting for deployment"
kubectl wait -n kyma-system --for=condition=Available --timeout=1m deployment warden-operator
if [[ $? -ne 0 ]]; then
	get_all_and_fail
fi

echo "waiting for operator"
kubectl wait -n kyma-system --for=condition=Ready --timeout=1m pod --selector "app.kubernetes.io/component"="warden-operator"
if [[ $? -ne 0 ]]; then
	get_all_and_fail
fi

echo "waiting for admission"
kubectl wait -n kyma-system --for=condition=Ready --timeout=1m pod --selector "app.kubernetes.io/component"="warden-admission"
if [[ $? -ne 0 ]]; then
	get_all_and_fail
fi
