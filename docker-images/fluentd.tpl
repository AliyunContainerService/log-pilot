{{range .configList}}
<source>
  @type tail
  tag {{ $.containerId }}.{{ .Name }}
  path {{ .HostDir }}/{{ .File }}
  format {{ .Format }}
  {{if .TimeFormat}} time_format {{ .TimeFormat }}  {{end}}
  pos_file /pilot/pos/fluentd.pos
  refresh_interval 5
</source>

<filter {{ $.containerId }}.{{ .Name }}>
@type record_transformer
<record>
host "#{Socket.gethostname}"
{{range $key, $value := .Tags}}
{{ $key }} {{ $value }}
{{end}}

{{if not .Tags.target}}target {{if $.source.Application}}{{ $.source.Application }}-{{end}}{{ .Name }}{{end}}

{{if $.source.Application}}docker_app {{ $.source.Application }} {{end}}
{{if $.source.Service}}docker_service {{ $.source.Service }} {{end}}
{{if $.source.Container}}docker_container {{ $.source.Container }} {{end}}
</record>
</filter>
{{end}}
