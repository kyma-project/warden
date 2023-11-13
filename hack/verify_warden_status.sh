#!/bin/bash

function get_kyma_status () {
	local number=1
	while [[ $number -le 100 ]] ; do
		echo ">--> checking warden deployment status #$number"
		local STATUS=$(kubectl get deployment warden-operator -n kyma-system -o jsonpath='{.status.conditions[0].status}')
		echo "warden ready: ${STATUS:='UNKNOWN'}"
		[[ "$STATUS" == "True" ]] && return 0
		sleep 5
        	((number = number + 1))
	done

	kubectl get all --all-namespaces
	exit 1
}

get_kyma_status
