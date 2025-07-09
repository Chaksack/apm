package validation

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// IstioMetricsValidator validates Istio service mesh metrics
type IstioMetricsValidator struct {
	prometheusURL string
	kubeClient    kubernetes.Interface
	httpClient    *http.Client
}

// PrometheusResponse represents the response from Prometheus API
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// NewIstioMetricsValidator creates a new Istio metrics validator
func NewIstioMetricsValidator(prometheusURL string) (*IstioMetricsValidator, error) {
	// Create Kubernetes client
	kubeClient, err := createKubernetesClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &IstioMetricsValidator{
		prometheusURL: prometheusURL,
		kubeClient:    kubeClient,
		httpClient:    httpClient,
	}, nil
}

// createKubernetesClient creates a Kubernetes client using the default kubeconfig
func createKubernetesClient() (kubernetes.Interface, error) {
	// Use in-cluster config first, then fallback to kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		// Try to load from default kubeconfig location
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

// queryPrometheus executes a query against Prometheus
func (v *IstioMetricsValidator) queryPrometheus(query string) (*PrometheusResponse, error) {
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", v.prometheusURL, url.QueryEscape(query))
	
	resp, err := v.httpClient.Get(queryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query Prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus query failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var promResp PrometheusResponse
	if err := json.Unmarshal(body, &promResp); err != nil {
		return nil, fmt.Errorf("failed to parse Prometheus response: %w", err)
	}

	return &promResp, nil
}

// TestIstioControlPlaneMetrics validates Istio control plane metrics
func TestIstioControlPlaneMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	controlPlaneMetrics := []string{
		"pilot_k8s_cfg_events_total",
		"pilot_xds_pushes_total",
		"pilot_xds_push_context_errors_total",
		"pilot_proxy_convergence_time",
		"citadel_secret_controller_svc_acc_created_cert_count",
		"galley_validation_passed_total",
		"galley_validation_failed_total",
	}

	for _, metric := range controlPlaneMetrics {
		t.Run(fmt.Sprintf("ControlPlane_%s", metric), func(t *testing.T) {
			resp, err := validator.queryPrometheus(metric)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
			assert.NotEmpty(t, resp.Data.Result, "Metric %s should have data", metric)
		})
	}
}

// TestIstioDataPlaneMetrics validates Istio data plane (Envoy) metrics
func TestIstioDataPlaneMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	dataPlaneMetrics := []string{
		"istio_requests_total",
		"istio_request_duration_milliseconds",
		"istio_request_bytes",
		"istio_response_bytes",
		"istio_tcp_connections_opened_total",
		"istio_tcp_connections_closed_total",
		"envoy_cluster_upstream_rq_total",
		"envoy_cluster_upstream_rq_pending_total",
		"envoy_http_downstream_rq_total",
	}

	for _, metric := range dataPlaneMetrics {
		t.Run(fmt.Sprintf("DataPlane_%s", metric), func(t *testing.T) {
			resp, err := validator.queryPrometheus(metric)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
			// Data plane metrics might not have data if no traffic
			if len(resp.Data.Result) == 0 {
				t.Logf("Warning: No data for metric %s (expected if no traffic)", metric)
			}
		})
	}
}

