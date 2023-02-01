package namespace

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/google/uuid"
	warden "github.com/kyma-project/warden/pkg"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodReconciler reconciles a Pod object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    *zap.SugaredLogger
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.And(
			newWardenLabelsAdded(predicateOps{logger: r.Log}),
		)).
		Complete(r)
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=list;update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqUUID := uuid.New().String()

	logger := r.Log.With("req", req).With("reqUUID", reqUUID)
	logger.Info("reconciliation started")

	var instance corev1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		logger.Error("unable to fetch namespace, requeueing")
		return ctrl.Result{
			Requeue: true,
		}, client.IgnoreNotFound(err)
	}

	if !nsValidationLabelSet(instance.Labels) {
		var result ctrl.Result
		logger.With("result", result).
			Debugf("validation lable: %s not found, omitting update namespace event", warden.NamespaceValidationLabel)
		return result, nil
	}

	// fetch all the pods in the given namespace
	var pods corev1.PodList
	if err := r.List(ctx, &pods, &client.ListOptions{Namespace: req.Name}); err != nil {
		return ctrl.Result{}, err
	}

	logger.With("podCount", len(pods.Items)).Debug("pod fetching succeeded")

	var labledCount int
	// label all pods with validation pending; requeue in case any error
	for i, pod := range pods.Items {
		if err := labelWithValidationPending(ctx, &pod, r.Patch); err != nil {
			logger.Errorf("pod labeling error: %s", err)
			continue
		}

		labledCount++
		logger.With("name", pod.Name).With("namespace", pod.Namespace).
			Debugf("pod labeling succeeded %d/%d", i, len(pods.Items))
	}

	logger.Debugf("%d/%d pod[s] labeled", labledCount, len(pods.Items))

	result := ctrl.Result{
		Requeue: len(pods.Items) != labledCount,
	}

	logger.With("result", result).Debug("reconciliation finished")

	return result, nil
}
