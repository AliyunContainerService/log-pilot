package pilot

import (
	"fmt"
	"os"
)

// Global variables for piloter
const (
	ENV_PILOT_TYPE = "PILOT_TYPE"

	PILOT_FILEBEAT = "filebeat"
	PILOT_FLUENTD  = "fluentd"
)

// Piloter interface for piloter
type Piloter interface {
	Name() string

	Start() error
	Reload() error
	Stop() error

	GetConfHome() string
	GetConfPath(container string) string

	OnDestroyEvent(container string) error
}

// NewPiloter instantiates a new piloter
func NewPiloter(baseDir string) (Piloter, error) {
	if os.Getenv(ENV_PILOT_TYPE) == PILOT_FILEBEAT {
		return NewFilebeatPiloter(baseDir)
	}
	if os.Getenv(ENV_PILOT_TYPE) == PILOT_FLUENTD {
		return NewFluentdPiloter()
	}
	return nil, fmt.Errorf("InvalidPilotType")
}
