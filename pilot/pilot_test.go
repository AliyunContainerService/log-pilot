package pilot

import (
	log "github.com/sirupsen/logrus"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gopkg.in/check.v1"
	"os"
	"testing"
)

func Test(t *testing.T) {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	check.TestingT(t)
}

type PilotSuite struct{}

var _ = check.Suite(&PilotSuite{})

func (p *PilotSuite) TestGetLogConfigs(c *check.C) {
	pilot := &Pilot{
		logPrefix: []string{"aliyun"},
	}

	labels := map[string]string{}
	configs, err := pilot.getLogConfigs("/path/to/json.log", []specs.Mount{}, labels)
	c.Assert(err, check.IsNil)
	c.Assert(configs, check.HasLen, 0)

	labels = map[string]string{
		"aliyun.logs.hello":                    "/var/log/hello.log",
		"aliyun.logs.hello.format":             "json",
		"aliyun.logs.hello.tags":               "name=hello,stage=test",
		"aliyun.logs.hello.format.time_format": "%Y-%m-%d",
	}

	//no mount
	configs, err = pilot.getLogConfigs("/path/to/json.log", []specs.Mount{}, labels)
	c.Assert(err, check.NotNil)

	mounts := []specs.Mount{
		{
			Source:      "/host",
			Destination: "/var/log",
		},
	}
	configs, err = pilot.getLogConfigs("/path/to/json.log", mounts, labels)
	c.Assert(err, check.IsNil)
	c.Assert(configs, check.HasLen, 1)
	c.Assert(configs[0].Format, check.Equals, "json")
	c.Assert(configs[0].ContainerDir, check.Equals, "/var/log")
	c.Assert(configs[0].File, check.Equals, "hello.log")
	c.Assert(configs[0].Tags, check.HasLen, 4)
	c.Assert(configs[0].FormatConfig, check.HasLen, 2)

	//Test regex format
	labels = map[string]string{
		"aliyun.logs.hello":                "/var/log/hello.log",
		"aliyun.logs.hello.format":         "regexp",
		"aliyun.logs.hello.tags":           "name=hello,stage=test",
		"aliyun.logs.hello.format.pattern": "(?=name:hello).*",
	}
	configs, err = pilot.getLogConfigs("/path/to/json.log", mounts, labels)
	c.Assert(err, check.IsNil)
	c.Assert(configs[0].Format, check.Equals, "/(?=name:hello).*/")
}

func (p *PilotSuite) TestRender(c *check.C) {
	template := `
	{{range .configList}}
	<source>
	  @type tail
	  tag {{ .Name }} {{ $.containerId }}
	  path {{ .HostDir }}/{{ .File }}
	  pos_file /var/log/fluentd.pos
	  refresh_interval 5
	</source>
	{{end}}
	`

	configs := []*LogConfig{
		{
			Name:    "hello",
			HostDir: "/path/to/hello",
			File:    "hello.log",
		},
		{
			Name:    "world",
			File:    "world.log",
			HostDir: "/path/to/world",
		},
	}
	os.Setenv(ENV_PILOT_TYPE, PILOT_FILEBEAT)
	pilot, err := New(template, "/")
	c.Assert(err, check.IsNil)
	_, err = pilot.render("id-1111", map[string]string{}, configs)
	c.Assert(err, check.IsNil)
}
