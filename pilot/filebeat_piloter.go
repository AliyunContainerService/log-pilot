package pilot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
)

// Global variables for FilebeatPiloter
const (
	FILEBEAT_EXEC_CMD  = "/usr/bin/filebeat"
	FILEBEAT_REGISTRY  = "/var/lib/filebeat/data/registry/filebeat/data.json"
	FILEBEAT_BASE_CONF = "/etc/filebeat"
	FILEBEAT_CONF_DIR  = FILEBEAT_BASE_CONF + "/prospectors.d"
	FILEBEAT_CONF_FILE = FILEBEAT_BASE_CONF + "/filebeat.yml"

	DOCKER_SYSTEM_PATH  = "/var/lib/docker/"
	KUBELET_SYSTEM_PATH = "/var/lib/kubelet/"

	ENV_FILEBEAT_OUTPUT = "FILEBEAT_OUTPUT"
)

var _ Piloter = (*FilebeatPiloter)(nil)

// FilebeatPiloter for filebeat plugin
type FilebeatPiloter struct {
	name           string
	baseDir        string
	watchDone      chan bool
	watchDuration  time.Duration
	watchContainer map[string]string
	fbExit         chan struct{}
	noticeStop     chan bool
	filebeat       *exec.Cmd
	mlock          sync.Mutex
}

// NewFilebeatPiloter returns a FilebeatPiloter instance
func NewFilebeatPiloter(baseDir string) (Piloter, error) {
	return &FilebeatPiloter{
		name:           PILOT_FILEBEAT,
		baseDir:        baseDir,
		watchDone:      make(chan bool),
		watchContainer: make(map[string]string, 0),
		watchDuration:  120 * time.Second,
		fbExit:         make(chan struct{}),
		noticeStop:     make(chan bool),
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
			log.Infof("%s watcher is stopping...", p.Name())
			p.noticeStop <- true

			//filebeat已经退出
			if p.filebeat == nil {
				log.Debug("Filebeat has exited")
				return nil
			}

			err := p.filebeat.Process.Kill()
			if err != nil {
				log.Errorf("Kill filebeat process faile, err: %v", err)
				pgroup := 0 - p.filebeat.Process.Pid
				syscall.Kill(pgroup, syscall.SIGKILL)
			}
			time.Sleep(3 * time.Second) // wait a little
			p.fbExit <- struct{}{}
			return err
		case <-time.After(p.watchDuration):
			log.Debugf("%s watcher scan", p.Name())
			go func() {
				err := p.scan()
				if err != nil {
					log.Errorf("%s watcher scan error: %v", p.Name(), err)
				}
			}()
		}
	}
}

func (p *FilebeatPiloter) scan() error {
	if len(p.watchContainer) == 0 {
		log.Debugf("No filebeat watch container, current scan end")
		return nil
	}

	states, err := p.getRegsitryState()
	if err != nil {
		log.Errorf("Get registry error: %v", err)
		return err
	}

	configPaths := p.loadConfigPaths()
	delConfs := make(map[string]string)
	delLogPaths := make(map[string]string)
	
	p.mlock.Lock()
	for container := range p.watchContainer {
		confPath := p.GetConfPath(container)
		if _, err := os.Stat(confPath); err != nil && os.IsNotExist(err) {
			log.Infof("log config %s.yml has been removed and ignore", container)
			delete(p.watchContainer, container)
		} else if logm, b := p.canRemoveConf(container, states, configPaths); b {
			// 在这里加入自定义的补充动作。
			// 这里config文件的清理动作做一个调整：
			// 不在循环中进行实际的文件删除动作，每次循环只记录要执行删除的container, 在循环结束后统一处理。
			delConfs[confPath] = container
			for log, c := range logm {
				delLogPaths[log] = c
			}
		}
	}
	p.mlock.Unlock()

	if len(delConfs) == 0 {
		log.Debugf("No filebeat config will modify, current scan end")
		return nil
	}

	// 对filebeat进行container释放清理操作
	p.Stop()   //停止filebeat
	<-p.fbExit //等待filebeat退出
	defer func() {
		time.Sleep(2 * time.Second)
		p.Start()
		time.Sleep(2 * time.Second)
	}()

	b, _ := ioutil.ReadFile(FILEBEAT_REGISTRY)
	origStates := make([]RegistryState, 0)
	newStates := make([]RegistryState, 0)
	if err := json.Unmarshal(b, &origStates); err != nil {
		log.Error("json error: ", err)
		return err
	}

	failDelContainers := make(map[string]bool)
	// 删除detroyed container的配置文件
	for delConf, container := range delConfs {
		log.Debug("start remove conf: ", delConf)
		if err := os.Remove(delConf); err != nil {
			log.Errorf("remove log config %s.yml fail: %v", container, err)
			failDelContainers[container] = true
		} else {
			log.Infof("%s removed, remove watchContainer map from `%s`", delConf, container)
			delete(p.watchContainer, container)
		}
	}

	// 更新registry文件
	for _, state := range origStates {
		if !FileExist(state.Source) {
			//当前的文件已经被删除了，可能是未清理的过期配置
			log.Debugf("logfile(%s) has been removed, the item could be deleted: %v", state.Source, state)
			continue
		} else if container, ok := delLogPaths[state.Source]; !ok {
			//当前state不是destroying container的log，需要继续保留
			newStates = append(newStates, state)
		} else if _, ok := failDelContainers[container]; ok {
			//当前state是destroying container的log，但是conf文件删除失败了，也需要继续保留
			newStates = append(newStates, state)
		}
	}
	nb, err := json.Marshal(newStates)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(FILEBEAT_REGISTRY, nb, 0600)
	return err
}

