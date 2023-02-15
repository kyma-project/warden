package pkg

const (
	NamespaceValidationLabel   = "namespaces.warden.kyma-project.io/validate"
	NamespaceValidationEnabled = "enabled"
)

const (
	PodValidationLabel      = "pods.warden.kyma-project.io/validate"
	ValidationStatusPending = "pending"
	ValidationStatusSuccess = "success"
	// Failed is status when pod validation didn't pass the controller check.
	// This value will go through
	ValidationStatusFailed = "failed"
	//Reject is used to pass status between webhooks
	ValidationStatusReject = "reject"

	PodValidationRejectAnnotation = "pods.warden.kyma-project.io/validate-reject"
	ValidationReject              = "reject"
)
