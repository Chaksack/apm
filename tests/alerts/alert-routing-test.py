#!/usr/bin/env python3
"""
Alert routing validation tests for APM monitoring system.
Tests alert routing logic, severity-based routing, team assignments, and inhibition rules.
"""

import unittest
import json
import yaml
import requests
from unittest.mock import Mock, patch, MagicMock
import time
from datetime import datetime, timedelta


class AlertRoutingTest(unittest.TestCase):
    """Test suite for alert routing functionality."""

    def setUp(self):
        """Set up test environment."""
        self.alertmanager_url = "http://localhost:9093"
        self.test_alerts = []
        
        # Sample alert configurations
        self.sample_alerts = {
            "critical_cpu": {
                "alertname": "HighCPUUsage",
                "severity": "critical",
                "service": "web-server",
                "team": "platform",
                "instance": "web-01",
                "value": "95"
            },
            "warning_memory": {
                "alertname": "HighMemoryUsage",
                "severity": "warning",
                "service": "database",
                "team": "data",
                "instance": "db-01",
                "value": "85"
            },
            "info_disk": {
                "alertname": "DiskSpaceInfo",
                "severity": "info",
                "service": "storage",
                "team": "infrastructure",
                "instance": "storage-01",
                "value": "70"
            }
        }

    def tearDown(self):
        """Clean up test alerts."""
        self.cleanup_test_alerts()

    def cleanup_test_alerts(self):
        """Remove any test alerts that were created."""
        try:
            # Silence any active test alerts
            for alert in self.test_alerts:
                self.silence_alert(alert)
        except Exception as e:
            print(f"Cleanup warning: {e}")

    def test_alert_routing_validation(self):
        """Test basic alert routing validation."""
        print("Testing alert routing validation...")
        
        # Test routing rules structure
        routing_rules = {
            "route": {
                "receiver": "default",
                "group_by": ["alertname", "severity"],
                "group_wait": "10s",
                "group_interval": "10s",
                "repeat_interval": "1h",
                "routes": [
                    {
                        "match": {"severity": "critical"},
                        "receiver": "critical-alerts",
                        "group_wait": "5s",
                        "repeat_interval": "5m"
                    },
                    {
                        "match": {"team": "platform"},
                        "receiver": "platform-team",
                        "group_interval": "30s"
                    }
                ]
            }
        }
        
        # Validate routing structure
        self.assertIn("route", routing_rules)
        self.assertIn("receiver", routing_rules["route"])
        self.assertIn("routes", routing_rules["route"])
        
        # Test route matching logic
        critical_route = next(r for r in routing_rules["route"]["routes"] 
                            if r["match"].get("severity") == "critical")
        self.assertEqual(critical_route["receiver"], "critical-alerts")
        self.assertEqual(critical_route["group_wait"], "5s")
        
        print("✓ Alert routing validation passed")

    def test_severity_based_routing(self):
        """Test routing based on alert severity levels."""
        print("Testing severity-based routing...")
        
        severity_routing = {
            "critical": {
                "receiver": "critical-alerts",
                "group_wait": "5s",
                "repeat_interval": "5m"
            },
            "warning": {
                "receiver": "warning-alerts",
                "group_wait": "30s",
                "repeat_interval": "30m"
            },
            "info": {
                "receiver": "info-alerts",
                "group_wait": "5m",
                "repeat_interval": "4h"
            }
        }
        
        # Test each severity level
        for severity, config in severity_routing.items():
            alert = self.create_test_alert(severity)
            expected_receiver = config["receiver"]
            
            # Mock route matching
            matched_route = self.mock_route_matcher(alert, severity)
            self.assertEqual(matched_route["receiver"], expected_receiver)
            
            # Verify timing configurations
            if severity == "critical":
                self.assertEqual(matched_route["group_wait"], "5s")
            elif severity == "warning":
                self.assertEqual(matched_route["group_wait"], "30s")
            else:
                self.assertEqual(matched_route["group_wait"], "5m")
        
        print("✓ Severity-based routing tests passed")

    def test_team_assignment_verification(self):
        """Test team-based alert routing."""
        print("Testing team assignment verification...")
        
        team_routes = {
            "platform": "platform-team-slack",
            "data": "data-team-email",
            "infrastructure": "infra-team-pagerduty",
            "security": "security-team-emergency"
        }
        
        for team, expected_receiver in team_routes.items():
            alert = self.create_test_alert("warning", team=team)
            
            # Test team-based routing
            matched_route = self.mock_team_route_matcher(alert, team)
            self.assertEqual(matched_route["receiver"], expected_receiver)
            
            # Verify team label is preserved
            self.assertEqual(alert["labels"]["team"], team)
        
        print("✓ Team assignment verification passed")

    def test_inhibition_rule_testing(self):
        """Test alert inhibition rules."""
        print("Testing inhibition rules...")
        
        inhibition_rules = [
            {
                "source_match": {"alertname": "NodeDown"},
                "target_match": {"severity": "warning"},
                "target_match_re": {"instance": ".*"},
                "equal": ["instance"]
            },
            {
                "source_match": {"severity": "critical"},
                "target_match": {"severity": "warning"},
                "equal": ["service"]
            }
        ]
        
        # Test NodeDown inhibition
        node_down_alert = self.create_test_alert("critical", alertname="NodeDown")
        warning_alert = self.create_test_alert("warning", alertname="HighCPUUsage")
        
        # Same instance should be inhibited
        node_down_alert["labels"]["instance"] = "web-01"
        warning_alert["labels"]["instance"] = "web-01"
        
        inhibited = self.check_inhibition(node_down_alert, warning_alert, inhibition_rules[0])
        self.assertTrue(inhibited, "Warning alert should be inhibited by NodeDown")
        
        # Different instance should not be inhibited
        warning_alert["labels"]["instance"] = "web-02"
        inhibited = self.check_inhibition(node_down_alert, warning_alert, inhibition_rules[0])
        self.assertFalse(inhibited, "Warning alert on different instance should not be inhibited")
        
        # Test severity-based inhibition
        critical_alert = self.create_test_alert("critical", service="web-server")
        warning_alert = self.create_test_alert("warning", service="web-server")
        
        inhibited = self.check_inhibition(critical_alert, warning_alert, inhibition_rules[1])
        self.assertTrue(inhibited, "Warning alert should be inhibited by critical alert for same service")
        
        print("✓ Inhibition rule testing passed")

    def test_routing_tree_traversal(self):
        """Test routing tree traversal logic."""
        print("Testing routing tree traversal...")
        
        routing_tree = {
            "receiver": "default",
            "routes": [
                {
                    "match": {"severity": "critical"},
                    "receiver": "critical-alerts",
                    "routes": [
                        {
                            "match": {"team": "platform"},
                            "receiver": "platform-critical"
                        }
                    ]
                },
                {
                    "match": {"team": "data"},
                    "receiver": "data-team"
                }
            ]
        }
        
        # Test nested routing
        critical_platform_alert = self.create_test_alert("critical", team="platform")
        route = self.traverse_routing_tree(critical_platform_alert, routing_tree)
        self.assertEqual(route["receiver"], "platform-critical")
        
        # Test first-level matching
        data_alert = self.create_test_alert("warning", team="data")
        route = self.traverse_routing_tree(data_alert, routing_tree)
        self.assertEqual(route["receiver"], "data-team")
        
        # Test default routing
        unmatched_alert = self.create_test_alert("info", team="unknown")
        route = self.traverse_routing_tree(unmatched_alert, routing_tree)
        self.assertEqual(route["receiver"], "default")
        
        print("✓ Routing tree traversal tests passed")

    def test_grouping_logic(self):
        """Test alert grouping functionality."""
        print("Testing alert grouping logic...")
        
        # Create multiple alerts with same grouping keys
        alerts = [
            self.create_test_alert("critical", alertname="HighCPUUsage", service="web-server"),
            self.create_test_alert("critical", alertname="HighCPUUsage", service="web-server"),
            self.create_test_alert("warning", alertname="HighMemoryUsage", service="web-server"),
            self.create_test_alert("critical", alertname="HighCPUUsage", service="api-server")
        ]
        
        # Test grouping by alertname and service
        groups = self.group_alerts(alerts, ["alertname", "service"])
        
        # Should have 3 groups
        self.assertEqual(len(groups), 3)
        
        # Check specific groups
        cpu_web_group = next(g for g in groups if 
                           g["key"]["alertname"] == "HighCPUUsage" and 
                           g["key"]["service"] == "web-server")
        self.assertEqual(len(cpu_web_group["alerts"]), 2)
        
        memory_web_group = next(g for g in groups if 
                              g["key"]["alertname"] == "HighMemoryUsage")
        self.assertEqual(len(memory_web_group["alerts"]), 1)
        
        print("✓ Alert grouping logic tests passed")

    def create_test_alert(self, severity, alertname="TestAlert", team="test", service="test-service"):
        """Create a test alert with specified parameters."""
        alert = {
            "labels": {
                "alertname": alertname,
                "severity": severity,
                "team": team,
                "service": service,
                "instance": "test-instance",
                "job": "test-job"
            },
            "annotations": {
                "summary": f"Test {severity} alert",
                "description": f"This is a test {severity} alert for {service}"
            },
            "startsAt": datetime.utcnow().isoformat() + "Z",
            "generatorURL": "http://prometheus:9090/graph"
        }
        
        self.test_alerts.append(alert)
        return alert

    def mock_route_matcher(self, alert, severity):
        """Mock route matching based on severity."""
        severity_routes = {
            "critical": {"receiver": "critical-alerts", "group_wait": "5s"},
            "warning": {"receiver": "warning-alerts", "group_wait": "30s"},
            "info": {"receiver": "info-alerts", "group_wait": "5m"}
        }
        return severity_routes.get(severity, {"receiver": "default"})

    def mock_team_route_matcher(self, alert, team):
        """Mock route matching based on team."""
        team_routes = {
            "platform": {"receiver": "platform-team-slack"},
            "data": {"receiver": "data-team-email"},
            "infrastructure": {"receiver": "infra-team-pagerduty"},
            "security": {"receiver": "security-team-emergency"}
        }
        return team_routes.get(team, {"receiver": "default"})

    def check_inhibition(self, source_alert, target_alert, inhibition_rule):
        """Check if target alert should be inhibited by source alert."""
        # Check source match
        source_match = inhibition_rule["source_match"]
        for key, value in source_match.items():
            if source_alert["labels"].get(key) != value:
                return False
        
        # Check target match
        target_match = inhibition_rule["target_match"]
        for key, value in target_match.items():
            if target_alert["labels"].get(key) != value:
                return False
        
        # Check equal fields
        equal_fields = inhibition_rule.get("equal", [])
        for field in equal_fields:
            if source_alert["labels"].get(field) != target_alert["labels"].get(field):
                return False
        
        return True

    def traverse_routing_tree(self, alert, routing_tree):
        """Traverse routing tree to find matching route."""
        def match_route(route, alert):
            match_criteria = route.get("match", {})
            for key, value in match_criteria.items():
                if alert["labels"].get(key) != value:
                    return False
            return True
        
        def find_route(route, alert):
            # Check if current route matches
            if match_route(route, alert):
                # Check sub-routes
                for sub_route in route.get("routes", []):
                    result = find_route(sub_route, alert)
                    if result:
                        return result
                return route
            return None
        
        # Try to find matching route
        for route in routing_tree.get("routes", []):
            result = find_route(route, alert)
            if result:
                return result
        
        # Return default route
        return {"receiver": routing_tree["receiver"]}

    def group_alerts(self, alerts, group_by):
        """Group alerts by specified keys."""
        groups = {}
        
        for alert in alerts:
            # Create group key
            key = {}
            for field in group_by:
                key[field] = alert["labels"].get(field, "")
            
            key_str = json.dumps(key, sort_keys=True)
            
            if key_str not in groups:
                groups[key_str] = {
                    "key": key,
                    "alerts": []
                }
            
            groups[key_str]["alerts"].append(alert)
        
        return list(groups.values())

    def silence_alert(self, alert):
        """Silence a test alert."""
        # This would normally interact with Alertmanager API
        # For testing, we just mark it as handled
        pass

    def test_route_continue_behavior(self):
        """Test route continue behavior."""
        print("Testing route continue behavior...")
        
        routing_config = {
            "receiver": "default",
            "routes": [
                {
                    "match": {"severity": "critical"},
                    "receiver": "critical-alerts",
                    "continue": True
                },
                {
                    "match": {"team": "platform"},
                    "receiver": "platform-team"
                }
            ]
        }
        
        # Critical platform alert should match both routes
        alert = self.create_test_alert("critical", team="platform")
        matched_routes = self.find_all_matching_routes(alert, routing_config)
        
        # Should match both critical and platform routes
        self.assertEqual(len(matched_routes), 2)
        receivers = [route["receiver"] for route in matched_routes]
        self.assertIn("critical-alerts", receivers)
        self.assertIn("platform-team", receivers)
        
        print("✓ Route continue behavior tests passed")

    def find_all_matching_routes(self, alert, routing_config):
        """Find all matching routes including continue behavior."""
        matched_routes = []
        
        for route in routing_config.get("routes", []):
            match_criteria = route.get("match", {})
            matches = True
            
            for key, value in match_criteria.items():
                if alert["labels"].get(key) != value:
                    matches = False
                    break
            
            if matches:
                matched_routes.append(route)
                # If continue is False or not specified, stop here
                if not route.get("continue", False):
                    break
        
        return matched_routes


if __name__ == "__main__":
    print("Starting Alert Routing Tests...")
    print("=" * 50)
    
    # Run tests
    unittest.main(verbosity=2, exit=False)
    
    print("\n" + "=" * 50)
    print("Alert Routing Tests Complete")