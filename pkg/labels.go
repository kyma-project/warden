package pkg

const (
	NamespaceValidationLabel = "namespaces.warden.kyma-project.io/validate"
	// Deprecated: use "system" instead
	NamespaceValidationEnabled   = "enabled"
	NamespaceValidationSystem    = "system"
	NamespaceValidationUser      = "user"
	NamespaceNotaryURLAnnotation = "namespaces.warden.kyma-project.io/notary-url"
)

const (
	PodValidationLabel = "pods.warden.kyma-project.io/validate"
	// Pending is status when pod validation result is unknown - probably where is some problem with infrastructure.
	ValidationStatusPending = "pending"
	// Success is status when pod validation passed the controller check.
	ValidationStatusSuccess = "success"
	// Failed is status when pod validation didn't pass the controller check.
	// This value will go through
	ValidationStatusFailed = "failed"
)
