package namespace

import (
	"github.com/kyma-project/warden/internal/validate"
	warden "github.com/kyma-project/warden/pkg"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// predicate options
type predicateOps struct {
	logger *zap.SugaredLogger
}

// buildNsCreateReject creates function to reject all incoming create events
func buildNsCreateReject(ops predicateOps) func(event.CreateEvent) bool {
	return func(_ event.CreateEvent) bool {
		ops.logger.Debug("omitting incoming create namespace event")
		return false
	}
}

// buildNsDeleteReject creates function to reject all incoming delete events
func buildNsDeleteReject(ops predicateOps) func(event.DeleteEvent) bool {
	return func(_ event.DeleteEvent) bool {
		ops.logger.Debug("omitting incoming delete namespace event")
		return false
	}
}

// buildNsGenericReject creates function to reject all incoming generic events
func buildNsGenericReject(ops predicateOps) func(event.GenericEvent) bool {
	return func(_ event.GenericEvent) bool {
		ops.logger.Debug("omitting incoming generic namespace event")
		return false
	}
}

// newWardenLabelsAdded creates predicate to check if validation label was added
func newWardenLabelsAdded(ops predicateOps) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  buildNsCreateReject(ops),
		DeleteFunc:  buildNsDeleteReject(ops),
		GenericFunc: buildNsGenericReject(ops),
		UpdateFunc:  buildNsUpdated(ops),
	}
}

// buildNsUpdated creates function to check if validation label was added
func buildNsUpdated(ops predicateOps) func(event.UpdateEvent) bool {
	return func(evt event.UpdateEvent) bool {
		oldLabels := evt.ObjectOld.GetLabels()
		newLabels := evt.ObjectNew.GetLabels()
		ops.logger.
			With("oldLabels", oldLabels).
			With("newLabels", newLabels).
			Debug("incoming update namespace event")

		//TODO-CV: check if annotations (also allow list, notary url, strict mode) for user validations was added or changed

		return checkValidationLabel(oldLabels, newLabels, ops.logger)
	}
}

func checkValidationLabel(oldLabels map[string]string, newLabels map[string]string, log *zap.SugaredLogger) bool {
	oldValue := oldLabels[warden.NamespaceValidationLabel]
	newValue := newLabels[warden.NamespaceValidationLabel]

	if !validate.IsSupportedValidationLabelValue(newValue) {
		log.Debugf("validation label: %s is removed or unsupported, omitting update namespace event", warden.NamespaceValidationLabel)
		return false
	}

	if !validate.IsChangedSupportedValidationLabelValue(oldValue, newValue) {
		log.Debugf("validation label: %s value not changed, omitting update namespace event", warden.NamespaceValidationLabel)
		return false
	}
	return true
}
