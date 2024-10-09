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

### Validating webhook

Validation webhook only checks `pods.warden.kyma-project.io/validate-reject: reject` annotation and rejects the pod if it is present.