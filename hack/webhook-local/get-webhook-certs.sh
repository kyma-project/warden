#!/bin/bash
set -e

CERT_DIR="/tmp/k8s-webhook-server/serving-certs"

echo "Getting pem files for webhook"
CERT_PEM_FILE=$(kubectl get secret -n default warden-webhook -ojson | jq '.data."server-cert.pem"' | tr -d "\"" | base64 -d)
KEY_PEM_FILE=$(kubectl get secret -n default warden-webhook -ojson | jq '.data."server-key.pem"' | tr -d "\""  | base64 -d)

echo "Writing pem files to: $CERT_DIR"
mkdir -p "$CERT_DIR"
echo "$CERT_PEM_FILE" > $CERT_DIR/"server-cert.pem"
echo "$KEY_PEM_FILE" > $CERT_DIR/"server-key.pem"

echo "Pem files written"
ls -l $CERT_DIR