func (p *FilebeatPiloter) canRemoveConf(container string, registry map[string]RegistryState,
	configPaths map[string]string) (map[string]string, bool) {
	config, err := p.loadConfig(container)
	if err != nil {
		log.Error(err)
		return nil, false
	}

	delLogPaths := make(map[string]string)
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
					return nil, false
				} else if _, ok := configPaths[path]; !ok { // host path bind
					log.Infof("%s->%s does not finish to read and not exist in other config",
						container, logFile)
					return nil, false
				}
			}
			delLogPaths[logFile] = container
		}
	}
	return delLogPaths, true
}

func (p *FilebeatPiloter) loadConfig(container string) (*Config, error) {
	confPath := p.GetConfPath(container)
	c, err := yaml.NewConfigWithFile(confPath, configOpts...)
	if err != nil {
		log.Errorf("read %s.yml log config error: %v", container, err)
		return nil, err
	}

	var config Config
	var configs []Config
	var paths []string
	if err := c.Unpack(&configs); err != nil {
		log.Errorf("parse %s.yml log config error: %v", container, err)
		return nil, err
	}

	for _, c := range configs {
		paths = append(paths, c.Paths...)
	}
	config.Paths = paths
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
	p.mlock.Lock()
	if _, ok := p.watchContainer[containerID]; !ok {
		p.watchContainer[containerID] = containerID
		log.Infof("begin to watch log config: %s.yml", containerID)
	}
	p.mlock.Unlock()
	return nil
}

// Start starting and watching filebeat process
func (p *FilebeatPiloter) Start() error {
	log.Debug("Start the filebeat piloter")
	if err := p.start(); err != nil {
		return err
	}

	go func() {
		log.Infof("filebeat started: %v", p.filebeat.Process.Pid)
		for {
			select {
			case err := <-Func2Chan(p.filebeat.Wait):
				if err != nil {
					log.Errorf("filebeat exited: %v", err)
					if exitError, ok := err.(*exec.ExitError); ok {
						processState := exitError.ProcessState
						log.Errorf("filebeat exited pid: %v", processState.Pid())
					}
				}

				// try to restart filebeat
				log.Warningf("filebeat exited and try to restart")
				if err := p.start(); err != nil {
					//启动失败，重启piloter，通知watchDone
					p.Stop()
				}
			case <-p.noticeStop:
				return
			}
		}
	}()

	go p.watch()
	return nil
}

// start filebeat process
func (p *FilebeatPiloter) start() error {
	if p.filebeat != nil {
		pid := p.filebeat.Process.Pid
		process, err := os.FindProcess(pid)
		if err == nil {
			err = process.Signal(syscall.Signal(0))
			if err == nil {
				log.Infof("filebeat started, pid: %v", pid)
				return err
			}
		}
	}

	p.filebeat = nil
	log.Info("starting filebeat")
	cmd := exec.Command(FILEBEAT_EXEC_CMD, "-c", FILEBEAT_CONF_FILE)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		log.Errorf("filebeat start fail: %v", err)
		return err
	}
	p.filebeat = cmd
	return nil
}

// Stop log collection
func (p *FilebeatPiloter) Stop() error {
	log.Debug("Stop the filebeat piloter")
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