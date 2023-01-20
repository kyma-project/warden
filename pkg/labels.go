package pkg

const (
	NamespaceValidationLabel   = "namespaces.warden.kyma-project.io/validate"
	NamespaceValidationEnabled = "enabled"
)

const (
	PodValidationLabel      = "pods.warden.kyma-project.io/validate"
	ValidationStatusPending = "pending"
	ValidationStatusSuccess = "success"
	ValidationStatusFailed  = "failed"
	ValidationStatusReject  = "reject"
)
