package pilot

import (
	"fmt"
	"os"
	"strings"
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

	GetBaseConf() string
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

// CustomConfig custom config
func CustomConfig(name string, customConfigs map[string]string, logConfig *LogConfig) {
	if os.Getenv(ENV_PILOT_TYPE) == PILOT_FILEBEAT {
		fields := make(map[string]string)
		configs := make(map[string]string)
		for k, v := range customConfigs {
			if strings.HasPrefix(k, name) {
				key := strings.TrimPrefix(k, name+".")
				if strings.HasPrefix(key, "fields") {
					key2 := strings.TrimPrefix(key, "fields.")
					fields[key2] = v
				} else {
					configs[key] = v
				}
			}
		}
		logConfig.CustomFields = fields
		logConfig.CustomConfigs = configs
	}
}
