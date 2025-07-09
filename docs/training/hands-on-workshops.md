# Hands-On Observability Workshops

## Workshop Overview
These practical workshops provide hands-on experience with observability tools and techniques. Each workshop includes setup instructions, guided exercises, and real-world scenarios.

## Workshop 1: Metrics and Alerting

### Duration: 4 hours

### Objectives
- Set up metrics collection
- Create effective dashboards
- Implement alerting rules
- Practice incident response

### Prerequisites
- Basic understanding of observability concepts
- Access to lab environment
- Familiarity with command line

### Lab Environment Setup

#### Required Tools
```bash
# Install Prometheus
wget https://github.com/prometheus/prometheus/releases/latest/download/prometheus-linux-amd64.tar.gz
tar -xvf prometheus-linux-amd64.tar.gz
cd prometheus-*
./prometheus --config.file=prometheus.yml

# Install Grafana
sudo apt-get install -y adduser libfontconfig1
wget https://dl.grafana.com/oss/release/grafana_9.5.0_amd64.deb
sudo dpkg -i grafana_9.5.0_amd64.deb

# Install Node Exporter
wget https://github.com/prometheus/node_exporter/releases/latest/download/node_exporter-linux-amd64.tar.gz
tar -xvf node_exporter-linux-amd64.tar.gz
cd node_exporter-*
./node_exporter
```

#### Sample Application
```python
# app.py - Simple Flask application with metrics
from flask import Flask, request, jsonify
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
import random
import time

app = Flask(__name__)

# Metrics
REQUEST_COUNT = Counter('http_requests_total', 'Total HTTP requests', ['method', 'endpoint', 'status'])
REQUEST_LATENCY = Histogram('http_request_duration_seconds', 'HTTP request latency', ['method', 'endpoint'])

@app.route('/')
def hello():
    start_time = time.time()
    
    # Simulate some processing time
    time.sleep(random.uniform(0.1, 0.5))
    
    # Record metrics
    REQUEST_COUNT.labels(method='GET', endpoint='/', status='200').inc()
    REQUEST_LATENCY.labels(method='GET', endpoint='/').observe(time.time() - start_time)
    
    return jsonify({'message': 'Hello World!', 'timestamp': time.time()})

@app.route('/api/users/<user_id>')
def get_user(user_id):
    start_time = time.time()
    
    # Simulate database lookup
    time.sleep(random.uniform(0.05, 0.2))
    
    # Simulate occasional errors
    if random.random() < 0.05:
        REQUEST_COUNT.labels(method='GET', endpoint='/api/users', status='500').inc()
        return jsonify({'error': 'Internal server error'}), 500
    
    REQUEST_COUNT.labels(method='GET', endpoint='/api/users', status='200').inc()
    REQUEST_LATENCY.labels(method='GET', endpoint='/api/users').observe(time.time() - start_time)
    
    return jsonify({'user_id': user_id, 'name': f'User {user_id}', 'active': True})

@app.route('/metrics')
def metrics():
    return generate_latest(), 200, {'Content-Type': CONTENT_TYPE_LATEST}

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

### Exercise 1: Metrics Collection Setup

#### Task 1.1: Configure Prometheus
Create `prometheus.yml`:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
  
  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']
  
  - job_name: 'sample-app'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    scrape_interval: 5s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

#### Task 1.2: Generate Load
```bash
# Load generation script
#!/bin/bash
for i in {1..1000}; do
    curl -s "http://localhost:8080/" > /dev/null
    curl -s "http://localhost:8080/api/users/$((RANDOM % 100))" > /dev/null
    sleep 0.1
done
```

#### Task 1.3: Explore Metrics
Navigate to `http://localhost:9090` and run these queries:
```promql
# Basic queries
http_requests_total
rate(http_requests_total[5m])
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Advanced queries
sum(rate(http_requests_total[5m])) by (endpoint)
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
```

### Exercise 2: Dashboard Creation

#### Task 2.1: Set up Grafana
1. Access Grafana at `http://localhost:3000`
2. Login with admin/admin
3. Add Prometheus data source: `http://localhost:9090`

#### Task 2.2: Create Service Dashboard
Create panels for:
```json
{
  "panels": [
    {
      "title": "Request Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "sum(rate(http_requests_total[5m])) by (endpoint)",
          "legendFormat": "{{endpoint}}"
        }
      ]
    },
    {
      "title": "Error Rate",
      "type": "singlestat",
      "targets": [
        {
          "expr": "rate(http_requests_total{status=~\"5..\"}[5m]) / rate(http_requests_total[5m]) * 100",
          "legendFormat": "Error Rate %"
        }
      ]
    },
    {
      "title": "Response Time",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
          "legendFormat": "95th percentile"
        },
        {
          "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
          "legendFormat": "50th percentile"
        }
      ]
    }
  ]
}
```

