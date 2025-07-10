package tools

import (
	"bytes"
	"fmt"
	"text/template"
)

// ConfigTemplate represents a configuration template
type ConfigTemplate struct {
	Name     string
	Template string
}

// PrometheusConfigTemplate is the template for Prometheus configuration
var PrometheusConfigTemplate = ConfigTemplate{
	Name: "prometheus",
	Template: `global:
  scrape_interval: {{ .ScrapeInterval | default "15s" }}
  evaluation_interval: {{ .EvaluationInterval | default "15s" }}
  external_labels:
    cluster: {{ .ClusterName | default "local" }}
    environment: {{ .Environment | default "development" }}

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          {{- range .AlertManagerTargets }}
          - {{ . }}
          {{- else }}
          - localhost:9093
          {{- end }}

rule_files:
  {{- range .RuleFiles }}
  - {{ . }}
  {{- else }}
  - /etc/prometheus/rules/*.yml
  {{- end }}

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'apm-application'
    static_configs:
      - targets: ['{{ .APMHost | default "localhost" }}:{{ .APMPort | default "8080" }}']
    metrics_path: '/metrics'

  {{- if .ServiceDiscovery }}
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          {{- range .ServiceDiscovery.Namespaces }}
          - {{ . }}
          {{- end }}
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
  {{- end }}

  {{- range .CustomScrapeConfigs }}
  - {{ . | toYaml | indent 4 }}
  {{- end }}
`,
}

// GrafanaConfigTemplate is the template for Grafana configuration
var GrafanaConfigTemplate = ConfigTemplate{
	Name: "grafana",
	Template: `[server]
protocol = {{ .Protocol | default "http" }}
http_port = {{ .Port | default "3000" }}
root_url = {{ .RootURL | default "%(protocol)s://%(domain)s:%(http_port)s/" }}
serve_from_sub_path = {{ .ServeFromSubPath | default "false" }}

[database]
type = {{ .Database.Type | default "sqlite3" }}
{{- if eq (.Database.Type | default "sqlite3") "postgres" }}
host = {{ .Database.Host }}
name = {{ .Database.Name }}
user = {{ .Database.User }}
password = {{ .Database.Password }}
ssl_mode = {{ .Database.SSLMode | default "disable" }}
{{- else if eq (.Database.Type | default "sqlite3") "mysql" }}
host = {{ .Database.Host }}
name = {{ .Database.Name }}
user = {{ .Database.User }}
password = {{ .Database.Password }}
{{- end }}

[security]
admin_user = {{ .AdminUser | default "admin" }}
admin_password = {{ .AdminPassword | default "admin" }}
disable_initial_admin_creation = {{ .DisableInitialAdmin | default "false" }}
allow_embedding = {{ .AllowEmbedding | default "false" }}

[auth]
disable_login_form = {{ .DisableLoginForm | default "false" }}
disable_signout_menu = {{ .DisableSignoutMenu | default "false" }}

[auth.anonymous]
enabled = {{ .AnonymousAuth | default "false" }}
org_role = {{ .AnonymousOrgRole | default "Viewer" }}

[users]
allow_sign_up = {{ .AllowSignUp | default "false" }}
allow_org_create = {{ .AllowOrgCreate | default "false" }}
auto_assign_org = {{ .AutoAssignOrg | default "true" }}
auto_assign_org_role = {{ .AutoAssignOrgRole | default "Viewer" }}

[alerting]
enabled = {{ .Alerting.Enabled | default "true" }}
execute_alerts = {{ .Alerting.ExecuteAlerts | default "true" }}

[unified_alerting]
enabled = {{ .UnifiedAlerting | default "true" }}

[log]
mode = {{ .LogMode | default "console" }}
level = {{ .LogLevel | default "info" }}

[metrics]
enabled = {{ .MetricsEnabled | default "true" }}
basic_auth_username = {{ .MetricsUsername | default "" }}
basic_auth_password = {{ .MetricsPassword | default "" }}
`,
}

