package kube

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Interface defines the operations for fetching resources from Kubernetes.
type Interface interface {
	// GetAlertmanagerConfig retrieves the alertmanager configuration from the
	// configured Kubernetes Secret and returns the raw YAML bytes.
	GetAlertmanagerConfig(ctx context.Context, taskID string) ([]byte, error)

	// UpdateAlertmanagerConfig updates the alertmanager configuration in the
	// configured Kubernetes Secret. It performs a dry-run validation against
	// the Kubernetes API before applying the change.
	UpdateAlertmanagerConfig(ctx context.Context, taskID string, data []byte) error
}

// client implements Interface by wrapping a Kubernetes clientset.
type client struct {
	clientset  kubernetes.Interface
	namespace  string
	secretName string
}

// NewClient creates a new Kubernetes client. It first attempts to initialize
// using the in-cluster service account; if that fails, it falls back to
// loading a kubeconfig file from standard locations
// (KUBECONFIG env var or ~/.kube/config).
func NewClient(namespace, secretName string) (Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		zap.L().Info("in-cluster config not available, trying kubeconfig",
			zap.String("reason", err.Error()))

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, configOverrides,
		)
		restConfig, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	} else {
		zap.L().Info("using in-cluster kubernetes config")
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	zap.L().Info("kubernetes client initialized",
		zap.String("namespace", namespace),
		zap.String("alertmanager_secret", secretName),
	)
	return &client{
		clientset:  clientset,
		namespace:  namespace,
		secretName: secretName,
	}, nil
}

// NewClientForConfig creates a Kubernetes client from a pre-built rest.Config.
// Useful for testing or when the caller already has a config.
func NewClientForConfig(cfg *rest.Config, namespace, secretName string) (Interface, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}
	return &client{
		clientset:  clientset,
		namespace:  namespace,
		secretName: secretName,
	}, nil
}

// GetAlertmanagerConfig fetches the alertmanager configuration from the
// configured Kubernetes Secret. The config is expected under the key
// "alertmanager.yaml" within the Secret's data.
func (c *client) GetAlertmanagerConfig(ctx context.Context, taskID string) ([]byte, error) {
	secret, err := c.clientset.CoreV1().Secrets(c.namespace).
		Get(ctx, c.secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w",
			c.namespace, c.secretName, err)
	}

	const configKey = "alertmanager.yaml"
	data, ok := secret.Data[configKey]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s does not contain key %q",
			c.namespace, c.secretName, configKey)
	}

	zap.L().Debug("fetched alertmanager config from secret",
		zap.String("taskID", taskID),
		zap.String("namespace", c.namespace),
		zap.String("secret", c.secretName),
		zap.Int("bytes", len(data)),
	)
	return data, nil
}

// UpdateAlertmanagerConfig validates and updates the alertmanager configuration
// stored in the Kubernetes Secret. The process:
//  1. Validates the input is well-formed YAML.
//  2. Dry-runs the Secret update against the Kubernetes API (schema validation).
//  3. Performs the real Secret update.
func (c *client) UpdateAlertmanagerConfig(ctx context.Context, taskID string, data []byte) error {
	const configKey = "alertmanager.yaml"

	// Step 1: basic content validation — ensure the config is parseable YAML.
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid alertmanager config (not valid YAML): %w", err)
	}
	zap.L().Info("alertmanager config content validation passed",
		zap.Int("bytes", len(data)),
	)

	// Step 2: fetch the current Secret as the base for the update.
	secret, err := c.clientset.CoreV1().Secrets(c.namespace).
		Get(ctx, c.secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s/%s for update: %w",
			c.namespace, c.secretName, err)
	}

	updatedSecret := secret.DeepCopy()
	if updatedSecret.Data == nil {
		updatedSecret.Data = make(map[string][]byte)
	}
	updatedSecret.Data[configKey] = data

	// Step 3: dry-run to let the Kubernetes API validate the resource.
	_, err = c.clientset.CoreV1().Secrets(c.namespace).
		Update(ctx, updatedSecret, metav1.UpdateOptions{DryRun: []string{"All"}})
	if err != nil {
		return fmt.Errorf("dry-run validation failed for secret %s/%s: %w",
			c.namespace, c.secretName, err)
	}
	zap.L().Info("dry-run passed, committing secret update",
		zap.String("taskID", taskID),
		zap.String("namespace", c.namespace),
		zap.String("secret", c.secretName),
		zap.Int("bytes", len(data)),
	)
	// return nil
	// Step 4: real update.
	if _, err = c.clientset.CoreV1().Secrets(c.namespace).
		Update(ctx, updatedSecret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update secret %s/%s: %w",
			c.namespace, c.secretName, err)
	}

	zap.L().Info("alertmanager config updated in secret",
		zap.String("namespace", c.namespace),
		zap.String("secret", c.secretName),
		zap.Int("bytes", len(data)),
	)
	return nil
}

// Ensure client implements Interface at compile time.
var _ Interface = (*client)(nil)