#### Task 2.3: Create Alerting Rules
Create `alert_rules.yml`:
```yaml
groups:
  - name: application_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100 > 5
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }}% for the last 5 minutes"
      
      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High response time detected"
          description: "95th percentile response time is {{ $value }}s"
```

### Exercise 3: Incident Simulation

#### Scenario 1: High Error Rate
```python
# Modify app.py to introduce errors
@app.route('/api/users/<user_id>')
def get_user(user_id):
    # Increase error rate to 20%
    if random.random() < 0.20:
        REQUEST_COUNT.labels(method='GET', endpoint='/api/users', status='500').inc()
        return jsonify({'error': 'Database connection failed'}), 500
```

**Tasks**:
1. Deploy the modified application
2. Observe alert firing
3. Investigate using dashboards
4. Implement fix and verify recovery

#### Scenario 2: Performance Degradation
```python
# Add artificial latency
@app.route('/api/users/<user_id>')
def get_user(user_id):
    # Add high latency
    time.sleep(random.uniform(1.0, 3.0))
```

**Tasks**:
1. Identify performance degradation
2. Correlate with business impact
3. Implement optimization
4. Validate improvements

## Workshop 2: Logging and Analysis

### Duration: 3 hours

### Objectives
- Implement structured logging
- Set up log aggregation
- Create log-based alerts
- Practice log analysis

### Lab Environment Setup

#### ELK Stack Setup
```bash
# Elasticsearch
docker run -d --name elasticsearch -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" elasticsearch:7.14.0

# Logstash
docker run -d --name logstash -p 5000:5000 -p 9600:9600 logstash:7.14.0

# Kibana
docker run -d --name kibana -p 5601:5601 --link elasticsearch:elasticsearch kibana:7.14.0
```

#### Structured Logging Example
```python
import logging
import json
from datetime import datetime
import uuid

class StructuredLogger:
    def __init__(self, service_name):
        self.service_name = service_name
        self.logger = logging.getLogger(service_name)
        self.logger.setLevel(logging.INFO)
        
        handler = logging.StreamHandler()
        handler.setFormatter(logging.Formatter('%(message)s'))
        self.logger.addHandler(handler)
    
    def log(self, level, message, **kwargs):
        log_entry = {
            'timestamp': datetime.utcnow().isoformat(),
            'level': level,
            'service': self.service_name,
            'message': message,
            'request_id': kwargs.get('request_id', str(uuid.uuid4())),
            **kwargs
        }
        self.logger.info(json.dumps(log_entry))

# Usage in Flask app
logger = StructuredLogger('user-service')

@app.route('/api/users/<user_id>')
def get_user(user_id):
    request_id = request.headers.get('X-Request-ID', str(uuid.uuid4()))
    
    logger.log('INFO', 'User lookup started', 
               request_id=request_id, 
               user_id=user_id,
               endpoint='/api/users')
    
    try:
        # Simulate database lookup
        time.sleep(random.uniform(0.05, 0.2))
        
        if random.random() < 0.05:
            logger.log('ERROR', 'Database connection failed',
                       request_id=request_id,
                       user_id=user_id,
                       error_code='DB_CONNECTION_FAILED')
            return jsonify({'error': 'Internal server error'}), 500
        
        logger.log('INFO', 'User lookup completed',
                   request_id=request_id,
                   user_id=user_id,
                   response_time=time.time() - start_time)
        
        return jsonify({'user_id': user_id, 'name': f'User {user_id}'})
    
    except Exception as e:
        logger.log('ERROR', 'Unexpected error',
                   request_id=request_id,
                   user_id=user_id,
                   error=str(e))
        return jsonify({'error': 'Internal server error'}), 500
```

### Exercise 1: Log Analysis

#### Task 1.1: Pattern Recognition
Analyze sample logs:
```json
{"timestamp": "2024-01-15T10:30:00Z", "level": "INFO", "service": "user-service", "message": "User lookup started", "request_id": "req-123", "user_id": "12345"}
{"timestamp": "2024-01-15T10:30:01Z", "level": "ERROR", "service": "user-service", "message": "Database connection failed", "request_id": "req-123", "user_id": "12345", "error_code": "DB_CONNECTION_FAILED"}
{"timestamp": "2024-01-15T10:30:02Z", "level": "INFO", "service": "user-service", "message": "User lookup started", "request_id": "req-124", "user_id": "12346"}
```

