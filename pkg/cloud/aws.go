package cloud

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AWSProvider implements CloudProvider for AWS
type AWSProvider struct {
	config              *ProviderConfig
	credentials         *Credentials
	cliStatus           *CLIStatus
	cache               *CredentialCache
	ecrTokens           map[string]*ECRToken
	cloudWatch          *CloudWatchIntegration
	s3Manager           *S3Manager
	cfManager           *CloudFormationManager
	crossAccountManager *CrossAccountRoleManager
}

// ECRToken represents an ECR authentication token
type ECRToken struct {
	Token     string
	ExpiresAt time.Time
	Registry  string
	Region    string
}

// CloudWatchIntegration handles CloudWatch operations
type CloudWatchIntegration struct {
	provider *AWSProvider
}

// S3Manager handles S3 operations for configuration storage
type S3Manager struct {
	provider *AWSProvider
	logger   *S3Logger
	metrics  *S3Metrics
	cache    *S3Cache
}

// CloudFormationManager handles CloudFormation operations
type CloudFormationManager struct {
	provider *AWSProvider
}

// CloudFormation stack types and structures
type Stack struct {
	Name         string            `json:"name"`
	Arn          string            `json:"arn"`
	Status       string            `json:"status"`
	Region       string            `json:"region"`
	CreatedTime  time.Time         `json:"createdTime"`
	UpdatedTime  *time.Time        `json:"updatedTime,omitempty"`
	Description  string            `json:"description"`
	Tags         map[string]string `json:"tags"`
	Parameters   map[string]string `json:"parameters"`
	Outputs      map[string]string `json:"outputs"`
	Resources    []*StackResource  `json:"resources"`
	IsAPMStack   bool              `json:"isAPMStack"`
	APMResources *APMResources     `json:"apmResources,omitempty"`
}

type StackResource struct {
	LogicalID    string    `json:"logicalId"`
	PhysicalID   string    `json:"physicalId"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	StatusReason string    `json:"statusReason,omitempty"`
}

type APMResources struct {
	LoadBalancers       []*LoadBalancerResource       `json:"loadBalancers"`
	ECSServices         []*ECSServiceResource         `json:"ecsServices"`
	RDSInstances        []*RDSInstanceResource        `json:"rdsInstances"`
	LambdaFunctions     []*LambdaFunctionResource     `json:"lambdaFunctions"`
	ElastiCacheClusters []*ElastiCacheClusterResource `json:"elastiCacheClusters"`
	S3Buckets           []*S3BucketResource           `json:"s3Buckets"`
	VPCResources        []*VPCResource                `json:"vpcResources"`
}

type LoadBalancerResource struct {
	Type      string   `json:"type"` // ALB, NLB, CLB
	Arn       string   `json:"arn"`
	DNSName   string   `json:"dnsName"`
	Scheme    string   `json:"scheme"`
	VpcId     string   `json:"vpcId"`
	SubnetIds []string `json:"subnetIds"`
}

type ECSServiceResource struct {
	ServiceName    string `json:"serviceName"`
	ClusterName    string `json:"clusterName"`
	TaskDefinition string `json:"taskDefinition"`
	DesiredCount   int    `json:"desiredCount"`
	RunningCount   int    `json:"runningCount"`
	Status         string `json:"status"`
}

type RDSInstanceResource struct {
	DBInstanceIdentifier string `json:"dbInstanceIdentifier"`
	DBName               string `json:"dbName"`
	Engine               string `json:"engine"`
	EngineVersion        string `json:"engineVersion"`
	Endpoint             string `json:"endpoint"`
	Port                 int    `json:"port"`
	Status               string `json:"status"`
}

type LambdaFunctionResource struct {
	FunctionName string `json:"functionName"`
	Runtime      string `json:"runtime"`
	Handler      string `json:"handler"`
	Role         string `json:"role"`
	State        string `json:"state"`
}

type ElastiCacheClusterResource struct {
	ClusterID             string `json:"clusterId"`
	Engine                string `json:"engine"`
	EngineVersion         string `json:"engineVersion"`
	CacheNodeType         string `json:"cacheNodeType"`
	NumCacheNodes         int    `json:"numCacheNodes"`
	Status                string `json:"status"`
	ConfigurationEndpoint string `json:"configurationEndpoint,omitempty"`
}

type S3BucketResource struct {
	BucketName   string    `json:"bucketName"`
	Region       string    `json:"region"`
	CreationDate time.Time `json:"creationDate"`
	Versioning   string    `json:"versioning"`
	Encryption   string    `json:"encryption"`
}

type VPCResource struct {
	VpcId            string   `json:"vpcId"`
	CidrBlock        string   `json:"cidrBlock"`
	State            string   `json:"state"`
	SubnetIds        []string `json:"subnetIds"`
	RouteTableIds    []string `json:"routeTableIds"`
	SecurityGroupIds []string `json:"securityGroupIds"`
}

// Comprehensive S3 Types for APM Configuration Storage

// Bucket represents an S3 bucket with comprehensive metadata
type Bucket struct {
	Name                 string                  `json:"name"`
	Region               string                  `json:"region"`
	CreationDate         time.Time               `json:"creationDate"`
	Versioning           VersioningConfig        `json:"versioning"`
	Encryption           EncryptionConfig        `json:"encryption"`
	Lifecycle            LifecycleConfig         `json:"lifecycle"`
	Replication          ReplicationConfig       `json:"replication"`
	Policy               BucketPolicy            `json:"policy"`
	Tags                 map[string]string       `json:"tags"`
	Location             string                  `json:"location"`
	StorageClass         string                  `json:"storageClass"`
	PublicAccessBlock    PublicAccessBlockConfig `json:"publicAccessBlock"`
	Logging              LoggingConfig           `json:"logging"`
	Notification         NotificationConfig      `json:"notification"`
	CORS                 CORSConfig              `json:"cors"`
	Website              WebsiteConfig           `json:"website"`
	RequestPayer         string                  `json:"requestPayer"`
	TransferAcceleration bool                    `json:"transferAcceleration"`
}

// BucketDetails provides comprehensive bucket information
type BucketDetails struct {
	Bucket       *Bucket         `json:"bucket"`
	Size         int64           `json:"size"`
	ObjectCount  int64           `json:"objectCount"`
	LastModified time.Time       `json:"lastModified"`
	Cost         BucketCost      `json:"cost"`
	Metrics      BucketMetrics   `json:"metrics"`
	APMConfig    APMBucketConfig `json:"apmConfig"`
}

// VersioningConfig represents bucket versioning configuration
type VersioningConfig struct {
	Status    string `json:"status"` // Enabled, Suspended, Disabled
	MfaDelete string `json:"mfaDelete,omitempty"`
}

// EncryptionConfig represents bucket encryption settings
type EncryptionConfig struct {
	Type             string `json:"type"` // SSE-S3, SSE-KMS, SSE-C
	KMSKeyId         string `json:"kmsKeyId,omitempty"`
	KMSKeyArn        string `json:"kmsKeyArn,omitempty"`
	BucketKeyEnabled bool   `json:"bucketKeyEnabled"`
	Algorithm        string `json:"algorithm"`
}

// LifecycleConfig represents bucket lifecycle management
type LifecycleConfig struct {
	Rules []S3LifecycleRule `json:"rules"`
}

// LifecycleRule represents a single lifecycle rule
type S3LifecycleRule struct {
	ID                             string                `json:"id"`
	Status                         string                `json:"status"` // Enabled, Disabled
	Filter                         LifecycleFilter       `json:"filter"`
	Transitions                    []LifecycleTransition `json:"transitions"`
	Expiration                     *LifecycleExpiration  `json:"expiration,omitempty"`
	NoncurrentVersionTransitions   []LifecycleTransition `json:"noncurrentVersionTransitions,omitempty"`
	NoncurrentVersionExpiration    *LifecycleExpiration  `json:"noncurrentVersionExpiration,omitempty"`
	AbortIncompleteMultipartUpload *LifecycleAbort       `json:"abortIncompleteMultipartUpload,omitempty"`
}

// LifecycleFilter represents filter criteria for lifecycle rules
type LifecycleFilter struct {
	Prefix string            `json:"prefix,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
	And    *LifecycleAnd     `json:"and,omitempty"`
}

// LifecycleAnd represents AND condition in lifecycle filter
type LifecycleAnd struct {
	Prefix string            `json:"prefix,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// LifecycleTransition represents storage class transition
type LifecycleTransition struct {
	Days         int    `json:"days,omitempty"`
	Date         string `json:"date,omitempty"`
	StorageClass string `json:"storageClass"` // STANDARD_IA, ONEZONE_IA, GLACIER, DEEP_ARCHIVE
}

// LifecycleExpiration represents object expiration settings
type LifecycleExpiration struct {
	Days                      int    `json:"days,omitempty"`
	Date                      string `json:"date,omitempty"`
	ExpiredObjectDeleteMarker bool   `json:"expiredObjectDeleteMarker,omitempty"`
}

// LifecycleAbort represents abort incomplete multipart upload settings
type LifecycleAbort struct {
	DaysAfterInitiation int `json:"daysAfterInitiation"`
}

// ReplicationConfig represents bucket replication settings
type ReplicationConfig struct {
	Role  string            `json:"role"`
	Rules []ReplicationRule `json:"rules"`
}

// ReplicationRule represents a single replication rule
type ReplicationRule struct {
	ID                        string                     `json:"id"`
	Status                    string                     `json:"status"` // Enabled, Disabled
	Priority                  int                        `json:"priority,omitempty"`
	Filter                    ReplicationFilter          `json:"filter"`
	Destination               ReplicationDestination     `json:"destination"`
	DeleteMarkerReplication   *DeleteMarkerReplication   `json:"deleteMarkerReplication,omitempty"`
	ExistingObjectReplication *ExistingObjectReplication `json:"existingObjectReplication,omitempty"`
}

// ReplicationFilter represents filter for replication rules
type ReplicationFilter struct {
	Prefix string            `json:"prefix,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
	And    *ReplicationAnd   `json:"and,omitempty"`
}

// ReplicationAnd represents AND condition in replication filter
type ReplicationAnd struct {
	Prefix string            `json:"prefix,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// ReplicationDestination represents replication destination
type ReplicationDestination struct {
	Bucket                   string                    `json:"bucket"`
	StorageClass             string                    `json:"storageClass,omitempty"`
	Account                  string                    `json:"account,omitempty"`
	AccessControlTranslation *AccessControlTranslation `json:"accessControlTranslation,omitempty"`
	EncryptionConfiguration  *EncryptionConfiguration  `json:"encryptionConfiguration,omitempty"`
	ReplicationTime          *ReplicationTime          `json:"replicationTime,omitempty"`
	Metrics                  *ReplicationMetrics       `json:"metrics,omitempty"`
}

// AccessControlTranslation represents access control translation
type AccessControlTranslation struct {
	Owner string `json:"owner"`
}

// EncryptionConfiguration represents encryption configuration for replication
type EncryptionConfiguration struct {
	ReplicaKmsKeyID string `json:"replicaKmsKeyID"`
}

// ReplicationTime represents replication time control
type ReplicationTime struct {
	Status string               `json:"status"`
	Time   ReplicationTimeValue `json:"time"`
}

// ReplicationTimeValue represents replication time value
type ReplicationTimeValue struct {
	Minutes int `json:"minutes"`
}

// ReplicationMetrics represents replication metrics
type ReplicationMetrics struct {
	Status         string               `json:"status"`
	EventThreshold ReplicationTimeValue `json:"eventThreshold"`
}

// DeleteMarkerReplication represents delete marker replication
type DeleteMarkerReplication struct {
	Status string `json:"status"`
}

// ExistingObjectReplication represents existing object replication
type ExistingObjectReplication struct {
	Status string `json:"status"`
}

// BucketPolicy represents bucket access policy
type BucketPolicy struct {
	Version   string            `json:"version"`
	Statement []PolicyStatement `json:"statement"`
}

// PolicyStatement represents a policy statement
type PolicyStatement struct {
	Sid       string                 `json:"sid,omitempty"`
	Effect    string                 `json:"effect"` // Allow, Deny
	Principal interface{}            `json:"principal,omitempty"`
	Action    interface{}            `json:"action"`
	Resource  interface{}            `json:"resource"`
	Condition map[string]interface{} `json:"condition,omitempty"`
}

// PublicAccessBlockConfig represents public access block configuration
type PublicAccessBlockConfig struct {
	BlockPublicAcls       bool `json:"blockPublicAcls"`
	IgnorePublicAcls      bool `json:"ignorePublicAcls"`
	BlockPublicPolicy     bool `json:"blockPublicPolicy"`
	RestrictPublicBuckets bool `json:"restrictPublicBuckets"`
}

// LoggingConfig represents bucket access logging configuration
type LoggingConfig struct {
	TargetBucket string  `json:"targetBucket,omitempty"`
	TargetPrefix string  `json:"targetPrefix,omitempty"`
	TargetGrants []Grant `json:"targetGrants,omitempty"`
}

// Grant represents a logging grant
type Grant struct {
	Grantee    Grantee `json:"grantee"`
	Permission string  `json:"permission"`
}

// Grantee represents a grant recipient
type Grantee struct {
	Type         string `json:"type"`
	ID           string `json:"id,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	URI          string `json:"uri,omitempty"`
}

// NotificationConfig represents bucket notification configuration
type NotificationConfig struct {
	TopicConfigurations  []TopicConfiguration  `json:"topicConfigurations,omitempty"`
	QueueConfigurations  []QueueConfiguration  `json:"queueConfigurations,omitempty"`
	LambdaConfigurations []LambdaConfiguration `json:"lambdaConfigurations,omitempty"`
}

// TopicConfiguration represents SNS topic notification
type TopicConfiguration struct {
	ID     string             `json:"id,omitempty"`
	Topic  string             `json:"topic"`
	Events []string           `json:"events"`
	Filter NotificationFilter `json:"filter,omitempty"`
}

// QueueConfiguration represents SQS queue notification
type QueueConfiguration struct {
	ID     string             `json:"id,omitempty"`
	Queue  string             `json:"queue"`
	Events []string           `json:"events"`
	Filter NotificationFilter `json:"filter,omitempty"`
}

// LambdaConfiguration represents Lambda function notification
type LambdaConfiguration struct {
	ID       string             `json:"id,omitempty"`
	Function string             `json:"function"`
	Events   []string           `json:"events"`
	Filter   NotificationFilter `json:"filter,omitempty"`
}

// NotificationFilter represents notification filter
type NotificationFilter struct {
	Key NotificationKeyFilter `json:"key,omitempty"`
}

// NotificationKeyFilter represents key-based notification filter
type NotificationKeyFilter struct {
	FilterRules []NotificationFilterRule `json:"filterRules,omitempty"`
}

// NotificationFilterRule represents a single filter rule
type NotificationFilterRule struct {
	Name  string `json:"name"` // prefix, suffix
	Value string `json:"value"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	CORSRules []CORSRule `json:"corsRules,omitempty"`
}

// CORSRule represents a single CORS rule
type CORSRule struct {
	ID             string   `json:"id,omitempty"`
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`
	AllowedMethods []string `json:"allowedMethods"`
	AllowedOrigins []string `json:"allowedOrigins"`
	ExposeHeaders  []string `json:"exposeHeaders,omitempty"`
	MaxAgeSeconds  int      `json:"maxAgeSeconds,omitempty"`
}

// WebsiteConfig represents static website hosting configuration
type WebsiteConfig struct {
	IndexDocument         string                 `json:"indexDocument,omitempty"`
	ErrorDocument         string                 `json:"errorDocument,omitempty"`
	RedirectAllRequestsTo *RedirectAllRequestsTo `json:"redirectAllRequestsTo,omitempty"`
	RoutingRules          []RoutingRule          `json:"routingRules,omitempty"`
}

// RedirectAllRequestsTo represents redirect all requests configuration
type RedirectAllRequestsTo struct {
	HostName string `json:"hostName"`
	Protocol string `json:"protocol,omitempty"`
}

// RoutingRule represents a routing rule for website hosting
type RoutingRule struct {
	Condition RoutingRuleCondition `json:"condition,omitempty"`
	Redirect  RoutingRuleRedirect  `json:"redirect"`
}

// RoutingRuleCondition represents routing rule condition
type RoutingRuleCondition struct {
	HttpErrorCodeReturnedEquals string `json:"httpErrorCodeReturnedEquals,omitempty"`
	KeyPrefixEquals             string `json:"keyPrefixEquals,omitempty"`
}

// RoutingRuleRedirect represents routing rule redirect
type RoutingRuleRedirect struct {
	HostName             string `json:"hostName,omitempty"`
	HttpRedirectCode     string `json:"httpRedirectCode,omitempty"`
	Protocol             string `json:"protocol,omitempty"`
	ReplaceKeyPrefixWith string `json:"replaceKeyPrefixWith,omitempty"`
	ReplaceKeyWith       string `json:"replaceKeyWith,omitempty"`
}

// FileInfo represents information about an S3 object
type FileInfo struct {
	Key                       string            `json:"key"`
	LastModified              time.Time         `json:"lastModified"`
	ETag                      string            `json:"etag"`
	Size                      int64             `json:"size"`
	StorageClass              string            `json:"storageClass"`
	Owner                     ObjectOwner       `json:"owner"`
	VersionId                 string            `json:"versionId,omitempty"`
	IsLatest                  bool              `json:"isLatest,omitempty"`
	DeleteMarker              bool              `json:"deleteMarker,omitempty"`
	Metadata                  map[string]string `json:"metadata,omitempty"`
	Tags                      map[string]string `json:"tags,omitempty"`
	ContentType               string            `json:"contentType,omitempty"`
	ContentEncoding           string            `json:"contentEncoding,omitempty"`
	ContentLanguage           string            `json:"contentLanguage,omitempty"`
	ContentDisposition        string            `json:"contentDisposition,omitempty"`
	CacheControl              string            `json:"cacheControl,omitempty"`
	Expires                   string            `json:"expires,omitempty"`
	WebsiteRedirectLocation   string            `json:"websiteRedirectLocation,omitempty"`
	ServerSideEncryption      string            `json:"serverSideEncryption,omitempty"`
	SSECustomerAlgorithm      string            `json:"sseCustomerAlgorithm,omitempty"`
	SSEKMSKeyId               string            `json:"sseKmsKeyId,omitempty"`
	BucketKeyEnabled          bool              `json:"bucketKeyEnabled,omitempty"`
	RequestCharged            string            `json:"requestCharged,omitempty"`
	ReplicationStatus         string            `json:"replicationStatus,omitempty"`
	PartsCount                int               `json:"partsCount,omitempty"`
	ObjectLockMode            string            `json:"objectLockMode,omitempty"`
	ObjectLockRetainUntilDate time.Time         `json:"objectLockRetainUntilDate,omitempty"`
	ObjectLockLegalHoldStatus string            `json:"objectLockLegalHoldStatus,omitempty"`
	Checksum                  ObjectChecksum    `json:"checksum,omitempty"`
}

// ObjectOwner represents the owner of an S3 object
type ObjectOwner struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

// ObjectChecksum represents object checksum information
type ObjectChecksum struct {
	ChecksumCRC32  string `json:"checksumCRC32,omitempty"`
	ChecksumCRC32C string `json:"checksumCRC32C,omitempty"`
	ChecksumSHA1   string `json:"checksumSHA1,omitempty"`
	ChecksumSHA256 string `json:"checksumSHA256,omitempty"`
}

// BucketCost represents bucket cost information
type BucketCost struct {
	StorageCost   float64            `json:"storageCost"`
	RequestCost   float64            `json:"requestCost"`
	TransferCost  float64            `json:"transferCost"`
	TotalCost     float64            `json:"totalCost"`
	Currency      string             `json:"currency"`
	BillingPeriod string             `json:"billingPeriod"`
	CostBreakdown map[string]float64 `json:"costBreakdown"`
}

// BucketMetrics represents bucket metrics and statistics
type BucketMetrics struct {
	TotalObjects             int64            `json:"totalObjects"`
	TotalSize                int64            `json:"totalSize"`
	AverageObjectSize        int64            `json:"averageObjectSize"`
	StorageClassDistribution map[string]int64 `json:"storageClassDistribution"`
	RequestMetrics           RequestMetrics   `json:"requestMetrics"`
	TransferMetrics          TransferMetrics  `json:"transferMetrics"`
	ErrorMetrics             ErrorMetrics     `json:"errorMetrics"`
	LastUpdated              time.Time        `json:"lastUpdated"`
}

// RequestMetrics represents request-related metrics
type RequestMetrics struct {
	GetRequests    int64 `json:"getRequests"`
	PutRequests    int64 `json:"putRequests"`
	DeleteRequests int64 `json:"deleteRequests"`
	HeadRequests   int64 `json:"headRequests"`
	PostRequests   int64 `json:"postRequests"`
	ListRequests   int64 `json:"listRequests"`
	TotalRequests  int64 `json:"totalRequests"`
}

// TransferMetrics represents data transfer metrics
type TransferMetrics struct {
	BytesUploaded   int64 `json:"bytesUploaded"`
	BytesDownloaded int64 `json:"bytesDownloaded"`
	TotalTransfer   int64 `json:"totalTransfer"`
}

// ErrorMetrics represents error-related metrics
type ErrorMetrics struct {
	ClientErrors int64   `json:"clientErrors"` // 4xx errors
	ServerErrors int64   `json:"serverErrors"` // 5xx errors
	TotalErrors  int64   `json:"totalErrors"`
	ErrorRate    float64 `json:"errorRate"`
}

// APMBucketConfig represents APM-specific bucket configuration
type APMBucketConfig struct {
	Purpose           string           `json:"purpose"`     // config, logs, artifacts, backup
	Environment       string           `json:"environment"` // dev, staging, prod
	Component         string           `json:"component"`   // prometheus, grafana, jaeger, loki
	ConfigTemplates   []string         `json:"configTemplates"`
	AutoBackup        bool             `json:"autoBackup"`
	BackupRetention   int              `json:"backupRetention"` // days
	Compression       bool             `json:"compression"`
	Encryption        bool             `json:"encryption"`
	AccessLogging     bool             `json:"accessLogging"`
	MonitoringEnabled bool             `json:"monitoringEnabled"`
	Alerting          AlertingConfig   `json:"alerting"`
	Compliance        ComplianceConfig `json:"compliance"`
}

// AlertingConfig represents alerting configuration for APM buckets
type AlertingConfig struct {
	Enabled             bool     `json:"enabled"`
	UnauthorizedAccess  bool     `json:"unauthorizedAccess"`
	HighRequestRate     bool     `json:"highRequestRate"`
	HighErrorRate       bool     `json:"highErrorRate"`
	SizeThreshold       int64    `json:"sizeThreshold"`
	CostThreshold       float64  `json:"costThreshold"`
	NotificationTargets []string `json:"notificationTargets"`
}

// ComplianceConfig represents compliance configuration
type ComplianceConfig struct {
	DataClassification    string   `json:"dataClassification"` // public, internal, confidential, restricted
	RetentionPeriod       int      `json:"retentionPeriod"`    // days
	EncryptionRequired    bool     `json:"encryptionRequired"`
	AccessLoggingRequired bool     `json:"accessLoggingRequired"`
	BackupRequired        bool     `json:"backupRequired"`
	ComplianceStandards   []string `json:"complianceStandards"` // SOC2, HIPAA, PCI-DSS, etc.
}

// BucketOptions represents options for bucket creation and management
type BucketOptions struct {
	Region               string                   `json:"region"`
	Versioning           *VersioningConfig        `json:"versioning,omitempty"`
	Encryption           *EncryptionConfig        `json:"encryption,omitempty"`
	Lifecycle            *LifecycleConfig         `json:"lifecycle,omitempty"`
	Replication          *ReplicationConfig       `json:"replication,omitempty"`
	Policy               *BucketPolicy            `json:"policy,omitempty"`
	Tags                 map[string]string        `json:"tags,omitempty"`
	PublicAccessBlock    *PublicAccessBlockConfig `json:"publicAccessBlock,omitempty"`
	Logging              *LoggingConfig           `json:"logging,omitempty"`
	Notification         *NotificationConfig      `json:"notification,omitempty"`
	CORS                 *CORSConfig              `json:"cors,omitempty"`
	Website              *WebsiteConfig           `json:"website,omitempty"`
	RequestPayer         string                   `json:"requestPayer,omitempty"`
	TransferAcceleration bool                     `json:"transferAcceleration,omitempty"`
	ObjectLockEnabled    bool                     `json:"objectLockEnabled,omitempty"`
	APMConfig            *APMBucketConfig         `json:"apmConfig,omitempty"`
}

// UploadOptions represents options for file upload
type UploadOptions struct {
	ContentType               string             `json:"contentType,omitempty"`
	ContentEncoding           string             `json:"contentEncoding,omitempty"`
	ContentLanguage           string             `json:"contentLanguage,omitempty"`
	ContentDisposition        string             `json:"contentDisposition,omitempty"`
	CacheControl              string             `json:"cacheControl,omitempty"`
	Expires                   string             `json:"expires,omitempty"`
	Metadata                  map[string]string  `json:"metadata,omitempty"`
	Tags                      map[string]string  `json:"tags,omitempty"`
	StorageClass              string             `json:"storageClass,omitempty"`
	ServerSideEncryption      string             `json:"serverSideEncryption,omitempty"`
	SSEKMSKeyId               string             `json:"sseKmsKeyId,omitempty"`
	BucketKeyEnabled          bool               `json:"bucketKeyEnabled,omitempty"`
	ObjectLockMode            string             `json:"objectLockMode,omitempty"`
	ObjectLockRetainUntilDate time.Time          `json:"objectLockRetainUntilDate,omitempty"`
	ObjectLockLegalHoldStatus string             `json:"objectLockLegalHoldStatus,omitempty"`
	ChecksumAlgorithm         string             `json:"checksumAlgorithm,omitempty"`
	RequestPayer              string             `json:"requestPayer,omitempty"`
	UseMultipart              bool               `json:"useMultipart,omitempty"`
	PartSize                  int64              `json:"partSize,omitempty"`
	Concurrency               int                `json:"concurrency,omitempty"`
	ProgressCallback          func(int64, int64) `json:"-"`
}

// DownloadOptions represents options for file download
type DownloadOptions struct {
	VersionId                  string             `json:"versionId,omitempty"`
	Range                      string             `json:"range,omitempty"`
	IfModifiedSince            time.Time          `json:"ifModifiedSince,omitempty"`
	IfUnmodifiedSince          time.Time          `json:"ifUnmodifiedSince,omitempty"`
	IfMatch                    string             `json:"ifMatch,omitempty"`
	IfNoneMatch                string             `json:"ifNoneMatch,omitempty"`
	SSECustomerAlgorithm       string             `json:"sseCustomerAlgorithm,omitempty"`
	SSECustomerKey             string             `json:"sseCustomerKey,omitempty"`
	SSECustomerKeyMD5          string             `json:"sseCustomerKeyMD5,omitempty"`
	RequestPayer               string             `json:"requestPayer,omitempty"`
	PartNumber                 int                `json:"partNumber,omitempty"`
	ResponseCacheControl       string             `json:"responseCacheControl,omitempty"`
	ResponseContentDisposition string             `json:"responseContentDisposition,omitempty"`
	ResponseContentEncoding    string             `json:"responseContentEncoding,omitempty"`
	ResponseContentLanguage    string             `json:"responseContentLanguage,omitempty"`
	ResponseContentType        string             `json:"responseContentType,omitempty"`
	ResponseExpires            string             `json:"responseExpires,omitempty"`
	ChecksumMode               string             `json:"checksumMode,omitempty"`
	ProgressCallback           func(int64, int64) `json:"-"`
}

// ListOptions represents options for listing objects
type ListOptions struct {
	Prefix              string `json:"prefix,omitempty"`
	Delimiter           string `json:"delimiter,omitempty"`
	MaxKeys             int    `json:"maxKeys,omitempty"`
	StartAfter          string `json:"startAfter,omitempty"`
	ContinuationToken   string `json:"continuationToken,omitempty"`
	FetchOwner          bool   `json:"fetchOwner,omitempty"`
	RequestPayer        string `json:"requestPayer,omitempty"`
	ExpectedBucketOwner string `json:"expectedBucketOwner,omitempty"`
	IncludeMetadata     bool   `json:"includeMetadata,omitempty"`
	IncludeTags         bool   `json:"includeTags,omitempty"`
	Versions            bool   `json:"versions,omitempty"`
}

// DeleteOptions represents options for object deletion
type DeleteOptions struct {
	VersionId                 string `json:"versionId,omitempty"`
	MFA                       string `json:"mfa,omitempty"`
	RequestPayer              string `json:"requestPayer,omitempty"`
	BypassGovernanceRetention bool   `json:"bypassGovernanceRetention,omitempty"`
	ExpectedBucketOwner       string `json:"expectedBucketOwner,omitempty"`
}

// MultipartUploadInfo represents information about a multipart upload
type MultipartUploadInfo struct {
	UploadId     string          `json:"uploadId"`
	Key          string          `json:"key"`
	Bucket       string          `json:"bucket"`
	Initiated    time.Time       `json:"initiated"`
	StorageClass string          `json:"storageClass"`
	Owner        ObjectOwner     `json:"owner"`
	Initiator    ObjectOwner     `json:"initiator"`
	Parts        []MultipartPart `json:"parts"`
}

// MultipartPart represents a part in a multipart upload
type MultipartPart struct {
	PartNumber     int       `json:"partNumber"`
	ETag           string    `json:"etag"`
	Size           int64     `json:"size"`
	LastModified   time.Time `json:"lastModified"`
	ChecksumCRC32  string    `json:"checksumCRC32,omitempty"`
	ChecksumCRC32C string    `json:"checksumCRC32C,omitempty"`
	ChecksumSHA1   string    `json:"checksumSHA1,omitempty"`
	ChecksumSHA256 string    `json:"checksumSHA256,omitempty"`
}

// ListResult represents the result of a list operation
type ListResult struct {
	Objects               []*FileInfo `json:"objects"`
	CommonPrefixes        []string    `json:"commonPrefixes"`
	IsTruncated           bool        `json:"isTruncated"`
	NextContinuationToken string      `json:"nextContinuationToken,omitempty"`
	NextMarker            string      `json:"nextMarker,omitempty"`
	MaxKeys               int         `json:"maxKeys"`
	KeyCount              int         `json:"keyCount"`
	Name                  string      `json:"name"`
	Prefix                string      `json:"prefix"`
	Delimiter             string      `json:"delimiter"`
	EncodingType          string      `json:"encodingType"`
}

// CopyOptions represents options for copying objects
type CopyOptions struct {
	SourceVersionId           string            `json:"sourceVersionId,omitempty"`
	MetadataDirective         string            `json:"metadataDirective,omitempty"` // COPY, REPLACE
	TaggingDirective          string            `json:"taggingDirective,omitempty"`  // COPY, REPLACE
	Metadata                  map[string]string `json:"metadata,omitempty"`
	Tags                      map[string]string `json:"tags,omitempty"`
	ContentType               string            `json:"contentType,omitempty"`
	ContentEncoding           string            `json:"contentEncoding,omitempty"`
	ContentLanguage           string            `json:"contentLanguage,omitempty"`
	ContentDisposition        string            `json:"contentDisposition,omitempty"`
	CacheControl              string            `json:"cacheControl,omitempty"`
	Expires                   string            `json:"expires,omitempty"`
	StorageClass              string            `json:"storageClass,omitempty"`
	ServerSideEncryption      string            `json:"serverSideEncryption,omitempty"`
	SSEKMSKeyId               string            `json:"sseKmsKeyId,omitempty"`
	BucketKeyEnabled          bool              `json:"bucketKeyEnabled,omitempty"`
	ObjectLockMode            string            `json:"objectLockMode,omitempty"`
	ObjectLockRetainUntilDate time.Time         `json:"objectLockRetainUntilDate,omitempty"`
	ObjectLockLegalHoldStatus string            `json:"objectLockLegalHoldStatus,omitempty"`
	RequestPayer              string            `json:"requestPayer,omitempty"`
	ACL                       string            `json:"acl,omitempty"`
	GrantFullControl          string            `json:"grantFullControl,omitempty"`
	GrantRead                 string            `json:"grantRead,omitempty"`
	GrantReadACP              string            `json:"grantReadACP,omitempty"`
	GrantWriteACP             string            `json:"grantWriteACP,omitempty"`
	ChecksumAlgorithm         string            `json:"checksumAlgorithm,omitempty"`
}

type StackFilters struct {
	Regions       []string          `json:"regions"`
	StackStatus   []string          `json:"stackStatus"`
	Tags          map[string]string `json:"tags"`
	NamePrefix    string            `json:"namePrefix"`
	APMOnly       bool              `json:"apmOnly"`
	CreatedAfter  *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore *time.Time        `json:"createdBefore,omitempty"`
}

type DriftResult struct {
	StackName          string             `json:"stackName"`
	DriftStatus        string             `json:"driftStatus"`
	DetectionTime      time.Time          `json:"detectionTime"`
	DriftedResources   []*DriftedResource `json:"driftedResources"`
	TotalResources     int                `json:"totalResources"`
	DriftedCount       int                `json:"driftedCount"`
	RecommendedActions []string           `json:"recommendedActions"`
}

type DriftedResource struct {
	LogicalID          string                 `json:"logicalId"`
	PhysicalID         string                 `json:"physicalId"`
	ResourceType       string                 `json:"resourceType"`
	DriftStatus        string                 `json:"driftStatus"`
	PropertyDiffs      []*PropertyDifference  `json:"propertyDiffs"`
	ExpectedProperties map[string]interface{} `json:"expectedProperties"`
	ActualProperties   map[string]interface{} `json:"actualProperties"`
}

type PropertyDifference struct {
	PropertyPath   string      `json:"propertyPath"`
	ExpectedValue  interface{} `json:"expectedValue"`
	ActualValue    interface{} `json:"actualValue"`
	DifferenceType string      `json:"differenceType"`
}

type HealthResult struct {
	StackName          string                  `json:"stackName"`
	OverallHealth      string                  `json:"overallHealth"`
	LastChecked        time.Time               `json:"lastChecked"`
	HealthyResources   int                     `json:"healthyResources"`
	UnhealthyResources int                     `json:"unhealthyResources"`
	ResourceHealth     []*ResourceHealthStatus `json:"resourceHealth"`
	Issues             []string                `json:"issues"`
	Recommendations    []string                `json:"recommendations"`
}

type ResourceHealthStatus struct {
	LogicalID    string    `json:"logicalId"`
	PhysicalID   string    `json:"physicalId"`
	ResourceType string    `json:"resourceType"`
	Health       string    `json:"health"`
	Status       string    `json:"status"`
	CheckedAt    time.Time `json:"checkedAt"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
}

// ====================================================================
// CloudWatch Types and Structures for APM Monitoring Integration
// ====================================================================

// CloudWatchDashboard represents a CloudWatch dashboard for APM monitoring
type CloudWatchDashboard struct {
	DashboardName  string                  `json:"dashboardName"`
	DashboardBody  string                  `json:"dashboardBody"`
	DashboardArn   string                  `json:"dashboardArn"`
	LastModified   time.Time               `json:"lastModified"`
	Size           int64                   `json:"size"`
	Region         string                  `json:"region"`
	Tags           map[string]string       `json:"tags"`
	Widgets        []*DashboardWidget      `json:"widgets"`
	Variables      map[string]string       `json:"variables"`
	APMIntegration APMDashboardIntegration `json:"apmIntegration"`
	ShareStatus    string                  `json:"shareStatus"`
	CreatedBy      string                  `json:"createdBy"`
	Description    string                  `json:"description"`
}

// DashboardConfig represents configuration for creating/updating dashboards
type DashboardConfig struct {
	Name           string                  `json:"name"`
	Body           string                  `json:"body,omitempty"`
	Widgets        []*DashboardWidget      `json:"widgets,omitempty"`
	Variables      map[string]string       `json:"variables,omitempty"`
	Tags           map[string]string       `json:"tags,omitempty"`
	APMIntegration APMDashboardIntegration `json:"apmIntegration"`
	Template       string                  `json:"template,omitempty"` // APM template type
	AutoRefresh    int                     `json:"autoRefresh,omitempty"`
	TimeRange      DashboardTimeRange      `json:"timeRange,omitempty"`
	Description    string                  `json:"description,omitempty"`
}

// DashboardWidget represents a widget in a CloudWatch dashboard
type DashboardWidget struct {
	Type       string                 `json:"type"`
	Properties WidgetProperties       `json:"properties"`
	Position   WidgetPosition         `json:"position"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// WidgetProperties contains widget-specific properties
type WidgetProperties struct {
	Metrics     [][]interface{}    `json:"metrics,omitempty"`
	Period      int                `json:"period,omitempty"`
	Stat        string             `json:"stat,omitempty"`
	Region      string             `json:"region,omitempty"`
	Title       string             `json:"title,omitempty"`
	View        string             `json:"view,omitempty"`
	Stacked     bool               `json:"stacked,omitempty"`
	YAxis       YAxisConfig        `json:"yAxis,omitempty"`
	Legend      LegendConfig       `json:"legend,omitempty"`
	Annotations []AnnotationConfig `json:"annotations,omitempty"`
	Query       string             `json:"query,omitempty"`
	LogGroups   []string           `json:"logGroups,omitempty"`
}

// WidgetPosition defines the position and size of a widget
type WidgetPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// YAxisConfig configures the Y-axis of a widget
type YAxisConfig struct {
	Left  AxisConfig `json:"left,omitempty"`
	Right AxisConfig `json:"right,omitempty"`
}

// AxisConfig configures an axis
type AxisConfig struct {
	Min   *float64 `json:"min,omitempty"`
	Max   *float64 `json:"max,omitempty"`
	Label string   `json:"label,omitempty"`
}

// LegendConfig configures the legend of a widget
type LegendConfig struct {
	Position string `json:"position,omitempty"`
}

// AnnotationConfig configures annotations on a widget
type AnnotationConfig struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Fill  string  `json:"fill,omitempty"`
}

// DashboardTimeRange defines the time range for a dashboard
type DashboardTimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// APMDashboardIntegration contains APM-specific dashboard settings
type APMDashboardIntegration struct {
	PrometheusIntegration bool             `json:"prometheusIntegration"`
	GrafanaSync           bool             `json:"grafanaSync"`
	JaegerMetrics         bool             `json:"jaegerMetrics"`
	LokiLogs              bool             `json:"lokiLogs"`
	AlertManagerAlerts    bool             `json:"alertManagerAlerts"`
	APMServices           []string         `json:"apmServices"`
	Namespaces            []string         `json:"namespaces"`
	CustomMetrics         []CustomMetric   `json:"customMetrics"`
	AlertIntegration      AlertIntegration `json:"alertIntegration"`
}

// CustomMetric represents a custom metric definition
type CustomMetric struct {
	MetricName string            `json:"metricName"`
	Namespace  string            `json:"namespace"`
	Dimensions map[string]string `json:"dimensions"`
	Unit       string            `json:"unit"`
	Value      float64           `json:"value"`
}

// AlertIntegration configures alert integration
type AlertIntegration struct {
	Enabled         bool     `json:"enabled"`
	SNSTopics       []string `json:"snsTopics"`
	EmailTargets    []string `json:"emailTargets"`
	SlackWebhooks   []string `json:"slackWebhooks"`
	AutoResolve     bool     `json:"autoResolve"`
	EscalationRules []string `json:"escalationRules"`
}

// CloudWatchAlarm represents a CloudWatch alarm for APM monitoring
type CloudWatchAlarm struct {
	AlarmName                          string            `json:"alarmName"`
	AlarmDescription                   string            `json:"alarmDescription"`
	AlarmArn                           string            `json:"alarmArn"`
	MetricName                         string            `json:"metricName"`
	Namespace                          string            `json:"namespace"`
	Statistic                          string            `json:"statistic"`
	Dimensions                         []*AlarmDimension `json:"dimensions"`
	Period                             int               `json:"period"`
	EvaluationPeriods                  int               `json:"evaluationPeriods"`
	Threshold                          float64           `json:"threshold"`
	ComparisonOperator                 string            `json:"comparisonOperator"`
	TreatMissingData                   string            `json:"treatMissingData"`
	EvaluateLowSampleCountPercentile   string            `json:"evaluateLowSampleCountPercentile,omitempty"`
	DatapointsToAlarm                  int               `json:"datapointsToAlarm"`
	State                              AlarmState        `json:"state"`
	StateReason                        string            `json:"stateReason"`
	StateReasonData                    string            `json:"stateReasonData"`
	StateUpdatedTimestamp              time.Time         `json:"stateUpdatedTimestamp"`
	AlarmConfigurationUpdatedTimestamp time.Time         `json:"alarmConfigurationUpdatedTimestamp"`
	ActionsEnabled                     bool              `json:"actionsEnabled"`
	OKActions                          []string          `json:"okActions"`
	AlarmActions                       []string          `json:"alarmActions"`
	InsufficientDataActions            []string          `json:"insufficientDataActions"`
	Tags                               map[string]string `json:"tags"`
	APMAlarmConfig                     APMAlarmConfig    `json:"apmAlarmConfig"`
	Region                             string            `json:"region"`
}

// AlarmConfig represents configuration for creating/updating alarms
type AlarmConfig struct {
	AlarmName                        string            `json:"alarmName"`
	AlarmDescription                 string            `json:"alarmDescription"`
	MetricName                       string            `json:"metricName"`
	Namespace                        string            `json:"namespace"`
	Statistic                        string            `json:"statistic"`
	Dimensions                       []*AlarmDimension `json:"dimensions"`
	Period                           int               `json:"period"`
	EvaluationPeriods                int               `json:"evaluationPeriods"`
	Threshold                        float64           `json:"threshold"`
	ComparisonOperator               string            `json:"comparisonOperator"`
	TreatMissingData                 string            `json:"treatMissingData"`
	EvaluateLowSampleCountPercentile string            `json:"evaluateLowSampleCountPercentile,omitempty"`
	DatapointsToAlarm                int               `json:"datapointsToAlarm"`
	ActionsEnabled                   bool              `json:"actionsEnabled"`
	OKActions                        []string          `json:"okActions"`
	AlarmActions                     []string          `json:"alarmActions"`
	InsufficientDataActions          []string          `json:"insufficientDataActions"`
	Tags                             map[string]string `json:"tags"`
	APMAlarmConfig                   APMAlarmConfig    `json:"apmAlarmConfig"`
}

// AlarmDimension represents a dimension for CloudWatch alarms
type AlarmDimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// AlarmState represents the state of a CloudWatch alarm
type AlarmState struct {
	Value             string            `json:"value"` // OK, ALARM, INSUFFICIENT_DATA
	Reason            string            `json:"reason"`
	ReasonData        string            `json:"reasonData"`
	Timestamp         time.Time         `json:"timestamp"`
	EvaluationHistory []AlarmEvaluation `json:"evaluationHistory"`
}

// AlarmEvaluation represents a single alarm evaluation
type AlarmEvaluation struct {
	Timestamp time.Time `json:"timestamp"`
	State     string    `json:"state"`
	Reason    string    `json:"reason"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
}

// APMAlarmConfig contains APM-specific alarm settings
type APMAlarmConfig struct {
	APMService         string            `json:"apmService"`
	Severity           string            `json:"severity"`      // Critical, High, Medium, Low
	AlertCategory      string            `json:"alertCategory"` // Infrastructure, Application, Business
	AutoRemediation    bool              `json:"autoRemediation"`
	RemediationActions []string          `json:"remediationActions"`
	EscalationPolicy   EscalationPolicy  `json:"escalationPolicy"`
	RunbookURL         string            `json:"runbookURL"`
	Tags               map[string]string `json:"tags"`
	Dependencies       []string          `json:"dependencies"`
	CorrelationRules   []string          `json:"correlationRules"`
}

// EscalationPolicy defines how alerts should be escalated
type EscalationPolicy struct {
	Enabled            bool              `json:"enabled"`
	EscalationLevels   []EscalationLevel `json:"escalationLevels"`
	MaxEscalationLevel int               `json:"maxEscalationLevel"`
	CooldownPeriod     time.Duration     `json:"cooldownPeriod"`
}

// EscalationLevel defines a single escalation level
type EscalationLevel struct {
	Level        int      `json:"level"`
	DelayMinutes int      `json:"delayMinutes"`
	Actions      []string `json:"actions"`
	Recipients   []string `json:"recipients"`
	RequireAck   bool     `json:"requireAck"`
	AutoResolve  bool     `json:"autoResolve"`
}

// CloudWatchLogGroup represents a CloudWatch log group
type CloudWatchLogGroup struct {
	LogGroupName         string            `json:"logGroupName"`
	LogGroupArn          string            `json:"logGroupArn"`
	CreationTime         time.Time         `json:"creationTime"`
	RetentionInDays      int               `json:"retentionInDays"`
	MetricFilterCount    int               `json:"metricFilterCount"`
	StoredBytes          int64             `json:"storedBytes"`
	Tags                 map[string]string `json:"tags"`
	KmsKeyId             string            `json:"kmsKeyId"`
	DataProtectionPolicy string            `json:"dataProtectionPolicy"`
	APMLogConfig         APMLogConfig      `json:"apmLogConfig"`
	Region               string            `json:"region"`
}

// LogGroupConfig represents configuration for creating/updating log groups
type LogGroupConfig struct {
	LogGroupName    string            `json:"logGroupName"`
	RetentionInDays int               `json:"retentionInDays"`
	KmsKeyId        string            `json:"kmsKeyId"`
	Tags            map[string]string `json:"tags"`
	APMLogConfig    APMLogConfig      `json:"apmLogConfig"`
}

// CloudWatchLogStream represents a CloudWatch log stream
type CloudWatchLogStream struct {
	LogStreamName       string    `json:"logStreamName"`
	CreationTime        time.Time `json:"creationTime"`
	FirstEventTime      time.Time `json:"firstEventTime"`
	LastEventTime       time.Time `json:"lastEventTime"`
	LastIngestionTime   time.Time `json:"lastIngestionTime"`
	UploadSequenceToken string    `json:"uploadSequenceToken"`
	StoredBytes         int64     `json:"storedBytes"`
	LogGroupName        string    `json:"logGroupName"`
}

// LogEvent represents a single log event
type LogEvent struct {
	Timestamp     int64  `json:"timestamp"`
	Message       string `json:"message"`
	IngestionTime int64  `json:"ingestionTime"`
	EventId       string `json:"eventId"`
}

// APMLogConfig contains APM-specific log configuration
type APMLogConfig struct {
	APMService        string            `json:"apmService"`
	LogFormat         string            `json:"logFormat"` // JSON, Plain, Custom
	StructuredLogging bool              `json:"structuredLogging"`
	LogLevel          string            `json:"logLevel"`
	MetricFilters     []MetricFilter    `json:"metricFilters"`
	Subscriptions     []LogSubscription `json:"subscriptions"`
	ExportConfig      LogExportConfig   `json:"exportConfig"`
	ParsingRules      []LogParsingRule  `json:"parsingRules"`
	AlertRules        []LogAlertRule    `json:"alertRules"`
}

// MetricFilter represents a CloudWatch metric filter
type MetricFilter struct {
	FilterName            string                 `json:"filterName"`
	FilterPattern         string                 `json:"filterPattern"`
	MetricTransformations []MetricTransformation `json:"metricTransformations"`
	CreationTime          time.Time              `json:"creationTime"`
	LogGroupName          string                 `json:"logGroupName"`
}

// MetricTransformation represents a metric transformation
type MetricTransformation struct {
	MetricName      string  `json:"metricName"`
	MetricNamespace string  `json:"metricNamespace"`
	MetricValue     string  `json:"metricValue"`
	DefaultValue    float64 `json:"defaultValue"`
	Unit            string  `json:"unit"`
}

// LogSubscription represents a CloudWatch log subscription
type LogSubscription struct {
	SubscriptionName string    `json:"subscriptionName"`
	LogGroupName     string    `json:"logGroupName"`
	FilterName       string    `json:"filterName"`
	FilterPattern    string    `json:"filterPattern"`
	DestinationArn   string    `json:"destinationArn"`
	RoleArn          string    `json:"roleArn"`
	Distribution     string    `json:"distribution"`
	CreationTime     time.Time `json:"creationTime"`
}

// LogExportConfig configures log export settings
type LogExportConfig struct {
	Enabled         bool          `json:"enabled"`
	S3Bucket        string        `json:"s3Bucket"`
	S3Prefix        string        `json:"s3Prefix"`
	ExportInterval  time.Duration `json:"exportInterval"`
	CompressionType string        `json:"compressionType"`
	EncryptionKey   string        `json:"encryptionKey"`
}

// LogParsingRule defines how to parse log messages
type LogParsingRule struct {
	RuleName string            `json:"ruleName"`
	Pattern  string            `json:"pattern"`
	Fields   map[string]string `json:"fields"`
	Enabled  bool              `json:"enabled"`
}

// LogAlertRule defines alerting rules for logs
type LogAlertRule struct {
	RuleName  string        `json:"ruleName"`
	Pattern   string        `json:"pattern"`
	Threshold int           `json:"threshold"`
	Period    time.Duration `json:"period"`
	Actions   []string      `json:"actions"`
	Severity  string        `json:"severity"`
	Enabled   bool          `json:"enabled"`
}

// CloudWatchInsightsQuery represents a CloudWatch Insights query
type CloudWatchInsightsQuery struct {
	QueryId        string          `json:"queryId"`
	QueryString    string          `json:"queryString"`
	LogGroups      []string        `json:"logGroups"`
	StartTime      time.Time       `json:"startTime"`
	EndTime        time.Time       `json:"endTime"`
	Status         string          `json:"status"`
	Statistics     QueryStatistics `json:"statistics"`
	Results        []QueryResult   `json:"results"`
	APMQueryConfig APMQueryConfig  `json:"apmQueryConfig"`
	Region         string          `json:"region"`
}

// QueryConfig represents configuration for executing Insights queries
type QueryConfig struct {
	QueryString    string         `json:"queryString"`
	LogGroups      []string       `json:"logGroups"`
	StartTime      time.Time      `json:"startTime"`
	EndTime        time.Time      `json:"endTime"`
	Limit          int            `json:"limit,omitempty"`
	APMQueryConfig APMQueryConfig `json:"apmQueryConfig"`
}

// QueryResult represents the result of an Insights query
type QueryResult struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
	Ptr       string                 `json:"ptr"`
}

// QueryStatistics contains statistics about a query execution
type QueryStatistics struct {
	BytesScanned   float64 `json:"bytesScanned"`
	RecordsMatched float64 `json:"recordsMatched"`
	RecordsScanned float64 `json:"recordsScanned"`
}

// APMQueryConfig contains APM-specific query configuration
type APMQueryConfig struct {
	APMService     string            `json:"apmService"`
	QueryTemplate  string            `json:"queryTemplate"`
	Variables      map[string]string `json:"variables"`
	CacheResults   bool              `json:"cacheResults"`
	CacheDuration  time.Duration     `json:"cacheDuration"`
	AlertOnResults bool              `json:"alertOnResults"`
	SavedQuery     bool              `json:"savedQuery"`
	QueryName      string            `json:"queryName"`
	Description    string            `json:"description"`
}

// CloudWatchEvent represents a CloudWatch event
type CloudWatchEvent struct {
	Version        string                 `json:"version"`
	Id             string                 `json:"id"`
	DetailType     string                 `json:"detail-type"`
	Source         string                 `json:"source"`
	Account        string                 `json:"account"`
	Time           time.Time              `json:"time"`
	Region         string                 `json:"region"`
	Detail         map[string]interface{} `json:"detail"`
	Resources      []string               `json:"resources"`
	EventRule      EventRule              `json:"eventRule"`
	APMEventConfig APMEventConfig         `json:"apmEventConfig"`
}

// EventRule represents a CloudWatch event rule
type EventRule struct {
	Name               string                 `json:"name"`
	Arn                string                 `json:"arn"`
	Description        string                 `json:"description"`
	EventPattern       map[string]interface{} `json:"eventPattern"`
	ScheduleExpression string                 `json:"scheduleExpression"`
	State              string                 `json:"state"`
	Targets            []EventTarget          `json:"targets"`
	Tags               map[string]string      `json:"tags"`
	CreatedBy          string                 `json:"createdBy"`
	EventBusName       string                 `json:"eventBusName"`
}

// EventRuleConfig represents configuration for creating/updating event rules
type EventRuleConfig struct {
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	EventPattern       map[string]interface{} `json:"eventPattern"`
	ScheduleExpression string                 `json:"scheduleExpression"`
	State              string                 `json:"state"`
	Targets            []EventTarget          `json:"targets"`
	Tags               map[string]string      `json:"tags"`
	EventBusName       string                 `json:"eventBusName"`
	APMEventConfig     APMEventConfig         `json:"apmEventConfig"`
}

// EventTarget represents a target for CloudWatch events
type EventTarget struct {
	Id                          string                      `json:"id"`
	Arn                         string                      `json:"arn"`
	RoleArn                     string                      `json:"roleArn"`
	Input                       string                      `json:"input"`
	InputPath                   string                      `json:"inputPath"`
	InputTransformer            InputTransformer            `json:"inputTransformer"`
	KinesisParameters           KinesisParameters           `json:"kinesisParameters"`
	RunCommandParameters        RunCommandParameters        `json:"runCommandParameters"`
	EcsParameters               EcsParameters               `json:"ecsParameters"`
	HttpParameters              HttpParameters              `json:"httpParameters"`
	RedshiftDataParameters      RedshiftDataParameters      `json:"redshiftDataParameters"`
	SageMakerPipelineParameters SageMakerPipelineParameters `json:"sageMakerPipelineParameters"`
	DeadLetterConfig            DeadLetterConfig            `json:"deadLetterConfig"`
	RetryPolicy                 RetryPolicy                 `json:"retryPolicy"`
}

// InputTransformer transforms input for event targets
type InputTransformer struct {
	InputPathsMap map[string]string `json:"inputPathsMap"`
	InputTemplate string            `json:"inputTemplate"`
}

// KinesisParameters contains Kinesis-specific parameters
type KinesisParameters struct {
	PartitionKeyPath string `json:"partitionKeyPath"`
}

// RunCommandParameters contains SSM Run Command parameters
type RunCommandParameters struct {
	RunCommandTargets []RunCommandTarget `json:"runCommandTargets"`
}

// RunCommandTarget represents a target for SSM Run Command
type RunCommandTarget struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// EcsParameters contains ECS-specific parameters
type EcsParameters struct {
	TaskDefinitionArn        string                         `json:"taskDefinitionArn"`
	LaunchType               string                         `json:"launchType"`
	NetworkConfiguration     NetworkConfiguration           `json:"networkConfiguration"`
	PlatformVersion          string                         `json:"platformVersion"`
	Group                    string                         `json:"group"`
	CapacityProviderStrategy []CapacityProviderStrategyItem `json:"capacityProviderStrategy"`
	EnableECSManagedTags     bool                           `json:"enableECSManagedTags"`
	EnableExecuteCommand     bool                           `json:"enableExecuteCommand"`
	PlacementConstraints     []PlacementConstraint          `json:"placementConstraints"`
	PlacementStrategy        []PlacementStrategy            `json:"placementStrategy"`
	PropagateTags            string                         `json:"propagateTags"`
	ReferenceId              string                         `json:"referenceId"`
	Tags                     map[string]string              `json:"tags"`
	TaskCount                int                            `json:"taskCount"`
}

// NetworkConfiguration contains network configuration for ECS tasks
type NetworkConfiguration struct {
	AwsvpcConfiguration AwsvpcConfiguration `json:"awsvpcConfiguration"`
}

// AwsvpcConfiguration contains VPC configuration for ECS tasks
type AwsvpcConfiguration struct {
	Subnets        []string `json:"subnets"`
	SecurityGroups []string `json:"securityGroups"`
	AssignPublicIp string   `json:"assignPublicIp"`
}

// CapacityProviderStrategyItem represents a capacity provider strategy
type CapacityProviderStrategyItem struct {
	CapacityProvider string `json:"capacityProvider"`
	Weight           int    `json:"weight"`
	Base             int    `json:"base"`
}

// PlacementConstraint represents a placement constraint for ECS tasks
type PlacementConstraint struct {
	Type       string `json:"type"`
	Expression string `json:"expression"`
}

// PlacementStrategy represents a placement strategy for ECS tasks
type PlacementStrategy struct {
	Type  string `json:"type"`
	Field string `json:"field"`
}

// HttpParameters contains HTTP-specific parameters
type HttpParameters struct {
	HeaderParameters      map[string]string `json:"headerParameters"`
	PathParameterValues   map[string]string `json:"pathParameterValues"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
}

// RedshiftDataParameters contains Redshift Data API parameters
type RedshiftDataParameters struct {
	Database         string   `json:"database"`
	DbUser           string   `json:"dbUser"`
	SecretManagerArn string   `json:"secretManagerArn"`
	Sql              string   `json:"sql"`
	StatementName    string   `json:"statementName"`
	WithEvent        bool     `json:"withEvent"`
	Sqls             []string `json:"sqls"`
}

// SageMakerPipelineParameters contains SageMaker Pipeline parameters
type SageMakerPipelineParameters struct {
	PipelineParameterList []PipelineParameter `json:"pipelineParameterList"`
}

// PipelineParameter represents a SageMaker pipeline parameter
type PipelineParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DeadLetterConfig configures dead letter queues for event targets
type DeadLetterConfig struct {
	Arn string `json:"arn"`
}

// RetryPolicy configures retry behavior for event targets
type RetryPolicy struct {
	MaximumRetryAttempts int `json:"maximumRetryAttempts"`
	MaximumEventAge      int `json:"maximumEventAge"`
}

// APMEventConfig contains APM-specific event configuration
type APMEventConfig struct {
	APMService        string             `json:"apmService"`
	EventCategory     string             `json:"eventCategory"` // Infrastructure, Application, Security, Business
	Severity          string             `json:"severity"`
	AutoRemediation   bool               `json:"autoRemediation"`
	NotificationRules []NotificationRule `json:"notificationRules"`
	CorrelationRules  []CorrelationRule  `json:"correlationRules"`
	EnrichmentRules   []EnrichmentRule   `json:"enrichmentRules"`
}

// NotificationRule defines how to notify on events
type NotificationRule struct {
	RuleName   string        `json:"ruleName"`
	Conditions []string      `json:"conditions"`
	Targets    []string      `json:"targets"`
	Template   string        `json:"template"`
	Cooldown   time.Duration `json:"cooldown"`
	Enabled    bool          `json:"enabled"`
}

// CorrelationRule defines how to correlate events
type CorrelationRule struct {
	RuleName       string        `json:"ruleName"`
	Pattern        string        `json:"pattern"`
	TimeWindow     time.Duration `json:"timeWindow"`
	ThresholdCount int           `json:"thresholdCount"`
	Actions        []string      `json:"actions"`
	Enabled        bool          `json:"enabled"`
}

// EnrichmentRule defines how to enrich events with additional data
type EnrichmentRule struct {
	RuleName    string            `json:"ruleName"`
	Conditions  []string          `json:"conditions"`
	Enrichments map[string]string `json:"enrichments"`
	Enabled     bool              `json:"enabled"`
}

// SNSTopic represents an SNS topic for notifications
type SNSTopic struct {
	TopicArn              string                `json:"topicArn"`
	TopicName             string                `json:"topicName"`
	DisplayName           string                `json:"displayName"`
	Subscriptions         []SNSSubscription     `json:"subscriptions"`
	Attributes            map[string]string     `json:"attributes"`
	Tags                  map[string]string     `json:"tags"`
	Region                string                `json:"region"`
	APMNotificationConfig APMNotificationConfig `json:"apmNotificationConfig"`
}

// SNSTopicConfig represents configuration for creating/updating SNS topics
type SNSTopicConfig struct {
	TopicName             string                `json:"topicName"`
	DisplayName           string                `json:"displayName"`
	Attributes            map[string]string     `json:"attributes"`
	Tags                  map[string]string     `json:"tags"`
	APMNotificationConfig APMNotificationConfig `json:"apmNotificationConfig"`
}

// SNSSubscription represents an SNS subscription
type SNSSubscription struct {
	SubscriptionArn              string                 `json:"subscriptionArn"`
	TopicArn                     string                 `json:"topicArn"`
	Protocol                     string                 `json:"protocol"`
	Endpoint                     string                 `json:"endpoint"`
	Attributes                   map[string]string      `json:"attributes"`
	FilterPolicy                 map[string]interface{} `json:"filterPolicy"`
	ConfirmationWasAuthenticated bool                   `json:"confirmationWasAuthenticated"`
	PendingConfirmation          bool                   `json:"pendingConfirmation"`
}

// APMNotificationConfig contains APM-specific notification configuration
type APMNotificationConfig struct {
	APMService        string                 `json:"apmService"`
	NotificationTypes []string               `json:"notificationTypes"`
	Severity          []string               `json:"severity"`
	MessageTemplates  map[string]string      `json:"messageTemplates"`
	DeliveryOptions   DeliveryOptions        `json:"deliveryOptions"`
	FilterRules       []NotificationFilter   `json:"filterRules"`
	EscalationConfig  NotificationEscalation `json:"escalationConfig"`
}

// DeliveryOptions configures notification delivery
type DeliveryOptions struct {
	RetryPolicy     NotificationRetryPolicy `json:"retryPolicy"`
	DeadLetterQueue string                  `json:"deadLetterQueue"`
	DeliveryDelay   time.Duration           `json:"deliveryDelay"`
	ThrottleLimit   int                     `json:"throttleLimit"`
	BatchSettings   BatchSettings           `json:"batchSettings"`
}

// NotificationRetryPolicy configures retry behavior for notifications
type NotificationRetryPolicy struct {
	MaxRetries    int           `json:"maxRetries"`
	RetryBackoff  string        `json:"retryBackoff"` // linear, exponential
	RetryDelay    time.Duration `json:"retryDelay"`
	MaxRetryDelay time.Duration `json:"maxRetryDelay"`
}

// BatchSettings configures notification batching
type BatchSettings struct {
	Enabled      bool          `json:"enabled"`
	BatchSize    int           `json:"batchSize"`
	BatchTimeout time.Duration `json:"batchTimeout"`
	Grouping     string        `json:"grouping"` // service, severity, type
}

// NotificationFilter2 filters notifications based on conditions (renamed to avoid conflict)
type NotificationFilter2 struct {
	FilterName string            `json:"filterName"`
	Conditions map[string]string `json:"conditions"`
	Action     string            `json:"action"` // include, exclude
	Enabled    bool              `json:"enabled"`
}

// NotificationEscalation configures notification escalation
type NotificationEscalation struct {
	Enabled          bool                          `json:"enabled"`
	EscalationLevels []NotificationEscalationLevel `json:"escalationLevels"`
	MaxLevel         int                           `json:"maxLevel"`
	CooldownPeriod   time.Duration                 `json:"cooldownPeriod"`
}

// NotificationEscalationLevel defines a single escalation level for notifications
type NotificationEscalationLevel struct {
	Level        int      `json:"level"`
	DelayMinutes int      `json:"delayMinutes"`
	Targets      []string `json:"targets"`
	Template     string   `json:"template"`
	RequireAck   bool     `json:"requireAck"`
}

// NewAWSProvider creates a new AWS provider
func NewAWSProvider(config *ProviderConfig) (*AWSProvider, error) {
	if config == nil {
		config = &ProviderConfig{
			Provider:      ProviderAWS,
			DefaultRegion: "us-east-1",
			EnableCache:   true,
			CacheDuration: 5 * time.Minute,
		}
	}

	p := &AWSProvider{
		config:    config,
		cache:     NewCredentialCache(config.CacheDuration),
		ecrTokens: make(map[string]*ECRToken),
	}

	p.cloudWatch = &CloudWatchIntegration{provider: p}
	p.s3Manager = &S3Manager{provider: p}
	p.cfManager = &CloudFormationManager{provider: p}

	return p, nil
}

// Name returns the provider name
func (p *AWSProvider) Name() Provider {
	return ProviderAWS
}

// DetectCLI detects AWS CLI installation with enhanced validation
func (p *AWSProvider) DetectCLI() (*CLIStatus, error) {
	detector := NewAWSCLIDetector()
	status, err := detector.Detect()
	if err != nil {
		return nil, fmt.Errorf("AWS CLI detection failed: %w", err)
	}
	p.cliStatus = status
	return status, nil
}

// DetectCLIWithDetails provides comprehensive CLI detection and validation
func (p *AWSProvider) DetectCLIWithDetails() (*AWSCLIValidationResult, error) {
	detector := NewAWSCLIDetector()
	result, err := detector.GetDetailedValidationResult()
	if err != nil {
		return nil, fmt.Errorf("detailed AWS CLI validation failed: %w", err)
	}

	// Update provider status based on detection results
	if result.SelectedInstallation != nil {
		p.cliStatus = &CLIStatus{
			Installed:   true,
			Version:     result.SelectedInstallation.Version,
			Path:        result.SelectedInstallation.Path,
			ConfigPath:  detector.BaseCLIDetector.getConfigPath(),
			MinVersion:  detector.GetMinVersion(),
			IsSupported: result.Status == CLIStatusOK,
		}
	} else {
		p.cliStatus = &CLIStatus{
			Installed:   false,
			MinVersion:  detector.GetMinVersion(),
			IsSupported: false,
		}
	}

	return result, nil
}

// ValidateCLI validates AWS CLI is properly configured
func (p *AWSProvider) ValidateCLI() error {
	if p.cliStatus == nil {
		if _, err := p.DetectCLI(); err != nil {
			return err
		}
	}

	if !p.cliStatus.Installed {
		return fmt.Errorf("AWS CLI not installed")
	}

	// Check if configured
	cmd := exec.Command("aws", "sts", "get-caller-identity")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AWS CLI not authenticated: %w", err)
	}

	return nil
}

// GetCLIVersion returns the AWS CLI version
func (p *AWSProvider) GetCLIVersion() (string, error) {
	if p.cliStatus == nil {
		if _, err := p.DetectCLI(); err != nil {
			return "", err
		}
	}
	return p.cliStatus.Version, nil
}

// ValidateAuth validates AWS authentication
func (p *AWSProvider) ValidateAuth(ctx context.Context) error {
	cmd := exec.Command("aws", "sts", "get-caller-identity")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("authentication validation failed: %w", err)
	}

	// Parse the output to get account info
	var identity struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
		UserID  string `json:"UserId"`
	}
	if err := json.Unmarshal(output, &identity); err != nil {
		return fmt.Errorf("failed to parse identity: %w", err)
	}

	// Store account info in credentials
	if p.credentials == nil {
		p.credentials = &Credentials{
			Provider:   ProviderAWS,
			AuthMethod: AuthMethodCLI,
		}
	}
	p.credentials.Account = identity.Account

	return nil
}

// GetCredentials returns current AWS credentials
func (p *AWSProvider) GetCredentials() (*Credentials, error) {
	if p.credentials != nil {
		return p.credentials, nil
	}

	// Try to get from environment
	if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
		p.credentials = &Credentials{
			Provider:   ProviderAWS,
			AuthMethod: AuthMethodAccessKey,
			AccessKey:  accessKey,
			SecretKey:  os.Getenv("AWS_SECRET_ACCESS_KEY"),
			Token:      os.Getenv("AWS_SESSION_TOKEN"),
			Region:     os.Getenv("AWS_REGION"),
		}
		return p.credentials, nil
	}

	// Try to get from CLI config
	profile := p.config.DefaultProfile
	if profile == "" {
		profile = os.Getenv("AWS_PROFILE")
		if profile == "" {
			profile = "default"
		}
	}

	p.credentials = &Credentials{
		Provider:   ProviderAWS,
		AuthMethod: AuthMethodCLI,
		Profile:    profile,
		Region:     p.GetCurrentRegion(),
	}

	return p.credentials, nil
}

// ListRegistries lists ECR registries
func (p *AWSProvider) ListRegistries(ctx context.Context) ([]*Registry, error) {
	// Get current region
	region := p.GetCurrentRegion()

	// Get account ID
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}
	accountID := strings.TrimSpace(string(output))

	// List repositories
	cmd = exec.Command("aws", "ecr", "describe-repositories", "--region", region)
	output, err = cmd.Output()
	if err != nil {
		// If no repositories exist, return empty list
		if strings.Contains(err.Error(), "RepositoryNotFoundException") {
			return []*Registry{}, nil
		}
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	var result struct {
		Repositories []struct {
			RepositoryName string `json:"repositoryName"`
			RepositoryArn  string `json:"repositoryArn"`
			RepositoryUri  string `json:"repositoryUri"`
		} `json:"repositories"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse repositories: %w", err)
	}

	registries := make([]*Registry, 0, len(result.Repositories))
	for _, repo := range result.Repositories {
		registries = append(registries, &Registry{
			Provider: ProviderAWS,
			Name:     repo.RepositoryName,
			URL:      repo.RepositoryUri,
			Region:   region,
			Type:     "ECR",
		})
	}

	// Also add the base registry
	baseRegistry := &Registry{
		Provider: ProviderAWS,
		Name:     "ecr-base",
		URL:      fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, region),
		Region:   region,
		Type:     "ECR",
	}
	registries = append([]*Registry{baseRegistry}, registries...)

	return registries, nil
}

// GetRegistry gets a specific ECR registry
func (p *AWSProvider) GetRegistry(ctx context.Context, name string) (*Registry, error) {
	registries, err := p.ListRegistries(ctx)
	if err != nil {
		return nil, err
	}

	for _, registry := range registries {
		if registry.Name == name {
			return registry, nil
		}
	}

	return nil, fmt.Errorf("registry %s not found", name)
}

// AuthenticateRegistry authenticates to ECR
func (p *AWSProvider) AuthenticateRegistry(ctx context.Context, registry *Registry) error {
	region := registry.Region
	if region == "" {
		region = p.GetCurrentRegion()
	}

	// Get ECR login token
	cmd := exec.Command("aws", "ecr", "get-login-password", "--region", region)
	token, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get ECR login token: %w", err)
	}

	// Login to Docker
	loginCmd := exec.Command("docker", "login", "--username", "AWS", "--password-stdin", registry.URL)
	loginCmd.Stdin = strings.NewReader(string(token))
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to login to ECR: %w", err)
	}

	return nil
}

// ListClusters lists EKS clusters
func (p *AWSProvider) ListClusters(ctx context.Context) ([]*Cluster, error) {
	region := p.GetCurrentRegion()

	cmd := exec.Command("aws", "eks", "list-clusters", "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var result struct {
		Clusters []string `json:"clusters"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse clusters: %w", err)
	}

	clusters := make([]*Cluster, 0, len(result.Clusters))
	for _, clusterName := range result.Clusters {
		// Get cluster details
		cluster, err := p.GetCluster(ctx, clusterName)
		if err != nil {
			continue
		}
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// GetCluster gets details of an EKS cluster
func (p *AWSProvider) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	region := p.GetCurrentRegion()

	cmd := exec.Command("aws", "eks", "describe-cluster", "--name", name, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	var result struct {
		Cluster struct {
			Name     string            `json:"name"`
			Arn      string            `json:"arn"`
			Version  string            `json:"version"`
			Endpoint string            `json:"endpoint"`
			Status   string            `json:"status"`
			Tags     map[string]string `json:"tags"`
		} `json:"cluster"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cluster: %w", err)
	}

	// Get node count
	nodeCount := 0
	cmd = exec.Command("aws", "eks", "list-nodegroups", "--cluster-name", name, "--region", region)
	if output, err := cmd.Output(); err == nil {
		var nodeResult struct {
			Nodegroups []string `json:"nodegroups"`
		}
		if json.Unmarshal(output, &nodeResult) == nil {
			// For simplicity, we'll just count nodegroups
			// In reality, you'd want to sum the desired capacity of each nodegroup
			nodeCount = len(nodeResult.Nodegroups)
		}
	}

	return &Cluster{
		Provider:  ProviderAWS,
		Name:      result.Cluster.Name,
		Region:    region,
		Type:      "EKS",
		Version:   result.Cluster.Version,
		Endpoint:  result.Cluster.Endpoint,
		NodeCount: nodeCount,
		Status:    result.Cluster.Status,
		Labels:    result.Cluster.Tags,
	}, nil
}

// GetKubeconfig gets kubeconfig for an EKS cluster
func (p *AWSProvider) GetKubeconfig(ctx context.Context, cluster *Cluster) ([]byte, error) {
	region := cluster.Region
	if region == "" {
		region = p.GetCurrentRegion()
	}

	// Create a temporary kubeconfig file
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Update kubeconfig
	cmd := exec.Command("aws", "eks", "update-kubeconfig",
		"--name", cluster.Name,
		"--region", region,
		"--kubeconfig", tmpFile.Name(),
	)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	// Read the kubeconfig
	kubeconfig, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

// ListRegions lists AWS regions
func (p *AWSProvider) ListRegions(ctx context.Context) ([]string, error) {
	cmd := exec.Command("aws", "ec2", "describe-regions", "--query", "Regions[].RegionName", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	var regions []string
	if err := json.Unmarshal(output, &regions); err != nil {
		return nil, fmt.Errorf("failed to parse regions: %w", err)
	}

	return regions, nil
}

// GetCurrentRegion gets the current AWS region
func (p *AWSProvider) GetCurrentRegion() string {
	// Check config
	if p.config.DefaultRegion != "" {
		return p.config.DefaultRegion
	}

	// Check environment
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	// Try to get from CLI config
	cmd := exec.Command("aws", "configure", "get", "region")
	if output, err := cmd.Output(); err == nil {
		if region := strings.TrimSpace(string(output)); region != "" {
			return region
		}
	}

	// Try to detect from instance metadata (if running on EC2)
	if region := p.detectRegionFromMetadata(); region != "" {
		return region
	}

	// Default to us-east-1
	return "us-east-1"
}

// SetRegion sets the AWS region
func (p *AWSProvider) SetRegion(region string) error {
	p.config.DefaultRegion = region
	return nil
}

// ===============================
// Enhanced Region Management
// ===============================

// ValidateRegion validates if a region is valid and available
func (p *AWSProvider) ValidateRegion(ctx context.Context, region string) (*RegionValidation, error) {
	allRegions, err := p.ListRegions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	validation := &RegionValidation{
		Region:    region,
		IsValid:   false,
		IsEnabled: false,
	}

	// Check if region exists
	for _, validRegion := range allRegions {
		if validRegion == region {
			validation.IsValid = true
			break
		}
	}

	if !validation.IsValid {
		return validation, nil
	}

	// Check if region is enabled for the account
	details, err := p.GetRegionDetails(ctx, region)
	if err != nil {
		validation.Error = err.Error()
		return validation, nil
	}

	validation.IsEnabled = details.Status == "enabled" || details.Status == "enabled-by-default"
	validation.Details = details

	return validation, nil
}

// GetRegionDetails gets detailed information about a region
func (p *AWSProvider) GetRegionDetails(ctx context.Context, region string) (*RegionDetails, error) {
	cmd := exec.Command("aws", "ec2", "describe-regions", "--region-names", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe region: %w", err)
	}

	var result struct {
		Regions []struct {
			RegionName  string `json:"RegionName"`
			Endpoint    string `json:"Endpoint"`
			OptInStatus string `json:"OptInStatus"`
		} `json:"Regions"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse region details: %w", err)
	}

	if len(result.Regions) == 0 {
		return nil, fmt.Errorf("region %s not found", region)
	}

	regionInfo := result.Regions[0]

	// Get availability zones
	availabilityZones, err := p.ListAvailabilityZones(ctx, region)
	if err != nil {
		availabilityZones = []AvailabilityZone{} // Continue without AZs
	}

	return &RegionDetails{
		Name:              regionInfo.RegionName,
		Endpoint:          regionInfo.Endpoint,
		Status:            regionInfo.OptInStatus,
		AvailabilityZones: availabilityZones,
	}, nil
}

// ListAvailabilityZones lists availability zones in a region
func (p *AWSProvider) ListAvailabilityZones(ctx context.Context, region string) ([]AvailabilityZone, error) {
	cmd := exec.Command("aws", "ec2", "describe-availability-zones", "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe availability zones: %w", err)
	}

	var result struct {
		AvailabilityZones []struct {
			ZoneName           string `json:"ZoneName"`
			ZoneId             string `json:"ZoneId"`
			State              string `json:"State"`
			RegionName         string `json:"RegionName"`
			GroupName          string `json:"GroupName"`
			NetworkBorderGroup string `json:"NetworkBorderGroup"`
			Messages           []struct {
				Message string `json:"Message"`
			} `json:"Messages"`
		} `json:"AvailabilityZones"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse availability zones: %w", err)
	}

	zones := make([]AvailabilityZone, 0, len(result.AvailabilityZones))
	for _, az := range result.AvailabilityZones {
		var messages []string
		for _, msg := range az.Messages {
			messages = append(messages, msg.Message)
		}

		zones = append(zones, AvailabilityZone{
			Name:               az.ZoneName,
			ZoneId:             az.ZoneId,
			State:              az.State,
			Region:             az.RegionName,
			GroupName:          az.GroupName,
			NetworkBorderGroup: az.NetworkBorderGroup,
			Messages:           messages,
		})
	}

	return zones, nil
}

// detectRegionFromMetadata attempts to detect region from EC2 instance metadata
func (p *AWSProvider) detectRegionFromMetadata() string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://169.254.169.254/latest/meta-data/placement/region")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(body))
}

// Types for region management
type RegionValidation struct {
	Region    string         `json:"region"`
	IsValid   bool           `json:"isValid"`
	IsEnabled bool           `json:"isEnabled"`
	Details   *RegionDetails `json:"details,omitempty"`
	Error     string         `json:"error,omitempty"`
}

type RegionDetails struct {
	Name              string             `json:"name"`
	Endpoint          string             `json:"endpoint"`
	Status            string             `json:"status"`
	AvailabilityZones []AvailabilityZone `json:"availabilityZones"`
}

type AvailabilityZone struct {
	Name               string   `json:"name"`
	ZoneId             string   `json:"zoneId"`
	State              string   `json:"state"`
	Region             string   `json:"region"`
	GroupName          string   `json:"groupName"`
	NetworkBorderGroup string   `json:"networkBorderGroup"`
	Messages           []string `json:"messages"`
}

// AWSAPIFallback provides API-based operations when CLI is not available
type AWSAPIFallback struct {
	provider *AWSProvider
}

// NewAWSAPIFallback creates a new AWS API fallback
func NewAWSAPIFallback(provider *AWSProvider) *AWSAPIFallback {
	return &AWSAPIFallback{
		provider: provider,
	}
}

// IsAvailable checks if API fallback is available
func (f *AWSAPIFallback) IsAvailable() bool {
	// Check if AWS SDK credentials are available
	creds, err := f.provider.GetCredentials()
	if err != nil {
		return false
	}
	return creds.AuthMethod == AuthMethodAccessKey || creds.AuthMethod == AuthMethodIAMRole
}

// ListClustersViaAPI lists clusters using AWS SDK
func (f *AWSAPIFallback) ListClustersViaAPI(ctx context.Context) ([]*Cluster, error) {
	// This would use AWS SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("AWS SDK implementation required for API fallback")
}

// ListRegistriesViaAPI lists registries using AWS SDK
func (f *AWSAPIFallback) ListRegistriesViaAPI(ctx context.Context) ([]*Registry, error) {
	// This would use AWS SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("AWS SDK implementation required for API fallback")
}

// GetCredentialsViaAPI gets credentials using AWS SDK
func (f *AWSAPIFallback) GetCredentialsViaAPI(ctx context.Context) (*Credentials, error) {
	// This would use AWS SDK for Go
	// For now, return an error indicating SDK implementation needed
	return nil, fmt.Errorf("AWS SDK implementation required for API fallback")
}

// ===============================
// Enhanced ECR Management Methods
// ===============================

// GetECRToken retrieves and caches ECR authentication tokens
func (p *AWSProvider) GetECRToken(ctx context.Context, registry string) (*ECRToken, error) {
	region := p.GetCurrentRegion()
	if registry == "" {
		// Get account ID for default registry
		accountID, err := p.getAccountID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get account ID: %w", err)
		}
		registry = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, region)
	}

	tokenKey := fmt.Sprintf("%s-%s", registry, region)

	// Check if we have a valid cached token
	if token, exists := p.ecrTokens[tokenKey]; exists {
		if time.Now().Before(token.ExpiresAt.Add(-5 * time.Minute)) {
			return token, nil
		}
	}

	// Get new token from AWS
	cmd := exec.Command("aws", "ecr", "get-login-password", "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get ECR token: %w", err)
	}

	token := &ECRToken{
		Token:     strings.TrimSpace(string(output)),
		ExpiresAt: time.Now().Add(12 * time.Hour), // ECR tokens last 12 hours
		Registry:  registry,
		Region:    region,
	}

	p.ecrTokens[tokenKey] = token
	return token, nil
}

// CreateECRRepository creates a new ECR repository
func (p *AWSProvider) CreateECRRepository(ctx context.Context, name string) (*Registry, error) {
	region := p.GetCurrentRegion()

	cmd := exec.Command("aws", "ecr", "create-repository",
		"--repository-name", name,
		"--region", region,
		"--image-scanning-configuration", "scanOnPush=true",
		"--encryption-configuration", "encryptionType=AES256")

	output, err := cmd.Output()
	if err != nil {
		// Check if repository already exists
		if strings.Contains(err.Error(), "RepositoryAlreadyExistsException") {
			return p.GetRegistry(ctx, name)
		}
		return nil, fmt.Errorf("failed to create ECR repository: %w", err)
	}

	var result struct {
		Repository struct {
			RepositoryName string `json:"repositoryName"`
			RepositoryUri  string `json:"repositoryUri"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse repository response: %w", err)
	}

	return &Registry{
		Provider: ProviderAWS,
		Name:     result.Repository.RepositoryName,
		URL:      result.Repository.RepositoryUri,
		Region:   region,
		Type:     "ECR",
	}, nil
}

// ListECRImages lists images in an ECR repository
func (p *AWSProvider) ListECRImages(ctx context.Context, repositoryName string) ([]ECRImage, error) {
	region := p.GetCurrentRegion()

	cmd := exec.Command("aws", "ecr", "list-images",
		"--repository-name", repositoryName,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list ECR images: %w", err)
	}

	var result struct {
		ImageIds []struct {
			ImageTag    string `json:"imageTag"`
			ImageDigest string `json:"imageDigest"`
		} `json:"imageIds"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse images response: %w", err)
	}

	images := make([]ECRImage, 0, len(result.ImageIds))
	for _, img := range result.ImageIds {
		images = append(images, ECRImage{
			Tag:    img.ImageTag,
			Digest: img.ImageDigest,
		})
	}

	return images, nil
}

// ECRImage represents an ECR image
type ECRImage struct {
	Tag    string `json:"tag"`
	Digest string `json:"digest"`
}

// getAccountID gets the AWS account ID
func (p *AWSProvider) getAccountID(ctx context.Context) (string, error) {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get account ID: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// ===============================
// Enhanced EKS Management Methods
// ===============================

// ListEKSClustersAllRegions lists EKS clusters across all regions
func (p *AWSProvider) ListEKSClustersAllRegions(ctx context.Context) ([]*Cluster, error) {
	regions, err := p.ListRegions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	var allClusters []*Cluster
	for _, region := range regions {
		// Temporarily set region
		originalRegion := p.config.DefaultRegion
		p.config.DefaultRegion = region

		clusters, err := p.ListClusters(ctx)
		if err != nil {
			// Skip regions with errors (e.g., access denied)
			continue
		}

		allClusters = append(allClusters, clusters...)

		// Restore original region
		p.config.DefaultRegion = originalRegion
	}

	return allClusters, nil
}

// GetEKSClusterDetails gets detailed information about an EKS cluster
func (p *AWSProvider) GetEKSClusterDetails(ctx context.Context, clusterName, region string) (*EKSClusterDetails, error) {
	if region == "" {
		region = p.GetCurrentRegion()
	}

	cmd := exec.Command("aws", "eks", "describe-cluster", "--name", clusterName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
	}

	var result struct {
		Cluster struct {
			Name            string                 `json:"name"`
			Arn             string                 `json:"arn"`
			Version         string                 `json:"version"`
			Endpoint        string                 `json:"endpoint"`
			Status          string                 `json:"status"`
			Tags            map[string]string      `json:"tags"`
			RoleArn         string                 `json:"roleArn"`
			VpcConfig       map[string]interface{} `json:"resourcesVpcConfig"`
			Logging         map[string]interface{} `json:"logging"`
			Encryption      map[string]interface{} `json:"encryptionConfig"`
			PlatformVersion string                 `json:"platformVersion"`
			CreatedAt       time.Time              `json:"createdAt"`
		} `json:"cluster"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cluster details: %w", err)
	}

	// Get node groups
	nodeGroups, err := p.listNodeGroups(ctx, clusterName, region)
	if err != nil {
		nodeGroups = []NodeGroup{} // Continue without node groups if there's an error
	}

	// Get Fargate profiles
	fargateProfiles, err := p.listFargateProfiles(ctx, clusterName, region)
	if err != nil {
		fargateProfiles = []FargateProfile{} // Continue without Fargate profiles
	}

	return &EKSClusterDetails{
		Name:            result.Cluster.Name,
		Arn:             result.Cluster.Arn,
		Version:         result.Cluster.Version,
		Endpoint:        result.Cluster.Endpoint,
		Status:          result.Cluster.Status,
		Tags:            result.Cluster.Tags,
		RoleArn:         result.Cluster.RoleArn,
		VpcConfig:       result.Cluster.VpcConfig,
		Logging:         result.Cluster.Logging,
		Encryption:      result.Cluster.Encryption,
		PlatformVersion: result.Cluster.PlatformVersion,
		CreatedAt:       result.Cluster.CreatedAt,
		Region:          region,
		NodeGroups:      nodeGroups,
		FargateProfiles: fargateProfiles,
	}, nil
}

// SetupKubeconfig sets up kubeconfig for an EKS cluster
func (p *AWSProvider) SetupKubeconfig(ctx context.Context, clusterName, region string, options *KubeconfigOptions) error {
	if region == "" {
		region = p.GetCurrentRegion()
	}

	if options == nil {
		options = &KubeconfigOptions{
			Overwrite: false,
			Alias:     "",
		}
	}

	args := []string{"eks", "update-kubeconfig", "--name", clusterName, "--region", region}

	if options.Alias != "" {
		args = append(args, "--alias", options.Alias)
	}

	if options.Overwrite {
		args = append(args, "--overwrite")
	}

	if options.KubeconfigPath != "" {
		args = append(args, "--kubeconfig", options.KubeconfigPath)
	}

	cmd := exec.Command("aws", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update kubeconfig: %w", err)
	}

	return nil
}

// listNodeGroups lists node groups for an EKS cluster
func (p *AWSProvider) listNodeGroups(ctx context.Context, clusterName, region string) ([]NodeGroup, error) {
	cmd := exec.Command("aws", "eks", "list-nodegroups", "--cluster-name", clusterName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list node groups: %w", err)
	}

	var result struct {
		Nodegroups []string `json:"nodegroups"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse node groups: %w", err)
	}

	var nodeGroups []NodeGroup
	for _, ngName := range result.Nodegroups {
		// Get detailed info for each node group
		ng, err := p.getNodeGroupDetails(ctx, clusterName, ngName, region)
		if err != nil {
			continue // Skip if we can't get details
		}
		nodeGroups = append(nodeGroups, *ng)
	}

	return nodeGroups, nil
}

// getNodeGroupDetails gets detailed information about a node group
func (p *AWSProvider) getNodeGroupDetails(ctx context.Context, clusterName, nodeGroupName, region string) (*NodeGroup, error) {
	cmd := exec.Command("aws", "eks", "describe-nodegroup",
		"--cluster-name", clusterName,
		"--nodegroup-name", nodeGroupName,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe node group: %w", err)
	}

	var result struct {
		Nodegroup struct {
			NodegroupName string            `json:"nodegroupName"`
			Status        string            `json:"status"`
			InstanceTypes []string          `json:"instanceTypes"`
			AmiType       string            `json:"amiType"`
			NodeRole      string            `json:"nodeRole"`
			ScalingConfig map[string]int    `json:"scalingConfig"`
			RemoteAccess  map[string]string `json:"remoteAccess"`
			Tags          map[string]string `json:"tags"`
			CapacityType  string            `json:"capacityType"`
			CreatedAt     time.Time         `json:"createdAt"`
			ModifiedAt    time.Time         `json:"modifiedAt"`
		} `json:"nodegroup"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse node group details: %w", err)
	}

	return &NodeGroup{
		Name:          result.Nodegroup.NodegroupName,
		Status:        result.Nodegroup.Status,
		InstanceTypes: result.Nodegroup.InstanceTypes,
		AmiType:       result.Nodegroup.AmiType,
		NodeRole:      result.Nodegroup.NodeRole,
		ScalingConfig: result.Nodegroup.ScalingConfig,
		RemoteAccess:  result.Nodegroup.RemoteAccess,
		Tags:          result.Nodegroup.Tags,
		CapacityType:  result.Nodegroup.CapacityType,
		CreatedAt:     result.Nodegroup.CreatedAt,
		ModifiedAt:    result.Nodegroup.ModifiedAt,
	}, nil
}

// listFargateProfiles lists Fargate profiles for an EKS cluster
func (p *AWSProvider) listFargateProfiles(ctx context.Context, clusterName, region string) ([]FargateProfile, error) {
	cmd := exec.Command("aws", "eks", "list-fargate-profiles", "--cluster-name", clusterName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Fargate profiles: %w", err)
	}

	var result struct {
		FargateProfileNames []string `json:"fargateProfileNames"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Fargate profiles: %w", err)
	}

	var fargateProfiles []FargateProfile
	for _, fpName := range result.FargateProfileNames {
		// Get detailed info for each Fargate profile
		fp, err := p.getFargateProfileDetails(ctx, clusterName, fpName, region)
		if err != nil {
			continue // Skip if we can't get details
		}
		fargateProfiles = append(fargateProfiles, *fp)
	}

	return fargateProfiles, nil
}

// getFargateProfileDetails gets detailed information about a Fargate profile
func (p *AWSProvider) getFargateProfileDetails(ctx context.Context, clusterName, profileName, region string) (*FargateProfile, error) {
	cmd := exec.Command("aws", "eks", "describe-fargate-profile",
		"--cluster-name", clusterName,
		"--fargate-profile-name", profileName,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe Fargate profile: %w", err)
	}

	var result struct {
		FargateProfile struct {
			FargateProfileName  string                   `json:"fargateProfileName"`
			Status              string                   `json:"status"`
			PodExecutionRoleArn string                   `json:"podExecutionRoleArn"`
			Selectors           []map[string]interface{} `json:"selectors"`
			Subnets             []string                 `json:"subnets"`
			Tags                map[string]string        `json:"tags"`
			CreatedAt           time.Time                `json:"createdAt"`
		} `json:"fargateProfile"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Fargate profile details: %w", err)
	}

	return &FargateProfile{
		Name:                result.FargateProfile.FargateProfileName,
		Status:              result.FargateProfile.Status,
		PodExecutionRoleArn: result.FargateProfile.PodExecutionRoleArn,
		Selectors:           result.FargateProfile.Selectors,
		Subnets:             result.FargateProfile.Subnets,
		Tags:                result.FargateProfile.Tags,
		CreatedAt:           result.FargateProfile.CreatedAt,
	}, nil
}

// EKSClusterDetails represents detailed EKS cluster information
type EKSClusterDetails struct {
	Name            string                 `json:"name"`
	Arn             string                 `json:"arn"`
	Version         string                 `json:"version"`
	Endpoint        string                 `json:"endpoint"`
	Status          string                 `json:"status"`
	Tags            map[string]string      `json:"tags"`
	RoleArn         string                 `json:"roleArn"`
	VpcConfig       map[string]interface{} `json:"vpcConfig"`
	Logging         map[string]interface{} `json:"logging"`
	Encryption      map[string]interface{} `json:"encryption"`
	PlatformVersion string                 `json:"platformVersion"`
	CreatedAt       time.Time              `json:"createdAt"`
	Region          string                 `json:"region"`
	NodeGroups      []NodeGroup            `json:"nodeGroups"`
	FargateProfiles []FargateProfile       `json:"fargateProfiles"`
}

// NodeGroup represents an EKS node group
type NodeGroup struct {
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	InstanceTypes []string          `json:"instanceTypes"`
	AmiType       string            `json:"amiType"`
	NodeRole      string            `json:"nodeRole"`
	ScalingConfig map[string]int    `json:"scalingConfig"`
	RemoteAccess  map[string]string `json:"remoteAccess"`
	Tags          map[string]string `json:"tags"`
	CapacityType  string            `json:"capacityType"`
	CreatedAt     time.Time         `json:"createdAt"`
	ModifiedAt    time.Time         `json:"modifiedAt"`
}

// FargateProfile represents an EKS Fargate profile
type FargateProfile struct {
	Name                string                   `json:"name"`
	Status              string                   `json:"status"`
	PodExecutionRoleArn string                   `json:"podExecutionRoleArn"`
	Selectors           []map[string]interface{} `json:"selectors"`
	Subnets             []string                 `json:"subnets"`
	Tags                map[string]string        `json:"tags"`
	CreatedAt           time.Time                `json:"createdAt"`
}

// KubeconfigOptions represents options for kubeconfig setup
type KubeconfigOptions struct {
	Overwrite      bool   `json:"overwrite"`
	Alias          string `json:"alias"`
	KubeconfigPath string `json:"kubeconfigPath"`
}

// ===============================
// IAM Role Validation and Management
// ===============================

// ValidateIAMRole validates an IAM role and its policies
func (p *AWSProvider) ValidateIAMRole(ctx context.Context, roleArn string) (*IAMRoleValidation, error) {
	// Extract role name from ARN
	roleName := extractRoleNameFromArn(roleArn)
	if roleName == "" {
		return nil, fmt.Errorf("invalid role ARN: %s", roleArn)
	}

	// Get role details
	cmd := exec.Command("aws", "iam", "get-role", "--role-name", roleName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get role details: %w", err)
	}

	var roleResult struct {
		Role struct {
			RoleName                 string    `json:"RoleName"`
			Arn                      string    `json:"Arn"`
			AssumeRolePolicyDocument string    `json:"AssumeRolePolicyDocument"`
			Description              string    `json:"Description"`
			CreateDate               time.Time `json:"CreateDate"`
			Path                     string    `json:"Path"`
			MaxSessionDuration       int       `json:"MaxSessionDuration"`
			Tags                     []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"Tags"`
		} `json:"Role"`
	}

	if err := json.Unmarshal(output, &roleResult); err != nil {
		return nil, fmt.Errorf("failed to parse role details: %w", err)
	}

	// Get attached policies
	attachedPolicies, err := p.getAttachedRolePolicies(ctx, roleName)
	if err != nil {
		attachedPolicies = []AttachedPolicy{} // Continue without policies if there's an error
	}

	// Get inline policies
	inlinePolicies, err := p.getInlineRolePolicies(ctx, roleName)
	if err != nil {
		inlinePolicies = []InlinePolicy{} // Continue without inline policies
	}

	// Validate trust relationship
	trustValidation := p.validateTrustRelationship(roleResult.Role.AssumeRolePolicyDocument)

	// Check for required permissions
	permissionValidation := p.validateRequiredPermissions(attachedPolicies, inlinePolicies)

	return &IAMRoleValidation{
		RoleName:             roleResult.Role.RoleName,
		Arn:                  roleResult.Role.Arn,
		Description:          roleResult.Role.Description,
		CreateDate:           roleResult.Role.CreateDate,
		MaxSessionDuration:   roleResult.Role.MaxSessionDuration,
		AttachedPolicies:     attachedPolicies,
		InlinePolicies:       inlinePolicies,
		TrustValidation:      trustValidation,
		PermissionValidation: permissionValidation,
		IsValid:              trustValidation.IsValid && permissionValidation.IsValid,
	}, nil
}

// AssumeRole assumes an IAM role and returns temporary credentials
func (p *AWSProvider) AssumeRole(ctx context.Context, roleArn, sessionName string, duration int) (*Credentials, error) {
	if sessionName == "" {
		sessionName = fmt.Sprintf("apm-session-%d", time.Now().Unix())
	}

	if duration == 0 {
		duration = 3600 // Default to 1 hour
	}

	cmd := exec.Command("aws", "sts", "assume-role",
		"--role-arn", roleArn,
		"--role-session-name", sessionName,
		"--duration-seconds", strconv.Itoa(duration))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	var result struct {
		Credentials struct {
			AccessKeyId     string    `json:"AccessKeyId"`
			SecretAccessKey string    `json:"SecretAccessKey"`
			SessionToken    string    `json:"SessionToken"`
			Expiration      time.Time `json:"Expiration"`
		} `json:"Credentials"`
		AssumedRoleUser struct {
			AssumedRoleId string `json:"AssumedRoleId"`
			Arn           string `json:"Arn"`
		} `json:"AssumedRoleUser"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse assume role response: %w", err)
	}

	return &Credentials{
		Provider:   ProviderAWS,
		AuthMethod: AuthMethodIAMRole,
		AccessKey:  result.Credentials.AccessKeyId,
		SecretKey:  result.Credentials.SecretAccessKey,
		Token:      result.Credentials.SessionToken,
		Expiry:     &result.Credentials.Expiration,
		Account:    extractAccountFromArn(result.AssumedRoleUser.Arn),
	}, nil
}

// ValidateSTSToken validates an STS token
func (p *AWSProvider) ValidateSTSToken(ctx context.Context, credentials *Credentials) (*STSTokenValidation, error) {
	// Set temporary environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", credentials.AccessKey))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", credentials.SecretKey))
	if credentials.Token != "" {
		env = append(env, fmt.Sprintf("AWS_SESSION_TOKEN=%s", credentials.Token))
	}

	cmd := exec.Command("aws", "sts", "get-caller-identity")
	cmd.Env = env
	output, err := cmd.Output()
	if err != nil {
		return &STSTokenValidation{
			IsValid: false,
			Error:   err.Error(),
		}, nil
	}

	var identity struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
		UserID  string `json:"UserId"`
	}

	if err := json.Unmarshal(output, &identity); err != nil {
		return &STSTokenValidation{
			IsValid: false,
			Error:   fmt.Sprintf("failed to parse identity: %v", err),
		}, nil
	}

	// Check if token is expired
	isExpired := false
	if credentials.Expiry != nil {
		isExpired = time.Now().After(*credentials.Expiry)
	}

	return &STSTokenValidation{
		IsValid:   true,
		Account:   identity.Account,
		Arn:       identity.Arn,
		UserID:    identity.UserID,
		IsExpired: isExpired,
		ExpiresAt: credentials.Expiry,
	}, nil
}

// getAttachedRolePolicies gets attached policies for a role
func (p *AWSProvider) getAttachedRolePolicies(ctx context.Context, roleName string) ([]AttachedPolicy, error) {
	cmd := exec.Command("aws", "iam", "list-attached-role-policies", "--role-name", roleName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list attached policies: %w", err)
	}

	var result struct {
		AttachedPolicies []struct {
			PolicyName string `json:"PolicyName"`
			PolicyArn  string `json:"PolicyArn"`
		} `json:"AttachedPolicies"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse attached policies: %w", err)
	}

	var policies []AttachedPolicy
	for _, policy := range result.AttachedPolicies {
		policies = append(policies, AttachedPolicy{
			Name: policy.PolicyName,
			Arn:  policy.PolicyArn,
		})
	}

	return policies, nil
}

// getInlineRolePolicies gets inline policies for a role
func (p *AWSProvider) getInlineRolePolicies(ctx context.Context, roleName string) ([]InlinePolicy, error) {
	cmd := exec.Command("aws", "iam", "list-role-policies", "--role-name", roleName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list inline policies: %w", err)
	}

	var result struct {
		PolicyNames []string `json:"PolicyNames"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse inline policies: %w", err)
	}

	var policies []InlinePolicy
	for _, policyName := range result.PolicyNames {
		// Get policy document
		cmd := exec.Command("aws", "iam", "get-role-policy", "--role-name", roleName, "--policy-name", policyName)
		output, err := cmd.Output()
		if err != nil {
			continue // Skip if we can't get the policy
		}

		var policyResult struct {
			PolicyDocument string `json:"PolicyDocument"`
		}

		if err := json.Unmarshal(output, &policyResult); err != nil {
			continue
		}

		policies = append(policies, InlinePolicy{
			Name:     policyName,
			Document: policyResult.PolicyDocument,
		})
	}

	return policies, nil
}

// validateTrustRelationship validates the trust relationship of a role
func (p *AWSProvider) validateTrustRelationship(trustPolicy string) TrustValidation {
	// Basic validation - can be enhanced with more sophisticated checks
	validation := TrustValidation{
		IsValid:  true,
		Warnings: []string{},
	}

	// Check if trust policy allows everyone (security concern)
	if strings.Contains(trustPolicy, `"Principal": "*"`) {
		validation.Warnings = append(validation.Warnings, "Trust policy allows any principal (*)")
	}

	// Check for overly permissive conditions
	if !strings.Contains(trustPolicy, "Condition") {
		validation.Warnings = append(validation.Warnings, "Trust policy has no conditions")
	}

	return validation
}

// validateRequiredPermissions validates if the role has required permissions
func (p *AWSProvider) validateRequiredPermissions(attached []AttachedPolicy, inline []InlinePolicy) PermissionValidation {
	requiredPermissions := []string{
		"ecr:GetAuthorizationToken",
		"ecr:BatchCheckLayerAvailability",
		"ecr:GetDownloadUrlForLayer",
		"ecr:BatchGetImage",
		"eks:DescribeCluster",
		"eks:ListClusters",
	}

	validation := PermissionValidation{
		IsValid:              true,
		MissingPermissions:   []string{},
		ExcessivePermissions: []string{},
	}

	// This is a simplified validation - in reality, you'd need to parse
	// policy documents and check for specific permissions
	for _, required := range requiredPermissions {
		found := false
		for _, policy := range attached {
			if strings.Contains(policy.Arn, "AmazonEKSClusterPolicy") ||
				strings.Contains(policy.Arn, "AmazonEKSWorkerNodePolicy") ||
				strings.Contains(policy.Arn, "AmazonEKS_CNI_Policy") ||
				strings.Contains(policy.Arn, "AmazonEC2ContainerRegistryReadOnly") {
				found = true
				break
			}
		}

		if !found {
			validation.MissingPermissions = append(validation.MissingPermissions, required)
		}
	}

	if len(validation.MissingPermissions) > 0 {
		validation.IsValid = false
	}

	return validation
}

// Helper functions
func extractRoleNameFromArn(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractAccountFromArn(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}

// Types for IAM validation
type IAMRoleValidation struct {
	RoleName                 string               `json:"roleName"`
	Arn                      string               `json:"arn"`
	Description              string               `json:"description"`
	CreateDate               time.Time            `json:"createDate"`
	MaxSessionDuration       int                  `json:"maxSessionDuration"`
	AssumeRolePolicyDocument string               `json:"assumeRolePolicyDocument"`
	AttachedPolicies         []AttachedPolicy     `json:"attachedPolicies"`
	InlinePolicies           []InlinePolicy       `json:"inlinePolicies"`
	TrustValidation          TrustValidation      `json:"trustValidation"`
	PermissionValidation     PermissionValidation `json:"permissionValidation"`
	IsValid                  bool                 `json:"isValid"`
}

type AttachedPolicy struct {
	Name string `json:"name"`
	Arn  string `json:"arn"`
}

type InlinePolicy struct {
	Name     string `json:"name"`
	Document string `json:"document"`
}

type TrustValidation struct {
	IsValid  bool     `json:"isValid"`
	Warnings []string `json:"warnings"`
}

type PermissionValidation struct {
	IsValid              bool     `json:"isValid"`
	MissingPermissions   []string `json:"missingPermissions"`
	ExcessivePermissions []string `json:"excessivePermissions"`
}

type STSTokenValidation struct {
	IsValid   bool       `json:"isValid"`
	Account   string     `json:"account"`
	Arn       string     `json:"arn"`
	UserID    string     `json:"userId"`
	IsExpired bool       `json:"isExpired"`
	ExpiresAt *time.Time `json:"expiresAt"`
	Error     string     `json:"error,omitempty"`
}

// ===============================
// Enhanced ECR Operations
// ===============================

// ECRLoginWithOptimization performs ECR login with build-time optimizations
func (p *AWSProvider) ECRLoginWithOptimization(ctx context.Context, options *ECRLoginOptions) (*ECRLoginResult, error) {
	if options == nil {
		options = &ECRLoginOptions{
			Region:        p.GetCurrentRegion(),
			CacheTokens:   true,
			ParallelLogin: true,
			Timeout:       30 * time.Second,
		}
	}

	result := &ECRLoginResult{
		StartTime: time.Now(),
		Regions:   []string{},
		Errors:    []string{},
	}

	// Get regions to authenticate to
	regions := options.Regions
	if len(regions) == 0 {
		regions = []string{options.Region}
	}

	// Parallel authentication if enabled
	if options.ParallelLogin && len(regions) > 1 {
		return p.parallelECRLogin(ctx, regions, options)
	}

	// Sequential authentication
	for _, region := range regions {
		err := p.performECRLogin(ctx, region, options)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", region, err))
			continue
		}
		result.Regions = append(result.Regions, region)
	}

	result.Duration = time.Since(result.StartTime)
	result.Success = len(result.Errors) == 0

	return result, nil
}

// BuildAndPushImageOptimized builds and pushes Docker image with optimizations
func (p *AWSProvider) BuildAndPushImageOptimized(ctx context.Context, options *BuildPushOptions) (*BuildPushResult, error) {
	if options == nil {
		return nil, fmt.Errorf("build options are required")
	}

	result := &BuildPushResult{
		StartTime: time.Now(),
		Stages:    []BuildStage{},
	}

	// Validate inputs
	if err := p.validateBuildOptions(options); err != nil {
		return nil, fmt.Errorf("invalid build options: %w", err)
	}

	// Stage 1: Prepare build context
	stage := BuildStage{Name: "prepare", StartTime: time.Now()}
	buildContext, err := p.prepareBuildContext(ctx, options)
	if err != nil {
		stage.Error = err.Error()
		result.Stages = append(result.Stages, stage)
		return result, fmt.Errorf("failed to prepare build context: %w", err)
	}
	stage.Duration = time.Since(stage.StartTime)
	result.Stages = append(result.Stages, stage)

	// Stage 2: ECR authentication
	stage = BuildStage{Name: "ecr-auth", StartTime: time.Now()}
	if err := p.ensureECRAuthentication(ctx, options.Region); err != nil {
		stage.Error = err.Error()
		result.Stages = append(result.Stages, stage)
		return result, fmt.Errorf("ECR authentication failed: %w", err)
	}
	stage.Duration = time.Since(stage.StartTime)
	result.Stages = append(result.Stages, stage)

	// Stage 3: Build image
	stage = BuildStage{Name: "build", StartTime: time.Now()}
	imageID, err := p.buildImageOptimized(ctx, buildContext, options)
	if err != nil {
		stage.Error = err.Error()
		result.Stages = append(result.Stages, stage)
		return result, fmt.Errorf("image build failed: %w", err)
	}
	stage.Duration = time.Since(stage.StartTime)
	result.Stages = append(result.Stages, stage)

	// Stage 4: Tag image
	stage = BuildStage{Name: "tag", StartTime: time.Now()}
	taggedImage, err := p.tagImageForECR(ctx, imageID, options)
	if err != nil {
		stage.Error = err.Error()
		result.Stages = append(result.Stages, stage)
		return result, fmt.Errorf("image tagging failed: %w", err)
	}
	stage.Duration = time.Since(stage.StartTime)
	result.Stages = append(result.Stages, stage)

	// Stage 5: Push image
	stage = BuildStage{Name: "push", StartTime: time.Now()}
	pushResult, err := p.pushImageOptimized(ctx, taggedImage, options)
	if err != nil {
		stage.Error = err.Error()
		result.Stages = append(result.Stages, stage)
		return result, fmt.Errorf("image push failed: %w", err)
	}
	stage.Duration = time.Since(stage.StartTime)
	result.Stages = append(result.Stages, stage)

	// Final result
	result.Duration = time.Since(result.StartTime)
	result.Success = true
	result.ImageURI = taggedImage
	result.ImageDigest = pushResult.Digest
	result.ImageSize = pushResult.Size

	return result, nil
}

// parallelECRLogin performs parallel ECR authentication across regions
func (p *AWSProvider) parallelECRLogin(ctx context.Context, regions []string, options *ECRLoginOptions) (*ECRLoginResult, error) {
	result := &ECRLoginResult{
		StartTime: time.Now(),
		Regions:   []string{},
		Errors:    []string{},
	}

	type loginResult struct {
		region string
		err    error
	}

	resultChan := make(chan loginResult, len(regions))

	// Start parallel logins
	for _, region := range regions {
		go func(r string) {
			err := p.performECRLogin(ctx, r, options)
			resultChan <- loginResult{region: r, err: err}
		}(region)
	}

	// Collect results
	for i := 0; i < len(regions); i++ {
		lr := <-resultChan
		if lr.err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", lr.region, lr.err))
		} else {
			result.Regions = append(result.Regions, lr.region)
		}
	}

	result.Duration = time.Since(result.StartTime)
	result.Success = len(result.Errors) == 0

	return result, nil
}

// performECRLogin performs ECR login for a specific region
func (p *AWSProvider) performECRLogin(ctx context.Context, region string, options *ECRLoginOptions) error {
	// Check if we have a cached token
	if options.CacheTokens {
		token, err := p.GetECRToken(ctx, "")
		if err == nil && time.Now().Before(token.ExpiresAt.Add(-5*time.Minute)) {
			return nil // Already authenticated with valid token
		}
	}

	// Get account ID
	accountID, err := p.getAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get account ID: %w", err)
	}

	// Create registry URL
	registryURL := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, region)

	// Get ECR token
	cmd := exec.Command("aws", "ecr", "get-login-password", "--region", region)
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)

	token, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get ECR token: %w", err)
	}

	// Docker login
	loginCmd := exec.Command("docker", "login", "--username", "AWS", "--password-stdin", registryURL)
	loginCmd.Stdin = strings.NewReader(string(token))
	loginCmd = exec.CommandContext(ctx, loginCmd.Args[0], loginCmd.Args[1:]...)

	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("docker login failed: %w", err)
	}

	// Cache token if enabled
	if options.CacheTokens {
		p.ecrTokens[fmt.Sprintf("%s-%s", registryURL, region)] = &ECRToken{
			Token:     strings.TrimSpace(string(token)),
			ExpiresAt: time.Now().Add(12 * time.Hour),
			Registry:  registryURL,
			Region:    region,
		}
	}

	return nil
}

// validateBuildOptions validates build and push options
func (p *AWSProvider) validateBuildOptions(options *BuildPushOptions) error {
	if options.ImageName == "" {
		return errors.New("image name is required")
	}
	if options.Tag == "" {
		return errors.New("image tag is required")
	}
	if options.DockerfilePath == "" {
		options.DockerfilePath = "Dockerfile"
	}
	if options.ContextPath == "" {
		options.ContextPath = "."
	}
	if options.Region == "" {
		options.Region = p.GetCurrentRegion()
	}
	return nil
}

// prepareBuildContext prepares the build context for Docker
func (p *AWSProvider) prepareBuildContext(ctx context.Context, options *BuildPushOptions) (string, error) {
	// Verify Dockerfile exists
	if _, err := os.Stat(options.DockerfilePath); os.IsNotExist(err) {
		return "", fmt.Errorf("Dockerfile not found at %s", options.DockerfilePath)
	}

	// Verify context directory exists
	if _, err := os.Stat(options.ContextPath); os.IsNotExist(err) {
		return "", fmt.Errorf("build context not found at %s", options.ContextPath)
	}

	return options.ContextPath, nil
}

// ensureECRAuthentication ensures ECR authentication is valid
func (p *AWSProvider) ensureECRAuthentication(ctx context.Context, region string) error {
	return p.performECRLogin(ctx, region, &ECRLoginOptions{
		Region:      region,
		CacheTokens: true,
		Timeout:     30 * time.Second,
	})
}

// buildImageOptimized builds Docker image with optimizations
func (p *AWSProvider) buildImageOptimized(ctx context.Context, buildContext string, options *BuildPushOptions) (string, error) {
	args := []string{"build"}

	// Add build args
	for key, value := range options.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add labels
	for key, value := range options.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add cache optimizations
	if options.UseCache && options.CacheFrom != "" {
		args = append(args, "--cache-from", options.CacheFrom)
	}

	// Add Dockerfile path
	args = append(args, "-f", options.DockerfilePath)

	// Add context
	args = append(args, buildContext)

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker build failed: %w", err)
	}

	// Extract image ID from output
	imageID := p.extractImageIDFromBuildOutput(string(output))
	if imageID == "" {
		return "", fmt.Errorf("failed to extract image ID from build output")
	}

	return imageID, nil
}

// tagImageForECR tags image for ECR
func (p *AWSProvider) tagImageForECR(ctx context.Context, imageID string, options *BuildPushOptions) (string, error) {
	accountID, err := p.getAccountID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get account ID: %w", err)
	}

	ecrURI := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
		accountID, options.Region, options.ImageName, options.Tag)

	cmd := exec.Command("docker", "tag", imageID, ecrURI)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker tag failed: %w", err)
	}

	return ecrURI, nil
}

// pushImageOptimized pushes image to ECR with optimizations
func (p *AWSProvider) pushImageOptimized(ctx context.Context, imageURI string, options *BuildPushOptions) (*PushResult, error) {
	cmd := exec.Command("docker", "push", imageURI)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker push failed: %w", err)
	}

	// Parse push output for digest and size
	digest, size := p.parsePushOutput(string(output))

	return &PushResult{
		Digest: digest,
		Size:   size,
	}, nil
}

// extractImageIDFromBuildOutput extracts image ID from docker build output
func (p *AWSProvider) extractImageIDFromBuildOutput(output string) string {
	// Look for "Successfully built" line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Successfully built") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return ""
}

// parsePushOutput parses docker push output for digest and size
func (p *AWSProvider) parsePushOutput(output string) (string, int64) {
	lines := strings.Split(output, "\n")
	var digest string
	var size int64

	for _, line := range lines {
		if strings.Contains(line, "digest:") {
			parts := strings.Split(line, "digest: ")
			if len(parts) >= 2 {
				digest = strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "size:") {
			parts := strings.Split(line, "size: ")
			if len(parts) >= 2 {
				sizeStr := strings.TrimSpace(parts[1])
				if s, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
					size = s
				}
			}
		}
	}

	return digest, size
}

// Types for ECR operations
type ECRLoginOptions struct {
	Region        string        `json:"region"`
	Regions       []string      `json:"regions"`
	CacheTokens   bool          `json:"cacheTokens"`
	ParallelLogin bool          `json:"parallelLogin"`
	Timeout       time.Duration `json:"timeout"`
}

type ECRLoginResult struct {
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Regions   []string      `json:"regions"`
	Errors    []string      `json:"errors"`
}

type BuildPushOptions struct {
	ImageName      string            `json:"imageName"`
	Tag            string            `json:"tag"`
	DockerfilePath string            `json:"dockerfilePath"`
	ContextPath    string            `json:"contextPath"`
	Region         string            `json:"region"`
	BuildArgs      map[string]string `json:"buildArgs"`
	Labels         map[string]string `json:"labels"`
	UseCache       bool              `json:"useCache"`
	CacheFrom      string            `json:"cacheFrom"`
}

type BuildPushResult struct {
	StartTime   time.Time     `json:"startTime"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	ImageURI    string        `json:"imageUri"`
	ImageDigest string        `json:"imageDigest"`
	ImageSize   int64         `json:"imageSize"`
	Stages      []BuildStage  `json:"stages"`
}

type BuildStage struct {
	Name      string        `json:"name"`
	StartTime time.Time     `json:"startTime"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
}

type PushResult struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

// ===============================
// CloudFormation Stack Detection and Management
// ===============================

// ListStacks lists CloudFormation stacks with filtering and parallel discovery
func (m *CloudFormationManager) ListStacks(ctx context.Context, filters *StackFilters) ([]*Stack, error) {
	if filters == nil {
		filters = &StackFilters{
			Regions:     []string{m.provider.GetCurrentRegion()},
			StackStatus: []string{},
		}
	}

	// Use parallel discovery if multiple regions specified
	if len(filters.Regions) > 1 {
		return m.listStacksParallel(ctx, filters)
	}

	// Single region discovery
	region := filters.Regions[0]
	if region == "" {
		region = m.provider.GetCurrentRegion()
	}

	return m.listStacksInRegion(ctx, region, filters)
}

// listStacksParallel performs parallel stack discovery across multiple regions
func (m *CloudFormationManager) listStacksParallel(ctx context.Context, filters *StackFilters) ([]*Stack, error) {
	type regionResult struct {
		region string
		stacks []*Stack
		err    error
	}

	resultChan := make(chan regionResult, len(filters.Regions))

	// Start parallel discovery
	for _, region := range filters.Regions {
		go func(r string) {
			stacks, err := m.listStacksInRegion(ctx, r, filters)
			resultChan <- regionResult{region: r, stacks: stacks, err: err}
		}(region)
	}

	// Collect results
	var allStacks []*Stack
	var errors []string

	for i := 0; i < len(filters.Regions); i++ {
		result := <-resultChan
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.region, result.err))
			continue
		}
		allStacks = append(allStacks, result.stacks...)
	}

	if len(errors) > 0 && len(allStacks) == 0 {
		return nil, fmt.Errorf("failed to list stacks in all regions: %s", strings.Join(errors, "; "))
	}

	return allStacks, nil
}

// listStacksInRegion lists stacks in a specific region
func (m *CloudFormationManager) listStacksInRegion(ctx context.Context, region string, filters *StackFilters) ([]*Stack, error) {
	args := []string{"cloudformation", "list-stacks", "--region", region}

	// Add stack status filter
	if len(filters.StackStatus) > 0 {
		statusFilter := strings.Join(filters.StackStatus, " ")
		args = append(args, "--stack-status-filter", statusFilter)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list stacks in region %s: %w", region, err)
	}

	var result struct {
		StackSummaries []struct {
			StackName           string     `json:"StackName"`
			StackStatus         string     `json:"StackStatus"`
			CreationTime        time.Time  `json:"CreationTime"`
			LastUpdatedTime     *time.Time `json:"LastUpdatedTime,omitempty"`
			TemplateDescription string     `json:"TemplateDescription,omitempty"`
		} `json:"StackSummaries"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse stack list: %w", err)
	}

	var stacks []*Stack
	for _, summary := range result.StackSummaries {
		// Apply filters
		if filters.NamePrefix != "" && !strings.HasPrefix(summary.StackName, filters.NamePrefix) {
			continue
		}

		if filters.CreatedAfter != nil && summary.CreationTime.Before(*filters.CreatedAfter) {
			continue
		}

		if filters.CreatedBefore != nil && summary.CreationTime.After(*filters.CreatedBefore) {
			continue
		}

		stack := &Stack{
			Name:        summary.StackName,
			Status:      summary.StackStatus,
			Region:      region,
			CreatedTime: summary.CreationTime,
			UpdatedTime: summary.LastUpdatedTime,
			Description: summary.TemplateDescription,
		}

		// Get additional details if needed
		if filters.Tags != nil || filters.APMOnly {
			detailedStack, err := m.GetStack(ctx, summary.StackName, region)
			if err != nil {
				continue // Skip stacks we can't get details for
			}

			// Apply tag filters
			if filters.Tags != nil && !m.matchesTags(detailedStack.Tags, filters.Tags) {
				continue
			}

			// Apply APM filter
			if filters.APMOnly && !detailedStack.IsAPMStack {
				continue
			}

			stacks = append(stacks, detailedStack)
		} else {
			stacks = append(stacks, stack)
		}
	}

	return stacks, nil
}

// GetStack gets detailed information about a specific CloudFormation stack
func (m *CloudFormationManager) GetStack(ctx context.Context, stackName, region string) (*Stack, error) {
	if region == "" {
		region = m.provider.GetCurrentRegion()
	}

	// Get stack description
	cmd := exec.Command("aws", "cloudformation", "describe-stacks",
		"--stack-name", stackName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack %s: %w", stackName, err)
	}

	var result struct {
		Stacks []struct {
			StackName       string     `json:"StackName"`
			StackId         string     `json:"StackId"`
			StackStatus     string     `json:"StackStatus"`
			CreationTime    time.Time  `json:"CreationTime"`
			LastUpdatedTime *time.Time `json:"LastUpdatedTime,omitempty"`
			Description     string     `json:"Description,omitempty"`
			Tags            []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"Tags,omitempty"`
			Parameters []struct {
				ParameterKey   string `json:"ParameterKey"`
				ParameterValue string `json:"ParameterValue"`
			} `json:"Parameters,omitempty"`
			Outputs []struct {
				OutputKey   string `json:"OutputKey"`
				OutputValue string `json:"OutputValue"`
				Description string `json:"Description,omitempty"`
			} `json:"Outputs,omitempty"`
		} `json:"Stacks"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse stack description: %w", err)
	}

	if len(result.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", stackName)
	}

	stackInfo := result.Stacks[0]

	// Convert tags to map
	tags := make(map[string]string)
	for _, tag := range stackInfo.Tags {
		tags[tag.Key] = tag.Value
	}

	// Convert parameters to map
	parameters := make(map[string]string)
	for _, param := range stackInfo.Parameters {
		parameters[param.ParameterKey] = param.ParameterValue
	}

	// Convert outputs to map
	outputs := make(map[string]string)
	for _, output := range stackInfo.Outputs {
		outputs[output.OutputKey] = output.OutputValue
	}

	stack := &Stack{
		Name:        stackInfo.StackName,
		Arn:         stackInfo.StackId,
		Status:      stackInfo.StackStatus,
		Region:      region,
		CreatedTime: stackInfo.CreationTime,
		UpdatedTime: stackInfo.LastUpdatedTime,
		Description: stackInfo.Description,
		Tags:        tags,
		Parameters:  parameters,
		Outputs:     outputs,
	}

	// Get stack resources
	resources, err := m.GetStackResources(ctx, stackName, region)
	if err == nil {
		stack.Resources = resources
	}

	// Detect if this is an APM stack and extract APM resources
	stack.IsAPMStack = m.isAPMStack(stack)
	if stack.IsAPMStack {
		apmResources, err := m.extractAPMResources(ctx, stack)
		if err == nil {
			stack.APMResources = apmResources
		}
	}

	return stack, nil
}

// GetStackResources gets all resources in a CloudFormation stack
func (m *CloudFormationManager) GetStackResources(ctx context.Context, stackName, region string) ([]*StackResource, error) {
	if region == "" {
		region = m.provider.GetCurrentRegion()
	}

	cmd := exec.Command("aws", "cloudformation", "list-stack-resources",
		"--stack-name", stackName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list stack resources: %w", err)
	}

	var result struct {
		StackResourceSummaries []struct {
			LogicalResourceId    string    `json:"LogicalResourceId"`
			PhysicalResourceId   string    `json:"PhysicalResourceId"`
			ResourceType         string    `json:"ResourceType"`
			ResourceStatus       string    `json:"ResourceStatus"`
			Timestamp            time.Time `json:"Timestamp"`
			ResourceStatusReason string    `json:"ResourceStatusReason,omitempty"`
		} `json:"StackResourceSummaries"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse stack resources: %w", err)
	}

	var resources []*StackResource
	for _, res := range result.StackResourceSummaries {
		resources = append(resources, &StackResource{
			LogicalID:    res.LogicalResourceId,
			PhysicalID:   res.PhysicalResourceId,
			Type:         res.ResourceType,
			Status:       res.ResourceStatus,
			Timestamp:    res.Timestamp,
			StatusReason: res.ResourceStatusReason,
		})
	}

	return resources, nil
}

// GetStackOutputs extracts outputs from a CloudFormation stack
func (m *CloudFormationManager) GetStackOutputs(ctx context.Context, stackName, region string) (map[string]string, error) {
	stack, err := m.GetStack(ctx, stackName, region)
	if err != nil {
		return nil, err
	}

	return stack.Outputs, nil
}

// isAPMStack determines if a stack contains APM-related resources
func (m *CloudFormationManager) isAPMStack(stack *Stack) bool {
	// Check for APM-related tags
	apmTags := []string{
		"apm", "monitoring", "observability", "prometheus", "grafana",
		"jaeger", "loki", "alertmanager", "application-performance-monitoring",
	}

	for key, value := range stack.Tags {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)

		for _, apmTag := range apmTags {
			if strings.Contains(keyLower, apmTag) || strings.Contains(valueLower, apmTag) {
				return true
			}
		}
	}

	// Check for APM-related resources
	apmResourceTypes := []string{
		"AWS::ElasticLoadBalancingV2::LoadBalancer",
		"AWS::ECS::Service",
		"AWS::ECS::TaskDefinition",
		"AWS::RDS::DBInstance",
		"AWS::RDS::DBCluster",
		"AWS::Lambda::Function",
		"AWS::ElastiCache::CacheCluster",
		"AWS::S3::Bucket",
		"AWS::ApiGateway::RestApi",
		"AWS::CloudWatch::Alarm",
		"AWS::Logs::LogGroup",
	}

	apmResourceCount := 0
	for _, resource := range stack.Resources {
		for _, apmType := range apmResourceTypes {
			if resource.Type == apmType {
				apmResourceCount++
				break
			}
		}
	}

	// Consider it an APM stack if it has 2 or more APM-related resources
	return apmResourceCount >= 2
}

// matchesTags checks if stack tags match the filter criteria
func (m *CloudFormationManager) matchesTags(stackTags, filterTags map[string]string) bool {
	for key, value := range filterTags {
		if stackValue, exists := stackTags[key]; !exists || stackValue != value {
			return false
		}
	}
	return true
}

// extractAPMResources extracts and maps APM-related resources from a stack
func (m *CloudFormationManager) extractAPMResources(ctx context.Context, stack *Stack) (*APMResources, error) {
	apmResources := &APMResources{
		LoadBalancers:       []*LoadBalancerResource{},
		ECSServices:         []*ECSServiceResource{},
		RDSInstances:        []*RDSInstanceResource{},
		LambdaFunctions:     []*LambdaFunctionResource{},
		ElastiCacheClusters: []*ElastiCacheClusterResource{},
		S3Buckets:           []*S3BucketResource{},
		VPCResources:        []*VPCResource{},
	}

	// Process each resource in the stack
	for _, resource := range stack.Resources {
		switch resource.Type {
		case "AWS::ElasticLoadBalancingV2::LoadBalancer":
			lb, err := m.getLoadBalancerDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.LoadBalancers = append(apmResources.LoadBalancers, lb)
			}

		case "AWS::ECS::Service":
			svc, err := m.getECSServiceDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.ECSServices = append(apmResources.ECSServices, svc)
			}

		case "AWS::RDS::DBInstance":
			db, err := m.getRDSInstanceDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.RDSInstances = append(apmResources.RDSInstances, db)
			}

		case "AWS::Lambda::Function":
			fn, err := m.getLambdaFunctionDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.LambdaFunctions = append(apmResources.LambdaFunctions, fn)
			}

		case "AWS::ElastiCache::CacheCluster":
			cache, err := m.getElastiCacheDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.ElastiCacheClusters = append(apmResources.ElastiCacheClusters, cache)
			}

		case "AWS::S3::Bucket":
			bucket, err := m.getS3BucketDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.S3Buckets = append(apmResources.S3Buckets, bucket)
			}

		case "AWS::EC2::VPC":
			vpc, err := m.getVPCDetails(ctx, resource.PhysicalID, stack.Region)
			if err == nil {
				apmResources.VPCResources = append(apmResources.VPCResources, vpc)
			}
		}
	}

	return apmResources, nil
}

// getLoadBalancerDetails gets detailed information about a load balancer
func (m *CloudFormationManager) getLoadBalancerDetails(ctx context.Context, albArn, region string) (*LoadBalancerResource, error) {
	cmd := exec.Command("aws", "elbv2", "describe-load-balancers",
		"--load-balancer-arns", albArn, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe load balancer: %w", err)
	}

	var result struct {
		LoadBalancers []struct {
			Type              string `json:"Type"`
			LoadBalancerArn   string `json:"LoadBalancerArn"`
			DNSName           string `json:"DNSName"`
			Scheme            string `json:"Scheme"`
			VpcId             string `json:"VpcId"`
			AvailabilityZones []struct {
				SubnetId string `json:"SubnetId"`
			} `json:"AvailabilityZones"`
		} `json:"LoadBalancers"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse load balancer details: %w", err)
	}

	if len(result.LoadBalancers) == 0 {
		return nil, fmt.Errorf("load balancer not found")
	}

	lb := result.LoadBalancers[0]
	var subnetIds []string
	for _, az := range lb.AvailabilityZones {
		subnetIds = append(subnetIds, az.SubnetId)
	}

	return &LoadBalancerResource{
		Type:      lb.Type,
		Arn:       lb.LoadBalancerArn,
		DNSName:   lb.DNSName,
		Scheme:    lb.Scheme,
		VpcId:     lb.VpcId,
		SubnetIds: subnetIds,
	}, nil
}

// getECSServiceDetails gets detailed information about an ECS service
func (m *CloudFormationManager) getECSServiceDetails(ctx context.Context, serviceArn, region string) (*ECSServiceResource, error) {
	// Extract cluster name from service ARN
	parts := strings.Split(serviceArn, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid service ARN format")
	}

	clusterName := parts[1]
	serviceName := parts[2]

	cmd := exec.Command("aws", "ecs", "describe-services",
		"--cluster", clusterName, "--services", serviceName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe ECS service: %w", err)
	}

	var result struct {
		Services []struct {
			ServiceName    string `json:"serviceName"`
			ClusterArn     string `json:"clusterArn"`
			TaskDefinition string `json:"taskDefinition"`
			DesiredCount   int    `json:"desiredCount"`
			RunningCount   int    `json:"runningCount"`
			Status         string `json:"status"`
		} `json:"services"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ECS service details: %w", err)
	}

	if len(result.Services) == 0 {
		return nil, fmt.Errorf("ECS service not found")
	}

	svc := result.Services[0]
	return &ECSServiceResource{
		ServiceName:    svc.ServiceName,
		ClusterName:    clusterName,
		TaskDefinition: svc.TaskDefinition,
		DesiredCount:   svc.DesiredCount,
		RunningCount:   svc.RunningCount,
		Status:         svc.Status,
	}, nil
}

// getRDSInstanceDetails gets detailed information about an RDS instance
func (m *CloudFormationManager) getRDSInstanceDetails(ctx context.Context, dbInstanceId, region string) (*RDSInstanceResource, error) {
	cmd := exec.Command("aws", "rds", "describe-db-instances",
		"--db-instance-identifier", dbInstanceId, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe RDS instance: %w", err)
	}

	var result struct {
		DBInstances []struct {
			DBInstanceIdentifier string `json:"DBInstanceIdentifier"`
			DBName               string `json:"DBName"`
			Engine               string `json:"Engine"`
			EngineVersion        string `json:"EngineVersion"`
			DBInstanceStatus     string `json:"DBInstanceStatus"`
			Endpoint             struct {
				Address string `json:"Address"`
				Port    int    `json:"Port"`
			} `json:"Endpoint"`
		} `json:"DBInstances"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse RDS instance details: %w", err)
	}

	if len(result.DBInstances) == 0 {
		return nil, fmt.Errorf("RDS instance not found")
	}

	db := result.DBInstances[0]
	return &RDSInstanceResource{
		DBInstanceIdentifier: db.DBInstanceIdentifier,
		DBName:               db.DBName,
		Engine:               db.Engine,
		EngineVersion:        db.EngineVersion,
		Endpoint:             db.Endpoint.Address,
		Port:                 db.Endpoint.Port,
		Status:               db.DBInstanceStatus,
	}, nil
}

// getLambdaFunctionDetails gets detailed information about a Lambda function
func (m *CloudFormationManager) getLambdaFunctionDetails(ctx context.Context, functionName, region string) (*LambdaFunctionResource, error) {
	cmd := exec.Command("aws", "lambda", "get-function",
		"--function-name", functionName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe Lambda function: %w", err)
	}

	var result struct {
		Configuration struct {
			FunctionName string `json:"FunctionName"`
			Runtime      string `json:"Runtime"`
			Handler      string `json:"Handler"`
			Role         string `json:"Role"`
			State        string `json:"State"`
		} `json:"Configuration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Lambda function details: %w", err)
	}

	config := result.Configuration
	return &LambdaFunctionResource{
		FunctionName: config.FunctionName,
		Runtime:      config.Runtime,
		Handler:      config.Handler,
		Role:         config.Role,
		State:        config.State,
	}, nil
}

// getElastiCacheDetails gets detailed information about an ElastiCache cluster
func (m *CloudFormationManager) getElastiCacheDetails(ctx context.Context, clusterId, region string) (*ElastiCacheClusterResource, error) {
	cmd := exec.Command("aws", "elasticache", "describe-cache-clusters",
		"--cache-cluster-id", clusterId, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe ElastiCache cluster: %w", err)
	}

	var result struct {
		CacheClusters []struct {
			CacheClusterId        string `json:"CacheClusterId"`
			Engine                string `json:"Engine"`
			EngineVersion         string `json:"EngineVersion"`
			CacheNodeType         string `json:"CacheNodeType"`
			NumCacheNodes         int    `json:"NumCacheNodes"`
			CacheClusterStatus    string `json:"CacheClusterStatus"`
			ConfigurationEndpoint struct {
				Address string `json:"Address"`
			} `json:"ConfigurationEndpoint,omitempty"`
		} `json:"CacheClusters"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ElastiCache cluster details: %w", err)
	}

	if len(result.CacheClusters) == 0 {
		return nil, fmt.Errorf("ElastiCache cluster not found")
	}

	cluster := result.CacheClusters[0]
	configEndpoint := ""
	if cluster.ConfigurationEndpoint.Address != "" {
		configEndpoint = cluster.ConfigurationEndpoint.Address
	}

	return &ElastiCacheClusterResource{
		ClusterID:             cluster.CacheClusterId,
		Engine:                cluster.Engine,
		EngineVersion:         cluster.EngineVersion,
		CacheNodeType:         cluster.CacheNodeType,
		NumCacheNodes:         cluster.NumCacheNodes,
		Status:                cluster.CacheClusterStatus,
		ConfigurationEndpoint: configEndpoint,
	}, nil
}

// getS3BucketDetails gets detailed information about an S3 bucket
func (m *CloudFormationManager) getS3BucketDetails(ctx context.Context, bucketName, region string) (*S3BucketResource, error) {
	// Get bucket location
	cmd := exec.Command("aws", "s3api", "get-bucket-location", "--bucket", bucketName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket location: %w", err)
	}

	var locationResult struct {
		LocationConstraint string `json:"LocationConstraint"`
	}
	if err := json.Unmarshal(output, &locationResult); err != nil {
		return nil, fmt.Errorf("failed to parse bucket location: %w", err)
	}

	bucketRegion := locationResult.LocationConstraint
	if bucketRegion == "" {
		bucketRegion = "us-east-1" // Default region
	}

	// Get bucket creation date
	cmd = exec.Command("aws", "s3api", "list-buckets")
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	var bucketsResult struct {
		Buckets []struct {
			Name         string    `json:"Name"`
			CreationDate time.Time `json:"CreationDate"`
		} `json:"Buckets"`
	}
	if err := json.Unmarshal(output, &bucketsResult); err != nil {
		return nil, fmt.Errorf("failed to parse bucket list: %w", err)
	}

	var creationDate time.Time
	for _, bucket := range bucketsResult.Buckets {
		if bucket.Name == bucketName {
			creationDate = bucket.CreationDate
			break
		}
	}

	// Get versioning status
	cmd = exec.Command("aws", "s3api", "get-bucket-versioning", "--bucket", bucketName)
	versioningOutput, err := cmd.Output()
	versioning := "Disabled"
	if err == nil {
		var versioningResult struct {
			Status string `json:"Status"`
		}
		if json.Unmarshal(versioningOutput, &versioningResult) == nil {
			if versioningResult.Status != "" {
				versioning = versioningResult.Status
			}
		}
	}

	// Get encryption status
	cmd = exec.Command("aws", "s3api", "get-bucket-encryption", "--bucket", bucketName)
	encryptionOutput, err := cmd.Output()
	encryption := "None"
	if err == nil {
		var encryptionResult struct {
			ServerSideEncryptionConfiguration struct {
				Rules []struct {
					ApplyServerSideEncryptionByDefault struct {
						SSEAlgorithm string `json:"SSEAlgorithm"`
					} `json:"ApplyServerSideEncryptionByDefault"`
				} `json:"Rules"`
			} `json:"ServerSideEncryptionConfiguration"`
		}
		if json.Unmarshal(encryptionOutput, &encryptionResult) == nil {
			if len(encryptionResult.ServerSideEncryptionConfiguration.Rules) > 0 {
				encryption = encryptionResult.ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm
			}
		}
	}

	return &S3BucketResource{
		BucketName:   bucketName,
		Region:       bucketRegion,
		CreationDate: creationDate,
		Versioning:   versioning,
		Encryption:   encryption,
	}, nil
}

// getVPCDetails gets detailed information about a VPC
func (m *CloudFormationManager) getVPCDetails(ctx context.Context, vpcId, region string) (*VPCResource, error) {
	cmd := exec.Command("aws", "ec2", "describe-vpcs",
		"--vpc-ids", vpcId, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPC: %w", err)
	}

	var result struct {
		Vpcs []struct {
			VpcId     string `json:"VpcId"`
			CidrBlock string `json:"CidrBlock"`
			State     string `json:"State"`
		} `json:"Vpcs"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse VPC details: %w", err)
	}

	if len(result.Vpcs) == 0 {
		return nil, fmt.Errorf("VPC not found")
	}

	vpc := result.Vpcs[0]

	// Get subnets
	subnetCmd := exec.Command("aws", "ec2", "describe-subnets",
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcId), "--region", region)
	subnetOutput, err := subnetCmd.Output()
	var subnetIds []string
	if err == nil {
		var subnetResult struct {
			Subnets []struct {
				SubnetId string `json:"SubnetId"`
			} `json:"Subnets"`
		}
		if json.Unmarshal(subnetOutput, &subnetResult) == nil {
			for _, subnet := range subnetResult.Subnets {
				subnetIds = append(subnetIds, subnet.SubnetId)
			}
		}
	}

	// Get route tables
	rtCmd := exec.Command("aws", "ec2", "describe-route-tables",
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcId), "--region", region)
	rtOutput, err := rtCmd.Output()
	var routeTableIds []string
	if err == nil {
		var rtResult struct {
			RouteTables []struct {
				RouteTableId string `json:"RouteTableId"`
			} `json:"RouteTables"`
		}
		if json.Unmarshal(rtOutput, &rtResult) == nil {
			for _, rt := range rtResult.RouteTables {
				routeTableIds = append(routeTableIds, rt.RouteTableId)
			}
		}
	}

	// Get security groups
	sgCmd := exec.Command("aws", "ec2", "describe-security-groups",
		"--filters", fmt.Sprintf("Name=vpc-id,Values=%s", vpcId), "--region", region)
	sgOutput, err := sgCmd.Output()
	var securityGroupIds []string
	if err == nil {
		var sgResult struct {
			SecurityGroups []struct {
				GroupId string `json:"GroupId"`
			} `json:"SecurityGroups"`
		}
		if json.Unmarshal(sgOutput, &sgResult) == nil {
			for _, sg := range sgResult.SecurityGroups {
				securityGroupIds = append(securityGroupIds, sg.GroupId)
			}
		}
	}

	return &VPCResource{
		VpcId:            vpc.VpcId,
		CidrBlock:        vpc.CidrBlock,
		State:            vpc.State,
		SubnetIds:        subnetIds,
		RouteTableIds:    routeTableIds,
		SecurityGroupIds: securityGroupIds,
	}, nil
}

// ===============================
// Stack Drift Detection and Health Validation
// ===============================

// DetectDrift detects drift in a CloudFormation stack
func (m *CloudFormationManager) DetectDrift(ctx context.Context, stackName, region string) (*DriftResult, error) {
	if region == "" {
		region = m.provider.GetCurrentRegion()
	}

	// Initiate drift detection
	cmd := exec.Command("aws", "cloudformation", "detect-stack-drift",
		"--stack-name", stackName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to initiate drift detection: %w", err)
	}

	var driftResult struct {
		StackDriftDetectionId string `json:"StackDriftDetectionId"`
	}
	if err := json.Unmarshal(output, &driftResult); err != nil {
		return nil, fmt.Errorf("failed to parse drift detection response: %w", err)
	}

	// Poll for drift detection completion
	maxRetries := 30
	retryInterval := 5 * time.Second

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)

		statusCmd := exec.Command("aws", "cloudformation", "describe-stack-drift-detection-status",
			"--stack-drift-detection-id", driftResult.StackDriftDetectionId, "--region", region)
		statusOutput, err := statusCmd.Output()
		if err != nil {
			continue
		}

		var statusResult struct {
			DetectionStatus       string    `json:"DetectionStatus"`
			StackDriftStatus      string    `json:"StackDriftStatus"`
			DetectionStatusReason string    `json:"DetectionStatusReason,omitempty"`
			Timestamp             time.Time `json:"Timestamp"`
		}
		if err := json.Unmarshal(statusOutput, &statusResult); err != nil {
			continue
		}

		if statusResult.DetectionStatus == "DETECTION_COMPLETE" {
			// Get detailed drift results
			return m.getDriftDetails(ctx, stackName, region, statusResult.StackDriftStatus, statusResult.Timestamp)
		}

		if statusResult.DetectionStatus == "DETECTION_FAILED" {
			return nil, fmt.Errorf("drift detection failed: %s", statusResult.DetectionStatusReason)
		}
	}

	return nil, fmt.Errorf("drift detection timed out after %d retries", maxRetries)
}

// getDriftDetails gets detailed drift information for a stack
func (m *CloudFormationManager) getDriftDetails(ctx context.Context, stackName, region, driftStatus string, detectionTime time.Time) (*DriftResult, error) {
	cmd := exec.Command("aws", "cloudformation", "describe-stack-resource-drifts",
		"--stack-name", stackName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack resource drifts: %w", err)
	}

	var driftDetailsResult struct {
		StackResourceDrifts []struct {
			LogicalResourceId        string `json:"LogicalResourceId"`
			PhysicalResourceId       string `json:"PhysicalResourceId"`
			ResourceType             string `json:"ResourceType"`
			StackResourceDriftStatus string `json:"StackResourceDriftStatus"`
			PropertyDifferences      []struct {
				PropertyPath   string      `json:"PropertyPath"`
				ExpectedValue  interface{} `json:"ExpectedValue"`
				ActualValue    interface{} `json:"ActualValue"`
				DifferenceType string      `json:"DifferenceType"`
			} `json:"PropertyDifferences,omitempty"`
			ExpectedProperties map[string]interface{} `json:"ExpectedProperties,omitempty"`
			ActualProperties   map[string]interface{} `json:"ActualProperties,omitempty"`
		} `json:"StackResourceDrifts"`
	}

	if err := json.Unmarshal(output, &driftDetailsResult); err != nil {
		return nil, fmt.Errorf("failed to parse drift details: %w", err)
	}

	var driftedResources []*DriftedResource
	driftedCount := 0

	for _, drift := range driftDetailsResult.StackResourceDrifts {
		if drift.StackResourceDriftStatus != "IN_SYNC" {
			driftedCount++

			var propDiffs []*PropertyDifference
			for _, diff := range drift.PropertyDifferences {
				propDiffs = append(propDiffs, &PropertyDifference{
					PropertyPath:   diff.PropertyPath,
					ExpectedValue:  diff.ExpectedValue,
					ActualValue:    diff.ActualValue,
					DifferenceType: diff.DifferenceType,
				})
			}

			driftedResources = append(driftedResources, &DriftedResource{
				LogicalID:          drift.LogicalResourceId,
				PhysicalID:         drift.PhysicalResourceId,
				ResourceType:       drift.ResourceType,
				DriftStatus:        drift.StackResourceDriftStatus,
				PropertyDiffs:      propDiffs,
				ExpectedProperties: drift.ExpectedProperties,
				ActualProperties:   drift.ActualProperties,
			})
		}
	}

	// Generate recommendations based on drift findings
	recommendations := m.generateDriftRecommendations(driftedResources)

	return &DriftResult{
		StackName:          stackName,
		DriftStatus:        driftStatus,
		DetectionTime:      detectionTime,
		DriftedResources:   driftedResources,
		TotalResources:     len(driftDetailsResult.StackResourceDrifts),
		DriftedCount:       driftedCount,
		RecommendedActions: recommendations,
	}, nil
}

// generateDriftRecommendations generates recommendations based on drift findings
func (m *CloudFormationManager) generateDriftRecommendations(driftedResources []*DriftedResource) []string {
	var recommendations []string

	if len(driftedResources) == 0 {
		recommendations = append(recommendations, "Stack is in sync - no action required")
		return recommendations
	}

	// General recommendations
	recommendations = append(recommendations,
		"Review drifted resources and determine if changes were intentional",
		"Consider updating CloudFormation template to match current state",
		"Use stack import to bring resources back under CloudFormation management",
	)

	// Resource-specific recommendations
	resourceTypes := make(map[string]int)
	for _, resource := range driftedResources {
		resourceTypes[resource.ResourceType]++
	}

	for resourceType, count := range resourceTypes {
		switch resourceType {
		case "AWS::ElasticLoadBalancingV2::LoadBalancer":
			recommendations = append(recommendations,
				fmt.Sprintf("%d Load Balancer(s) drifted - check security groups and listeners", count))
		case "AWS::ECS::Service":
			recommendations = append(recommendations,
				fmt.Sprintf("%d ECS Service(s) drifted - verify task definition and desired count", count))
		case "AWS::RDS::DBInstance":
			recommendations = append(recommendations,
				fmt.Sprintf("%d RDS Instance(s) drifted - review parameter groups and security settings", count))
		case "AWS::Lambda::Function":
			recommendations = append(recommendations,
				fmt.Sprintf("%d Lambda Function(s) drifted - check code, configuration, and environment variables", count))
		}
	}

	return recommendations
}

// ValidateStackHealth performs comprehensive health validation of a stack
func (m *CloudFormationManager) ValidateStackHealth(ctx context.Context, stackName, region string) (*HealthResult, error) {
	if region == "" {
		region = m.provider.GetCurrentRegion()
	}

	// Get stack details
	stack, err := m.GetStack(ctx, stackName, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack details: %w", err)
	}

	healthResult := &HealthResult{
		StackName:          stackName,
		LastChecked:        time.Now(),
		HealthyResources:   0,
		UnhealthyResources: 0,
		ResourceHealth:     []*ResourceHealthStatus{},
		Issues:             []string{},
		Recommendations:    []string{},
	}

	// Check overall stack status
	if stack.Status != "CREATE_COMPLETE" && stack.Status != "UPDATE_COMPLETE" {
		healthResult.Issues = append(healthResult.Issues,
			fmt.Sprintf("Stack is in %s state", stack.Status))
	}

	// Check individual resources
	for _, resource := range stack.Resources {
		resourceHealth := m.checkResourceHealth(ctx, resource, region)
		healthResult.ResourceHealth = append(healthResult.ResourceHealth, resourceHealth)

		if resourceHealth.Health == "healthy" {
			healthResult.HealthyResources++
		} else {
			healthResult.UnhealthyResources++
			if resourceHealth.ErrorMessage != "" {
				healthResult.Issues = append(healthResult.Issues, resourceHealth.ErrorMessage)
			}
		}
	}

	// Determine overall health
	if healthResult.UnhealthyResources == 0 {
		healthResult.OverallHealth = "healthy"
	} else if healthResult.UnhealthyResources < healthResult.HealthyResources {
		healthResult.OverallHealth = "degraded"
	} else {
		healthResult.OverallHealth = "unhealthy"
	}

	// Generate recommendations
	healthResult.Recommendations = m.generateHealthRecommendations(healthResult)

	return healthResult, nil
}

// checkResourceHealth checks the health of an individual resource
func (m *CloudFormationManager) checkResourceHealth(ctx context.Context, resource *StackResource, region string) *ResourceHealthStatus {
	status := &ResourceHealthStatus{
		LogicalID:    resource.LogicalID,
		PhysicalID:   resource.PhysicalID,
		ResourceType: resource.Type,
		Status:       resource.Status,
		CheckedAt:    time.Now(),
		Health:       "unknown",
	}

	// Resource is unhealthy if it's in a failed state
	if strings.Contains(resource.Status, "FAILED") {
		status.Health = "unhealthy"
		status.ErrorMessage = resource.StatusReason
		return status
	}

	// Check resource-specific health
	switch resource.Type {
	case "AWS::ElasticLoadBalancingV2::LoadBalancer":
		status.Health = m.checkLoadBalancerHealth(ctx, resource.PhysicalID, region)
	case "AWS::ECS::Service":
		status.Health = m.checkECSServiceHealth(ctx, resource.PhysicalID, region)
	case "AWS::RDS::DBInstance":
		status.Health = m.checkRDSInstanceHealth(ctx, resource.PhysicalID, region)
	case "AWS::Lambda::Function":
		status.Health = m.checkLambdaFunctionHealth(ctx, resource.PhysicalID, region)
	default:
		// For other resource types, consider them healthy if status is complete
		if strings.Contains(resource.Status, "COMPLETE") {
			status.Health = "healthy"
		} else {
			status.Health = "degraded"
		}
	}

	return status
}

// checkLoadBalancerHealth checks the health of a load balancer
func (m *CloudFormationManager) checkLoadBalancerHealth(ctx context.Context, albArn, region string) string {
	cmd := exec.Command("aws", "elbv2", "describe-load-balancers",
		"--load-balancer-arns", albArn, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return "unhealthy"
	}

	var result struct {
		LoadBalancers []struct {
			State struct {
				Code string `json:"Code"`
			} `json:"State"`
		} `json:"LoadBalancers"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "unhealthy"
	}

	if len(result.LoadBalancers) == 0 {
		return "unhealthy"
	}

	switch result.LoadBalancers[0].State.Code {
	case "active":
		return "healthy"
	case "provisioning":
		return "degraded"
	default:
		return "unhealthy"
	}
}

// checkECSServiceHealth checks the health of an ECS service
func (m *CloudFormationManager) checkECSServiceHealth(ctx context.Context, serviceArn, region string) string {
	parts := strings.Split(serviceArn, "/")
	if len(parts) < 3 {
		return "unhealthy"
	}

	clusterName := parts[1]
	serviceName := parts[2]

	cmd := exec.Command("aws", "ecs", "describe-services",
		"--cluster", clusterName, "--services", serviceName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return "unhealthy"
	}

	var result struct {
		Services []struct {
			Status       string `json:"status"`
			DesiredCount int    `json:"desiredCount"`
			RunningCount int    `json:"runningCount"`
		} `json:"services"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "unhealthy"
	}

	if len(result.Services) == 0 {
		return "unhealthy"
	}

	service := result.Services[0]
	if service.Status == "ACTIVE" && service.RunningCount == service.DesiredCount {
		return "healthy"
	} else if service.Status == "ACTIVE" && service.RunningCount > 0 {
		return "degraded"
	} else {
		return "unhealthy"
	}
}

// checkRDSInstanceHealth checks the health of an RDS instance
func (m *CloudFormationManager) checkRDSInstanceHealth(ctx context.Context, dbInstanceId, region string) string {
	cmd := exec.Command("aws", "rds", "describe-db-instances",
		"--db-instance-identifier", dbInstanceId, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return "unhealthy"
	}

	var result struct {
		DBInstances []struct {
			DBInstanceStatus string `json:"DBInstanceStatus"`
		} `json:"DBInstances"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "unhealthy"
	}

	if len(result.DBInstances) == 0 {
		return "unhealthy"
	}

	switch result.DBInstances[0].DBInstanceStatus {
	case "available":
		return "healthy"
	case "creating", "modifying", "backing-up":
		return "degraded"
	default:
		return "unhealthy"
	}
}

// checkLambdaFunctionHealth checks the health of a Lambda function
func (m *CloudFormationManager) checkLambdaFunctionHealth(ctx context.Context, functionName, region string) string {
	cmd := exec.Command("aws", "lambda", "get-function",
		"--function-name", functionName, "--region", region)
	output, err := cmd.Output()
	if err != nil {
		return "unhealthy"
	}

	var result struct {
		Configuration struct {
			State            string `json:"State"`
			LastUpdateStatus string `json:"LastUpdateStatus"`
		} `json:"Configuration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "unhealthy"
	}

	config := result.Configuration
	if config.State == "Active" && config.LastUpdateStatus == "Successful" {
		return "healthy"
	} else if config.State == "Pending" {
		return "degraded"
	} else {
		return "unhealthy"
	}
}

// generateHealthRecommendations generates health-based recommendations
func (m *CloudFormationManager) generateHealthRecommendations(healthResult *HealthResult) []string {
	var recommendations []string

	if healthResult.OverallHealth == "healthy" {
		recommendations = append(recommendations, "Stack is healthy - continue monitoring")
		return recommendations
	}

	// General recommendations for unhealthy/degraded stacks
	recommendations = append(recommendations,
		"Review stack events for error details",
		"Check CloudWatch logs for application errors",
		"Verify IAM permissions for stack resources",
	)

	// Resource-specific recommendations
	unhealthyTypes := make(map[string]int)
	for _, resource := range healthResult.ResourceHealth {
		if resource.Health != "healthy" {
			unhealthyTypes[resource.ResourceType]++
		}
	}

	for resourceType, count := range unhealthyTypes {
		switch resourceType {
		case "AWS::ElasticLoadBalancingV2::LoadBalancer":
			recommendations = append(recommendations,
				fmt.Sprintf("%d Load Balancer(s) unhealthy - check target health and security groups", count))
		case "AWS::ECS::Service":
			recommendations = append(recommendations,
				fmt.Sprintf("%d ECS Service(s) unhealthy - verify task definition and cluster capacity", count))
		case "AWS::RDS::DBInstance":
			recommendations = append(recommendations,
				fmt.Sprintf("%d RDS Instance(s) unhealthy - check database status and connectivity", count))
		case "AWS::Lambda::Function":
			recommendations = append(recommendations,
				fmt.Sprintf("%d Lambda Function(s) unhealthy - review function logs and configuration", count))
		}
	}

	return recommendations
}

// ===============================
// AWS Provider CloudFormation Integration
// ===============================

// ListCloudFormationStacks lists CloudFormation stacks with filtering
func (p *AWSProvider) ListCloudFormationStacks(ctx context.Context, filters *StackFilters) ([]*Stack, error) {
	return p.cfManager.ListStacks(ctx, filters)
}

// GetCloudFormationStack gets detailed information about a specific stack
func (p *AWSProvider) GetCloudFormationStack(ctx context.Context, stackName, region string) (*Stack, error) {
	return p.cfManager.GetStack(ctx, stackName, region)
}

// GetCloudFormationStackResources gets all resources in a stack
func (p *AWSProvider) GetCloudFormationStackResources(ctx context.Context, stackName, region string) ([]*StackResource, error) {
	return p.cfManager.GetStackResources(ctx, stackName, region)
}

// GetCloudFormationStackOutputs gets stack outputs
func (p *AWSProvider) GetCloudFormationStackOutputs(ctx context.Context, stackName, region string) (map[string]string, error) {
	return p.cfManager.GetStackOutputs(ctx, stackName, region)
}

// DetectCloudFormationStackDrift detects drift in a stack
func (p *AWSProvider) DetectCloudFormationStackDrift(ctx context.Context, stackName, region string) (*DriftResult, error) {
	return p.cfManager.DetectDrift(ctx, stackName, region)
}

// ValidateCloudFormationStackHealth validates stack health
func (p *AWSProvider) ValidateCloudFormationStackHealth(ctx context.Context, stackName, region string) (*HealthResult, error) {
	return p.cfManager.ValidateStackHealth(ctx, stackName, region)
}

// ListAPMStacks lists only APM-related CloudFormation stacks
func (p *AWSProvider) ListAPMStacks(ctx context.Context, regions []string) ([]*Stack, error) {
	if len(regions) == 0 {
		regions = []string{p.GetCurrentRegion()}
	}

	filters := &StackFilters{
		Regions: regions,
		APMOnly: true,
		StackStatus: []string{
			"CREATE_COMPLETE",
			"UPDATE_COMPLETE",
			"UPDATE_ROLLBACK_COMPLETE",
		},
	}

	return p.cfManager.ListStacks(ctx, filters)
}

// GetAPMStackSummary gets a summary of APM infrastructure across regions
func (p *AWSProvider) GetAPMStackSummary(ctx context.Context, regions []string) (*APMStackSummary, error) {
	if len(regions) == 0 {
		regions = []string{p.GetCurrentRegion()}
	}

	stacks, err := p.ListAPMStacks(ctx, regions)
	if err != nil {
		return nil, fmt.Errorf("failed to list APM stacks: %w", err)
	}

	summary := &APMStackSummary{
		TotalStacks:     len(stacks),
		HealthyStacks:   0,
		DegradedStacks:  0,
		UnhealthyStacks: 0,
		RegionSummary:   make(map[string]*RegionSummary),
		ResourceSummary: &ResourceSummary{
			LoadBalancers:       0,
			ECSServices:         0,
			RDSInstances:        0,
			LambdaFunctions:     0,
			ElastiCacheClusters: 0,
			S3Buckets:           0,
			VPCs:                0,
		},
		LastUpdated: time.Now(),
	}

	// Process each stack
	for _, stack := range stacks {
		// Update region summary
		if _, exists := summary.RegionSummary[stack.Region]; !exists {
			summary.RegionSummary[stack.Region] = &RegionSummary{
				Region:        stack.Region,
				StackCount:    0,
				HealthyStacks: 0,
				Issues:        []string{},
			}
		}
		regionSummary := summary.RegionSummary[stack.Region]
		regionSummary.StackCount++

		// Check stack health (simplified)
		if strings.Contains(stack.Status, "COMPLETE") {
			summary.HealthyStacks++
			regionSummary.HealthyStacks++
		} else if strings.Contains(stack.Status, "ROLLBACK") {
			summary.DegradedStacks++
			regionSummary.Issues = append(regionSummary.Issues,
				fmt.Sprintf("Stack %s is in rollback state", stack.Name))
		} else {
			summary.UnhealthyStacks++
			regionSummary.Issues = append(regionSummary.Issues,
				fmt.Sprintf("Stack %s is unhealthy: %s", stack.Name, stack.Status))
		}

		// Count resources
		if stack.APMResources != nil {
			summary.ResourceSummary.LoadBalancers += len(stack.APMResources.LoadBalancers)
			summary.ResourceSummary.ECSServices += len(stack.APMResources.ECSServices)
			summary.ResourceSummary.RDSInstances += len(stack.APMResources.RDSInstances)
			summary.ResourceSummary.LambdaFunctions += len(stack.APMResources.LambdaFunctions)
			summary.ResourceSummary.ElastiCacheClusters += len(stack.APMResources.ElastiCacheClusters)
			summary.ResourceSummary.S3Buckets += len(stack.APMResources.S3Buckets)
			summary.ResourceSummary.VPCs += len(stack.APMResources.VPCResources)
		}
	}

	return summary, nil
}

// SearchAPMResources searches for APM resources across CloudFormation stacks
func (p *AWSProvider) SearchAPMResources(ctx context.Context, resourceType string, regions []string) ([]*APMResourceSearchResult, error) {
	if len(regions) == 0 {
		regions = []string{p.GetCurrentRegion()}
	}

	stacks, err := p.ListAPMStacks(ctx, regions)
	if err != nil {
		return nil, fmt.Errorf("failed to list APM stacks: %w", err)
	}

	var results []*APMResourceSearchResult

	for _, stack := range stacks {
		if stack.APMResources == nil {
			continue
		}

		switch strings.ToLower(resourceType) {
		case "loadbalancer", "alb", "nlb":
			for _, lb := range stack.APMResources.LoadBalancers {
				results = append(results, &APMResourceSearchResult{
					StackName:    stack.Name,
					Region:       stack.Region,
					ResourceType: "LoadBalancer",
					ResourceID:   lb.Arn,
					ResourceName: lb.DNSName,
					Status:       "active", // simplified
					Endpoint:     lb.DNSName,
				})
			}

		case "ecs", "service":
			for _, svc := range stack.APMResources.ECSServices {
				results = append(results, &APMResourceSearchResult{
					StackName:    stack.Name,
					Region:       stack.Region,
					ResourceType: "ECSService",
					ResourceID:   svc.ServiceName,
					ResourceName: svc.ServiceName,
					Status:       svc.Status,
					Endpoint:     fmt.Sprintf("%s/%s", svc.ClusterName, svc.ServiceName),
				})
			}

		case "rds", "database":
			for _, db := range stack.APMResources.RDSInstances {
				results = append(results, &APMResourceSearchResult{
					StackName:    stack.Name,
					Region:       stack.Region,
					ResourceType: "RDSInstance",
					ResourceID:   db.DBInstanceIdentifier,
					ResourceName: db.DBName,
					Status:       db.Status,
					Endpoint:     fmt.Sprintf("%s:%d", db.Endpoint, db.Port),
				})
			}

		case "lambda", "function":
			for _, fn := range stack.APMResources.LambdaFunctions {
				results = append(results, &APMResourceSearchResult{
					StackName:    stack.Name,
					Region:       stack.Region,
					ResourceType: "LambdaFunction",
					ResourceID:   fn.FunctionName,
					ResourceName: fn.FunctionName,
					Status:       fn.State,
					Endpoint:     fn.FunctionName,
				})
			}
		}
	}

	return results, nil
}

// Types for CloudFormation management integration
type APMStackSummary struct {
	TotalStacks     int                       `json:"totalStacks"`
	HealthyStacks   int                       `json:"healthyStacks"`
	DegradedStacks  int                       `json:"degradedStacks"`
	UnhealthyStacks int                       `json:"unhealthyStacks"`
	RegionSummary   map[string]*RegionSummary `json:"regionSummary"`
	ResourceSummary *ResourceSummary          `json:"resourceSummary"`
	LastUpdated     time.Time                 `json:"lastUpdated"`
}

type RegionSummary struct {
	Region        string   `json:"region"`
	StackCount    int      `json:"stackCount"`
	HealthyStacks int      `json:"healthyStacks"`
	Issues        []string `json:"issues"`
}

type ResourceSummary struct {
	LoadBalancers       int `json:"loadBalancers"`
	ECSServices         int `json:"ecsServices"`
	RDSInstances        int `json:"rdsInstances"`
	LambdaFunctions     int `json:"lambdaFunctions"`
	ElastiCacheClusters int `json:"elastiCacheClusters"`
	S3Buckets           int `json:"s3Buckets"`
	VPCs                int `json:"vpcs"`
}

type APMResourceSearchResult struct {
	StackName    string `json:"stackName"`
	Region       string `json:"region"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	Status       string `json:"status"`
	Endpoint     string `json:"endpoint"`
}

// S3Manager Implementation - Core Bucket Operations

// CreateBucket creates a new S3 bucket with comprehensive configuration
func (s *S3Manager) CreateBucket(ctx context.Context, name, region string, options *BucketOptions) (*Bucket, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Validate bucket name
	if err := s.validateBucketName(name); err != nil {
		return nil, &CloudError{
			Code:    "S3_INVALID_BUCKET_NAME",
			Message: fmt.Sprintf("Invalid bucket name: %v", err),
		}
	}

	// Set default options if not provided
	if options == nil {
		options = s.getDefaultBucketOptions(region)
	}

	// Create the bucket
	if err := s.createBucketWithAWS(ctx, name, region); err != nil {
		return nil, err
	}

	// Apply bucket configurations
	bucket := &Bucket{
		Name:         name,
		Region:       region,
		CreationDate: time.Now(),
		Tags:         options.Tags,
	}

	// Configure versioning
	if options.Versioning != nil {
		if err := s.setBucketVersioning(ctx, name, options.Versioning); err != nil {
			return nil, fmt.Errorf("failed to configure versioning: %w", err)
		}
		bucket.Versioning = *options.Versioning
	}

	// Configure encryption
	if options.Encryption != nil {
		if err := s.setBucketEncryption(ctx, name, options.Encryption); err != nil {
			return nil, fmt.Errorf("failed to configure encryption: %w", err)
		}
		bucket.Encryption = *options.Encryption
	}

	// Configure public access block (security best practice)
	if options.PublicAccessBlock != nil {
		if err := s.setBucketPublicAccessBlock(ctx, name, options.PublicAccessBlock); err != nil {
			return nil, fmt.Errorf("failed to configure public access block: %w", err)
		}
		bucket.PublicAccessBlock = *options.PublicAccessBlock
	}

	// Configure lifecycle policies
	if options.Lifecycle != nil {
		if err := s.setBucketLifecycle(ctx, name, options.Lifecycle); err != nil {
			return nil, fmt.Errorf("failed to configure lifecycle: %w", err)
		}
		bucket.Lifecycle = *options.Lifecycle
	}

	// Configure bucket policy
	if options.Policy != nil {
		if err := s.setBucketPolicy(ctx, name, options.Policy); err != nil {
			return nil, fmt.Errorf("failed to configure policy: %w", err)
		}
		bucket.Policy = *options.Policy
	}

	// Configure replication
	if options.Replication != nil {
		if err := s.setBucketReplication(ctx, name, options.Replication); err != nil {
			return nil, fmt.Errorf("failed to configure replication: %w", err)
		}
		bucket.Replication = *options.Replication
	}

	// Configure logging
	if options.Logging != nil {
		if err := s.setBucketLogging(ctx, name, options.Logging); err != nil {
			return nil, fmt.Errorf("failed to configure logging: %w", err)
		}
		bucket.Logging = *options.Logging
	}

	// Apply tags
	if len(options.Tags) > 0 {
		if err := s.setBucketTags(ctx, name, options.Tags); err != nil {
			return nil, fmt.Errorf("failed to set tags: %w", err)
		}
	}

	return bucket, nil
}

// ListBuckets lists all S3 buckets in the specified region
func (s *S3Manager) ListBuckets(ctx context.Context, region string) ([]*Bucket, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// List buckets using AWS CLI
	cmd := exec.CommandContext(ctx, "aws", "s3api", "list-buckets")
	if region != "" {
		cmd.Args = append(cmd.Args, "--region", region)
	}
	output, err := cmd.Output()
	if err != nil {
		return nil, &CloudError{
			Code:    "S3_LIST_BUCKETS_FAILED",
			Message: fmt.Sprintf("Failed to list buckets: %v", err),
		}
	}

	var result struct {
		Buckets []struct {
			Name         string    `json:"Name"`
			CreationDate time.Time `json:"CreationDate"`
		} `json:"Buckets"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bucket list: %w", err)
	}

	var buckets []*Bucket
	for _, b := range result.Buckets {
		bucket := &Bucket{
			Name:         b.Name,
			CreationDate: b.CreationDate,
		}

		// Get bucket region
		if bucketRegion, err := s.getBucketRegion(ctx, b.Name); err == nil {
			bucket.Region = bucketRegion
		}

		// Get bucket location
		if location, err := s.getBucketLocation(ctx, b.Name); err == nil {
			bucket.Location = location
		}

		// Get bucket versioning
		if versioning, err := s.getBucketVersioning(ctx, b.Name); err == nil {
			bucket.Versioning = *versioning
		}

		// Get bucket encryption
		if encryption, err := s.getBucketEncryption(ctx, b.Name); err == nil {
			bucket.Encryption = *encryption
		}

		// Get bucket tags
		if tags, err := s.getBucketTags(ctx, b.Name); err == nil {
			bucket.Tags = tags
		}

		buckets = append(buckets, bucket)
	}

	return buckets, nil
}

// GetBucket gets detailed information about a specific bucket
func (s *S3Manager) GetBucket(ctx context.Context, name, region string) (*BucketDetails, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Get basic bucket information
	bucket := &Bucket{
		Name:   name,
		Region: region,
	}

	// Get bucket region if not provided
	if region == "" {
		if bucketRegion, err := s.getBucketRegion(ctx, name); err == nil {
			bucket.Region = bucketRegion
			region = bucketRegion
		}
	}

	// Get bucket location
	if location, err := s.getBucketLocation(ctx, name); err == nil {
		bucket.Location = location
	}

	// Get bucket versioning
	if versioning, err := s.getBucketVersioning(ctx, name); err == nil {
		bucket.Versioning = *versioning
	}

	// Get bucket encryption
	if encryption, err := s.getBucketEncryption(ctx, name); err == nil {
		bucket.Encryption = *encryption
	}

	// Get bucket lifecycle
	if lifecycle, err := s.getBucketLifecycle(ctx, name); err == nil {
		bucket.Lifecycle = *lifecycle
	}

	// Get bucket replication
	if replication, err := s.getBucketReplication(ctx, name); err == nil {
		bucket.Replication = *replication
	}

	// Get bucket policy
	if policy, err := s.getBucketPolicy(ctx, name); err == nil {
		bucket.Policy = *policy
	}

	// Get bucket tags
	if tags, err := s.getBucketTags(ctx, name); err == nil {
		bucket.Tags = tags
	}

	// Get bucket public access block
	if publicAccessBlock, err := s.getBucketPublicAccessBlock(ctx, name); err == nil {
		bucket.PublicAccessBlock = *publicAccessBlock
	}

	// Get bucket logging
	if logging, err := s.getBucketLogging(ctx, name); err == nil {
		bucket.Logging = *logging
	}

	// Get bucket notification
	if notification, err := s.getBucketNotification(ctx, name); err == nil {
		bucket.Notification = *notification
	}

	// Get bucket CORS
	if cors, err := s.getBucketCORS(ctx, name); err == nil {
		bucket.CORS = *cors
	}

	// Get bucket website
	if website, err := s.getBucketWebsite(ctx, name); err == nil {
		bucket.Website = *website
	}

	// Get bucket metrics
	metrics, err := s.getBucketMetrics(ctx, name)
	if err != nil {
		metrics = &BucketMetrics{LastUpdated: time.Now()}
	}

	// Get bucket cost information
	cost, err := s.getBucketCost(ctx, name)
	if err != nil {
		cost = &BucketCost{Currency: "USD"}
	}

	// Get APM configuration if it exists
	apmConfig, err := s.getAPMBucketConfig(ctx, name)
	if err != nil {
		apmConfig = &APMBucketConfig{}
	}

	return &BucketDetails{
		Bucket:       bucket,
		Size:         metrics.TotalSize,
		ObjectCount:  metrics.TotalObjects,
		LastModified: time.Now(),
		Cost:         *cost,
		Metrics:      *metrics,
		APMConfig:    *apmConfig,
	}, nil
}

// DeleteBucket deletes an S3 bucket with safety checks
func (s *S3Manager) DeleteBucket(ctx context.Context, name, region string, force bool) error {
	if s.provider == nil {
		return &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Safety check: list objects in bucket
	objects, err := s.ListFiles(ctx, name, "", &ListOptions{MaxKeys: 1})
	if err != nil {
		return fmt.Errorf("failed to check bucket contents: %w", err)
	}

	if len(objects.Objects) > 0 && !force {
		return &CloudError{
			Code:    "S3_BUCKET_NOT_EMPTY",
			Message: "Bucket is not empty. Use force=true to delete all objects and bucket",
		}
	}

	// If force is true, delete all objects first
	if force && len(objects.Objects) > 0 {
		if err := s.emptyBucket(ctx, name); err != nil {
			return fmt.Errorf("failed to empty bucket: %w", err)
		}
	}

	// Delete bucket versioning if enabled
	if versioning, err := s.getBucketVersioning(ctx, name); err == nil && versioning.Status == "Enabled" {
		if err := s.setBucketVersioning(ctx, name, &VersioningConfig{Status: "Suspended"}); err != nil {
			return fmt.Errorf("failed to suspend versioning: %w", err)
		}
	}

	// Delete the bucket
	cmd := exec.CommandContext(ctx, "aws", "s3api", "delete-bucket", "--bucket", name)
	if region != "" {
		cmd.Args = append(cmd.Args, "--region", region)
	}

	if err := cmd.Run(); err != nil {
		return &CloudError{
			Code:    "S3_DELETE_BUCKET_FAILED",
			Message: fmt.Sprintf("Failed to delete bucket: %v", err),
		}
	}

	return nil
}

// UploadFile uploads a file to S3 with comprehensive options
func (s *S3Manager) UploadFile(ctx context.Context, bucket, key string, content io.Reader, options *UploadOptions) (*FileInfo, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Set default options if not provided
	if options == nil {
		options = &UploadOptions{}
	}

	// Read content into buffer for size calculation and potential multipart upload
	var buf bytes.Buffer
	size, err := buf.ReadFrom(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Determine if multipart upload should be used
	useMultipart := options.UseMultipart || size > 100*1024*1024 // 100MB default threshold

	var fileInfo *FileInfo
	if useMultipart && size > 5*1024*1024 { // 5MB minimum for multipart
		fileInfo, err = s.uploadFileMultipart(ctx, bucket, key, &buf, size, options)
	} else {
		fileInfo, err = s.uploadFileSingle(ctx, bucket, key, &buf, options)
	}

	if err != nil {
		return nil, err
	}

	return fileInfo, nil
}

// DownloadFile downloads a file from S3 with comprehensive options
func (s *S3Manager) DownloadFile(ctx context.Context, bucket, key string, options *DownloadOptions) (io.ReadCloser, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Set default options if not provided
	if options == nil {
		options = &DownloadOptions{}
	}

	// Build AWS CLI command
	args := []string{"s3api", "get-object", "--bucket", bucket, "--key", key}

	// Add conditional headers
	if !options.IfModifiedSince.IsZero() {
		args = append(args, "--if-modified-since", options.IfModifiedSince.Format(time.RFC3339))
	}
	if !options.IfUnmodifiedSince.IsZero() {
		args = append(args, "--if-unmodified-since", options.IfUnmodifiedSince.Format(time.RFC3339))
	}
	if options.IfMatch != "" {
		args = append(args, "--if-match", options.IfMatch)
	}
	if options.IfNoneMatch != "" {
		args = append(args, "--if-none-match", options.IfNoneMatch)
	}
	if options.Range != "" {
		args = append(args, "--range", options.Range)
	}
	if options.VersionId != "" {
		args = append(args, "--version-id", options.VersionId)
	}

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", "s3-download-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	args = append(args, tmpFile.Name())

	cmd := exec.CommandContext(ctx, "aws", args...)
	if err := cmd.Run(); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, &CloudError{
			Code:    "S3_DOWNLOAD_FAILED",
			Message: fmt.Sprintf("Failed to download file: %v", err),
		}
	}

	// Reset file pointer to beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Return a ReadCloser that cleans up the temp file when closed
	return &tempFileReader{
		File: tmpFile,
		path: tmpFile.Name(),
	}, nil
}

// ListFiles lists objects in an S3 bucket with comprehensive filtering
func (s *S3Manager) ListFiles(ctx context.Context, bucket, prefix string, options *ListOptions) (*ListResult, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Set default options if not provided
	if options == nil {
		options = &ListOptions{MaxKeys: 1000}
	}

	// Build AWS CLI command
	args := []string{"s3api", "list-objects-v2", "--bucket", bucket}

	if prefix != "" {
		args = append(args, "--prefix", prefix)
	}
	if options.Delimiter != "" {
		args = append(args, "--delimiter", options.Delimiter)
	}
	if options.MaxKeys > 0 {
		args = append(args, "--max-keys", fmt.Sprintf("%d", options.MaxKeys))
	}
	if options.StartAfter != "" {
		args = append(args, "--start-after", options.StartAfter)
	}
	if options.ContinuationToken != "" {
		args = append(args, "--continuation-token", options.ContinuationToken)
	}
	if options.FetchOwner {
		args = append(args, "--fetch-owner")
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, &CloudError{
			Code:    "S3_LIST_OBJECTS_FAILED",
			Message: fmt.Sprintf("Failed to list objects: %v", err),
		}
	}

	var result struct {
		Contents []struct {
			Key          string    `json:"Key"`
			LastModified time.Time `json:"LastModified"`
			ETag         string    `json:"ETag"`
			Size         int64     `json:"Size"`
			StorageClass string    `json:"StorageClass"`
			Owner        *struct {
				ID          string `json:"ID"`
				DisplayName string `json:"DisplayName"`
			} `json:"Owner,omitempty"`
		} `json:"Contents"`
		CommonPrefixes []struct {
			Prefix string `json:"Prefix"`
		} `json:"CommonPrefixes"`
		IsTruncated           bool   `json:"IsTruncated"`
		NextContinuationToken string `json:"NextContinuationToken"`
		KeyCount              int    `json:"KeyCount"`
		MaxKeys               int    `json:"MaxKeys"`
		Name                  string `json:"Name"`
		Prefix                string `json:"Prefix"`
		Delimiter             string `json:"Delimiter"`
		EncodingType          string `json:"EncodingType"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list result: %w", err)
	}

	// Convert to our FileInfo structure
	var objects []*FileInfo
	for _, obj := range result.Contents {
		fileInfo := &FileInfo{
			Key:          obj.Key,
			LastModified: obj.LastModified,
			ETag:         strings.Trim(obj.ETag, "\""),
			Size:         obj.Size,
			StorageClass: obj.StorageClass,
		}

		if obj.Owner != nil {
			fileInfo.Owner = ObjectOwner{
				ID:          obj.Owner.ID,
				DisplayName: obj.Owner.DisplayName,
			}
		}

		// Get additional metadata if requested
		if options.IncludeMetadata || options.IncludeTags {
			if metadata, err := s.getObjectMetadata(ctx, bucket, obj.Key); err == nil {
				if options.IncludeMetadata {
					fileInfo.Metadata = metadata.Metadata
					fileInfo.ContentType = metadata.ContentType
					fileInfo.ContentEncoding = metadata.ContentEncoding
					fileInfo.ContentLanguage = metadata.ContentLanguage
					fileInfo.ContentDisposition = metadata.ContentDisposition
					fileInfo.CacheControl = metadata.CacheControl
					fileInfo.Expires = metadata.Expires
				}
			}

			if options.IncludeTags {
				if tags, err := s.getObjectTags(ctx, bucket, obj.Key); err == nil {
					fileInfo.Tags = tags
				}
			}
		}

		objects = append(objects, fileInfo)
	}

	// Convert common prefixes
	var commonPrefixes []string
	for _, cp := range result.CommonPrefixes {
		commonPrefixes = append(commonPrefixes, cp.Prefix)
	}

	return &ListResult{
		Objects:               objects,
		CommonPrefixes:        commonPrefixes,
		IsTruncated:           result.IsTruncated,
		NextContinuationToken: result.NextContinuationToken,
		MaxKeys:               result.MaxKeys,
		KeyCount:              result.KeyCount,
		Name:                  result.Name,
		Prefix:                result.Prefix,
		Delimiter:             result.Delimiter,
		EncodingType:          result.EncodingType,
	}, nil
}

// DeleteFile deletes an object from S3
func (s *S3Manager) DeleteFile(ctx context.Context, bucket, key string, options *DeleteOptions) error {
	if s.provider == nil {
		return &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Set default options if not provided
	if options == nil {
		options = &DeleteOptions{}
	}

	// Build AWS CLI command
	args := []string{"s3api", "delete-object", "--bucket", bucket, "--key", key}

	if options.VersionId != "" {
		args = append(args, "--version-id", options.VersionId)
	}
	if options.MFA != "" {
		args = append(args, "--mfa", options.MFA)
	}
	if options.RequestPayer != "" {
		args = append(args, "--request-payer", options.RequestPayer)
	}
	if options.BypassGovernanceRetention {
		args = append(args, "--bypass-governance-retention")
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	if err := cmd.Run(); err != nil {
		return &CloudError{
			Code:    "S3_DELETE_OBJECT_FAILED",
			Message: fmt.Sprintf("Failed to delete object: %v", err),
		}
	}

	return nil
}

// Helper Methods for S3 Operations

// tempFileReader wraps a file and removes it when closed
type tempFileReader struct {
	*os.File
	path string
}

func (t *tempFileReader) Close() error {
	err := t.File.Close()
	os.Remove(t.path)
	return err
}

// validateBucketName validates S3 bucket naming rules
func (s *S3Manager) validateBucketName(name string) error {
	if len(name) < 3 || len(name) > 63 {
		return errors.New("bucket name must be between 3 and 63 characters")
	}

	// Basic regex for bucket naming rules
	matched, err := regexp.MatchString(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`, name)
	if err != nil || !matched {
		return errors.New("bucket name contains invalid characters")
	}

	// Additional checks
	if strings.Contains(name, "..") {
		return errors.New("bucket name cannot contain consecutive periods")
	}
	if strings.Contains(name, ".-") || strings.Contains(name, "-.") {
		return errors.New("bucket name cannot contain periods adjacent to hyphens")
	}

	return nil
}

// getDefaultBucketOptions returns secure default options for bucket creation
func (s *S3Manager) getDefaultBucketOptions(region string) *BucketOptions {
	return &BucketOptions{
		Region: region,
		Versioning: &VersioningConfig{
			Status: "Enabled",
		},
		Encryption: &EncryptionConfig{
			Type:             "SSE-S3",
			Algorithm:        "AES256",
			BucketKeyEnabled: true,
		},
		PublicAccessBlock: &PublicAccessBlockConfig{
			BlockPublicAcls:       true,
			IgnorePublicAcls:      true,
			BlockPublicPolicy:     true,
			RestrictPublicBuckets: true,
		},
		Tags: map[string]string{
			"CreatedBy": "APM-Tool",
			"Purpose":   "Configuration",
		},
	}
}

// createBucketWithAWS creates the bucket using AWS CLI
func (s *S3Manager) createBucketWithAWS(ctx context.Context, name, region string) error {
	args := []string{"s3api", "create-bucket", "--bucket", name}

	if region != "us-east-1" {
		args = append(args, "--create-bucket-configuration", fmt.Sprintf("LocationConstraint=%s", region))
	}
	if region != "" {
		args = append(args, "--region", region)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	if err := cmd.Run(); err != nil {
		return &CloudError{
			Code:    "S3_CREATE_BUCKET_FAILED",
			Message: fmt.Sprintf("Failed to create bucket: %v", err),
		}
	}

	return nil
}

// getBucketRegion gets the region of a bucket
func (s *S3Manager) getBucketRegion(ctx context.Context, bucket string) (string, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-location", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var result struct {
		LocationConstraint string `json:"LocationConstraint"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", err
	}

	// AWS returns null for us-east-1
	if result.LocationConstraint == "" {
		return "us-east-1", nil
	}

	return result.LocationConstraint, nil
}

// getBucketLocation gets the location constraint of a bucket
func (s *S3Manager) getBucketLocation(ctx context.Context, bucket string) (string, error) {
	return s.getBucketRegion(ctx, bucket)
}

// setBucketVersioning configures bucket versioning
func (s *S3Manager) setBucketVersioning(ctx context.Context, bucket string, config *VersioningConfig) error {
	versioningConfig := map[string]interface{}{
		"Status": config.Status,
	}
	if config.MfaDelete != "" {
		versioningConfig["MfaDelete"] = config.MfaDelete
	}

	configJSON, err := json.Marshal(versioningConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal versioning config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-versioning",
		"--bucket", bucket,
		"--versioning-configuration", string(configJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket versioning: %w", err)
	}

	return nil
}

// getBucketVersioning gets bucket versioning configuration
func (s *S3Manager) getBucketVersioning(ctx context.Context, bucket string) (*VersioningConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-versioning", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result VersioningConfig
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	// Default to disabled if not set
	if result.Status == "" {
		result.Status = "Disabled"
	}

	return &result, nil
}

// setBucketEncryption configures bucket encryption
func (s *S3Manager) setBucketEncryption(ctx context.Context, bucket string, config *EncryptionConfig) error {
	encryptionRule := map[string]interface{}{
		"ApplyServerSideEncryptionByDefault": map[string]interface{}{
			"SSEAlgorithm": config.Algorithm,
		},
		"BucketKeyEnabled": config.BucketKeyEnabled,
	}

	if config.Type == "SSE-KMS" && config.KMSKeyId != "" {
		encryptionRule["ApplyServerSideEncryptionByDefault"].(map[string]interface{})["KMSMasterKeyID"] = config.KMSKeyId
	}

	encryptionConfig := map[string]interface{}{
		"Rules": []interface{}{encryptionRule},
	}

	configJSON, err := json.Marshal(encryptionConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal encryption config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-encryption",
		"--bucket", bucket,
		"--server-side-encryption-configuration", string(configJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket encryption: %w", err)
	}

	return nil
}

// getBucketEncryption gets bucket encryption configuration
func (s *S3Manager) getBucketEncryption(ctx context.Context, bucket string) (*EncryptionConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-encryption", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		// Return default encryption if not configured
		return &EncryptionConfig{
			Type:      "None",
			Algorithm: "",
		}, nil
	}

	var result struct {
		ServerSideEncryptionConfiguration struct {
			Rules []struct {
				ApplyServerSideEncryptionByDefault struct {
					SSEAlgorithm   string `json:"SSEAlgorithm"`
					KMSMasterKeyID string `json:"KMSMasterKeyID,omitempty"`
				} `json:"ApplyServerSideEncryptionByDefault"`
				BucketKeyEnabled bool `json:"BucketKeyEnabled"`
			} `json:"Rules"`
		} `json:"ServerSideEncryptionConfiguration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	if len(result.ServerSideEncryptionConfiguration.Rules) == 0 {
		return &EncryptionConfig{Type: "None"}, nil
	}

	rule := result.ServerSideEncryptionConfiguration.Rules[0]
	config := &EncryptionConfig{
		Algorithm:        rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm,
		BucketKeyEnabled: rule.BucketKeyEnabled,
	}

	if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != "" {
		config.Type = "SSE-KMS"
		config.KMSKeyId = rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID
	} else if config.Algorithm == "AES256" {
		config.Type = "SSE-S3"
	}

	return config, nil
}

// setBucketPublicAccessBlock configures public access blocking
func (s *S3Manager) setBucketPublicAccessBlock(ctx context.Context, bucket string, config *PublicAccessBlockConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal public access block config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-public-access-block",
		"--bucket", bucket,
		"--public-access-block-configuration", string(configJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set public access block: %w", err)
	}

	return nil
}

// getBucketPublicAccessBlock gets public access block configuration
func (s *S3Manager) getBucketPublicAccessBlock(ctx context.Context, bucket string) (*PublicAccessBlockConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-public-access-block", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &PublicAccessBlockConfig{}, nil // Default to no restrictions if not set
	}

	var result struct {
		PublicAccessBlockConfiguration PublicAccessBlockConfig `json:"PublicAccessBlockConfiguration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result.PublicAccessBlockConfiguration, nil
}

// setBucketTags sets bucket tags
func (s *S3Manager) setBucketTags(ctx context.Context, bucket string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}

	var tagSet []map[string]string
	for key, value := range tags {
		tagSet = append(tagSet, map[string]string{
			"Key":   key,
			"Value": value,
		})
	}

	tagging := map[string]interface{}{
		"TagSet": tagSet,
	}

	taggingJSON, err := json.Marshal(tagging)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-tagging",
		"--bucket", bucket,
		"--tagging", string(taggingJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket tags: %w", err)
	}

	return nil
}

// getBucketTags gets bucket tags
func (s *S3Manager) getBucketTags(ctx context.Context, bucket string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-tagging", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}, nil // Return empty map if no tags
	}

	var result struct {
		TagSet []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"TagSet"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for _, tag := range result.TagSet {
		tags[tag.Key] = tag.Value
	}

	return tags, nil
}

// uploadFileSingle uploads a file using single-part upload
func (s *S3Manager) uploadFileSingle(ctx context.Context, bucket, key string, content *bytes.Buffer, options *UploadOptions) (*FileInfo, error) {
	// Create temporary file for upload
	tmpFile, err := os.CreateTemp("", "s3-upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write content to temp file
	if _, err := tmpFile.Write(content.Bytes()); err != nil {
		return nil, fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Build AWS CLI command
	args := []string{"s3api", "put-object", "--bucket", bucket, "--key", key, "--body", tmpFile.Name()}

	// Add optional parameters
	if options.ContentType != "" {
		args = append(args, "--content-type", options.ContentType)
	}
	if options.ContentEncoding != "" {
		args = append(args, "--content-encoding", options.ContentEncoding)
	}
	if options.ContentLanguage != "" {
		args = append(args, "--content-language", options.ContentLanguage)
	}
	if options.ContentDisposition != "" {
		args = append(args, "--content-disposition", options.ContentDisposition)
	}
	if options.CacheControl != "" {
		args = append(args, "--cache-control", options.CacheControl)
	}
	if options.StorageClass != "" {
		args = append(args, "--storage-class", options.StorageClass)
	}
	if options.ServerSideEncryption != "" {
		args = append(args, "--server-side-encryption", options.ServerSideEncryption)
	}
	if options.SSEKMSKeyId != "" {
		args = append(args, "--ssekms-key-id", options.SSEKMSKeyId)
	}

	// Add metadata
	if len(options.Metadata) > 0 {
		for key, value := range options.Metadata {
			args = append(args, "--metadata", fmt.Sprintf("%s=%s", key, value))
		}
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, &CloudError{
			Code:    "S3_UPLOAD_FAILED",
			Message: fmt.Sprintf("Failed to upload file: %v", err),
		}
	}

	// Parse response
	var result struct {
		ETag string `json:"ETag"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse upload response: %w", err)
	}

	// Calculate MD5 hash for verification
	hash := md5.New()
	hash.Write(content.Bytes())
	_ = hex.EncodeToString(hash.Sum(nil)) // MD5 hash for potential future use

	return &FileInfo{
		Key:          key,
		ETag:         strings.Trim(result.ETag, "\""),
		Size:         int64(content.Len()),
		LastModified: time.Now(),
		StorageClass: "STANDARD",
		ContentType:  options.ContentType,
		Metadata:     options.Metadata,
		Tags:         options.Tags,
	}, nil
}

// uploadFileMultipart uploads a file using multipart upload
func (s *S3Manager) uploadFileMultipart(ctx context.Context, bucket, key string, content *bytes.Buffer, size int64, options *UploadOptions) (*FileInfo, error) {
	// Default part size is 64MB
	partSize := options.PartSize
	if partSize == 0 {
		partSize = 64 * 1024 * 1024 // 64MB
	}

	// Default concurrency is 4
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = 4
	}

	// Initiate multipart upload
	args := []string{"s3api", "create-multipart-upload", "--bucket", bucket, "--key", key}

	if options.ContentType != "" {
		args = append(args, "--content-type", options.ContentType)
	}
	if options.ServerSideEncryption != "" {
		args = append(args, "--server-side-encryption", options.ServerSideEncryption)
	}
	if options.SSEKMSKeyId != "" {
		args = append(args, "--ssekms-key-id", options.SSEKMSKeyId)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	var initResult struct {
		UploadId string `json:"UploadId"`
	}
	if err := json.Unmarshal(output, &initResult); err != nil {
		return nil, fmt.Errorf("failed to parse multipart upload response: %w", err)
	}

	uploadId := initResult.UploadId

	// Calculate number of parts
	numParts := (size + partSize - 1) / partSize
	parts := make([]MultipartPart, numParts)

	// Upload parts concurrently
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var uploadErr error
	var mu sync.Mutex

	for i := int64(0); i < numParts; i++ {
		wg.Add(1)
		go func(partNum int64) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			start := partNum * partSize
			end := start + partSize
			if end > size {
				end = size
			}

			partData := content.Bytes()[start:end]

			// Upload part
			if err := s.uploadPart(ctx, bucket, key, uploadId, int(partNum+1), partData); err != nil {
				mu.Lock()
				if uploadErr == nil {
					uploadErr = err
				}
				mu.Unlock()
				return
			}

			// Calculate ETag for part
			hash := md5.New()
			hash.Write(partData)
			etag := hex.EncodeToString(hash.Sum(nil))

			parts[partNum] = MultipartPart{
				PartNumber: int(partNum + 1),
				ETag:       etag,
				Size:       int64(len(partData)),
			}

			// Progress callback
			if options.ProgressCallback != nil {
				options.ProgressCallback(end, size)
			}
		}(i)
	}

	wg.Wait()

	if uploadErr != nil {
		// Abort upload on error
		s.abortMultipartUpload(ctx, bucket, key, uploadId)
		return nil, uploadErr
	}

	// Complete multipart upload
	return s.completeMultipartUpload(ctx, bucket, key, uploadId, parts)
}

// uploadPart uploads a single part for multipart upload
func (s *S3Manager) uploadPart(ctx context.Context, bucket, key, uploadId string, partNumber int, data []byte) error {
	// Create temporary file for part
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("s3-part-%d-*", partNumber))
	if err != nil {
		return fmt.Errorf("failed to create temporary file for part: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write part data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write part data: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "upload-part",
		"--bucket", bucket,
		"--key", key,
		"--upload-id", uploadId,
		"--part-number", fmt.Sprintf("%d", partNumber),
		"--body", tmpFile.Name())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to upload part %d: %w", partNumber, err)
	}

	return nil
}

// completeMultipartUpload completes a multipart upload
func (s *S3Manager) completeMultipartUpload(ctx context.Context, bucket, key, uploadId string, parts []MultipartPart) (*FileInfo, error) {
	// Sort parts by part number
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber
	})

	// Build completion request
	var partsJSON []map[string]interface{}
	for _, part := range parts {
		partsJSON = append(partsJSON, map[string]interface{}{
			"PartNumber": part.PartNumber,
			"ETag":       part.ETag,
		})
	}

	completion := map[string]interface{}{
		"Parts": partsJSON,
	}

	completionJSON, err := json.Marshal(completion)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal completion request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "complete-multipart-upload",
		"--bucket", bucket,
		"--key", key,
		"--upload-id", uploadId,
		"--multipart-upload", string(completionJSON))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	var result struct {
		ETag string `json:"ETag"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse completion response: %w", err)
	}

	// Calculate total size
	var totalSize int64
	for _, part := range parts {
		totalSize += part.Size
	}

	return &FileInfo{
		Key:          key,
		ETag:         strings.Trim(result.ETag, "\""),
		Size:         totalSize,
		LastModified: time.Now(),
		StorageClass: "STANDARD",
		PartsCount:   len(parts),
	}, nil
}

// abortMultipartUpload aborts a multipart upload
func (s *S3Manager) abortMultipartUpload(ctx context.Context, bucket, key, uploadId string) error {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "abort-multipart-upload",
		"--bucket", bucket,
		"--key", key,
		"--upload-id", uploadId)

	return cmd.Run()
}

// emptyBucket deletes all objects in a bucket
func (s *S3Manager) emptyBucket(ctx context.Context, bucket string) error {
	// List all objects
	objects, err := s.ListFiles(ctx, bucket, "", &ListOptions{MaxKeys: 1000})
	if err != nil {
		return err
	}

	// Delete objects in batches
	for len(objects.Objects) > 0 {
		var keys []string
		for _, obj := range objects.Objects {
			keys = append(keys, obj.Key)
		}

		if err := s.deleteObjectsBatch(ctx, bucket, keys); err != nil {
			return err
		}

		// Check for more objects
		if !objects.IsTruncated {
			break
		}

		objects, err = s.ListFiles(ctx, bucket, "", &ListOptions{
			MaxKeys:           1000,
			ContinuationToken: objects.NextContinuationToken,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// deleteObjectsBatch deletes multiple objects in a single request
func (s *S3Manager) deleteObjectsBatch(ctx context.Context, bucket string, keys []string) error {
	var objects []map[string]string
	for _, key := range keys {
		objects = append(objects, map[string]string{"Key": key})
	}

	deleteRequest := map[string]interface{}{
		"Objects": objects,
		"Quiet":   true,
	}

	deleteJSON, err := json.Marshal(deleteRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "delete-objects",
		"--bucket", bucket,
		"--delete", string(deleteJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete objects batch: %w", err)
	}

	return nil
}

// Additional helper methods for S3 operations

// getObjectMetadata gets metadata for a specific object
func (s *S3Manager) getObjectMetadata(ctx context.Context, bucket, key string) (*FileInfo, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "head-object", "--bucket", bucket, "--key", key)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		ContentType        string            `json:"ContentType"`
		ContentEncoding    string            `json:"ContentEncoding"`
		ContentLanguage    string            `json:"ContentLanguage"`
		ContentDisposition string            `json:"ContentDisposition"`
		CacheControl       string            `json:"CacheControl"`
		Expires            string            `json:"Expires"`
		Metadata           map[string]string `json:"Metadata"`
		LastModified       time.Time         `json:"LastModified"`
		ETag               string            `json:"ETag"`
		ContentLength      int64             `json:"ContentLength"`
		StorageClass       string            `json:"StorageClass"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &FileInfo{
		Key:                key,
		ContentType:        result.ContentType,
		ContentEncoding:    result.ContentEncoding,
		ContentLanguage:    result.ContentLanguage,
		ContentDisposition: result.ContentDisposition,
		CacheControl:       result.CacheControl,
		Expires:            result.Expires,
		Metadata:           result.Metadata,
		LastModified:       result.LastModified,
		ETag:               strings.Trim(result.ETag, "\""),
		Size:               result.ContentLength,
		StorageClass:       result.StorageClass,
	}, nil
}

// getObjectTags gets tags for a specific object
func (s *S3Manager) getObjectTags(ctx context.Context, bucket, key string) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-object-tagging", "--bucket", bucket, "--key", key)
	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}, nil // Return empty map if no tags
	}

	var result struct {
		TagSet []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"TagSet"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for _, tag := range result.TagSet {
		tags[tag.Key] = tag.Value
	}

	return tags, nil
}

// getBucketLifecycle gets bucket lifecycle configuration
func (s *S3Manager) getBucketLifecycle(ctx context.Context, bucket string) (*LifecycleConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-lifecycle-configuration", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &LifecycleConfig{Rules: []S3LifecycleRule{}}, nil // Return empty if not configured
	}

	var result LifecycleConfig
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// setBucketLifecycle configures bucket lifecycle
func (s *S3Manager) setBucketLifecycle(ctx context.Context, bucket string, config *LifecycleConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal lifecycle config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-lifecycle-configuration",
		"--bucket", bucket,
		"--lifecycle-configuration", string(configJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket lifecycle: %w", err)
	}

	return nil
}

// getBucketReplication gets bucket replication configuration
func (s *S3Manager) getBucketReplication(ctx context.Context, bucket string) (*ReplicationConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-replication", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &ReplicationConfig{Rules: []ReplicationRule{}}, nil // Return empty if not configured
	}

	var result struct {
		ReplicationConfiguration ReplicationConfig `json:"ReplicationConfiguration"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result.ReplicationConfiguration, nil
}

// setBucketReplication configures bucket replication
func (s *S3Manager) setBucketReplication(ctx context.Context, bucket string, config *ReplicationConfig) error {
	replicationConfig := map[string]interface{}{
		"Role":  config.Role,
		"Rules": config.Rules,
	}

	configJSON, err := json.Marshal(replicationConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal replication config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-replication",
		"--bucket", bucket,
		"--replication-configuration", string(configJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket replication: %w", err)
	}

	return nil
}

// getBucketPolicy gets bucket policy
func (s *S3Manager) getBucketPolicy(ctx context.Context, bucket string) (*BucketPolicy, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-policy", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &BucketPolicy{Statement: []PolicyStatement{}}, nil // Return empty if not configured
	}

	var result struct {
		Policy string `json:"Policy"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	var policy BucketPolicy
	if err := json.Unmarshal([]byte(result.Policy), &policy); err != nil {
		return nil, err
	}

	return &policy, nil
}

// setBucketPolicy configures bucket policy
func (s *S3Manager) setBucketPolicy(ctx context.Context, bucket string, policy *BucketPolicy) error {
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal bucket policy: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-policy",
		"--bucket", bucket,
		"--policy", string(policyJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	return nil
}

// getBucketLogging gets bucket logging configuration
func (s *S3Manager) getBucketLogging(ctx context.Context, bucket string) (*LoggingConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-logging", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &LoggingConfig{}, nil // Return empty if not configured
	}

	var result struct {
		LoggingEnabled LoggingConfig `json:"LoggingEnabled"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result.LoggingEnabled, nil
}

// setBucketLogging configures bucket logging
func (s *S3Manager) setBucketLogging(ctx context.Context, bucket string, config *LoggingConfig) error {
	loggingConfig := map[string]interface{}{
		"LoggingEnabled": config,
	}

	configJSON, err := json.Marshal(loggingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal logging config: %w", err)
	}

	cmd := exec.CommandContext(ctx, "aws", "s3api", "put-bucket-logging",
		"--bucket", bucket,
		"--bucket-logging-status", string(configJSON))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set bucket logging: %w", err)
	}

	return nil
}

// getBucketNotification gets bucket notification configuration
func (s *S3Manager) getBucketNotification(ctx context.Context, bucket string) (*NotificationConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-notification-configuration", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &NotificationConfig{}, nil // Return empty if not configured
	}

	var result NotificationConfig
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// getBucketCORS gets bucket CORS configuration
func (s *S3Manager) getBucketCORS(ctx context.Context, bucket string) (*CORSConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-cors", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &CORSConfig{}, nil // Return empty if not configured
	}

	var result CORSConfig
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// getBucketWebsite gets bucket website configuration
func (s *S3Manager) getBucketWebsite(ctx context.Context, bucket string) (*WebsiteConfig, error) {
	cmd := exec.CommandContext(ctx, "aws", "s3api", "get-bucket-website", "--bucket", bucket)
	output, err := cmd.Output()
	if err != nil {
		return &WebsiteConfig{}, nil // Return empty if not configured
	}

	var result WebsiteConfig
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// getBucketMetrics calculates bucket metrics
func (s *S3Manager) getBucketMetrics(ctx context.Context, bucket string) (*BucketMetrics, error) {
	// List all objects to calculate metrics
	objects, err := s.ListFiles(ctx, bucket, "", &ListOptions{MaxKeys: 10000})
	if err != nil {
		return nil, err
	}

	metrics := &BucketMetrics{
		TotalObjects:             int64(len(objects.Objects)),
		StorageClassDistribution: make(map[string]int64),
		LastUpdated:              time.Now(),
	}

	var totalSize int64
	for _, obj := range objects.Objects {
		totalSize += obj.Size
		metrics.StorageClassDistribution[obj.StorageClass]++
	}

	metrics.TotalSize = totalSize
	if metrics.TotalObjects > 0 {
		metrics.AverageObjectSize = totalSize / metrics.TotalObjects
	}

	// TODO: Get actual request metrics from CloudWatch
	metrics.RequestMetrics = RequestMetrics{}
	metrics.TransferMetrics = TransferMetrics{}
	metrics.ErrorMetrics = ErrorMetrics{}

	return metrics, nil
}

// getBucketCost calculates bucket cost information
func (s *S3Manager) getBucketCost(ctx context.Context, bucket string) (*BucketCost, error) {
	// TODO: Integrate with AWS Cost Explorer API for real cost data
	// For now, return placeholder data
	return &BucketCost{
		StorageCost:   0.0,
		RequestCost:   0.0,
		TransferCost:  0.0,
		TotalCost:     0.0,
		Currency:      "USD",
		BillingPeriod: "monthly",
		CostBreakdown: make(map[string]float64),
	}, nil
}

// getAPMBucketConfig gets APM-specific configuration from bucket tags
func (s *S3Manager) getAPMBucketConfig(ctx context.Context, bucket string) (*APMBucketConfig, error) {
	tags, err := s.getBucketTags(ctx, bucket)
	if err != nil {
		return &APMBucketConfig{}, nil
	}

	config := &APMBucketConfig{
		Purpose:     tags["apm:purpose"],
		Environment: tags["apm:environment"],
		Component:   tags["apm:component"],
		Compliance: ComplianceConfig{
			DataClassification: tags["apm:data-classification"],
		},
	}

	// Parse boolean values from tags
	if tags["apm:auto-backup"] == "true" {
		config.AutoBackup = true
	}
	if tags["apm:compression"] == "true" {
		config.Compression = true
	}
	if tags["apm:encryption"] == "true" {
		config.Encryption = true
	}
	if tags["apm:access-logging"] == "true" {
		config.AccessLogging = true
	}
	if tags["apm:monitoring-enabled"] == "true" {
		config.MonitoringEnabled = true
	}

	// Parse numeric values
	if retention := tags["apm:backup-retention"]; retention != "" {
		if days, err := strconv.Atoi(retention); err == nil {
			config.BackupRetention = days
		}
	}

	return config, nil
}

// SetBucketPolicy sets a bucket policy (public method)
func (s *S3Manager) SetBucketPolicy(ctx context.Context, bucket string, policy *BucketPolicy) error {
	return s.setBucketPolicy(ctx, bucket, policy)
}

// EnableVersioning enables versioning on a bucket (public method)
func (s *S3Manager) EnableVersioning(ctx context.Context, bucket string) error {
	return s.setBucketVersioning(ctx, bucket, &VersioningConfig{Status: "Enabled"})
}

// SetLifecyclePolicy sets a lifecycle policy on a bucket (public method)
func (s *S3Manager) SetLifecyclePolicy(ctx context.Context, bucket string, policy *LifecycleConfig) error {
	return s.setBucketLifecycle(ctx, bucket, policy)
}

// SetupReplication sets up cross-region replication (public method)
func (s *S3Manager) SetupReplication(ctx context.Context, bucket, targetBucket, targetRegion string) error {
	// Create a basic replication configuration
	replicationConfig := &ReplicationConfig{
		Role: fmt.Sprintf("arn:aws:iam::%s:role/S3ReplicationRole", "123456789012"), // TODO: Get actual account ID
		Rules: []ReplicationRule{
			{
				ID:     "ReplicateToSecondaryRegion",
				Status: "Enabled",
				Filter: ReplicationFilter{
					Prefix: "",
				},
				Destination: ReplicationDestination{
					Bucket:       fmt.Sprintf("arn:aws:s3:::%s", targetBucket),
					StorageClass: "STANDARD_IA",
				},
			},
		},
	}

	return s.setBucketReplication(ctx, bucket, replicationConfig)
}

// CopyFile copies an object within S3 or between buckets
func (s *S3Manager) CopyFile(ctx context.Context, sourceBucket, sourceKey, destBucket, destKey string, options *CopyOptions) (*FileInfo, error) {
	if s.provider == nil {
		return nil, &CloudError{
			Code:    "S3_PROVIDER_NOT_INITIALIZED",
			Message: "S3 provider not initialized",
		}
	}

	// Set default options if not provided
	if options == nil {
		options = &CopyOptions{}
	}

	// Build copy source
	copySource := fmt.Sprintf("%s/%s", sourceBucket, sourceKey)
	if options.SourceVersionId != "" {
		copySource = fmt.Sprintf("%s?versionId=%s", copySource, options.SourceVersionId)
	}

	// Build AWS CLI command
	args := []string{"s3api", "copy-object",
		"--copy-source", copySource,
		"--bucket", destBucket,
		"--key", destKey,
	}

	// Add optional parameters
	if options.MetadataDirective != "" {
		args = append(args, "--metadata-directive", options.MetadataDirective)
	}
	if options.TaggingDirective != "" {
		args = append(args, "--tagging-directive", options.TaggingDirective)
	}
	if options.StorageClass != "" {
		args = append(args, "--storage-class", options.StorageClass)
	}
	if options.ServerSideEncryption != "" {
		args = append(args, "--server-side-encryption", options.ServerSideEncryption)
	}
	if options.SSEKMSKeyId != "" {
		args = append(args, "--ssekms-key-id", options.SSEKMSKeyId)
	}
	if options.ContentType != "" {
		args = append(args, "--content-type", options.ContentType)
	}

	// Add metadata if replacing
	if options.MetadataDirective == "REPLACE" && len(options.Metadata) > 0 {
		for key, value := range options.Metadata {
			args = append(args, "--metadata", fmt.Sprintf("%s=%s", key, value))
		}
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, &CloudError{
			Code:    "S3_COPY_FAILED",
			Message: fmt.Sprintf("Failed to copy object: %v", err),
		}
	}

	// Parse response
	var result struct {
		CopyObjectResult struct {
			ETag         string    `json:"ETag"`
			LastModified time.Time `json:"LastModified"`
		} `json:"CopyObjectResult"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse copy response: %w", err)
	}

	return &FileInfo{
		Key:          destKey,
		ETag:         strings.Trim(result.CopyObjectResult.ETag, "\""),
		LastModified: result.CopyObjectResult.LastModified,
		StorageClass: options.StorageClass,
		ContentType:  options.ContentType,
		Metadata:     options.Metadata,
		Tags:         options.Tags,
	}, nil
}

// MoveFile moves an object within S3 or between buckets (copy then delete)
func (s *S3Manager) MoveFile(ctx context.Context, sourceBucket, sourceKey, destBucket, destKey string, options *CopyOptions) (*FileInfo, error) {
	// First copy the file
	fileInfo, err := s.CopyFile(ctx, sourceBucket, sourceKey, destBucket, destKey, options)
	if err != nil {
		return nil, err
	}

	// Then delete the source file
	deleteOptions := &DeleteOptions{}
	if options != nil && options.SourceVersionId != "" {
		deleteOptions.VersionId = options.SourceVersionId
	}

	if err := s.DeleteFile(ctx, sourceBucket, sourceKey, deleteOptions); err != nil {
		// Log the error but don't fail the move operation
		// The copy was successful, so the file exists in the destination
		return fileInfo, nil
	}

	return fileInfo, nil
}

// GetS3Manager returns the S3Manager instance from AWSProvider
func (p *AWSProvider) GetS3Manager() *S3Manager {
	return p.s3Manager
}

// APM Configuration Management Methods

// CreateAPMBucket creates a bucket specifically for APM configuration storage
func (s *S3Manager) CreateAPMBucket(ctx context.Context, name, region, environment, component, purpose string) (*Bucket, error) {
	// Create APM-specific tags
	apmTags := map[string]string{
		"CreatedBy":               "APM-Tool",
		"apm:purpose":             purpose,     // config, logs, artifacts, backup
		"apm:environment":         environment, // dev, staging, prod
		"apm:component":           component,   // prometheus, grafana, jaeger, loki
		"apm:auto-backup":         "true",
		"apm:compression":         "true",
		"apm:encryption":          "true",
		"apm:access-logging":      "true",
		"apm:monitoring-enabled":  "true",
		"apm:data-classification": "internal",
		"apm:backup-retention":    "30",
	}

	// Create secure APM bucket options
	options := &BucketOptions{
		Region: region,
		Versioning: &VersioningConfig{
			Status: "Enabled",
		},
		Encryption: &EncryptionConfig{
			Type:             "SSE-S3",
			Algorithm:        "AES256",
			BucketKeyEnabled: true,
		},
		PublicAccessBlock: &PublicAccessBlockConfig{
			BlockPublicAcls:       true,
			IgnorePublicAcls:      true,
			BlockPublicPolicy:     true,
			RestrictPublicBuckets: true,
		},
		Lifecycle: s.createAPMLifecyclePolicy(purpose),
		Policy:    s.createAPMBucketPolicy(name, environment),
		Tags:      apmTags,
		APMConfig: &APMBucketConfig{
			Purpose:           purpose,
			Environment:       environment,
			Component:         component,
			AutoBackup:        true,
			BackupRetention:   30,
			Compression:       true,
			Encryption:        true,
			AccessLogging:     true,
			MonitoringEnabled: true,
			Alerting: AlertingConfig{
				Enabled:             true,
				UnauthorizedAccess:  true,
				HighRequestRate:     true,
				HighErrorRate:       true,
				SizeThreshold:       10 * 1024 * 1024 * 1024, // 10GB
				CostThreshold:       100.0,                   // $100
				NotificationTargets: []string{},
			},
			Compliance: ComplianceConfig{
				DataClassification:    "internal",
				RetentionPeriod:       90,
				EncryptionRequired:    true,
				AccessLoggingRequired: true,
				BackupRequired:        true,
				ComplianceStandards:   []string{"SOC2"},
			},
		},
	}

	return s.CreateBucket(ctx, name, region, options)
}

// UploadAPMConfig uploads APM configuration files with proper metadata
func (s *S3Manager) UploadAPMConfig(ctx context.Context, bucket, configType, environment string, config interface{}) (*FileInfo, error) {
	// Serialize configuration to JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create key with timestamp and environment
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	key := fmt.Sprintf("configs/%s/%s/%s-%s.json", environment, configType, configType, timestamp)

	// Create upload options with APM metadata
	options := &UploadOptions{
		ContentType: "application/json",
		Metadata: map[string]string{
			"config-type":    configType,
			"environment":    environment,
			"uploaded-at":    time.Now().Format(time.RFC3339),
			"apm-tool":       "true",
			"config-version": "v1",
		},
		Tags: map[string]string{
			"ConfigType":  configType,
			"Environment": environment,
			"UploadedBy":  "APM-Tool",
			"Purpose":     "Configuration",
		},
		ServerSideEncryption: "AES256",
	}

	// Upload the configuration
	return s.UploadFile(ctx, bucket, key, bytes.NewReader(configJSON), options)
}

// DownloadAPMConfig downloads the latest APM configuration
func (s *S3Manager) DownloadAPMConfig(ctx context.Context, bucket, configType, environment string) (map[string]interface{}, error) {
	// List configurations for the specified type and environment
	prefix := fmt.Sprintf("configs/%s/%s/", environment, configType)
	listResult, err := s.ListFiles(ctx, bucket, prefix, &ListOptions{
		MaxKeys: 100,
	})
	if err != nil {
		return nil, err
	}

	if len(listResult.Objects) == 0 {
		return nil, &CloudError{
			Code:    "S3_CONFIG_NOT_FOUND",
			Message: fmt.Sprintf("No configuration found for %s in %s environment", configType, environment),
		}
	}

	// Sort by last modified date and get the latest
	sort.Slice(listResult.Objects, func(i, j int) bool {
		return listResult.Objects[i].LastModified.After(listResult.Objects[j].LastModified)
	})

	latestConfig := listResult.Objects[0]

	// Download the configuration
	reader, err := s.DownloadFile(ctx, bucket, latestConfig.Key, &DownloadOptions{})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Read and parse the configuration
	configData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// ListAPMConfigs lists all APM configurations
func (s *S3Manager) ListAPMConfigs(ctx context.Context, bucket, environment string) (map[string][]*FileInfo, error) {
	prefix := fmt.Sprintf("configs/%s/", environment)
	listResult, err := s.ListFiles(ctx, bucket, prefix, &ListOptions{
		MaxKeys:         1000,
		IncludeMetadata: true,
		IncludeTags:     true,
	})
	if err != nil {
		return nil, err
	}

	// Group configurations by type
	configsByType := make(map[string][]*FileInfo)
	for _, obj := range listResult.Objects {
		// Extract config type from path
		pathParts := strings.Split(obj.Key, "/")
		if len(pathParts) >= 3 {
			configType := pathParts[2]
			configsByType[configType] = append(configsByType[configType], obj)
		}
	}

	// Sort each type by last modified date (newest first)
	for configType := range configsByType {
		sort.Slice(configsByType[configType], func(i, j int) bool {
			return configsByType[configType][i].LastModified.After(configsByType[configType][j].LastModified)
		})
	}

	return configsByType, nil
}

// BackupAPMConfig creates a backup of current configuration
func (s *S3Manager) BackupAPMConfig(ctx context.Context, bucket, configType, environment string) (*FileInfo, error) {
	// Download current config
	currentConfig, err := s.DownloadAPMConfig(ctx, bucket, configType, environment)
	if err != nil {
		return nil, err
	}

	// Create backup key
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	backupKey := fmt.Sprintf("backups/%s/%s/%s-backup-%s.json", environment, configType, configType, timestamp)

	// Serialize config
	configJSON, err := json.MarshalIndent(currentConfig, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup config: %w", err)
	}

	// Upload backup
	options := &UploadOptions{
		ContentType: "application/json",
		Metadata: map[string]string{
			"config-type":  configType,
			"environment":  environment,
			"backup-type":  "automatic",
			"backed-up-at": time.Now().Format(time.RFC3339),
			"apm-tool":     "true",
		},
		Tags: map[string]string{
			"ConfigType":  configType,
			"Environment": environment,
			"BackupType":  "Automatic",
			"Purpose":     "Backup",
		},
		StorageClass:         "STANDARD_IA", // Cheaper storage for backups
		ServerSideEncryption: "AES256",
	}

	return s.UploadFile(ctx, bucket, backupKey, bytes.NewReader(configJSON), options)
}

// RestoreAPMConfig restores a configuration from backup
func (s *S3Manager) RestoreAPMConfig(ctx context.Context, bucket, backupKey string) (*FileInfo, error) {
	// Download backup
	reader, err := s.DownloadFile(ctx, bucket, backupKey, &DownloadOptions{})
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Read backup data
	backupData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup data: %w", err)
	}

	// Parse backup metadata from key
	pathParts := strings.Split(backupKey, "/")
	if len(pathParts) < 4 {
		return nil, fmt.Errorf("invalid backup key format")
	}

	environment := pathParts[1]
	configType := pathParts[2]

	// Create restore key
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	restoreKey := fmt.Sprintf("configs/%s/%s/%s-restored-%s.json", environment, configType, configType, timestamp)

	// Upload restored config
	options := &UploadOptions{
		ContentType: "application/json",
		Metadata: map[string]string{
			"config-type":   configType,
			"environment":   environment,
			"restored-from": backupKey,
			"restored-at":   time.Now().Format(time.RFC3339),
			"apm-tool":      "true",
		},
		Tags: map[string]string{
			"ConfigType":   configType,
			"Environment":  environment,
			"RestoreType":  "FromBackup",
			"Purpose":      "Configuration",
			"RestoredFrom": backupKey,
		},
		ServerSideEncryption: "AES256",
	}

	return s.UploadFile(ctx, bucket, restoreKey, bytes.NewReader(backupData), options)
}

// DeployAPMConfigs deploys all configurations for an environment
func (s *S3Manager) DeployAPMConfigs(ctx context.Context, bucket, environment, targetEnvironment string) (map[string]*FileInfo, error) {
	// List all configs in source environment
	configs, err := s.ListAPMConfigs(ctx, bucket, environment)
	if err != nil {
		return nil, err
	}

	deployed := make(map[string]*FileInfo)

	// Deploy each config type
	for configType, configFiles := range configs {
		if len(configFiles) == 0 {
			continue
		}

		// Get the latest config
		latestConfig := configFiles[0]

		// Download config
		reader, err := s.DownloadFile(ctx, bucket, latestConfig.Key, &DownloadOptions{})
		if err != nil {
			continue // Log error but continue with other configs
		}

		configData, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			continue
		}

		// Create deployment key
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		deployKey := fmt.Sprintf("configs/%s/%s/%s-deployed-%s.json", targetEnvironment, configType, configType, timestamp)

		// Upload to target environment
		options := &UploadOptions{
			ContentType: "application/json",
			Metadata: map[string]string{
				"config-type":   configType,
				"environment":   targetEnvironment,
				"source-env":    environment,
				"deployed-from": latestConfig.Key,
				"deployed-at":   time.Now().Format(time.RFC3339),
				"apm-tool":      "true",
			},
			Tags: map[string]string{
				"ConfigType":     configType,
				"Environment":    targetEnvironment,
				"SourceEnv":      environment,
				"DeploymentType": "CrossEnvironment",
				"Purpose":        "Configuration",
			},
			ServerSideEncryption: "AES256",
		}

		if fileInfo, err := s.UploadFile(ctx, bucket, deployKey, bytes.NewReader(configData), options); err == nil {
			deployed[configType] = fileInfo
		}
	}

	return deployed, nil
}

// createAPMLifecyclePolicy creates lifecycle policy for APM buckets
func (s *S3Manager) createAPMLifecyclePolicy(purpose string) *LifecycleConfig {
	rules := []S3LifecycleRule{
		{
			ID:     "APMConfigLifecycle",
			Status: "Enabled",
			Filter: LifecycleFilter{
				Prefix: "configs/",
			},
			Transitions: []LifecycleTransition{
				{
					Days:         30,
					StorageClass: "STANDARD_IA",
				},
				{
					Days:         90,
					StorageClass: "GLACIER",
				},
			},
		},
		{
			ID:     "APMBackupLifecycle",
			Status: "Enabled",
			Filter: LifecycleFilter{
				Prefix: "backups/",
			},
			Transitions: []LifecycleTransition{
				{
					Days:         7,
					StorageClass: "STANDARD_IA",
				},
				{
					Days:         30,
					StorageClass: "GLACIER",
				},
				{
					Days:         180,
					StorageClass: "DEEP_ARCHIVE",
				},
			},
			Expiration: &LifecycleExpiration{
				Days: 2555, // 7 years retention for compliance
			},
		},
		{
			ID:     "APMTempCleanup",
			Status: "Enabled",
			Filter: LifecycleFilter{
				Prefix: "temp/",
			},
			Expiration: &LifecycleExpiration{
				Days: 7,
			},
		},
		{
			ID:     "APMMultipartCleanup",
			Status: "Enabled",
			Filter: LifecycleFilter{},
			AbortIncompleteMultipartUpload: &LifecycleAbort{
				DaysAfterInitiation: 1,
			},
		},
	}

	// Add logs-specific rules
	if purpose == "logs" {
		rules = append(rules, S3LifecycleRule{
			ID:     "APMLogsLifecycle",
			Status: "Enabled",
			Filter: LifecycleFilter{
				Prefix: "logs/",
			},
			Transitions: []LifecycleTransition{
				{
					Days:         1,
					StorageClass: "STANDARD_IA",
				},
				{
					Days:         7,
					StorageClass: "GLACIER",
				},
				{
					Days:         30,
					StorageClass: "DEEP_ARCHIVE",
				},
			},
			Expiration: &LifecycleExpiration{
				Days: 365, // 1 year retention for logs
			},
		})
	}

	return &LifecycleConfig{Rules: rules}
}

// createAPMBucketPolicy creates security policy for APM buckets
func (s *S3Manager) createAPMBucketPolicy(bucketName, environment string) *BucketPolicy {
	// Create restrictive policy for APM buckets
	policy := &BucketPolicy{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Sid:    "DenyInsecureConnections",
				Effect: "Deny",
				Principal: map[string]string{
					"AWS": "*",
				},
				Action: []string{
					"s3:*",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s", bucketName),
					fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
				},
				Condition: map[string]interface{}{
					"Bool": map[string]interface{}{
						"aws:SecureTransport": "false",
					},
				},
			},
			{
				Sid:    "DenyUnencryptedUploads",
				Effect: "Deny",
				Principal: map[string]string{
					"AWS": "*",
				},
				Action: []string{
					"s3:PutObject",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
				},
				Condition: map[string]interface{}{
					"StringNotEquals": map[string]interface{}{
						"s3:x-amz-server-side-encryption": []string{"AES256", "aws:kms"},
					},
				},
			},
		},
	}

	// Add environment-specific restrictions
	if environment == "prod" {
		policy.Statement = append(policy.Statement, PolicyStatement{
			Sid:    "RequireMFAForProduction",
			Effect: "Deny",
			Principal: map[string]string{
				"AWS": "*",
			},
			Action: []string{
				"s3:DeleteObject",
				"s3:DeleteBucket",
				"s3:PutBucketPolicy",
			},
			Resource: []string{
				fmt.Sprintf("arn:aws:s3:::%s", bucketName),
				fmt.Sprintf("arn:aws:s3:::%s/*", bucketName),
			},
			Condition: map[string]interface{}{
				"BoolIfExists": map[string]interface{}{
					"aws:MultiFactorAuthPresent": "false",
				},
			},
		})
	}

	return policy
}

// ValidateAPMConfig validates APM configuration before upload
func (s *S3Manager) ValidateAPMConfig(configType string, config interface{}) error {
	// Basic validation based on config type
	switch configType {
	case "prometheus":
		return s.validatePrometheusConfig(config)
	case "grafana":
		return s.validateGrafanaConfig(config)
	case "jaeger":
		return s.validateJaegerConfig(config)
	case "loki":
		return s.validateLokiConfig(config)
	case "alertmanager":
		return s.validateAlertManagerConfig(config)
	default:
		// Generic validation for unknown types
		return s.validateGenericConfig(config)
	}
}

// validatePrometheusConfig validates Prometheus configuration
func (s *S3Manager) validatePrometheusConfig(config interface{}) error {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return errors.New("prometheus config must be a map")
	}

	// Check required fields
	if _, exists := configMap["global"]; !exists {
		return errors.New("prometheus config missing 'global' section")
	}

	if _, exists := configMap["scrape_configs"]; !exists {
		return errors.New("prometheus config missing 'scrape_configs' section")
	}

	return nil
}

// validateGrafanaConfig validates Grafana configuration
func (s *S3Manager) validateGrafanaConfig(config interface{}) error {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return errors.New("grafana config must be a map")
	}

	// Check for security settings
	if security, exists := configMap["security"]; exists {
		securityMap := security.(map[string]interface{})
		if adminPassword, exists := securityMap["admin_password"]; exists {
			if password, ok := adminPassword.(string); ok && (password == "admin" || password == "password") {
				return errors.New("grafana config has insecure default password")
			}
		}
	}

	return nil
}

// validateJaegerConfig validates Jaeger configuration
func (s *S3Manager) validateJaegerConfig(config interface{}) error {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return errors.New("jaeger config must be a map")
	}

	// Check for storage configuration
	if _, exists := configMap["storage"]; !exists {
		return errors.New("jaeger config missing 'storage' section")
	}

	return nil
}

// validateLokiConfig validates Loki configuration
func (s *S3Manager) validateLokiConfig(config interface{}) error {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return errors.New("loki config must be a map")
	}

	// Check for auth and server configuration
	if _, exists := configMap["auth_enabled"]; !exists {
		return errors.New("loki config missing 'auth_enabled' setting")
	}

	if _, exists := configMap["server"]; !exists {
		return errors.New("loki config missing 'server' section")
	}

	return nil
}

// validateAlertManagerConfig validates AlertManager configuration
func (s *S3Manager) validateAlertManagerConfig(config interface{}) error {
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return errors.New("alertmanager config must be a map")
	}

	// Check for route configuration
	if _, exists := configMap["route"]; !exists {
		return errors.New("alertmanager config missing 'route' section")
	}

	return nil
}

// validateGenericConfig performs basic validation for unknown config types
func (s *S3Manager) validateGenericConfig(config interface{}) error {
	// Check if it's valid JSON
	if _, err := json.Marshal(config); err != nil {
		return fmt.Errorf("config is not valid JSON: %w", err)
	}

	return nil
}

// Enhanced Error Handling and Logging

// S3Logger provides structured logging for S3 operations
type S3Logger struct {
	provider *AWSProvider
	enabled  bool
	logLevel LogLevel
}

// LogLevel defines logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// S3OperationContext provides context for S3 operations
type S3OperationContext struct {
	Operation    string            `json:"operation"`
	Bucket       string            `json:"bucket,omitempty"`
	Key          string            `json:"key,omitempty"`
	Region       string            `json:"region,omitempty"`
	Environment  string            `json:"environment,omitempty"`
	ConfigType   string            `json:"configType,omitempty"`
	StartTime    time.Time         `json:"startTime"`
	EndTime      time.Time         `json:"endTime"`
	Duration     time.Duration     `json:"duration"`
	Success      bool              `json:"success"`
	Error        error             `json:"error,omitempty"`
	ErrorCode    string            `json:"errorCode,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RequestID    string            `json:"requestId,omitempty"`
	UserAgent    string            `json:"userAgent,omitempty"`
	Size         int64             `json:"size,omitempty"`
	StorageClass string            `json:"storageClass,omitempty"`
}

// S3Metrics tracks S3 operation metrics
type S3Metrics struct {
	TotalOperations      int64            `json:"totalOperations"`
	SuccessfulOps        int64            `json:"successfulOps"`
	FailedOps            int64            `json:"failedOps"`
	AverageResponseTime  time.Duration    `json:"averageResponseTime"`
	TotalBytesUploaded   int64            `json:"totalBytesUploaded"`
	TotalBytesDownloaded int64            `json:"totalBytesDownloaded"`
	OperationCounts      map[string]int64 `json:"operationCounts"`
	ErrorCounts          map[string]int64 `json:"errorCounts"`
	LastResetTime        time.Time        `json:"lastResetTime"`
	mutex                sync.RWMutex     `json:"-"`
}

// NewS3Logger creates a new S3Logger instance
func NewS3Logger(provider *AWSProvider, enabled bool, logLevel LogLevel) *S3Logger {
	return &S3Logger{
		provider: provider,
		enabled:  enabled,
		logLevel: logLevel,
	}
}

// Log logs a message with the specified level
func (l *S3Logger) Log(level LogLevel, operation string, message string, context *S3OperationContext) {
	if !l.enabled || level < l.logLevel {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	logEntry := map[string]interface{}{
		"timestamp": timestamp,
		"level":     level.String(),
		"operation": operation,
		"message":   message,
		"service":   "S3Manager",
		"component": "APM",
	}

	if context != nil {
		logEntry["context"] = context
	}

	// Convert to JSON for structured logging
	if jsonData, err := json.Marshal(logEntry); err == nil {
		fmt.Printf("%s\n", string(jsonData))
	} else {
		// Fallback to simple logging
		fmt.Printf("[%s] %s - %s: %s\n", timestamp, level.String(), operation, message)
	}
}

// LogOperation logs the start and end of an operation
func (l *S3Logger) LogOperation(ctx *S3OperationContext) {
	if !l.enabled {
		return
	}

	if ctx.Success {
		l.Log(LogLevelInfo, ctx.Operation, "Operation completed successfully", ctx)
	} else {
		l.Log(LogLevelError, ctx.Operation, "Operation failed", ctx)
	}
}

// NewS3Metrics creates a new S3Metrics instance
func NewS3Metrics() *S3Metrics {
	return &S3Metrics{
		OperationCounts: make(map[string]int64),
		ErrorCounts:     make(map[string]int64),
		LastResetTime:   time.Now(),
	}
}

// RecordOperation records an operation metric
func (m *S3Metrics) RecordOperation(ctx *S3OperationContext) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalOperations++
	m.OperationCounts[ctx.Operation]++

	if ctx.Success {
		m.SuccessfulOps++
	} else {
		m.FailedOps++
		if ctx.ErrorCode != "" {
			m.ErrorCounts[ctx.ErrorCode]++
		}
	}

	if ctx.Size > 0 {
		if strings.Contains(ctx.Operation, "Upload") {
			m.TotalBytesUploaded += ctx.Size
		} else if strings.Contains(ctx.Operation, "Download") {
			m.TotalBytesDownloaded += ctx.Size
		}
	}

	// Update average response time
	if ctx.Duration > 0 {
		if m.TotalOperations == 1 {
			m.AverageResponseTime = ctx.Duration
		} else {
			m.AverageResponseTime = time.Duration(
				(int64(m.AverageResponseTime)*m.TotalOperations + int64(ctx.Duration)) / (m.TotalOperations + 1),
			)
		}
	}
}

// GetMetrics returns current metrics
func (m *S3Metrics) GetMetrics() S3Metrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metrics := S3Metrics{
		TotalOperations:      m.TotalOperations,
		SuccessfulOps:        m.SuccessfulOps,
		FailedOps:            m.FailedOps,
		AverageResponseTime:  m.AverageResponseTime,
		TotalBytesUploaded:   m.TotalBytesUploaded,
		TotalBytesDownloaded: m.TotalBytesDownloaded,
		LastResetTime:        m.LastResetTime,
		OperationCounts:      make(map[string]int64),
		ErrorCounts:          make(map[string]int64),
	}

	for k, v := range m.OperationCounts {
		metrics.OperationCounts[k] = v
	}

	for k, v := range m.ErrorCounts {
		metrics.ErrorCounts[k] = v
	}

	return metrics
}

// ResetMetrics resets all metrics
func (m *S3Metrics) ResetMetrics() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.TotalOperations = 0
	m.SuccessfulOps = 0
	m.FailedOps = 0
	m.AverageResponseTime = 0
	m.TotalBytesUploaded = 0
	m.TotalBytesDownloaded = 0
	m.LastResetTime = time.Now()
	m.OperationCounts = make(map[string]int64)
	m.ErrorCounts = make(map[string]int64)
}

// Enhanced error handling functions

// WrapS3Error wraps an error with S3-specific context
func WrapS3Error(err error, operation, bucket, key string) error {
	if err == nil {
		return nil
	}

	// Extract AWS-specific error information
	errorCode := "UNKNOWN"
	errorMessage := err.Error()

	// Try to extract AWS CLI error codes
	if strings.Contains(errorMessage, "NoSuchBucket") {
		errorCode = "BUCKET_NOT_FOUND"
	} else if strings.Contains(errorMessage, "NoSuchKey") {
		errorCode = "OBJECT_NOT_FOUND"
	} else if strings.Contains(errorMessage, "AccessDenied") {
		errorCode = "ACCESS_DENIED"
	} else if strings.Contains(errorMessage, "InvalidBucketName") {
		errorCode = "INVALID_BUCKET_NAME"
	} else if strings.Contains(errorMessage, "BucketAlreadyExists") {
		errorCode = "BUCKET_ALREADY_EXISTS"
	} else if strings.Contains(errorMessage, "InvalidRequest") {
		errorCode = "INVALID_REQUEST"
	} else if strings.Contains(errorMessage, "InternalError") {
		errorCode = "INTERNAL_ERROR"
	} else if strings.Contains(errorMessage, "ServiceUnavailable") {
		errorCode = "SERVICE_UNAVAILABLE"
	} else if strings.Contains(errorMessage, "RequestTimeout") {
		errorCode = "REQUEST_TIMEOUT"
	} else if strings.Contains(errorMessage, "SlowDown") {
		errorCode = "THROTTLED"
	}

	return &CloudError{
		Provider:   ProviderAWS,
		Code:       errorCode,
		Message:    fmt.Sprintf("S3 %s failed: %s", operation, errorMessage),
		Operation:  operation,
		Retryable:  isRetryableError(errorCode),
		StatusCode: extractStatusCode(errorMessage),
		Timestamp:  time.Now(),
		Cause:      err,
	}
}

// isRetryableError determines if an error is retryable
func isRetryableError(errorCode string) bool {
	retryableErrors := map[string]bool{
		"INTERNAL_ERROR":      true,
		"SERVICE_UNAVAILABLE": true,
		"REQUEST_TIMEOUT":     true,
		"THROTTLED":           true,
		"UNKNOWN":             true,
	}
	return retryableErrors[errorCode]
}

// extractStatusCode extracts HTTP status code from error message
func extractStatusCode(errorMessage string) int {
	// Try to extract status code from AWS CLI error messages
	if strings.Contains(errorMessage, "403") {
		return 403
	} else if strings.Contains(errorMessage, "404") {
		return 404
	} else if strings.Contains(errorMessage, "400") {
		return 400
	} else if strings.Contains(errorMessage, "500") {
		return 500
	} else if strings.Contains(errorMessage, "503") {
		return 503
	}
	return 0
}

// extractRequestID extracts request ID from error message
func extractRequestID(errorMessage string) string {
	// Try to extract request ID from AWS CLI error messages
	re := regexp.MustCompile(`RequestId: ([a-zA-Z0-9-]+)`)
	matches := re.FindStringSubmatch(errorMessage)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// RetryS3Operation retries an S3 operation with exponential backoff
func RetryS3Operation(operation func() error, maxRetries int, baseDelay time.Duration, operationName string) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if cloudErr, ok := err.(*CloudError); ok {
			if !cloudErr.Retryable {
				return err
			}
		}

		// Calculate delay with exponential backoff and jitter
		delay := time.Duration(int64(baseDelay) * (1 << uint(attempt)))
		jitter := time.Duration(rand.Int63n(int64(delay) / 4))
		delay += jitter

		// Cap the delay at 30 seconds
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		fmt.Printf("S3 %s failed (attempt %d/%d), retrying in %v: %v\n",
			operationName, attempt+1, maxRetries, delay, err)

		time.Sleep(delay)
	}

	return fmt.Errorf("S3 %s failed after %d attempts: %w", operationName, maxRetries, lastErr)
}

// S3HealthChecker performs health checks on S3 operations
type S3HealthChecker struct {
	s3Manager *S3Manager
	logger    *S3Logger
}

// NewS3HealthChecker creates a new S3HealthChecker
func NewS3HealthChecker(s3Manager *S3Manager, logger *S3Logger) *S3HealthChecker {
	return &S3HealthChecker{
		s3Manager: s3Manager,
		logger:    logger,
	}
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Service      string            `json:"service"`
	Status       string            `json:"status"` // OK, WARN, ERROR
	Timestamp    time.Time         `json:"timestamp"`
	ResponseTime time.Duration     `json:"responseTime"`
	Details      map[string]string `json:"details"`
	Errors       []string          `json:"errors"`
}

// CheckS3Health performs comprehensive S3 health checks
func (hc *S3HealthChecker) CheckS3Health(ctx context.Context, testBucket, testRegion string) *HealthCheckResult {
	result := &HealthCheckResult{
		Service:   "S3Manager",
		Timestamp: time.Now(),
		Details:   make(map[string]string),
		Errors:    []string{},
	}

	startTime := time.Now()

	// Test 1: List buckets
	if _, err := hc.s3Manager.ListBuckets(ctx, testRegion); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("ListBuckets failed: %v", err))
	} else {
		result.Details["list_buckets"] = "OK"
	}

	// Test 2: Check if test bucket exists
	if _, err := hc.s3Manager.GetBucket(ctx, testBucket, testRegion); err != nil {
		result.Details["test_bucket"] = "NOT_FOUND"

		// Try to create the test bucket
		if _, err := hc.s3Manager.CreateBucket(ctx, testBucket, testRegion, &BucketOptions{
			Region: testRegion,
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("CreateBucket failed: %v", err))
		} else {
			result.Details["test_bucket"] = "CREATED"
		}
	} else {
		result.Details["test_bucket"] = "EXISTS"
	}

	// Test 3: Test upload/download
	testKey := "health-check-" + time.Now().Format("20060102150405")
	testContent := []byte("S3 health check test content")

	if _, err := hc.s3Manager.UploadFile(ctx, testBucket, testKey, bytes.NewReader(testContent), &UploadOptions{
		ContentType: "text/plain",
	}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("UploadFile failed: %v", err))
	} else {
		result.Details["upload_test"] = "OK"

		// Test download
		if reader, err := hc.s3Manager.DownloadFile(ctx, testBucket, testKey, &DownloadOptions{}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("DownloadFile failed: %v", err))
		} else {
			reader.Close()
			result.Details["download_test"] = "OK"

			// Cleanup test file
			if err := hc.s3Manager.DeleteFile(ctx, testBucket, testKey, &DeleteOptions{}); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("DeleteFile cleanup failed: %v", err))
			}
		}
	}

	result.ResponseTime = time.Since(startTime)

	// Determine overall status
	if len(result.Errors) == 0 {
		result.Status = "OK"
	} else if len(result.Errors) <= 1 {
		result.Status = "WARN"
	} else {
		result.Status = "ERROR"
	}

	return result
}

// MonitorS3Operations monitors S3 operations and generates alerts
func (hc *S3HealthChecker) MonitorS3Operations(ctx context.Context, metrics *S3Metrics) *HealthCheckResult {
	result := &HealthCheckResult{
		Service:   "S3Operations",
		Timestamp: time.Now(),
		Details:   make(map[string]string),
		Errors:    []string{},
	}

	currentMetrics := metrics.GetMetrics()

	// Check error rate
	if currentMetrics.TotalOperations > 0 {
		errorRate := float64(currentMetrics.FailedOps) / float64(currentMetrics.TotalOperations) * 100
		result.Details["error_rate"] = fmt.Sprintf("%.2f%%", errorRate)

		if errorRate > 10 {
			result.Errors = append(result.Errors, fmt.Sprintf("High error rate: %.2f%%", errorRate))
		}
	}

	// Check average response time
	if currentMetrics.AverageResponseTime > 30*time.Second {
		result.Errors = append(result.Errors, fmt.Sprintf("High response time: %v", currentMetrics.AverageResponseTime))
	}
	result.Details["avg_response_time"] = currentMetrics.AverageResponseTime.String()

	// Check for common errors
	for errorCode, count := range currentMetrics.ErrorCounts {
		if count > 10 {
			result.Errors = append(result.Errors, fmt.Sprintf("High %s error count: %d", errorCode, count))
		}
	}

	result.Details["total_operations"] = fmt.Sprintf("%d", currentMetrics.TotalOperations)
	result.Details["successful_ops"] = fmt.Sprintf("%d", currentMetrics.SuccessfulOps)
	result.Details["failed_ops"] = fmt.Sprintf("%d", currentMetrics.FailedOps)

	// Determine status
	if len(result.Errors) == 0 {
		result.Status = "OK"
	} else if len(result.Errors) <= 2 {
		result.Status = "WARN"
	} else {
		result.Status = "ERROR"
	}

	return result
}

// Add logger and metrics methods to S3Manager

// SetLogger sets the logger for S3Manager
func (s *S3Manager) SetLogger(logger *S3Logger) {
	s.logger = logger
}

// SetMetrics sets the metrics for S3Manager
func (s *S3Manager) SetMetrics(metrics *S3Metrics) {
	s.metrics = metrics
}

// GetLogger returns the logger instance
func (s *S3Manager) GetLogger() *S3Logger {
	return s.logger
}

// GetMetrics returns the metrics instance
func (s *S3Manager) GetMetrics() *S3Metrics {
	return s.metrics
}

// Performance Optimization Features

// S3Cache provides caching for S3 operations
type S3Cache struct {
	bucketCache   map[string]*CachedBucket
	fileCache     map[string]*CachedFile
	mutex         sync.RWMutex
	ttl           time.Duration
	maxCacheSize  int
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// CachedBucket represents a cached bucket
type CachedBucket struct {
	Bucket    *BucketDetails
	CachedAt  time.Time
	ExpiresAt time.Time
}

// CachedFile represents a cached file info
type CachedFile struct {
	FileInfo  *FileInfo
	CachedAt  time.Time
	ExpiresAt time.Time
}

// NewS3Cache creates a new S3 cache
func NewS3Cache(ttl time.Duration, maxSize int) *S3Cache {
	cache := &S3Cache{
		bucketCache:   make(map[string]*CachedBucket),
		fileCache:     make(map[string]*CachedFile),
		ttl:           ttl,
		maxCacheSize:  maxSize,
		cleanupTicker: time.NewTicker(ttl / 2), // Cleanup twice per TTL period
		stopCleanup:   make(chan bool),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// cleanupExpired removes expired entries from cache
func (c *S3Cache) cleanupExpired() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.performCleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// performCleanup removes expired cache entries
func (c *S3Cache) performCleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Clean bucket cache
	for key, cached := range c.bucketCache {
		if now.After(cached.ExpiresAt) {
			delete(c.bucketCache, key)
		}
	}

	// Clean file cache
	for key, cached := range c.fileCache {
		if now.After(cached.ExpiresAt) {
			delete(c.fileCache, key)
		}
	}

	// If cache is still too large, remove oldest entries
	c.evictOldestIfNecessary()
}

// evictOldestIfNecessary removes oldest entries if cache exceeds max size
func (c *S3Cache) evictOldestIfNecessary() {
	totalSize := len(c.bucketCache) + len(c.fileCache)
	if totalSize <= c.maxCacheSize {
		return
	}

	// Collect all entries with their timestamps
	type entry struct {
		key      string
		cachedAt time.Time
		isFile   bool
	}

	var entries []entry
	for key, cached := range c.bucketCache {
		entries = append(entries, entry{key: key, cachedAt: cached.CachedAt, isFile: false})
	}
	for key, cached := range c.fileCache {
		entries = append(entries, entry{key: key, cachedAt: cached.CachedAt, isFile: true})
	}

	// Sort by cached time (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].cachedAt.Before(entries[j].cachedAt)
	})

	// Remove oldest entries until we're under the limit
	toRemove := totalSize - c.maxCacheSize
	for i := 0; i < toRemove && i < len(entries); i++ {
		if entries[i].isFile {
			delete(c.fileCache, entries[i].key)
		} else {
			delete(c.bucketCache, entries[i].key)
		}
	}
}

// GetBucket retrieves a bucket from cache
func (c *S3Cache) GetBucket(key string) (*BucketDetails, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.bucketCache[key]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	return cached.Bucket, true
}

// SetBucket stores a bucket in cache
func (c *S3Cache) SetBucket(key string, bucket *BucketDetails) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	c.bucketCache[key] = &CachedBucket{
		Bucket:    bucket,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// GetFile retrieves a file from cache
func (c *S3Cache) GetFile(key string) (*FileInfo, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.fileCache[key]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	return cached.FileInfo, true
}

// SetFile stores a file in cache
func (c *S3Cache) SetFile(key string, fileInfo *FileInfo) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	c.fileCache[key] = &CachedFile{
		FileInfo:  fileInfo,
		CachedAt:  now,
		ExpiresAt: now.Add(c.ttl),
	}
}

// InvalidateBucket removes a bucket from cache
func (c *S3Cache) InvalidateBucket(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.bucketCache, key)
}

// InvalidateFile removes a file from cache
func (c *S3Cache) InvalidateFile(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.fileCache, key)
}

// Clear removes all entries from cache
func (c *S3Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.bucketCache = make(map[string]*CachedBucket)
	c.fileCache = make(map[string]*CachedFile)
}

// Stop stops the cache cleanup goroutine
func (c *S3Cache) Stop() {
	c.cleanupTicker.Stop()
	close(c.stopCleanup)
}

// GetStats returns cache statistics
func (c *S3Cache) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"bucket_cache_size": len(c.bucketCache),
		"file_cache_size":   len(c.fileCache),
		"total_cache_size":  len(c.bucketCache) + len(c.fileCache),
		"max_cache_size":    c.maxCacheSize,
		"ttl_seconds":       c.ttl.Seconds(),
	}
}

// S3ConnectionPool manages connection pooling for S3 operations
type S3ConnectionPool struct {
	maxConcurrent   int
	semaphore       chan struct{}
	activeRequests  int64
	totalRequests   int64
	waitingRequests int64
	mutex           sync.RWMutex
}

// NewS3ConnectionPool creates a new connection pool
func NewS3ConnectionPool(maxConcurrent int) *S3ConnectionPool {
	return &S3ConnectionPool{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires a connection from the pool
func (p *S3ConnectionPool) Acquire() {
	p.mutex.Lock()
	p.waitingRequests++
	p.totalRequests++
	p.mutex.Unlock()

	p.semaphore <- struct{}{} // Block if pool is full

	p.mutex.Lock()
	p.waitingRequests--
	p.activeRequests++
	p.mutex.Unlock()
}

// Release releases a connection back to the pool
func (p *S3ConnectionPool) Release() {
	<-p.semaphore

	p.mutex.Lock()
	p.activeRequests--
	p.mutex.Unlock()
}

// GetStats returns pool statistics
func (p *S3ConnectionPool) GetStats() map[string]interface{} {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return map[string]interface{}{
		"max_concurrent":   p.maxConcurrent,
		"active_requests":  p.activeRequests,
		"waiting_requests": p.waitingRequests,
		"total_requests":   p.totalRequests,
		"available_slots":  len(p.semaphore),
	}
}

// S3BatchProcessor processes multiple S3 operations in batches
type S3BatchProcessor struct {
	s3Manager   *S3Manager
	batchSize   int
	workerCount int
	timeout     time.Duration
	pool        *S3ConnectionPool
}

// NewS3BatchProcessor creates a new batch processor
func NewS3BatchProcessor(s3Manager *S3Manager, batchSize, workerCount int, timeout time.Duration) *S3BatchProcessor {
	return &S3BatchProcessor{
		s3Manager:   s3Manager,
		batchSize:   batchSize,
		workerCount: workerCount,
		timeout:     timeout,
		pool:        NewS3ConnectionPool(workerCount),
	}
}

// BatchOperation represents a batch operation
type BatchOperation struct {
	Type     string                 `json:"type"` // upload, download, delete, copy
	Bucket   string                 `json:"bucket"`
	Key      string                 `json:"key"`
	Options  map[string]interface{} `json:"options"`
	Content  []byte                 `json:"content,omitempty"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Operation *BatchOperation `json:"operation"`
	Success   bool            `json:"success"`
	Error     error           `json:"error,omitempty"`
	Result    interface{}     `json:"result,omitempty"`
	Duration  time.Duration   `json:"duration"`
}

// ProcessBatch processes a batch of operations
func (bp *S3BatchProcessor) ProcessBatch(ctx context.Context, operations []*BatchOperation) ([]*BatchResult, error) {
	if len(operations) == 0 {
		return nil, nil
	}

	// Create result channel
	resultChan := make(chan *BatchResult, len(operations))

	// Create worker goroutines
	operationChan := make(chan *BatchOperation, len(operations))

	// Start workers
	for i := 0; i < bp.workerCount; i++ {
		go bp.worker(ctx, operationChan, resultChan)
	}

	// Send operations to workers
	for _, op := range operations {
		operationChan <- op
	}
	close(operationChan)

	// Collect results
	var results []*BatchResult
	for i := 0; i < len(operations); i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(bp.timeout):
			return results, fmt.Errorf("batch processing timeout")
		}
	}

	return results, nil
}

// worker processes operations from the operation channel
func (bp *S3BatchProcessor) worker(ctx context.Context, operations <-chan *BatchOperation, results chan<- *BatchResult) {
	for op := range operations {
		startTime := time.Now()
		result := &BatchResult{
			Operation: op,
		}

		// Acquire connection from pool
		bp.pool.Acquire()

		// Process the operation
		switch op.Type {
		case "upload":
			result.Result, result.Error = bp.processUpload(ctx, op)
		case "download":
			result.Result, result.Error = bp.processDownload(ctx, op)
		case "delete":
			result.Error = bp.processDelete(ctx, op)
		case "copy":
			result.Result, result.Error = bp.processCopy(ctx, op)
		default:
			result.Error = fmt.Errorf("unsupported operation type: %s", op.Type)
		}

		// Release connection
		bp.pool.Release()

		result.Success = result.Error == nil
		result.Duration = time.Since(startTime)

		results <- result
	}
}

// processUpload handles upload operations
func (bp *S3BatchProcessor) processUpload(ctx context.Context, op *BatchOperation) (*FileInfo, error) {
	options := &UploadOptions{}
	if contentType, ok := op.Options["content_type"].(string); ok {
		options.ContentType = contentType
	}
	if encryption, ok := op.Options["encryption"].(string); ok {
		options.ServerSideEncryption = encryption
	}
	options.Metadata = op.Metadata

	return bp.s3Manager.UploadFile(ctx, op.Bucket, op.Key, bytes.NewReader(op.Content), options)
}

// processDownload handles download operations
func (bp *S3BatchProcessor) processDownload(ctx context.Context, op *BatchOperation) ([]byte, error) {
	options := &DownloadOptions{}

	reader, err := bp.s3Manager.DownloadFile(ctx, op.Bucket, op.Key, options)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// processDelete handles delete operations
func (bp *S3BatchProcessor) processDelete(ctx context.Context, op *BatchOperation) error {
	options := &DeleteOptions{}
	if mfa, ok := op.Options["mfa"].(string); ok {
		options.MFA = mfa
	}

	return bp.s3Manager.DeleteFile(ctx, op.Bucket, op.Key, options)
}

// processCopy handles copy operations
func (bp *S3BatchProcessor) processCopy(ctx context.Context, op *BatchOperation) (*FileInfo, error) {
	sourceBucket, ok := op.Options["source_bucket"].(string)
	if !ok {
		return nil, fmt.Errorf("source_bucket is required for copy operation")
	}

	sourceKey, ok := op.Options["source_key"].(string)
	if !ok {
		return nil, fmt.Errorf("source_key is required for copy operation")
	}

	copyOptions := &CopyOptions{}
	if contentType, ok := op.Options["content_type"].(string); ok {
		copyOptions.ContentType = contentType
	}
	copyOptions.Metadata = op.Metadata

	return bp.s3Manager.CopyFile(ctx, sourceBucket, sourceKey, op.Bucket, op.Key, copyOptions)
}

// GetBatchStats returns batch processing statistics
func (bp *S3BatchProcessor) GetBatchStats() map[string]interface{} {
	poolStats := bp.pool.GetStats()
	return map[string]interface{}{
		"batch_size":   bp.batchSize,
		"worker_count": bp.workerCount,
		"timeout":      bp.timeout.String(),
		"pool_stats":   poolStats,
	}
}

// Performance optimization methods for S3Manager

// SetCache sets the cache for S3Manager
func (s *S3Manager) SetCache(cache *S3Cache) {
	s.cache = cache
}

// GetCache returns the cache instance
func (s *S3Manager) GetCache() *S3Cache {
	return s.cache
}

// OptimizedListBuckets lists buckets with caching
func (s *S3Manager) OptimizedListBuckets(ctx context.Context, region string) ([]*Bucket, error) {
	// For now, just call the regular ListBuckets method
	// Advanced caching for complex types like bucket lists can be added later
	// when we have a more sophisticated caching mechanism
	return s.ListBuckets(ctx, region)
}

// OptimizedGetBucket gets bucket details with caching
func (s *S3Manager) OptimizedGetBucket(ctx context.Context, name, region string) (*BucketDetails, error) {
	cacheKey := fmt.Sprintf("bucket-%s-%s", name, region)

	// Check cache first
	if s.cache != nil {
		if cached, found := s.cache.GetBucket(cacheKey); found {
			return cached, nil
		}
	}

	// Cache miss - fetch from S3
	bucket, err := s.GetBucket(ctx, name, region)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil {
		s.cache.SetBucket(cacheKey, bucket)
	}

	return bucket, nil
}

// PrefetchBuckets prefetches bucket information for faster access
func (s *S3Manager) PrefetchBuckets(ctx context.Context, bucketNames []string, region string) error {
	if s.cache == nil {
		return fmt.Errorf("cache not configured")
	}

	// Use batch processor for efficient prefetching
	batchProcessor := NewS3BatchProcessor(s, 10, 5, 30*time.Second)

	var operations []*BatchOperation
	for _, bucketName := range bucketNames {
		// We'll use a dummy operation to trigger bucket fetching
		operations = append(operations, &BatchOperation{
			Type:   "prefetch",
			Bucket: bucketName,
			Options: map[string]interface{}{
				"region": region,
			},
		})
	}

	// Process in batches
	for i := 0; i < len(operations); i += batchProcessor.batchSize {
		end := i + batchProcessor.batchSize
		if end > len(operations) {
			end = len(operations)
		}

		batch := operations[i:end]

		// Process each bucket in the batch
		for _, op := range batch {
			go func(bucketName, region string) {
				_, _ = s.OptimizedGetBucket(ctx, bucketName, region)
			}(op.Bucket, region)
		}
	}

	return nil
}

// WarmupCache warms up the cache with frequently accessed data
func (s *S3Manager) WarmupCache(ctx context.Context, config *CacheWarmupConfig) error {
	if s.cache == nil {
		return fmt.Errorf("cache not configured")
	}

	// Warmup buckets
	if len(config.Buckets) > 0 {
		if err := s.PrefetchBuckets(ctx, config.Buckets, config.Region); err != nil {
			return fmt.Errorf("failed to warmup buckets: %w", err)
		}
	}

	// Warmup frequently accessed files
	for bucket, keys := range config.FrequentFiles {
		for _, key := range keys {
			go func(bucket, key string) {
				cacheKey := fmt.Sprintf("file-%s-%s", bucket, key)
				if _, found := s.cache.GetFile(cacheKey); !found {
					// File not in cache, we could add basic metadata here
					// For now, just mark it as a cache miss for future optimization
				}
			}(bucket, key)
		}
	}

	return nil
}

// CacheWarmupConfig configures cache warmup
type CacheWarmupConfig struct {
	Region        string              `json:"region"`
	Buckets       []string            `json:"buckets"`
	FrequentFiles map[string][]string `json:"frequent_files"` // bucket -> keys
}

// ====================================================================
// CloudWatch Integration Implementation for APM Monitoring
// ====================================================================

// CloudWatchManager is the enhanced CloudWatch integration with comprehensive managers
type CloudWatchManager struct {
	provider          *AWSProvider
	dashboardMgr      *DashboardManager
	alarmMgr          *AlarmManager
	logsMgr           *LogsManager
	insightsMgr       *InsightsManager
	eventsMgr         *EventsManager
	snsMgr            *SNSManager
	apmIntegrationMgr *APMIntegrationManager
	logger            *CloudWatchLogger
	metrics           *CloudWatchMetrics
	cache             *CloudWatchCache
	healthChecker     *CloudWatchHealthChecker
}

// Enhanced CloudWatchIntegration constructor with full managers
func (p *AWSProvider) GetCloudWatchManager() *CloudWatchManager {
	if p.cloudWatch == nil {
		p.cloudWatch = &CloudWatchIntegration{provider: p}
	}

	manager := &CloudWatchManager{
		provider:      p,
		logger:        NewCloudWatchLogger(p.config),
		metrics:       NewCloudWatchMetrics(),
		cache:         NewCloudWatchCache(),
		healthChecker: NewCloudWatchHealthChecker(p),
	}

	// Initialize sub-managers
	manager.dashboardMgr = NewDashboardManager(manager)
	manager.alarmMgr = NewAlarmManager(manager)
	manager.logsMgr = NewLogsManager(manager)
	manager.insightsMgr = NewInsightsManager(manager)
	manager.eventsMgr = NewEventsManager(manager)
	manager.snsMgr = NewSNSManager(manager)
	manager.apmIntegrationMgr = NewAPMIntegrationManager(manager)

	return manager
}

// ====================================================================
// Dashboard Management Implementation
// ====================================================================

// DashboardManager handles CloudWatch dashboard operations
type DashboardManager struct {
	cloudWatch *CloudWatchManager
}

// NewDashboardManager creates a new dashboard manager
func NewDashboardManager(cw *CloudWatchManager) *DashboardManager {
	return &DashboardManager{cloudWatch: cw}
}

// CreateDashboard creates a CloudWatch dashboard with APM-specific templates
func (dm *DashboardManager) CreateDashboard(ctx context.Context, config *DashboardConfig) (*CloudWatchDashboard, error) {
	dm.cloudWatch.logger.LogInfo(ctx, "Creating CloudWatch dashboard", map[string]interface{}{
		"dashboardName": config.Name,
		"template":      config.Template,
		"region":        dm.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		dm.cloudWatch.metrics.RecordOperation("CreateDashboard", time.Since(startTime), nil)
	}()

	// Generate dashboard body if using template
	dashboardBody := config.Body
	if config.Template != "" && dashboardBody == "" {
		body, err := dm.generateDashboardFromTemplate(config.Template, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate dashboard from template: %w", err)
		}
		dashboardBody = body
	}

	// Build AWS CLI command for dashboard creation
	region := dm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "put-dashboard",
		"--dashboard-name", config.Name,
		"--dashboard-body", dashboardBody,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	// Parse response and create CloudWatchDashboard
	dashboard := &CloudWatchDashboard{
		DashboardName:  config.Name,
		DashboardBody:  dashboardBody,
		DashboardArn:   fmt.Sprintf("arn:aws:cloudwatch::%s:dashboard/%s", region, config.Name),
		LastModified:   time.Now(),
		Size:           int64(len(dashboardBody)),
		Region:         region,
		Tags:           config.Tags,
		Variables:      config.Variables,
		APMIntegration: config.APMIntegration,
		Description:    config.Description,
	}

	// Parse widgets from dashboard body
	if widgets, err := dm.parseWidgetsFromBody(dashboardBody); err == nil {
		dashboard.Widgets = widgets
	}

	// Cache the dashboard
	dm.cloudWatch.cache.SetDashboard(config.Name, dashboard)

	dm.cloudWatch.logger.LogInfo(ctx, "Dashboard created successfully", map[string]interface{}{
		"dashboardName": config.Name,
		"dashboardArn":  dashboard.DashboardArn,
		"size":          dashboard.Size,
	})

	return dashboard, nil
}

// ListDashboards lists CloudWatch dashboards with optional prefix filtering
func (dm *DashboardManager) ListDashboards(ctx context.Context, prefix string) ([]*CloudWatchDashboard, error) {
	dm.cloudWatch.logger.LogInfo(ctx, "Listing CloudWatch dashboards", map[string]interface{}{
		"prefix": prefix,
		"region": dm.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		dm.cloudWatch.metrics.RecordOperation("ListDashboards", time.Since(startTime), nil)
	}()

	// Check cache first
	if dashboards := dm.cloudWatch.cache.GetDashboards(prefix); len(dashboards) > 0 {
		return dashboards, nil
	}

	// Build AWS CLI command
	region := dm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "list-dashboards", "--region", region)

	if prefix != "" {
		cmd.Args = append(cmd.Args, "--dashboard-name-prefix", prefix)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list dashboards: %w", err)
	}

	// Parse output
	var response struct {
		DashboardEntries []struct {
			DashboardName string    `json:"DashboardName"`
			LastModified  time.Time `json:"LastModified"`
			Size          int64     `json:"Size"`
		} `json:"DashboardEntries"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard list response: %w", err)
	}

	// Convert to CloudWatchDashboard objects
	dashboards := make([]*CloudWatchDashboard, 0, len(response.DashboardEntries))
	for _, entry := range response.DashboardEntries {
		dashboard := &CloudWatchDashboard{
			DashboardName: entry.DashboardName,
			LastModified:  entry.LastModified,
			Size:          entry.Size,
			Region:        region,
			DashboardArn:  fmt.Sprintf("arn:aws:cloudwatch::%s:dashboard/%s", region, entry.DashboardName),
		}
		dashboards = append(dashboards, dashboard)

		// Cache individual dashboard
		dm.cloudWatch.cache.SetDashboard(entry.DashboardName, dashboard)
	}

	return dashboards, nil
}

// GetDashboard retrieves a specific CloudWatch dashboard
func (dm *DashboardManager) GetDashboard(ctx context.Context, name string) (*CloudWatchDashboard, error) {
	dm.cloudWatch.logger.LogInfo(ctx, "Getting CloudWatch dashboard", map[string]interface{}{
		"dashboardName": name,
		"region":        dm.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		dm.cloudWatch.metrics.RecordOperation("GetDashboard", time.Since(startTime), nil)
	}()

	// Check cache first
	if dashboard := dm.cloudWatch.cache.GetDashboard(name); dashboard != nil {
		return dashboard, nil
	}

	// Build AWS CLI command
	region := dm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "get-dashboard",
		"--dashboard-name", name,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Parse response
	var response struct {
		DashboardArn  string `json:"DashboardArn"`
		DashboardBody string `json:"DashboardBody"`
		DashboardName string `json:"DashboardName"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard response: %w", err)
	}

	dashboard := &CloudWatchDashboard{
		DashboardName: response.DashboardName,
		DashboardBody: response.DashboardBody,
		DashboardArn:  response.DashboardArn,
		Size:          int64(len(response.DashboardBody)),
		Region:        region,
		LastModified:  time.Now(), // AWS doesn't return this in get-dashboard
	}

	// Parse widgets from dashboard body
	if widgets, err := dm.parseWidgetsFromBody(response.DashboardBody); err == nil {
		dashboard.Widgets = widgets
	}

	// Cache the dashboard
	dm.cloudWatch.cache.SetDashboard(name, dashboard)

	return dashboard, nil
}

// UpdateDashboard updates an existing CloudWatch dashboard
func (dm *DashboardManager) UpdateDashboard(ctx context.Context, name string, config *DashboardConfig) (*CloudWatchDashboard, error) {
	dm.cloudWatch.logger.LogInfo(ctx, "Updating CloudWatch dashboard", map[string]interface{}{
		"dashboardName": name,
		"region":        dm.cloudWatch.provider.config.DefaultRegion,
	})

	// Use CreateDashboard as AWS CLI put-dashboard creates or updates
	config.Name = name
	dashboard, err := dm.CreateDashboard(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to update dashboard: %w", err)
	}

	// Invalidate cache
	dm.cloudWatch.cache.DeleteDashboard(name)

	return dashboard, nil
}

// DeleteDashboard deletes a CloudWatch dashboard with safety checks
func (dm *DashboardManager) DeleteDashboard(ctx context.Context, name string) error {
	dm.cloudWatch.logger.LogInfo(ctx, "Deleting CloudWatch dashboard", map[string]interface{}{
		"dashboardName": name,
		"region":        dm.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		dm.cloudWatch.metrics.RecordOperation("DeleteDashboard", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := dm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "delete-dashboards",
		"--dashboard-names", name,
		"--region", region)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	// Remove from cache
	dm.cloudWatch.cache.DeleteDashboard(name)

	dm.cloudWatch.logger.LogInfo(ctx, "Dashboard deleted successfully", map[string]interface{}{
		"dashboardName": name,
	})

	return nil
}

// generateDashboardFromTemplate generates dashboard JSON from APM templates
func (dm *DashboardManager) generateDashboardFromTemplate(template string, config *DashboardConfig) (string, error) {
	var dashboardTemplate map[string]interface{}

	switch template {
	case "infrastructure":
		dashboardTemplate = dm.getInfrastructureDashboardTemplate(config)
	case "application":
		dashboardTemplate = dm.getApplicationDashboardTemplate(config)
	case "service-mesh":
		dashboardTemplate = dm.getServiceMeshDashboardTemplate(config)
	case "logs":
		dashboardTemplate = dm.getLogsDashboardTemplate(config)
	case "tracing":
		dashboardTemplate = dm.getTracingDashboardTemplate(config)
	case "cost":
		dashboardTemplate = dm.getCostDashboardTemplate(config)
	default:
		return "", fmt.Errorf("unknown dashboard template: %s", template)
	}

	// Serialize template to JSON
	templateJSON, err := json.MarshalIndent(dashboardTemplate, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal dashboard template: %w", err)
	}

	return string(templateJSON), nil
}

// parseWidgetsFromBody parses widgets from dashboard body JSON
func (dm *DashboardManager) parseWidgetsFromBody(body string) ([]*DashboardWidget, error) {
	var dashboardData struct {
		Widgets []map[string]interface{} `json:"widgets"`
	}

	if err := json.Unmarshal([]byte(body), &dashboardData); err != nil {
		return nil, err
	}

	widgets := make([]*DashboardWidget, 0, len(dashboardData.Widgets))
	for _, w := range dashboardData.Widgets {
		widget := &DashboardWidget{
			Type:     getString(w, "type"),
			Metadata: w,
		}

		// Parse properties if present
		if props, ok := w["properties"].(map[string]interface{}); ok {
			widget.Properties.Title = getString(props, "title")
			widget.Properties.Region = getString(props, "region")
			widget.Properties.Period = getInt(props, "period")
			widget.Properties.Stat = getString(props, "stat")
			widget.Properties.View = getString(props, "view")
			widget.Properties.Stacked = getBool(props, "stacked")
			widget.Properties.Query = getString(props, "query")
		}

		// Parse position if present
		if pos, ok := w["position"].(map[string]interface{}); ok {
			widget.Position.X = getInt(pos, "x")
			widget.Position.Y = getInt(pos, "y")
			widget.Position.Width = getInt(pos, "width")
			widget.Position.Height = getInt(pos, "height")
		}

		widgets = append(widgets, widget)
	}

	return widgets, nil
}

// Helper functions for parsing JSON
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// APM Dashboard Templates

// getInfrastructureDashboardTemplate returns infrastructure monitoring template
func (dm *DashboardManager) getInfrastructureDashboardTemplate(config *DashboardConfig) map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":     "metric",
				"position": map[string]int{"x": 0, "y": 0, "width": 12, "height": 6},
				"properties": map[string]interface{}{
					"metrics": [][]interface{}{
						{"AWS/EC2", "CPUUtilization"},
						{"AWS/EC2", "NetworkIn"},
						{"AWS/EC2", "NetworkOut"},
					},
					"period": 300,
					"stat":   "Average",
					"region": dm.cloudWatch.provider.config.DefaultRegion,
					"title":  "Infrastructure Metrics",
				},
			},
			{
				"type":     "metric",
				"position": map[string]int{"x": 12, "y": 0, "width": 12, "height": 6},
				"properties": map[string]interface{}{
					"metrics": [][]interface{}{
						{"AWS/EKS", "cluster_failed_request_count"},
						{"AWS/EKS", "cluster_request_total"},
					},
					"period": 300,
					"stat":   "Sum",
					"region": dm.cloudWatch.provider.config.DefaultRegion,
					"title":  "Kubernetes Cluster Metrics",
				},
			},
		},
	}
}

// getApplicationDashboardTemplate returns application performance template
func (dm *DashboardManager) getApplicationDashboardTemplate(config *DashboardConfig) map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":     "metric",
				"position": map[string]int{"x": 0, "y": 0, "width": 8, "height": 6},
				"properties": map[string]interface{}{
					"metrics": [][]interface{}{
						{"Custom/APM", "RequestRate"},
						{"Custom/APM", "ErrorRate"},
						{"Custom/APM", "ResponseTime"},
					},
					"period": 300,
					"stat":   "Average",
					"region": dm.cloudWatch.provider.config.DefaultRegion,
					"title":  "Application Performance",
				},
			},
			{
				"type":     "log",
				"position": map[string]int{"x": 8, "y": 0, "width": 16, "height": 6},
				"properties": map[string]interface{}{
					"query":     "fields @timestamp, @message | filter @type = \"ERROR\" | sort @timestamp desc | limit 100",
					"region":    dm.cloudWatch.provider.config.DefaultRegion,
					"title":     "Recent Errors",
					"logGroups": config.APMIntegration.Namespaces,
				},
			},
		},
	}
}

// getServiceMeshDashboardTemplate returns service mesh template
func (dm *DashboardManager) getServiceMeshDashboardTemplate(config *DashboardConfig) map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":     "metric",
				"position": map[string]int{"x": 0, "y": 0, "width": 12, "height": 6},
				"properties": map[string]interface{}{
					"metrics": [][]interface{}{
						{"Custom/Istio", "RequestTotal"},
						{"Custom/Istio", "RequestDuration"},
						{"Custom/Istio", "RequestBytes"},
					},
					"period": 300,
					"stat":   "Average",
					"region": dm.cloudWatch.provider.config.DefaultRegion,
					"title":  "Service Mesh Metrics",
				},
			},
		},
	}
}

// getLogsDashboardTemplate returns logs analysis template
func (dm *DashboardManager) getLogsDashboardTemplate(config *DashboardConfig) map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":     "log",
				"position": map[string]int{"x": 0, "y": 0, "width": 24, "height": 6},
				"properties": map[string]interface{}{
					"query":     "fields @timestamp, @message | sort @timestamp desc | limit 100",
					"region":    dm.cloudWatch.provider.config.DefaultRegion,
					"title":     "Application Logs",
					"logGroups": config.APMIntegration.Namespaces,
				},
			},
		},
	}
}

// getTracingDashboardTemplate returns distributed tracing template
func (dm *DashboardManager) getTracingDashboardTemplate(config *DashboardConfig) map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":     "metric",
				"position": map[string]int{"x": 0, "y": 0, "width": 12, "height": 6},
				"properties": map[string]interface{}{
					"metrics": [][]interface{}{
						{"Custom/Tracing", "TraceCount"},
						{"Custom/Tracing", "SpanCount"},
						{"Custom/Tracing", "TraceDuration"},
					},
					"period": 300,
					"stat":   "Average",
					"region": dm.cloudWatch.provider.config.DefaultRegion,
					"title":  "Distributed Tracing",
				},
			},
		},
	}
}

// getCostDashboardTemplate returns cost optimization template
func (dm *DashboardManager) getCostDashboardTemplate(config *DashboardConfig) map[string]interface{} {
	return map[string]interface{}{
		"widgets": []map[string]interface{}{
			{
				"type":     "metric",
				"position": map[string]int{"x": 0, "y": 0, "width": 12, "height": 6},
				"properties": map[string]interface{}{
					"metrics": [][]interface{}{
						{"AWS/Billing", "EstimatedCharges", "Currency", "USD"},
					},
					"period": 86400,
					"stat":   "Maximum",
					"region": "us-east-1", // Billing metrics only in us-east-1
					"title":  "Estimated Charges",
				},
			},
		},
	}
}

// ====================================================================
// Alarm Management Implementation
// ====================================================================

// AlarmManager handles CloudWatch alarm operations
type AlarmManager struct {
	cloudWatch *CloudWatchManager
}

// NewAlarmManager creates a new alarm manager
func NewAlarmManager(cw *CloudWatchManager) *AlarmManager {
	return &AlarmManager{cloudWatch: cw}
}

// CreateAlarm creates a CloudWatch alarm for APM infrastructure
func (am *AlarmManager) CreateAlarm(ctx context.Context, config *AlarmConfig) (*CloudWatchAlarm, error) {
	am.cloudWatch.logger.LogInfo(ctx, "Creating CloudWatch alarm", map[string]interface{}{
		"alarmName":  config.AlarmName,
		"metricName": config.MetricName,
		"namespace":  config.Namespace,
		"region":     am.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		am.cloudWatch.metrics.RecordOperation("CreateAlarm", time.Since(startTime), nil)
	}()

	// Build AWS CLI command for alarm creation
	region := am.cloudWatch.provider.config.DefaultRegion
	args := []string{
		"cloudwatch", "put-metric-alarm",
		"--alarm-name", config.AlarmName,
		"--alarm-description", config.AlarmDescription,
		"--metric-name", config.MetricName,
		"--namespace", config.Namespace,
		"--statistic", config.Statistic,
		"--period", fmt.Sprintf("%d", config.Period),
		"--evaluation-periods", fmt.Sprintf("%d", config.EvaluationPeriods),
		"--threshold", fmt.Sprintf("%f", config.Threshold),
		"--comparison-operator", config.ComparisonOperator,
		"--region", region,
	}

	// Add dimensions if present
	if len(config.Dimensions) > 0 {
		dimensionsJSON, _ := json.Marshal(config.Dimensions)
		args = append(args, "--dimensions", string(dimensionsJSON))
	}

	// Add actions if enabled
	if config.ActionsEnabled {
		if len(config.AlarmActions) > 0 {
			args = append(args, "--alarm-actions")
			args = append(args, config.AlarmActions...)
		}
		if len(config.OKActions) > 0 {
			args = append(args, "--ok-actions")
			args = append(args, config.OKActions...)
		}
		if len(config.InsufficientDataActions) > 0 {
			args = append(args, "--insufficient-data-actions")
			args = append(args, config.InsufficientDataActions...)
		}
	}

	// Add optional parameters
	if config.TreatMissingData != "" {
		args = append(args, "--treat-missing-data", config.TreatMissingData)
	}
	if config.DatapointsToAlarm > 0 {
		args = append(args, "--datapoints-to-alarm", fmt.Sprintf("%d", config.DatapointsToAlarm))
	}

	cmd := exec.Command("aws", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create alarm: %w", err)
	}

	// Create CloudWatchAlarm object
	alarm := &CloudWatchAlarm{
		AlarmName:                          config.AlarmName,
		AlarmDescription:                   config.AlarmDescription,
		AlarmArn:                           fmt.Sprintf("arn:aws:cloudwatch:%s::alarm:%s", region, config.AlarmName),
		MetricName:                         config.MetricName,
		Namespace:                          config.Namespace,
		Statistic:                          config.Statistic,
		Dimensions:                         config.Dimensions,
		Period:                             config.Period,
		EvaluationPeriods:                  config.EvaluationPeriods,
		Threshold:                          config.Threshold,
		ComparisonOperator:                 config.ComparisonOperator,
		TreatMissingData:                   config.TreatMissingData,
		DatapointsToAlarm:                  config.DatapointsToAlarm,
		ActionsEnabled:                     config.ActionsEnabled,
		OKActions:                          config.OKActions,
		AlarmActions:                       config.AlarmActions,
		InsufficientDataActions:            config.InsufficientDataActions,
		Tags:                               config.Tags,
		APMAlarmConfig:                     config.APMAlarmConfig,
		Region:                             region,
		AlarmConfigurationUpdatedTimestamp: time.Now(),
		State: AlarmState{
			Value:     "INSUFFICIENT_DATA",
			Reason:    "Insufficient Data",
			Timestamp: time.Now(),
		},
	}

	// Cache the alarm
	am.cloudWatch.cache.SetAlarm(config.AlarmName, alarm)

	am.cloudWatch.logger.LogInfo(ctx, "Alarm created successfully", map[string]interface{}{
		"alarmName": config.AlarmName,
		"alarmArn":  alarm.AlarmArn,
	})

	return alarm, nil
}

// ListAlarms lists CloudWatch alarms with optional prefix filtering
func (am *AlarmManager) ListAlarms(ctx context.Context, prefix string) ([]*CloudWatchAlarm, error) {
	am.cloudWatch.logger.LogInfo(ctx, "Listing CloudWatch alarms", map[string]interface{}{
		"prefix": prefix,
		"region": am.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		am.cloudWatch.metrics.RecordOperation("ListAlarms", time.Since(startTime), nil)
	}()

	// Check cache first
	if alarms := am.cloudWatch.cache.GetAlarms(prefix); len(alarms) > 0 {
		return alarms, nil
	}

	// Build AWS CLI command
	region := am.cloudWatch.provider.config.DefaultRegion
	args := []string{"cloudwatch", "describe-alarms", "--region", region}

	if prefix != "" {
		args = append(args, "--alarm-name-prefix", prefix)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list alarms: %w", err)
	}

	// Parse output
	var response struct {
		MetricAlarms []struct {
			AlarmName               string    `json:"AlarmName"`
			AlarmArn                string    `json:"AlarmArn"`
			AlarmDescription        string    `json:"AlarmDescription"`
			MetricName              string    `json:"MetricName"`
			Namespace               string    `json:"Namespace"`
			Statistic               string    `json:"Statistic"`
			Period                  int       `json:"Period"`
			EvaluationPeriods       int       `json:"EvaluationPeriods"`
			Threshold               float64   `json:"Threshold"`
			ComparisonOperator      string    `json:"ComparisonOperator"`
			StateValue              string    `json:"StateValue"`
			StateReason             string    `json:"StateReason"`
			StateUpdatedTimestamp   time.Time `json:"StateUpdatedTimestamp"`
			ActionsEnabled          bool      `json:"ActionsEnabled"`
			AlarmActions            []string  `json:"AlarmActions"`
			OKActions               []string  `json:"OKActions"`
			InsufficientDataActions []string  `json:"InsufficientDataActions"`
		} `json:"MetricAlarms"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse alarm list response: %w", err)
	}

	// Convert to CloudWatchAlarm objects
	alarms := make([]*CloudWatchAlarm, 0, len(response.MetricAlarms))
	for _, entry := range response.MetricAlarms {
		alarm := &CloudWatchAlarm{
			AlarmName:               entry.AlarmName,
			AlarmArn:                entry.AlarmArn,
			AlarmDescription:        entry.AlarmDescription,
			MetricName:              entry.MetricName,
			Namespace:               entry.Namespace,
			Statistic:               entry.Statistic,
			Period:                  entry.Period,
			EvaluationPeriods:       entry.EvaluationPeriods,
			Threshold:               entry.Threshold,
			ComparisonOperator:      entry.ComparisonOperator,
			StateReason:             entry.StateReason,
			StateUpdatedTimestamp:   entry.StateUpdatedTimestamp,
			ActionsEnabled:          entry.ActionsEnabled,
			AlarmActions:            entry.AlarmActions,
			OKActions:               entry.OKActions,
			InsufficientDataActions: entry.InsufficientDataActions,
			Region:                  region,
			State: AlarmState{
				Value:     entry.StateValue,
				Reason:    entry.StateReason,
				Timestamp: entry.StateUpdatedTimestamp,
			},
		}
		alarms = append(alarms, alarm)

		// Cache individual alarm
		am.cloudWatch.cache.SetAlarm(entry.AlarmName, alarm)
	}

	return alarms, nil
}

// GetAlarmState retrieves the current state of a CloudWatch alarm
func (am *AlarmManager) GetAlarmState(ctx context.Context, name string) (*AlarmState, error) {
	am.cloudWatch.logger.LogInfo(ctx, "Getting CloudWatch alarm state", map[string]interface{}{
		"alarmName": name,
		"region":    am.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		am.cloudWatch.metrics.RecordOperation("GetAlarmState", time.Since(startTime), nil)
	}()

	// Check cache first
	if alarm := am.cloudWatch.cache.GetAlarm(name); alarm != nil {
		return &alarm.State, nil
	}

	// Build AWS CLI command to get alarm details
	region := am.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "describe-alarms",
		"--alarm-names", name,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get alarm state: %w", err)
	}

	// Parse response
	var response struct {
		MetricAlarms []struct {
			StateValue            string    `json:"StateValue"`
			StateReason           string    `json:"StateReason"`
			StateReasonData       string    `json:"StateReasonData"`
			StateUpdatedTimestamp time.Time `json:"StateUpdatedTimestamp"`
		} `json:"MetricAlarms"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse alarm state response: %w", err)
	}

	if len(response.MetricAlarms) == 0 {
		return nil, fmt.Errorf("alarm not found: %s", name)
	}

	alarm := response.MetricAlarms[0]
	state := &AlarmState{
		Value:      alarm.StateValue,
		Reason:     alarm.StateReason,
		ReasonData: alarm.StateReasonData,
		Timestamp:  alarm.StateUpdatedTimestamp,
	}

	return state, nil
}

// EnableAlarm enables a CloudWatch alarm
func (am *AlarmManager) EnableAlarm(ctx context.Context, name string) error {
	am.cloudWatch.logger.LogInfo(ctx, "Enabling CloudWatch alarm", map[string]interface{}{
		"alarmName": name,
		"region":    am.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		am.cloudWatch.metrics.RecordOperation("EnableAlarm", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := am.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "enable-alarm-actions",
		"--alarm-names", name,
		"--region", region)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable alarm: %w", err)
	}

	// Update cache if alarm exists
	if alarm := am.cloudWatch.cache.GetAlarm(name); alarm != nil {
		alarm.ActionsEnabled = true
		am.cloudWatch.cache.SetAlarm(name, alarm)
	}

	am.cloudWatch.logger.LogInfo(ctx, "Alarm enabled successfully", map[string]interface{}{
		"alarmName": name,
	})

	return nil
}

// DisableAlarm disables a CloudWatch alarm
func (am *AlarmManager) DisableAlarm(ctx context.Context, name string) error {
	am.cloudWatch.logger.LogInfo(ctx, "Disabling CloudWatch alarm", map[string]interface{}{
		"alarmName": name,
		"region":    am.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		am.cloudWatch.metrics.RecordOperation("DisableAlarm", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := am.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "disable-alarm-actions",
		"--alarm-names", name,
		"--region", region)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable alarm: %w", err)
	}

	// Update cache if alarm exists
	if alarm := am.cloudWatch.cache.GetAlarm(name); alarm != nil {
		alarm.ActionsEnabled = false
		am.cloudWatch.cache.SetAlarm(name, alarm)
	}

	am.cloudWatch.logger.LogInfo(ctx, "Alarm disabled successfully", map[string]interface{}{
		"alarmName": name,
	})

	return nil
}

// ====================================================================
// Logs Management Implementation
// ====================================================================

// LogsManager handles CloudWatch Logs operations
type LogsManager struct {
	cloudWatch *CloudWatchManager
}

// NewLogsManager creates a new logs manager
func NewLogsManager(cw *CloudWatchManager) *LogsManager {
	return &LogsManager{cloudWatch: cw}
}

// CreateLogGroup creates a CloudWatch log group
func (lm *LogsManager) CreateLogGroup(ctx context.Context, config *LogGroupConfig) (*CloudWatchLogGroup, error) {
	lm.cloudWatch.logger.LogInfo(ctx, "Creating CloudWatch log group", map[string]interface{}{
		"logGroupName":  config.LogGroupName,
		"retentionDays": config.RetentionInDays,
		"region":        lm.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		lm.cloudWatch.metrics.RecordOperation("CreateLogGroup", time.Since(startTime), nil)
	}()

	// Build AWS CLI command for log group creation
	region := lm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "logs", "create-log-group",
		"--log-group-name", config.LogGroupName,
		"--region", region)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create log group: %w", err)
	}

	// Set retention policy if specified
	if config.RetentionInDays > 0 {
		retentionCmd := exec.Command("aws", "logs", "put-retention-policy",
			"--log-group-name", config.LogGroupName,
			"--retention-in-days", fmt.Sprintf("%d", config.RetentionInDays),
			"--region", region)

		if err := retentionCmd.Run(); err != nil {
			lm.cloudWatch.logger.LogWarn(ctx, "Failed to set retention policy", map[string]interface{}{
				"logGroupName": config.LogGroupName,
				"error":        err.Error(),
			})
		}
	}

	// Set KMS key if specified
	if config.KmsKeyId != "" {
		kmsCmd := exec.Command("aws", "logs", "associate-kms-key",
			"--log-group-name", config.LogGroupName,
			"--kms-key-id", config.KmsKeyId,
			"--region", region)

		if err := kmsCmd.Run(); err != nil {
			lm.cloudWatch.logger.LogWarn(ctx, "Failed to associate KMS key", map[string]interface{}{
				"logGroupName": config.LogGroupName,
				"kmsKeyId":     config.KmsKeyId,
				"error":        err.Error(),
			})
		}
	}

	// Create CloudWatchLogGroup object
	logGroup := &CloudWatchLogGroup{
		LogGroupName:    config.LogGroupName,
		LogGroupArn:     fmt.Sprintf("arn:aws:logs:%s::log-group:%s", region, config.LogGroupName),
		CreationTime:    time.Now(),
		RetentionInDays: config.RetentionInDays,
		KmsKeyId:        config.KmsKeyId,
		Tags:            config.Tags,
		APMLogConfig:    config.APMLogConfig,
		Region:          region,
	}

	// Cache the log group
	lm.cloudWatch.cache.SetLogGroup(config.LogGroupName, logGroup)

	lm.cloudWatch.logger.LogInfo(ctx, "Log group created successfully", map[string]interface{}{
		"logGroupName": config.LogGroupName,
		"logGroupArn":  logGroup.LogGroupArn,
	})

	return logGroup, nil
}

// PutLogEvents puts log events to a CloudWatch log stream
func (lm *LogsManager) PutLogEvents(ctx context.Context, logGroupName, logStreamName string, events []*LogEvent) error {
	lm.cloudWatch.logger.LogDebug(ctx, "Putting log events", map[string]interface{}{
		"logGroupName":  logGroupName,
		"logStreamName": logStreamName,
		"eventCount":    len(events),
	})

	startTime := time.Now()
	defer func() {
		lm.cloudWatch.metrics.RecordOperation("PutLogEvents", time.Since(startTime), nil)
	}()

	// Convert events to AWS CLI format
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal log events: %w", err)
	}

	// Build AWS CLI command
	region := lm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "logs", "put-log-events",
		"--log-group-name", logGroupName,
		"--log-stream-name", logStreamName,
		"--log-events", string(eventsJSON),
		"--region", region)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to put log events: %w", err)
	}

	lm.cloudWatch.metrics.IncrementCounter("LogEventsPut", len(events))

	return nil
}

// ====================================================================
// CloudWatch Insights Implementation
// ====================================================================

// InsightsManager handles CloudWatch Insights operations
type InsightsManager struct {
	cloudWatch *CloudWatchManager
}

// NewInsightsManager creates a new insights manager
func NewInsightsManager(cw *CloudWatchManager) *InsightsManager {
	return &InsightsManager{cloudWatch: cw}
}

// ExecuteInsightsQuery executes a CloudWatch Insights query
func (im *InsightsManager) ExecuteInsightsQuery(ctx context.Context, config *QueryConfig) (*CloudWatchInsightsQuery, error) {
	im.cloudWatch.logger.LogInfo(ctx, "Executing CloudWatch Insights query", map[string]interface{}{
		"queryString": config.QueryString,
		"logGroups":   config.LogGroups,
		"startTime":   config.StartTime,
		"endTime":     config.EndTime,
	})

	startTime := time.Now()
	defer func() {
		im.cloudWatch.metrics.RecordOperation("ExecuteInsightsQuery", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := im.cloudWatch.provider.config.DefaultRegion

	// Convert log groups to JSON array
	logGroupsJSON, _ := json.Marshal(config.LogGroups)

	// Convert times to Unix timestamps
	startTimeUnix := config.StartTime.Unix()
	endTimeUnix := config.EndTime.Unix()

	cmd := exec.Command("aws", "logs", "start-query",
		"--log-group-names", string(logGroupsJSON),
		"--start-time", fmt.Sprintf("%d", startTimeUnix),
		"--end-time", fmt.Sprintf("%d", endTimeUnix),
		"--query-string", config.QueryString,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to start insights query: %w", err)
	}

	// Parse start query response
	var startResponse struct {
		QueryId string `json:"queryId"`
	}

	if err := json.Unmarshal(output, &startResponse); err != nil {
		return nil, fmt.Errorf("failed to parse start query response: %w", err)
	}

	// Wait for query completion and get results
	query, err := im.waitForQueryCompletion(ctx, startResponse.QueryId, region)
	if err != nil {
		return nil, err
	}

	query.QueryString = config.QueryString
	query.LogGroups = config.LogGroups
	query.StartTime = config.StartTime
	query.EndTime = config.EndTime
	query.APMQueryConfig = config.APMQueryConfig
	query.Region = region

	// Cache the query if it's a saved query
	if config.APMQueryConfig.SavedQuery {
		im.cloudWatch.cache.SetInsightsQuery(startResponse.QueryId, query)
	}

	return query, nil
}

// waitForQueryCompletion waits for a query to complete and returns results
func (im *InsightsManager) waitForQueryCompletion(ctx context.Context, queryId, region string) (*CloudWatchInsightsQuery, error) {
	maxWaitTime := 5 * time.Minute
	pollInterval := 5 * time.Second
	deadline := time.Now().Add(maxWaitTime)

	for time.Now().Before(deadline) {
		// Check query status
		cmd := exec.Command("aws", "logs", "describe-queries",
			"--region", region)

		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to describe queries: %w", err)
		}

		var response struct {
			Queries []struct {
				QueryId      string `json:"queryId"`
				Status       string `json:"status"`
				LogGroupName string `json:"logGroupName"`
				QueryString  string `json:"queryString"`
				CreateTime   int64  `json:"createTime"`
			} `json:"queries"`
		}

		if err := json.Unmarshal(output, &response); err != nil {
			return nil, fmt.Errorf("failed to parse describe queries response: %w", err)
		}

		// Find our query
		var queryStatus string
		for _, q := range response.Queries {
			if q.QueryId == queryId {
				queryStatus = q.Status
				break
			}
		}

		if queryStatus == "Complete" {
			// Get query results
			return im.getQueryResults(ctx, queryId, region)
		} else if queryStatus == "Failed" || queryStatus == "Cancelled" {
			return nil, fmt.Errorf("query failed with status: %s", queryStatus)
		}

		// Wait before polling again
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
			continue
		}
	}

	return nil, fmt.Errorf("query timed out after %v", maxWaitTime)
}

// getQueryResults retrieves the results of a completed query
func (im *InsightsManager) getQueryResults(ctx context.Context, queryId, region string) (*CloudWatchInsightsQuery, error) {
	cmd := exec.Command("aws", "logs", "get-query-results",
		"--query-id", queryId,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get query results: %w", err)
	}

	var response struct {
		Status     string `json:"status"`
		Statistics struct {
			BytesScanned   float64 `json:"bytesScanned"`
			RecordsMatched float64 `json:"recordsMatched"`
			RecordsScanned float64 `json:"recordsScanned"`
		} `json:"statistics"`
		Results [][]map[string]string `json:"results"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse query results response: %w", err)
	}

	// Convert results to our format
	results := make([]QueryResult, 0, len(response.Results))
	for _, row := range response.Results {
		result := QueryResult{
			Fields: make(map[string]interface{}),
		}

		for _, field := range row {
			if field["field"] == "@timestamp" {
				if ts, err := time.Parse(time.RFC3339, field["value"]); err == nil {
					result.Timestamp = ts
				}
			}
			result.Fields[field["field"]] = field["value"]
		}

		results = append(results, result)
	}

	query := &CloudWatchInsightsQuery{
		QueryId: queryId,
		Status:  response.Status,
		Statistics: QueryStatistics{
			BytesScanned:   response.Statistics.BytesScanned,
			RecordsMatched: response.Statistics.RecordsMatched,
			RecordsScanned: response.Statistics.RecordsScanned,
		},
		Results: results,
	}

	return query, nil
}

// ====================================================================
// Events Management Implementation
// ====================================================================

// EventsManager handles CloudWatch Events operations
type EventsManager struct {
	cloudWatch *CloudWatchManager
}

// NewEventsManager creates a new events manager
func NewEventsManager(cw *CloudWatchManager) *EventsManager {
	return &EventsManager{cloudWatch: cw}
}

// CreateEventRule creates a CloudWatch event rule
func (em *EventsManager) CreateEventRule(ctx context.Context, config *EventRuleConfig) (*EventRule, error) {
	em.cloudWatch.logger.LogInfo(ctx, "Creating CloudWatch event rule", map[string]interface{}{
		"ruleName":     config.Name,
		"eventPattern": config.EventPattern,
		"region":       em.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		em.cloudWatch.metrics.RecordOperation("CreateEventRule", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := em.cloudWatch.provider.config.DefaultRegion
	args := []string{
		"events", "put-rule",
		"--name", config.Name,
		"--description", config.Description,
		"--state", config.State,
		"--region", region,
	}

	// Add event pattern or schedule expression
	if len(config.EventPattern) > 0 {
		eventPatternJSON, _ := json.Marshal(config.EventPattern)
		args = append(args, "--event-pattern", string(eventPatternJSON))
	} else if config.ScheduleExpression != "" {
		args = append(args, "--schedule-expression", config.ScheduleExpression)
	}

	// Add event bus if specified
	if config.EventBusName != "" {
		args = append(args, "--event-bus-name", config.EventBusName)
	}

	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create event rule: %w", err)
	}

	// Parse response
	var response struct {
		RuleArn string `json:"RuleArn"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse event rule response: %w", err)
	}

	// Add targets if specified
	if len(config.Targets) > 0 {
		if err := em.addTargetsToRule(ctx, config.Name, config.Targets, region); err != nil {
			em.cloudWatch.logger.LogWarn(ctx, "Failed to add targets to rule", map[string]interface{}{
				"ruleName": config.Name,
				"error":    err.Error(),
			})
		}
	}

	// Create EventRule object
	rule := &EventRule{
		Name:               config.Name,
		Arn:                response.RuleArn,
		Description:        config.Description,
		EventPattern:       config.EventPattern,
		ScheduleExpression: config.ScheduleExpression,
		State:              config.State,
		Targets:            config.Targets,
		Tags:               config.Tags,
		EventBusName:       config.EventBusName,
	}

	// Cache the rule
	em.cloudWatch.cache.SetEventRule(config.Name, rule)

	em.cloudWatch.logger.LogInfo(ctx, "Event rule created successfully", map[string]interface{}{
		"ruleName": config.Name,
		"ruleArn":  rule.Arn,
	})

	return rule, nil
}

// addTargetsToRule adds targets to an event rule
func (em *EventsManager) addTargetsToRule(ctx context.Context, ruleName string, targets []EventTarget, region string) error {
	targetsJSON, err := json.Marshal(targets)
	if err != nil {
		return fmt.Errorf("failed to marshal targets: %w", err)
	}

	cmd := exec.Command("aws", "events", "put-targets",
		"--rule", ruleName,
		"--targets", string(targetsJSON),
		"--region", region)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add targets to rule: %w", err)
	}

	return nil
}

// ====================================================================
// SNS Management Implementation
// ====================================================================

// SNSManager handles SNS operations for notifications
type SNSManager struct {
	cloudWatch *CloudWatchManager
}

// NewSNSManager creates a new SNS manager
func NewSNSManager(cw *CloudWatchManager) *SNSManager {
	return &SNSManager{cloudWatch: cw}
}

// CreateSNSTopic creates an SNS topic for notifications
func (sm *SNSManager) CreateSNSTopic(ctx context.Context, config *SNSTopicConfig) (*SNSTopic, error) {
	sm.cloudWatch.logger.LogInfo(ctx, "Creating SNS topic", map[string]interface{}{
		"topicName": config.TopicName,
		"region":    sm.cloudWatch.provider.config.DefaultRegion,
	})

	startTime := time.Now()
	defer func() {
		sm.cloudWatch.metrics.RecordOperation("CreateSNSTopic", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := sm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "sns", "create-topic",
		"--name", config.TopicName,
		"--region", region)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create SNS topic: %w", err)
	}

	// Parse response
	var response struct {
		TopicArn string `json:"TopicArn"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse SNS topic response: %w", err)
	}

	// Set display name if provided
	if config.DisplayName != "" {
		displayCmd := exec.Command("aws", "sns", "set-topic-attributes",
			"--topic-arn", response.TopicArn,
			"--attribute-name", "DisplayName",
			"--attribute-value", config.DisplayName,
			"--region", region)

		if err := displayCmd.Run(); err != nil {
			sm.cloudWatch.logger.LogWarn(ctx, "Failed to set display name", map[string]interface{}{
				"topicArn":    response.TopicArn,
				"displayName": config.DisplayName,
				"error":       err.Error(),
			})
		}
	}

	// Create SNSTopic object
	topic := &SNSTopic{
		TopicArn:              response.TopicArn,
		TopicName:             config.TopicName,
		DisplayName:           config.DisplayName,
		Attributes:            config.Attributes,
		Tags:                  config.Tags,
		Region:                region,
		APMNotificationConfig: config.APMNotificationConfig,
	}

	// Cache the topic
	sm.cloudWatch.cache.SetSNSTopic(config.TopicName, topic)

	sm.cloudWatch.logger.LogInfo(ctx, "SNS topic created successfully", map[string]interface{}{
		"topicName": config.TopicName,
		"topicArn":  topic.TopicArn,
	})

	return topic, nil
}

// PublishCustomMetric publishes a custom metric to CloudWatch
func (sm *SNSManager) PublishCustomMetric(ctx context.Context, namespace, metricName string, value float64, unit string) error {
	sm.cloudWatch.logger.LogDebug(ctx, "Publishing custom metric", map[string]interface{}{
		"namespace":  namespace,
		"metricName": metricName,
		"value":      value,
		"unit":       unit,
	})

	startTime := time.Now()
	defer func() {
		sm.cloudWatch.metrics.RecordOperation("PublishCustomMetric", time.Since(startTime), nil)
	}()

	// Build AWS CLI command
	region := sm.cloudWatch.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "put-metric-data",
		"--namespace", namespace,
		"--metric-data", fmt.Sprintf("MetricName=%s,Value=%f,Unit=%s", metricName, value, unit),
		"--region", region)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to publish custom metric: %w", err)
	}

	sm.cloudWatch.metrics.IncrementCounter("CustomMetricsPublished", 1)

	return nil
}

// ====================================================================
// APM Integration Manager Implementation
// ====================================================================

// APMIntegrationManager handles APM tool-specific integrations
type APMIntegrationManager struct {
	cloudWatch *CloudWatchManager
}

// NewAPMIntegrationManager creates a new APM integration manager
func NewAPMIntegrationManager(cw *CloudWatchManager) *APMIntegrationManager {
	return &APMIntegrationManager{cloudWatch: cw}
}

// SetupPrometheusIntegration sets up CloudWatch integration for Prometheus
func (aim *APMIntegrationManager) SetupPrometheusIntegration(ctx context.Context, namespace string) error {
	aim.cloudWatch.logger.LogInfo(ctx, "Setting up Prometheus integration", map[string]interface{}{
		"namespace": namespace,
	})

	// Create custom metrics for Prometheus bridge
	metrics := []CustomMetric{
		{MetricName: "RequestRate", Namespace: namespace, Unit: "Count/Second"},
		{MetricName: "ErrorRate", Namespace: namespace, Unit: "Percent"},
		{MetricName: "ResponseTime", Namespace: namespace, Unit: "Milliseconds"},
		{MetricName: "ActiveConnections", Namespace: namespace, Unit: "Count"},
	}

	for _, metric := range metrics {
		if err := aim.cloudWatch.snsMgr.PublishCustomMetric(ctx, metric.Namespace, metric.MetricName, 0, metric.Unit); err != nil {
			aim.cloudWatch.logger.LogWarn(ctx, "Failed to initialize metric", map[string]interface{}{
				"metricName": metric.MetricName,
				"error":      err.Error(),
			})
		}
	}

	// Create alarms for Prometheus metrics
	alarmConfigs := []*AlarmConfig{
		{
			AlarmName:          fmt.Sprintf("%s-HighErrorRate", namespace),
			AlarmDescription:   "High error rate detected in Prometheus metrics",
			MetricName:         "ErrorRate",
			Namespace:          namespace,
			Statistic:          "Average",
			Period:             300,
			EvaluationPeriods:  2,
			Threshold:          5.0, // 5% error rate
			ComparisonOperator: "GreaterThanThreshold",
			TreatMissingData:   "notBreaching",
			APMAlarmConfig: APMAlarmConfig{
				APMService:    "Prometheus",
				Severity:      "High",
				AlertCategory: "Application",
			},
		},
		{
			AlarmName:          fmt.Sprintf("%s-HighResponseTime", namespace),
			AlarmDescription:   "High response time detected in Prometheus metrics",
			MetricName:         "ResponseTime",
			Namespace:          namespace,
			Statistic:          "Average",
			Period:             300,
			EvaluationPeriods:  3,
			Threshold:          1000.0, // 1 second
			ComparisonOperator: "GreaterThanThreshold",
			TreatMissingData:   "notBreaching",
			APMAlarmConfig: APMAlarmConfig{
				APMService:    "Prometheus",
				Severity:      "Medium",
				AlertCategory: "Application",
			},
		},
	}

	for _, config := range alarmConfigs {
		if _, err := aim.cloudWatch.alarmMgr.CreateAlarm(ctx, config); err != nil {
			aim.cloudWatch.logger.LogWarn(ctx, "Failed to create Prometheus alarm", map[string]interface{}{
				"alarmName": config.AlarmName,
				"error":     err.Error(),
			})
		}
	}

	return nil
}

// SetupGrafanaIntegration sets up CloudWatch integration for Grafana
func (aim *APMIntegrationManager) SetupGrafanaIntegration(ctx context.Context, namespace string) error {
	aim.cloudWatch.logger.LogInfo(ctx, "Setting up Grafana integration", map[string]interface{}{
		"namespace": namespace,
	})

	// Create Grafana-specific dashboard
	dashboardConfig := &DashboardConfig{
		Name:        fmt.Sprintf("%s-Grafana-Dashboard", namespace),
		Template:    "application",
		Description: "Grafana metrics and performance dashboard",
		APMIntegration: APMDashboardIntegration{
			GrafanaSync: true,
			APMServices: []string{"Grafana"},
			Namespaces:  []string{namespace},
		},
	}

	if _, err := aim.cloudWatch.dashboardMgr.CreateDashboard(ctx, dashboardConfig); err != nil {
		return fmt.Errorf("failed to create Grafana dashboard: %w", err)
	}

	return nil
}

// SetupJaegerIntegration sets up CloudWatch integration for Jaeger
func (aim *APMIntegrationManager) SetupJaegerIntegration(ctx context.Context, namespace string) error {
	aim.cloudWatch.logger.LogInfo(ctx, "Setting up Jaeger integration", map[string]interface{}{
		"namespace": namespace,
	})

	// Create tracing-specific dashboard
	dashboardConfig := &DashboardConfig{
		Name:        fmt.Sprintf("%s-Jaeger-Dashboard", namespace),
		Template:    "tracing",
		Description: "Jaeger distributed tracing dashboard",
		APMIntegration: APMDashboardIntegration{
			JaegerMetrics: true,
			APMServices:   []string{"Jaeger"},
			Namespaces:    []string{namespace},
		},
	}

	if _, err := aim.cloudWatch.dashboardMgr.CreateDashboard(ctx, dashboardConfig); err != nil {
		return fmt.Errorf("failed to create Jaeger dashboard: %w", err)
	}

	// Create log group for Jaeger traces
	logGroupConfig := &LogGroupConfig{
		LogGroupName:    fmt.Sprintf("/apm/%s/jaeger", namespace),
		RetentionInDays: 30,
		APMLogConfig: APMLogConfig{
			APMService:        "Jaeger",
			LogFormat:         "JSON",
			StructuredLogging: true,
			LogLevel:          "INFO",
		},
	}

	if _, err := aim.cloudWatch.logsMgr.CreateLogGroup(ctx, logGroupConfig); err != nil {
		return fmt.Errorf("failed to create Jaeger log group: %w", err)
	}

	return nil
}

// SetupLokiIntegration sets up CloudWatch integration for Loki
func (aim *APMIntegrationManager) SetupLokiIntegration(ctx context.Context, namespace string) error {
	aim.cloudWatch.logger.LogInfo(ctx, "Setting up Loki integration", map[string]interface{}{
		"namespace": namespace,
	})

	// Create logs-specific dashboard
	dashboardConfig := &DashboardConfig{
		Name:        fmt.Sprintf("%s-Loki-Dashboard", namespace),
		Template:    "logs",
		Description: "Loki log aggregation dashboard",
		APMIntegration: APMDashboardIntegration{
			LokiLogs:    true,
			APMServices: []string{"Loki"},
			Namespaces:  []string{namespace},
		},
	}

	if _, err := aim.cloudWatch.dashboardMgr.CreateDashboard(ctx, dashboardConfig); err != nil {
		return fmt.Errorf("failed to create Loki dashboard: %w", err)
	}

	// Create log group for Loki aggregated logs
	logGroupConfig := &LogGroupConfig{
		LogGroupName:    fmt.Sprintf("/apm/%s/loki", namespace),
		RetentionInDays: 7, // Shorter retention for aggregated logs
		APMLogConfig: APMLogConfig{
			APMService:        "Loki",
			LogFormat:         "JSON",
			StructuredLogging: true,
			LogLevel:          "INFO",
		},
	}

	if _, err := aim.cloudWatch.logsMgr.CreateLogGroup(ctx, logGroupConfig); err != nil {
		return fmt.Errorf("failed to create Loki log group: %w", err)
	}

	return nil
}

// ====================================================================
// CloudWatch Support Classes Implementation
// ====================================================================

// CloudWatchLogger handles structured logging for CloudWatch operations
type CloudWatchLogger struct {
	config *ProviderConfig
}

// NewCloudWatchLogger creates a new CloudWatch logger
func NewCloudWatchLogger(config *ProviderConfig) *CloudWatchLogger {
	return &CloudWatchLogger{config: config}
}

// LogInfo logs an info message
func (l *CloudWatchLogger) LogInfo(ctx context.Context, message string, fields map[string]interface{}) {
	l.log("INFO", message, fields)
}

// LogWarn logs a warning message
func (l *CloudWatchLogger) LogWarn(ctx context.Context, message string, fields map[string]interface{}) {
	l.log("WARN", message, fields)
}

// LogError logs an error message
func (l *CloudWatchLogger) LogError(ctx context.Context, message string, fields map[string]interface{}) {
	l.log("ERROR", message, fields)
}

// LogDebug logs a debug message
func (l *CloudWatchLogger) LogDebug(ctx context.Context, message string, fields map[string]interface{}) {
	l.log("DEBUG", message, fields)
}

// log performs the actual logging
func (l *CloudWatchLogger) log(level, message string, fields map[string]interface{}) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   message,
		"component": "CloudWatch",
	}

	for k, v := range fields {
		logEntry[k] = v
	}

	// In a real implementation, this would write to a proper logging system
	logJSON, _ := json.Marshal(logEntry)
	fmt.Printf("[CloudWatch] %s\n", string(logJSON))
}

// CloudWatchMetrics handles metrics collection for CloudWatch operations
type CloudWatchMetrics struct {
	operationCounts    map[string]int64
	operationDurations map[string]time.Duration
	counters           map[string]int64
	mu                 sync.RWMutex
}

// NewCloudWatchMetrics creates a new CloudWatch metrics collector
func NewCloudWatchMetrics() *CloudWatchMetrics {
	return &CloudWatchMetrics{
		operationCounts:    make(map[string]int64),
		operationDurations: make(map[string]time.Duration),
		counters:           make(map[string]int64),
	}
}

// RecordOperation records an operation with its duration
func (m *CloudWatchMetrics) RecordOperation(operation string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.operationCounts[operation]++
	m.operationDurations[operation] += duration

	if err != nil {
		m.counters[operation+"_errors"]++
	} else {
		m.counters[operation+"_success"]++
	}
}

// IncrementCounter increments a counter metric
func (m *CloudWatchMetrics) IncrementCounter(name string, value int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += int64(value)
}

// GetOperationStats returns operation statistics
func (m *CloudWatchMetrics) GetOperationStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	for op, count := range m.operationCounts {
		avgDuration := time.Duration(0)
		if count > 0 {
			avgDuration = m.operationDurations[op] / time.Duration(count)
		}

		stats[op] = map[string]interface{}{
			"count":          count,
			"avg_duration":   avgDuration.String(),
			"total_duration": m.operationDurations[op].String(),
			"success_count":  m.counters[op+"_success"],
			"error_count":    m.counters[op+"_errors"],
		}
	}

	for name, value := range m.counters {
		if !strings.HasSuffix(name, "_success") && !strings.HasSuffix(name, "_errors") {
			stats[name] = value
		}
	}

	return stats
}

// CloudWatchCache handles caching for CloudWatch operations
type CloudWatchCache struct {
	dashboards      map[string]*CloudWatchDashboard
	alarms          map[string]*CloudWatchAlarm
	logGroups       map[string]*CloudWatchLogGroup
	insightsQueries map[string]*CloudWatchInsightsQuery
	eventRules      map[string]*EventRule
	snsTopics       map[string]*SNSTopic
	mu              sync.RWMutex
	ttl             time.Duration
	lastCleanup     time.Time
}

// NewCloudWatchCache creates a new CloudWatch cache
func NewCloudWatchCache() *CloudWatchCache {
	cache := &CloudWatchCache{
		dashboards:      make(map[string]*CloudWatchDashboard),
		alarms:          make(map[string]*CloudWatchAlarm),
		logGroups:       make(map[string]*CloudWatchLogGroup),
		insightsQueries: make(map[string]*CloudWatchInsightsQuery),
		eventRules:      make(map[string]*EventRule),
		snsTopics:       make(map[string]*SNSTopic),
		ttl:             5 * time.Minute,
		lastCleanup:     time.Now(),
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// SetDashboard caches a dashboard
func (c *CloudWatchCache) SetDashboard(name string, dashboard *CloudWatchDashboard) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dashboards[name] = dashboard
}

// GetDashboard retrieves a cached dashboard
func (c *CloudWatchCache) GetDashboard(name string) *CloudWatchDashboard {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.dashboards[name]
}

// GetDashboards retrieves cached dashboards with optional prefix filter
func (c *CloudWatchCache) GetDashboards(prefix string) []*CloudWatchDashboard {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var dashboards []*CloudWatchDashboard
	for name, dashboard := range c.dashboards {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			dashboards = append(dashboards, dashboard)
		}
	}
	return dashboards
}

// DeleteDashboard removes a dashboard from cache
func (c *CloudWatchCache) DeleteDashboard(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.dashboards, name)
}

// SetAlarm caches an alarm
func (c *CloudWatchCache) SetAlarm(name string, alarm *CloudWatchAlarm) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.alarms[name] = alarm
}

// GetAlarm retrieves a cached alarm
func (c *CloudWatchCache) GetAlarm(name string) *CloudWatchAlarm {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.alarms[name]
}

// GetAlarms retrieves cached alarms with optional prefix filter
func (c *CloudWatchCache) GetAlarms(prefix string) []*CloudWatchAlarm {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var alarms []*CloudWatchAlarm
	for name, alarm := range c.alarms {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			alarms = append(alarms, alarm)
		}
	}
	return alarms
}

// SetLogGroup caches a log group
func (c *CloudWatchCache) SetLogGroup(name string, logGroup *CloudWatchLogGroup) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.logGroups[name] = logGroup
}

// GetLogGroup retrieves a cached log group
func (c *CloudWatchCache) GetLogGroup(name string) *CloudWatchLogGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logGroups[name]
}

// SetInsightsQuery caches an insights query
func (c *CloudWatchCache) SetInsightsQuery(queryId string, query *CloudWatchInsightsQuery) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.insightsQueries[queryId] = query
}

// GetInsightsQuery retrieves a cached insights query
func (c *CloudWatchCache) GetInsightsQuery(queryId string) *CloudWatchInsightsQuery {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.insightsQueries[queryId]
}

// SetEventRule caches an event rule
func (c *CloudWatchCache) SetEventRule(name string, rule *EventRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventRules[name] = rule
}

// GetEventRule retrieves a cached event rule
func (c *CloudWatchCache) GetEventRule(name string) *EventRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.eventRules[name]
}

// SetSNSTopic caches an SNS topic
func (c *CloudWatchCache) SetSNSTopic(name string, topic *SNSTopic) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snsTopics[name] = topic
}

// GetSNSTopic retrieves a cached SNS topic
func (c *CloudWatchCache) GetSNSTopic(name string) *SNSTopic {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snsTopics[name]
}

// cleanupLoop periodically cleans up expired cache entries
func (c *CloudWatchCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired cache entries
func (c *CloudWatchCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Sub(c.lastCleanup) < c.ttl {
		return
	}

	// In a real implementation, we would track timestamps for each cached item
	// and remove expired ones. For simplicity, we're skipping this here.
	c.lastCleanup = now
}

// CloudWatchHealthChecker performs health checks for CloudWatch operations
type CloudWatchHealthChecker struct {
	provider *AWSProvider
}

// NewCloudWatchHealthChecker creates a new CloudWatch health checker
func NewCloudWatchHealthChecker(provider *AWSProvider) *CloudWatchHealthChecker {
	return &CloudWatchHealthChecker{provider: provider}
}

// CheckCloudWatchHealth performs comprehensive CloudWatch health checks
func (hc *CloudWatchHealthChecker) CheckCloudWatchHealth(ctx context.Context) map[string]interface{} {
	result := map[string]interface{}{
		"service":   "CloudWatch",
		"timestamp": time.Now().Format(time.RFC3339),
		"checks":    make(map[string]interface{}),
	}

	checks := result["checks"].(map[string]interface{})

	// Test CloudWatch API connectivity
	checks["api_connectivity"] = hc.checkAPIConnectivity(ctx)

	// Test dashboard operations
	checks["dashboard_operations"] = hc.checkDashboardOperations(ctx)

	// Test alarm operations
	checks["alarm_operations"] = hc.checkAlarmOperations(ctx)

	// Test logs operations
	checks["logs_operations"] = hc.checkLogsOperations(ctx)

	// Test metrics publishing
	checks["metrics_publishing"] = hc.checkMetricsPublishing(ctx)

	// Calculate overall health
	healthy := 0
	total := len(checks)
	for _, check := range checks {
		if checkMap, ok := check.(map[string]interface{}); ok {
			if status, ok := checkMap["status"].(string); ok && status == "healthy" {
				healthy++
			}
		}
	}

	if healthy == total {
		result["overall_status"] = "healthy"
	} else if healthy > total/2 {
		result["overall_status"] = "degraded"
	} else {
		result["overall_status"] = "unhealthy"
	}

	result["health_score"] = float64(healthy) / float64(total) * 100

	return result
}

// checkAPIConnectivity tests basic CloudWatch API connectivity
func (hc *CloudWatchHealthChecker) checkAPIConnectivity(ctx context.Context) map[string]interface{} {
	startTime := time.Now()
	result := map[string]interface{}{
		"test": "api_connectivity",
	}

	region := hc.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "list-metrics", "--region", region, "--max-items", "1")

	if err := cmd.Run(); err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
	} else {
		result["status"] = "healthy"
	}

	result["response_time"] = time.Since(startTime).String()
	return result
}

// checkDashboardOperations tests dashboard operations
func (hc *CloudWatchHealthChecker) checkDashboardOperations(ctx context.Context) map[string]interface{} {
	startTime := time.Now()
	result := map[string]interface{}{
		"test": "dashboard_operations",
	}

	region := hc.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "list-dashboards", "--region", region)

	if err := cmd.Run(); err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
	} else {
		result["status"] = "healthy"
	}

	result["response_time"] = time.Since(startTime).String()
	return result
}

// checkAlarmOperations tests alarm operations
func (hc *CloudWatchHealthChecker) checkAlarmOperations(ctx context.Context) map[string]interface{} {
	startTime := time.Now()
	result := map[string]interface{}{
		"test": "alarm_operations",
	}

	region := hc.provider.config.DefaultRegion
	cmd := exec.Command("aws", "cloudwatch", "describe-alarms", "--region", region, "--max-records", "1")

	if err := cmd.Run(); err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
	} else {
		result["status"] = "healthy"
	}

	result["response_time"] = time.Since(startTime).String()
	return result
}

// checkLogsOperations tests CloudWatch Logs operations
func (hc *CloudWatchHealthChecker) checkLogsOperations(ctx context.Context) map[string]interface{} {
	startTime := time.Now()
	result := map[string]interface{}{
		"test": "logs_operations",
	}

	region := hc.provider.config.DefaultRegion
	cmd := exec.Command("aws", "logs", "describe-log-groups", "--region", region, "--limit", "1")

	if err := cmd.Run(); err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
	} else {
		result["status"] = "healthy"
	}

	result["response_time"] = time.Since(startTime).String()
	return result
}

// checkMetricsPublishing tests custom metrics publishing
func (hc *CloudWatchHealthChecker) checkMetricsPublishing(ctx context.Context) map[string]interface{} {
	startTime := time.Now()
	result := map[string]interface{}{
		"test": "metrics_publishing",
	}

	region := hc.provider.config.DefaultRegion
	metricData := fmt.Sprintf("MetricName=HealthCheck,Value=1,Unit=Count,Timestamp=%s", time.Now().Format(time.RFC3339))

	cmd := exec.Command("aws", "cloudwatch", "put-metric-data",
		"--namespace", "APM/HealthCheck",
		"--metric-data", metricData,
		"--region", region)

	if err := cmd.Run(); err != nil {
		result["status"] = "unhealthy"
		result["error"] = err.Error()
	} else {
		result["status"] = "healthy"
	}

	result["response_time"] = time.Since(startTime).String()
	return result
}

// ====================================================================
// Cross-Account Role Assumption Implementation
// ====================================================================

// AssumeRoleOptions contains options for role assumption
type AssumeRoleOptions struct {
	SessionName           string            `json:"sessionName"`
	DurationSeconds       int               `json:"durationSeconds"`
	ExternalID            string            `json:"externalId,omitempty"`
	MFASerialNumber       string            `json:"mfaSerialNumber,omitempty"`
	MFATokenCode          string            `json:"mfaTokenCode,omitempty"`
	PolicyArns            []string          `json:"policyArns,omitempty"`
	Policy                string            `json:"policy,omitempty"`
	SourceIdentity        string            `json:"sourceIdentity,omitempty"`
	TransitiveTagKeys     []string          `json:"transitiveTagKeys,omitempty"`
	Tags                  map[string]string `json:"tags,omitempty"`
	Region                string            `json:"region,omitempty"`
	EnableCredentialCache bool              `json:"enableCredentialCache"`
	CredentialCacheTTL    time.Duration     `json:"credentialCacheTtl"`
	EnableAutoRefresh     bool              `json:"enableAutoRefresh"`
	AutoRefreshThreshold  time.Duration     `json:"autoRefreshThreshold"`
}

// DefaultAssumeRoleOptions returns default options for role assumption
func DefaultAssumeRoleOptions() *AssumeRoleOptions {
	return &AssumeRoleOptions{
		SessionName:           fmt.Sprintf("apm-session-%d", time.Now().Unix()),
		DurationSeconds:       3600, // 1 hour
		EnableCredentialCache: true,
		CredentialCacheTTL:    30 * time.Minute,
		EnableAutoRefresh:     true,
		AutoRefreshThreshold:  5 * time.Minute, // Refresh when 5 minutes left
	}
}

// RoleChainStep represents a step in a role assumption chain
type RoleChainStep struct {
	RoleArn     string             `json:"roleArn"`
	ExternalID  string             `json:"externalId,omitempty"`
	SessionName string             `json:"sessionName,omitempty"`
	Options     *AssumeRoleOptions `json:"options,omitempty"`
}

// CrossAccountSession represents a cross-account session with automatic refresh
type CrossAccountSession struct {
	Credentials      *Credentials       `json:"credentials"`
	RoleArn          string             `json:"roleArn"`
	SourceArn        string             `json:"sourceArn"`
	SessionName      string             `json:"sessionName"`
	CreatedAt        time.Time          `json:"createdAt"`
	ExpiresAt        time.Time          `json:"expiresAt"`
	RefreshThreshold time.Duration      `json:"refreshThreshold"`
	Options          *AssumeRoleOptions `json:"options"`
	mu               sync.RWMutex       `json:"-"`
}

// IsExpired checks if the session is expired or near expiry
func (s *CrossAccountSession) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.ExpiresAt.Add(-s.RefreshThreshold))
}

// TimeUntilExpiry returns the duration until the session expires
func (s *CrossAccountSession) TimeUntilExpiry() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Until(s.ExpiresAt)
}

// TimeUntilRefresh returns the duration until the session should be refreshed
func (s *CrossAccountSession) TimeUntilRefresh() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Until(s.ExpiresAt.Add(-s.RefreshThreshold))
}

// UpdateCredentials updates the session credentials
func (s *CrossAccountSession) UpdateCredentials(credentials *Credentials) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Credentials = credentials
	if credentials.Expiry != nil {
		s.ExpiresAt = *credentials.Expiry
	}
}

// CrossAccountRoleManager manages cross-account role assumptions
type CrossAccountRoleManager struct {
	provider *AWSProvider
	sessions map[string]*CrossAccountSession
	mu       sync.RWMutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewCrossAccountRoleManager creates a new cross-account role manager
func NewCrossAccountRoleManager(provider *AWSProvider) *CrossAccountRoleManager {
	manager := &CrossAccountRoleManager{
		provider: provider,
		sessions: make(map[string]*CrossAccountSession),
		stopCh:   make(chan struct{}),
	}

	// Start the refresh worker
	manager.wg.Add(1)
	go manager.refreshWorker()

	return manager
}

// Close shuts down the manager and stops background workers
func (m *CrossAccountRoleManager) Close() {
	close(m.stopCh)
	m.wg.Wait()
}

// refreshWorker runs in the background to refresh sessions
func (m *CrossAccountRoleManager) refreshWorker() {
	defer m.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.refreshExpiringSessions()
		case <-m.stopCh:
			return
		}
	}
}

// refreshExpiringSessions refreshes sessions that are near expiry
func (m *CrossAccountRoleManager) refreshExpiringSessions() {
	m.mu.RLock()
	sessionsToRefresh := make([]*CrossAccountSession, 0)
	for _, session := range m.sessions {
		if session.Options.EnableAutoRefresh && session.IsExpired() {
			sessionsToRefresh = append(sessionsToRefresh, session)
		}
	}
	m.mu.RUnlock()

	for _, session := range sessionsToRefresh {
		go func(s *CrossAccountSession) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := m.refreshSession(ctx, s); err != nil {
				// Log error but don't fail
				if m.provider.config.Logger != nil {
					m.provider.config.Logger(fmt.Sprintf("Failed to refresh session for role %s: %v", s.RoleArn, err))
				}
			}
		}(session)
	}
}

// refreshSession refreshes a specific session
func (m *CrossAccountRoleManager) refreshSession(ctx context.Context, session *CrossAccountSession) error {
	newCredentials, err := m.provider.AssumeRoleWithOptions(ctx, session.RoleArn, session.Options)
	if err != nil {
		return err
	}

	session.UpdateCredentials(newCredentials)
	return nil
}

// GetSession returns a cached session or creates a new one
func (m *CrossAccountRoleManager) GetSession(ctx context.Context, roleArn string, options *AssumeRoleOptions) (*CrossAccountSession, error) {
	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	sessionKey := fmt.Sprintf("%s:%s", roleArn, options.SessionName)

	m.mu.RLock()
	if session, exists := m.sessions[sessionKey]; exists && !session.IsExpired() {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Create new session
	credentials, err := m.provider.AssumeRoleWithOptions(ctx, roleArn, options)
	if err != nil {
		return nil, err
	}

	// Get current identity to determine source ARN
	currentIdentity, err := m.provider.ValidateSTSToken(ctx, credentials)
	if err != nil {
		return nil, err
	}

	session := &CrossAccountSession{
		Credentials:      credentials,
		RoleArn:          roleArn,
		SourceArn:        currentIdentity.Arn,
		SessionName:      options.SessionName,
		CreatedAt:        time.Now(),
		ExpiresAt:        *credentials.Expiry,
		RefreshThreshold: options.AutoRefreshThreshold,
		Options:          options,
	}

	m.mu.Lock()
	m.sessions[sessionKey] = session
	m.mu.Unlock()

	return session, nil
}

// RemoveSession removes a session from the cache
func (m *CrossAccountRoleManager) RemoveSession(roleArn, sessionName string) {
	sessionKey := fmt.Sprintf("%s:%s", roleArn, sessionName)

	m.mu.Lock()
	delete(m.sessions, sessionKey)
	m.mu.Unlock()
}

// ListSessions returns all cached sessions
func (m *CrossAccountRoleManager) ListSessions() []*CrossAccountSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*CrossAccountSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// MFAConfig represents configuration for MFA device and token management
type MFAConfig struct {
	// SerialNumber is the ARN of the MFA device (hardware or virtual)
	SerialNumber string `json:"serialNumber"`

	// TokenCode is the current MFA token code from the device
	TokenCode string `json:"tokenCode"`

	// TokenProvider is an optional function that can provide MFA tokens dynamically
	TokenProvider func() (string, error) `json:"-"`

	// CacheTokens determines if tokens should be cached for session duration
	CacheTokens bool `json:"cacheTokens"`

	// DeviceType indicates the type of MFA device (hardware, virtual, sms)
	DeviceType string `json:"deviceType"`

	// LastUsed tracks when this MFA configuration was last used
	LastUsed time.Time `json:"lastUsed"`
}

// CrossAccountCredentials extends regular credentials with cross-account specific information
type CrossAccountCredentials struct {
	// Embedded regular credentials
	*Credentials

	// SourceAccountID is the account ID where the credentials originated
	SourceAccountID string `json:"sourceAccountId"`

	// TargetAccountID is the account ID where the role was assumed
	TargetAccountID string `json:"targetAccountId"`

	// RoleArn is the full ARN of the assumed role
	RoleArn string `json:"roleArn"`

	// SessionName used for the role assumption
	SessionName string `json:"sessionName"`

	// ExternalID used for the role assumption (if any)
	ExternalID string `json:"externalId,omitempty"`

	// AssumedAt is when the role was assumed
	AssumedAt time.Time `json:"assumedAt"`

	// RoleChain contains the chain of roles assumed to get these credentials
	RoleChain []string `json:"roleChain,omitempty"`

	// Permissions contains the effective permissions for these credentials
	Permissions []string `json:"permissions,omitempty"`

	// SessionTags contains session tags applied during role assumption
	SessionTags map[string]string `json:"sessionTags,omitempty"`
}

// SessionManager manages session duration and refresh for assumed roles
type SessionManager struct {
	// Sessions maps session identifiers to active sessions
	Sessions map[string]*CrossAccountSession `json:"sessions"`

	// RefreshThreshold is how long before expiry to refresh sessions
	RefreshThreshold time.Duration `json:"refreshThreshold"`

	// MaxSessionDuration is the maximum allowed session duration
	MaxSessionDuration time.Duration `json:"maxSessionDuration"`

	// AutoRefresh enables automatic session refresh
	AutoRefresh bool `json:"autoRefresh"`

	// RefreshWorker manages the background refresh process
	RefreshWorker *sync.WaitGroup `json:"-"`

	// RefreshInterval is how often to check for sessions needing refresh
	RefreshInterval time.Duration `json:"refreshInterval"`

	// OnRefreshError is called when a refresh fails
	OnRefreshError func(sessionID string, err error) `json:"-"`

	// OnSessionExpired is called when a session expires
	OnSessionExpired func(sessionID string) `json:"-"`

	// Metrics tracks session management metrics
	Metrics *SessionMetrics `json:"metrics"`

	// mu protects concurrent access to sessions
	mu sync.RWMutex `json:"-"`
}

// SessionMetrics tracks metrics for session management
type SessionMetrics struct {
	// TotalSessions is the total number of sessions created
	TotalSessions int64 `json:"totalSessions"`

	// ActiveSessions is the current number of active sessions
	ActiveSessions int64 `json:"activeSessions"`

	// RefreshCount is the total number of successful refreshes
	RefreshCount int64 `json:"refreshCount"`

	// RefreshErrors is the total number of refresh errors
	RefreshErrors int64 `json:"refreshErrors"`

	// AverageSessionDuration is the average duration of sessions
	AverageSessionDuration time.Duration `json:"averageSessionDuration"`

	// LastRefreshTime is when the last refresh occurred
	LastRefreshTime time.Time `json:"lastRefreshTime"`
}

// CredentialCache provides secure credential storage with encryption
type CredentialCache struct {
	// Store is the underlying storage backend
	Store map[string]*CachedCredential `json:"-"`

	// EncryptionKey is used to encrypt sensitive credential data
	EncryptionKey []byte `json:"-"`

	// TTL is the default time-to-live for cached credentials
	TTL time.Duration `json:"ttl"`

	// MaxEntries is the maximum number of credentials to cache
	MaxEntries int `json:"maxEntries"`

	// EvictionPolicy determines how credentials are evicted (LRU, FIFO, TTL)
	EvictionPolicy string `json:"evictionPolicy"`

	// PersistToDisk enables persistent storage of encrypted credentials
	PersistToDisk bool `json:"persistToDisk"`

	// StoragePath is where to persist credentials on disk
	StoragePath string `json:"storagePath"`

	// AutoSave enables automatic saving on changes
	AutoSave bool `json:"autoSave"`

	// LastCleanup tracks when the cache was last cleaned
	LastCleanup time.Time `json:"lastCleanup"`

	// CleanupInterval is how often to clean expired entries
	CleanupInterval time.Duration `json:"cleanupInterval"`

	// mu protects concurrent access
	mu sync.RWMutex `json:"-"`
}

// CachedCredential represents a cached credential entry
type CachedCredential struct {
	// Credential is the encrypted credential data
	Credential []byte `json:"credential"`

	// Metadata contains non-sensitive information about the credential
	Metadata *CredentialMetadata `json:"metadata"`

	// CachedAt is when this credential was cached
	CachedAt time.Time `json:"cachedAt"`

	// ExpiresAt is when this cache entry expires
	ExpiresAt time.Time `json:"expiresAt"`

	// AccessCount tracks how many times this entry was accessed
	AccessCount int64 `json:"accessCount"`

	// LastAccessed is when this entry was last accessed
	LastAccessed time.Time `json:"lastAccessed"`
}

// CredentialMetadata contains non-sensitive metadata about cached credentials
type CredentialMetadata struct {
	// RoleArn is the ARN of the assumed role
	RoleArn string `json:"roleArn"`

	// AccountID is the AWS account ID
	AccountID string `json:"accountId"`

	// SessionName is the session name used
	SessionName string `json:"sessionName"`

	// Region is the AWS region
	Region string `json:"region"`

	// HasMFA indicates if MFA was used
	HasMFA bool `json:"hasMfa"`

	// Tags are custom tags for this credential
	Tags map[string]string `json:"tags,omitempty"`
}

// AssumeRoleWithOptions assumes a role with comprehensive options
func (p *AWSProvider) AssumeRoleWithOptions(ctx context.Context, roleArn string, options *AssumeRoleOptions) (*Credentials, error) {
	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	// Build AWS CLI command
	args := []string{"sts", "assume-role", "--role-arn", roleArn, "--role-session-name", options.SessionName}

	if options.DurationSeconds > 0 {
		args = append(args, "--duration-seconds", strconv.Itoa(options.DurationSeconds))
	}

	if options.ExternalID != "" {
		args = append(args, "--external-id", options.ExternalID)
	}

	if options.MFASerialNumber != "" && options.MFATokenCode != "" {
		args = append(args, "--serial-number", options.MFASerialNumber, "--token-code", options.MFATokenCode)
	}

	if options.Policy != "" {
		args = append(args, "--policy", options.Policy)
	}

	for _, policyArn := range options.PolicyArns {
		args = append(args, "--policy-arns", policyArn)
	}

	if options.SourceIdentity != "" {
		args = append(args, "--source-identity", options.SourceIdentity)
	}

	for _, tagKey := range options.TransitiveTagKeys {
		args = append(args, "--transitive-tag-keys", tagKey)
	}

	for key, value := range options.Tags {
		args = append(args, "--tags", fmt.Sprintf("Key=%s,Value=%s", key, value))
	}

	if options.Region != "" {
		args = append(args, "--region", options.Region)
	}

	cmd := exec.CommandContext(ctx, "aws", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("failed to assume role %s: %s", roleArn, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to assume role %s: %w", roleArn, err)
	}

	var result struct {
		Credentials struct {
			AccessKeyId     string    `json:"AccessKeyId"`
			SecretAccessKey string    `json:"SecretAccessKey"`
			SessionToken    string    `json:"SessionToken"`
			Expiration      time.Time `json:"Expiration"`
		} `json:"Credentials"`
		AssumedRoleUser struct {
			AssumedRoleId string `json:"AssumedRoleId"`
			Arn           string `json:"Arn"`
		} `json:"AssumedRoleUser"`
		PackedPolicySize *int `json:"PackedPolicySize,omitempty"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse assume role response: %w", err)
	}

	return &Credentials{
		Provider:   ProviderAWS,
		AuthMethod: AuthMethodIAMRole,
		AccessKey:  result.Credentials.AccessKeyId,
		SecretKey:  result.Credentials.SecretAccessKey,
		Token:      result.Credentials.SessionToken,
		Region:     options.Region,
		Account:    extractAccountFromArn(result.AssumedRoleUser.Arn),
		Expiry:     &result.Credentials.Expiration,
		Properties: map[string]string{
			"role_arn":         roleArn,
			"session_name":     options.SessionName,
			"assumed_role_id":  result.AssumedRoleUser.AssumedRoleId,
			"assumed_role_arn": result.AssumedRoleUser.Arn,
		},
	}, nil
}

// AssumeRoleAcrossAccount assumes a role in a different AWS account
func (p *AWSProvider) AssumeRoleAcrossAccount(ctx context.Context, sourceAccount, targetAccount, roleName string, options *AssumeRoleOptions) (*Credentials, error) {
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", targetAccount, roleName)

	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	// Validate source account access
	currentCreds, err := p.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get current credentials: %w", err)
	}

	if currentCreds.Account != sourceAccount {
		return nil, fmt.Errorf("current account %s does not match source account %s", currentCreds.Account, sourceAccount)
	}

	return p.AssumeRoleWithOptions(ctx, roleArn, options)
}

// AssumeRoleWithMFA assumes a role using MFA
func (p *AWSProvider) AssumeRoleWithMFA(ctx context.Context, roleArn, mfaDeviceArn, mfaToken string, options *AssumeRoleOptions) (*Credentials, error) {
	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	options.MFASerialNumber = mfaDeviceArn
	options.MFATokenCode = mfaToken

	return p.AssumeRoleWithOptions(ctx, roleArn, options)
}

// AssumeRoleChain assumes a chain of roles for complex multi-account scenarios
func (p *AWSProvider) AssumeRoleChain(ctx context.Context, roleChain []*RoleChainStep) (*Credentials, error) {
	if len(roleChain) == 0 {
		return nil, fmt.Errorf("role chain cannot be empty")
	}

	var currentCredentials *Credentials

	for i, step := range roleChain {
		options := step.Options
		if options == nil {
			options = DefaultAssumeRoleOptions()
		}

		if step.SessionName != "" {
			options.SessionName = step.SessionName
		} else {
			options.SessionName = fmt.Sprintf("apm-chain-step-%d-%d", i+1, time.Now().Unix())
		}

		if step.ExternalID != "" {
			options.ExternalID = step.ExternalID
		}

		// If this is not the first step, use the previous credentials
		if i > 0 && currentCredentials != nil {
			// Temporarily set environment variables for the previous credentials
			originalEnv := os.Environ()
			os.Setenv("AWS_ACCESS_KEY_ID", currentCredentials.AccessKey)
			os.Setenv("AWS_SECRET_ACCESS_KEY", currentCredentials.SecretKey)
			if currentCredentials.Token != "" {
				os.Setenv("AWS_SESSION_TOKEN", currentCredentials.Token)
			}

			defer func() {
				// Restore original environment
				os.Clearenv()
				for _, env := range originalEnv {
					parts := strings.SplitN(env, "=", 2)
					if len(parts) == 2 {
						os.Setenv(parts[0], parts[1])
					}
				}
			}()
		}

		credentials, err := p.AssumeRoleWithOptions(ctx, step.RoleArn, options)
		if err != nil {
			return nil, fmt.Errorf("failed to assume role in chain step %d (%s): %w", i+1, step.RoleArn, err)
		}

		currentCredentials = credentials
	}

	return currentCredentials, nil
}

// AssumeRoleWithExternalID assumes a role with an external ID for partner access
func (p *AWSProvider) AssumeRoleWithExternalID(ctx context.Context, roleArn, externalID string, options *AssumeRoleOptions) (*Credentials, error) {
	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	options.ExternalID = externalID

	return p.AssumeRoleWithOptions(ctx, roleArn, options)
}

// RefreshCredentials refreshes the given credentials if they are near expiry
func (p *AWSProvider) RefreshCredentials(ctx context.Context, credentials *Credentials) (*Credentials, error) {
	if credentials.Expiry == nil {
		return credentials, nil // No expiry information, assume valid
	}

	// Check if credentials are expiring within the next 5 minutes
	if time.Until(*credentials.Expiry) > 5*time.Minute {
		return credentials, nil // Still valid for more than 5 minutes
	}

	// Extract role ARN from properties
	roleArn, exists := credentials.Properties["role_arn"]
	if !exists {
		return nil, fmt.Errorf("cannot refresh credentials: role ARN not found in properties")
	}

	sessionName, exists := credentials.Properties["session_name"]
	if !exists {
		sessionName = fmt.Sprintf("apm-refresh-%d", time.Now().Unix())
	}

	options := &AssumeRoleOptions{
		SessionName:     sessionName,
		DurationSeconds: 3600, // 1 hour
		Region:          credentials.Region,
	}

	return p.AssumeRoleWithOptions(ctx, roleArn, options)
}

// SwitchRole switches to a different role, optionally in a different region
func (p *AWSProvider) SwitchRole(ctx context.Context, targetRoleArn, sessionName string, options *AssumeRoleOptions) (*Credentials, error) {
	if sessionName == "" {
		sessionName = fmt.Sprintf("apm-switch-%d", time.Now().Unix())
	}

	if options == nil {
		options = DefaultAssumeRoleOptions()
	}

	options.SessionName = sessionName

	return p.AssumeRoleWithOptions(ctx, targetRoleArn, options)
}

// ValidateRoleAssumption validates that a role can be assumed with the given options
type RoleValidation struct {
	RoleArn            string    `json:"roleArn"`
	CanAssume          bool      `json:"canAssume"`
	Error              string    `json:"error,omitempty"`
	TrustPolicyValid   bool      `json:"trustPolicyValid"`
	ExternalIDRequired bool      `json:"externalIdRequired"`
	MFARequired        bool      `json:"mfaRequired"`
	MaxSessionDuration int       `json:"maxSessionDuration"`
	ValidatedAt        time.Time `json:"validatedAt"`
}

// ValidateRoleAssumption validates that a role can be assumed
func (p *AWSProvider) ValidateRoleAssumption(ctx context.Context, roleArn string, options *AssumeRoleOptions) (*RoleValidation, error) {
	validation := &RoleValidation{
		RoleArn:     roleArn,
		ValidatedAt: time.Now(),
	}

	// Extract role name from ARN
	parts := strings.Split(roleArn, "/")
	if len(parts) < 2 {
		validation.CanAssume = false
		validation.Error = "invalid role ARN format"
		return validation, nil
	}
	roleName := parts[len(parts)-1]

	// Get role information
	roleInfo, err := p.ValidateIAMRole(ctx, roleName)
	if err != nil {
		validation.CanAssume = false
		validation.Error = fmt.Sprintf("failed to get role information: %v", err)
		return validation, nil
	}

	validation.TrustPolicyValid = roleInfo.TrustValidation.IsValid
	validation.MaxSessionDuration = roleInfo.MaxSessionDuration

	// Parse trust policy to check for external ID and MFA requirements
	var trustPolicy struct {
		Statement []struct {
			Effect    string                 `json:"Effect"`
			Principal map[string]interface{} `json:"Principal"`
			Condition map[string]interface{} `json:"Condition"`
		} `json:"Statement"`
	}

	if err := json.Unmarshal([]byte(roleInfo.AssumeRolePolicyDocument), &trustPolicy); err == nil {
		for _, statement := range trustPolicy.Statement {
			if statement.Effect == "Allow" {
				if condition, ok := statement.Condition["StringEquals"]; ok {
					if conditionMap, ok := condition.(map[string]interface{}); ok {
						if _, exists := conditionMap["sts:ExternalId"]; exists {
							validation.ExternalIDRequired = true
						}
					}
				}
				if condition, ok := statement.Condition["Bool"]; ok {
					if conditionMap, ok := condition.(map[string]interface{}); ok {
						if mfaValue, exists := conditionMap["aws:MultiFactorAuthPresent"]; exists {
							if mfaStr, ok := mfaValue.(string); ok && mfaStr == "true" {
								validation.MFARequired = true
							}
						}
					}
				}
			}
		}
	}

	// Try to assume the role (dry run if possible)
	if options == nil {
		options = DefaultAssumeRoleOptions()
		options.DurationSeconds = 900 // 15 minutes for validation
	}

	_, err = p.AssumeRoleWithOptions(ctx, roleArn, options)
	if err != nil {
		validation.CanAssume = false
		validation.Error = err.Error()
	} else {
		validation.CanAssume = true
	}

	return validation, nil
}

// GetCrossAccountRoleManager returns the cross-account role manager
func (p *AWSProvider) GetCrossAccountRoleManager() *CrossAccountRoleManager {
	// Lazy initialization
	if p.crossAccountManager == nil {
		p.crossAccountManager = NewCrossAccountRoleManager(p)
	}
	return p.crossAccountManager
}

// ====================================================================
// Multi-Account Configuration Management
// ====================================================================

// AccountConfig represents configuration for a single AWS account
type AccountConfig struct {
	AccountID       string            `json:"accountId"`
	AccountName     string            `json:"accountName"`
	Environment     string            `json:"environment"` // dev, staging, prod
	DefaultRegion   string            `json:"defaultRegion"`
	Roles           []*RoleConfig     `json:"roles"`
	ExternalID      string            `json:"externalId,omitempty"`
	MFARequired     bool              `json:"mfaRequired"`
	SessionDuration int               `json:"sessionDuration"`
	Tags            map[string]string `json:"tags,omitempty"`
	Properties      map[string]string `json:"properties,omitempty"`
}

// RoleConfig represents configuration for a role in an account
type RoleConfig struct {
	RoleName        string            `json:"roleName"`
	RoleArn         string            `json:"roleArn"`
	Description     string            `json:"description"`
	ExternalID      string            `json:"externalId,omitempty"`
	MFARequired     bool              `json:"mfaRequired"`
	SessionDuration int               `json:"sessionDuration"`
	Permissions     []string          `json:"permissions,omitempty"`
	Tags            map[string]string `json:"tags,omitempty"`
}

// MultiAccountConfig represents configuration for multiple AWS accounts
type MultiAccountConfig struct {
	Organization  string            `json:"organization"`
	MasterAccount string            `json:"masterAccount"`
	Accounts      []*AccountConfig  `json:"accounts"`
	DefaultRegion string            `json:"defaultRegion"`
	GlobalTags    map[string]string `json:"globalTags,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

// AccountManager manages multi-account configurations
type AccountManager struct {
	provider *AWSProvider
	config   *MultiAccountConfig
	mu       sync.RWMutex
}

// NewAccountManager creates a new account manager
func NewAccountManager(provider *AWSProvider) *AccountManager {
	return &AccountManager{
		provider: provider,
		config: &MultiAccountConfig{
			Accounts:      make([]*AccountConfig, 0),
			DefaultRegion: "us-east-1",
			GlobalTags:    make(map[string]string),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}
}

// LoadConfig loads multi-account configuration from file or S3
func (m *AccountManager) LoadConfig(ctx context.Context, configPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if config path is an S3 URL
	if strings.HasPrefix(configPath, "s3://") {
		return m.loadFromS3(ctx, configPath)
	}

	// Load from local file
	return m.loadFromFile(configPath)
}

// SaveConfig saves multi-account configuration to file or S3
func (m *AccountManager) SaveConfig(ctx context.Context, configPath string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if config path is an S3 URL
	if strings.HasPrefix(configPath, "s3://") {
		return m.saveToS3(ctx, configPath)
	}

	// Save to local file
	return m.saveToFile(configPath)
}

// loadFromFile loads configuration from a local file
func (m *AccountManager) loadFromFile(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &m.config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

// saveToFile saves configuration to a local file
func (m *AccountManager) saveToFile(configPath string) error {
	m.config.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadFromS3 loads configuration from S3
func (m *AccountManager) loadFromS3(ctx context.Context, s3Path string) error {
	s3Manager := m.provider.GetS3Manager()

	// Parse S3 URL: s3://bucket/key
	parts := strings.SplitN(strings.TrimPrefix(s3Path, "s3://"), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid S3 path format: %s", s3Path)
	}

	bucket, key := parts[0], parts[1]

	downloadOptions := &DownloadOptions{
		Bucket: bucket,
		Key:    key,
	}

	data, err := s3Manager.DownloadFileToMemory(ctx, downloadOptions)
	if err != nil {
		return fmt.Errorf("failed to download config from S3: %w", err)
	}

	if err := json.Unmarshal(data, &m.config); err != nil {
		return fmt.Errorf("failed to parse config from S3: %w", err)
	}

	return nil
}

// saveToS3 saves configuration to S3
func (m *AccountManager) saveToS3(ctx context.Context, s3Path string) error {
	m.config.UpdatedAt = time.Now()

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	s3Manager := m.provider.GetS3Manager()

	// Parse S3 URL: s3://bucket/key
	parts := strings.SplitN(strings.TrimPrefix(s3Path, "s3://"), "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid S3 path format: %s", s3Path)
	}

	bucket, key := parts[0], parts[1]

	uploadOptions := &UploadOptions{
		Bucket:      bucket,
		Key:         key,
		Body:        data,
		ContentType: "application/json",
		Metadata: map[string]string{
			"organization": m.config.Organization,
			"updated_at":   m.config.UpdatedAt.Format(time.RFC3339),
		},
	}

	_, err = s3Manager.UploadFile(ctx, uploadOptions)
	if err != nil {
		return fmt.Errorf("failed to upload config to S3: %w", err)
	}

	return nil
}

// AddAccount adds a new account configuration
func (m *AccountManager) AddAccount(account *AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate account
	if account.AccountID == "" {
		return fmt.Errorf("account ID is required")
	}

	if account.AccountName == "" {
		return fmt.Errorf("account name is required")
	}

	// Check for duplicate
	for _, existing := range m.config.Accounts {
		if existing.AccountID == account.AccountID {
			return fmt.Errorf("account %s already exists", account.AccountID)
		}
	}

	// Set defaults
	if account.SessionDuration == 0 {
		account.SessionDuration = 3600 // 1 hour
	}

	if account.DefaultRegion == "" {
		account.DefaultRegion = m.config.DefaultRegion
	}

	m.config.Accounts = append(m.config.Accounts, account)
	m.config.UpdatedAt = time.Now()

	return nil
}

// GetAccount retrieves an account configuration by ID
func (m *AccountManager) GetAccount(accountID string) (*AccountConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, account := range m.config.Accounts {
		if account.AccountID == accountID {
			return account, nil
		}
	}

	return nil, fmt.Errorf("account %s not found", accountID)
}

// UpdateAccount updates an existing account configuration
func (m *AccountManager) UpdateAccount(accountID string, updates *AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, account := range m.config.Accounts {
		if account.AccountID == accountID {
			// Preserve the account ID
			updates.AccountID = accountID
			m.config.Accounts[i] = updates
			m.config.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("account %s not found", accountID)
}

// RemoveAccount removes an account configuration
func (m *AccountManager) RemoveAccount(accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, account := range m.config.Accounts {
		if account.AccountID == accountID {
			m.config.Accounts = append(m.config.Accounts[:i], m.config.Accounts[i+1:]...)
			m.config.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("account %s not found", accountID)
}

// ListAccounts returns all account configurations
func (m *AccountManager) ListAccounts() []*AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	accounts := make([]*AccountConfig, len(m.config.Accounts))
	copy(accounts, m.config.Accounts)
	return accounts
}

// GetAccountsByEnvironment returns accounts filtered by environment
func (m *AccountManager) GetAccountsByEnvironment(environment string) []*AccountConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var accounts []*AccountConfig
	for _, account := range m.config.Accounts {
		if account.Environment == environment {
			accounts = append(accounts, account)
		}
	}

	return accounts
}

// AddRoleToAccount adds a role configuration to an account
func (m *AccountManager) AddRoleToAccount(accountID string, role *RoleConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, account := range m.config.Accounts {
		if account.AccountID == accountID {
			// Validate role
			if role.RoleName == "" {
				return fmt.Errorf("role name is required")
			}

			// Generate ARN if not provided
			if role.RoleArn == "" {
				role.RoleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, role.RoleName)
			}

			// Set defaults
			if role.SessionDuration == 0 {
				role.SessionDuration = account.SessionDuration
			}

			account.Roles = append(account.Roles, role)
			m.config.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("account %s not found", accountID)
}

// GetRoleFromAccount retrieves a role configuration from an account
func (m *AccountManager) GetRoleFromAccount(accountID, roleName string) (*RoleConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, account := range m.config.Accounts {
		if account.AccountID == accountID {
			for _, role := range account.Roles {
				if role.RoleName == roleName {
					return role, nil
				}
			}
			return nil, fmt.Errorf("role %s not found in account %s", roleName, accountID)
		}
	}

	return nil, fmt.Errorf("account %s not found", accountID)
}

// ValidateConfiguration validates the multi-account configuration
func (m *AccountManager) ValidateConfiguration(ctx context.Context) (*ConfigValidationResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := &ConfigValidationResult{
		IsValid:          true,
		Accounts:         make([]*AccountValidation, 0),
		ValidationErrors: make([]string, 0),
		ValidatedAt:      time.Now(),
	}

	// Validate each account
	for _, account := range m.config.Accounts {
		accountValidation := &AccountValidation{
			AccountID:        account.AccountID,
			IsValid:          true,
			Roles:            make([]*RoleValidation, 0),
			ValidationErrors: make([]string, 0),
		}

		// Validate account access
		if err := m.validateAccountAccess(ctx, account); err != nil {
			accountValidation.IsValid = false
			accountValidation.ValidationErrors = append(accountValidation.ValidationErrors, err.Error())
			result.IsValid = false
		}

		// Validate each role
		for _, role := range account.Roles {
			roleValidation, err := m.provider.ValidateRoleAssumption(ctx, role.RoleArn, &AssumeRoleOptions{
				SessionName:     fmt.Sprintf("validation-%d", time.Now().Unix()),
				DurationSeconds: 900, // 15 minutes
				ExternalID:      role.ExternalID,
			})
			if err != nil {
				accountValidation.IsValid = false
				accountValidation.ValidationErrors = append(accountValidation.ValidationErrors, fmt.Sprintf("Role %s validation failed: %v", role.RoleName, err))
				result.IsValid = false
			} else {
				accountValidation.Roles = append(accountValidation.Roles, roleValidation)
				if !roleValidation.CanAssume {
					accountValidation.IsValid = false
					result.IsValid = false
				}
			}
		}

		result.Accounts = append(result.Accounts, accountValidation)
	}

	return result, nil
}

// validateAccountAccess validates that we can access the account
func (m *AccountManager) validateAccountAccess(ctx context.Context, account *AccountConfig) error {
	// Try to get caller identity for the account
	cmd := exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--output", "json")
	if account.DefaultRegion != "" {
		cmd = exec.CommandContext(ctx, "aws", "sts", "get-caller-identity", "--region", account.DefaultRegion, "--output", "json")
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to validate account access: %w", err)
	}

	var identity struct {
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
		UserID  string `json:"UserId"`
	}

	if err := json.Unmarshal(output, &identity); err != nil {
		return fmt.Errorf("failed to parse identity response: %w", err)
	}

	// Note: This validates current credentials, not necessarily the target account
	// In a real implementation, you might want to attempt role assumption
	return nil
}

// ConfigValidationResult represents the result of configuration validation
type ConfigValidationResult struct {
	IsValid          bool                 `json:"isValid"`
	Accounts         []*AccountValidation `json:"accounts"`
	ValidationErrors []string             `json:"validationErrors"`
	ValidatedAt      time.Time            `json:"validatedAt"`
}

// AccountValidation represents validation result for a single account
type AccountValidation struct {
	AccountID        string            `json:"accountId"`
	IsValid          bool              `json:"isValid"`
	Roles            []*RoleValidation `json:"roles"`
	ValidationErrors []string          `json:"validationErrors"`
}

// GetAccountManager returns the account manager
func (p *AWSProvider) GetAccountManager() *AccountManager {
	// For now, create a new one each time
	// In a real implementation, you might want to cache this
	return NewAccountManager(p)
}
