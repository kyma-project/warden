package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kyma-project/warden/internal/env"
	"github.com/kyma-project/warden/internal/logging"
	"github.com/kyma-project/warden/internal/webhook"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/go-logr/zapr"
	"github.com/kyma-project/warden/internal/admission"
	"github.com/kyma-project/warden/internal/config"
	"github.com/kyma-project/warden/internal/validate"
	"github.com/kyma-project/warden/internal/webhook/certs"
	"go.uber.org/zap"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	ctrladmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	scheme = runtime.NewScheme()
)

// nolint
func init() {
	_ = admissionregistrationv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

var (
	setupLog = ctrl.Log.WithName("setup")
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config-path", "./hack/config.yaml", "The path to the configuration file.")
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

	deployName := env.Get("ADMISSION_DEPLOYMENT_NAME")
	addOwnerRef, err := env.GetBool("ADDMISSION_ADD_CERT_OWNER_REF")
	if err != nil {
		setupLog.Error(err, "while configuring env")
		os.Exit(1)
	}

	if err := certs.SetupCertSecret(
		context.Background(),
		appConfig.Admission.SecretName,
		appConfig.Admission.SystemNamespace,
		appConfig.Admission.ServiceName,
		deployName,
		addOwnerRef,
		logger); err != nil {
		logger.Error("failed to setup certificates and webhook secret", err.Error())
		os.Exit(1)
	}

	if err := certs.SaveToDirectory(
		context.Background(),
		appConfig.Admission.SecretName,
		appConfig.Admission.SystemNamespace,
		certs.DefaultCertDir,
		logger); err != nil {
		logger.Error("failed to save certificate from secret", err.Error())
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), manager.Options{
		Scheme: scheme,
		Metrics: ctrlmetrics.Options{
			BindAddress: ":9090",
		},
		Logger:                 logrZap,
		HealthProbeBindAddress: ":8090",
		WebhookServer: ctrlwebhook.NewServer(ctrlwebhook.Options{
			CertName: certs.CertFile,
			KeyName:  certs.KeyFile,
			Port:     appConfig.Admission.Port,
		}),
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Secret{}: {
					Field: fields.SelectorFromSet(fields.Set{"metadata.name": appConfig.Admission.SecretName,
						"metadata.namespace": appConfig.Admission.SystemNamespace}),
				},
			},
		},
	})
	if err != nil {
		logger.Error("failed to start manager", err.Error())
		os.Exit(2)
	}

	if err := mgr.AddReadyzCheck("readiness check", healthz.Ping); err != nil {
		logger.Error(err, "unable to register readyz")
		os.Exit(1)
	}

	if err := webhook.SetupResourcesController(context.TODO(), mgr,
		appConfig.Admission.ServiceName,
		appConfig.Admission.SystemNamespace,
		appConfig.Admission.SecretName,
		deployName,
		addOwnerRef,
		logger); err != nil {
		logger.Error("failed to setup webhook resource controller ", err.Error())
		os.Exit(5)
	}

	repoFactory := validate.NotaryRepoFactory{Timeout: appConfig.Notary.Timeout}
	allowedRegistries := validate.ParseAllowedRegistries(appConfig.Notary.AllowedRegistries)

	validatorSvcConfig := validate.ServiceConfig{
		NotaryConfig:      validate.NotaryConfig{Url: appConfig.Notary.URL},
		AllowedRegistries: allowedRegistries,
	}
	podValidatorSvc := validate.NewImageValidator(&validatorSvcConfig, repoFactory)
	validatorSvc := validate.NewPodValidator(podValidatorSvc)

	logger.Info("setting up webhook server")
	// webhook server setup
	whs := mgr.GetWebhookServer()
	decoder := ctrladmission.NewDecoder(mgr.GetScheme())
	whs.Register(admission.ValidationPath, &ctrlwebhook.Admission{
		Handler: admission.NewValidationWebhook(logger.With("webhook", "validation"), decoder),
	})

	whs.Register(admission.DefaultingPath, &ctrlwebhook.Admission{
		Handler: admission.NewDefaultingWebhook(mgr.GetClient(), validatorSvc, appConfig.Admission.Timeout, appConfig.Admission.StrictMode, decoder, logger.With("webhook", "defaulting")),
	})

	logger.Info("starting the controller-manager")

	// start the server manager
	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		logger.Error(err, "failed to start controller-manager")
		os.Exit(1)
	}
}
