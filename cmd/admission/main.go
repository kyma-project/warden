package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/kyma-project/warden/internal/logging"
	"go.uber.org/zap/zapcore"
	"os"

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
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
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

	if err := certs.SetupCertSecret(
		context.Background(),
		appConfig.Admission.SecretName,
		appConfig.Admission.SystemNamespace,
		appConfig.Admission.ServiceName,
		logger); err != nil {
		logger.Error("failed to setup certificates and webhook secret", err.Error())
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), manager.Options{
		Scheme:             scheme,
		Port:               appConfig.Admission.Port,
		MetricsBindAddress: ":9090",
		Logger:             logrZap,
	})
	if err != nil {
		logger.Error("failed to start manager", err.Error())
		os.Exit(2)
	}

	if err := certs.SetupResourcesController(context.TODO(), mgr,
		appConfig.Admission.ServiceName,
		appConfig.Admission.SystemNamespace,
		appConfig.Admission.SecretName,
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
	whs.CertName = certs.CertFile
	whs.KeyName = certs.KeyFile

	whs.Register(admission.ValidationPath, &ctrlwebhook.Admission{
		Handler: admission.NewValidationWebhook(logger.With("webhook", "validation")),
	})

	whs.Register(admission.DefaultingPath, &ctrlwebhook.Admission{
		Handler: admission.NewDefaultingWebhook(mgr.GetClient(), validatorSvc, appConfig.Admission.Timeout, appConfig.Admission.StrictMode, logger.With("webhook", "defaulting")),
	})

	logger.Info("starting the controller-manager")
	// start the server manager
	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		logger.Error(err, "failed to start controller-manager")
		os.Exit(1)
	}
}
