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

1. Enable Warden validation on a namespace by adding the required `namespaces.warden.kyma-project.io/notary-url` annotation and the `namespaces.warden.kyma-project.io/validate: user` label to the namespace.
   ```bash
   kubectl annotate namespace <namespace-name> namespaces.warden.kyma-project.io/notary-url=<notary-url>
   kubectl label namespace <namespace-name> namespaces.warden.kyma-project.io/validate=user
   ```
   > [!WARNING]
   > If you add label before annotation, Warden will not validate images in the namespace.
2. Create pod with signed image.
   ```bash
   kubectl run <pod-name>  --namespace <namespace-name> --image <signed-image>
   ```
3. Verify that the pod has the `pods.warden.kyma-project.io/validate: success` label.
   ```bash
   kubectl get pods <pod-name> --namespace <namespace-name> -o jsonpath='{.metadata.labels.pods\.warden\.kyma-project\.io/validate}'
   ```
   The output should be `success` if validation has succeeded.
4. Create pod with unsigned image.
   ```bash
    kubectl run <pod-name>  --namespace <namespace-name> --image <unsigned-image>
   ```
5. Verify that the pod has the `pods.warden.kyma-project.io/validate: failed` label.
   ```bash
   kubectl get pods <pod-name> --namespace <namespace-name> -o jsonpath='{.metadata.labels.pods\.warden\.kyma-project\.io/validate}'
   ```
   The output should be `failed` if validation has failed.
