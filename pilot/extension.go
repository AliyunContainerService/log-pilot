package pilot

import (
	"strings"

	"github.com/docker/docker/api/types"
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
	env := containerJSON.Config.Env
	containerEnvMap := make(map[string]string)     // all container envs map
	containerMetaEnvMap := make(map[string]string) // container meta env which will be injected into log-pilot
	for _, e := range env {
		envKV := strings.SplitN(e, "=", 2)
		containerEnvMap[envKV[0]] = envKV[1]
		if strings.HasPrefix(e, "COM_ALIYUN_META_ENVS_") {
			// e.g. COM_ALIYUN_META_ENVS_MY_POD_IP = k8s_pod_ip
			// MY_POD_IP is the original env key,
			// k8s_pod_ip is the target field name which put into the tplt.
			metaEnvName := strings.TrimPrefix(envKV[0], "COM_ALIYUN_META_ENVS_")
			containerMetaEnvMap[metaEnvName] = envKV[1]
		}
	}
	for metaEnvKey, metaEnvValueAsFieldName := range containerMetaEnvMap {
		if envValue, exists := containerEnvMap[metaEnvKey]; exists {
			putIfNotEmpty(container, metaEnvValueAsFieldName, envValue)
		}
	}
}
