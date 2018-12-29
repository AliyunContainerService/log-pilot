package pilot

import (
	"os"
	"fmt"
)

const (
	ENV_PILOT_TYPE = "PILOT_TYPE"

	PILOT_FILEBEAT = "filebeat"
	PILOT_FLUENTD  = "fluentd"
	PILOT_FLUME = "flume"
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
	if os.Getenv(ENV_PILOT_TYPE) == PILOT_FLUME {
		return NewFlumePiloter()
	}
	return nil, fmt.Errorf("InvalidPilotType")
}
