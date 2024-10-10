# Warden Flow

## Pod Create and Update Operations

Warden validates images in Pods during the Pod create and update operations. It checks if the images are signed by the Notary server. If the images are not signed, Warden rejects the Pod creation or update.

![Pod create and update flow](../assets/user_operations.svg)

Strict mode determines if Warden must conditionally approve Pod when the Notary server is unavailable. If strict mode is enabled, Warden rejects all images when the Notary server is unavailable. If strict mode is disabled, Warden adds the `pods.warden.kyma-project.io/validate: pending` label to the Pod and retries validation later.

## Pod Reconciliation

Warden periodically reconciles Pods that are already running in the cluster. It checks if the images in the Pods are signed by the Notary server. If the images are not signed, Warden adds the `pods.warden.kyma-project.io/validate: failed` label to the Pod and retries validation later.

![Pod reconciliation flow](../assets/user_reconcile.svg)

Reconciliation can be triggered periodically or when the namespace is updated.

## Namespace Update

When a namespace has the `namespaces.warden.kyma-project.io/validate: user` label and the `namespaces.warden.kyma-project.io/notary-url` annotation, Warden protects all Pods in the namespace.

When a protected namespace is updated, Warden schedules a reconciliation on all Pods in the namespace. See [Pod Reconciliation](#pod-reconciliation).

For a complete list of configuration options, see [User Configuration](01-10-configure-user.md).

## Verification Results

Warden adds the labels `pods.warden.kyma-project.io/validate` to Pods to indicate the verification status:
 * `success` - the Pod passed the controller check.
 * `failed` - the Pod did not pass the controller check.
 * `pending` - the verification status is unknown, and the Pod is waiting for validation.
