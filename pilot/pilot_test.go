package pilot

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
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
	pilot := &Pilot{}
	labels := map[string]string{}
	configs := pilot.getLogConfigs(labels)
	c.Assert(configs, check.HasLen, 0)

	labels = map[string]string{
		"aliyun.logs.hello":        "/var/log/hello.log",
		"aliyun.logs.hello.format": "json",
	}
	configs = pilot.getLogConfigs(labels)
	c.Assert(configs, check.HasLen, 1)
	c.Assert(configs[0].Format, check.Equals, "json")
	c.Assert(configs[0].ContainerDir, check.Equals, "/var/log")
	c.Assert(configs[0].File, check.Equals, "hello.log")
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

	configs := []LogConfig{
		LogConfig{
			Name:    "hello",
			HostDir: "/path/to/hello",
			File:    "hello.log",
		},
		LogConfig{
			Name:    "world",
			File:    "world.log",
			HostDir: "/path/to/world",
		},
	}
	pilot, err := New(template)
	c.Assert(err, check.IsNil)
	result, err := pilot.render("id-1111", configs)
	c.Assert(err, check.IsNil)
	fmt.Print(result)
}
