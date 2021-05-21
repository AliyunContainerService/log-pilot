package pilot

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Global variables for FilebeatPiloter
const (
	FILEBEAT_EXEC_CMD  = "/usr/bin/filebeat"
	FILEBEAT_REGISTRY  = "/var/lib/filebeat/registry"
	FILEBEAT_BASE_CONF = "/etc/filebeat"
	FILEBEAT_CONF_DIR  = FILEBEAT_BASE_CONF + "/prospectors.d"
	FILEBEAT_CONF_FILE = FILEBEAT_BASE_CONF + "/filebeat.yml"

	DOCKER_SYSTEM_PATH  = "/var/lib/docker/"
	KUBELET_SYSTEM_PATH = "/var/lib/kubelet/"

	ENV_FILEBEAT_OUTPUT = "FILEBEAT_OUTPUT"
)

var filebeat *exec.Cmd
var _ Piloter = (*FilebeatPiloter)(nil)

// FilebeatPiloter for filebeat plugin
type FilebeatPiloter struct {
	name           string
	baseDir        string
	watchDone      chan bool
	watchDuration  time.Duration
	watchContainer map[string]string
}

// NewFilebeatPiloter returns a FilebeatPiloter instance
func NewFilebeatPiloter(baseDir string) (Piloter, error) {
	return &FilebeatPiloter{
		name:           PILOT_FILEBEAT,
		baseDir:        baseDir,
		watchDone:      make(chan bool),
		watchContainer: make(map[string]string, 0),
		watchDuration:  60 * time.Second,
	}, nil
}

var configOpts = []ucfg.Option{
	ucfg.PathSep("."),
	ucfg.ResolveEnv,
	ucfg.VarExp,
}

// Config contains all log paths
type Config struct {
	Paths []string `config:"paths"`
}

// FileInode identify a unique log file
type FileInode struct {
	Inode  uint64 `json:"inode,"`
	Device uint64 `json:"device,"`
}

// RegistryState represents log offsets
type RegistryState struct {
	Source      string        `json:"source"`
	Offset      int64         `json:"offset"`
	Timestamp   time.Time     `json:"timestamp"`
	TTL         time.Duration `json:"ttl"`
	Type        string        `json:"type"`
	FileStateOS FileInode
}

func (p *FilebeatPiloter) watch() error {
	log.Infof("%s watcher start", p.Name())
	for {
		select {
		case <-p.watchDone:
			log.Infof("%s watcher stop", p.Name())
			return nil
		case <-time.After(p.watchDuration):
			//log.Debugf("%s watcher scan", p.Name())
			err := p.scan()
			if err != nil {
				log.Errorf("%s watcher scan error: %v", p.Name(), err)
			}
		}
	}
}

func (p *FilebeatPiloter) scan() error {
	states, err := p.getRegsitryState()
	if err != nil {
		return nil
	}

	configPaths := p.loadConfigPaths()
	for container := range p.watchContainer {
		confPath := p.GetConfPath(container)
		if _, err := os.Stat(confPath); err != nil && os.IsNotExist(err) {
			log.Infof("log config %s.yml has been removed and ignore", container)
			delete(p.watchContainer, container)
		} else if p.canRemoveConf(container, states, configPaths) {
			log.Infof("try to remove log config %s.yml", container)
			if err := os.Remove(confPath); err != nil {
				log.Errorf("remove log config %s.yml fail: %v", container, err)
			} else {
				delete(p.watchContainer, container)
			}
		}
	}
	return nil
}

func (p *FilebeatPiloter) canRemoveConf(container string, registry map[string]RegistryState,
	configPaths map[string]string) bool {
	config, err := p.loadConfig(container)
	if err != nil {
		return false
	}

	for _, path := range config.Paths {
		autoMount := p.isAutoMountPath(filepath.Dir(path))
		logFiles, _ := filepath.Glob(path)
		for _, logFile := range logFiles {
			info, err := os.Stat(logFile)
			if err != nil && os.IsNotExist(err) {
				continue
			}
			if _, ok := registry[logFile]; !ok {
				log.Warnf("%s->%s registry not exist", container, logFile)
				continue
			}
			if registry[logFile].Offset < info.Size() {
				if autoMount { // ephemeral logs
					log.Infof("%s->%s does not finish to read", container, logFile)
					return false
				} else if _, ok := configPaths[path]; !ok { // host path bind
					log.Infof("%s->%s does not finish to read and not exist in other config",
						container, logFile)
					return false
				}
			}
		}
	}
	return true
}

