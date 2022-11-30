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
	"github.com/kyma-project/warden/pkg/util/sets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	NamespaceValidationLabel = "namespaces.warden.kyma-project.io/validate"
	PodValidationLabel       = "pods.warden.kyma-project.io/validate"
	ValidationStatusPending  = "pending"
	ValidationStatusSuccess  = "success"
	ValidationStatusFailed   = "failed"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=validate.warden.kyma-project.io,resources=imagepolicies,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// TODO (Ressetkk): find a way to filter requests from non-labeled namespaces
	var ns corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if ns.GetLabels()[NamespaceValidationLabel] != "enabled" {
		return ctrl.Result{}, nil
	}

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var images sets.Strings
	for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		images.Add(c.Image)
	}

	matched := make(map[string]string)

	images.Walk(func(s string) {
		matched[s] = ""
	})

	shouldRetry := ctrl.Result{RequeueAfter: 10 * time.Second}

	admitResult := admitPod(&pod)
	switch admitResult {
	case ValidationStatusSuccess:
		l.Info("pod validated without errors")
		shouldRetry = ctrl.Result{}
		break
	case ValidationStatusFailed:
		//TODO this should return some kind of error
		l.Info("pod validated with errors")
		shouldRetry = ctrl.Result{}
		break
	}
	pod.Labels[PodValidationLabel] = admitResult
	if err := r.Update(ctx, &pod); client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	return shouldRetry, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				if e.ObjectOld.GetLabels()[PodValidationLabel] != e.ObjectNew.GetLabels()[PodValidationLabel] {
					return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
				}
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			}}).
		Complete(r)
}

func admitPod(pod *corev1.Pod) string {
	return ValidationStatusSuccess
}
