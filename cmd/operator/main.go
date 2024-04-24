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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/kyma-project/warden/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/warden/internal/config"
	"github.com/kyma-project/warden/internal/controllers"
	"github.com/kyma-project/warden/internal/controllers/namespace"
	"github.com/kyma-project/warden/internal/validate"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	zapk8s "sigs.k8s.io/controller-runtime/pkg/log/zap"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	//+kubebuilder:scaffold:scheme
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config-path", "./hack/config.yaml", "The path to the configuration file.")
	opts := zapk8s.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	appConfig, err := config.Load(configPath)
	if err != nil {
		setupLog.Error(err, fmt.Sprintf("unable to load configuration from path '%s'", configPath))
		os.Exit(1)
	}

	atomic := zap.NewAtomicLevel()
	parsedLevel, err := zapcore.ParseLevel(appConfig.Logging.Level)
	if err != nil {
		setupLog.Error(err, "unable to parse logger level")
		os.Exit(1)
	}
	atomic.SetLevel(parsedLevel)

	l, err := logging.ConfigureLogger(appConfig.Logging.Level, appConfig.Logging.Format, atomic)
	if err != nil {
		setupLog.Error(err, "while configuring logger")
		os.Exit(10)
	}
	logger := l.WithContext()
	if err := config.Watch(configPath, logger.Named("config watcher")); err != nil {
		setupLog.Error(err, "while setup file watcher")
		os.Exit(2)
	}

	logrZap := zapr.NewLogger(logger.Desugar())
	ctrl.SetLogger(logrZap)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: ctrlmetrics.Options{
			BindAddress: appConfig.Operator.MetricsBindAddress,
		},
		WebhookServer: ctrlwebhook.NewServer(ctrlwebhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: appConfig.Operator.HealthProbeBindAddress,
		LeaderElection:         appConfig.Operator.LeaderElect,
		LeaderElectionID:       "c3790980.warden.kyma-project.io",
		Logger:                 logrZap,
		Cache: cache.Options{
			ByObject: map[ctrlclient.Object]cache.ByObject{
				&corev1.Secret{}:    {},
				&corev1.ConfigMap{}: {},
			},
		},
	})
	if err != nil {
		logger.Error(err, "unable to start manager")
		os.Exit(1)
	}

	repoFactory := validate.NotaryRepoFactory{Timeout: appConfig.Notary.Timeout}
	allowedRegistries := validate.ParseAllowedRegistries(appConfig.Notary.AllowedRegistries)

	notaryConfig := &validate.ServiceConfig{NotaryConfig: validate.NotaryConfig{Url: appConfig.Notary.URL}, AllowedRegistries: allowedRegistries}

	imageValidator := validate.NewImageValidator(notaryConfig, repoFactory)
	podValidator := validate.NewPodValidator(imageValidator)

	if err = (controllers.NewPodReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		podValidator,
		controllers.PodReconcilerConfig{RequeueAfter: appConfig.Operator.PodReconcilerRequeueAfter},
		logger.Named("pod-controller"),
	)).SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to create controller", "controller", "Pod")
		os.Exit(1)
	}

	// add namespace controller
	if err = (&namespace.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    logger.Named("namespace-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Namespace")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Error(err, "problem running manager")
		os.Exit(1)
	}

}
