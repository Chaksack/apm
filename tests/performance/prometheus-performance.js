const axios = require('axios');
const { performance } = require('perf_hooks');
const os = require('os');

class PrometheusPerformanceTest {
    constructor(prometheusUrl = 'http://localhost:9090') {
        this.prometheusUrl = prometheusUrl;
        this.results = {
            queryPerformance: [],
            highCardinalityMetrics: [],
            storagePerformance: [],
            memoryUsage: []
        };
    }

    async runQueryPerformanceTest() {
        console.log('Running Prometheus query performance tests...');
        
        const queries = [
            // Basic queries
            'up',
            'rate(http_requests_total[5m])',
            'histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))',
            
            // Complex aggregations
            'sum(rate(http_requests_total[5m])) by (instance)',
            'avg_over_time(cpu_usage_percent[1h])',
            'topk(10, sum(rate(http_requests_total[5m])) by (job))',
            
            // Range queries
            'increase(http_requests_total[1h])',
            'rate(node_cpu_seconds_total[5m])',
            'irate(node_network_receive_bytes_total[1m])',
            
            // Mathematical operations
            '(1 - rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100',
            'node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes',
            'rate(node_disk_read_bytes_total[5m]) + rate(node_disk_written_bytes_total[5m])'
        ];

        for (const query of queries) {
            const iterations = 10;
            const queryResults = [];

            for (let i = 0; i < iterations; i++) {
                const start = performance.now();
                
                try {
                    const response = await axios.get(`${this.prometheusUrl}/api/v1/query`, {
                        params: { query },
                        timeout: 10000
                    });
                    
                    const end = performance.now();
                    const duration = end - start;
                    
                    queryResults.push({
                        query,
                        duration,
                        dataPoints: response.data.data.result.length,
                        status: response.status
                    });
                } catch (error) {
                    queryResults.push({
                        query,
                        duration: null,
                        error: error.message,
                        status: 'error'
                    });
                }
            }

            const avgDuration = queryResults
                .filter(r => r.duration !== null)
                .reduce((sum, r) => sum + r.duration, 0) / queryResults.length;

            this.results.queryPerformance.push({
                query,
                avgDuration,
                results: queryResults
            });
        }
    }

    async runHighCardinalityTest() {
        console.log('Running high cardinality metric tests...');
        
        const highCardinalityQueries = [
            // High cardinality queries that stress the system
            'count by (instance) ({__name__=~".+"})',
            'group by (job) ({__name__=~"http_.*"})',
            'topk(100, {__name__=~"node_.*"})',
            'count by (__name__) ({__name__=~".+"})',
            'sum by (instance, job) (up)',
            'avg by (instance, mode) (rate(node_cpu_seconds_total[5m]))',
            'max by (device) (rate(node_disk_io_time_seconds_total[5m]))',
            'min by (mountpoint) (node_filesystem_free_bytes)'
        ];

        for (const query of highCardinalityQueries) {
            const start = performance.now();
            
            try {
                const response = await axios.get(`${this.prometheusUrl}/api/v1/query`, {
                    params: { query },
                    timeout: 30000
                });
                
                const end = performance.now();
                const duration = end - start;
                
                this.results.highCardinalityMetrics.push({
                    query,
                    duration,
                    seriesCount: response.data.data.result.length,
                    memoryUsage: process.memoryUsage().heapUsed
                });
            } catch (error) {
                this.results.highCardinalityMetrics.push({
                    query,
                    error: error.message,
                    duration: null
                });
            }
        }
    }

    async runStoragePerformanceTest() {
        console.log('Running storage performance tests...');
        
        // Test range queries with different time ranges
        const timeRanges = ['1h', '6h', '1d', '7d', '30d'];
        const baseQuery = 'rate(http_requests_total[5m])';
        
        for (const range of timeRanges) {
            const start = performance.now();
            const endTime = Math.floor(Date.now() / 1000);
            const startTime = endTime - this.parseTimeRange(range);
            
            try {
                const response = await axios.get(`${this.prometheusUrl}/api/v1/query_range`, {
                    params: {
                        query: baseQuery,
                        start: startTime,
                        end: endTime,
                        step: '60s'
                    },
                    timeout: 60000
                });
                
                const end = performance.now();
                const duration = end - start;
                
                this.results.storagePerformance.push({
                    range,
                    duration,
                    dataPoints: response.data.data.result.reduce((sum, series) => 
                        sum + series.values.length, 0),
                    responseSize: JSON.stringify(response.data).length
                });
            } catch (error) {
                this.results.storagePerformance.push({
                    range,
                    error: error.message,
                    duration: null
                });
            }
        }
    }

