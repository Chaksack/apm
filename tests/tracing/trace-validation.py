#!/usr/bin/env python3
"""
Trace Validation Test Suite for GoFiber Application
Tests trace completeness, span relationships, timing, and annotations
"""

import json
import time
import requests
import pytest
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
from datetime import datetime, timedelta
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class TraceSpan:
    """Represents a trace span with all relevant metadata"""
    trace_id: str
    span_id: str
    parent_span_id: Optional[str]
    operation_name: str
    start_time: int
    duration: int
    tags: Dict[str, Any]
    process: Dict[str, Any]
    references: List[Dict[str, Any]]
    logs: List[Dict[str, Any]]

class TraceValidator:
    """Validates traces for completeness, relationships, and correctness"""
    
    def __init__(self, jaeger_url: str = "http://localhost:16686"):
        self.jaeger_url = jaeger_url
        self.api_url = f"{jaeger_url}/api"
        
    def get_traces(self, service_name: str, lookback: str = "1h") -> List[Dict]:
        """Retrieve traces from Jaeger API"""
        try:
            url = f"{self.api_url}/traces"
            params = {
                "service": service_name,
                "lookback": lookback,
                "limit": 100
            }
            
            response = requests.get(url, params=params, timeout=30)
            response.raise_for_status()
            
            data = response.json()
            return data.get("data", [])
        except Exception as e:
            logger.error(f"Failed to retrieve traces: {e}")
            return []
    
    def parse_span(self, span_data: Dict) -> TraceSpan:
        """Parse span data into TraceSpan object"""
        return TraceSpan(
            trace_id=span_data.get("traceID", ""),
            span_id=span_data.get("spanID", ""),
            parent_span_id=span_data.get("parentSpanID"),
            operation_name=span_data.get("operationName", ""),
            start_time=span_data.get("startTime", 0),
            duration=span_data.get("duration", 0),
            tags=self._parse_tags(span_data.get("tags", [])),
            process=span_data.get("process", {}),
            references=span_data.get("references", []),
            logs=span_data.get("logs", [])
        )
    
    def _parse_tags(self, tags: List[Dict]) -> Dict[str, Any]:
        """Parse tag list into dictionary"""
        tag_dict = {}
        for tag in tags:
            key = tag.get("key", "")
            value = tag.get("value", "")
            tag_dict[key] = value
        return tag_dict
    
    def validate_trace_completeness(self, trace: Dict) -> Dict[str, Any]:
        """Validate that a trace has all required components"""
        result = {
            "valid": True,
            "errors": [],
            "warnings": []
        }
        
        # Check if trace has spans
        spans = trace.get("spans", [])
        if not spans:
            result["valid"] = False
            result["errors"].append("Trace has no spans")
            return result
        
        # Check for root span
        root_spans = [s for s in spans if not s.get("parentSpanID")]
        if not root_spans:
            result["valid"] = False
            result["errors"].append("No root span found")
        elif len(root_spans) > 1:
            result["warnings"].append(f"Multiple root spans found: {len(root_spans)}")
        
        # Check span completeness
        for span in spans:
            span_obj = self.parse_span(span)
            if not span_obj.operation_name:
                result["errors"].append(f"Span {span_obj.span_id} missing operation name")
            
            if span_obj.duration <= 0:
                result["warnings"].append(f"Span {span_obj.span_id} has zero or negative duration")
            
            # Check for required GoFiber tags
            required_tags = ["http.method", "http.url", "component"]
            for tag in required_tags:
                if tag not in span_obj.tags:
                    result["warnings"].append(f"Span {span_obj.span_id} missing tag: {tag}")
        
        if result["errors"]:
            result["valid"] = False
        
        return result
    
    def validate_span_relationships(self, trace: Dict) -> Dict[str, Any]:
        """Validate parent-child relationships between spans"""
        result = {
            "valid": True,
            "errors": [],
            "warnings": []
        }
        
        spans = trace.get("spans", [])
        span_map = {span["spanID"]: span for span in spans}
        
        for span in spans:
            span_id = span["spanID"]
            parent_id = span.get("parentSpanID")
            
            if parent_id:
                # Check if parent exists
                if parent_id not in span_map:
                    result["valid"] = False
                    result["errors"].append(f"Span {span_id} references non-existent parent {parent_id}")
                else:
                    parent_span = span_map[parent_id]
                    
                    # Check timing relationship
                    if span["startTime"] < parent_span["startTime"]:
                        result["errors"].append(f"Span {span_id} starts before parent {parent_id}")
                    
                    parent_end = parent_span["startTime"] + parent_span["duration"]
                    span_end = span["startTime"] + span["duration"]
                    
                    if span_end > parent_end:
                        result["warnings"].append(f"Span {span_id} ends after parent {parent_id}")
        
        if result["errors"]:
            result["valid"] = False
        
        return result
    
    def validate_timing_and_duration(self, trace: Dict) -> Dict[str, Any]:
        """Validate timing consistency and duration reasonableness"""
        result = {
            "valid": True,
            "errors": [],
            "warnings": [],
            "metrics": {}
        }
        
        spans = trace.get("spans", [])
        if not spans:
            return result
        
        # Calculate trace duration
        start_times = [span["startTime"] for span in spans]
        end_times = [span["startTime"] + span["duration"] for span in spans]
        
        trace_start = min(start_times)
        trace_end = max(end_times)
        trace_duration = trace_end - trace_start
        
        result["metrics"] = {
            "trace_duration_us": trace_duration,
            "trace_duration_ms": trace_duration / 1000,
            "span_count": len(spans),
            "root_span_duration": max([s["duration"] for s in spans if not s.get("parentSpanID")], default=0)
        }
        
        # Check for unreasonable durations
        for span in spans:
            duration_ms = span["duration"] / 1000
            
            # Check for very long durations (>30 seconds)
            if duration_ms > 30000:
                result["warnings"].append(f"Span {span['spanID']} has very long duration: {duration_ms}ms")
            
            # Check for very short durations (<0.1ms)
            if duration_ms < 0.1:
                result["warnings"].append(f"Span {span['spanID']} has very short duration: {duration_ms}ms")
        
        # Check for time ordering
        for span in spans:
            parent_id = span.get("parentSpanID")
            if parent_id:
                parent_spans = [s for s in spans if s["spanID"] == parent_id]
                if parent_spans:
                    parent = parent_spans[0]
                    if span["startTime"] < parent["startTime"]:
                        result["valid"] = False
                        result["errors"].append(f"Span {span['spanID']} starts before parent")
        
        return result
    
    def validate_tags_and_annotations(self, trace: Dict) -> Dict[str, Any]:
        """Validate tags and annotations for GoFiber specific requirements"""
        result = {
            "valid": True,
            "errors": [],
            "warnings": [],
            "tag_coverage": {}
        }
        
        spans = trace.get("spans", [])
        
        # Expected tags for GoFiber HTTP spans
        expected_http_tags = {
            "http.method": "HTTP method",
            "http.url": "Request URL",
            "http.status_code": "HTTP status code",
            "component": "Component name",
            "span.kind": "Span kind"
        }
        
        tag_coverage = {tag: 0 for tag in expected_http_tags}
        
        for span in spans:
            span_obj = self.parse_span(span)
            
            # Check HTTP spans
            if span_obj.tags.get("component") == "fiber" or "http" in span_obj.operation_name.lower():
                for tag, description in expected_http_tags.items():
                    if tag in span_obj.tags:
                        tag_coverage[tag] += 1
                    else:
                        result["warnings"].append(f"HTTP span {span_obj.span_id} missing {description}")
                
                # Validate HTTP status code
                status_code = span_obj.tags.get("http.status_code")
                if status_code:
                    try:
                        status_int = int(status_code)
                        if status_int >= 400:
                            # Check for error tag
                            if not span_obj.tags.get("error"):
                                result["warnings"].append(f"HTTP error span {span_obj.span_id} missing error tag")
                    except ValueError:
                        result["errors"].append(f"Invalid HTTP status code: {status_code}")
            
            # Check for error annotations
            if span_obj.tags.get("error") == "true":
                error_logs = [log for log in span_obj.logs if any(field.get("key") == "event" and "error" in field.get("value", "") for field in log.get("fields", []))]
                if not error_logs:
                    result["warnings"].append(f"Error span {span_obj.span_id} has no error logs")
        
        result["tag_coverage"] = tag_coverage
        
        return result

