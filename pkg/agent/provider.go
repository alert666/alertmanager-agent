package agent

import (
	"github.com/alert666/alertmanager-agent/base/server"
	"github.com/google/wire"
)

var AgentProviderSet = wire.NewSet(
	NewAgent,
	wire.Bind(new(server.ServerInterface), new(*Agent)),
)
