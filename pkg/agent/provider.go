package agent

import (
	"github.com/google/wire"
)

var AgentProviderSet = wire.NewSet(
	NewAgent,
)