class TraceTestRunner:
    """Runs comprehensive trace validation tests"""
    
    def __init__(self, jaeger_url: str = "http://localhost:16686"):
        self.validator = TraceValidator(jaeger_url)
    
    def run_validation_suite(self, service_name: str) -> Dict[str, Any]:
        """Run complete validation suite for a service"""
        logger.info(f"Starting trace validation for service: {service_name}")
        
        # Get traces
        traces = self.validator.get_traces(service_name)
        if not traces:
            return {
                "success": False,
                "error": "No traces found for service",
                "service": service_name
            }
        
        results = {
            "service": service_name,
            "trace_count": len(traces),
            "validation_results": [],
            "summary": {
                "total_traces": len(traces),
                "valid_traces": 0,
                "invalid_traces": 0,
                "warnings": 0
            }
        }
        
        for i, trace in enumerate(traces):
            logger.info(f"Validating trace {i+1}/{len(traces)}")
            
            trace_result = {
                "trace_id": trace.get("traceID", "unknown"),
                "completeness": self.validator.validate_trace_completeness(trace),
                "relationships": self.validator.validate_span_relationships(trace),
                "timing": self.validator.validate_timing_and_duration(trace),
                "tags": self.validator.validate_tags_and_annotations(trace)
            }
            
            # Calculate overall validity
            all_valid = all([
                trace_result["completeness"]["valid"],
                trace_result["relationships"]["valid"],
                trace_result["timing"]["valid"],
                trace_result["tags"]["valid"]
            ])
            
            trace_result["overall_valid"] = all_valid
            results["validation_results"].append(trace_result)
            
            if all_valid:
                results["summary"]["valid_traces"] += 1
            else:
                results["summary"]["invalid_traces"] += 1
            
            # Count warnings
            warning_count = sum([
                len(trace_result["completeness"]["warnings"]),
                len(trace_result["relationships"]["warnings"]),
                len(trace_result["timing"]["warnings"]),
                len(trace_result["tags"]["warnings"])
            ])
            results["summary"]["warnings"] += warning_count
        
        logger.info(f"Validation complete. Valid: {results['summary']['valid_traces']}, Invalid: {results['summary']['invalid_traces']}")
        return results

