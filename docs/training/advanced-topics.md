# Advanced Observability Topics

## Learning Objectives
By the end of this advanced training, you will be able to:
- Implement service mesh observability
- Design distributed tracing strategies
- Optimize observability performance
- Address security considerations
- Handle complex monitoring scenarios

## Service Mesh Observability

### Introduction to Service Mesh
A service mesh is a dedicated infrastructure layer for handling service-to-service communication in microservices architectures.

**Key Components**:
- **Data Plane**: Handles service-to-service communication
- **Control Plane**: Manages and configures the data plane
- **Sidecar Proxies**: Intercept and manage traffic

### Istio Observability Setup

#### Basic Istio Configuration
```yaml
# istio-observability.yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: control-plane
spec:
  values:
    telemetry:
      v2:
        enabled: true
    pilot:
      env:
        EXTERNAL_ISTIOD: false
    global:
      meshID: mesh1
      multiCluster:
        clusterName: cluster1
      network: network1
  components:
    pilot:
      k8s:
        env:
          - name: PILOT_ENABLE_WORKLOAD_ENTRY_AUTOREGISTRATION
            value: true
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          service:
            type: LoadBalancer
```

#### Telemetry Configuration
```yaml
# telemetry-v2.yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: default
  namespace: istio-system
spec:
  metrics:
  - providers:
    - name: prometheus
  - overrides:
    - match:
        metric: ALL_METRICS
      tagOverrides:
        request_protocol:
          value: "unknown"
    - match:
        metric: REQUEST_COUNT
      disabled: false
    - match:
        metric: REQUEST_DURATION
      disabled: false
    - match:
        metric: REQUEST_SIZE
      disabled: false
    - match:
        metric: RESPONSE_SIZE
      disabled: false
  tracing:
  - providers:
    - name: jaeger
  accessLogging:
  - providers:
    - name: otel
```

### Service Mesh Metrics

#### Istio Standard Metrics
```promql
# Request Rate
sum(rate(istio_requests_total[5m])) by (source_service_name, destination_service_name)

# Error Rate
sum(rate(istio_requests_total{response_code!~"2.."}[5m])) by (destination_service_name) / 
sum(rate(istio_requests_total[5m])) by (destination_service_name)

# Request Duration
histogram_quantile(0.99, 
  sum(rate(istio_request_duration_milliseconds_bucket[5m])) by (le, destination_service_name)
)

# Service Mesh Health
up{job="istio-proxy"}
istio_agent_pilot_xds_pushes_total
istio_agent_pilot_xds_expired_nonce_total
```

#### Custom Metrics Configuration
```yaml
# custom-metrics.yaml
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: custom-metrics
  namespace: production
spec:
  metrics:
  - providers:
    - name: prometheus
  - overrides:
    - match:
        metric: REQUEST_COUNT
      dimensionOverrides:
        source_version:
          value: "source.labels['version'] | 'unknown'"
        destination_version:
          value: "destination.labels['version'] | 'unknown'"
    - match:
        metric: REQUEST_DURATION
      dimensionOverrides:
        user_type:
          value: "request.headers['user-type'] | 'anonymous'"
```

### Advanced Service Mesh Observability

#### Circuit Breaker Monitoring
```yaml
# circuit-breaker.yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: user-service
spec:
  host: user-service
  trafficPolicy:
    outlierDetection:
      consecutiveErrors: 5
      interval: 10s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
      minHealthPercent: 20
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 10
```

#### Fault Injection for Testing
```yaml
# fault-injection.yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: user-service-fault
spec:
  hosts:
  - user-service
  http:
  - match:
    - headers:
        test-fault:
          exact: "true"
    fault:
      delay:
        percentage:
          value: 50
        fixedDelay: 5s
      abort:
        percentage:
          value: 10
        httpStatus: 503
    route:
    - destination:
        host: user-service
```

## Distributed Tracing Patterns

### Sampling Strategies

