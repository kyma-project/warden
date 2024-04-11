#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
PULL_BASE_REF=${PULL_BASE_REF?"Define PULL_BASE_REF env"} # name of the tag
GITHUB_TOKEN=${GITHUB_TOKEN?"Define GITHUB_TOKEN env"} # github token used to upload the template yaml

uploadFile() {
  filePath=${1}
  ghAsset=${2}

  echo "Uploading ${filePath} as ${ghAsset}"
  response=$(curl -s -o output.txt -w "%{http_code}" \
                  --request POST --data-binary @"$filePath" \
                  -H "Authorization: token $GITHUB_TOKEN" \
                  -H "Content-Type: text/yaml" \
                   $ghAsset)
  if [[ "$response" != "201" ]]; then
    echo "Unable to upload the asset ($filePath): "
    echo "HTTP Status: $response"
    cat output.txt
    exit 1
  else
    echo "$filePath uploaded"
  fi
}

echo "Fetching releases"
CURL_RESPONSE=$(curl -w "%{http_code}" -sL \
                -H "Accept: application/vnd.github+json" \
                -H "Authorization: Bearer $GITHUB_TOKEN" \
                https://api.github.com/repos/kyma-project/warden/releases)
JSON_RESPONSE=$(sed '$ d' <<< "${CURL_RESPONSE}")
HTTP_CODE=$(tail -n1 <<< "${CURL_RESPONSE}")
if [[ "${HTTP_CODE}" != "200" ]]; then
  echo "${CURL_RESPONSE}"
  exit 1
fi

echo "Finding release id for: ${PULL_BASE_REF}"
RELEASE_ID=$(jq <<< ${JSON_RESPONSE} --arg tag "${PULL_BASE_REF}" '.[] | select(.tag_name == $ARGS.named.tag) | .id')

echo "Got '${RELEASE_ID}' release id"
if [ -z "${RELEASE_ID}" ]
then
  echo "No release with tag = ${PULL_BASE_REF}"
  exit 1
fi

echo "Updating github release with assets"
UPLOAD_URL="https://uploads.github.com/repos/kyma-project/warden/releases/${RELEASE_ID}/assets"



( cd charts ; tar czf "warden-${PULL_BASE_REF}.tgz" warden)

uploadFile "charts/warden-${PULL_BASE_REF}.tgz" "${UPLOAD_URL}?name=warden-chart.tgz"




