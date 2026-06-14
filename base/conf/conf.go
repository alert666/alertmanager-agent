package conf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	defaultLogLevel   = "info"
	defaultLogEncoder = "console"
)

	// LoadConfig loads configuration from the given YAML file path.
	func LoadConfig(configPath string) error {
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("configuration file %s does not exist", configPath)
	}
	if err != nil {
		return fmt.Errorf("stat configuration file %s failed: %w", configPath, err)
	}
	zap.L().Info("loading configuration", zap.String("path", configPath))

	configDir := filepath.Dir(configPath)
	configBase := filepath.Base(configPath)
	viper.SetConfigName(configBase)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	if err = viper.ReadInConfig(); err != nil {
		return fmt.Errorf("reading configuration file %s failed: %w", configPath, err)
	}
	return nil
}

// AllConfig returns all settings as a map.
func AllConfig() map[string]any {
	return viper.AllSettings()
}

// GetLogLevel returns the log level.
func GetLogLevel() string {
	level := viper.GetString("log.level")
	if level == "" {
		level = defaultLogLevel
	}
	return level
}

// GetLogEncoder returns the log encoder type (console or json).
func GetLogEncoder() string {
	enc := viper.GetString("log.encoder")
	if enc == "" {
		enc = defaultLogEncoder
	}
	return enc
}

// GetTimeZone returns the server timezone.
func GetTimeZone() string {
	tz := viper.GetString("timeZone")
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	return tz
}

// GetAgentServerAddr returns the api-server gRPC address.
func GetAgentServerAddr() string {
	return viper.GetString("agent.serverAddr")
}

// GetAgentID returns the agent's unique identifier.
// If not configured, defaults to the hostname (unique per Pod).
func GetAgentID() string {
	id := viper.GetString("agent.agentID")
	if id == "" {
		hostname, err := os.Hostname()
		if err == nil && hostname != "" {
			return hostname
		}
		return "unknown-agent"
	}
	return id
}

// GetAgentClusterID returns the agent's cluster identifier.
func GetAgentClusterID() string {
	return viper.GetString("agent.clusterID")
}

// GetKubeNamespace returns the Kubernetes namespace for alertmanager resources.
func GetKubeNamespace() string {
	ns := viper.GetString("kube.namespace")
	if ns == "" {
		ns = "monitoring"
	}
	return ns
}

// GetAlertmanagerSecretName returns the name of the Kubernetes Secret
// that holds the alertmanager configuration (alertmanager.yaml).
func GetAlertmanagerSecretName() string {
	name := viper.GetString("kube.alertmanagerSecretName")
	if name == "" {
		name = "alertmanager-config"
	}
	return name
}

// GetAgentVersion returns the agent version set at build time.
func GetAgentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("failed to get build info")
		return "unknow"
	}

	// search vcs.revision (commit hash)
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return "unknow"
}

// GetAgentHealthPort returns the port for the Kubernetes health HTTP server.
func GetAgentHealthPort() int {
	port := viper.GetInt("agent.healthPort")
	if port <= 0 {
		port = 9090
	}
	return port
}

// GetGrpcTLSCertFile returns the gRPC mTLS client certificate file path.
func GetGrpcTLSCertFile() string {
	return viper.GetString("grpc.tls.certFile")
}

// GetGrpcTLSKeyFile returns the gRPC mTLS client private key file path.
func GetGrpcTLSKeyFile() string {
	return viper.GetString("grpc.tls.keyFile")
}

// GetGrpcTLSCAFile returns the CA certificate file path for verifying the server.
func GetGrpcTLSCAFile() string {
	return viper.GetString("grpc.tls.caFile")
}