#### Probabilistic Sampling
```python
# probabilistic-sampling.py
from opentelemetry.sdk.trace.sampling import ProbabilitySampler, ParentBased
from opentelemetry.sdk.trace import TracerProvider

# Sample 10% of traces
sampler = ParentBased(root=ProbabilitySampler(0.1))
tracer_provider = TracerProvider(sampler=sampler)
```

#### Adaptive Sampling
```python
# adaptive-sampling.py
import time
from opentelemetry.sdk.trace.sampling import Sampler, SamplingResult

class AdaptiveSampler(Sampler):
    def __init__(self, max_traces_per_second=100):
        self.max_traces_per_second = max_traces_per_second
        self.traces_this_second = 0
        self.current_second = int(time.time())
        
    def should_sample(self, parent_context, trace_id, name, kind, attributes, links):
        current_time = int(time.time())
        
        if current_time != self.current_second:
            self.current_second = current_time
            self.traces_this_second = 0
        
        if self.traces_this_second < self.max_traces_per_second:
            self.traces_this_second += 1
            return SamplingResult.RECORD_AND_SAMPLE
        
        return SamplingResult.DROP
```

#### Business Logic Sampling
```python
# business-logic-sampling.py
class BusinessLogicSampler(Sampler):
    def should_sample(self, parent_context, trace_id, name, kind, attributes, links):
        # Always sample errors
        if attributes and attributes.get("error") == "true":
            return SamplingResult.RECORD_AND_SAMPLE
        
        # Always sample critical operations
        if name in ["checkout", "payment", "login"]:
            return SamplingResult.RECORD_AND_SAMPLE
        
        # Sample 1% of regular operations
        if trace_id % 100 == 0:
            return SamplingResult.RECORD_AND_SAMPLE
        
        return SamplingResult.DROP
```

### Trace Context Propagation

#### HTTP Headers Propagation
```python
# context-propagation.py
from opentelemetry import trace
from opentelemetry.propagate import inject, extract
import requests

def make_request_with_context(url, data=None):
    # Get current span context
    current_span = trace.get_current_span()
    
    # Prepare headers for propagation
    headers = {}
    inject(headers)
    
    # Add custom headers
    headers.update({
        'Content-Type': 'application/json',
        'User-Agent': 'MyService/1.0'
    })
    
    # Make request with propagated context
    response = requests.post(url, json=data, headers=headers)
    
    # Add response information to span
    current_span.set_attribute("http.status_code", response.status_code)
    current_span.set_attribute("http.response_size", len(response.content))
    
    return response

def handle_incoming_request(request):
    # Extract context from incoming request
    context = extract(request.headers)
    
    # Start span with extracted context
    with trace.set_span_in_context(context):
        with tracer.start_as_current_span("handle_request") as span:
            # Process request
            return process_request(request)
```

#### Message Queue Propagation
```python
# message-queue-propagation.py
import json
from opentelemetry import trace
from opentelemetry.propagate import inject, extract

class MessageProducer:
    def __init__(self, queue_client):
        self.queue_client = queue_client
        self.tracer = trace.get_tracer(__name__)
    
    def send_message(self, queue_name, message):
        with self.tracer.start_as_current_span("send_message") as span:
            span.set_attribute("queue.name", queue_name)
            span.set_attribute("message.id", message.get("id", "unknown"))
            
            # Inject trace context into message
            headers = {}
            inject(headers)
            
            # Add headers to message
            message["_trace_headers"] = headers
            
            # Send message
            self.queue_client.send(queue_name, json.dumps(message))

class MessageConsumer:
    def __init__(self, queue_client):
        self.queue_client = queue_client
        self.tracer = trace.get_tracer(__name__)
    
    def process_message(self, message_data):
        message = json.loads(message_data)
        
        # Extract trace context from message
        headers = message.get("_trace_headers", {})
        context = extract(headers)
        
        # Process message with extracted context
        with trace.set_span_in_context(context):
            with self.tracer.start_as_current_span("process_message") as span:
                span.set_attribute("message.id", message.get("id", "unknown"))
                
                # Process message
                return self.handle_message(message)
```

### Advanced Tracing Techniques