// JaegerConfigTemplate is the template for Jaeger configuration
var JaegerConfigTemplate = ConfigTemplate{
	Name: "jaeger",
	Template: `span-storage-type: {{ .StorageType | default "memory" }}

{{- if eq (.StorageType | default "memory") "elasticsearch" }}
es:
  server-urls: {{ .Elasticsearch.URLs | default "http://localhost:9200" }}
  index-prefix: {{ .Elasticsearch.IndexPrefix | default "jaeger" }}
  {{- if .Elasticsearch.Username }}
  username: {{ .Elasticsearch.Username }}
  password: {{ .Elasticsearch.Password }}
  {{- end }}
{{- else if eq .StorageType "cassandra" }}
cassandra:
  servers: {{ .Cassandra.Servers | default "localhost" }}
  keyspace: {{ .Cassandra.Keyspace | default "jaeger_v1_local" }}
  {{- if .Cassandra.Username }}
  username: {{ .Cassandra.Username }}
  password: {{ .Cassandra.Password }}
  {{- end }}
{{- end }}

collector:
  zipkin:
    http-port: {{ .Collector.ZipkinHTTPPort | default "9411" }}
  otlp:
    enabled: {{ .Collector.OTLPEnabled | default "true" }}
    grpc-port: {{ .Collector.OTLPGRPCPort | default "4317" }}
    http-port: {{ .Collector.OTLPHTTPPort | default "4318" }}

query:
  base-path: {{ .Query.BasePath | default "/" }}
  static-files: {{ .Query.StaticFiles | default "/usr/share/jaeger/ui" }}

processor:
  jaeger-compact:
    server-host-port: {{ .Processor.CompactPort | default "6831" }}
  jaeger-binary:
    server-host-port: {{ .Processor.BinaryPort | default "6832" }}

sampling:
  strategies-file: {{ .Sampling.StrategiesFile | default "/etc/jaeger/sampling_strategies.json" }}
`,
}

// LokiConfigTemplate is the template for Loki configuration
var LokiConfigTemplate = ConfigTemplate{
	Name: "loki",
	Template: `auth_enabled: {{ .AuthEnabled | default "false" }}

server:
  http_listen_port: {{ .HTTPPort | default "3100" }}
  grpc_listen_port: {{ .GRPCPort | default "9096" }}
  log_level: {{ .LogLevel | default "info" }}

ingester:
  lifecycler:
    address: 127.0.0.1
    ring:
      kvstore:
        store: {{ .KVStore | default "inmemory" }}
      replication_factor: {{ .ReplicationFactor | default "1" }}
    final_sleep: 0s
  chunk_idle_period: {{ .ChunkIdlePeriod | default "5m" }}
  chunk_retain_period: {{ .ChunkRetainPeriod | default "30s" }}
  max_transfer_retries: 0
  wal:
    enabled: {{ .WALEnabled | default "true" }}
    dir: {{ .WALDir | default "/loki/wal" }}

schema_config:
  configs:
    - from: {{ .SchemaStartDate | default "2020-05-15" }}
      store: {{ .Store | default "boltdb-shipper" }}
      object_store: {{ .ObjectStore | default "filesystem" }}
      schema: v11
      index:
        prefix: loki_index_
        period: {{ .IndexPeriod | default "24h" }}

storage_config:
  {{- if eq (.Store | default "boltdb-shipper") "boltdb-shipper" }}
  boltdb_shipper:
    active_index_directory: {{ .DataDir | default "/loki" }}/index
    cache_location: {{ .DataDir | default "/loki" }}/index_cache
    shared_store: {{ .ObjectStore | default "filesystem" }}
  {{- end }}
  
  {{- if eq (.ObjectStore | default "filesystem") "filesystem" }}
  filesystem:
    directory: {{ .DataDir | default "/loki" }}/chunks
  {{- else if eq .ObjectStore "s3" }}
  aws:
    s3: {{ .S3.URL }}
    bucketnames: {{ .S3.BucketName }}
    region: {{ .S3.Region }}
    access_key_id: {{ .S3.AccessKeyID }}
    secret_access_key: {{ .S3.SecretAccessKey }}
  {{- end }}

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: {{ .MaxSampleAge | default "168h" }}
  max_entries_limit_per_query: {{ .MaxEntriesLimit | default "5000" }}
  max_query_series: {{ .MaxQuerySeries | default "5000" }}
  max_query_parallelism: {{ .MaxQueryParallelism | default "32" }}

chunk_store_config:
  max_look_back_period: {{ .MaxLookBackPeriod | default "0s" }}

table_manager:
  retention_deletes_enabled: {{ .RetentionDeletesEnabled | default "false" }}
  retention_period: {{ .RetentionPeriod | default "0s" }}
`,
}