**Questions**:
1. What patterns do you identify?
2. How would you track request success/failure rates?
3. What alerting rules would you create?

#### Task 1.2: Kibana Queries
Create queries for:
```json
{
  "query": {
    "bool": {
      "must": [
        {"match": {"service": "user-service"}},
        {"match": {"level": "ERROR"}}
      ],
      "filter": [
        {"range": {"timestamp": {"gte": "now-1h"}}}
      ]
    }
  }
}
```

## Workshop 3: Distributed Tracing

### Duration: 3 hours

### Objectives
- Set up distributed tracing
- Instrument microservices
- Analyze trace data
- Optimize performance

### Lab Environment Setup

#### Jaeger Setup
```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

#### Microservice Example
```python
# Service A (API Gateway)
from opentelemetry import trace
from opentelemetry.exporter.jaeger.thrift import JaegerExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
import requests

# Configure tracing
trace.set_tracer_provider(TracerProvider())
tracer = trace.get_tracer(__name__)

jaeger_exporter = JaegerExporter(
    agent_host_name="localhost",
    agent_port=6831,
)
span_processor = BatchSpanProcessor(jaeger_exporter)
trace.get_tracer_provider().add_span_processor(span_processor)

@app.route('/api/user/<user_id>')
def get_user_profile(user_id):
    with tracer.start_as_current_span("get_user_profile") as span:
        span.set_attribute("user.id", user_id)
        
        # Call user service
        with tracer.start_as_current_span("call_user_service") as child_span:
            response = requests.get(f"http://user-service:8080/users/{user_id}")
            child_span.set_attribute("http.status_code", response.status_code)
        
        # Call profile service
        with tracer.start_as_current_span("call_profile_service") as child_span:
            profile_response = requests.get(f"http://profile-service:8080/profiles/{user_id}")
            child_span.set_attribute("http.status_code", profile_response.status_code)
        
        return jsonify({"user": response.json(), "profile": profile_response.json()})
```

### Exercise 1: Trace Analysis

#### Task 1.1: Identify Bottlenecks
Analyze trace data to find:
1. Slowest operations
2. Error patterns
3. Service dependencies
4. Optimization opportunities

#### Task 1.2: Performance Optimization
Based on trace analysis:
1. Implement caching
2. Optimize database queries
3. Add async processing
4. Validate improvements

## Assessment Criteria

### Workshop 1: Metrics and Alerting
- [ ] Successfully set up Prometheus and Grafana
- [ ] Created comprehensive dashboards
- [ ] Implemented effective alerting rules
- [ ] Demonstrated incident response skills

### Workshop 2: Logging and Analysis
- [ ] Implemented structured logging
- [ ] Set up log aggregation
- [ ] Created log-based alerts
- [ ] Analyzed log patterns effectively

### Workshop 3: Distributed Tracing
- [ ] Set up distributed tracing
- [ ] Instrumented microservices
- [ ] Analyzed trace data
- [ ] Identified performance optimizations

## Practical Assessment

### Scenario: E-commerce Platform Monitoring
You are tasked with implementing observability for an e-commerce platform with the following services:
- API Gateway
- User Service
- Product Service
- Order Service
- Payment Service
- Notification Service

### Requirements
1. Set up comprehensive monitoring
2. Create service dashboards
3. Implement SLO-based alerting
4. Configure distributed tracing
5. Set up log aggregation
6. Create incident response procedures

### Evaluation Criteria
- **Technical Implementation** (40%)
  - Correct tool configuration
  - Proper instrumentation
  - Effective queries and dashboards

- **Observability Strategy** (30%)
  - Appropriate SLI/SLO definitions
  - Effective alerting rules
  - Comprehensive coverage

- **Practical Skills** (20%)
  - Troubleshooting abilities
  - Performance optimization
  - Incident response

- **Documentation** (10%)
  - Clear procedures
  - Troubleshooting guides
  - Knowledge transfer

## Additional Resources

### Sample Configurations
- Prometheus configurations for various scenarios
- Grafana dashboard templates
- Jaeger deployment examples
- ELK stack configurations

### Troubleshooting Guides
- Common setup issues
- Performance tuning tips
- Debugging techniques
- Best practices

### Next Steps
1. Complete advanced topics workshop
2. Practice with real-world scenarios
3. Pursue certification
4. Join observability community