#### Span Linking
```python
# span-linking.py
from opentelemetry import trace
from opentelemetry.trace import Link

def batch_processor(requests):
    """Process multiple requests in a batch, linking to original traces"""
    links = []
    
    # Create links to all request traces
    for request in requests:
        if hasattr(request, 'trace_context'):
            link = Link(context=request.trace_context)
            links.append(link)
    
    # Start batch processing span with links
    with tracer.start_as_current_span("batch_process", links=links) as span:
        span.set_attribute("batch.size", len(requests))
        
        results = []
        for request in requests:
            result = process_single_request(request)
            results.append(result)
        
        return results
```

#### Custom Span Events
```python
# span-events.py
import time
from opentelemetry import trace

def process_with_events(data):
    with tracer.start_as_current_span("process_data") as span:
        # Add event at start
        span.add_event("processing_started", {"data_size": len(data)})
        
        # Validation phase
        validation_start = time.time()
        if not validate_data(data):
            span.add_event("validation_failed", {"error": "Invalid data format"})
            span.set_status(trace.Status(trace.StatusCode.ERROR, "Validation failed"))
            return None
        
        span.add_event("validation_completed", {
            "duration_ms": (time.time() - validation_start) * 1000
        })
        
        # Processing phase
        processing_start = time.time()
        result = expensive_operation(data)
        
        span.add_event("processing_completed", {
            "duration_ms": (time.time() - processing_start) * 1000,
            "result_size": len(result)
        })
        
        return result
```

## Performance Optimization

### Metrics Storage Optimization

#### Prometheus Optimization
```yaml
# prometheus-optimized.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'production'
    region: 'us-west-2'

# Storage optimization
storage:
  tsdb:
    retention.time: 15d
    retention.size: 50GB
    wal-compression: true
    block-compression: true

# Query optimization
query:
  max_concurrent: 20
  timeout: 2m
  lookback_delta: 5m

# Scrape optimization
scrape_configs:
  - job_name: 'high-frequency'
    scrape_interval: 5s
    scrape_timeout: 2s
    static_configs:
      - targets: ['critical-service:8080']
    metric_relabel_configs:
      # Drop high-cardinality metrics
      - source_labels: [__name__]
        regex: 'histogram_bucket.*'
        target_label: __tmp_drop
        replacement: 'true'
      - source_labels: [__tmp_drop]
        regex: 'true'
        action: drop
```

#### Metrics Aggregation
```python
# metrics-aggregation.py
from prometheus_client import Counter, Histogram, Gauge
import time

class MetricsAggregator:
    def __init__(self, window_size=60):
        self.window_size = window_size
        self.request_count = Counter('aggregated_requests_total', 'Total requests', ['service', 'method'])
        self.response_time = Histogram('aggregated_response_time_seconds', 'Response time', ['service'])
        self.active_connections = Gauge('aggregated_active_connections', 'Active connections', ['service'])
        
        # Internal aggregation state
        self.request_buffer = []
        self.last_flush = time.time()
    
    def record_request(self, service, method, response_time):
        self.request_buffer.append({
            'service': service,
            'method': method,
            'response_time': response_time,
            'timestamp': time.time()
        })
        
        # Flush if window is full
        if time.time() - self.last_flush > self.window_size:
            self.flush_metrics()
    
    def flush_metrics(self):
        """Aggregate and flush metrics"""
        current_time = time.time()
        
        # Group by service and method
        grouped = {}
        for record in self.request_buffer:
            key = (record['service'], record['method'])
            if key not in grouped:
                grouped[key] = []
            grouped[key].append(record)
        
        # Update metrics
        for (service, method), records in grouped.items():
            self.request_count.labels(service=service, method=method).inc(len(records))
            
            # Calculate average response time
            avg_response_time = sum(r['response_time'] for r in records) / len(records)
            self.response_time.labels(service=service).observe(avg_response_time)
        
        # Clear buffer
        self.request_buffer = []
        self.last_flush = current_time
```

### Log Processing Optimization

