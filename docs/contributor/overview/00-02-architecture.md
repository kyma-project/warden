# Warden internal architecture

## Components

Warden contains controllers and webhooks that are responsible for verification.

### Pod controller

This controller checks pods during pod update and periodic reconciliation.
Pod controller use operation filter to exclude operations that are not relevant for verification.
For example, it does not verify pods that changed unnecessary fields. Verified are pods that have changed images or validation status.

### Namespace controller

Namespace controller watches for namespace changes like changing `namespaces.warden.kyma-project.io/validate` label annotations with user mode configuration.

### Mutating webhook

Mutating webhook is responsible for adding the `pods.warden.kyma-project.io/validate` label to the pod.
It does the same operations like Pod controller but additionally could decide to reject the pod creation or update. For this purpose, it adds internal `pods.warden.kyma-project.io/validate-reject: reject` annotation to the pod.
This webhook also use strictMode configuration to decide if pod should be rejected when Notary server is not available.

Mutating webhook based on current status of the pod skips verification if pod is updating and status is `pending` or `failed`.
It do this because status was previously set by the Pod controller and it is not necessary to verify the pod again.

### Validating webhook

Validation webhook only checks `pods.warden.kyma-project.io/validate-reject: reject` annotation and rejects the pod if it is present.

## Image verification

Warden verifies that images used in pods are signed by a Notary server by comparing the digest of the image in the Docker registry with the digest stored in the Notary server.
Warden checks if the artifact it is checking is an image or a list of images. If it is a list of images, Warden checks digest stored in Notary against the digest of the whole list. This is necessary, since calling `remote.Image(ref)` method on a list of images returns only data for the first image in the list, which would allow tampering with the image list.

If the artifact is an image, Warden checks the digest stored in Notary against the digest of the image. If that check fails, Warden will make a deprecated check against the image manifest digest. This check will be removed in the future.
