package pilot

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
	"time"
	"regexp"
	"io/ioutil"
	"bufio"
	"syscall"
	"io"
)

const (
	FLUME_EXEC_BIN = "/pilot/flume/bin/flume-ng"    // agent -n a1 -c conf -f conf/flume.properties
	FLUME_CONF_HOME = "/pilot/flume/conf"
	FLUME_CONF_DIR = FLUME_CONF_HOME + "/tmp"       // 存放各个容器的日志采集配置文件
	FLUME_CONF_FILE = FLUME_CONF_HOME + "/flume-conf.properties"

	ENV_FLUME_OUTPUT = "FLUME_OUTPUT"
)

var flume *exec.Cmd

type FlumePiloter struct {
	name           string
}

func NewFlumePiloter() (Piloter, error) {
	return &FlumePiloter{
		name:           PILOT_FLUME,
	}, nil
}

func (p *FlumePiloter) Start() error {
	if flume != nil {
		pid := flume.Process.Pid
		log.Infof("flume started, pid: %v", pid)
		return fmt.Errorf(ERR_ALREADY_STARTED)
	}

	log.Info("start generate flume conf")
	err := p.GenConf(FLUME_CONF_HOME)
	if err != nil{
		log.Errorf("flume conf error in start : %v", err)
		//return err
	} else {
		//agent -n a1 -c conf -f conf/flume.properties
		log.Info("starting flume")
		flume = exec.Command(FLUME_EXEC_BIN, "agent",
			"-c", FLUME_CONF_HOME,
			fmt.Sprintf("-Dlog4j.configuration=file:%s/log4j.properties", FLUME_CONF_HOME),
			"-n", "a1",
			"-f", FLUME_CONF_FILE)
		flume.Stderr = os.Stderr
		flume.Stdout = os.Stdout
		err := flume.Start()
		if err != nil {
			log.Errorf("flume start fail: %v", err)
		}

		go func() {
			log.Infof("flume started: %v", flume.Process.Pid)
			err := flume.Wait()
			if err != nil {
				log.Errorf("flume exited: %v", err)
				if exitError, ok := err.(*exec.ExitError); ok {
					processState := exitError.ProcessState
					log.Errorf("flume exited pid: %v", processState.Pid())
				}
			}

			// try to restart flume
			log.Warningf("flume exited and try to restart")
			flume = nil
			//time.Sleep(5 * time.Second)
			p.Start()
		}()
	}

	return err
}

func (p *FlumePiloter) Stop() error {
	pid := flume.Process.Pid
	command := fmt.Sprintf("ps -ef | grep %d | grep flume | grep -v grep | head 1 | awk '{print $1}'", pid)
	childId := flumeFindShell(command)
	log.Infof("before stop flume childId : %s", childId)
	flume.Process.Signal(syscall.SIGHUP)
	time.Sleep(5 * time.Second)
	afterChildId := shell(command)
	log.Infof("after stop flume childId : %s", afterChildId)
	if childId == afterChildId {
		log.Infof("kill childId : %s", childId)
		shell("kill -9 " + childId)
	}
	return nil
}

func (p *FlumePiloter) Reload() error {
	if flume == nil {
		err := fmt.Errorf("flume have not started")
		log.Error(err)
		return err
	}

	log.Info("reload flume")
	ch := make(chan struct{})
	go func(pid int) {
		// jps | grep Application | grep pid | grep -v grep | awk '{print $1}'  测试下哪个好用
		command := fmt.Sprintf("ps -ef | grep %d | grep flume | grep -v grep | head 1 | awk '{print $1}'", pid)
		childId := flumeFindShell(command)
		log.Infof("before reload flume childId : %s", childId)
		flume.Process.Signal(syscall.SIGHUP)
		time.Sleep(5 * time.Second)
		afterChildId := flumeFindShell(command)
		log.Infof("after reload flume childId : %s", afterChildId)
		if childId == afterChildId {
			log.Infof("kill childId : %s", childId)
			flumeFindShell("kill -9 " + childId)
		}
		close(ch)
	}(flume.Process.Pid)
	<-ch
	return nil
}

