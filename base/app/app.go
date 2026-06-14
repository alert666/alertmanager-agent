package app

import (
	"context"
	"sync"

	"github.com/alert666/alertmanager-agent/base/server"
	"go.uber.org/zap"
)

// Application holds all runtime components.
type Application struct {
	servers []server.ServerInterface
	wg      *sync.WaitGroup
}

// NewApplication creates an Application with the given agent.
func NewApplication(servers []server.ServerInterface) *Application {
	return &Application{
		servers: servers,
		wg:      &sync.WaitGroup{},
	}
}

func (app *Application) Run(ctx context.Context) error {
	if len(app.servers) == 0 {
		return nil
	}
	errCh := make(chan error, len(app.servers))
	for _, s := range app.servers {
		go func(s server.ServerInterface) {
			errCh <- s.Start()
		}(s)
	}

	select {
	case err := <-errCh:
		app.Stop()
		return err
	case <-ctx.Done():
		app.Stop()
		return nil
	}
}

func (app *Application) Stop() {
	for _, s := range app.servers {
		app.wg.Add(1)
		go func(s server.ServerInterface) {
			defer app.wg.Done()
			if err := s.Stop(); err != nil {
				zap.S().Errorf("stop error: %v", err)
			}
		}(s)
	}
	app.wg.Wait()
}
