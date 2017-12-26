package pilot

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var fluentd *exec.Cmd

const ERR_ALREADY_STARTED = "fluentd already started"

func StartFluentd() error {
	if fluentd != nil {
		return fmt.Errorf(ERR_ALREADY_STARTED)
	}

	log.Info("start fluentd")
	fluentd = exec.Command("/usr/bin/fluentd", "-c", "/etc/fluentd/fluentd.conf", "-p", "/etc/fluentd/plugins")
	fluentd.Stderr = os.Stderr
	fluentd.Stdout = os.Stdout
	err := fluentd.Start()
	if err != nil {
		log.Error(err)
	}
	go func() {
		err := fluentd.Wait()
		if err != nil {
			log.Error(err)
		}
	}()
	return err
}

func shell(command string) string {
	cmd := exec.Command("/bin/sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("error %v", err)
	}
	return string(out)
}

func ReloadFluentd() error {
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
		log.Infof("after reload childId : %s", childId)
		if childId == afterChildId {
			log.Infof("kill childId : %s", childId)
			shell("kill -9 " + childId)
		}
		close(ch)
	}(fluentd.Process.Pid)
	<-ch
	return nil
}
