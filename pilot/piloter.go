package pilot

import (
	"fmt"
	"os"
)

const (
	ENV_PILOT_TYPE = "PILOT_TYPE"

	PILOT_FILEBEAT = "filebeat"
	PILOT_FLUENTD  = "fluentd"
)

type Piloter interface {
	Name() string

	Start() error
	Reload() error
	Stop() error

	GetConfHome() string
	GetConfPath(container string) string

	OnDestroyEvent(container string) error
}

func NewPiloter(baseDir string) (Piloter, error) {
	if os.Getenv(ENV_PILOT_TYPE) == PILOT_FILEBEAT {
		return NewFilebeatPiloter(baseDir)
	}
	if os.Getenv(ENV_PILOT_TYPE) == PILOT_FLUENTD {
		return NewFluentdPiloter()
	}
	return nil, fmt.Errorf("InvalidPilotType")
}
