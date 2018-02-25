{{range .configList}}
<source>
  @type tail
  tag docker.{{ $.containerId }}.{{ .Name }}
  path {{ .HostDir }}/{{ .File }}

  <parse>
  {{if .Stdout}}
  @type json
  {{else}}
  @type {{ .Format }}
  {{end}}
  {{ $time_key := "" }}
  {{if .FormatConfig}}
  {{range $key, $value := .FormatConfig}}
  {{ $key }} {{ $value }}
  {{end}}
  {{end}}
  {{ if .EstimateTime }}
  estimate_current_event true
  {{end}}
  keep_time_key true
  </parse>

  read_from_head true
  pos_file /pilot/pos/{{ $.containerId }}.{{ .Name }}.pos
</source>

<filter docker.{{ $.containerId }}.{{ .Name }}>
  @type record_transformer
  enable_ruby true
  <record>
    host "#{Socket.gethostname}"
    {{range $key, $value := .Tags}}
    {{ $key }} {{ $value }}
    {{end}}

    {{if eq $.output "elasticsearch"}}
    _target {{if .Target}}{{.Target}}-${time.strftime('%Y.%m.%d')}{{else}}{{ .Name }}-${time.strftime('%Y.%m.%d')}{{end}}
    {{else}}
    _target {{if .Target}}{{.Target}}{{else}}{{ .Name }}{{end}}
    {{end}}

    {{range $key, $value := $.container}}
    {{ $key }} {{ $value }}
    {{end}}
  </record>
</filter>
{{end}}
