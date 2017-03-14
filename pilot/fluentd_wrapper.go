package pilot

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

var fluentd *exec.Cmd

func StartFluentd() error {
	if fluentd != nil {
		return fmt.Errorf("fluentd already started")
	}
	log.Warn("start fluentd")
	fluentd = exec.Command("/usr/bin/fluentd", "-c", "/etc/fluentd/fluentd.conf", "-p", "/etc/fluentd/plugins")
	fluentd.Stderr = os.Stderr
	fluentd.Stdout = os.Stdout
	return fluentd.Start()
}

func ReloadFluentd() error {
	if fluentd == nil {
		return fmt.Errorf("fluentd have not started")
	}
	log.Warn("reload fluentd")
	return fluentd.Process.Signal(syscall.SIGHUP)
}
