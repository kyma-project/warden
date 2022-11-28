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
	"fmt"
	"github.com/kyma-project/warden/pkg/validate"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/url"
	"path"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	validatev1alpha1 "github.com/kyma-project/warden/api/v1alpha1"
)

// ValidatorReconciler reconciles a Validator object
type ValidatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=validate.warden.kyma-project.io,resources=validators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=validate.warden.kyma-project.io,resources=validators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=validate.warden.kyma-project.io,resources=validators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Validator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ValidatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	var v validatev1alpha1.Validator
	if err := r.Get(ctx, req.NamespacedName, &v); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// TODO (Ressetkk): Add support for custom TLS configuration
	t := v.Spec.Type
	prevState := v.Status.State
	nextState := ""
	var errored error
	switch t {
	case "allow", "deny":
		// static mappings for validation
		nextState = t
		break
	case "notary":
		nextState = "available"
		if err := validateNotary(v.Spec.NotaryConfig); err != nil {
			nextState = "unavailable"
			errored = err
			l.Error(err, "notary validation error")
		}
		break
	default:
		nextState = "unavailable"
		l.Error(fmt.Errorf("type \"%s\" is not supported at this moment", t), "not supported")
		break
	}
	if prevState != nextState {
		v.Status.State = nextState
		if err := r.Status().Update(ctx, &v); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}
	return ctrl.Result{}, errored
}

// SetupWithManager sets up the controller with the Manager.
func (r *ValidatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&validatev1alpha1.Validator{}).
		Complete(r)
}

func validateNotary(c validate.NotaryConfig) error {
	// code taken and adapted from github.com/theupdateframework/notary
	endpoint, err := url.Parse(c.Url)
	if err != nil {
		return fmt.Errorf("could not parse remote trust server url (%s): %w", c.Url, err)
	}

	if endpoint.Scheme == "" {
		return fmt.Errorf("trust server url has to be in the form of http(s)://URL:PORT. Got: %s", c.Url)
	}
	subPath, err := url.Parse(path.Join(endpoint.Path, "/v2") + "/")
	if err != nil {
		return fmt.Errorf("failed to parse v2 subpath. This error should not have been reached. Please report it as an issue at https://github.com/theupdateframework/notary/issues: %w", err)
	}
	endpoint = endpoint.ResolveReference(subPath)

	pingClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return err
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach %s: %w", c.Url, err)
	}
	defer resp.Body.Close()
	if (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) &&
		resp.StatusCode != http.StatusUnauthorized {
		// If we didn't get a 2XX range or 401 status code, we're not talking to a notary server.
		// The http client should be configured to handle redirects so at this point, 3XX is
		// not a valid status code.
		return fmt.Errorf("could not reach %s: %d", c.Url, resp.StatusCode)
	}
	return nil
}