// AlertManagerConfigTemplate is the template for AlertManager configuration
var AlertManagerConfigTemplate = ConfigTemplate{
	Name: "alertmanager",
	Template: `global:
  resolve_timeout: {{ .ResolveTimeout | default "5m" }}
  {{- if .SMTPConfig }}
  smtp_smarthost: '{{ .SMTPConfig.Host }}:{{ .SMTPConfig.Port }}'
  smtp_from: '{{ .SMTPConfig.From }}'
  smtp_auth_username: '{{ .SMTPConfig.Username }}'
  smtp_auth_password: '{{ .SMTPConfig.Password }}'
  smtp_require_tls: {{ .SMTPConfig.RequireTLS | default "true" }}
  {{- end }}
  {{- if .SlackAPIURL }}
  slack_api_url: '{{ .SlackAPIURL }}'
  {{- end }}

route:
  receiver: '{{ .DefaultReceiver | default "default" }}'
  group_by: [{{ .GroupBy | default "alertname, cluster, service" }}]
  group_wait: {{ .GroupWait | default "10s" }}
  group_interval: {{ .GroupInterval | default "10s" }}
  repeat_interval: {{ .RepeatInterval | default "1h" }}
  
  routes:
  {{- range .Routes }}
  - match:
      severity: {{ .Severity }}
    receiver: {{ .Receiver }}
    {{- if .Continue }}
    continue: true
    {{- end }}
  {{- end }}

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'cluster', 'service']

receivers:
  - name: 'default'
    {{- if .DefaultWebhookURL }}
    webhook_configs:
      - url: '{{ .DefaultWebhookURL }}'
    {{- end }}
    
  {{- range .Receivers }}
  - name: '{{ .Name }}'
    {{- if .EmailConfigs }}
    email_configs:
    {{- range .EmailConfigs }}
      - to: '{{ .To }}'
        {{- if .Headers }}
        headers:
          {{- range $key, $value := .Headers }}
          {{ $key }}: '{{ $value }}'
          {{- end }}
        {{- end }}
    {{- end }}
    {{- end }}
    
    {{- if .SlackConfigs }}
    slack_configs:
    {{- range .SlackConfigs }}
      - channel: '{{ .Channel }}'
        {{- if .Title }}
        title: '{{ .Title }}'
        {{- end }}
        {{- if .Text }}
        text: '{{ .Text }}'
        {{- end }}
        {{- if .Username }}
        username: '{{ .Username }}'
        {{- end }}
    {{- end }}
    {{- end }}
    
    {{- if .WebhookConfigs }}
    webhook_configs:
    {{- range .WebhookConfigs }}
      - url: '{{ .URL }}'
        {{- if .SendResolved }}
        send_resolved: {{ .SendResolved }}
        {{- end }}
    {{- end }}
    {{- end }}
  {{- end }}
`,
}

// ConfigTemplateRenderer renders configuration templates
type ConfigTemplateRenderer struct {
	templates map[string]*template.Template
}

// NewConfigTemplateRenderer creates a new configuration template renderer
func NewConfigTemplateRenderer() (*ConfigTemplateRenderer, error) {
	renderer := &ConfigTemplateRenderer{
		templates: make(map[string]*template.Template),
	}

	// Register all templates
	templates := []ConfigTemplate{
		PrometheusConfigTemplate,
		GrafanaConfigTemplate,
		JaegerConfigTemplate,
		LokiConfigTemplate,
		AlertManagerConfigTemplate,
	}

	for _, configTemplate := range templates {
		tmpl, err := template.New(configTemplate.Name).Funcs(templateFuncs()).Parse(configTemplate.Template)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", configTemplate.Name, err)
		}
		renderer.templates[configTemplate.Name] = tmpl
	}

	return renderer, nil
}

// Render renders a configuration template with the provided data
func (ctr *ConfigTemplateRenderer) Render(toolType ToolType, data interface{}) (string, error) {
	tmpl, exists := ctr.templates[string(toolType)]
	if !exists {
		return "", fmt.Errorf("no template found for tool type: %s", toolType)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"default": func(defaultVal, val interface{}) interface{} {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
		"toYaml": func(v interface{}) string {
			// Simple YAML conversion for demo purposes
			return fmt.Sprintf("%v", v)
		},
		"indent": func(spaces int, text string) string {
			padding := ""
			for i := 0; i < spaces; i++ {
				padding += " "
			}
			return padding + text
		},
	}
}