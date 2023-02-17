package admission

const (
	// Reject is used to pass status between webhooks
	PodValidationRejectAnnotation = "pods.warden.kyma-project.io/validate-reject"
	ValidationReject              = "reject"
)
