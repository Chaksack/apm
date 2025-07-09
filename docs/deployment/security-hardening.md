# Security Hardening Guide

Comprehensive security hardening guidelines for your APM system deployment, covering network security, access control, secrets management, and compliance.

## Table of Contents
1. [Network Security](#network-security)
2. [RBAC Configuration](#rbac-configuration)
3. [Pod Security Standards](#pod-security-standards)
4. [Secrets Management](#secrets-management)
5. [Image Security](#image-security)
6. [Runtime Security](#runtime-security)
7. [Compliance and Auditing](#compliance-and-auditing)

---

## Network Security

### 1. Network Policies

#### Default Deny All Policy
```yaml
# network-policy-default-deny.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: apm-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

#### APM API Network Policy
```yaml
# network-policy-api.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: apm-api-netpol
  namespace: apm-system
spec:
  podSelector:
    matchLabels:
      app: apm-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: apm-frontend
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgresql
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
  - to: []  # Allow DNS
    ports:
    - protocol: UDP
      port: 53
```

#### Database Network Policy
```yaml
# network-policy-database.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: postgresql-netpol
  namespace: apm-system
spec:
  podSelector:
    matchLabels:
      app: postgresql
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: apm-api
    ports:
    - protocol: TCP
      port: 5432
  egress:
  - to: []  # Allow DNS
    ports:
    - protocol: UDP
      port: 53
```

### 2. Service Mesh Security (Istio)

#### Istio Installation with Security
```bash
# Install Istio with security features
istioctl install --set values.pilot.env.EXTERNAL_ISTIOD=false \
  --set values.global.meshID=apm-mesh \
  --set values.global.network=apm-network \
  --set values.global.proxy.privileged=false

# Enable sidecar injection
kubectl label namespace apm-system istio-injection=enabled
```

#### Mutual TLS Configuration
```yaml
# istio-mtls-policy.yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: apm-system
spec:
  mtls:
    mode: STRICT
---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: apm-api-authz
  namespace: apm-system
spec:
  selector:
    matchLabels:
      app: apm-api
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/apm-system/sa/apm-frontend-sa"]
    to:
    - operation:
        methods: ["GET", "POST"]
        paths: ["/api/*"]
```

### 3. Ingress Security

#### Nginx Ingress with Security Headers
```yaml
# secure-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apm-secure-ingress
  namespace: apm-system
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/hsts: "true"
    nginx.ingress.kubernetes.io/hsts-max-age: "31536000"
    nginx.ingress.kubernetes.io/hsts-include-subdomains: "true"
    nginx.ingress.kubernetes.io/ssl-protocols: "TLSv1.2 TLSv1.3"
    nginx.ingress.kubernetes.io/ssl-ciphers: "ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      add_header X-Frame-Options "SAMEORIGIN" always;
      add_header X-Content-Type-Options "nosniff" always;
      add_header X-XSS-Protection "1; mode=block" always;
      add_header Referrer-Policy "strict-origin-when-cross-origin" always;
      add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self';" always;
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
spec:
  tls:
  - hosts:
    - apm.yourdomain.com
    secretName: apm-tls-secret
  rules:
  - host: apm.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: apm-frontend
            port:
              number: 80
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: apm-api
            port:
              number: 8080
```

---

## RBAC Configuration

### 1. Service Accounts

#### APM API Service Account
```yaml
# service-accounts.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: apm-api-sa
  namespace: apm-system
automountServiceAccountToken: true
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: apm-frontend-sa
  namespace: apm-system
automountServiceAccountToken: false
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: apm-monitoring-sa
  namespace: apm-system
automountServiceAccountToken: true
```

### 2. Roles and ClusterRoles

#### APM API Role
```yaml
# rbac-roles.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: apm-api-role
  namespace: apm-system
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: apm-api-role-binding
  namespace: apm-system
subjects:
- kind: ServiceAccount
  name: apm-api-sa
  namespace: apm-system
roleRef:
  kind: Role
  name: apm-api-role
  apiGroup: rbac.authorization.k8s.io
```

#### Monitoring ClusterRole
```yaml
# monitoring-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: apm-monitoring-role
rules:
- apiGroups: [""]
  resources: ["nodes", "nodes/proxy", "services", "endpoints", "pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["extensions"]
  resources: ["ingresses"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
- nonResourceURLs: ["/metrics"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: apm-monitoring-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: apm-monitoring-role
subjects:
- kind: ServiceAccount
  name: apm-monitoring-sa
  namespace: apm-system
```

### 3. User Access Control

#### Developer Role
```yaml
# user-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: apm-developer-role
  namespace: apm-system
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log", "services", "configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: apm-developer-binding
  namespace: apm-system
subjects:
- kind: User
  name: developer@yourdomain.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: apm-developer-role
  apiGroup: rbac.authorization.k8s.io
```

#### Operations Role
```yaml
# ops-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: apm-ops-role
  namespace: apm-system
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: apm-ops-binding
  namespace: apm-system
subjects:
- kind: User
  name: ops@yourdomain.com
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: apm-ops-role
  apiGroup: rbac.authorization.k8s.io
```

---

## Pod Security Standards

### 1. Pod Security Policy (Deprecated but for reference)

#### Restricted Pod Security Policy
```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: apm-restricted-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
  readOnlyRootFilesystem: true
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 2000
```

### 2. Pod Security Standards (Current)

#### Namespace Pod Security Standards
```yaml
# pod-security-standards.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: apm-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

#### Security Context Configuration
```yaml
# secure-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apm-api-secure
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
    spec:
      serviceAccountName: apm-api-sa
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 3000
        fsGroup: 2000
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: apm-api
        image: your-registry.com/apm-api:v1.0.0
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 1000
          capabilities:
            drop:
            - ALL
        resources:
          limits:
            memory: "2Gi"
            cpu: "1"
          requests:
            memory: "1Gi"
            cpu: "500m"
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: cache
          mountPath: /app/cache
        ports:
        - containerPort: 8080
          name: http
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
      volumes:
      - name: tmp
        emptyDir: {}
      - name: cache
        emptyDir: {}
```

### 3. OPA Gatekeeper Policies

#### Gatekeeper Installation
```bash
# Install Gatekeeper
kubectl apply -f https://raw.githubusercontent.com/open-policy-agent/gatekeeper/release-3.14/deploy/gatekeeper.yaml
```

#### Security Constraint Templates
```yaml
# gatekeeper-templates.yaml
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata:
  name: k8srequiredsecuritycontext
spec:
  crd:
    spec:
      names:
        kind: K8sRequiredSecurityContext
      validation:
        openAPIV3Schema:
          type: object
          properties:
            runAsNonRoot:
              type: boolean
            readOnlyRootFilesystem:
              type: boolean
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package k8srequiredsecuritycontext

        violation[{"msg": msg}] {
          container := input.review.object.spec.containers[_]
          not container.securityContext.runAsNonRoot
          msg := "Container must run as non-root user"
        }

        violation[{"msg": msg}] {
          container := input.review.object.spec.containers[_]
          not container.securityContext.readOnlyRootFilesystem
          msg := "Container must have read-only root filesystem"
        }
---
apiVersion: config.gatekeeper.sh/v1alpha1
kind: K8sRequiredSecurityContext
metadata:
  name: must-have-security-context
spec:
  match:
    - apiGroups: ["apps"]
      kinds: ["Deployment"]
      namespaces: ["apm-system"]
  parameters:
    runAsNonRoot: true
    readOnlyRootFilesystem: true
```

---

## Secrets Management

### 1. Kubernetes Secrets Best Practices

#### Encrypted Secrets at Rest
```yaml
# encryption-config.yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
- resources:
  - secrets
  providers:
  - aescbc:
      keys:
      - name: key1
        secret: your-base64-encoded-32-byte-key
  - identity: {}
```

#### Secret Creation with Rotation
```bash
# Create secret with rotation label
kubectl create secret generic apm-db-secret \
  --from-literal=username=apm_user \
  --from-literal=password=your-secure-password \
  --from-literal=rotation-date=$(date +%Y%m%d)

# Update secret with rotation
kubectl patch secret apm-db-secret \
  --type merge \
  -p '{"data":{"password":"'$(echo -n new-password | base64)'","rotation-date":"'$(date +%Y%m%d | base64)'"}}'
```

### 2. External Secrets Operator

#### HashiCorp Vault Integration
```yaml
# vault-secret-store.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
  namespace: apm-system
spec:
  provider:
    vault:
      server: "https://vault.yourdomain.com:8200"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "apm-role"
          serviceAccountRef:
            name: "apm-api-sa"
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: apm-vault-secret
  namespace: apm-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: apm-db-secret
    creationPolicy: Owner
  data:
  - secretKey: username
    remoteRef:
      key: apm/database
      property: username
  - secretKey: password
    remoteRef:
      key: apm/database
      property: password
```

#### AWS Secrets Manager Integration
```yaml
# aws-secrets-store.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: apm-system
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-west-2
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: awssm-secret
            key: access-key
          secretAccessKeySecretRef:
            name: awssm-secret
            key: secret-access-key
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: apm-aws-secret
  namespace: apm-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: apm-db-secret
    creationPolicy: Owner
  data:
  - secretKey: password
    remoteRef:
      key: apm/database/password
```

### 3. Sealed Secrets

#### Sealed Secrets Controller
```bash
# Install Sealed Secrets
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml

# Create sealed secret
echo -n your-secret-password | kubectl create secret generic apm-secret \
  --dry-run=client --from-file=password=/dev/stdin -o yaml | \
  kubeseal -o yaml > sealed-secret.yaml
```

#### Sealed Secret Configuration
```yaml
# sealed-secret.yaml
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: apm-sealed-secret
  namespace: apm-system
spec:
  encryptedData:
    password: AgBy3i4OJSWK+PiTySYZZA9rO43cGDEQAx...
  template:
    metadata:
      name: apm-secret
      namespace: apm-system
    type: Opaque
```

---

## Image Security

### 1. Container Image Scanning

#### Trivy Security Scanner
```bash
# Scan image for vulnerabilities
trivy image your-registry.com/apm-api:v1.0.0

# Generate report
trivy image --format json --output scan-report.json your-registry.com/apm-api:v1.0.0
```

#### Admission Controller for Image Scanning
```yaml
# image-scanner-admission.yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionWebhook
metadata:
  name: image-scanner-webhook
webhooks:
- name: image-scanner.security.io
  clientConfig:
    service:
      name: image-scanner
      namespace: security-system
      path: /validate
  rules:
  - operations: ["CREATE", "UPDATE"]
    apiGroups: [""]
    apiVersions: ["v1"]
    resources: ["pods"]
  admissionReviewVersions: ["v1", "v1beta1"]
  sideEffects: None
  failurePolicy: Fail
```

### 2. Image Signing and Verification

#### Cosign Image Signing
```bash
# Generate key pair
cosign generate-key-pair

# Sign image
cosign sign --key cosign.key your-registry.com/apm-api:v1.0.0

# Verify signature
cosign verify --key cosign.pub your-registry.com/apm-api:v1.0.0
```

#### Admission Controller for Image Verification
```yaml
# image-verification-policy.yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: verify-image-signature
spec:
  validationFailureAction: enforce
  background: false
  rules:
  - name: check-image-signature
    match:
      any:
      - resources:
          kinds:
          - Pod
          namespaces:
          - apm-system
    verifyImages:
    - imageReferences:
      - "your-registry.com/apm-*"
      attestors:
      - entries:
        - keys:
            publicKeys: |-
              -----BEGIN PUBLIC KEY-----
              MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE...
              -----END PUBLIC KEY-----
```

### 3. Distroless Images

#### Distroless Dockerfile
```dockerfile
# Dockerfile.distroless
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/main /
USER 65532:65532
ENTRYPOINT ["/main"]
```

---

## Runtime Security

### 1. Falco Runtime Security

#### Falco Installation
```bash
# Install Falco using Helm
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm install falco falcosecurity/falco \
  --namespace falco-system \
  --create-namespace \
  --set falco.grpc.enabled=true \
  --set falco.grpcOutput.enabled=true
```

#### Custom Falco Rules
```yaml
# falco-rules.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: falco-rules
  namespace: falco-system
data:
  apm_rules.yaml: |
    - rule: Unauthorized Process in APM Container
      desc: Detect unauthorized processes in APM containers
      condition: >
        spawned_process and
        container.image.repository contains "apm" and
        not proc.name in (apm-api, apm-frontend)
      output: >
        Unauthorized process spawned in APM container
        (user=%user.name command=%proc.cmdline container_id=%container.id
        image=%container.image.repository)
      priority: WARNING
    
    - rule: Sensitive File Access in APM
      desc: Detect access to sensitive files in APM containers
      condition: >
        open_read and
        container.image.repository contains "apm" and
        (fd.name startswith /etc/passwd or
         fd.name startswith /etc/shadow or
         fd.name startswith /etc/sudoers)
      output: >
        Sensitive file accessed in APM container
        (user=%user.name file=%fd.name container_id=%container.id
        image=%container.image.repository)
      priority: CRITICAL
```

### 2. AppArmor Profiles

#### AppArmor Profile for APM API
```bash
# apparmor-profile
#include <tunables/global>

profile apm-api flags=(attach_disconnected,mediate_deleted) {
  #include <abstractions/base>

  # Allow network access
  network inet tcp,
  network inet udp,

  # Allow read access to necessary files
  /usr/bin/apm-api ix,
  /etc/passwd r,
  /etc/group r,
  /etc/ssl/certs/** r,

  # Allow write access to specific directories
  /tmp/** rw,
  /var/log/apm/** rw,

  # Deny access to sensitive files
  deny /etc/shadow r,
  deny /etc/sudoers r,
  deny /root/** rw,

  # Allow necessary system calls
  capability net_bind_service,
  capability setuid,
  capability setgid,
}
```

#### Apply AppArmor Profile
```yaml
# apparmor-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apm-api-apparmor
spec:
  template:
    metadata:
      annotations:
        container.apparmor.security.beta.kubernetes.io/apm-api: localhost/apm-api
    spec:
      containers:
      - name: apm-api
        image: your-registry.com/apm-api:v1.0.0
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
```

---

## Compliance and Auditing

### 1. Audit Logging

#### Audit Policy Configuration
```yaml
# audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
# Log all requests to APM namespace
- level: RequestResponse
  namespaces: ["apm-system"]
  resources:
  - group: ""
    resources: ["secrets", "configmaps"]
  - group: "apps"
    resources: ["deployments", "statefulsets"]

# Log security-related events
- level: Metadata
  omitStages:
  - RequestReceived
  resources:
  - group: ""
    resources: ["secrets", "configmaps"]
  - group: "rbac.authorization.k8s.io"
    resources: ["roles", "rolebindings", "clusterroles", "clusterrolebindings"]

# Log failed requests
- level: Request
  omitStages:
  - RequestReceived
  namespaces: ["apm-system"]
  verbs: ["create", "update", "patch", "delete"]
  resources:
  - group: ""
    resources: ["pods", "services"]
```

### 2. Compliance Scanning

#### Kube-bench CIS Benchmarks
```bash
# Run CIS Kubernetes Benchmark
kubectl apply -f https://raw.githubusercontent.com/aquasecurity/kube-bench/main/job.yaml

# View results
kubectl logs job/kube-bench
```

#### Kube-score Security Analysis
```bash
# Analyze deployment security
kube-score score deployment.yaml

# Generate report
kube-score score deployment.yaml --output-format json > security-report.json
```

### 3. Continuous Compliance

#### Compliance Monitoring CronJob
```yaml
# compliance-monitoring.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: compliance-scan
  namespace: apm-system
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: compliance-scanner
            image: aquasec/kube-bench:latest
            command:
            - /bin/sh
            - -c
            - |
              kube-bench run --targets node,policies,managedservices > /tmp/compliance-report.txt
              # Upload to compliance system
              curl -X POST -H "Content-Type: text/plain" \
                --data-binary @/tmp/compliance-report.txt \
                $COMPLIANCE_ENDPOINT
            env:
            - name: COMPLIANCE_ENDPOINT
              value: "https://compliance.yourdomain.com/api/reports"
          restartPolicy: OnFailure
```

## Security Monitoring Dashboard

### Grafana Security Dashboard
```json
{
  "dashboard": {
    "title": "APM Security Dashboard",
    "panels": [
      {
        "title": "Failed Authentication Attempts",
        "type": "graph",
        "targets": [
          {
            "expr": "increase(kubernetes_audit_total{verb=\"create\",objectRef_resource=\"secrets\",responseStatus_code!=\"200\"}[5m])",
            "legendFormat": "Failed Secret Access"
          }
        ]
      },
      {
        "title": "Privilege Escalation Attempts",
        "type": "graph",
        "targets": [
          {
            "expr": "increase(falco_events_total{rule=\"Privilege Escalation\"}[5m])",
            "legendFormat": "Privilege Escalation"
          }
        ]
      },
      {
        "title": "Network Policy Violations",
        "type": "graph",
        "targets": [
          {
            "expr": "increase(kubernetes_audit_total{verb=\"create\",objectRef_resource=\"pods\",responseStatus_code=\"403\"}[5m])",
            "legendFormat": "Network Policy Violations"
          }
        ]
      }
    ]
  }
}
```

---

## Security Checklist

### Pre-Deployment Security
- [ ] Network policies configured
- [ ] RBAC roles and bindings created
- [ ] Pod security standards enforced
- [ ] Secrets management configured
- [ ] Image scanning enabled
- [ ] Security contexts defined

### Runtime Security
- [ ] Falco rules configured
- [ ] AppArmor/SELinux profiles applied
- [ ] Audit logging enabled
- [ ] Compliance scanning scheduled
- [ ] Security monitoring dashboard deployed

### Post-Deployment
- [ ] Security scan results reviewed
- [ ] Compliance benchmarks passed
- [ ] Access controls tested
- [ ] Incident response procedures verified
- [ ] Security documentation updated

---

**Implementation Time**: 4-6 hours  
**Complexity**: Advanced  
**Prerequisites**: Kubernetes cluster, security tools installed  
**Next Steps**: Regular security audits and compliance monitoring