#### Structured Log Parsing
```python
# log-parsing.py
import json
import re
from typing import Dict, Any

class LogParser:
    def __init__(self):
        self.parsers = [
            self.parse_json,
            self.parse_nginx,
            self.parse_apache,
            self.parse_generic
        ]
    
    def parse_json(self, log_line: str) -> Dict[str, Any]:
        """Parse structured JSON logs"""
        try:
            return json.loads(log_line)
        except json.JSONDecodeError:
            return None
    
    def parse_nginx(self, log_line: str) -> Dict[str, Any]:
        """Parse Nginx access logs"""
        pattern = r'(\S+) \S+ \S+ \[(.*?)\] "(.*?)" (\d+) (\d+) "(.*?)" "(.*?)"'
        match = re.match(pattern, log_line)
        
        if match:
            return {
                'ip': match.group(1),
                'timestamp': match.group(2),
                'request': match.group(3),
                'status': int(match.group(4)),
                'size': int(match.group(5)),
                'referer': match.group(6),
                'user_agent': match.group(7)
            }
        return None
    
    def parse_apache(self, log_line: str) -> Dict[str, Any]:
        """Parse Apache access logs"""
        pattern = r'(\S+) \S+ \S+ \[(.*?)\] "(.*?)" (\d+) (\d+|-)'
        match = re.match(pattern, log_line)
        
        if match:
            size = match.group(5)
            return {
                'ip': match.group(1),
                'timestamp': match.group(2),
                'request': match.group(3),
                'status': int(match.group(4)),
                'size': int(size) if size != '-' else 0
            }
        return None
    
    def parse_generic(self, log_line: str) -> Dict[str, Any]:
        """Parse generic log format"""
        return {
            'raw_message': log_line,
            'timestamp': 'unknown',
            'level': 'INFO'
        }
    
    def parse(self, log_line: str) -> Dict[str, Any]:
        """Parse log line using available parsers"""
        for parser in self.parsers:
            result = parser(log_line)
            if result:
                return result
        
        return {'raw_message': log_line, 'parsed': False}
```

### Trace Storage Optimization

#### Jaeger Storage Configuration
```yaml
# jaeger-production.yml
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: jaeger-production
spec:
  strategy: production
  
  storage:
    type: elasticsearch
    elasticsearch:
      serverUrls: ["http://elasticsearch:9200"]
      indexPrefix: jaeger
      
      # Optimize for write performance
      bulkSize: 1000
      bulkWorkers: 10
      bulkFlushInterval: 1s
      
      # Index management
      createIndexTemplate: true
      version: 7
      
      # Retention policy
      indexCleaner:
        enabled: true
        numberOfDays: 7
        schedule: "0 0 * * *"
  
  collector:
    maxReplicas: 10
    resources:
      limits:
        memory: 2Gi
        cpu: 1000m
    
    # Sampling configuration
    config: |
      sampling:
        default_strategy:
          type: probabilistic
          param: 0.1
        per_service_strategies:
          - service: "critical-service"
            type: probabilistic
            param: 1.0
          - service: "batch-service"
            type: probabilistic
            param: 0.01
  
  query:
    replicas: 3
    resources:
      limits:
        memory: 1Gi
        cpu: 500m
```

## Security Considerations

### Metrics Security

#### Access Control
```yaml
# prometheus-rbac.yml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: prometheus
  namespace: monitoring

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: prometheus
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/metrics", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions"]
  resources: ["ingresses"]
  verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: prometheus
subjects:
- kind: ServiceAccount
  name: prometheus
  namespace: monitoring
```

