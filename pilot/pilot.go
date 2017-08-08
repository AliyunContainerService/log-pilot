package pilot

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"
)

/**
Label:
aliyun.log: /var/log/hello.log[:json][;/var/log/abc/def.log[:txt]]
*/

const LABEL_SERVICE_LOGS = "aliyun.logs."
const ENV_SERVICE_LOGS = "aliyun_logs_"
const FLUENTD_CONF_HOME = "/etc/fluentd"

const LABEL_PROJECT = "com.docker.compose.project"
const LABEL_SERVICE = "com.docker.compose.service"
const LABEL_POD = "io.kubernetes.pod.name"

type Pilot struct {
	mutex        sync.Mutex
	tpl          *template.Template
	base         string
	dockerClient *client.Client
	reloadable   bool
}

func New(tplStr string, baseDir string) (*Pilot, error) {
	tpl, err := template.New("fluentd").Parse(tplStr)
	if err != nil {
		return nil, err
	}

	if os.Getenv("DOCKER_API_VERSION") == "" {
		os.Setenv("DOCKER_API_VERSION", "1.23")
	}
	client, err := client.NewEnvClient()

	if err != nil {
		return nil, err
	}

	return &Pilot{
		dockerClient: client,
		tpl:          tpl,
		base:         baseDir,
	}, nil
}

func (p *Pilot) watch() error {

	p.reloadable = false
	if err := p.processAllContainers(); err != nil {
		return err
	}
	StartFluentd()
	p.reloadable = true

	ctx := context.Background()
	filter := filters.NewArgs()
	filter.Add("type", "container")

	options := types.EventsOptions{
		Filters: filter,
	}
	msgs, errs := p.client().Events(ctx, options)
	for {
		select {
		case msg := <-msgs:
			if err := p.processEvent(msg); err != nil {
				log.Errorf("fail to process event: %v,  %v", msg, err)
			}
		case err := <-errs:
			log.Warnf("error: %v", err)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			} else {
				msgs, errs = p.client().Events(ctx, options)
			}
		}
	}
}

type Source struct {
	Application string
	Service     string
	POD         string
	Container   string
}

type LogConfig struct {
	Name         string
	HostDir      string
	ContainerDir string
	Format       string
	FormatConfig map[string]string
	File         string
	Tags         map[string]string
	Target       string
	TimeKey      string
	TimeFormat   string
	HostKey      string
}

