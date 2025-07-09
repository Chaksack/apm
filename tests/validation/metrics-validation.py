#!/usr/bin/env python3
"""
Prometheus Metrics Validation Tests
Validates Prometheus metrics collection, presence, and correctness.
"""

import requests
import time
import json
import sys
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any


class PrometheusValidator:
    def __init__(self, prometheus_url: str = "http://localhost:9090"):
        self.prometheus_url = prometheus_url
        self.session = requests.Session()
        self.session.timeout = 30
        
    def _query_prometheus(self, query: str) -> Optional[Dict[str, Any]]:
        """Execute Prometheus query and return results"""
        try:
            response = self.session.get(
                f"{self.prometheus_url}/api/v1/query",
                params={"query": query}
            )
            response.raise_for_status()
            return response.json()
        except requests.RequestException as e:
            print(f"Error querying Prometheus: {e}")
            return None
    
    def _query_range(self, query: str, start: str, end: str, step: str = "1m") -> Optional[Dict[str, Any]]:
        """Execute Prometheus range query"""
        try:
            response = self.session.get(
                f"{self.prometheus_url}/api/v1/query_range",
                params={
                    "query": query,
                    "start": start,
                    "end": end,
                    "step": step
                }
            )
            response.raise_for_status()
            return response.json()
        except requests.RequestException as e:
            print(f"Error querying Prometheus range: {e}")
            return None
    
    def check_prometheus_connectivity(self) -> bool:
        """Check if Prometheus is accessible"""
        try:
            response = self.session.get(f"{self.prometheus_url}/api/v1/status/config")
            return response.status_code == 200
        except requests.RequestException:
            return False
    
    def validate_metric_presence(self, metric_name: str) -> bool:
        """Validate that a metric exists in Prometheus"""
        result = self._query_prometheus(f"up{{job=\"{metric_name}\"}}")
        if not result or result.get("status") != "success":
            return False
        
        data = result.get("data", {})
        return len(data.get("result", [])) > 0
    
    def validate_metric_labels(self, metric_name: str, required_labels: List[str]) -> bool:
        """Validate that metric has required labels"""
        result = self._query_prometheus(f"{metric_name}")
        if not result or result.get("status") != "success":
            return False
        
        data = result.get("data", {})
        results = data.get("result", [])
        
        if not results:
            return False
        
        # Check if all required labels are present in at least one result
        for result_item in results:
            metric_labels = result_item.get("metric", {})
            if all(label in metric_labels for label in required_labels):
                return True
        
        return False
    
    def validate_metric_values(self, metric_name: str, min_value: float = None, max_value: float = None) -> bool:
        """Validate metric values are within expected range"""
        result = self._query_prometheus(f"{metric_name}")
        if not result or result.get("status") != "success":
            return False
        
        data = result.get("data", {})
        results = data.get("result", [])
        
        if not results:
            return False
        
        for result_item in results:
            value = result_item.get("value", [])
            if len(value) < 2:
                continue
            
            try:
                metric_value = float(value[1])
                if min_value is not None and metric_value < min_value:
                    return False
                if max_value is not None and metric_value > max_value:
                    return False
            except (ValueError, TypeError):
                return False
        
        return True
    
    def validate_time_series_data(self, metric_name: str, duration_minutes: int = 5) -> bool:
        """Validate time series data availability"""
        end_time = datetime.now()
        start_time = end_time - timedelta(minutes=duration_minutes)
        
        result = self._query_range(
            metric_name,
            start_time.strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
            end_time.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
        )
        
        if not result or result.get("status") != "success":
            return False
        
        data = result.get("data", {})
        results = data.get("result", [])
        
        if not results:
            return False
        
        # Check if we have data points
        for result_item in results:
            values = result_item.get("values", [])
            if len(values) > 0:
                return True
        
        return False


