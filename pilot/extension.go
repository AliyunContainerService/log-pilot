package pilot

import (
	"github.com/docker/docker/api/types"
	"strings"
)

func extension(container map[string]string, containerJSON *types.ContainerJSON) {
	labels := containerJSON.Config.Labels
	for name, value := range labels {
		if strings.HasPrefix(name, "com.aliyun.access.") {
			//fmt.Printf("label: %s=%s\n", name, value)
			name = strings.Replace(name, ".", "_", -1)
			putIfNotEmpty(container, name, value)
		}
	}
}