func (p *FlumePiloter) GetConfPath(container string) string {
	return fmt.Sprintf("%s/%s.properties", FLUME_CONF_DIR, container)
}

func (p *FlumePiloter) GetConfHome() string {
	return FLUME_CONF_DIR
}

func (p *FlumePiloter) Name() string {
	return p.name
}

func flumeFindShell(command string) string {
	cmd := exec.Command("/bin/sh", "-c", command)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("error %v", err)
	}
	return string(out)
}

func (p *FlumePiloter) GenConf(path string) error {
	tmpConf, err := ioutil.ReadDir(fmt.Sprintf("%s/tmp", path))
	if err != nil {
		log.Error(err)
		err := fmt.Errorf("flume conf read tmp path error")
		//log.Error(err)
		return err
	}

	// 正则取出source channel sink name
	sourceRep, _ := regexp.Compile("([0-9a-zA-Z]+_[0-9a-zA-Z]+_source)(.+=.+)")
	channelRep, _ := regexp.Compile("([0-9a-zA-Z]+_[0-9a-zA-Z]+_channel)(.+=.+)")
	sinkRep, _ := regexp.Compile("([0-9a-zA-Z]+_[0-9a-zA-Z]+_sink)(.+=.+)")

	var sources = map[string]string{}
	var channels = map[string]string{}
	var sinks = map[string]string{}
	var buf []byte
	for _, file := range tmpConf {
		fi, err := os.Open(fmt.Sprintf("%s/tmp/%s", path, file.Name()))
		if err != nil {
			fmt.Printf("read tmp conf error: %s\n", err)
			return nil
		}
		defer fi.Close()

		br := bufio.NewReader(fi)
		for {
			line, _, c := br.ReadLine()
			if c == io.EOF {
				break
			}
			buf = append(buf, line...)
			buf = append(buf, []byte("\n")...)

			source := sourceRep.FindStringSubmatch(string(line))
			if source != nil {
				if _, no := sources[source[1]]; !no {
					sources[source[1]] = source[1]
				}
				//continue
			}

			channel := channelRep.FindStringSubmatch(string(line))
			if channel != nil {
				if _, no := channels[channel[1]]; !no {
					channels[channel[1]] = channel[1]
				}
				//continue
			}

			sink := sinkRep.FindStringSubmatch(string(line))
			if sink != nil {
				if _, no := sinks[sink[1]]; !no {
					sinks[sink[1]] = sink[1]
				}
				//continue
			}

		}
	}

	var resultBuf []byte
	var sourceBuf []byte
	var channelBuf []byte
	var sinkBuf []byte
	for k, _ := range sources {
		sourceBuf = append(sourceBuf, []byte(" " + k)...)
	}
	for k, _ := range channels {
		channelBuf = append(channelBuf, []byte(" " + k)...)
	}
	for k, _ := range sinks {
		sinkBuf = append(sinkBuf, []byte(" " + k)...)
	}

	if len(sourceBuf) > 0 {
		resultBuf = append(resultBuf, []byte("a1.sources =")...)
		resultBuf = append(resultBuf, sourceBuf...)
		resultBuf = append(resultBuf, []byte("\n")...)
	}
	if len(channelBuf) > 0 {
		resultBuf = append(resultBuf, []byte("a1.channels =")...)
		resultBuf = append(resultBuf, channelBuf...)
		resultBuf = append(resultBuf, []byte("\n")...)
	}
	if len(sinkBuf) > 0 {
		resultBuf = append(resultBuf, []byte("a1.sinks =")...)
		resultBuf = append(resultBuf, sinkBuf...)
		resultBuf = append(resultBuf, []byte("\n")...)
	}
	resultBuf = append(resultBuf, buf...)

	if err = ioutil.WriteFile(fmt.Sprintf("%s/flume-conf.properties", path), resultBuf, os.FileMode(0644)); err != nil {
		return err
	}
	return nil
}

func (p *FlumePiloter) OnDestroyEvent(container string) error {
	log.Info("refactor in the future!!!")
	return nil
}