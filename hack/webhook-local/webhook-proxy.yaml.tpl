apiVersion: v1
kind: Pod
metadata:
  labels:
    app: warden-admission
  name: webhook-proxy
  namespace: default
spec:
  containers:
  - image: ${WEBHOOK_PROXY_NAME}:${HASH_TAG}
    imagePullPolicy: Never
    name: webhook-proxy
    resources: {}
  dnsPolicy: ClusterFirst
  restartPolicy: Always
status: {}
