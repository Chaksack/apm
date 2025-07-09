# APM Monitoring Setup Guide

Complete guide for setting up comprehensive monitoring for your APM system, including deployment order, configuration verification, and health check procedures.

## Table of Contents
1. [Monitoring Architecture](#monitoring-architecture)
2. [Deployment Order](#deployment-order)
3. [Prometheus Setup](#prometheus-setup)
4. [Grafana Configuration](#grafana-configuration)
5. [Jaeger Tracing](#jaeger-tracing)
6. [Alerting Setup](#alerting-setup)
7. [Health Checks](#health-checks)
8. [Configuration Verification](#configuration-verification)

---

## Monitoring Architecture

### 1. Monitoring Stack Overview

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│   APM Application   │    │      Grafana        │    │    AlertManager     │
│                     │    │   (Dashboards)      │    │   (Notifications)   │
│  ┌─────────────────┐│    │                     │    │                     │
│  │   Metrics       ││────┤  ┌─────────────────┐│    │  ┌─────────────────┐│
│  │   Endpoint      ││    │  │   Prometheus    ││────┤  │     Alerts      ││
│  └─────────────────┘│    │  │   (Storage)     ││    │  │   & Routing     ││
│                     │    │  └─────────────────┘│    │  └─────────────────┘│
│  ┌─────────────────┐│    └─────────────────────┘    └─────────────────────┘
│  │   Tracing       ││
│  │   (OpenTelemetry│││    ┌─────────────────────┐
│  │   /Jaeger)      ││────┤      Jaeger         │
│  └─────────────────┘│    │   (Distributed      │
│                     │    │    Tracing)         │
│  ┌─────────────────┐│    └─────────────────────┘
│  │   Logging       ││
│  │   (Structured)  ││    ┌─────────────────────┐
│  └─────────────────┘│────┤   Log Aggregation   │
└─────────────────────┘    │   (ELK/Loki)        │
                           └─────────────────────┘
```

### 2. Component Dependencies

```yaml
# deployment-order.yaml
# Deployment order for monitoring components
order:
  1. Storage Layer:
     - PostgreSQL (for Grafana)
     - Elasticsearch (for Jaeger)
  2. Core Monitoring:
     - Prometheus
     - Jaeger
  3. Visualization:
     - Grafana
  4. Alerting:
     - AlertManager
  5. Application:
     - APM API
     - APM Frontend
```

---

## Deployment Order

### Phase 1: Storage Layer (5-10 minutes)

#### PostgreSQL for Grafana
```yaml
# 01-postgresql-grafana.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: grafana-postgresql
  namespace: monitoring
spec:
  serviceName: grafana-postgresql
  replicas: 1
  selector:
    matchLabels:
      app: grafana-postgresql
  template:
    metadata:
      labels:
        app: grafana-postgresql
    spec:
      containers:
      - name: postgresql
        image: postgres:14
        env:
        - name: POSTGRES_DB
          value: grafana
        - name: POSTGRES_USER
          value: grafana
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-db-secret
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgresql-data
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
  volumeClaimTemplates:
  - metadata:
      name: postgresql-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi
---
apiVersion: v1
kind: Service
metadata:
  name: grafana-postgresql
  namespace: monitoring
spec:
  selector:
    app: grafana-postgresql
  ports:
  - port: 5432
    targetPort: 5432
```

#### Elasticsearch for Jaeger
```yaml
# 02-elasticsearch-jaeger.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elasticsearch
  namespace: monitoring
spec:
  serviceName: elasticsearch
  replicas: 3
  selector:
    matchLabels:
      app: elasticsearch
  template:
    metadata:
      labels:
        app: elasticsearch
    spec:
      containers:
      - name: elasticsearch
        image: docker.elastic.co/elasticsearch/elasticsearch:7.17.0
        env:
        - name: cluster.name
          value: "jaeger"
        - name: discovery.type
          value: "zen"
        - name: discovery.zen.minimum_master_nodes
          value: "2"
        - name: discovery.zen.ping.unicast.hosts
          value: "elasticsearch-0.elasticsearch,elasticsearch-1.elasticsearch,elasticsearch-2.elasticsearch"
        - name: node.name
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: ES_JAVA_OPTS
          value: "-Xms1g -Xmx1g"
        ports:
        - containerPort: 9200
        - containerPort: 9300
        volumeMounts:
        - name: elasticsearch-data
          mountPath: /usr/share/elasticsearch/data
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
  volumeClaimTemplates:
  - metadata:
      name: elasticsearch-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 50Gi
```

### Phase 2: Core Monitoring (10-15 minutes)

#### Prometheus Configuration
```yaml
# 03-prometheus-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
    
    rule_files:
    - "/etc/prometheus/rules/*.yml"
    
    alerting:
      alertmanagers:
      - static_configs:
        - targets:
          - alertmanager:9093
    
    scrape_configs:
    - job_name: 'prometheus'
      static_configs:
      - targets: ['localhost:9090']
    
    - job_name: 'apm-api'
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
          - apm-system
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: apm-api
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics
    
    - job_name: 'apm-frontend'
      kubernetes_sd_configs:
      - role: endpoints
        namespaces:
          names:
          - apm-system
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_name]
        action: keep
        regex: apm-frontend
      - source_labels: [__meta_kubernetes_endpoint_port_name]
        action: keep
        regex: metrics
    
    - job_name: 'kubernetes-pods'
      kubernetes_sd_configs:
      - role: pod
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
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: kubernetes_pod_name
    
    - job_name: 'kubernetes-service-endpoints'
      kubernetes_sd_configs:
      - role: endpoints
      relabel_configs:
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scheme]
        action: replace
        target_label: __scheme__
        regex: (https?)
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
        action: replace
        target_label: __address__
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
      - action: labelmap
        regex: __meta_kubernetes_service_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_service_name]
        action: replace
        target_label: kubernetes_name
```

#### Prometheus Deployment
```yaml
# 04-prometheus-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      serviceAccountName: prometheus
      containers:
      - name: prometheus
        image: prom/prometheus:v2.40.0
        args:
        - '--config.file=/etc/prometheus/prometheus.yml'
        - '--storage.tsdb.path=/prometheus/'
        - '--web.console.libraries=/etc/prometheus/console_libraries'
        - '--web.console.templates=/etc/prometheus/consoles'
        - '--storage.tsdb.retention.time=200h'
        - '--web.enable-lifecycle'
        - '--web.enable-admin-api'
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: prometheus-config
          mountPath: /etc/prometheus
        - name: prometheus-storage
          mountPath: /prometheus
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /-/healthy
            port: 9090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /-/ready
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: prometheus-config
        configMap:
          name: prometheus-config
      - name: prometheus-storage
        persistentVolumeClaim:
          claimName: prometheus-pvc
```

#### Jaeger Deployment
```yaml
# 05-jaeger-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger-collector
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: jaeger-collector
  template:
    metadata:
      labels:
        app: jaeger-collector
    spec:
      containers:
      - name: jaeger-collector
        image: jaegertracing/jaeger-collector:1.40.0
        env:
        - name: SPAN_STORAGE_TYPE
          value: elasticsearch
        - name: ES_SERVER_URLS
          value: http://elasticsearch:9200
        - name: ES_NUM_SHARDS
          value: "3"
        - name: ES_NUM_REPLICAS
          value: "1"
        ports:
        - containerPort: 14267
        - containerPort: 14268
        - containerPort: 14269
        - containerPort: 9411
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger-query
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger-query
  template:
    metadata:
      labels:
        app: jaeger-query
    spec:
      containers:
      - name: jaeger-query
        image: jaegertracing/jaeger-query:1.40.0
        env:
        - name: SPAN_STORAGE_TYPE
          value: elasticsearch
        - name: ES_SERVER_URLS
          value: http://elasticsearch:9200
        ports:
        - containerPort: 16686
        - containerPort: 16687
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Phase 3: Visualization (5-10 minutes)

#### Grafana Configuration
```yaml
# 06-grafana-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-config
  namespace: monitoring
data:
  grafana.ini: |
    [analytics]
    check_for_updates = true
    
    [grafana_net]
    url = https://grafana.net
    
    [log]
    mode = console
    
    [paths]
    data = /var/lib/grafana/data
    logs = /var/log/grafana
    plugins = /var/lib/grafana/plugins
    provisioning = /etc/grafana/provisioning
    
    [server]
    root_url = %(protocol)s://%(domain)s:%(http_port)s/grafana
    serve_from_sub_path = true
    
    [database]
    type = postgres
    host = grafana-postgresql:5432
    name = grafana
    user = grafana
    password = $__file{/etc/secrets/db_password}
    
    [session]
    provider = postgres
    provider_config = user=grafana password=$__file{/etc/secrets/db_password} host=grafana-postgresql port=5432 dbname=grafana sslmode=disable
    
    [security]
    admin_user = admin
    admin_password = $__file{/etc/secrets/admin_password}
    
    [users]
    allow_sign_up = false
    
  datasources.yaml: |
    apiVersion: 1
    datasources:
    - name: Prometheus
      type: prometheus
      access: proxy
      url: http://prometheus:9090
      isDefault: true
    - name: Jaeger
      type: jaeger
      access: proxy
      url: http://jaeger-query:16686
    - name: Loki
      type: loki
      access: proxy
      url: http://loki:3100
  
  dashboards.yaml: |
    apiVersion: 1
    providers:
    - name: 'default'
      orgId: 1
      folder: ''
      type: file
      disableDeletion: false
      editable: true
      options:
        path: /var/lib/grafana/dashboards
```

#### Grafana Deployment
```yaml
# 07-grafana-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:9.3.0
        ports:
        - containerPort: 3000
        env:
        - name: GF_SECURITY_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: grafana-secret
              key: admin_password
        volumeMounts:
        - name: grafana-config
          mountPath: /etc/grafana
        - name: grafana-secrets
          mountPath: /etc/secrets
        - name: grafana-storage
          mountPath: /var/lib/grafana
        - name: grafana-dashboards
          mountPath: /var/lib/grafana/dashboards
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1"
        livenessProbe:
          httpGet:
            path: /api/health
            port: 3000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/health
            port: 3000
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: grafana-config
        configMap:
          name: grafana-config
      - name: grafana-secrets
        secret:
          secretName: grafana-secret
      - name: grafana-storage
        persistentVolumeClaim:
          claimName: grafana-pvc
      - name: grafana-dashboards
        configMap:
          name: grafana-dashboards
```

### Phase 4: Alerting (5-10 minutes)

#### AlertManager Configuration
```yaml
# 08-alertmanager-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alertmanager-config
  namespace: monitoring
data:
  alertmanager.yml: |
    global:
      smtp_smarthost: 'localhost:587'
      smtp_from: 'apm-alerts@yourdomain.com'
      smtp_auth_username: 'apm-alerts@yourdomain.com'
      smtp_auth_password: 'your-email-password'
    
    route:
      group_by: ['alertname']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 1h
      receiver: 'web.hook'
      routes:
      - match:
          severity: critical
        receiver: 'critical-alerts'
      - match:
          severity: warning
        receiver: 'warning-alerts'
    
    receivers:
    - name: 'web.hook'
      webhook_configs:
      - url: 'http://localhost:5001/'
    
    - name: 'critical-alerts'
      email_configs:
      - to: 'ops-team@yourdomain.com'
        subject: 'CRITICAL: APM Alert - {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}
      slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#apm-alerts'
        title: 'CRITICAL: APM Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
    
    - name: 'warning-alerts'
      email_configs:
      - to: 'dev-team@yourdomain.com'
        subject: 'WARNING: APM Alert - {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}
    
    inhibit_rules:
    - source_match:
        severity: 'critical'
      target_match:
        severity: 'warning'
      equal: ['alertname', 'dev', 'instance']
```

#### AlertManager Deployment
```yaml
# 09-alertmanager-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alertmanager
  namespace: monitoring
spec:
  replicas: 1
  selector:
    matchLabels:
      app: alertmanager
  template:
    metadata:
      labels:
        app: alertmanager
    spec:
      containers:
      - name: alertmanager
        image: prom/alertmanager:v0.25.0
        args:
        - '--config.file=/etc/alertmanager/alertmanager.yml'
        - '--storage.path=/alertmanager'
        - '--web.external-url=http://localhost:9093'
        - '--web.route-prefix=/'
        ports:
        - containerPort: 9093
        volumeMounts:
        - name: alertmanager-config
          mountPath: /etc/alertmanager
        - name: alertmanager-storage
          mountPath: /alertmanager
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: alertmanager-config
        configMap:
          name: alertmanager-config
      - name: alertmanager-storage
        persistentVolumeClaim:
          claimName: alertmanager-pvc
```

### Phase 5: Application Deployment (10-15 minutes)

#### APM API with Monitoring
```yaml
# 10-apm-api-monitored.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apm-api
  namespace: apm-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: apm-api
  template:
    metadata:
      labels:
        app: apm-api
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: apm-api
        image: your-registry.com/apm-api:v1.0.0
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081
          name: metrics
        env:
        - name: JAEGER_AGENT_HOST
          value: "jaeger-agent"
        - name: JAEGER_AGENT_PORT
          value: "6831"
        - name: OTEL_EXPORTER_JAEGER_AGENT_HOST
          value: "jaeger-agent"
        - name: OTEL_EXPORTER_JAEGER_AGENT_PORT
          value: "6831"
        - name: OTEL_SERVICE_NAME
          value: "apm-api"
        - name: OTEL_RESOURCE_ATTRIBUTES
          value: "service.name=apm-api,service.version=1.0.0"
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

---

## Prometheus Setup

### 1. Prometheus Rules

#### APM-Specific Rules
```yaml
# prometheus-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: apm-rules
  namespace: monitoring
spec:
  groups:
  - name: apm.rules
    rules:
    # SLI: Request Rate
    - record: apm:request_rate
      expr: sum(rate(http_requests_total[5m])) by (service, method, status)
    
    # SLI: Request Duration
    - record: apm:request_duration_p95
      expr: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (service, method, le))
    
    # SLI: Error Rate
    - record: apm:error_rate
      expr: sum(rate(http_requests_total{status=~"5.."}[5m])) by (service) / sum(rate(http_requests_total[5m])) by (service)
    
    # SLO: Availability
    - record: apm:availability
      expr: 1 - apm:error_rate
    
    # Database Connection Pool
    - record: apm:db_connections_active
      expr: sum(db_connections_active) by (service)
    
    # Memory Usage
    - record: apm:memory_usage_percent
      expr: (process_resident_memory_bytes / process_virtual_memory_max_bytes) * 100
    
    # CPU Usage
    - record: apm:cpu_usage_percent
      expr: rate(process_cpu_seconds_total[5m]) * 100
    
    # Disk Usage
    - record: apm:disk_usage_percent
      expr: (1 - node_filesystem_avail_bytes / node_filesystem_size_bytes) * 100
```

#### Alert Rules
```yaml
# alert-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: apm-alerts
  namespace: monitoring
spec:
  groups:
  - name: apm.alerts
    rules:
    # High Error Rate
    - alert: HighErrorRate
      expr: apm:error_rate > 0.05
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "High error rate detected"
        description: "Error rate for {{ $labels.service }} is {{ $value | humanizePercentage }}"
    
    # High Response Time
    - alert: HighResponseTime
      expr: apm:request_duration_p95 > 2
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "High response time detected"
        description: "95th percentile response time for {{ $labels.service }} is {{ $value }}s"
    
    # Low Availability
    - alert: LowAvailability
      expr: apm:availability < 0.99
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Low availability detected"
        description: "Availability for {{ $labels.service }} is {{ $value | humanizePercentage }}"
    
    # High Memory Usage
    - alert: HighMemoryUsage
      expr: apm:memory_usage_percent > 85
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "High memory usage detected"
        description: "Memory usage for {{ $labels.service }} is {{ $value }}%"
    
    # High CPU Usage
    - alert: HighCPUUsage
      expr: apm:cpu_usage_percent > 80
      for: 10m
      labels:
        severity: warning
      annotations:
        summary: "High CPU usage detected"
        description: "CPU usage for {{ $labels.service }} is {{ $value }}%"
    
    # Database Connection Pool Exhaustion
    - alert: DatabaseConnectionPoolExhaustion
      expr: apm:db_connections_active > 80
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Database connection pool near exhaustion"
        description: "Active database connections for {{ $labels.service }} is {{ $value }}"
    
    # Pod Restart Alert
    - alert: PodRestartAlert
      expr: increase(kube_pod_container_status_restarts_total[1h]) > 5
      for: 0m
      labels:
        severity: warning
      annotations:
        summary: "Pod restarting frequently"
        description: "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has restarted {{ $value }} times in the last hour"
```

### 2. Service Discovery

#### Kubernetes Service Discovery
```yaml
# service-discovery.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-sd-config
  namespace: monitoring
data:
  kubernetes-sd.yml: |
    - job_name: 'kubernetes-apiservers'
      kubernetes_sd_configs:
      - role: endpoints
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_service_name, __meta_kubernetes_endpoint_port_name]
        action: keep
        regex: default;kubernetes;https
    
    - job_name: 'kubernetes-nodes'
      kubernetes_sd_configs:
      - role: node
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - target_label: __address__
        replacement: kubernetes.default.svc:443
      - source_labels: [__meta_kubernetes_node_name]
        regex: (.+)
        target_label: __metrics_path__
        replacement: /api/v1/nodes/${1}/proxy/metrics
    
    - job_name: 'kubernetes-cadvisor'
      kubernetes_sd_configs:
      - role: node
      scheme: https
      tls_config:
        ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
      bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
      relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - target_label: __address__
        replacement: kubernetes.default.svc:443
      - source_labels: [__meta_kubernetes_node_name]
        regex: (.+)
        target_label: __metrics_path__
        replacement: /api/v1/nodes/${1}/proxy/metrics/cadvisor
```

---

## Grafana Configuration

### 1. APM Dashboards

#### API Performance Dashboard
```json
{
  "dashboard": {
    "id": null,
    "title": "APM API Performance",
    "tags": ["apm", "api", "performance"],
    "style": "dark",
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(rate(http_requests_total[5m])) by (service)",
            "legendFormat": "{{ service }}",
            "refId": "A"
          }
        ],
        "yAxes": [
          {
            "label": "Requests/sec",
            "min": 0
          }
        ],
        "xAxes": [
          {
            "mode": "time"
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 0
        }
      },
      {
        "id": 2,
        "title": "Response Time (95th percentile)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (service, le))",
            "legendFormat": "{{ service }}",
            "refId": "A"
          }
        ],
        "yAxes": [
          {
            "label": "Seconds",
            "min": 0
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 12,
          "y": 0
        }
      },
      {
        "id": 3,
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(rate(http_requests_total{status=~\"5..\"}[5m])) by (service) / sum(rate(http_requests_total[5m])) by (service)",
            "legendFormat": "{{ service }}",
            "refId": "A"
          }
        ],
        "yAxes": [
          {
            "label": "Error Rate",
            "min": 0,
            "max": 1
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 8
        }
      },
      {
        "id": 4,
        "title": "Active Database Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(db_connections_active) by (service)",
            "legendFormat": "{{ service }}",
            "refId": "A"
          }
        ],
        "yAxes": [
          {
            "label": "Connections",
            "min": 0
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 12,
          "y": 8
        }
      }
    ],
    "time": {
      "from": "now-1h",
      "to": "now"
    },
    "refresh": "10s"
  }
}
```

#### System Resource Dashboard
```json
{
  "dashboard": {
    "id": null,
    "title": "APM System Resources",
    "tags": ["apm", "system", "resources"],
    "panels": [
      {
        "id": 1,
        "title": "CPU Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(process_cpu_seconds_total[5m]) * 100",
            "legendFormat": "{{ service }}",
            "refId": "A"
          }
        ],
        "yAxes": [
          {
            "label": "CPU %",
            "min": 0,
            "max": 100
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 0
        }
      },
      {
        "id": 2,
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "process_resident_memory_bytes / 1024 / 1024",
            "legendFormat": "{{ service }}",
            "refId": "A"
          }
        ],
        "yAxes": [
          {
            "label": "Memory (MB)",
            "min": 0
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 12,
          "y": 0
        }
      },
      {
        "id": 3,
        "title": "Disk I/O",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(node_disk_read_bytes_total[5m])",
            "legendFormat": "Read - {{ device }}",
            "refId": "A"
          },
          {
            "expr": "rate(node_disk_written_bytes_total[5m])",
            "legendFormat": "Write - {{ device }}",
            "refId": "B"
          }
        ],
        "yAxes": [
          {
            "label": "Bytes/sec",
            "min": 0
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 0,
          "y": 8
        }
      },
      {
        "id": 4,
        "title": "Network I/O",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(node_network_receive_bytes_total[5m])",
            "legendFormat": "Receive - {{ device }}",
            "refId": "A"
          },
          {
            "expr": "rate(node_network_transmit_bytes_total[5m])",
            "legendFormat": "Transmit - {{ device }}",
            "refId": "B"
          }
        ],
        "yAxes": [
          {
            "label": "Bytes/sec",
            "min": 0
          }
        ],
        "gridPos": {
          "h": 8,
          "w": 12,
          "x": 12,
          "y": 8
        }
      }
    ]
  }
}
```

### 2. Alert Notifications

#### Grafana Alert Configuration
```yaml
# grafana-alerts.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-alerts
  namespace: monitoring
