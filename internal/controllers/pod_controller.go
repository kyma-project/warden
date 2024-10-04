/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/kyma-project/warden/internal/helpers"

	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/pkg"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type PodReconcilerConfig struct {
	RequeueAfter time.Duration
}

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client                   client.Client
	scheme                   *runtime.Scheme
	systemValidator          validate.PodValidator
	userValidationSvcFactory validate.ValidatorSvcFactory
	baseLogger               *zap.SugaredLogger
	PodReconcilerConfig
}

func NewPodReconciler(client client.Client, scheme *runtime.Scheme,
	validator validate.PodValidator, userValidationSvcFactory validate.ValidatorSvcFactory,
	reconcileCfg PodReconcilerConfig, logger *zap.SugaredLogger) *PodReconciler {
	return &PodReconciler{
		client:                   client,
		scheme:                   scheme,
		systemValidator:          validator,
		userValidationSvcFactory: userValidationSvcFactory,
		baseLogger:               logger,
		PodReconcilerConfig:      reconcileCfg,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return r.isValidationEnabledForNS(e.Object.GetNamespace())
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				// don't trigger if there is no change
				if e.ObjectOld.GetResourceVersion() == e.ObjectNew.GetResourceVersion() {
					return false
				}
				// don't trigger if namespace validation is not enabled
				if !r.isValidationEnabledForNS(e.ObjectNew.GetNamespace()) {
					return false
				}
				// trigger, if there is container images including init container changes
				if areImagesChanged(e.ObjectOld.DeepCopyObject().(*corev1.Pod), e.ObjectNew.DeepCopyObject().(*corev1.Pod)) {
					return true
				}
				// trigger, only if validation label is failed or missing
				if e.ObjectOld.GetLabels()[pkg.PodValidationLabel] != pkg.ValidationStatusSuccess ||
					e.ObjectNew.GetLabels()[pkg.PodValidationLabel] != pkg.ValidationStatusSuccess {
					return true
				}
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(genericEvent event.GenericEvent) bool {
				return false
			},
		}).
		Complete(r)
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqUUID := uuid.New().String()
	logger := r.baseLogger.With("req", req).With("req-id", reqUUID)
	ctxLogger := helpers.LoggerToContext(ctx, logger)
	logger.Debugf("reconciliation started")

	var pod corev1.Pod
	if err := r.client.Get(ctxLogger, req.NamespacedName, &pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	result, err := r.checkPod(ctxLogger, &pod)
	if err != nil {
		return ctrl.Result{}, err
	}

	shouldRetry := ctrl.Result{RequeueAfter: r.RequeueAfter}
	switch result {
	case validate.Valid:
		logger.Info("pod validated successfully")
		shouldRetry = ctrl.Result{}
	case validate.Invalid:
		logger.Info("pod validation failed")
		shouldRetry = ctrl.Result{}
	}
	if err := r.labelPod(ctx, pod, result); err != nil {
		logger.Info("pod labeling failed ", "err", err.Error())
		shouldRetry.Requeue = true
	}
	return shouldRetry, nil
}

func (r *PodReconciler) checkPod(ctx context.Context, pod *corev1.Pod) (validate.ValidationStatus, error) {
	var ns corev1.Namespace
	if err := r.client.Get(ctx, client.ObjectKey{Name: pod.Namespace}, &ns); err != nil {
		return validate.NoAction, err
	}

	validator := r.systemValidator
	if validate.IsUserValidationForNS(&ns) {
		var err error
		validator, err = validate.NewUserValidationSvc(&ns, r.userValidationSvcFactory)
		if err != nil {
			return validate.NoAction, err
		}
	}

	result, err := validator.ValidatePod(ctx, pod, &ns)
	if err != nil {
		return validate.NoAction, err
	}

	return result.Status, nil
}

func (r *PodReconciler) labelPod(ctx context.Context, pod corev1.Pod, result validate.ValidationStatus) error {

	resultLabel := labelForValidationResult(result)
	if resultLabel == "" {
		return nil
	}
	if pod.Labels[pkg.PodValidationLabel] != resultLabel {
		out := pod.DeepCopy()
		if out.ObjectMeta.Labels == nil {
			out.ObjectMeta.Labels = map[string]string{}
		}
		out.Labels[pkg.PodValidationLabel] = resultLabel
		if err := r.client.Patch(ctx, out, client.MergeFrom(&pod)); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

func areImagesChanged(oldPod *corev1.Pod, newPod *corev1.Pod) bool {
	oldImages := getPodImages(oldPod)
	newImages := getPodImages(newPod)
	if len(oldImages) != len(newImages) {
		return true
	}
	sort.Strings(oldImages)
	sort.Strings(newImages)
	for i := 0; i < len(oldImages); i++ {
		if oldImages[i] != newImages[i] {
			return true
		}
	}
	return false
}

func getPodImages(pod *corev1.Pod) []string {
	var result []string
	for _, container := range pod.Spec.InitContainers {
		result = append(result, container.Image)
	}
	for _, container := range pod.Spec.Containers {
		result = append(result, container.Image)
	}
	return result
}

func (r *PodReconciler) isValidationEnabledForNS(namespace string) bool {
	var ns corev1.Namespace
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: namespace}, &ns); err != nil {
		return false
	}
	return validate.IsValidationEnabledForNS(&ns)
}

func labelForValidationResult(result validate.ValidationStatus) string {
	switch result {
	case validate.NoAction:
		return ""
	case validate.Invalid:
		return pkg.ValidationStatusFailed
	case validate.Valid:
		return pkg.ValidationStatusSuccess
	case validate.ServiceUnavailable:
		return pkg.ValidationStatusPending
	default:
		return pkg.ValidationStatusPending
	}
}
