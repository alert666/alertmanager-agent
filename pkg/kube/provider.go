package kube

import (
	"github.com/alert666/alertmanager-agent/base/conf"
	"github.com/google/wire"
)

// KubeProviderSet wires kube.Interface using configuration values.
var KubeProviderSet = wire.NewSet(
	ProvideKubeClient,
)

// ProvideKubeClient creates a kube.Interface by reading namespace and secret name from config.
func ProvideKubeClient() (Interface, error) {
	return NewClient(conf.GetKubeNamespace(), conf.GetAlertmanagerSecretName())
}
