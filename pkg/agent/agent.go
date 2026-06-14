package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alert666/alertmanager-agent/base/conf"
	"github.com/alert666/alertmanager-agent/pkg/kube"
	v1 "github.com/alert666/alertmanager-proto/gen/go/data_tunnel/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
		"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
)

// Agent is the gRPC client that registers with the api-server via a data tunnel.
type Agent struct {
	conn   *grpc.ClientConn
	stream v1.TunnelService_DataTunnelClient
	cancel context.CancelFunc
	kube   kube.Interface
}

// NewAgent creates a new Agent and initiates the data tunnel registration.
func NewAgent(kc kube.Interface) (*Agent, error) {
	serverAddr := conf.GetAgentServerAddr()
	agentID := conf.GetAgentID()
	clusterID := conf.GetAgentClusterID()
	version := conf.GetAgentVersion()

	if serverAddr == "" {
		return nil, fmt.Errorf("agent.server_addr is empty")
	}
	if agentID == "" {
		return nil, fmt.Errorf("agent.agent_id is empty")
	}

	zap.L().Info("connecting to api-server",
		zap.String("server_addr", serverAddr),
		zap.String("agent_id", agentID),
		zap.String("cluster_id", clusterID),
		zap.String("version", version),
	)

	// 构建传输凭证：有客户端证书则 mTLS，否则明文
	creds := loadTransportCredentials()

	conn, err := grpc.NewClient(serverAddr,
				grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", serverAddr, err)
	}

	client := v1.NewTunnelServiceClient(conn)

	ctx, cancel := context.WithCancel(context.Background())

	stream, err := client.DataTunnel(ctx)
	if err != nil {
		conn.Close()
		cancel()
		return nil, fmt.Errorf("failed to open data tunnel: %w", err)
	}

	// send Init message to register
	initMsg := &v1.TunnelMessage{
		Payload: &v1.TunnelMessage_Init{
			Init: &v1.Init{
				AgentID:   agentID,
				ClusterID: clusterID,
				Version:   version,
			},
		},
	}
	if err := stream.Send(initMsg); err != nil {
		conn.Close()
		cancel()
		return nil, fmt.Errorf("failed to send init message: %w", err)
	}

	zap.L().Info("agent registered with api-server",
		zap.String("agentId", agentID),
		zap.String("serverAddr", serverAddr),
	)

	return &Agent{
		conn:   conn,
		stream: stream,
		cancel: cancel,
		kube:   kc,
	}, nil
}

// loadTransportCredentials 根据配置选择传输凭证：mTLS 或明文
func loadTransportCredentials() credentials.TransportCredentials {
	certFile := conf.GetGrpcTLSCertFile()
	keyFile := conf.GetGrpcTLSKeyFile()
	caFile := conf.GetGrpcTLSCAFile()

	// 没有配客户端证书 → 明文
	if certFile == "" || keyFile == "" {
		zap.L().Warn("gRPC TLS not configured, using insecure connection")
		return insecure.NewCredentials()
	}

	// 加载客户端证书
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		zap.L().Warn("failed to load client cert, fallback to insecure", zap.Error(err))
		return insecure.NewCredentials()
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// 加载 CA 证书验证服务端
	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			zap.L().Warn("failed to read CA cert, fallback to insecure", zap.Error(err))
			return insecure.NewCredentials()
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			zap.L().Warn("failed to parse CA cert, fallback to insecure")
			return insecure.NewCredentials()
		}
		tlsCfg.RootCAs = pool
	}

	zap.L().Info("gRPC mTLS enabled",
		zap.String("certFile", certFile),
		zap.String("caFile", caFile),
	)
	return credentials.NewTLS(tlsCfg)
}

// Start begins the agent's main loop: listens for commands from the server.
func (a *Agent) Start() error {
	zap.L().Info("agent started, waiting for commands")
	for {
		msg, err := a.stream.Recv()
		if err == io.EOF {
			zap.L().Info("agent stream closed by server")
			return nil
		}
		if err != nil {
			return fmt.Errorf("agent stream recv error: %w", err)
		}

		if cmd := msg.GetCommand(); cmd != nil {
			zap.L().Info("received command",
				zap.String("taskID", msg.GetTaskId()),
				zap.Int32("commandType", int32(cmd.GetType())),
			)
			result := a.handleCommand(cmd, msg.GetTaskId())
			resultMsg := &v1.TunnelMessage{
				TaskId: msg.GetTaskId(),
				Payload: &v1.TunnelMessage_CommandResult{
					CommandResult: result,
				},
			}
			if err := a.stream.Send(resultMsg); err != nil {
				zap.L().Info("failed to send command result",
					zap.String("taskID", msg.GetTaskId()),
					zap.Int32("commandType", int32(cmd.GetType())),
					zap.Error(err),
				)
			}
		}
	}
}

