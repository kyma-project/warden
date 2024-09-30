package namespace

import (
	"context"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/pkg/errors"
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
			wardenPredicate(predicateOps{logger: r.Log}),
		)).
		Complete(r)
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=list;update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqUUID := uuid.New().String()

	logger := r.Log.With("req", req).With("req-id", reqUUID)
	logger.Info("reconciliation started")

	var instance corev1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		logger.Error("unable to fetch namespace, requeueing")
		return ctrl.Result{
			Requeue: true,
		}, client.IgnoreNotFound(err)
	}

	if !validate.IsSupportedValidationLabelValue(instance.Labels[warden.NamespaceValidationLabel]) {
		var result ctrl.Result
		logger.With("result", result).
			Debugf("validation label: %s not found or not supported value, omitting update namespace event", warden.NamespaceValidationLabel)
		return result, nil
	}

	// fetch all the pods in the given namespace
	var pods corev1.PodList
	if err := r.List(ctx, &pods, &client.ListOptions{Namespace: req.Name}); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "while fetching list of pods")
	}

	logger.With("pod-count", len(pods.Items)).Debug("pod fetching succeeded")

	var labelCount int
	// label all pods with validation pending; requeue in case any error
	for i, pod := range pods.Items {
		loopLogger := logger.With("name", pod.Name).With("namespace", pod.Namespace)
		if err := labelWithValidationPending(ctx, &pod, r.Patch); err != nil {
			loopLogger.Errorf("pod labeling error: %s", err)
			continue
		}

		labelCount++
		loopLogger.With("name", pod.Name).With("namespace", pod.Namespace).
			Debugf("pod labeling succeeded %d/%d", i, len(pods.Items))
	}

	logger.Debugf("%d/%d pod[s] labeled", labelCount, len(pods.Items))

	result := ctrl.Result{
		Requeue: len(pods.Items) != labelCount,
	}

	logger.With("result", result).Debug("reconciliation finished")

	return result, nil
}
