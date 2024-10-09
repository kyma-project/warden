# Warden flow

Warden can work on pod create, and update operations. It can also reconcile pods that are already running in the cluster.

## Pod create and update operations

Warden validates images in pods during pod create and update operations. It checks if the images are signed by a Notary server. If the images are not signed, Warden rejects the pod creation or update.

![Pod create and update flow](../../assets/user_operations.svg)

Strict mode determines if Warden should conditionally approve pod when the Notary server is not available. If strict mode is enabled, Warden rejects all images when the Notary server is unavailable. If strict mode is disabled, Warden adds the `pods.warden.kyma-project.io/validate: pending` label to the pod and retries validation later.

## Pod reconciliation

Warden will periodically reconcile pods that are already running in the cluster. It checks if the images in the pods are signed by a Notary server. If the images are not signed, Warden adds the `pods.warden.kyma-project.io/validate: failed` label to the pod and retries validation later.

![Pod reconciliation flow](../../assets/user_reconcile.svg)

Reconciliation can be triggered periodically, or when the namespace is updated.

## Namespace update

When a namespace has the `namespaces.warden.kyma-project.io/validate: user` label and the `namespaces.warden.kyma-project.io/notary-url` annotation, Warden will protect all pods in the namespace.

When a protected namespace is updated, Warden will schedule a reconcile on all pods in the namespace. The [reconciliation process](#pod-reconciliation) is described above.

See [available namespace labels and annotations](tutorial/01-10-configure-user.md) for a complete list of configuration options.
