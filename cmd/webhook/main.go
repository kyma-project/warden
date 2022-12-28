package main

import (
	"context"
	"github.com/go-logr/zapr"
	"github.com/kyma-project/warden/internal/webhook"
	"github.com/kyma-project/warden/internal/webhook/certs"
	"github.com/kyma-project/warden/internal/webhook/defaulting"
	"github.com/vrischmann/envconfig"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	cfg := &webhook.Config{}
	if err := envconfig.InitWithPrefix(cfg, "WEBHOOK"); err != nil {
		panic(err)
	}

	if err := certs.SetupCertSecret(
		context.Background(),
		cfg.SecretName,
		cfg.SystemNamespace,
		cfg.ServiceName,
		logger.Sugar()); err != nil {
		logger.Sugar().Error("failed to setup certificates and webhook secret", err.Error())
		os.Exit(1)
	}

	logrZap := zapr.NewLogger(logger)

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		Scheme:             scheme,
		Port:               cfg.Port,
		MetricsBindAddress: ":9090",
		Logger:             logrZap,
	})

	logger.Info("setting up webhook server")
	// webhook server setup
	whs := mgr.GetWebhookServer()
	whs.CertName = certs.CertFile
	whs.KeyName = certs.KeyFile

	whs.Register(defaulting.WebhookPath, &ctrlwebhook.Admission{
		Handler: defaulting.NewWebhook(mgr.GetClient()),
	})

	//whs.Register(resources.FunctionValidationWebhookPath, &ctrlwebhook.Admission{
	//	Handler: webhook.NewValidatingHook(validationConfigv1alpha1, validationConfigv1alpha2, mgr.GetClient()),
	//})

	logrZap.Info("starting the controller-manager")
	// start the server manager
	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		logrZap.Error(err, "failed to start controller-manager")
		os.Exit(1)
	}
}
