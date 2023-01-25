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
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/pkg"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Validator validate.PodValidator
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	result, err := r.checkPod(ctx, &pod)
	if err != nil {
		return ctrl.Result{}, err
	}

	shouldRetry := ctrl.Result{RequeueAfter: 10 * time.Minute}
	switch result {
	case validate.Valid:
		l.Info("pod validated successfully", "name", pod.Name, "namespace", pod.Namespace)
		shouldRetry = ctrl.Result{}
	case validate.Invalid:
		l.Info("pod validation failed", "name", pod.Name, "namespace", pod.Namespace)
		shouldRetry = ctrl.Result{}
	}
	if err := r.labelPod(ctx, pod, result); err != nil {
		l.Info("pod labeling failed", "name", pod.Name, "namespace", pod.Namespace, "err", err.Error())
		shouldRetry.Requeue = true
	}
	return shouldRetry, nil
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
				if r.areImagesChanged(e.ObjectOld.DeepCopyObject(), e.ObjectNew.DeepCopyObject()) {
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

func (r *PodReconciler) checkPod(ctx context.Context, pod *corev1.Pod) (validate.ValidationResult, error) {
	var ns corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: pod.Namespace}, &ns); err != nil {
		return validate.NoAction, err
	}

	result, err := r.Validator.ValidatePod(ctx, pod, &ns)
	if err != nil {
		return validate.NoAction, err
	}

	return result, nil
}

func (r *PodReconciler) labelPod(ctx context.Context, pod corev1.Pod, result validate.ValidationResult) error {

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
		if err := r.Patch(ctx, out, client.MergeFrom(&pod)); client.IgnoreNotFound(err) != nil {
			return err
		}
	}
	return nil
}

// TODO: This function should check images not the whole spec.
func (r *PodReconciler) areImagesChanged(oldObject runtime.Object, newObject runtime.Object) bool {
	oldPod := oldObject.(*corev1.Pod)
	newPod := newObject.(*corev1.Pod)
	return !reflect.DeepEqual(oldPod.Spec.InitContainers, newPod.Spec.InitContainers) || !reflect.DeepEqual(oldPod.Spec.Containers, newPod.Spec.Containers)
}

func (r *PodReconciler) isValidationEnabledForNS(namespace string) bool {
	var ns corev1.Namespace
	if err := r.Get(context.TODO(), client.ObjectKey{Name: namespace}, &ns); err != nil {
		return false
	}
	return validate.IsValidationEnabledForNS(&ns)
}

func labelForValidationResult(result validate.ValidationResult) string {
	switch result {
	case validate.NoAction:
		return ""
	case validate.Invalid:
		return pkg.ValidationStatusFailed
	case validate.Valid:
		return pkg.ValidationStatusSuccess
	case validate.ServiceUnAvailable:
		return pkg.ValidationStatusPending
	default:
		return pkg.ValidationStatusPending
	}
}