func (p *FilebeatPiloter) loadConfig(container string) (*Config, error) {
	confPath := p.GetConfPath(container)
	c, err := yaml.NewConfigWithFile(confPath, configOpts...)
	if err != nil {
		log.Errorf("read %s.yml log config error: %v", container, err)
		return nil, err
	}

	var config Config
	if err := c.Unpack(&config); err != nil {
		log.Errorf("parse %s.yml log config error: %v", container, err)
		return nil, err
	}
	return &config, nil
}

func (p *FilebeatPiloter) loadConfigPaths() map[string]string {
	paths := make(map[string]string, 0)
	confs, _ := ioutil.ReadDir(p.GetConfHome())
	for _, conf := range confs {
		container := strings.TrimRight(conf.Name(), ".yml")
		if _, ok := p.watchContainer[container]; ok {
			continue // ignore removed container
		}

		config, err := p.loadConfig(container)
		if err != nil || config == nil {
			continue
		}

		for _, path := range config.Paths {
			if _, ok := paths[path]; !ok {
				paths[path] = container
			}
		}
	}
	return paths
}

func (p *FilebeatPiloter) isAutoMountPath(path string) bool {
	dockerVolumePattern := fmt.Sprintf("^%s.*$", filepath.Join(p.baseDir, DOCKER_SYSTEM_PATH))
	if ok, _ := regexp.MatchString(dockerVolumePattern, path); ok {
		return true
	}

	kubeletVolumePattern := fmt.Sprintf("^%s.*$", filepath.Join(p.baseDir, KUBELET_SYSTEM_PATH))
	ok, _ := regexp.MatchString(kubeletVolumePattern, path)
	return ok
}

func (p *FilebeatPiloter) getRegsitryState() (map[string]RegistryState, error) {
	f, err := os.Open(FILEBEAT_REGISTRY)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	states := make([]RegistryState, 0)
	err = decoder.Decode(&states)
	if err != nil {
		return nil, err
	}

	statesMap := make(map[string]RegistryState, 0)
	for _, state := range states {
		if _, ok := statesMap[state.Source]; !ok {
			statesMap[state.Source] = state
		}
	}
	return statesMap, nil
}

func (p *FilebeatPiloter) feed(containerID string) error {
	if _, ok := p.watchContainer[containerID]; !ok {
		p.watchContainer[containerID] = containerID
		log.Infof("begin to watch log config: %s.yml", containerID)
	}
	return nil
}

// Start starting and watching filebeat process
func (p *FilebeatPiloter) Start() error {
	if filebeat != nil {
		pid := filebeat.Process.Pid
		log.Infof("filebeat started, pid: %v", pid)
		return fmt.Errorf(ERR_ALREADY_STARTED)
	}

	log.Info("starting filebeat")
	filebeat = exec.Command(FILEBEAT_EXEC_CMD, "-c", FILEBEAT_CONF_FILE)
	filebeat.Stderr = os.Stderr
	filebeat.Stdout = os.Stdout
	err := filebeat.Start()
	if err != nil {
		log.Errorf("filebeat start fail: %v", err)
	}

	go func() {
		log.Infof("filebeat started: %v", filebeat.Process.Pid)
		err := filebeat.Wait()
		if err != nil {
			log.Errorf("filebeat exited: %v", err)
			if exitError, ok := err.(*exec.ExitError); ok {
				processState := exitError.ProcessState
				log.Errorf("filebeat exited pid: %v", processState.Pid())
			}
		}

		// try to restart filebeat
		log.Warningf("filebeat exited and try to restart")
		filebeat = nil
		p.Start()
	}()

	go p.watch()
	return err
}

// Stop log collection
func (p *FilebeatPiloter) Stop() error {
	p.watchDone <- true
	return nil
}

// Reload reload configuration file
func (p *FilebeatPiloter) Reload() error {
	log.Debug("do not need to reload filebeat")
	return nil
}

// GetConfPath returns log configuration path
func (p *FilebeatPiloter) GetConfPath(container string) string {
	return fmt.Sprintf("%s/%s.yml", FILEBEAT_CONF_DIR, container)
}

// GetConfHome returns configuration directory
func (p *FilebeatPiloter) GetConfHome() string {
	return FILEBEAT_CONF_DIR
}

// Name returns plugin name
func (p *FilebeatPiloter) Name() string {
	return p.name
}

// OnDestroyEvent watching destroy event
func (p *FilebeatPiloter) OnDestroyEvent(container string) error {
	return p.feed(container)
}

// GetBaseConf returns plugin root directory
func (p *FilebeatPiloter) GetBaseConf() string {
	return FILEBEAT_BASE_CONF
}
