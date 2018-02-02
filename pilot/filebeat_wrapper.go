package pilot

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/elastic/go-ucfg"
	"github.com/elastic/go-ucfg/yaml"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const PILOT_FILEBEAT = "filebeat"
const FILEBEAT_HOME = "/usr/share/filebeat"
const FILEBEAT_CONF_HOME = FILEBEAT_HOME
const FILEBEAT_CONF_DIR = FILEBEAT_CONF_HOME + "/prospectors.d"
const FILEBEAT_CONF_FILE = FILEBEAT_CONF_HOME + "/filebeat.yml"
const FILEBEAT_LOG_DIR = FILEBEAT_HOME + "/logs"
const FILEBEAT_DATA_DIR = FILEBEAT_HOME + "/data"
const FILEBEAT_EXEC_BIN = FILEBEAT_HOME + "/filebeat"
const FILEBEAT_REGISTRY_FILE = FILEBEAT_HOME + "/registry"

var filebeat *exec.Cmd

type FilebeatPiloter struct {
	name           string
	watchDone      chan bool
	watchDuration  time.Duration
	watchContainer map[string]string
}

func NewFilebeatPiloter() (Piloter, error) {
	return &FilebeatPiloter{
		name:           PILOT_FILEBEAT,
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

type Config struct {
	Paths []string `config:"paths"`
}

type FileInode struct {
	Inode  uint64 `json:"inode,"`
	Device uint64 `json:"device,"`
}

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
	return nil
}

func (p *FilebeatPiloter) scan() error {
	states, err := p.getRegsitryState()
	if err != nil {
		return nil
	}

	for container := range p.watchContainer {
		confPath := p.ConfPathOf(container)
		if _, err := os.Stat(confPath); err != nil && os.IsNotExist(err) {
			log.Infof("log config %s.yml has removed and ignore", container)
			delete(p.watchContainer, container)
			continue
		}

		c, err := yaml.NewConfigWithFile(confPath, configOpts...)
		if err != nil {
			log.Errorf("read %s.yml log config error: %v", container, err)
			continue
		}

		var config Config
		if err := c.Unpack(&config); err != nil {
			log.Errorf("parse %s.yml log config error: %v", container, err)
			continue
		}

		finished := true
		for _, path := range config.Paths {
			log.Debugf("scan %s log path: %s", container, path)
			files, _ := filepath.Glob(path)
			for _, file := range files {
				info, err := os.Stat(file)
				if err != nil && os.IsNotExist(err) {
					log.Infof("%s->%s not exist", container, file)
					continue
				}
				if _, ok := states[file]; !ok {
					log.Infof("%s->%s registry not exist", container, file)
					continue
				}
				if states[file].Offset < info.Size() {
					log.Infof("%s->%s has not read finished", container, file)
					finished = false
					break
				}
				log.Infof("%s->%s has read finished", container, file)
			}
			if !finished {
				break
			}
		}

		if !finished {
			log.Infof("ignore to remove log config %s.yml", container)
			continue
		}

		log.Infof("try to remove log config %s.yml", container)
		if err := os.Remove(confPath); err != nil {
			log.Errorf("remove log config failure %s.yml", container)
			continue
		}
		delete(p.watchContainer, container)
	}
	return nil
}

func (p *FilebeatPiloter) getRegsitryState() (map[string]RegistryState, error) {
	f, err := os.Open(FILEBEAT_REGISTRY_FILE)
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

func (p *FilebeatPiloter) Start() error {
	if filebeat != nil {
		return fmt.Errorf(ERR_ALREADY_STARTED)
	}

	log.Info("start filebeat")
	filebeat = exec.Command(FILEBEAT_EXEC_BIN, "-c", FILEBEAT_CONF_FILE)
	filebeat.Stderr = os.Stderr
	filebeat.Stdout = os.Stdout
	err := filebeat.Start()
	if err != nil {
		log.Error(err)
	}

	go func() {
		err := filebeat.Wait()
		if err != nil {
			log.Error(err)
		}
	}()

	go p.watch()
	return err
}

func (p *FilebeatPiloter) Stop() error {
	p.watchDone <- true
	return nil
}

func (p *FilebeatPiloter) Reload() error {
	log.Debug("not need to reload filebeat")
	return nil
}

func (p *FilebeatPiloter) ConfPathOf(container string) string {
	return fmt.Sprintf("%s/%s.yml", FILEBEAT_CONF_DIR, container)
}

func (p *FilebeatPiloter) ConfHome() string {
	return FILEBEAT_CONF_DIR
}

func (p *FilebeatPiloter) Name() string {
	return p.name
}

func (p *FilebeatPiloter) OnDestroyEvent(container string) error {
	return p.feed(container)
}