// Stop gracefully shuts down the agent connection.
func (a *Agent) Stop() error {
	zap.L().Info("agent stopping...")
	a.cancel()
	if err := a.stream.CloseSend(); err != nil {
		zap.L().Error("error closing agent stream", zap.Error(err))
	}
	if err := a.conn.Close(); err != nil {
		zap.L().Error("error closing agent connection", zap.Error(err))
	}
	zap.L().Info("agent stopped")
	return nil
}

// IsReady reports whether the gRPC connection to the api-server is healthy.
func (a *Agent) IsReady() bool {
	if a.conn == nil {
		return false
	}
	return a.conn.GetState() == connectivity.Ready
}

// handleCommand dispatches a command to the appropriate handler.
func (a *Agent) handleCommand(cmd *v1.Command, taskID string) *v1.CommandResult {
	switch cmd.GetType() {
	case v1.CommandType_COMMAND_TYPE_GET_ALERTMANAGER_CONFIG:
		return a.handleGetAlertmanagerConfig(cmd, taskID)
	case v1.CommandType_COMMAND_TYPE_GET_PROMETHEUS_CONFIG:
		return a.handleGetPrometheusConfig(cmd)
	case v1.CommandType_COMMAND_TYPE_RELOAD_ALERTMANAGER:
		return a.handleReloadAlertmanager(cmd)
	case v1.CommandType_COMMAND_TYPE_RELOAD_PROMETHEUS:
		return a.handleReloadPrometheus(cmd)
	case v1.CommandType_COMMAND_TYPE_UPDATE_ALERTMANAGER_CONFIG:
		return a.handleUpdateAlertmanagerConfig(cmd, taskID)
	default:
		return &v1.CommandResult{
			CommandType: cmd.GetType(),
			Error:       fmt.Sprintf("unknown command type: %v", cmd.GetType()),
		}
	}
}

func (a *Agent) handleGetAlertmanagerConfig(cmd *v1.Command, taskID string) *v1.CommandResult {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*15)
	defer cancel()
	data, err := a.kube.GetAlertmanagerConfig(ctx, taskID)
	if err != nil {
		zap.L().Error("failed to get alertmanager config", zap.String("taskID", taskID), zap.Error(err))
		return &v1.CommandResult{
			CommandType: cmd.GetType(),
			Error:       fmt.Sprintf("failed to get alertmanager config: %v", err),
		}
	}
	return &v1.CommandResult{
		CommandType: cmd.GetType(),
		Data:        data,
	}
}

func (a *Agent) handleReloadAlertmanager(cmd *v1.Command) *v1.CommandResult {
	// TODO: implement actual alertmanager reload
	return &v1.CommandResult{
		CommandType: cmd.GetType(),
	}
}

func (a *Agent) handleReloadPrometheus(cmd *v1.Command) *v1.CommandResult {
	// TODO: implement actual prometheus reload
	return &v1.CommandResult{
		CommandType: cmd.GetType(),
	}
}

func (a *Agent) handleUpdateAlertmanagerConfig(cmd *v1.Command, taskID string) *v1.CommandResult {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*15)
	defer cancel()

	const configParamKey = "alertmanager.yaml"

	cfgYAML, ok := cmd.GetParams()[configParamKey]
	if !ok || cfgYAML == "" {
		zap.L().Error("missing param %q in command params", zap.String("taskID", taskID))
		return &v1.CommandResult{
			CommandType: cmd.GetType(),
			Error:       fmt.Sprintf("missing param %q in command params", configParamKey),
		}
	}

	rawCfg, err := base64.StdEncoding.DecodeString(cfgYAML)
	if err != nil {
		zap.L().Error("failed to decode base64 config", zap.String("taskID", taskID), zap.Error(err))
		return &v1.CommandResult{
			CommandType: cmd.GetType(),
			Error:       fmt.Sprintf("failed to decode base64 config: %v", err),
		}
	}

	if err := a.kube.UpdateAlertmanagerConfig(ctx, taskID, rawCfg); err != nil {
		zap.L().Error("failed to update alertmanager config", zap.String("taskID", taskID), zap.Error(err))
		return &v1.CommandResult{
			CommandType: cmd.GetType(),
			Error:       fmt.Sprintf("failed to update alertmanager config: %v", err),
		}
	}

	return &v1.CommandResult{
		CommandType: cmd.GetType(),
	}
}

func (a *Agent) handleGetPrometheusConfig(cmd *v1.Command) *v1.CommandResult {
	// TODO: implement actual prometheus config retrieval
	return &v1.CommandResult{
		CommandType: cmd.GetType(),
		Data:        []byte(`{"status":"not yet implemented"}`),
	}
}


