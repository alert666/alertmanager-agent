//go:build wireinject
// +build wireinject

package infra

import (
	"github.com/alert666/alertmanager-agent/base/server"
	"github.com/alert666/alertmanager-agent/base/app"
	"github.com/alert666/alertmanager-agent/pkg/agent"
	"github.com/alert666/alertmanager-agent/pkg/health"
	"github.com/alert666/alertmanager-agent/pkg/kube"
	"github.com/google/wire"
)

// InitApplication wires up the full dependency graph.
func InitApplication() (*app.Application, func(), error) {
	panic(wire.Build(
		agent.AgentProviderSet,
		health.HealthProviderSet,
		kube.KubeProviderSet,
		app.AppProviderSet,
		provideServers,
		// Bind *Agent as the health.Checker for readiness probes.
		wire.Bind(new(health.Checker), new(*agent.Agent)),
	))
}

// provideServers collects all ServerInterface implementations into a slice
// so Application can run them both.
func provideServers(agent *agent.Agent, healthSrv *health.Server) []server.ServerInterface {
	return []server.ServerInterface{agent, healthSrv}
}
