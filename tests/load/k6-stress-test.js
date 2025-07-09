import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const memoryUsage = new Trend('memory_usage');
const cpuUsage = new Trend('cpu_usage');
const timeoutCounter = new Counter('timeout_errors');

// Stress test configuration - gradually increase load beyond normal capacity
export const options = {
    stages: [
        { duration: '2m', target: 50 },    // Ramp up to 50 users
        { duration: '5m', target: 50 },    // Stay at 50 users
        { duration: '2m', target: 100 },   // Ramp up to 100 users
        { duration: '5m', target: 100 },   // Stay at 100 users
        { duration: '2m', target: 200 },   // Ramp up to 200 users
        { duration: '5m', target: 200 },   // Stay at 200 users
        { duration: '2m', target: 400 },   // Ramp up to 400 users
        { duration: '5m', target: 400 },   // Stay at 400 users
        { duration: '2m', target: 800 },   // Ramp up to 800 users (stress point)
        { duration: '5m', target: 800 },   // Stay at 800 users
        { duration: '3m', target: 0 },     // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<2000'],  // 95% under 2s (relaxed for stress)
        http_req_failed: ['rate<0.5'],      // Less than 50% failure rate
        errors: ['rate<0.5'],               // Custom error rate under 50%
        timeout_errors: ['count<100'],      // Less than 100 timeout errors
    },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';

// Stress test data - larger payloads to increase memory pressure
const stressTestData = {
    largeUser: {
        name: 'A'.repeat(100),
        email: 'stress.test@example.com',
        age: 30,
        bio: 'B'.repeat(1000),
        preferences: {
            theme: 'dark',
            language: 'en',
            notifications: true,
            settings: 'S'.repeat(500),
        },
        metadata: Array.from({ length: 50 }, (_, i) => ({
            key: `key_${i}`,
            value: `value_${'V'.repeat(50)}`,
        })),
    },
    largeProduct: {
        name: 'Large Product Name ' + 'X'.repeat(200),
        description: 'D'.repeat(2000),
        price: 999.99,
        category: 'stress-test',
        features: Array.from({ length: 20 }, (_, i) => `Feature ${i}: ${'F'.repeat(100)}`),
        images: Array.from({ length: 10 }, (_, i) => `https://example.com/image${i}.jpg`),
        reviews: Array.from({ length: 100 }, (_, i) => ({
            id: i,
            rating: Math.floor(Math.random() * 5) + 1,
            comment: 'Review comment ' + 'R'.repeat(200),
            author: `Author ${i}`,
        })),
    },
};

// Headers for requests
const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
};