#### Sensitive Data Filtering
```python
# sensitive-data-filter.py
import re
from typing import Dict, Any

class SensitiveDataFilter:
    def __init__(self):
        self.patterns = [
            (r'\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b', '****-****-****-****'),  # Credit card
            (r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b', '[EMAIL]'),  # Email
            (r'\b\d{3}-\d{2}-\d{4}\b', '***-**-****'),  # SSN
            (r'\b(?:\d{1,3}\.){3}\d{1,3}\b', '[IP]'),  # IP address
            (r'password["\']?\s*[:=]\s*["\']?([^"\']+)["\']?', 'password="[REDACTED]"'),  # Password
            (r'token["\']?\s*[:=]\s*["\']?([^"\']+)["\']?', 'token="[REDACTED]"'),  # Token
        ]
    
    def filter_log_message(self, message: str) -> str:
        """Filter sensitive data from log messages"""
        filtered = message
        for pattern, replacement in self.patterns:
            filtered = re.sub(pattern, replacement, filtered, flags=re.IGNORECASE)
        return filtered
    
    def filter_metrics_labels(self, labels: Dict[str, str]) -> Dict[str, str]:
        """Filter sensitive data from metric labels"""
        filtered = {}
        for key, value in labels.items():
            if key.lower() in ['password', 'token', 'secret', 'key']:
                filtered[key] = '[REDACTED]'
            else:
                filtered[key] = self.filter_log_message(value)
        return filtered
    
    def filter_span_attributes(self, attributes: Dict[str, Any]) -> Dict[str, Any]:
        """Filter sensitive data from span attributes"""
        filtered = {}
        for key, value in attributes.items():
            if isinstance(value, str):
                if key.lower() in ['password', 'token', 'secret', 'authorization']:
                    filtered[key] = '[REDACTED]'
                else:
                    filtered[key] = self.filter_log_message(value)
            else:
                filtered[key] = value
        return filtered
```

### Log Security

#### Audit Logging
```python
# audit-logging.py
import json
import hashlib
from datetime import datetime
from typing import Dict, Any

class AuditLogger:
    def __init__(self, service_name: str):
        self.service_name = service_name
        self.sensitive_fields = ['password', 'token', 'secret', 'key', 'authorization']
    
    def log_access(self, user_id: str, resource: str, action: str, outcome: str, **kwargs):
        """Log access attempts"""
        audit_entry = {
            'timestamp': datetime.utcnow().isoformat(),
            'event_type': 'access',
            'service': self.service_name,
            'user_id': user_id,
            'resource': resource,
            'action': action,
            'outcome': outcome,  # success, failure, error
            'session_id': kwargs.get('session_id'),
            'ip_address': kwargs.get('ip_address'),
            'user_agent': kwargs.get('user_agent'),
            'request_id': kwargs.get('request_id')
        }
        
        # Add integrity hash
        audit_entry['integrity_hash'] = self._calculate_hash(audit_entry)
        
        # Log to secure audit log
        self._write_audit_log(audit_entry)
    
    def log_data_access(self, user_id: str, data_type: str, record_ids: list, action: str):
        """Log data access for compliance"""
        audit_entry = {
            'timestamp': datetime.utcnow().isoformat(),
            'event_type': 'data_access',
            'service': self.service_name,
            'user_id': user_id,
            'data_type': data_type,
            'record_count': len(record_ids),
            'record_ids': record_ids[:10],  # Limit to first 10 for space
            'action': action,
            'compliance_category': self._get_compliance_category(data_type)
        }
        
        audit_entry['integrity_hash'] = self._calculate_hash(audit_entry)
        self._write_audit_log(audit_entry)
    
    def _calculate_hash(self, entry: Dict[str, Any]) -> str:
        """Calculate integrity hash for audit entry"""
        # Remove hash field if present
        entry_copy = {k: v for k, v in entry.items() if k != 'integrity_hash'}
        entry_json = json.dumps(entry_copy, sort_keys=True)
        return hashlib.sha256(entry_json.encode()).hexdigest()
    
    def _write_audit_log(self, entry: Dict[str, Any]):
        """Write to secure audit log"""
        # In production, this would write to a secure, tamper-evident log store
        print(f"AUDIT: {json.dumps(entry)}")
    
    def _get_compliance_category(self, data_type: str) -> str:
        """Determine compliance category for data type"""
        pii_types = ['user_profile', 'payment_info', 'contact_info']
        if data_type in pii_types:
            return 'PII'
        return 'GENERAL'
```

### Trace Security