// TestIstioSecurityMetrics validates Istio security and mTLS metrics
func TestIstioSecurityMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	securityMetrics := []string{
		"istio_build",
		"pilot_k8s_cfg_events_total",
		"citadel_server_root_cert_expiry_timestamp",
		"citadel_server_cert_chain_expiry_timestamp",
	}

	for _, metric := range securityMetrics {
		t.Run(fmt.Sprintf("Security_%s", metric), func(t *testing.T) {
			resp, err := validator.queryPrometheus(metric)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioMTLSMetrics validates mTLS specific metrics
func TestIstioMTLSMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	// Test for mTLS connections
	mtlsQuery := `istio_requests_total{security_policy="mutual_tls"}`
	resp, err := validator.queryPrometheus(mtlsQuery)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)

	// Test for non-mTLS connections
	nonMtlsQuery := `istio_requests_total{security_policy="none"}`
	resp, err = validator.queryPrometheus(nonMtlsQuery)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
}

// TestIstioServiceMeshConnectivity validates service mesh connectivity
func TestIstioServiceMeshConnectivity(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	// Check for service-to-service communication
	serviceConnectivityQueries := []string{
		`istio_requests_total{source_app!="unknown",destination_app!="unknown"}`,
		`istio_request_duration_milliseconds{source_app!="unknown",destination_app!="unknown"}`,
		`envoy_cluster_upstream_rq_total{cluster_name!=""}`,
	}

	for i, query := range serviceConnectivityQueries {
		t.Run(fmt.Sprintf("ServiceConnectivity_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioSidecarMetrics validates sidecar proxy metrics
func TestIstioSidecarMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	sidecarMetrics := []string{
		"envoy_server_memory_allocated",
		"envoy_server_memory_heap_size",
		"envoy_server_uptime",
		"envoy_server_live",
		"envoy_cluster_membership_healthy",
		"envoy_cluster_membership_total",
		"envoy_listener_manager_listener_added",
	}

	for _, metric := range sidecarMetrics {
		t.Run(fmt.Sprintf("Sidecar_%s", metric), func(t *testing.T) {
			resp, err := validator.queryPrometheus(metric)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioTrafficMetrics validates traffic-related metrics
func TestIstioTrafficMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	// Test HTTP traffic metrics
	httpTrafficQueries := []string{
		`rate(istio_requests_total[5m])`,
		`histogram_quantile(0.99, rate(istio_request_duration_milliseconds_bucket[5m]))`,
		`histogram_quantile(0.95, rate(istio_request_duration_milliseconds_bucket[5m]))`,
		`histogram_quantile(0.50, rate(istio_request_duration_milliseconds_bucket[5m]))`,
	}

	for i, query := range httpTrafficQueries {
		t.Run(fmt.Sprintf("HTTPTraffic_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}

	// Test TCP traffic metrics
	tcpTrafficQueries := []string{
		`rate(istio_tcp_connections_opened_total[5m])`,
		`rate(istio_tcp_connections_closed_total[5m])`,
		`rate(istio_tcp_received_bytes_total[5m])`,
		`rate(istio_tcp_sent_bytes_total[5m])`,
	}

	for i, query := range tcpTrafficQueries {
		t.Run(fmt.Sprintf("TCPTraffic_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioErrorMetrics validates error and failure metrics
func TestIstioErrorMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	errorMetricQueries := []string{
		`istio_requests_total{response_code=~"4.*"}`,
		`istio_requests_total{response_code=~"5.*"}`,
		`pilot_xds_push_context_errors_total`,
		`pilot_total_xds_internal_errors`,
		`galley_validation_failed_total`,
	}

	for i, query := range errorMetricQueries {
		t.Run(fmt.Sprintf("ErrorMetrics_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioNamespaceMetrics validates namespace-specific metrics
func TestIstioNamespaceMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	// Check for metrics in istio-system namespace
	istioSystemQueries := []string{
		`istio_requests_total{destination_service_namespace="istio-system"}`,
		`istio_requests_total{source_workload_namespace="istio-system"}`,
		`up{job="istio-proxy", namespace="istio-system"}`,
	}

	for i, query := range istioSystemQueries {
		t.Run(fmt.Sprintf("IstioSystem_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioVersionMetrics validates version information metrics
func TestIstioVersionMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	versionQueries := []string{
		`istio_build{component="pilot"}`,
		`istio_build{component="citadel"}`,
		`istio_build{component="galley"}`,
		`envoy_server_version`,
	}

	for i, query := range versionQueries {
		t.Run(fmt.Sprintf("Version_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioComponentHealth validates Istio component health
func TestIstioComponentHealth(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	// Check if Istio components are up
	componentHealthQueries := []string{
		`up{job="istio-proxy"}`,
		`up{job="pilot"}`,
		`up{job="citadel"}`,
		`up{job="galley"}`,
		`up{job="mixer"}`,
	}

	for i, query := range componentHealthQueries {
		t.Run(fmt.Sprintf("ComponentHealth_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioConfigurationMetrics validates configuration-related metrics
func TestIstioConfigurationMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	configMetrics := []string{
		"pilot_k8s_cfg_events_total",
		"pilot_services",
		"pilot_virt_services",
		"pilot_dest_rules",
		"pilot_k8s_endpoints",
		"galley_validation_passed_total",
		"galley_validation_failed_total",
	}

	for _, metric := range configMetrics {
		t.Run(fmt.Sprintf("Configuration_%s", metric), func(t *testing.T) {
			resp, err := validator.queryPrometheus(metric)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// TestIstioWorkloadMetrics validates workload-specific metrics
func TestIstioWorkloadMetrics(t *testing.T) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(t, err)

	workloadQueries := []string{
		`istio_requests_total{source_workload!="unknown"}`,
		`istio_requests_total{destination_workload!="unknown"}`,
		`istio_request_duration_milliseconds{source_workload!="unknown"}`,
		`istio_request_duration_milliseconds{destination_workload!="unknown"}`,
	}

	for i, query := range workloadQueries {
		t.Run(fmt.Sprintf("Workload_%d", i), func(t *testing.T) {
			resp, err := validator.queryPrometheus(query)
			require.NoError(t, err)
			assert.Equal(t, "success", resp.Status)
		})
	}
}

// Helper function to check if Istio is installed in the cluster
func (v *IstioMetricsValidator) isIstioInstalled() bool {
	ctx := context.Background()
	
	// Check if istio-system namespace exists
	_, err := v.kubeClient.CoreV1().Namespaces().Get(ctx, "istio-system", metav1.GetOptions{})
	if err != nil {
		return false
	}
	
	// Check if Istio components are running
	pods, err := v.kubeClient.CoreV1().Pods("istio-system").List(ctx, metav1.ListOptions{})
	if err != nil {
		return false
	}
	
	istioComponents := []string{"pilot", "citadel", "galley", "mixer"}
	foundComponents := 0
	
	for _, pod := range pods.Items {
		for _, component := range istioComponents {
			if strings.Contains(pod.Name, component) {
				foundComponents++
				break
			}
		}
	}
	
	return foundComponents > 0
}

// BenchmarkIstioMetricsQuery benchmarks Prometheus query performance
func BenchmarkIstioMetricsQuery(b *testing.B) {
	validator, err := NewIstioMetricsValidator("http://localhost:9090")
	require.NoError(b, err)

	query := "istio_requests_total"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.queryPrometheus(query)
		if err != nil {
			b.Fatal(err)
		}
	}
}