data:
  notification-channels.json: |
    {
      "channels": [
        {
          "name": "email-alerts",
          "type": "email",
          "settings": {
            "addresses": "ops-team@yourdomain.com",
            "subject": "APM Alert: {{ .CommonLabels.alertname }}",
            "body": "{{ range .Alerts }}{{ .Annotations.summary }}\n{{ .Annotations.description }}{{ end }}"
          }
        },
        {
          "name": "slack-alerts",
          "type": "slack",
          "settings": {
            "url": "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
            "channel": "#apm-alerts",
            "username": "grafana",
            "title": "APM Alert",
            "text": "{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}"
          }
        }
      ]
    }
```

---

## Jaeger Tracing

### 1. Jaeger Agent Configuration

#### Jaeger Agent DaemonSet
```yaml
# jaeger-agent.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: jaeger-agent
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: jaeger-agent
  template:
    metadata:
      labels:
        app: jaeger-agent
    spec:
      containers:
      - name: jaeger-agent
        image: jaegertracing/jaeger-agent:1.40.0
        ports:
        - containerPort: 5775
          protocol: UDP
        - containerPort: 6831
          protocol: UDP
        - containerPort: 6832
          protocol: UDP
        - containerPort: 5778
          protocol: TCP
        - containerPort: 14271
          protocol: TCP
        args:
        - --collector.host-port=jaeger-collector:14267
        - --log-level=info
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
      hostNetwork: true
      dnsPolicy: ClusterFirstWithHostNet
