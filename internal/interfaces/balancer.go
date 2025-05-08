package interfaces

type IBalancer interface {
	NextBackend() (string, error)
	ResetBackends(backs []string)
}