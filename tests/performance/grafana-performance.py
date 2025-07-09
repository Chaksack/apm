#!/usr/bin/env python3
"""
Grafana Performance Testing Suite
Tests dashboard rendering, query responsiveness, concurrent users, and data source performance
"""

import asyncio
import aiohttp
import json
import time
import statistics
import psutil
import sys
from datetime import datetime, timedelta
from concurrent.futures import ThreadPoolExecutor
import threading

class GrafanaPerformanceTest:
    def __init__(self, grafana_url='http://localhost:3000', username='admin', password='admin'):
        self.grafana_url = grafana_url
        self.username = username
        self.password = password
        self.session = None
        self.results = {
            'dashboard_rendering': [],
            'query_responsiveness': [],
            'concurrent_users': [],
            'data_source_performance': []
        }
    
    async def setup_session(self):
        """Setup authenticated session with Grafana"""
        auth = aiohttp.BasicAuth(self.username, self.password)
        self.session = aiohttp.ClientSession(auth=auth, timeout=aiohttp.ClientTimeout(total=30))
        
        # Test authentication
        try:
            async with self.session.get(f"{self.grafana_url}/api/user") as response:
                if response.status != 200:
                    raise Exception(f"Authentication failed: {response.status}")
        except Exception as e:
            print(f"Failed to authenticate with Grafana: {e}")
            raise
    
    async def test_dashboard_rendering(self):
        """Test dashboard rendering performance"""
        print("Testing dashboard rendering performance...")
        
        # Get list of dashboards
        dashboards = await self.get_dashboards()
        
        for dashboard in dashboards[:5]:  # Test first 5 dashboards
            dashboard_uid = dashboard['uid']
            dashboard_title = dashboard['title']
            
            # Test dashboard load times
            load_times = []
            for i in range(5):
                start_time = time.time()
                
                try:
                    async with self.session.get(
                        f"{self.grafana_url}/api/dashboards/uid/{dashboard_uid}"
                    ) as response:
                        if response.status == 200:
                            dashboard_data = await response.json()
                            load_time = time.time() - start_time
                            load_times.append(load_time)
                            
                            # Test panel rendering
                            panel_count = len(dashboard_data.get('dashboard', {}).get('panels', []))
                            
                except Exception as e:
                    print(f"Error loading dashboard {dashboard_title}: {e}")
                    continue
            
            if load_times:
                self.results['dashboard_rendering'].append({
                    'dashboard_uid': dashboard_uid,
                    'dashboard_title': dashboard_title,
                    'avg_load_time': statistics.mean(load_times),
                    'min_load_time': min(load_times),
                    'max_load_time': max(load_times),
                    'panel_count': panel_count
                })
    
    async def test_query_responsiveness(self):
        """Test query responsiveness across different data sources"""
        print("Testing query responsiveness...")
        
        # Get data sources
        data_sources = await self.get_data_sources()
        
        # Test queries for different data source types
        test_queries = {
            'prometheus': [
                'up',
                'rate(http_requests_total[5m])',
                'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))',
                'sum(rate(node_cpu_seconds_total[5m])) by (instance)',
                'avg_over_time(memory_usage_percent[1h])'
            ],
            'loki': [
                '{job="app"} |= "error"',
                '{job="nginx"} | json | line_format "{{.message}}"',
                'rate({job="app"}[5m])',
                'sum(rate({job="app"} |= "error" [5m])) by (level)',
                'topk(10, sum by (pod) (rate({job="kubernetes"} [5m])))'
            ]
        }
        
        for ds in data_sources:
            ds_type = ds.get('type', '').lower()
            ds_name = ds.get('name', '')
            ds_uid = ds.get('uid', '')
            
            if ds_type in test_queries:
                for query in test_queries[ds_type]:
                    response_times = []
                    
                    for i in range(3):
                        start_time = time.time()
                        
                        try:
                            query_data = {
                                'queries': [{
                                    'refId': 'A',
                                    'expr': query,
                                    'datasource': {'uid': ds_uid}
                                }],
                                'range': {
                                    'from': 'now-1h',
                                    'to': 'now'
                                }
                            }
                            
                            async with self.session.post(
                                f"{self.grafana_url}/api/ds/query",
                                json=query_data
                            ) as response:
                                if response.status == 200:
                                    result = await response.json()
                                    response_time = time.time() - start_time
                                    response_times.append(response_time)
                                    
                        except Exception as e:
                            print(f"Query error for {ds_name}: {e}")
                            continue
                    
                    if response_times:
                        self.results['query_responsiveness'].append({
                            'data_source': ds_name,
                            'data_source_type': ds_type,
                            'query': query,
                            'avg_response_time': statistics.mean(response_times),
                            'min_response_time': min(response_times),
                            'max_response_time': max(response_times)
                        })
    
    async def test_concurrent_users(self):
        """Test concurrent user load"""
        print("Testing concurrent user performance...")
        
        # Simulate concurrent users
        concurrent_users = [5, 10, 20, 50]
        
        for user_count in concurrent_users:
            print(f"Testing {user_count} concurrent users...")
            
            # Create tasks for concurrent users
            tasks = []
            for i in range(user_count):
                task = asyncio.create_task(self.simulate_user_session(i))
                tasks.append(task)
            
            start_time = time.time()
            results = await asyncio.gather(*tasks, return_exceptions=True)
            total_time = time.time() - start_time
            
            # Analyze results
            successful_requests = sum(1 for r in results if isinstance(r, dict))
            failed_requests = user_count - successful_requests
            
            if successful_requests > 0:
                avg_session_time = statistics.mean([
                    r['session_time'] for r in results 
                    if isinstance(r, dict) and 'session_time' in r
                ])
                
                self.results['concurrent_users'].append({
                    'user_count': user_count,
                    'total_time': total_time,
                    'successful_requests': successful_requests,
                    'failed_requests': failed_requests,
                    'avg_session_time': avg_session_time,
                    'requests_per_second': successful_requests / total_time
                })
    
    async def simulate_user_session(self, user_id):
        """Simulate a single user session"""
        session_start = time.time()
        
        try:
            # Simulate user actions
            auth = aiohttp.BasicAuth(self.username, self.password)
            async with aiohttp.ClientSession(auth=auth) as session:
                
                # Load dashboard list
                await session.get(f"{self.grafana_url}/api/search")
                
                # Load a dashboard
                dashboards = await self.get_dashboards()
                if dashboards:
                    dashboard_uid = dashboards[0]['uid']
                    await session.get(f"{self.grafana_url}/api/dashboards/uid/{dashboard_uid}")
                
                # Perform some queries
                await asyncio.sleep(0.1)  # Simulate user think time
                
                session_time = time.time() - session_start
                return {
                    'user_id': user_id,
                    'session_time': session_time,
                    'success': True
                }
                
        except Exception as e:
            return {
                'user_id': user_id,
                'error': str(e),
                'success': False
            }
    
    async def test_data_source_performance(self):
        """Test data source connection and query performance"""
        print("Testing data source performance...")
        
        data_sources = await self.get_data_sources()
        
        for ds in data_sources:
            ds_name = ds.get('name', '')
            ds_uid = ds.get('uid', '')
            ds_type = ds.get('type', '')
            
            # Test connection
            connection_times = []
            for i in range(3):
                start_time = time.time()
                
                try:
                    async with self.session.get(
                        f"{self.grafana_url}/api/datasources/proxy/{ds['id']}/api/v1/label/__name__/values"
                    ) as response:
                        connection_time = time.time() - start_time
                        if response.status == 200:
                            connection_times.append(connection_time)
                        
                except Exception as e:
                    print(f"Connection test failed for {ds_name}: {e}")
                    continue
            
            if connection_times:
                self.results['data_source_performance'].append({
                    'data_source': ds_name,
                    'data_source_type': ds_type,
                    'avg_connection_time': statistics.mean(connection_times),
                    'min_connection_time': min(connection_times),
                    'max_connection_time': max(connection_times)
                })
    
    async def get_dashboards(self):
        """Get list of dashboards"""
        try:
            async with self.session.get(f"{self.grafana_url}/api/search") as response:
                if response.status == 200:
                    return await response.json()
        except Exception as e:
            print(f"Error getting dashboards: {e}")
        return []
    
    async def get_data_sources(self):
        """Get list of data sources"""
        try:
            async with self.session.get(f"{self.grafana_url}/api/datasources") as response:
                if response.status == 200:
                    return await response.json()
        except Exception as e:
            print(f"Error getting data sources: {e}")
        return []
    
    async def run_all_tests(self):
        """Run all performance tests"""
        print("Starting Grafana performance tests...")
        
        try:
            await self.setup_session()
            
            await self.test_dashboard_rendering()
            await self.test_query_responsiveness()
            await self.test_concurrent_users()
            await self.test_data_source_performance()
            
            self.generate_report()
            
        except Exception as e:
            print(f"Test execution failed: {e}")
        finally:
            if self.session:
                await self.session.close()
    
    def generate_report(self):
        """Generate performance test report"""
        print("\n=== Grafana Performance Test Report ===")
        
        # Dashboard Rendering Report
        print("\n--- Dashboard Rendering Performance ---")
        for result in self.results['dashboard_rendering']:
            print(f"Dashboard: {result['dashboard_title']}")
            print(f"Average Load Time: {result['avg_load_time']:.2f}s")
            print(f"Panel Count: {result['panel_count']}")
            print("---")
        
        # Query Responsiveness Report
        print("\n--- Query Responsiveness ---")
        for result in self.results['query_responsiveness']:
            print(f"Data Source: {result['data_source']} ({result['data_source_type']})")
            print(f"Query: {result['query'][:50]}...")
            print(f"Average Response Time: {result['avg_response_time']:.2f}s")
            print("---")
        
        # Concurrent Users Report
        print("\n--- Concurrent Users Performance ---")
        for result in self.results['concurrent_users']:
            print(f"User Count: {result['user_count']}")
            print(f"Requests per Second: {result['requests_per_second']:.2f}")
            print(f"Success Rate: {result['successful_requests']}/{result['user_count']}")
            print("---")
        
        # Data Source Performance Report
        print("\n--- Data Source Performance ---")
        for result in self.results['data_source_performance']:
            print(f"Data Source: {result['data_source']} ({result['data_source_type']})")
            print(f"Average Connection Time: {result['avg_connection_time']:.2f}s")
            print("---")
        
        # Save results to file
        with open('grafana-performance-results.json', 'w') as f:
            json.dump(self.results, f, indent=2)
        
        print("\nResults saved to grafana-performance-results.json")
    
    def get_system_metrics(self):
        """Get current system metrics"""
        return {
            'cpu_percent': psutil.cpu_percent(),
            'memory_percent': psutil.virtual_memory().percent,
            'disk_usage': psutil.disk_usage('/').percent,
            'network_io': psutil.net_io_counters()._asdict()
        }

async def main():
    """Main function to run tests"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Grafana Performance Testing')
    parser.add_argument('--url', default='http://localhost:3000', help='Grafana URL')
    parser.add_argument('--username', default='admin', help='Grafana username')
    parser.add_argument('--password', default='admin', help='Grafana password')
    
    args = parser.parse_args()
    
    test = GrafanaPerformanceTest(args.url, args.username, args.password)
    await test.run_all_tests()

if __name__ == "__main__":
    asyncio.run(main())