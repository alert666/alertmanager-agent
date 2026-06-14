package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Checker is implemented by components that can report readiness.
type Checker interface {
	IsReady() bool
}

// Server exposes HTTP endpoints for Kubernetes health probes.
type Server struct {
	srv  *http.Server
	port int
	rdy  Checker
}

// NewServer creates a health Server on the given port.
func NewServer(port int, ready Checker) *Server {
	mux := http.NewServeMux()
	s := &Server{port: port, rdy: ready}
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	s.srv = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	return s
}

// Start begins serving health probes. Blocks until the server is stopped.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("health server listen on :%d: %w", s.port, err)
	}
	zap.L().Info("health server listening", zap.Int("port", s.port))
	return s.srv.Serve(lis)
}

// Stop gracefully shuts down the health server.
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if s.rdy != nil && s.rdy.IsReady() {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("not ready"))
}
