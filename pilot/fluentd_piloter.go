package pilot

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
	"time"
	"strconv"
	"strings"
)

const (
	FLUENTD_EXEC_CMD  = "/usr/bin/fluentd"
	FLUENTD_BASE_CONF = "/etc/fluentd"
	FLUENTD_CONF_DIR  = FLUENTD_BASE_CONF + "/conf.d"
	FLUENTD_CONF_FILE = FLUENTD_BASE_CONF + "/fluentd.conf"
	FLUENTD_PLUGINS   = FLUENTD_BASE_CONF + "/plugins"

	ENV_FLUENTD_OUTPUT = "FLUENTD_OUTPUT"
	ENV_FLUENTD_WORKER = "FLUENTD_WORKER"
)

var fluentd *exec.Cmd

type FluentdPiloter struct {
	name string
}

func NewFluentdPiloter() (Piloter, error) {
	return &FluentdPiloter{
		name: PILOT_FLUENTD,
	}, nil
}

func (p *FluentdPiloter) Start() error {
	if fluentd != nil {
		pid := fluentd.Process.Pid
		log.Infof("fluentd started, pid: %v", pid)
		return fmt.Errorf(ERR_ALREADY_STARTED)
	}

	log.Info("starting fluentd")
	worker := os.Getenv(ENV_FLUENTD_WORKER)
	if _, err := strconv.Atoi(worker); worker == "" || err != nil {
		worker = "1"
	}

	fluentd = exec.Command(FLUENTD_EXEC_CMD,
		"-c", FLUENTD_CONF_FILE,
		"-p", FLUENTD_PLUGINS,
		"--workers", worker)
	fluentd.Stderr = os.Stderr
	fluentd.Stdout = os.Stdout
	err := fluentd.Start()
	if err != nil {
		log.Errorf("fluentd start fail: %v", err)
	}

	go func() {
		err := fluentd.Wait()
		if err != nil {
			log.Errorf("fluentd exited: %v", err)
			if exitError, ok := err.(*exec.ExitError); ok {
				processState := exitError.ProcessState
				log.Errorf("fluentd exited pid: %v", processState.Pid())
			}
		}

		// try to restart fluentd
		log.Warningf("fluentd exited and try to restart")
		fluentd = nil
		p.Start()
	}()
	return err
}

func (p *FluentdPiloter) Stop() error {
	return nil
}

func (p *FluentdPiloter) Reload() error {
	if fluentd == nil {
		err := fmt.Errorf("fluentd have not started")
		log.Error(err)
		return err
	}

	log.Info("reload fluentd")
	ch := make(chan struct{})
	go func(pid int) {
		command := fmt.Sprintf("pgrep -P %d", pid)
		childId := shell(command)
		log.Infof("before reload childId : %s", childId)
		fluentd.Process.Signal(syscall.SIGHUP)
		time.Sleep(5 * time.Second)
		afterChildId := shell(command)
		log.Infof("after reload childId : %s", afterChildId)
		if childId == afterChildId {
			log.Infof("kill childId : %s", childId)
			shell("kill -9 " + childId)
		}
		close(ch)
	}(fluentd.Process.Pid)
	<-ch
	return nil
}

func (p *FluentdPiloter) GetConfPath(container string) string {
	return fmt.Sprintf("%s/%s.conf", FLUENTD_CONF_DIR, container)
}

func shell(command string) string {
	cmd := exec.Command("/bin/sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("error %v", err)
	}
	return strings.TrimSpace(string(out))
}

func (p *FluentdPiloter) GetConfHome() string {
	return FLUENTD_CONF_DIR
}

func (p *FluentdPiloter) Name() string {
	return p.name
}

func (p *FluentdPiloter) OnDestroyEvent(container string) error {
	log.Info("refactor in the future!!!")
	return nil
}