```

### 2. OpenTelemetry Configuration

#### OpenTelemetry Collector
```yaml
# otel-collector.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: monitoring
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
      jaeger:
        protocols:
          grpc:
            endpoint: 0.0.0.0:14250
          thrift_http:
            endpoint: 0.0.0.0:14268
          thrift_compact:
            endpoint: 0.0.0.0:6831
          thrift_binary:
            endpoint: 0.0.0.0:6832
    
    processors:
      batch:
        timeout: 1s
        send_batch_size: 1024
      resource:
        attributes:
        - key: service.name
          value: apm-system
          action: upsert
    
    exporters:
      jaeger:
        endpoint: jaeger-collector:14250
        tls:
          insecure: true
      prometheus:
        endpoint: "0.0.0.0:8889"
    
    service:
      pipelines:
        traces:
          receivers: [otlp, jaeger]
          processors: [batch, resource]
          exporters: [jaeger]
        metrics:
          receivers: [otlp]
          processors: [batch, resource]
          exporters: [prometheus]
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: monitoring
spec:
  replicas: 2
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
    spec:
      containers:
      - name: otel-collector
        image: otel/opentelemetry-collector:0.70.0
        command:
        - "/otelcol"
        - "--config=/etc/otel-collector-config/config.yaml"
        volumeMounts:
        - name: otel-collector-config
          mountPath: /etc/otel-collector-config
        ports:
        - containerPort: 4317
        - containerPort: 4318
        - containerPort: 14250
        - containerPort: 14268
        - containerPort: 6831
        - containerPort: 6832
        - containerPort: 8889
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1"
      volumes:
      - name: otel-collector-config
        configMap:
          name: otel-collector-config
