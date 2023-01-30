package namespace

import (
	warden "github.com/kyma-project/warden/pkg"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type predicateOps struct {
	logger *zap.SugaredLogger
}

func buildNsCreateReject(ops predicateOps) func(event.CreateEvent) bool {
	return func(_ event.CreateEvent) bool {
		ops.logger.Debug("omitting incomming create namespace event")
		return false
	}
}

func buildNsDeleteReject(ops predicateOps) func(event.DeleteEvent) bool {
	return func(_ event.DeleteEvent) bool {
		ops.logger.Debug("omitting incomming delete namespace event")
		return false
	}
}

func buildNsGenericReject(ops predicateOps) func(event.GenericEvent) bool {
	return func(_ event.GenericEvent) bool {
		ops.logger.Debug("omitting incomming generic namespace event")
		return false
	}
}

func newWardenLabelsAdded(ops predicateOps) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  buildNsCreateReject(ops),
		DeleteFunc:  buildNsDeleteReject(ops),
		GenericFunc: buildNsGenericReject(ops),
		UpdateFunc:  buildNsUpdated(ops),
	}
}

func buildNsUpdated(ops predicateOps) func(event.UpdateEvent) bool {
	return func(evt event.UpdateEvent) bool {
		ops.logger.
			With("oldLabels", evt.ObjectOld.GetLabels()).
			With("newLabels", evt.ObjectNew.GetLabels()).
			Debug("incomming update namespace event")

		oldValue, found := evt.ObjectOld.GetLabels()[warden.NamespaceValidationLabel]
		if found && oldValue == warden.NamespaceValidationEnabled {
			ops.logger.Debug("omitting update namespace event")
			return false
		}

		newValue, found := evt.ObjectNew.GetLabels()[warden.NamespaceValidationLabel]
		if !found || newValue != warden.NamespaceValidationEnabled {
			ops.logger.Debug("omitting update namespace event")
			return false
		}

		return true
	}
}
