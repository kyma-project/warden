package certs

import (
	"context"
	"os"
	"path"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// save data from secret into given dir
func SaveToDirectory(ctx context.Context, secretName, secretNamespace, dirPath string, log *zap.SugaredLogger) error {
	// We are going to talk to the API server _before_ we start the manager.
	// Since the default manager client reads from cache, we will get an error.
	// So, we create a "serverClient" that would read from the API directly.
	// We only use it here, this only runs at start up, so it shouldn't be to much for the API
	serverClient, err := ctrlclient.New(ctrl.GetConfigOrDie(), ctrlclient.Options{})
	if err != nil {
		return errors.Wrap(err, "failed to create a server client")
	}
	if err := apiextensionsv1.AddToScheme(serverClient.Scheme()); err != nil {
		return errors.Wrap(err, "while adding apiextensions.v1 schema to k8s client")
	}

	return saveToFile(ctx, serverClient, secretName, secretNamespace, dirPath, log)
}

func saveToFile(ctx context.Context, client ctrlclient.Client, secretName, secretNamespace, dirPath string, log *zap.SugaredLogger) error {
	secret := &corev1.Secret{}
	log.Info("saving certs to dir")
	err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret)
	if err != nil {
		return errors.Wrap(err, "failed to get webhook secret")
	}

	err = ensureDirExists(dirPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir '%s'", dirPath)
	}

	certFilePath := path.Join(dirPath, CertFile)
	err = os.WriteFile(certFilePath, secret.Data[CertFile], os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "failed to save server cert to file '%s'", certFilePath)
	}

	keyFilePath := path.Join(dirPath, KeyFile)
	err = os.WriteFile(keyFilePath, secret.Data[KeyFile], os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "failed to save server key to file '%s'", certFilePath)
	}

	return nil
}

func ensureDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
