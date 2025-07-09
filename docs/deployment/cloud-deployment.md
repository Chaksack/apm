# Cloud Deployment Guide

Deploy your APM system across major cloud providers with optimized configurations for AWS, Google Cloud Platform (GCP), and Microsoft Azure.

## Table of Contents
1. [AWS Deployment](#aws-deployment)
2. [Google Cloud Platform (GCP)](#google-cloud-platform-gcp)
3. [Microsoft Azure](#microsoft-azure)
4. [Multi-Cloud Setup](#multi-cloud-setup)
5. [Cost Optimization](#cost-optimization)

---

## AWS Deployment

### Prerequisites
- AWS CLI configured with appropriate permissions
- eksctl for EKS cluster management
- Terraform (optional) for infrastructure as code

### 1. EKS Cluster Setup

#### Create EKS Cluster
```bash
# Create EKS cluster with eksctl
eksctl create cluster \
  --name apm-cluster \
  --region us-west-2 \
  --version 1.25 \
  --nodegroup-name apm-nodes \
  --node-type m5.xlarge \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 10 \
  --managed \
  --with-oidc \
  --ssh-access \
  --ssh-public-key your-key-pair

# Configure kubectl
aws eks update-kubeconfig --region us-west-2 --name apm-cluster
```

#### Terraform Configuration
```hcl
# terraform/aws/main.tf
provider "aws" {
  region = var.aws_region
}

module "eks" {
  source = "terraform-aws-modules/eks/aws"
  version = "19.15.3"

  cluster_name    = "apm-cluster"
  cluster_version = "1.25"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  # EKS Managed Node Groups
  eks_managed_node_groups = {
    apm_nodes = {
      instance_types = ["m5.xlarge"]
      min_size       = 3
      max_size       = 10
      desired_size   = 3
      
      k8s_labels = {
        Environment = "production"
        Application = "apm"
      }
    }
  }

  # AWS Load Balancer Controller
  enable_irsa = true
  
  tags = {
    Environment = "production"
    Application = "apm"
  }
}

module "vpc" {
  source = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"

  name = "apm-vpc"
  cidr = "10.0.0.0/16"

  azs             = ["us-west-2a", "us-west-2b", "us-west-2c"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]

  enable_nat_gateway = true
  enable_vpn_gateway = true
  enable_dns_hostnames = true
  enable_dns_support = true

  tags = {
    "kubernetes.io/cluster/apm-cluster" = "shared"
  }
}
```

### 2. AWS-Specific Services Integration

#### RDS PostgreSQL Setup
```yaml
# aws-rds-values.yaml
postgresql:
  enabled: false  # Disable in-cluster PostgreSQL

externalDatabase:
  host: "apm-db.cluster-xxx.us-west-2.rds.amazonaws.com"
  port: 5432
  database: "apm_db"
  username: "apm_user"
  existingSecret: "aws-rds-secret"
  existingSecretPasswordKey: "password"

# Create RDS instance
aws rds create-db-cluster \
  --db-cluster-identifier apm-db-cluster \
  --engine aurora-postgresql \
  --engine-version 14.6 \
  --master-username apm_user \
  --master-user-password your-secure-password \
  --database-name apm_db \
  --vpc-security-group-ids sg-xxxxx \
  --db-subnet-group-name apm-db-subnet-group \
  --backup-retention-period 7 \
  --preferred-backup-window 03:00-04:00 \
  --preferred-maintenance-window sun:04:00-sun:05:00 \
  --storage-encrypted
```

#### ElastiCache Redis
```bash
# Create ElastiCache Redis cluster
aws elasticache create-replication-group \
  --replication-group-id apm-redis \
  --description "APM Redis cluster" \
  --cache-node-type cache.r6g.large \
  --engine redis \
  --engine-version 7.0 \
  --num-cache-clusters 3 \
  --subnet-group-name apm-redis-subnet-group \
  --security-group-ids sg-xxxxx \
  --at-rest-encryption-enabled \
  --transit-encryption-enabled \
  --auth-token your-redis-auth-token
```

#### Application Load Balancer
```yaml
# aws-alb-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apm-ingress
  annotations:
    kubernetes.io/ingress.class: alb
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/ssl-redirect: '443'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-west-2:123456789012:certificate/xxx
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS": 443}]'
spec:
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

### 3. AWS Storage Configuration

#### EBS Storage Classes
```yaml
# aws-storage-classes.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
  encrypted: "true"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer

---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: slow-hdd
provisioner: ebs.csi.aws.com
parameters:
  type: sc1
  encrypted: "true"
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

#### S3 Backup Integration
```yaml
# aws-backup-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: backup-config
data:
  backup-script.sh: |
    #!/bin/bash
    DATE=$(date +%Y%m%d%H%M%S)
    BACKUP_FILE="/tmp/apm-backup-${DATE}.sql.gz"
    
    # Database backup
    pg_dump -h $DB_HOST -U $DB_USER $DB_NAME | gzip > $BACKUP_FILE
    
    # Upload to S3
    aws s3 cp $BACKUP_FILE s3://your-backup-bucket/apm-backups/
    
    # Cleanup
    rm $BACKUP_FILE
```

### 4. AWS Monitoring Integration

#### CloudWatch Container Insights
```yaml
# cloudwatch-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-info
  namespace: amazon-cloudwatch
data:
  cluster.name: apm-cluster
  logs.region: us-west-2
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cloudwatch-agent
  namespace: amazon-cloudwatch
spec:
  selector:
    matchLabels:
      name: cloudwatch-agent
  template:
    metadata:
      labels:
        name: cloudwatch-agent
    spec:
      containers:
      - name: cloudwatch-agent
        image: amazon/cloudwatch-agent:1.247354.0b252275
        env:
        - name: AWS_REGION
          valueFrom:
            configMapKeyRef:
              name: cluster-info
              key: logs.region
        - name: CLUSTER_NAME
          valueFrom:
            configMapKeyRef:
              name: cluster-info
              key: cluster.name
        volumeMounts:
        - name: cwagentconfig
          mountPath: /etc/cwagentconfig
```

---

## Google Cloud Platform (GCP)

### Prerequisites
- gcloud CLI configured with appropriate permissions
- GKE cluster access
- Terraform (optional) for infrastructure as code

### 1. GKE Cluster Setup

#### Create GKE Cluster
```bash
# Create GKE cluster
gcloud container clusters create apm-cluster \
  --zone us-central1-a \
  --machine-type n1-standard-4 \
  --num-nodes 3 \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 10 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-network-policy \
  --enable-ip-alias \
  --enable-stackdriver-kubernetes

# Get credentials
gcloud container clusters get-credentials apm-cluster --zone us-central1-a
```

#### Terraform Configuration
```hcl
# terraform/gcp/main.tf
provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_container_cluster" "apm_cluster" {
  name     = "apm-cluster"
  location = var.zone

  remove_default_node_pool = true
  initial_node_count       = 1

  network    = google_compute_network.vpc.name
  subnetwork = google_compute_subnetwork.subnet.name

  ip_allocation_policy {
    cluster_secondary_range_name  = "k8s-pod-range"
    services_secondary_range_name = "k8s-service-range"
  }

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  monitoring_config {
    enable_components = [
      "SYSTEM_COMPONENTS",
      "WORKLOADS"
    ]
  }

  logging_config {
    enable_components = [
      "SYSTEM_COMPONENTS",
      "WORKLOADS"
    ]
  }
}

resource "google_container_node_pool" "apm_nodes" {
  name       = "apm-node-pool"
  location   = var.zone
  cluster    = google_container_cluster.apm_cluster.name
  node_count = 3

  autoscaling {
    min_node_count = 3
    max_node_count = 10
  }

  node_config {
    preemptible  = false
    machine_type = "n1-standard-4"

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    labels = {
      environment = "production"
      application = "apm"
    }
  }
}
```

### 2. GCP-Specific Services Integration

#### Cloud SQL PostgreSQL
```yaml
# gcp-cloudsql-values.yaml
postgresql:
  enabled: false

externalDatabase:
  host: "127.0.0.1"  # Via Cloud SQL Proxy
  port: 5432
  database: "apm_db"
  username: "apm_user"
  existingSecret: "gcp-cloudsql-secret"
  existingSecretPasswordKey: "password"

# Cloud SQL Proxy deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cloudsql-proxy
spec:
  selector:
    matchLabels:
      app: cloudsql-proxy
  template:
    metadata:
      labels:
        app: cloudsql-proxy
    spec:
      containers:
      - name: cloudsql-proxy
        image: gcr.io/cloudsql-docker/gce-proxy:1.33.2
        command:
        - "/cloud_sql_proxy"
        - "-instances=your-project:us-central1:apm-db=tcp:0.0.0.0:5432"
        - "-credential_file=/secrets/cloudsql/credentials.json"
        volumeMounts:
        - name: cloudsql-instance-credentials
          mountPath: /secrets/cloudsql
          readOnly: true
        ports:
        - containerPort: 5432
      volumes:
      - name: cloudsql-instance-credentials
        secret:
          secretName: cloudsql-instance-credentials
```

#### Memorystore Redis
```bash
# Create Memorystore Redis instance
gcloud redis instances create apm-redis \
  --size=5 \
  --region=us-central1 \
  --network=projects/your-project/global/networks/default \
  --redis-version=redis_6_x \
  --tier=standard \
  --auth-enabled \
  --transit-encryption-mode=SERVER_AUTHENTICATION
```

#### Google Cloud Load Balancer
```yaml
# gcp-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apm-ingress
  annotations:
    kubernetes.io/ingress.class: gce
    kubernetes.io/ingress.global-static-ip-name: apm-ip
    networking.gke.io/managed-certificates: apm-ssl-cert
    kubernetes.io/ingress.allow-http: "false"
spec:
  rules:
  - host: apm.yourdomain.com
    http:
      paths:
      - path: /*
        pathType: ImplementationSpecific
        backend:
          service:
            name: apm-frontend
            port:
              number: 80
      - path: /api/*
        pathType: ImplementationSpecific
        backend:
          service:
            name: apm-api
            port:
              number: 8080
---
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: apm-ssl-cert
spec:
  domains:
  - apm.yourdomain.com
```

### 3. GCP Storage Configuration

#### Persistent Disk Storage Classes
```yaml
# gcp-storage-classes.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
  zones: us-central1-a,us-central1-b,us-central1-c
  replication-type: regional-pd
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer

---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: slow-hdd
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
  zones: us-central1-a,us-central1-b,us-central1-c
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

### 4. GCP Monitoring Integration

#### Google Cloud Monitoring
```yaml
# gcp-monitoring-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: stackdriver-config
data:
  config.yaml: |
    cluster_name: apm-cluster
    cluster_location: us-central1-a
    project_id: your-project-id
    enable_workload_metrics: true
    enable_resource_metrics: true
```

---

## Microsoft Azure

### Prerequisites
- Azure CLI configured with appropriate permissions
- Azure Kubernetes Service (AKS) access
- Terraform (optional) for infrastructure as code

### 1. AKS Cluster Setup

#### Create AKS Cluster
```bash
# Create resource group
az group create --name apm-rg --location eastus

# Create AKS cluster
az aks create \
  --resource-group apm-rg \
  --name apm-cluster \
  --node-count 3 \
  --node-vm-size Standard_D4s_v3 \
  --enable-addons monitoring \
  --enable-managed-identity \
  --generate-ssh-keys \
  --network-plugin azure \
  --enable-cluster-autoscaler \
  --min-count 3 \
  --max-count 10

# Get credentials
az aks get-credentials --resource-group apm-rg --name apm-cluster
```

#### Terraform Configuration
```hcl
# terraform/azure/main.tf
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "apm" {
  name     = "apm-rg"
  location = "East US"
}

resource "azurerm_kubernetes_cluster" "apm" {
  name                = "apm-cluster"
  location            = azurerm_resource_group.apm.location
  resource_group_name = azurerm_resource_group.apm.name
  dns_prefix          = "apm"

  default_node_pool {
    name                = "default"
    node_count          = 3
    vm_size             = "Standard_D4s_v3"
    enable_auto_scaling = true
    min_count          = 3
    max_count          = 10
  }

  identity {
    type = "SystemAssigned"
  }

  addon_profile {
    oms_agent {
      enabled                    = true
      log_analytics_workspace_id = azurerm_log_analytics_workspace.apm.id
    }
  }

  tags = {
    Environment = "Production"
    Application = "APM"
  }
}

resource "azurerm_log_analytics_workspace" "apm" {
  name                = "apm-logs"
  location            = azurerm_resource_group.apm.location
  resource_group_name = azurerm_resource_group.apm.name
  sku                 = "PerGB2018"
  retention_in_days   = 30
}
```

### 2. Azure-Specific Services Integration

#### Azure Database for PostgreSQL
```yaml
# azure-postgresql-values.yaml
postgresql:
  enabled: false

externalDatabase:
  host: "apm-db.postgres.database.azure.com"
  port: 5432
  database: "apm_db"
  username: "apm_user@apm-db"
  existingSecret: "azure-postgresql-secret"
  existingSecretPasswordKey: "password"

# Create Azure Database for PostgreSQL
az postgres server create \
  --name apm-db \
  --resource-group apm-rg \
  --location eastus \
  --admin-user apm_user \
  --admin-password your-secure-password \
  --sku-name GP_Gen5_4 \
  --version 14 \
  --storage-size 51200 \
  --backup-retention 7 \
  --ssl-enforcement Enabled
```

#### Azure Cache for Redis
```bash
# Create Azure Cache for Redis
az redis create \
  --name apm-redis \
  --resource-group apm-rg \
  --location eastus \
  --sku Premium \
  --vm-size P1 \
  --enable-non-ssl-port \
  --redis-configuration maxmemory-policy=allkeys-lru
```

#### Azure Application Gateway
```yaml
# azure-ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apm-ingress
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/ssl-redirect: "true"
    appgw.ingress.kubernetes.io/certificate-name: apm-ssl-cert
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

### 3. Azure Storage Configuration

#### Azure Disk Storage Classes
```yaml
# azure-storage-classes.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: fast-ssd
provisioner: kubernetes.io/azure-disk
parameters:
  storageaccounttype: Premium_LRS
  kind: Managed
  cachingmode: ReadOnly
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer

---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: slow-hdd
provisioner: kubernetes.io/azure-disk
parameters:
  storageaccounttype: Standard_LRS
  kind: Managed
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

---

## Multi-Cloud Setup

### 1. Multi-Cloud Architecture
```yaml
# multi-cloud-values.yaml
global:
  multiCloud:
    enabled: true
    primaryRegion: "aws-us-west-2"
    secondaryRegions:
      - "gcp-us-central1"
      - "azure-eastus"

# Cross-cloud networking
networking:
  crossCloudVPN:
    enabled: true
    providers:
      aws:
        vpcId: "vpc-12345678"
        subnets: ["subnet-12345678", "subnet-87654321"]
      gcp:
        networkName: "apm-network"
        subnetName: "apm-subnet"
      azure:
        vnetName: "apm-vnet"
        subnetName: "apm-subnet"
```

### 2. Cross-Cloud Load Balancing
```yaml
# global-load-balancer.yaml
apiVersion: v1
kind: Service
metadata:
  name: apm-global-lb
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/azure-load-balancer-resource-group: apm-rg
spec:
  type: LoadBalancer
  selector:
    app: apm-api
  ports:
  - port: 80
    targetPort: 8080
```

---

## Cost Optimization

### 1. Resource Optimization

#### Spot Instances/Preemptible Nodes
```yaml
# spot-instances.yaml
# AWS Spot Instances
apiVersion: karpenter.sh/v1alpha5
kind: NodePool
metadata:
  name: spot-nodepool
spec:
  requirements:
    - key: karpenter.sh/capacity-type
      operator: In
      values: ["spot"]
    - key: node.kubernetes.io/instance-type
      operator: In
      values: ["m5.large", "m5.xlarge", "c5.large"]
  limits:
    resources:
      cpu: 1000
      memory: 1000Gi
  disruption:
    consolidationPolicy: WhenEmpty
    consolidateAfter: 30s
```

#### Horizontal Pod Autoscaler for Cost Efficiency
```yaml
# cost-efficient-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: apm-cost-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: apm-api
  minReplicas: 1  # Reduced for cost
  maxReplicas: 5  # Capped for cost control
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80  # Higher threshold for cost efficiency
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 600  # Slower scale-down
      policies:
      - type: Percent
        value: 25
        periodSeconds: 120
```

### 2. Storage Cost Optimization

#### Lifecycle Policies
```yaml
# storage-lifecycle.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: storage-lifecycle-policy
data:
  policy.json: |
    {
      "Rules": [
        {
          "ID": "APMBackupLifecycle",
          "Status": "Enabled",
          "Transitions": [
            {
              "Days": 30,
              "StorageClass": "STANDARD_IA"
            },
            {
              "Days": 90,
              "StorageClass": "GLACIER"
            },
            {
              "Days": 365,
              "StorageClass": "DEEP_ARCHIVE"
            }
          ]
        }
      ]
    }
```

### 3. Monitoring and Alerting for Cost Control

#### Cost Monitoring Dashboard
```yaml
# cost-monitoring-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cost-monitoring-config
data:
  cost-alerts.yaml: |
    groups:
    - name: cost-alerts
      rules:
      - alert: HighCloudCosts
        expr: increase(cloud_billing_cost_total[1h]) > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High cloud costs detected"
          description: "Cloud costs increased by ${{ $value }} in the last hour"
      
      - alert: UnusedResources
        expr: avg_over_time(node_cpu_usage_percent[24h]) < 20
        for: 1h
        labels:
          severity: info
        annotations:
          summary: "Underutilized resources detected"
          description: "Node {{ $labels.node }} CPU usage below 20% for 24 hours"
```

### 4. Reserved Instances and Savings Plans

#### AWS Reserved Instances
```bash
# Purchase Reserved Instances for predictable workloads
aws ec2 purchase-reserved-instances-offering \
  --reserved-instances-offering-id xxx \
  --instance-count 3

# View recommendations
aws ce get-rightsizing-recommendation \
  --service EC2-Instance \
  --filter file://rightsizing-filter.json
```

#### GCP Committed Use Discounts
```bash
# Purchase committed use discounts
gcloud compute commitments create apm-commitment \
  --resources=vcpu=24,memory=96GB \
  --plan=12-month \
  --region=us-central1
```

#### Azure Reserved VM Instances
```bash
# Purchase Azure Reserved VM Instances
az reservations reservation-order purchase \
  --applied-scope-type Shared \
  --billing-scope-id /subscriptions/your-subscription-id \
  --display-name "APM VM Reservations" \
  --quantity 3 \
  --reserved-resource-type VirtualMachines \
  --sku Standard_D4s_v3 \
  --term P1Y
```

---

## Deployment Checklist

### Pre-Deployment
- [ ] Cloud provider account configured
- [ ] Kubernetes cluster created and accessible
- [ ] Container registry setup
- [ ] DNS records configured
- [ ] SSL certificates obtained
- [ ] Monitoring tools installed

### Deployment
- [ ] Secrets created and configured
- [ ] Database and cache services deployed
- [ ] Application services deployed
- [ ] Ingress/Load balancer configured
- [ ] Health checks passing
- [ ] Monitoring configured

### Post-Deployment
- [ ] Performance testing completed
- [ ] Security scanning passed
- [ ] Backup procedures tested
- [ ] Disaster recovery plan verified
- [ ] Cost optimization implemented
- [ ] Documentation updated

---

**Deployment Time**: 2-4 hours per cloud provider  
**Complexity**: Advanced  
**Cost**: Varies by provider and configuration  
**Next Steps**: See `monitoring-setup.md` for comprehensive monitoring configuration