    async runMemoryUsageTest() {
        console.log('Running memory usage monitoring...');
        
        const startMemory = process.memoryUsage();
        
        // Simulate heavy query load
        const concurrentQueries = 50;
        const queries = Array(concurrentQueries).fill().map((_, i) => 
            axios.get(`${this.prometheusUrl}/api/v1/query`, {
                params: { query: `up{instance=~".*${i % 10}.*"}` },
                timeout: 10000
            }).catch(error => ({ error: error.message }))
        );
        
        await Promise.all(queries);
        
        const endMemory = process.memoryUsage();
        
        this.results.memoryUsage.push({
            startMemory,
            endMemory,
            memoryDelta: {
                heapUsed: endMemory.heapUsed - startMemory.heapUsed,
                heapTotal: endMemory.heapTotal - startMemory.heapTotal,
                external: endMemory.external - startMemory.external
            },
            systemMemory: {
                total: os.totalmem(),
                free: os.freemem(),
                used: os.totalmem() - os.freemem()
            }
        });
    }

    parseTimeRange(range) {
        const unit = range.slice(-1);
        const value = parseInt(range.slice(0, -1));
        
        switch (unit) {
            case 'h': return value * 3600;
            case 'd': return value * 86400;
            default: return value;
        }
    }

    async runAllTests() {
        console.log('Starting Prometheus performance tests...');
        
        try {
            // Verify Prometheus is accessible
            await axios.get(`${this.prometheusUrl}/api/v1/status/config`);
            
            await this.runQueryPerformanceTest();
            await this.runHighCardinalityTest();
            await this.runStoragePerformanceTest();
            await this.runMemoryUsageTest();
            
            this.generateReport();
        } catch (error) {
            console.error('Failed to connect to Prometheus:', error.message);
        }
    }

    generateReport() {
        console.log('\n=== Prometheus Performance Test Report ===');
        
        // Query Performance Report
        console.log('\n--- Query Performance ---');
        this.results.queryPerformance.forEach(result => {
            console.log(`Query: ${result.query}`);
            console.log(`Average Duration: ${result.avgDuration?.toFixed(2)}ms`);
            console.log('---');
        });
        
        // High Cardinality Report
        console.log('\n--- High Cardinality Metrics ---');
        this.results.highCardinalityMetrics.forEach(result => {
            if (result.duration) {
                console.log(`Query: ${result.query}`);
                console.log(`Duration: ${result.duration.toFixed(2)}ms`);
                console.log(`Series Count: ${result.seriesCount}`);
                console.log('---');
            }
        });
        
        // Storage Performance Report
        console.log('\n--- Storage Performance ---');
        this.results.storagePerformance.forEach(result => {
            if (result.duration) {
                console.log(`Range: ${result.range}`);
                console.log(`Duration: ${result.duration.toFixed(2)}ms`);
                console.log(`Data Points: ${result.dataPoints}`);
                console.log(`Response Size: ${(result.responseSize / 1024).toFixed(2)}KB`);
                console.log('---');
            }
        });
        
        // Memory Usage Report
        console.log('\n--- Memory Usage ---');
        this.results.memoryUsage.forEach(result => {
            console.log(`Heap Used Delta: ${(result.memoryDelta.heapUsed / 1024 / 1024).toFixed(2)}MB`);
            console.log(`System Memory Used: ${((result.systemMemory.used / 1024 / 1024 / 1024)).toFixed(2)}GB`);
            console.log(`System Memory Free: ${((result.systemMemory.free / 1024 / 1024 / 1024)).toFixed(2)}GB`);
        });
        
        // Save results to file
        const fs = require('fs');
        fs.writeFileSync(
            'prometheus-performance-results.json',
            JSON.stringify(this.results, null, 2)
        );
        
        console.log('\nResults saved to prometheus-performance-results.json');
    }
}

// Run tests if this file is executed directly
if (require.main === module) {
    const test = new PrometheusPerformanceTest();
    test.runAllTests();
}

module.exports = PrometheusPerformanceTest;