package health

import (
	"github.com/alert666/alertmanager-agent/base/conf"
	"github.com/google/wire"
)

// ProvideHealthServer creates a health-check HTTP server bound to the
// configured agent.healthPort (default 9090).
func ProvideHealthServer(ready Checker) (*Server, error) {
	port := conf.GetAgentHealthPort()
	if port <= 0 {
		port = 9090
	}
	return NewServer(port, ready), nil
}

// HealthProviderSet wires up the health server and binds it as a
// server.ServerInterface so the Application runs it.
var HealthProviderSet = wire.NewSet(
	ProvideHealthServer,
)
