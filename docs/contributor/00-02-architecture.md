# Warden Internal Architecture

## Components

Warden contains controllers and webhooks that are responsible for verification.

### Pod Controller

This controller checks Pods during the Pod update and periodic reconciliation.
The Pod controller uses an operation filter to exclude operations that are not relevant for verification.
For example, it does not verify Pods that changed unnecessary fields. Pods that changed images or validation status are verified.

### Namespace Controller

Namespace controller watches for namespace changes like changing the `namespaces.warden.kyma-project.io/validate` label annotations with user mode configuration.

### Mutating Webhook

Mutating webhook adds the `pods.warden.kyma-project.io/validate` label to the Pod.
It does the same operations as the Pod controller but additionally could decide to reject the Pod creation or update. For this purpose, it adds the internal `pods.warden.kyma-project.io/validate-reject: reject` annotation to the Pod.
This webhook also uses the strictMode configuration to decide if the Pod should be rejected when the Notary server is unavailable.

Mutating webhook based on the current status of the Pod skips verification if the Pod is updating and its status is `pending` or `failed`.
It does this because the Pod controller previously set the status, and it is not necessary to verify the Pod again.

### Validating Webhook

Validation webhook only checks the `pods.warden.kyma-project.io/validate-reject: reject` annotation and rejects the Pod if it is present.

## Image Verification

Warden verifies that images used in Pods are signed by the Notary server by comparing the digest of the image in the Docker registry with the digest stored in the Notary server.
Warden checks if the checked artifact is an image or a list of images. If it is a list of images, Warden checks digest stored in Notary against the digest of the whole list. This is necessary, since calling the `remote.Image(ref)` method on a list of images returns only data for the first image in the list, which would allow tampering with the image list.

If the artifact is an image, Warden checks the digest stored in Notary against the digest of the image. If that check fails, Warden makes a deprecated check against the image manifest digest. This check will be removed in the future.