func (p *Pilot) cleanConfigs() error {
	confDir := fmt.Sprintf("%s/conf.d", FLUENTD_CONF_HOME)
	d, err := os.Open(confDir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		path := filepath.Join(confDir, name)
		stat, err := os.Stat(filepath.Join(confDir, name))
		if err != nil {
			return err
		}
		if stat.Mode().IsRegular() {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil

}

func (p *Pilot) processAllContainers() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	opts := types.ContainerListOptions{All: true}
	containers, err := p.client().ContainerList(context.Background(), opts)
	if err != nil {
		return err
	}

	//clean config
	if err := p.cleanConfigs(); err != nil {
		return err
	}

	for _, c := range containers {
		if c.State == "removing" {
			continue
		}
		containerJSON, err := p.client().ContainerInspect(context.Background(), c.ID)
		if err != nil {
			return err
		}
		if err = p.newContainer(containerJSON); err != nil {
			log.Errorf("fail to process container %s: %v", containerJSON.Name, err)
		}
	}

	return nil
}

func (p *Pilot) newContainer(containerJSON types.ContainerJSON) error {
	id := containerJSON.ID
	jsonLogPath := containerJSON.LogPath
	mounts := containerJSON.Mounts
	labels := containerJSON.Config.Labels
	env := containerJSON.Config.Env

	//logConfig.containerDir match types.mountPoint
	/**
	场景：
	1. 容器一个路径，中间有多级目录对应宿主机不同的目录
	2. containerdir对应的目录不是直接挂载的，挂载的是它上级的目录

	查找：从containerdir开始查找最近的一层挂载
	*/

	source := Source{
		Application: labels[LABEL_PROJECT],
		Service:     labels[LABEL_SERVICE],
		POD:         labels[LABEL_POD],
		Container:   strings.TrimPrefix(containerJSON.Name, "/"),
	}

	for _, e := range env {
		if !strings.HasPrefix(e, ENV_SERVICE_LOGS) {
			continue
		}
		envLabel := strings.Split(e, "=")
		if len(envLabel) == 2 {
			labelKey := strings.Replace(envLabel[0], "_", ".", -1)
			labels[labelKey] = envLabel[1]
		}

	}

	logConfigs, err := p.getLogConfigs(jsonLogPath, mounts, labels)
	if err != nil {
		return err
	}

	if len(logConfigs) == 0 {
		log.Debugf("%s has not log config, skip", id)
		return nil
	}

	//pilot.findMounts(logConfigs, jsonLogPath, mounts)
	//生成配置
	fluentdConfig, err := p.render(id, source, logConfigs)
	if err != nil {
		return err
	}
	//TODO validate config before save
	//log.Infof("Save %s to %s", fluentdConfig, p.pathOf(id))
	if err = ioutil.WriteFile(p.pathOf(id), []byte(fluentdConfig), os.FileMode(0644)); err != nil {
		return err
	}
	p.tryReload()
	return nil
}

func (p *Pilot) tryReload() {
	if p.reloadable {
		ReloadFluentd()
	}
}

func (p *Pilot) pathOf(container string) string {
	return fmt.Sprintf("%s/conf.d/%s.conf", FLUENTD_CONF_HOME, container)
}

func (p *Pilot) delContainer(id string) error {
	log.Infof("Try remove config %s", id)
	if err := os.Remove(p.pathOf(id)); err != nil {
		return err
	}
	p.tryReload()
	return nil
}

func (p *Pilot) client() *client.Client {
	return p.dockerClient
}

func (p *Pilot) processEvent(msg events.Message) error {
	containerId := msg.Actor.ID
	ctx := context.Background()
	switch msg.Action {
	case "start":
		log.Debugf("Process container start event: %s", containerId)
		if p.exists(containerId) {
			log.Debugf("%s is already exists.", containerId)
			return nil
		}
		containerJSON, err := p.client().ContainerInspect(ctx, containerId)
		if err != nil {
			return err
		}
		return p.newContainer(containerJSON)
	case "destroy":
		log.Debugf("Process container destory event: %s", containerId)
		p.delContainer(containerId)
	}
	return nil
}

func (p *Pilot) hostDirOf(path string, mounts map[string]types.MountPoint) string {
	confPath := path
	for {
		if point, ok := mounts[path]; ok {
			if confPath == path {
				return point.Source
			} else {
				relPath, err := filepath.Rel(path, confPath)
				if err != nil {
					panic(err)
				}
				return fmt.Sprintf("%s/%s", point.Source, relPath)
			}
		}
		path = filepath.Dir(path)
		if path == "/" || path == "." {
			break
		}
	}
	return ""
}

func (p *Pilot) parseTags(tags string) (map[string]string, error) {
	tagMap := make(map[string]string)
	if tags == "" {
		return tagMap, nil
	}

	kvArray := strings.Split(tags, ",")
	for _, kv := range kvArray {
		arr := strings.Split(kv, "=")
		if len(arr) != 2 {
			return nil, fmt.Errorf("%s is not a valid k=v format", kv)
		}
		key := strings.TrimSpace(arr[0])
		value := strings.TrimSpace(arr[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("%s is not a valid k=v format", kv)
		}
		tagMap[key] = value
	}
	return tagMap, nil
}

func (p *Pilot) parseLogConfig(name string, info *LogInfoNode, jsonLogPath string, mounts map[string]types.MountPoint) (*LogConfig, error) {
	path := info.value
	if path == "" {
		return nil, fmt.Errorf("path for %s is empty", name)
	}

	tags := info.get("tags")
	tagMap, err := p.parseTags(tags)

	if err != nil {
		return nil, fmt.Errorf("parse tags for %s error: %v", name, err)
	}

	target := info.get("target")

	timeKey := info.get("time_key")
	if timeKey == "" {
		timeKey = "@timestamp"
	}

	timeFormat := info.get("time_format")
	if timeFormat == "" {
		timeFormat = "%Y-%m-%dT%H:%M:%S.%L"
	}

	hostKey := info.get("host_key")
	if hostKey == "" {
		hostKey = "host"
	}

	if path == "stdout" {
		return &LogConfig{
			Name:         name,
			HostDir:      filepath.Join(p.base, filepath.Dir(jsonLogPath)),
			Format:       "json",
			File:         filepath.Base(jsonLogPath),
			Tags:         tagMap,
			FormatConfig: map[string]string{"time_format": "%Y-%m-%dT%H:%M:%S.%NZ"},
			Target:       target,
			TimeKey:      timeKey,
			TimeFormat:   timeFormat,
			HostKey:      hostKey,
		}, nil
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%s must be absolute path, for %s", path, name)
	}
	containerDir := filepath.Dir(path)
	file := filepath.Base(path)
	if file == "" {
		return nil, fmt.Errorf("%s must be a file path, not directory, for %s", path, name)
	}

	hostDir := p.hostDirOf(containerDir, mounts)
	if hostDir == "" {
		return nil, fmt.Errorf("in log %s: %s is not mount on host", name, path)
	}

	format := info.children["format"]
	if format == nil {
		format = newLogInfoNode("none")
	}

	formatConfig, err := Convert(format)
	if err != nil {
		return nil, fmt.Errorf("in log %s: format error: %v", name, err)
	}

	//特殊处理regex
	if format.value == "regexp" {
		format.value = fmt.Sprintf("/%s/", formatConfig["pattern"])
		delete(formatConfig, "pattern")
	}

	return &LogConfig{
		Name:         name,
		ContainerDir: containerDir,
		Format:       format.value,
		File:         file,
		Tags:         tagMap,
		HostDir:      filepath.Join(p.base, hostDir),
		FormatConfig: formatConfig,
		Target:       target,
		TimeKey:      timeKey,
		TimeFormat:   timeFormat,
		HostKey:      hostKey,
	}, nil
}

type LogInfoNode struct {
	value    string
	children map[string]*LogInfoNode
}

func newLogInfoNode(value string) *LogInfoNode {
	return &LogInfoNode{
		value:    value,
		children: make(map[string]*LogInfoNode),
	}
}

func (node *LogInfoNode) insert(keys []string, value string) error {
	if len(keys) == 0 {
		return nil
	}
	key := keys[0]
	if len(keys) > 1 {
		if child, ok := node.children[key]; ok {
			child.insert(keys[1:], value)
		} else {
			return fmt.Errorf("%s has no parent node", key)
		}
	} else {
		child := newLogInfoNode(value)
		node.children[key] = child
	}
	return nil
}

func (node *LogInfoNode) get(key string) string {
	if child, ok := node.children[key]; ok {
		return child.value
	}
	return ""
}

func (p *Pilot) getLogConfigs(jsonLogPath string, mounts []types.MountPoint, labels map[string]string) ([]*LogConfig, error) {
	var ret []*LogConfig

	mountsMap := make(map[string]types.MountPoint)
	for _, mount := range mounts {
		mountsMap[mount.Destination] = mount
	}

	var labelNames []string
	//sort keys
	for k, _ := range labels {
		labelNames = append(labelNames, k)
	}

	sort.Strings(labelNames)
	root := newLogInfoNode("")
	for _, k := range labelNames {
		if !strings.HasPrefix(k, LABEL_SERVICE_LOGS) || strings.Count(k, ".") == 1 {
			continue
		}
		logLabel := strings.TrimPrefix(k, LABEL_SERVICE_LOGS)
		if err := root.insert(strings.Split(logLabel, "."), labels[k]); err != nil {
			return nil, err
		}
	}

	for name, node := range root.children {
		logConfig, err := p.parseLogConfig(name, node, jsonLogPath, mountsMap)
		if err != nil {
			return nil, err
		}
		ret = append(ret, logConfig)
	}
	return ret, nil
}

func (p *Pilot) exists(containId string) bool {
	if _, err := os.Stat(p.pathOf(containId)); os.IsNotExist(err) {
		return false
	}
	return true
}

func (p *Pilot) render(containerId string, source Source, configList []*LogConfig) (string, error) {
	log.Infof("logs: %v", configList)
	var buf bytes.Buffer

	context := map[string]interface{}{
		"containerId": containerId,
		"configList":  configList,
		"source":      source,
	}
	if err := p.tpl.Execute(&buf, context); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (p *Pilot) reload() error {
	log.Info("Reload fluentd")
	return ReloadFluentd()
}

func Run(tpl string, baseDir string) error {
	p, err := New(tpl, baseDir)
	if err != nil {
		panic(err)
	}
	return p.watch()
}
