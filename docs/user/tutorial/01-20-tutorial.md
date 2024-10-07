# Use Warden on a user namespace

This tutorial shows how to enable Warden validation on a namespace, and validate integrity of images in pods.

## Prerequisites

Before you start, ensure that you have:

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/).
- [Warden](/warden/user/00-00-overview-warden.md) enabled on your cluster.
- A namespace where you want to enable Warden validation.
- [Notary](https://github.com/notaryproject/notary) server instance.
- Signed images.

## Steps

1. Set up folloiwng environment variables:
   ```bash
   export NAMESPACE=<namespace>
   export SIGNED_POD_NAME=<signed-pod-name>
   export SIGNED_IMAGE=<signed-image>
   export UNSIGNED_POD_NAME=<unsigned-pod-name>
   export UNSIGNED_IMAGE=<unsigned-image>
   export NOTARY_URL=<notary-url>
   ```
2. Enable Warden validation on a namespace by adding the required `namespaces.warden.kyma-project.io/notary-url` annotation and the `namespaces.warden.kyma-project.io/validate: user` label to the namespace.
   ```bash
   kubectl annotate namespace $NAMESPACE namespaces.warden.kyma-project.io/notary-url=$NOTARY_URL
   kubectl label namespace $NAMESPACE namespaces.warden.kyma-project.io/validate=user
   ```
   > [!WARNING]
   > If you add label before annotation, Warden will not validate images in the namespace.
3. Create pod with signed image.
   ```bash
   kubectl run $SIGNED_POD_NAME --namespace $NAMESPACE --image $SIGNED_IMAGE
   ```
4. Verify that the pod has the `pods.warden.kyma-project.io/validate: success` label.
   ```bash
   kubectl get pods $SIGNED_POD_NAME --namespace $NAMESPACE -o jsonpath='{.metadata.labels.pods\.warden\.kyma-project\.io/validate}'
   ```
   The output should be `success` if validation has succeeded.
5. Try to create pod with unsigned image.
   ```bash
    kubectl run $UNSIGNED_POD_NAME --namespace $NAMESPACE --image $UNSIGNED_IMAGE
   ```
   You should get the following error:
   `Error from server (Forbidden): admission webhook "validation.webhook.warden.kyma-project.io" denied the request: Pod images nginx:latest validation failed`
