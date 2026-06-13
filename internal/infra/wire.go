//go:build wireinject
// +build wireinject

package infra

import (
	"github.com/alert666/alertmanager-agent/base/app"
	"github.com/alert666/alertmanager-agent/pkg/agent"
	"github.com/alert666/alertmanager-agent/pkg/kube"
	"github.com/google/wire"
)

// InitApplication wires up the full dependency graph.
func InitApplication() (*app.Application, func(), error) {
	panic(wire.Build(
		agent.AgentProviderSet,
		kube.KubeProviderSet,
		app.AppProviderSet,
	))
}
