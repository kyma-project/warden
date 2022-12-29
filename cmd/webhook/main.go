package main

import (
	"context"
	"fmt"
	"github.com/go-logr/zapr"
	"github.com/kyma-project/warden/internal/webhook"
	"github.com/kyma-project/warden/internal/webhook/certs"
	"github.com/kyma-project/warden/internal/webhook/defaulting"
	"github.com/kyma-project/warden/internal/webhook/validation"
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
	tmpLog, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("failed to start controller-manager", err.Error())
		os.Exit(1)
	}
	logger := tmpLog.Sugar()

	cfg := &webhook.Config{}
	if err := envconfig.InitWithPrefix(cfg, "WEBHOOK"); err != nil {
		logger.Error("failed to start controller-manager", err.Error())
		os.Exit(1)
	}

	if err := certs.SetupCertSecret(
		context.Background(),
		cfg.SecretName,
		cfg.SystemNamespace,
		cfg.ServiceName,
		logger); err != nil {
		logger.Error("failed to setup certificates and webhook secret", err.Error())
		os.Exit(1)
	}

	logrZap := zapr.NewLogger(logger.Desugar())

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		Scheme:             scheme,
		Port:               cfg.Port,
		MetricsBindAddress: ":9090",
		Logger:             logrZap,
	})

	if err := certs.SetupResourcesController(context.TODO(), mgr, cfg.ServiceName, cfg.SystemNamespace, cfg.SecretName, logger); err != nil {
		logger.Error("failed to setup webhook resource controller ", err.Error())
		os.Exit(5)
	}

	logger.Info("setting up webhook server")
	// webhook server setup
	whs := mgr.GetWebhookServer()
	whs.CertName = certs.CertFile
	whs.KeyName = certs.KeyFile

	whs.Register(defaulting.WebhookPath, &ctrlwebhook.Admission{
		Handler: defaulting.NewWebhook(mgr.GetClient()),
	})

	whs.Register(validation.WebhookPath, &ctrlwebhook.Admission{
		Handler: validation.NewWebhook(mgr.GetClient()),
	})

	logrZap.Info("starting the controller-manager")
	// start the server manager
	err = mgr.Start(ctrl.SetupSignalHandler())
	if err != nil {
		logrZap.Error(err, "failed to start controller-manager")
		os.Exit(1)
	}
}