# Test functions for pytest
def test_trace_completeness():
    """Test trace completeness validation"""
    runner = TraceTestRunner()
    results = runner.run_validation_suite("apm-service")
    
    assert results["summary"]["total_traces"] > 0, "No traces found"
    assert results["summary"]["valid_traces"] > 0, "No valid traces found"
    
    # Check that at least 80% of traces are valid
    validity_ratio = results["summary"]["valid_traces"] / results["summary"]["total_traces"]
    assert validity_ratio >= 0.8, f"Only {validity_ratio:.2%} of traces are valid"

def test_span_relationships():
    """Test span relationship validation"""
    runner = TraceTestRunner()
    results = runner.run_validation_suite("apm-service")
    
    for trace_result in results["validation_results"]:
        assert trace_result["relationships"]["valid"], f"Trace {trace_result['trace_id']} has invalid relationships"

def test_timing_consistency():
    """Test timing and duration validation"""
    runner = TraceTestRunner()
    results = runner.run_validation_suite("apm-service")
    
    for trace_result in results["validation_results"]:
        timing = trace_result["timing"]
        assert timing["valid"], f"Trace {trace_result['trace_id']} has timing issues"
        assert timing["metrics"]["trace_duration_us"] > 0, "Trace duration should be positive"

def test_gofiber_tags():
    """Test GoFiber specific tags and annotations"""
    runner = TraceTestRunner()
    results = runner.run_validation_suite("apm-service")
    
    for trace_result in results["validation_results"]:
        tags = trace_result["tags"]
        coverage = tags["tag_coverage"]
        
        # Check that HTTP spans have required tags
        if coverage["http.method"] > 0:
            assert coverage["http.status_code"] > 0, "HTTP spans should have status codes"
            assert coverage["component"] > 0, "HTTP spans should have component tags"

if __name__ == "__main__":
    # Run validation when script is executed directly
    runner = TraceTestRunner()
    results = runner.run_validation_suite("apm-service")
    
    print(json.dumps(results, indent=2))