export default function () {
    // High-frequency health checks to stress the system
    group('Stress Health Checks', () => {
        for (let i = 0; i < 3; i++) {
            const response = http.get(`${BASE_URL}/health`, {
                timeout: '5s',
            });
            
            check(response, {
                'health check completed': (r) => r.status !== 0,
                'health check not timeout': (r) => r.status !== 0,
            });
            
            if (response.status === 0) {
                timeoutCounter.add(1);
            }
            
            errorRate.add(response.status >= 400);
            responseTime.add(response.timings.duration);
        }
    });

    sleep(0.1); // Reduced sleep for more aggressive testing

    // Memory stress test with large payloads
    group('Memory Stress Test', () => {
        const largeUserData = {
            ...stressTestData.largeUser,
            timestamp: Date.now(),
            sessionId: Math.random().toString(36).substring(7),
        };
        
        const createResponse = http.post(`${BASE_URL}/api/users`, JSON.stringify(largeUserData), {
            headers: headers,
            timeout: '10s',
        });
        
        check(createResponse, {
            'large user creation completed': (r) => r.status !== 0,
            'large user creation not server error': (r) => r.status < 500,
        });
        
        if (createResponse.status === 0) {
            timeoutCounter.add(1);
        }
        
        errorRate.add(createResponse.status >= 400);
        responseTime.add(createResponse.timings.duration);
        
        // Stress test with bulk operations
        const bulkUsers = Array.from({ length: 10 }, (_, i) => ({
            ...stressTestData.largeUser,
            email: `bulk.user.${i}@example.com`,
            id: i,
        }));
        
        const bulkResponse = http.post(`${BASE_URL}/api/users/bulk`, JSON.stringify(bulkUsers), {
            headers: headers,
            timeout: '15s',
        });
        
        check(bulkResponse, {
            'bulk operation completed': (r) => r.status !== 0,
            'bulk operation not server error': (r) => r.status < 500,
        });
        
        if (bulkResponse.status === 0) {
            timeoutCounter.add(1);
        }
        
        errorRate.add(bulkResponse.status >= 400);
        responseTime.add(bulkResponse.timings.duration);
    });

    sleep(0.1);

    // CPU stress test with complex queries
    group('CPU Stress Test', () => {
        // Complex search with multiple parameters
        const complexSearchResponse = http.get(
            `${BASE_URL}/api/products/search?q=stress&category=all&sort=price&order=desc&limit=100&offset=0&filters=color:red,size:large,rating:4+`,
            { timeout: '10s' }
        );
        
        check(complexSearchResponse, {
            'complex search completed': (r) => r.status !== 0,
            'complex search not server error': (r) => r.status < 500,
        });
        
        if (complexSearchResponse.status === 0) {
            timeoutCounter.add(1);
        }
        
        errorRate.add(complexSearchResponse.status >= 400);
        responseTime.add(complexSearchResponse.timings.duration);
        
        // Report generation endpoint (CPU intensive)
        const reportResponse = http.post(`${BASE_URL}/api/reports/generate`, JSON.stringify({
            type: 'detailed',
            dateRange: {
                start: '2024-01-01',
                end: '2024-12-31',
            },
            includeMetrics: true,
            format: 'json',
        }), {
            headers: headers,
            timeout: '20s',
        });
        
        check(reportResponse, {
            'report generation completed': (r) => r.status !== 0,
            'report generation not server error': (r) => r.status < 500,
        });
        
        if (reportResponse.status === 0) {
            timeoutCounter.add(1);
        }
        
        errorRate.add(reportResponse.status >= 400);
        responseTime.add(reportResponse.timings.duration);
    });

    sleep(0.1);

    // Database stress test
    group('Database Stress Test', () => {
        // Multiple concurrent database operations
        const operations = [
            http.get(`${BASE_URL}/api/users?limit=1000`, { timeout: '8s' }),
            http.get(`${BASE_URL}/api/products?limit=1000`, { timeout: '8s' }),
            http.get(`${BASE_URL}/api/orders?limit=1000`, { timeout: '8s' }),
        ];
        
        operations.forEach((response, index) => {
            check(response, {
                [`db operation ${index} completed`]: (r) => r.status !== 0,
                [`db operation ${index} not server error`]: (r) => r.status < 500,
            });
            
            if (response.status === 0) {
                timeoutCounter.add(1);
            }
            
            errorRate.add(response.status >= 400);
            responseTime.add(response.timings.duration);
        });
        
        // Simulate database write stress
        const writeOperations = Array.from({ length: 5 }, (_, i) => {
            return http.post(`${BASE_URL}/api/stress-test/write`, JSON.stringify({
                id: i,
                data: 'D'.repeat(500),
                timestamp: Date.now(),
            }), {
                headers: headers,
                timeout: '10s',
            });
        });
        
        writeOperations.forEach((response, index) => {
            check(response, {
                [`write operation ${index} completed`]: (r) => r.status !== 0,
                [`write operation ${index} not server error`]: (r) => r.status < 500,
            });
            
            if (response.status === 0) {
                timeoutCounter.add(1);
            }
            
            errorRate.add(response.status >= 400);
            responseTime.add(response.timings.duration);
        });
    });

    sleep(0.1);

    // Resource exhaustion test
    group('Resource Exhaustion Test', () => {
        // File upload simulation (if endpoint exists)
        const fileData = 'F'.repeat(1024 * 1024); // 1MB of data
        const uploadResponse = http.post(`${BASE_URL}/api/upload`, fileData, {
            headers: {
                'Content-Type': 'application/octet-stream',
            },
            timeout: '30s',
        });
        
        check(uploadResponse, {
            'file upload completed': (r) => r.status !== 0,
            'file upload not server error': (r) => r.status < 500,
        });
        
        if (uploadResponse.status === 0) {
            timeoutCounter.add(1);
        }
        
        errorRate.add(uploadResponse.status >= 400);
        responseTime.add(uploadResponse.timings.duration);
    });

    // Monitor system metrics if available
    const metricsResponse = http.get(`${BASE_URL}/metrics`, { timeout: '5s' });
    if (metricsResponse.status === 200) {
        const metrics = metricsResponse.json();
        if (metrics.memory) {
            memoryUsage.add(metrics.memory.used);
        }
        if (metrics.cpu) {
            cpuUsage.add(metrics.cpu.usage);
        }
    }

    sleep(0.1);
}

// Setup function
export function setup() {
    console.log('Setting up stress test environment...');
    
    // Warm up the service
    const warmupResponse = http.get(`${BASE_URL}/health`);
    if (warmupResponse.status !== 200) {
        console.error(`Service not available at ${BASE_URL} for stress testing`);
        return null;
    }
    
    console.log('Starting stress test - this will push the system beyond normal capacity');
    return { serviceAvailable: true };
}

// Teardown function
export function teardown(data) {
    console.log('Stress test completed');
    
    if (data && data.serviceAvailable) {
        console.log('Cleaning up stress test data...');
        
        // Clean up any test data created during stress test
        const cleanupResponse = http.delete(`${BASE_URL}/api/cleanup/stress-test`, {
            headers: headers,
            timeout: '30s',
        });
        
        if (cleanupResponse.status === 200) {
            console.log('Stress test data cleaned up successfully');
        } else {
            console.log('Note: Manual cleanup may be required for stress test data');
        }
    }
    
    console.log('Review the metrics to identify system breaking points and bottlenecks');
}