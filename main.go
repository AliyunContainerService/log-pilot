package main

import (
	"flag"
	"path/filepath"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"os"
	"github.com/jzwlqx/fluentd-pilot/pilot"
)

func main() {

	template := flag.String("template", "", "Template filepath for fluentd.")
	base := flag.String("base", "", "Directory which mount host root.")
	level := flag.String("log-level", "INFO", "Log level")
	flag.Parse()

	baseDir, err := filepath.Abs(*base)
	if err != nil {
		panic(err)
	}

	if baseDir == "/" {
		baseDir = ""
	}

	if *template ==  "" {
		panic("template file can not be emtpy")
	}

	log.SetOutput(os.Stdout)
	logLevel, err := log.ParseLevel(*level)
	if err != nil {
		panic(err)
	}
	log.SetLevel(logLevel)

	b, err := ioutil.ReadFile(*template)
	if err!= nil {
		panic(err)
	}

	log.Fatal(pilot.Run(string(b), baseDir))
}
