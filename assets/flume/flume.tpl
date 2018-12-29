{{range .configList}}

a1.sources.{{ $.containerId }}_{{ .Name }}_source.type = TAILDIR
a1.sources.{{ $.containerId }}_{{ .Name }}_source.channels = {{ $.containerId }}_{{ .Name }}_channel
a1.sources.{{ $.containerId }}_{{ .Name }}_source.positionFile = /flume/log_meta/source/{{ $.containerId }}/{{ .Name }}/taildir_position.json
a1.sources.{{ $.containerId }}_{{ .Name }}_source.filegroups = f1
a1.sources.{{ $.containerId }}_{{ .Name }}_source.filegroups.f1 = {{ .HostDir }}/{{ .File }}

a1.channels.{{ $.containerId }}_{{ .Name }}_channel.type = file
a1.channels.{{ $.containerId }}_{{ .Name }}_channel.checkpointDir = /flume/log_meta/channel/{{ $.containerId }}/{{ .Name }}/checkpoint
a1.channels.{{ $.containerId }}_{{ .Name }}_channel.dataDirs = /flume/log_meta/channel/{{ $.containerId }}/{{ .Name }}/buffer

a1.sinks.{{ $.containerId }}_{{ .Name }}_sink.type = file_roll
a1.sinks.{{ $.containerId }}_{{ .Name }}_sink.channel = {{ $.containerId }}_{{ .Name }}_channel
a1.sinks.{{ $.containerId }}_{{ .Name }}_sink.sink.directory = {{ $.output }}/{{ index $.container "docker_container" }}
a1.sinks.{{ $.containerId }}_{{ .Name }}_sink.sink.rollInterval = 3600000
a1.sinks.{{ $.containerId }}_{{ .Name }}_sink.sink.pathManager.prefix = {{ .Name }}
a1.sinks.{{ $.containerId }}_{{ .Name }}_sink.sink.pathManager.extension = log

{{end}}