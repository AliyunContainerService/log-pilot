{{range .configList}}
<source>
  @type tail
  tag docker.{{ $.containerId }}.{{ .Name }}
  path {{ .HostDir }}/{{ .File }}
  format {{ .Format }}

{{if .FormatConfig}}
{{range $key, $value := .FormatConfig}}
{{ $key }} {{ $value }}
{{end}}
{{end}}
  pos_file /pilot/pos/fluentd.pos
  refresh_interval 1
</source>

<filter docker.{{ $.containerId }}.{{ .Name }}>
@type add_time
time_key {{ .TimeKey}}
time_format {{ .TimeFormat}}
</filter>


<filter docker.{{ $.containerId }}.{{ .Name }}>
@type record_transformer
<record>
{{ .HostKey}} "#{Socket.gethostname}"
{{range $key, $value := .Tags}}
{{ $key }} {{ $value }}
{{end}}

@target {{if .Target}}{{.Target}}{{else}}{{ .Name }}{{end}}

{{if $.source.Application}}docker_app {{ $.source.Application }} {{end}}
{{if $.source.Service}}docker_service {{ $.source.Service }} {{end}}
{{if $.source.POD}}k8s_pod {{ $.source.POD }} {{end}}
{{if $.source.Container}}docker_container {{ $.source.Container }} {{end}}
</record>
</filter>
{{end}}
