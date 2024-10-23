package namespace

import (
	"context"
	"fmt"
	sysruntime "runtime"

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
			newWardenLabelsAdded(predicateOps{logger: r.Log}),
		)).
		Complete(r)
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=list;update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=watch

var reconciliation_count = 0

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconciliation_count++
	reqUUID := uuid.New().String()

	logger := r.Log.With("req", req).With("req-id", reqUUID)
	logger.Warnf("reconciliation started")

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
			Warnf("validation lable: %s not found, omitting update namespace event", warden.NamespaceValidationLabel)
		return result, nil
	}

	PrintMemUsage(fmt.Sprintf("OOM - before podList (%d)", reconciliation_count))
	var labelCount int
	var podCount int
	// fetch all the pods in the given namespace
	opts := &client.ListOptions{
		Namespace: req.Name,
		Limit:     17,
	}
	for {
		var pods corev1.PodList
		err := r.List(ctx, &pods, opts)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "while fetching list of pods")
		}

		logger.With("pod-count", len(pods.Items)).Warn("pod fetching succeeded")
		podCount += len(pods.Items)

		// label all pods with validation pending; requeue in case any error
		for i := range pods.Items {
			loopLogger := logger.With("name", pods.Items[i].Name).With("namespace", pods.Items[i].Namespace)
			if err := labelWithValidationPending(ctx, &pods.Items[i], r.Patch); err != nil {
				loopLogger.Errorf("pod labeling error: %s", err)
				continue
			}

			labelCount++
			loopLogger.With("name", pods.Items[i].Name).With("namespace", pods.Items[i].Namespace).
				Debugf("pod labeling succeeded %d/%d", i, len(pods.Items))
		}
		logger.Warnf("pods.continue: `%s`; pods.remainingitemscount: %d", pods.Continue, pods.RemainingItemCount)
		// there is no more objects
		if pods.Continue == "" {
			break
		}
		opts.Continue = pods.Continue
	}
	PrintMemUsage(fmt.Sprintf("OOM - after podList (%d)", reconciliation_count))

	logger.Warnf("%d/%d pod[s] labeled", labelCount, podCount)

	result := ctrl.Result{
		//TODO: what it is mean?
		Requeue: podCount != labelCount,
	}

	logger.With("result", result).Debug("reconciliation finished")

	return result, nil
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage(message string) {
	var m sysruntime.MemStats
	sysruntime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("=====================================\n")
	fmt.Printf("%s\n", message)
	fmt.Printf("-------------------------\n")
	fmt.Printf("Alloc = %v MiB\n", bToMb(m.Alloc))
	fmt.Printf("\tStackInuse = %v\n", bToMb(m.StackInuse))
	fmt.Printf("\tHeapInuse = %v\n", bToMb(m.HeapInuse))
	fmt.Printf("\tHeapIdle = %v\n", bToMb(m.HeapIdle))
	fmt.Printf("-------------------------\n")
	fmt.Printf("Sys = %v MiB\n", bToMb(m.Sys))
	fmt.Printf("\tHeapSys = %v MiB\n", bToMb(m.HeapSys))
	fmt.Printf("\tStackSys = %v MiB\n", bToMb(m.StackSys))
	fmt.Printf("\tOtherSys = %v MiB\n", bToMb(m.OtherSys))
	fmt.Printf("-------------------------\n")
	fmt.Printf("NumGC = %v\n", m.NumGC)
	fmt.Printf("HeapObjects = %v\n", m.HeapObjects)
	fmt.Printf("=====================================\n")
}
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
