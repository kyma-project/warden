package annotations

const (
	// Reject is used to pass status between webhooks
	PodValidationRejectAnnotation = "pods.warden.kyma-project.io/validate-reject"
	InvalidImagesAnnotation       = "pods.warden.kyma-project.io/invalid-images"
	ValidationReject              = "reject"
)
