package namespace

import (
	warden "github.com/kyma-project/warden/pkg"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// predicate options
type predicateOps struct {
	logger *zap.SugaredLogger
}

// buildNsCreateReject creates function to reject all incomming create events
func buildNsCreateReject(ops predicateOps) func(event.CreateEvent) bool {
	return func(_ event.CreateEvent) bool {
		ops.logger.Debug("omitting incomming create namespace event")
		return false
	}
}

// buildNsDeleteReject creates function to reject all incomming delete events
func buildNsDeleteReject(ops predicateOps) func(event.DeleteEvent) bool {
	return func(_ event.DeleteEvent) bool {
		ops.logger.Debug("omitting incomming delete namespace event")
		return false
	}
}

// buildNsGenericReject creates function to reject all incomming generic events
func buildNsGenericReject(ops predicateOps) func(event.GenericEvent) bool {
	return func(_ event.GenericEvent) bool {
		ops.logger.Debug("omitting incomming generic namespace event")
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

func nsValidationLabelSet(labels map[string]string) bool {
	value, found := labels[warden.NamespaceValidationLabel]
	if found && value == warden.NamespaceValidationEnabled {
		return true
	}
	return false
}

// buildNsUpdated creates function to check if validation label was added
func buildNsUpdated(ops predicateOps) func(event.UpdateEvent) bool {
	return func(evt event.UpdateEvent) bool {
		ops.logger.
			With("oldLabels", evt.ObjectOld.GetLabels()).
			With("newLabels", evt.ObjectNew.GetLabels()).
			Debug("incomming update namespace event")

		if nsValidationLabelSet(evt.ObjectOld.GetLabels()) {
			ops.logger.Debugf("validation label '%s' already exists, omitting update namespace event",
				warden.NamespaceValidationLabel)
			return false
		}

		if !nsValidationLabelSet(evt.ObjectNew.GetLabels()) {
			ops.logger.Debugf("validation lable: %s not found, omitting update namespace event",
				warden.NamespaceValidationLabel)
			return false
		}

		return true
	}
}
