package main

import (
	"context"
	"flag"
	"fmt"
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

func main() {
	var configPath string
	flag.StringVar(&configPath, "config-path", "./hack/config.yaml", "The path to the configuration file.")
	flag.Parse()

	tmpLog, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("failed to start controller-manager", err.Error())
		os.Exit(1)
	}
	logger := tmpLog.Sugar()

	config, err := config.Load(configPath)
	if err != nil {
		logger.Error(err, fmt.Sprintf("unable to load configuration from path '%s'", configPath))
		os.Exit(1)
	}

	if err := certs.SetupCertSecret(
		context.Background(),
		config.Admission.SecretName,
		config.Admission.SystemNamespace,
		config.Admission.ServiceName,
		logger); err != nil {
		logger.Error("failed to setup certificates and webhook secret", err.Error())
		os.Exit(1)
	}

	logrZap := zapr.NewLogger(logger.Desugar())

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		Scheme:             scheme,
		Port:               config.Admission.Port,
		MetricsBindAddress: ":9090",
		Logger:             logrZap,
	})
	if err != nil {
		logger.Error("failed to start manager", err.Error())
		os.Exit(2)
	}

	if err := certs.SetupResourcesController(context.TODO(), mgr,
		config.Admission.ServiceName,
		config.Admission.SystemNamespace,
		config.Admission.SecretName,
		logger); err != nil {

		logger.Error("failed to setup webhook resource controller ", err.Error())
		os.Exit(5)
	}

	repoFactory := validate.NotaryRepoFactory{Timeout: config.Notary.Timeout}
	allowedRegistries := validate.ParseAllowedRegistries(config.Notary.AllowedRegistries)

	validatorSvcConfig := validate.ServiceConfig{
		NotaryConfig:      validate.NotaryConfig{Url: config.Notary.URL},
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
		Handler: admission.NewValidationWebhook(),
	})

	whs.Register(admission.DefaultingPath, &ctrlwebhook.Admission{
		Handler: admission.NewDefaultingWebhook(mgr.GetClient(), validatorSvc, config.Admission.Timeout, logger.With("webhook", "defaulting")),
	})

	logrZap.Info("starting the controller-manager")
	// start the server manager
	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		logrZap.Error(err, "failed to start controller-manager")
		os.Exit(1)
	}
}