```

---

## Health Checks

### 1. Monitoring Stack Health Checks

#### Health Check Script
```bash
#!/bin/bash
# health-check.sh

set -e

NAMESPACE="monitoring"
TIMEOUT=300  # 5 minutes

echo "Starting APM monitoring stack health check..."

# Function to check if deployment is ready
check_deployment() {
    local name=$1
    local namespace=$2
    
    echo "Checking deployment: $name"
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available --timeout=${TIMEOUT}s deployment/$name -n $namespace
    
    # Check if all pods are running
    local ready_replicas=$(kubectl get deployment $name -n $namespace -o jsonpath='{.status.readyReplicas}')
    local desired_replicas=$(kubectl get deployment $name -n $namespace -o jsonpath='{.spec.replicas}')
    
    if [ "$ready_replicas" != "$desired_replicas" ]; then
        echo "ERROR: $name deployment not fully ready ($ready_replicas/$desired_replicas)"
        return 1
    fi
    
    echo "✓ $name deployment is healthy"
}

# Function to check if service endpoints are available
check_service_endpoints() {
    local service=$1
    local namespace=$2
    local port=$3
    
    echo "Checking service endpoints: $service"
    
    # Get service cluster IP
    local cluster_ip=$(kubectl get service $service -n $namespace -o jsonpath='{.spec.clusterIP}')
    
    if [ -z "$cluster_ip" ]; then
        echo "ERROR: Could not get cluster IP for service $service"
        return 1
    fi
    
    # Check if service is responding
    kubectl run health-check-$(date +%s) --rm -i --restart=Never --image=curlimages/curl -- \
        curl -f -s -m 10 http://$cluster_ip:$port/
    
    if [ $? -eq 0 ]; then
        echo "✓ $service service is responding"
    else
        echo "ERROR: $service service is not responding"
        return 1
    fi
}