class MetricsValidationSuite:
    def __init__(self, prometheus_url: str = "http://localhost:9090"):
        self.validator = PrometheusValidator(prometheus_url)
        self.test_results = []
        
    def run_test(self, test_name: str, test_func, *args, **kwargs) -> bool:
        """Run a single test and record results"""
        try:
            start_time = time.time()
            result = test_func(*args, **kwargs)
            end_time = time.time()
            
            self.test_results.append({
                "test_name": test_name,
                "result": result,
                "duration": end_time - start_time,
                "timestamp": datetime.now().isoformat()
            })
            
            status = "PASS" if result else "FAIL"
            print(f"[{status}] {test_name} ({end_time - start_time:.2f}s)")
            return result
            
        except Exception as e:
            self.test_results.append({
                "test_name": test_name,
                "result": False,
                "error": str(e),
                "duration": 0,
                "timestamp": datetime.now().isoformat()
            })
            print(f"[ERROR] {test_name}: {e}")
            return False
    
    def validate_core_metrics(self) -> bool:
        """Validate core Prometheus metrics"""
        core_metrics = [
            "up",
            "prometheus_build_info",
            "prometheus_config_last_reload_successful",
            "prometheus_notifications_total",
            "prometheus_rule_evaluation_duration_seconds"
        ]
        
        all_passed = True
        for metric in core_metrics:
            passed = self.run_test(
                f"Core metric presence: {metric}",
                self.validator.validate_metric_presence,
                metric
            )
            all_passed = all_passed and passed
        
        return all_passed
    
    def validate_application_metrics(self) -> bool:
        """Validate application-specific metrics"""
        app_metrics = [
            ("http_requests_total", ["method", "status"]),
            ("http_request_duration_seconds", ["method"]),
            ("process_cpu_seconds_total", []),
            ("process_resident_memory_bytes", []),
            ("go_memstats_alloc_bytes", [])
        ]
        
        all_passed = True
        for metric_name, required_labels in app_metrics:
            # Test metric presence
            passed = self.run_test(
                f"Application metric presence: {metric_name}",
                self.validator.validate_metric_presence,
                metric_name
            )
            all_passed = all_passed and passed
            
            # Test required labels
            if required_labels:
                passed = self.run_test(
                    f"Application metric labels: {metric_name}",
                    self.validator.validate_metric_labels,
                    metric_name,
                    required_labels
                )
                all_passed = all_passed and passed
        
        return all_passed
    
    def validate_infrastructure_metrics(self) -> bool:
        """Validate infrastructure metrics"""
        infra_metrics = [
            ("node_cpu_seconds_total", ["cpu", "mode"]),
            ("node_memory_MemAvailable_bytes", []),
            ("node_filesystem_avail_bytes", ["device", "mountpoint"]),
            ("node_load1", []),
            ("node_network_receive_bytes_total", ["device"])
        ]
        
        all_passed = True
        for metric_name, required_labels in infra_metrics:
            passed = self.run_test(
                f"Infrastructure metric presence: {metric_name}",
                self.validator.validate_metric_presence,
                metric_name
            )
            all_passed = all_passed and passed
            
            if required_labels:
                passed = self.run_test(
                    f"Infrastructure metric labels: {metric_name}",
                    self.validator.validate_metric_labels,
                    metric_name,
                    required_labels
                )
                all_passed = all_passed and passed
        
        return all_passed
    
    def validate_metric_values(self) -> bool:
        """Validate metric values are within reasonable ranges"""
        value_tests = [
            ("up", 0.0, 1.0),
            ("node_load1", 0.0, 100.0),
            ("process_resident_memory_bytes", 0.0, None),
            ("http_request_duration_seconds", 0.0, 60.0)
        ]
        
        all_passed = True
        for metric_name, min_val, max_val in value_tests:
            passed = self.run_test(
                f"Metric value validation: {metric_name}",
                self.validator.validate_metric_values,
                metric_name,
                min_val,
                max_val
            )
            all_passed = all_passed and passed
        
        return all_passed
    
    def validate_time_series(self) -> bool:
        """Validate time series data availability"""
        time_series_metrics = [
            "up",
            "http_requests_total",
            "node_cpu_seconds_total",
            "process_cpu_seconds_total"
        ]
        
        all_passed = True
        for metric in time_series_metrics:
            passed = self.run_test(
                f"Time series data: {metric}",
                self.validator.validate_time_series_data,
                metric
            )
            all_passed = all_passed and passed
        
        return all_passed
    
    def run_all_validations(self) -> bool:
        """Run all validation tests"""
        print("Starting Prometheus Metrics Validation Suite")
        print("=" * 50)
        
        # Check connectivity first
        if not self.run_test("Prometheus connectivity", self.validator.check_prometheus_connectivity):
            print("Cannot connect to Prometheus. Stopping validation.")
            return False
        
        # Run all validation categories
        validations = [
            ("Core Metrics", self.validate_core_metrics),
            ("Application Metrics", self.validate_application_metrics),
            ("Infrastructure Metrics", self.validate_infrastructure_metrics),
            ("Metric Values", self.validate_metric_values),
            ("Time Series Data", self.validate_time_series)
        ]
        
        overall_result = True
        for category, validation_func in validations:
            print(f"\n--- {category} ---")
            result = validation_func()
            overall_result = overall_result and result
        
        self.print_summary()
        return overall_result
    
    def print_summary(self):
        """Print test results summary"""
        print("\n" + "=" * 50)
        print("VALIDATION SUMMARY")
        print("=" * 50)
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results if result["result"])
        failed_tests = total_tests - passed_tests
        
        print(f"Total Tests: {total_tests}")
        print(f"Passed: {passed_tests}")
        print(f"Failed: {failed_tests}")
        print(f"Success Rate: {(passed_tests/total_tests)*100:.1f}%")
        
        if failed_tests > 0:
            print("\nFailed Tests:")
            for result in self.test_results:
                if not result["result"]:
                    error_msg = result.get("error", "Test failed")
                    print(f"  - {result['test_name']}: {error_msg}")
    
    def save_results(self, filename: str = "metrics_validation_results.json"):
        """Save test results to JSON file"""
        with open(filename, 'w') as f:
            json.dump({
                "validation_run": {
                    "timestamp": datetime.now().isoformat(),
                    "total_tests": len(self.test_results),
                    "passed": sum(1 for r in self.test_results if r["result"]),
                    "failed": sum(1 for r in self.test_results if not r["result"])
                },
                "test_results": self.test_results
            }, f, indent=2)


def main():
    """Main function to run metrics validation"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Validate Prometheus metrics")
    parser.add_argument(
        "--prometheus-url",
        default="http://localhost:9090",
        help="Prometheus server URL"
    )
    parser.add_argument(
        "--output",
        default="metrics_validation_results.json",
        help="Output file for results"
    )
    
    args = parser.parse_args()
    
    # Run validation suite
    suite = MetricsValidationSuite(args.prometheus_url)
    success = suite.run_all_validations()
    
    # Save results
    suite.save_results(args.output)
    
    # Exit with appropriate code
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()