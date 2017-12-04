{{range .configList}}
<source>
  @type tail
  tag docker.{{ $.containerId }}.{{ .Name }}
  path {{ .HostDir }}/{{ .File }}
  format {{ .Format }}
  read_from_head true

  {{if .FormatConfig}}
  {{range $key, $value := .FormatConfig}}
  {{ $key }} {{ $value }}
  {{end}}
  {{end}}
  pos_file /pilot/pos/{{ $.containerId }}.{{ .Name }}.pos
</source>

<filter docker.{{ $.containerId }}.{{ .Name }}>
  @type add_time
  time_key {{ .TimeKey}}
</filter>

<filter docker.{{ $.containerId }}.{{ .Name }}>
  @type record_transformer
  enable_ruby true
  <record>
    host "#{Socket.gethostname}"
    {{range $key, $value := .Tags}}
    {{ $key }} {{ $value }}
    {{end}}

    {{if eq $.fluentdOutput "elasticsearch"}}
    @target {{if .Target}}{{.Target}}-${time.strftime('%Y.%m.%d')}{{else}}{{ .Name }}-${time.strftime('%Y.%m.%d')}{{end}}
    {{else}}
    @target {{if .Target}}{{.Target}}{{else}}{{ .Name }}{{end}}
    {{end}}

    {{range $key, $value := $.container}}
    {{ $key }} {{ $value }}
    {{end}}
  </record>
</filter>
{{end}}