# Function to check Prometheus targets
check_prometheus_targets() {
    echo "Checking Prometheus targets..."
    
    # Port forward to Prometheus
    kubectl port-forward -n $NAMESPACE service/prometheus 9090:9090 &
    local port_forward_pid=$!
    
    sleep 5
    
    # Check if all targets are up
    local targets_down=$(curl -s http://localhost:9090/api/v1/targets | jq -r '.data.activeTargets[] | select(.health != "up") | .scrapeUrl')
    
    if [ -n "$targets_down" ]; then
        echo "WARNING: Some Prometheus targets are down:"
        echo "$targets_down"
    else
        echo "✓ All Prometheus targets are up"
    fi
    
    # Clean up port forward
    kill $port_forward_pid
}

# Function to check Grafana datasources
check_grafana_datasources() {
    echo "Checking Grafana datasources..."
    
    # Port forward to Grafana
    kubectl port-forward -n $NAMESPACE service/grafana 3000:3000 &
    local port_forward_pid=$!
    
    sleep 5
    
    # Check datasources (requires admin credentials)
    local admin_password=$(kubectl get secret grafana-secret -n $NAMESPACE -o jsonpath='{.data.admin_password}' | base64 -d)
    
    local datasources_status=$(curl -s -u admin:$admin_password http://localhost:3000/api/datasources | jq -r '.[] | select(.type == "prometheus") | .name')
    
    if [ -n "$datasources_status" ]; then
        echo "✓ Grafana datasources are configured"
    else
        echo "ERROR: Grafana datasources not configured properly"
        kill $port_forward_pid
        return 1
    fi
    
    # Clean up port forward
    kill $port_forward_pid
}

# Main health check sequence
main() {
    echo "=== APM Monitoring Stack Health Check ==="
    
    # Check storage layer
    echo "1. Checking storage layer..."
    check_deployment "grafana-postgresql" $NAMESPACE
    check_deployment "elasticsearch" $NAMESPACE
    
    # Check core monitoring
    echo "2. Checking core monitoring..."
    check_deployment "prometheus" $NAMESPACE
    check_deployment "jaeger-collector" $NAMESPACE
    check_deployment "jaeger-query" $NAMESPACE
    
    # Check visualization
    echo "3. Checking visualization..."
    check_deployment "grafana" $NAMESPACE
    
    # Check alerting
    echo "4. Checking alerting..."
    check_deployment "alertmanager" $NAMESPACE
    
    # Check service endpoints
    echo "5. Checking service endpoints..."
    check_service_endpoints "prometheus" $NAMESPACE 9090
    check_service_endpoints "grafana" $NAMESPACE 3000
    check_service_endpoints "jaeger-query" $NAMESPACE 16686
    check_service_endpoints "alertmanager" $NAMESPACE 9093
    
    # Check Prometheus targets
    echo "6. Checking Prometheus targets..."
    check_prometheus_targets
    
    # Check Grafana datasources
    echo "7. Checking Grafana datasources..."
    check_grafana_datasources
    
    echo "=== Health Check Complete ==="
    echo "✓ APM monitoring stack is healthy"
}

# Run main function
main "$@"
```

### 2. Automated Health Monitoring

#### Health Check CronJob
```yaml
# health-check-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: monitoring-health-check
  namespace: monitoring
spec:
  schedule: "*/15 * * * *"  # Every 15 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: health-checker
            image: your-registry.com/health-checker:latest
            env:
            - name: NAMESPACE
              value: "monitoring"
            - name: SLACK_WEBHOOK
              valueFrom:
                secretKeyRef:
                  name: monitoring-secrets
                  key: slack-webhook
            command:
            - /bin/bash
            - -c
            - |
              # Run health checks
              ./health-check.sh
              
              # Send notification if health check fails
              if [ $? -ne 0 ]; then
                curl -X POST -H 'Content-type: application/json' \
                  --data '{"text":"APM monitoring stack health check failed!"}' \
                  $SLACK_WEBHOOK
              fi
          restartPolicy: OnFailure
```

---

## Configuration Verification

### 1. Verification Commands

#### Complete Verification Script
```bash
#!/bin/bash
# verify-monitoring.sh

set -e

echo "=== APM Monitoring Configuration Verification ==="

# Check namespace
echo "1. Verifying namespace..."
kubectl get namespace monitoring apm-system || {
    echo "ERROR: Required namespaces not found"
    exit 1
}

# Check secrets
echo "2. Verifying secrets..."
kubectl get secret -n monitoring grafana-secret prometheus-secret alertmanager-secret || {
    echo "ERROR: Required secrets not found"
    exit 1
}

# Check ConfigMaps
echo "3. Verifying ConfigMaps..."
kubectl get configmap -n monitoring prometheus-config grafana-config alertmanager-config || {
    echo "ERROR: Required ConfigMaps not found"
    exit 1
}

# Check PVCs
echo "4. Verifying PVCs..."
kubectl get pvc -n monitoring prometheus-pvc grafana-pvc alertmanager-pvc || {
    echo "ERROR: Required PVCs not found"
    exit 1
}

# Check deployments
echo "5. Verifying deployments..."
kubectl get deployment -n monitoring prometheus grafana alertmanager jaeger-collector jaeger-query || {
    echo "ERROR: Required deployments not found"
    exit 1
}

# Check services
echo "6. Verifying services..."
kubectl get service -n monitoring prometheus grafana alertmanager jaeger-collector jaeger-query || {
    echo "ERROR: Required services not found"
    exit 1
}

# Check ServiceMonitors (if using Prometheus Operator)
echo "7. Verifying ServiceMonitors..."
kubectl get servicemonitor -n monitoring || {
    echo "WARNING: ServiceMonitors not found (may not be using Prometheus Operator)"
}

# Check PrometheusRules
echo "8. Verifying PrometheusRules..."
kubectl get prometheusrule -n monitoring || {
    echo "WARNING: PrometheusRules not found"
}

# Verify metrics are being scraped
echo "9. Verifying metrics collection..."
kubectl port-forward -n monitoring service/prometheus 9090:9090 &
PROMETHEUS_PID=$!
sleep 5

# Check if APM metrics are being scraped
METRICS_COUNT=$(curl -s http://localhost:9090/api/v1/label/__name__/values | jq '.data | length')
if [ "$METRICS_COUNT" -gt 100 ]; then
    echo "✓ Metrics are being collected ($METRICS_COUNT metrics found)"
else
    echo "WARNING: Low metric count ($METRICS_COUNT metrics found)"
fi

kill $PROMETHEUS_PID

echo "=== Configuration Verification Complete ==="
```

### 2. Performance Baseline

#### Baseline Metrics Collection
```bash
#!/bin/bash
# collect-baseline.sh

echo "Collecting baseline metrics..."

# Create baseline directory
mkdir -p baseline-metrics

# Collect Prometheus metrics
kubectl port-forward -n monitoring service/prometheus 9090:9090 &
PROMETHEUS_PID=$!
sleep 5

# APM API metrics
curl -s http://localhost:9090/api/v1/query?query=up{job="apm-api"} > baseline-metrics/apm-api-up.json
curl -s http://localhost:9090/api/v1/query?query=rate(http_requests_total[5m]) > baseline-metrics/apm-api-request-rate.json
curl -s http://localhost:9090/api/v1/query?query=histogram_quantile(0.95,rate(http_request_duration_seconds_bucket[5m])) > baseline-metrics/apm-api-response-time.json

# System metrics
curl -s http://localhost:9090/api/v1/query?query=rate(process_cpu_seconds_total[5m]) > baseline-metrics/cpu-usage.json
curl -s http://localhost:9090/api/v1/query?query=process_resident_memory_bytes > baseline-metrics/memory-usage.json

kill $PROMETHEUS_PID

echo "Baseline metrics collected in baseline-metrics/"
```

---

## Summary

This comprehensive monitoring setup provides:

1. **Complete observability** with metrics, logs, and traces
2. **Automated alerting** for critical issues
3. **Visual dashboards** for monitoring system health
4. **Health checks** to ensure monitoring stack reliability
5. **Configuration verification** to validate setup

### Access Points After Deployment:
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686
- **AlertManager**: http://localhost:9093

### Key Monitoring Metrics:
- Request rate and response times
- Error rates and availability
- Resource utilization (CPU, memory, disk)
- Database connection pool status
- System-level metrics

**Total Setup Time**: 45-60 minutes  
**Complexity**: Advanced  
**Storage Requirements**: ~200GB for production setup  
**Maintenance**: Regular backup and retention policy management required