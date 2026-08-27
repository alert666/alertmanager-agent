# alertmanager-agent

部署在 Kubernetes 集群内的 Agent：通过 gRPC 双向流（data tunnel）注册到 api-server，接收远程指令并管理集群内的 Alertmanager / Prometheus 配置。

## 功能特性

- **gRPC 数据隧道**：通过 `TunnelService.DataTunnel` 双向流连接 api-server，上报 agentID / clusterID / version，持续接收指令并回传结果。
- **传输安全**：支持 mTLS（CA + 客户端证书），未配置证书时回退为明文连接。
- **Alertmanager 配置管理**：从 Kubernetes Secret 读取 `alertmanager.yaml`；更新时依次做 base64 解码 → YAML 校验 → K8s dry-run 校验 → 提交更新。
- **Prometheus 配置管理**：获取 / 重载指令（当前为占位实现，待完善）。
- **Kubernetes 集成**：优先使用集群内 ServiceAccount，失败时回退本地 kubeconfig。
- **健康检查**：`/healthz`（存活）、`/readyz`（就绪，取决于与 api-server 的 gRPC 连接状态）。
- **配置灵活**：YAML + 环境变量 + 命令行参数（viper + cobra）。
- **依赖注入**：Google Wire 组装依赖图；**结构化日志**：zap（json / console）。

## 目录结构

```
├── main.go            # 入口
├── base/              # app / conf / log / server 等基础组件
├── internal/infra/    # cobra 命令与 wire 注入
├── pkg/agent/         # gRPC 隧道客户端与指令处理
├── pkg/kube/          # Kubernetes Secret 读写
├── pkg/health/        # 健康检查 HTTP 服务
├── config.yaml        # 本地配置示例
├── k8s-deploy.yaml    # K8s 部署清单（RBAC/ConfigMap/Secret/Deployment）
└── Dockerfile         # 多阶段构建
```

## 快速开始

```bash
# 构建
go build -o alertmanager-agent ./main.go

# 运行（默认读取 ./config.yaml）
./alertmanager-agent -c ./config.yaml

# 指定日志级别
./alertmanager-agent -c ./config.yaml -l debug
```

命令行参数：

| 参数 | 简写 | 默认值 | 说明 |
|------|------|--------|------|
| `--config-path` | `-c` | `./config.yaml` | 配置文件路径 |
| `--log-level` | `-l` | `info` | 日志级别（debug/info/warn/error） |

## 配置

示例见 [`config.yaml`](config.yaml)：

| 配置键 | 说明 |
|--------|------|
| `grpc.tls.caFile` / `certFile` / `keyFile` | gRPC mTLS 证书路径；留空则明文连接 |
| `log.encoder` / `log.level` | 日志编码（json/console）与级别 |
| `timeZone` | 时区（默认 `Asia/Shanghai`） |
| `agent.serverAddr` | api-server gRPC 地址 |
| `agent.agentID` | Agent 唯一标识（默认取 hostname） |
| `agent.clusterID` | 所属集群标识 |
| `agent.healthPort` | 健康检查端口（默认 9090） |
| `kube.namespace` | Alertmanager 资源所在命名空间 |
| `kube.alertmanagerSecretName` | 存储 `alertmanager.yaml` 的 Secret 名称 |

配置项可通过环境变量覆盖：默认前缀 `ALERTMANAGER_AGENT`，键名中 `.` 转为 `_`，例如 `agent.clusterID` → `ALERTMANAGER_AGENT_AGENT_CLUSTERID`（前缀可由 `SERVICE_NAME` 改变）。

## 生成 pb（protobuf 代码）

本仓库不直接包含 `.proto` 文件，pb 代码由独立模块 [`github.com/alert666/alertmanager-proto`](https://github.com/alert666/alertmanager-proto) 生成（输出到该模块的 `gen/go/`），本仓库通过 Go module 引入。

在 **alertmanager-proto 仓库**中执行：

```bash
# 安装 buf（已安装可跳过）
brew install buf          # macOS
# 或 go install github.com/bufbuild/buf/cmd/buf@latest

# 拉取外部依赖（buf.yaml 声明了 deps 时才需要）
buf dep update

# 读取 buf.gen.yaml 生成 pb 代码
buf generate
```

生成后回到本仓库升级依赖：

```bash
go get github.com/alert666/alertmanager-proto@<tag>
go mod tidy
```

## 构建与部署

### Docker

```bash
docker build -t alertmanager-agent .
docker run --rm -v $(pwd)/config.yaml:/app/config.yaml alertmanager-agent -c /app/config.yaml
```

### Kubernetes

```bash
kubectl apply -f k8s-deploy.yaml
```

`k8s-deploy.yaml` 包含：ConfigMap（注入配置）、Secret（gRPC 客户端 TLS 证书）、RBAC（对指定 Secret 的 get/update/patch 授权）、Deployment（含存活/就绪探针）。

## License

详见 [LICENSE](LICENSE)。