#### Trace Sampling for Security
```python
# security-sampling.py
from opentelemetry.sdk.trace.sampling import Sampler, SamplingResult
from opentelemetry.trace import SpanKind

class SecurityAwareSampler(Sampler):
    def __init__(self):
        self.security_operations = [
            'login', 'logout', 'password_change', 'permission_check',
            'admin_access', 'data_export', 'user_creation', 'user_deletion'
        ]
        self.sensitive_attributes = [
            'user.id', 'user.email', 'payment.method', 'personal.data'
        ]
    
    def should_sample(self, parent_context, trace_id, name, kind, attributes, links):
        # Always sample security-related operations
        if name.lower() in self.security_operations:
            return SamplingResult.RECORD_AND_SAMPLE
        
        # Always sample if contains sensitive attributes
        if attributes:
            for attr_key in attributes.keys():
                if any(sensitive in attr_key.lower() for sensitive in self.sensitive_attributes):
                    return SamplingResult.RECORD_AND_SAMPLE
        
        # Always sample error conditions
        if attributes and attributes.get('error') == 'true':
            return SamplingResult.RECORD_AND_SAMPLE
        
        # Sample 10% of other operations
        if trace_id % 10 == 0:
            return SamplingResult.RECORD_AND_SAMPLE
        
        return SamplingResult.DROP
```

## Advanced Alerting Patterns

### Multi-Window Alerting
```yaml
# multi-window-alerts.yml
groups:
  - name: slo_alerts
    rules:
      - alert: ErrorBudgetBurnRateFast
        expr: |
          (
            sum(rate(http_requests_total{status=~"5.."}[5m])) / 
            sum(rate(http_requests_total[5m]))
          ) > (14.4 * 0.001)  # 14.4x burn rate for 1h budget consumption
        for: 2m
        labels:
          severity: critical
          burn_rate: fast
        annotations:
          summary: "Fast burn rate detected"
          description: "Error budget will be exhausted in 1 hour"
      
      - alert: ErrorBudgetBurnRateSlow
        expr: |
          (
            sum(rate(http_requests_total{status=~"5.."}[30m])) / 
            sum(rate(http_requests_total[30m]))
          ) > (6 * 0.001)  # 6x burn rate for 5h budget consumption
        for: 15m
        labels:
          severity: warning
          burn_rate: slow
        annotations:
          summary: "Slow burn rate detected"
          description: "Error budget will be exhausted in 5 hours"
```

### Correlation Rules
```yaml
# correlation-alerts.yml
groups:
  - name: correlation_alerts
    rules:
      - alert: CascadingFailure
        expr: |
          count(up == 0) by (job) > 0
          and
          increase(http_requests_total{status=~"5.."}[5m]) > 100
        for: 1m
        labels:
          severity: critical
          pattern: cascading_failure
        annotations:
          summary: "Cascading failure detected"
          description: "Multiple services down with increased error rate"
      
      - alert: PerformanceDegradation
        expr: |
          histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
          and
          rate(http_requests_total[5m]) > 100
        for: 5m
        labels:
          severity: warning
          pattern: performance_degradation
        annotations:
          summary: "Performance degradation under load"
          description: "High latency with normal request rate"
```

## Next Steps

### Advanced Certifications
1. **Certified Kubernetes Administrator (CKA)**
2. **Prometheus Certified Associate**
3. **Grafana Certified Professional**
4. **Site Reliability Engineering Certification**

### Specialization Paths
1. **Cloud-Native Observability**
2. **Security and Compliance**
3. **Performance Engineering**
4. **Chaos Engineering**

### Recommended Projects
1. Build a custom metrics exporter
2. Implement advanced sampling strategies
3. Create observability platform
4. Develop SLO monitoring system

### Community Engagement
1. Join CNCF working groups
2. Contribute to open source projects
3. Speak at conferences
4. Write technical blogs

### Continuous Learning
1. Follow observability blogs and newsletters
2. Attend conferences and meetups
3. Practice with new tools and techniques
4. Mentor junior engineers