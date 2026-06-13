package server

// ServerInterface defines the lifecycle for any component in the application.
type ServerInterface interface {
	Start() error
	Stop() error
}
