#!/bin/bash

function get_kyma_file_name () {

	local _VERSION=$1
	local _OS_TYPE=$2
	local _OS_ARCH=$3

	[ "$_OS_TYPE" == "Linux"   ] && [ "$_OS_ARCH" == "x86_64" ] && echo "helm-${_VERSION}-linux-amd64.tar.gz"     ||
	[ "$_OS_TYPE" == "Linux"   ] && [ "$_OS_ARCH" == "arm64"  ] && echo "helm-${_VERSION}-linux-arm64.tar.gz" ||
	[ "$_OS_TYPE" == "Darwin"  ] && [ "$_OS_ARCH" == "x86_64" ] && echo "helm-${_VERSION}-darwin-amd64.tar.gz"    ||
	[ "$_OS_TYPE" == "Darwin"  ] && [ "$_OS_ARCH" == "arm64"  ] && echo "helm-${_VERSION}-darwin-arm64.tar.gz"
}

get_kyma_file_name "$@"
