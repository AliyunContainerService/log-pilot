{{range .configList}}
{{if .Stdout}}
- type: container
{{ else }}
- type: log
{{end}}
  enabled: true
  paths:
      - {{ .HostDir }}/{{ .File }}
  scan_frequency: 10s
  fields_under_root: true
  {{if eq .Format "json"}}
  json.keys_under_root: true
  json.overwrite_keys: true
  json.add_error_key: true
  json.message_key: message
  {{end}}
  fields:
      {{range $key, $value := .CustomFields}}
      {{ $key }}: {{ $value }}
      {{end}}
      {{range $key, $value := .Tags}}
      {{ $key }}: {{ $value }}
      {{end}}
      {{range $key, $value := $.container}}
      {{ $key }}: {{ $value }}
      {{end}}
  {{range $key, $value := .CustomConfigs}}
  {{ $key }}: {{ $value }}
  {{end}}
  tail_files: false
  close_inactive: 2h
  close_eof: false
  close_removed: true
  clean_removed: true
  close_renamed: false
  {{if not .Stdout}}
  clean_inactive: 72h
  ignore_older: 70h
  {{end}}
  max_bytes: 20480
{{end}}
