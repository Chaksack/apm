# S3 Bucket Management for APM Configuration Storage

This document provides comprehensive documentation for the S3 bucket management functionality implemented in the APM tool. The S3Manager provides secure, scalable configuration storage for APM tools including Prometheus, Grafana, Jaeger, Loki, and AlertManager.

## Table of Contents

1. [Overview](#overview)
2. [Features](#features)
3. [Getting Started](#getting-started)
4. [Core Operations](#core-operations)
5. [APM Configuration Management](#apm-configuration-management)
6. [Performance Optimization](#performance-optimization)
7. [Error Handling and Logging](#error-handling-and-logging)
8. [Security Features](#security-features)
9. [Examples](#examples)
10. [Best Practices](#best-practices)
11. [Troubleshooting](#troubleshooting)

## Overview

The S3Manager is a comprehensive solution for managing AWS S3 buckets specifically designed for APM (Application Performance Monitoring) configuration storage. It provides:

- **Secure Configuration Storage**: Encrypted storage with access controls
- **Versioning and Lifecycle Management**: Automatic versioning and cost-effective storage transitions
- **Cross-Region Replication**: Backup and disaster recovery capabilities
- **Performance Optimization**: Caching, connection pooling, and batch processing
- **Comprehensive Monitoring**: Metrics, logging, and health checks

## Features

### Core S3 Operations
- ✅ Bucket creation, listing, and deletion
- ✅ File upload, download, list, and delete operations
- ✅ Multipart upload for large files
- ✅ Copy and move operations between buckets/keys

### Security and Compliance
- ✅ Server-side encryption (SSE-S3, SSE-KMS)
- ✅ Bucket policies and access controls
- ✅ Public access blocking
- ✅ MFA delete protection for production environments

### Lifecycle Management
- ✅ Automatic storage class transitions (Standard → IA → Glacier → Deep Archive)
- ✅ Configurable retention policies
- ✅ Automatic cleanup of incomplete multipart uploads

### Performance Features
- ✅ Intelligent caching with TTL
- ✅ Connection pooling for concurrent operations
- ✅ Batch processing for bulk operations
- ✅ Prefetching and cache warming

### Monitoring and Observability
- ✅ Comprehensive metrics collection
- ✅ Structured logging with multiple levels
- ✅ Health checks and monitoring
- ✅ Error classification and retry mechanisms

## Getting Started

### Prerequisites

1. **AWS CLI Installation**: Ensure AWS CLI v2.x is installed and configured
2. **AWS Credentials**: Configure AWS credentials with appropriate S3 permissions
3. **Go Environment**: Go 1.23+ required

### Installation

```go
import "github.com/chaksack/apm/pkg/cloud"
```

### Basic Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/chaksack/apm/pkg/cloud"
)

func main() {
    // Create AWS provider
    provider, err := cloud.NewAWSProvider(nil)
    if err != nil {
        log.Fatalf("Failed to create AWS provider: %v", err)
    }

    // Get S3 manager
    s3Manager := provider.GetS3Manager()

    // Setup logging and metrics
    logger := cloud.NewS3Logger(provider, true, cloud.LogLevelInfo)
    metrics := cloud.NewS3Metrics()
    
    s3Manager.SetLogger(logger)
    s3Manager.SetMetrics(metrics)

    // Optional: Setup caching for better performance
    cache := cloud.NewS3Cache(5*time.Minute, 1000)
    s3Manager.SetCache(cache)
    defer cache.Stop() // Important: stop cleanup goroutine

    // Now you can use S3 operations
    ctx := context.Background()
    buckets, err := s3Manager.ListBuckets(ctx, "us-east-1")
    if err != nil {
        log.Printf("Error listing buckets: %v", err)
    } else {
        log.Printf("Found %d buckets", len(buckets))
    }
}
```

## Core Operations

### Bucket Operations

#### Create Bucket
```go
bucket, err := s3Manager.CreateBucket(ctx, "my-apm-bucket", "us-east-1", &cloud.BucketOptions{
    Region: "us-east-1",
    Versioning: &cloud.VersioningConfig{
        Status: "Enabled",
    },
    Encryption: &cloud.EncryptionConfig{
        Type:      "SSE-S3",
        Algorithm: "AES256",
    },
    Tags: map[string]string{
        "Environment": "production",
        "Purpose":     "apm-config",
    },
})
```

#### List Buckets
```go
buckets, err := s3Manager.ListBuckets(ctx, "us-east-1")
for _, bucket := range buckets {
    fmt.Printf("Bucket: %s, Created: %s\n", bucket.Name, bucket.CreationDate)
}
```

#### Get Bucket Details
```go
details, err := s3Manager.GetBucket(ctx, "my-apm-bucket", "us-east-1")
if err == nil {
    fmt.Printf("Versioning: %s\n", details.Versioning.Status)
    fmt.Printf("Encryption: %s\n", details.Encryption.Type)
}
```

#### Delete Bucket
```go
err := s3Manager.DeleteBucket(ctx, "my-apm-bucket", "us-east-1", true) // force=true
```

### File Operations

#### Upload File
```go
content := []byte("prometheus configuration content")
fileInfo, err := s3Manager.UploadFile(ctx, "my-apm-bucket", "configs/prometheus.yml", 
    bytes.NewReader(content), &cloud.UploadOptions{
        ContentType: "application/yaml",
        Metadata: map[string]string{
            "config-type": "prometheus",
            "environment": "production",
        },
        ServerSideEncryption: "AES256",
    })
```

#### Download File
```go
reader, err := s3Manager.DownloadFile(ctx, "my-apm-bucket", "configs/prometheus.yml", 
    &cloud.DownloadOptions{})
if err == nil {
    defer reader.Close()
    content, _ := io.ReadAll(reader)
    fmt.Printf("Config content: %s\n", content)
}
```

#### List Files
```go
result, err := s3Manager.ListFiles(ctx, "my-apm-bucket", "configs/", &cloud.ListOptions{
    MaxKeys: 100,
    IncludeMetadata: true,
})
for _, file := range result.Objects {
    fmt.Printf("File: %s, Size: %d, Modified: %s\n", 
        file.Key, file.Size, file.LastModified)
}
```

#### Delete File
```go
err := s3Manager.DeleteFile(ctx, "my-apm-bucket", "configs/old-config.yml", 
    &cloud.DeleteOptions{})
```

## APM Configuration Management

The S3Manager provides specialized methods for managing APM tool configurations:

### Create APM Bucket
```go
bucket, err := s3Manager.CreateAPMBucket(ctx, "apm-production-configs", "us-east-1", 
    "production", "prometheus", "config")
```

This creates a bucket with:
- Secure defaults (encryption, versioning, public access blocking)
- APM-specific lifecycle policies
- Appropriate tagging for organization
- Security policies based on environment

### Upload APM Configuration
```go
prometheusConfig := map[string]interface{}{
    "global": map[string]interface{}{
        "scrape_interval": "15s",
    },
    "scrape_configs": []map[string]interface{}{
        {
            "job_name": "prometheus",
            "static_configs": []map[string]interface{}{
                {"targets": []string{"localhost:9090"}},
            },
        },
    },
}

fileInfo, err := s3Manager.UploadAPMConfig(ctx, "apm-production-configs", 
    "prometheus", "production", prometheusConfig)
```

### Download APM Configuration
```go
config, err := s3Manager.DownloadAPMConfig(ctx, "apm-production-configs", 
    "prometheus", "production")
if err == nil {
    fmt.Printf("Prometheus config: %+v\n", config)
}
```

### Configuration Validation
```go
// Validate before upload
err := s3Manager.ValidateAPMConfig("prometheus", prometheusConfig)
if err != nil {
    log.Printf("Invalid config: %v", err)
    return
}

// Upload only if validation passes
fileInfo, err := s3Manager.UploadAPMConfig(ctx, bucket, "prometheus", "production", prometheusConfig)
```

### Backup and Restore
```go
// Create backup
backup, err := s3Manager.BackupAPMConfig(ctx, "apm-production-configs", 
    "prometheus", "production")

// Restore from backup
restored, err := s3Manager.RestoreAPMConfig(ctx, "apm-production-configs", 
    backup.Key)
```

### Cross-Environment Deployment
```go
// Deploy configs from staging to production
deployed, err := s3Manager.DeployAPMConfigs(ctx, "apm-staging-configs", 
    "staging", "production")
for configType, fileInfo := range deployed {
    fmt.Printf("Deployed %s: %s\n", configType, fileInfo.Key)
}
```

## Performance Optimization

### Caching

```go
// Create cache with 5-minute TTL and 1000 entry limit
cache := cloud.NewS3Cache(5*time.Minute, 1000)
s3Manager.SetCache(cache)
defer cache.Stop()

// Use optimized methods for better performance
buckets, err := s3Manager.OptimizedListBuckets(ctx, "us-east-1")
bucket, err := s3Manager.OptimizedGetBucket(ctx, "my-bucket", "us-east-1")

// Check cache statistics
stats := cache.GetStats()
fmt.Printf("Cache size: %v, Hit rate: %v\n", stats["total_cache_size"], stats["ttl_seconds"])
```

### Connection Pooling

```go
// Create connection pool for concurrent operations
pool := cloud.NewS3ConnectionPool(10) // Max 10 concurrent operations

// Use in your operations
pool.Acquire()
defer pool.Release()
// ... perform S3 operation
```

### Batch Processing

```go
// Create batch processor
batchProcessor := cloud.NewS3BatchProcessor(s3Manager, 10, 5, 30*time.Second)

// Prepare batch operations
operations := []*cloud.BatchOperation{
    {
        Type:    "upload",
        Bucket:  "my-bucket",
        Key:     "config1.yml",
        Content: []byte("config content 1"),
        Options: map[string]interface{}{
            "content_type": "application/yaml",
        },
    },
    {
        Type:   "download",
        Bucket: "my-bucket",
        Key:    "config2.yml",
    },
}

// Process batch
results, err := batchProcessor.ProcessBatch(ctx, operations)
for _, result := range results {
    fmt.Printf("Operation %s: Success=%v, Duration=%v\n", 
        result.Operation.Type, result.Success, result.Duration)
}
```

### Cache Warming

```go
// Prefetch frequently accessed buckets
err := s3Manager.PrefetchBuckets(ctx, []string{"bucket1", "bucket2"}, "us-east-1")

// Warm up cache with configuration
warmupConfig := &cloud.CacheWarmupConfig{
    Region:  "us-east-1",
    Buckets: []string{"apm-prod-configs", "apm-staging-configs"},
    FrequentFiles: map[string][]string{
        "apm-prod-configs": {"configs/prometheus.yml", "configs/grafana.json"},
    },
}
err := s3Manager.WarmupCache(ctx, warmupConfig)
```

## Error Handling and Logging

### Structured Logging

```go
// Create logger with different levels
logger := cloud.NewS3Logger(provider, true, cloud.LogLevelInfo)

// Log levels: Debug, Info, Warn, Error, Fatal
logger.Log(cloud.LogLevelInfo, "UploadFile", "Uploading configuration", &cloud.S3OperationContext{
    Operation: "UploadFile",
    Bucket:    "my-bucket",
    Key:       "config.yml",
    StartTime: time.Now(),
    Success:   true,
})
```

### Error Classification and Retry

```go
// Automatic retry with exponential backoff
err := cloud.RetryS3Operation(func() error {
    _, err := s3Manager.UploadFile(ctx, bucket, key, content, options)
    return err
}, 3, 100*time.Millisecond, "UploadFile")

// Check error types
if cloudErr, ok := err.(*cloud.CloudError); ok {
    fmt.Printf("Error code: %s, Retryable: %v\n", cloudErr.Code, cloudErr.Retryable)
}
```

### Health Checks

```go
// Create health checker
healthChecker := cloud.NewS3HealthChecker(s3Manager, logger)

// Perform health check
result := healthChecker.CheckS3Health(ctx, "health-check-bucket", "us-east-1")
fmt.Printf("Status: %s, Response Time: %v\n", result.Status, result.ResponseTime)

// Monitor operations
metrics := s3Manager.GetMetrics()
operationResult := healthChecker.MonitorS3Operations(ctx, metrics)
```

### Metrics Collection

```go
// Get comprehensive metrics
currentMetrics := metrics.GetMetrics()
fmt.Printf("Total Operations: %d\n", currentMetrics.TotalOperations)
fmt.Printf("Success Rate: %.2f%%\n", 
    float64(currentMetrics.SuccessfulOps)/float64(currentMetrics.TotalOperations)*100)
fmt.Printf("Average Response Time: %v\n", currentMetrics.AverageResponseTime)

// Reset metrics if needed
metrics.ResetMetrics()
```

## Security Features

### Encryption Configuration

```go
// SSE-S3 encryption (default)
encryptionConfig := &cloud.EncryptionConfig{
    Type:             "SSE-S3",
    Algorithm:        "AES256",
    BucketKeyEnabled: true,
}

// SSE-KMS encryption
kmsConfig := &cloud.EncryptionConfig{
    Type:      "SSE-KMS",
    KMSKeyId:  "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
    Algorithm: "aws:kms",
}
```

### Access Control

```go
// Create bucket with strict access controls
options := &cloud.BucketOptions{
    PublicAccessBlock: &cloud.PublicAccessBlockConfig{
        BlockPublicAcls:       true,
        IgnorePublicAcls:      true,
        BlockPublicPolicy:     true,
        RestrictPublicBuckets: true,
    },
    Policy: createRestrictiveBucketPolicy(),
}
```

### MFA Delete Protection

```go
// Delete file with MFA (production environments)
err := s3Manager.DeleteFile(ctx, bucket, key, &cloud.DeleteOptions{
    MFA: "arn:aws:iam::123456789012:mfa/user 123456", // MFA token
})
```

## Examples

### Complete APM Setup Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/chaksack/apm/pkg/cloud"
)

func main() {
    ctx := context.Background()
    
    // 1. Setup AWS provider and S3 manager
    provider, err := cloud.NewAWSProvider(nil)
    if err != nil {
        log.Fatalf("Failed to create AWS provider: %v", err)
    }
    
    s3Manager := provider.GetS3Manager()
    
    // 2. Setup monitoring and caching
    logger := cloud.NewS3Logger(provider, true, cloud.LogLevelInfo)
    metrics := cloud.NewS3Metrics()
    cache := cloud.NewS3Cache(5*time.Minute, 1000)
    
    s3Manager.SetLogger(logger)
    s3Manager.SetMetrics(metrics)
    s3Manager.SetCache(cache)
    defer cache.Stop()
    
    // 3. Create APM bucket for production
    bucket, err := s3Manager.CreateAPMBucket(ctx, "apm-production-configs", 
        "us-east-1", "production", "all", "config")
    if err != nil {
        log.Printf("Bucket may already exist: %v", err)
    } else {
        log.Printf("Created bucket: %s", bucket.Name)
    }
    
    // 4. Upload APM configurations
    configs := map[string]map[string]interface{}{
        "prometheus": {
            "global": map[string]interface{}{
                "scrape_interval": "15s",
            },
            "scrape_configs": []map[string]interface{}{
                {
                    "job_name": "prometheus",
                    "static_configs": []map[string]interface{}{
                        {"targets": []string{"localhost:9090"}},
                    },
                },
            },
        },
        "grafana": {
            "server": map[string]interface{}{
                "http_port": 3000,
            },
            "database": map[string]interface{}{
                "type": "sqlite3",
            },
        },
    }
    
    for configType, config := range configs {
        // Validate configuration
        if err := s3Manager.ValidateAPMConfig(configType, config); err != nil {
            log.Printf("Invalid %s config: %v", configType, err)
            continue
        }
        
        // Upload configuration
        fileInfo, err := s3Manager.UploadAPMConfig(ctx, "apm-production-configs", 
            configType, "production", config)
        if err != nil {
            log.Printf("Failed to upload %s config: %v", configType, err)
        } else {
            log.Printf("Uploaded %s config: %s", configType, fileInfo.Key)
        }
        
        // Create backup
        backup, err := s3Manager.BackupAPMConfig(ctx, "apm-production-configs", 
            configType, "production")
        if err != nil {
            log.Printf("Failed to backup %s config: %v", configType, err)
        } else {
            log.Printf("Created backup: %s", backup.Key)
        }
    }
    
    // 5. Health check
    healthChecker := cloud.NewS3HealthChecker(s3Manager, logger)
    healthResult := healthChecker.CheckS3Health(ctx, "apm-production-configs", "us-east-1")
    log.Printf("Health check status: %s", healthResult.Status)
    
    // 6. Show metrics
    currentMetrics := metrics.GetMetrics()
    log.Printf("Total operations: %d", currentMetrics.TotalOperations)
    log.Printf("Success rate: %.2f%%", 
        float64(currentMetrics.SuccessfulOps)/float64(currentMetrics.TotalOperations)*100)
}
```

## Best Practices

### 1. Bucket Naming
- Use descriptive names: `apm-{environment}-{purpose}`
- Include environment: `apm-production-configs`
- Follow AWS naming conventions (lowercase, hyphens)

### 2. Security
- Always enable encryption (SSE-S3 minimum, SSE-KMS for sensitive data)
- Enable versioning for configuration files
- Use MFA delete for production environments
- Block public access by default
- Implement least-privilege IAM policies

### 3. Cost Optimization
- Configure lifecycle policies for automatic storage class transitions
- Set up automatic cleanup of incomplete multipart uploads
- Use compression for large configuration files
- Monitor storage costs with tags

### 4. Performance
- Enable caching for frequently accessed configurations
- Use batch operations for bulk uploads/downloads
- Implement connection pooling for concurrent operations
- Prefetch commonly used configurations

### 5. Monitoring
- Enable comprehensive logging
- Monitor error rates and response times
- Set up health checks
- Use structured logging for better searchability

### 6. Backup and Recovery
- Implement cross-region replication for critical configurations
- Regular backup of configurations
- Test restore procedures
- Document recovery processes

## Troubleshooting

### Common Issues

#### 1. Authentication Errors
```
Error: Access denied (CODE: ACCESS_DENIED)
```
**Solution**: 
- Check AWS credentials: `aws configure list`
- Verify IAM permissions for S3 operations
- Ensure AWS CLI is properly configured

#### 2. Bucket Already Exists
```
Error: Bucket already exists (CODE: BUCKET_ALREADY_EXISTS)
```
**Solution**:
- Bucket names are globally unique across AWS
- Use a more specific name or check existing buckets
- Consider using `GetBucket` to check if you already own it

#### 3. Large File Upload Failures
```
Error: Request timeout (CODE: REQUEST_TIMEOUT)
```
**Solution**:
- Use multipart upload for files > 100MB
- Increase timeout values
- Check network connectivity
- Consider batch processing

#### 4. High Error Rates
```
Error: Service unavailable (CODE: SERVICE_UNAVAILABLE)
```
**Solution**:
- Implement exponential backoff
- Check AWS service status
- Reduce request rate
- Use connection pooling

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
logger := cloud.NewS3Logger(provider, true, cloud.LogLevelDebug)
s3Manager.SetLogger(logger)
```

### Performance Issues

If experiencing slow operations:

1. **Enable caching**:
```go
cache := cloud.NewS3Cache(5*time.Minute, 1000)
s3Manager.SetCache(cache)
```

2. **Use batch operations**:
```go
batchProcessor := cloud.NewS3BatchProcessor(s3Manager, 10, 5, 30*time.Second)
```

3. **Monitor metrics**:
```go
currentMetrics := metrics.GetMetrics()
if currentMetrics.AverageResponseTime > 5*time.Second {
    log.Println("High response times detected")
}
```

### Error Recovery

Implement robust error recovery:

```go
err := cloud.RetryS3Operation(func() error {
    return s3Manager.UploadFile(ctx, bucket, key, content, options)
}, 3, 100*time.Millisecond, "UploadFile")

if err != nil {
    // Log error and continue with next operation
    logger.Log(cloud.LogLevelError, "UploadFile", fmt.Sprintf("Failed after retries: %v", err), nil)
}
```

## Support and Resources

- **AWS S3 Documentation**: https://docs.aws.amazon.com/s3/
- **AWS CLI Documentation**: https://docs.aws.amazon.com/cli/
- **Go SDK for AWS**: https://docs.aws.amazon.com/sdk-for-go/

For issues specific to this implementation, check the source code and tests in:
- `/pkg/cloud/aws.go` - Main implementation
- `/examples/aws-cli-detection/` - Examples and tests

---

This documentation covers the comprehensive S3 bucket management functionality for APM configuration storage. The implementation provides enterprise-grade features including security, performance optimization, monitoring, and robust error handling.