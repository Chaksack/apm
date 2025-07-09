import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const spikeRecoveryTime = new Trend('spike_recovery_time');
const circuitBreakerTriggers = new Counter('circuit_breaker_triggers');
const autoScalingEvents = new Counter('auto_scaling_events');

// Spike test configuration - sudden traffic spikes
export const options = {
    stages: [
        { duration: '30s', target: 5 },    // Baseline
        { duration: '10s', target: 100 },  // Sudden spike
        { duration: '1m', target: 100 },   // Sustain spike
        { duration: '10s', target: 5 },    // Drop back to baseline
        { duration: '30s', target: 5 },    // Maintain baseline
        { duration: '5s', target: 200 },   // Bigger spike
        { duration: '2m', target: 200 },   // Sustain bigger spike
        { duration: '10s', target: 5 },    // Drop back
        { duration: '30s', target: 5 },    // Baseline
        { duration: '5s', target: 500 },   // Extreme spike
        { duration: '1m', target: 500 },   // Sustain extreme spike
        { duration: '30s', target: 0 },    // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<3000'],  // 95% under 3s (allowing for spike impact)
        http_req_failed: ['rate<0.3'],      // Less than 30% failure rate
        errors: ['rate<0.3'],               // Custom error rate under 30%
        spike_recovery_time: ['p(95)<5000'], // Recovery time under 5s
    },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';

// Spike test scenarios
const spikeScenarios = [
    'user_registration_spike',
    'product_search_spike',
    'order_processing_spike',
    'authentication_spike',
    'api_gateway_spike',
];

// Headers for requests
const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
};

// Track spike start time for recovery measurement
let spikeStartTime = null;

export default function () {
    const currentVUs = __ENV.K6_VU || 1;
    const scenario = spikeScenarios[Math.floor(Math.random() * spikeScenarios.length)];
    
    // Detect spike conditions (simplified logic)
    const isSpikeCondition = currentVUs > 50;
    
    if (isSpikeCondition && !spikeStartTime) {
        spikeStartTime = Date.now();
    }
    
    // Circuit breaker testing
    group('Circuit Breaker Testing', () => {
        // Rapid-fire requests to potentially trigger circuit breaker
        for (let i = 0; i < 3; i++) {
            const response = http.get(`${BASE_URL}/api/circuit-breaker-test`, {
                timeout: '2s',
            });
            
            check(response, {
                'circuit breaker request completed': (r) => r.status !== 0,
                'circuit breaker active': (r) => r.status === 503,
            });
            
            if (response.status === 503) {
                circuitBreakerTriggers.add(1);
            }
            
            errorRate.add(response.status >= 400 && response.status !== 503);
            responseTime.add(response.timings.duration);
        }
    });

    sleep(0.1);

    // Auto-scaling validation
    group('Auto-scaling Validation', () => {
        // Check if auto-scaling metrics endpoint exists
        const metricsResponse = http.get(`${BASE_URL}/metrics/scaling`, {
            timeout: '3s',
        });
        
        if (metricsResponse.status === 200) {
            const metrics = metricsResponse.json();
            if (metrics.scaling_events) {
                autoScalingEvents.add(metrics.scaling_events);
            }
        }
        
        // Test primary endpoints under spike conditions
        const primaryEndpoints = [
            `${BASE_URL}/api/users`,
            `${BASE_URL}/api/products`,
            `${BASE_URL}/api/orders`,
            `${BASE_URL}/health`,
        ];
        
        primaryEndpoints.forEach((endpoint, index) => {
            const response = http.get(endpoint, { timeout: '5s' });
            
            check(response, {
                [`endpoint ${index} responded`]: (r) => r.status !== 0,
                [`endpoint ${index} not server error`]: (r) => r.status < 500,
            });
            
            errorRate.add(response.status >= 400);
            responseTime.add(response.timings.duration);
        });
    });

    sleep(0.1);

    // Scenario-specific spike testing
    switch (scenario) {
        case 'user_registration_spike':
            group('User Registration Spike', () => {
                const userData = {
                    name: `SpikeUser${Math.random().toString(36).substring(7)}`,
                    email: `spike.user.${Date.now()}@example.com`,
                    password: 'spiketest123',
                    source: 'spike_test',
                };
                
                const regResponse = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify(userData), {
                    headers: headers,
                    timeout: '10s',
                });
                
                check(regResponse, {
                    'registration spike handled': (r) => r.status !== 0,
                    'registration not overwhelmed': (r) => r.status !== 429,
                });
                
                errorRate.add(regResponse.status >= 400);
                responseTime.add(regResponse.timings.duration);
            });
            break;
            
        case 'product_search_spike':
            group('Product Search Spike', () => {
                const searchQueries = [
                    'laptop', 'phone', 'book', 'clothing', 'electronics',
                    'gaming', 'sports', 'home', 'garden', 'automotive',
                ];
                
                const query = searchQueries[Math.floor(Math.random() * searchQueries.length)];
                const searchResponse = http.get(`${BASE_URL}/api/products/search?q=${query}&limit=20`, {
                    timeout: '8s',
                });
                
                check(searchResponse, {
                    'search spike handled': (r) => r.status !== 0,
                    'search not rate limited': (r) => r.status !== 429,
                    'search results returned': (r) => r.status === 200,
                });
                
                errorRate.add(searchResponse.status >= 400);
                responseTime.add(searchResponse.timings.duration);
            });
            break;
            
        case 'order_processing_spike':
            group('Order Processing Spike', () => {
                const orderData = {
                    userId: Math.floor(Math.random() * 1000) + 1,
                    items: [
                        {
                            productId: Math.floor(Math.random() * 100) + 1,
                            quantity: Math.floor(Math.random() * 3) + 1,
                        },
                    ],
                    totalAmount: Math.floor(Math.random() * 500) + 50,
                    source: 'spike_test',
                };
                
                const orderResponse = http.post(`${BASE_URL}/api/orders`, JSON.stringify(orderData), {
                    headers: headers,
                    timeout: '15s',
                });
                
                check(orderResponse, {
                    'order spike handled': (r) => r.status !== 0,
                    'order processing not overwhelmed': (r) => r.status !== 503,
                });
                
                errorRate.add(orderResponse.status >= 400);
                responseTime.add(orderResponse.timings.duration);
            });
            break;
            
        case 'authentication_spike':
            group('Authentication Spike', () => {
                const authData = {
                    email: `spike.auth.${Math.random().toString(36).substring(7)}@example.com`,
                    password: 'testpassword123',
                };
                
                const authResponse = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify(authData), {
                    headers: headers,
                    timeout: '5s',
                });
                
                check(authResponse, {
                    'auth spike handled': (r) => r.status !== 0,
                    'auth not rate limited': (r) => r.status !== 429,
                });
                
                errorRate.add(authResponse.status >= 400 && authResponse.status !== 401);
                responseTime.add(authResponse.timings.duration);
            });
            break;
            
        case 'api_gateway_spike':
            group('API Gateway Spike', () => {
                // Test multiple endpoints simultaneously
                const endpoints = [
                    `${BASE_URL}/api/users/1`,
                    `${BASE_URL}/api/products/1`,
                    `${BASE_URL}/api/categories`,
                    `${BASE_URL}/api/health`,
                ];
                
                endpoints.forEach((endpoint, index) => {
                    const response = http.get(endpoint, { timeout: '5s' });
                    
                    check(response, {
                        [`gateway endpoint ${index} responded`]: (r) => r.status !== 0,
                        [`gateway endpoint ${index} not overloaded`]: (r) => r.status !== 503,
                    });
                    
                    errorRate.add(response.status >= 400);
                    responseTime.add(response.timings.duration);
                });
            });
            break;
    }

    sleep(0.1);

    // Recovery time measurement
    group('Recovery Time Measurement', () => {
        if (spikeStartTime && currentVUs <= 10) {
            const recoveryTime = Date.now() - spikeStartTime;
            spikeRecoveryTime.add(recoveryTime);
            spikeStartTime = null; // Reset for next spike
        }
        
        // Test system responsiveness during recovery
        const healthResponse = http.get(`${BASE_URL}/health`, { timeout: '3s' });
        
        check(healthResponse, {
            'system recovering': (r) => r.status === 200,
            'recovery response time acceptable': (r) => r.timings.duration < 1000,
        });
        
        errorRate.add(healthResponse.status !== 200);
        responseTime.add(healthResponse.timings.duration);
    });

    // Rate limiting validation
    group('Rate Limiting Validation', () => {
        // Make multiple rapid requests to test rate limiting
        for (let i = 0; i < 5; i++) {
            const response = http.get(`${BASE_URL}/api/rate-limit-test`, {
                timeout: '3s',
            });
            
            check(response, {
                'rate limit request processed': (r) => r.status !== 0,
                'rate limit properly enforced': (r) => r.status === 200 || r.status === 429,
            });
            
            if (response.status === 429) {
                // Rate limiting is working
                const retryAfter = response.headers['Retry-After'];
                if (retryAfter) {
                    sleep(parseInt(retryAfter) / 1000);
                }
            }
            
            errorRate.add(response.status >= 400 && response.status !== 429);
            responseTime.add(response.timings.duration);
        }
    });

    sleep(0.1);
}

// Setup function
export function setup() {
    console.log('Setting up spike test environment...');
    
    // Verify service is available
    const healthResponse = http.get(`${BASE_URL}/health`);
    if (healthResponse.status !== 200) {
        console.error(`Service not available at ${BASE_URL} for spike testing`);
        return null;
    }
    
    // Pre-warm the service
    console.log('Pre-warming service before spike test...');
    for (let i = 0; i < 5; i++) {
        http.get(`${BASE_URL}/api/warmup`);
        sleep(1);
    }
    
    console.log('Starting spike test - simulating sudden traffic surges');
    return { serviceAvailable: true };
}

// Teardown function
export function teardown(data) {
    console.log('Spike test completed');
    
    if (data && data.serviceAvailable) {
        console.log('Cleaning up spike test data...');
        
        // Clean up test data
        const cleanupResponse = http.delete(`${BASE_URL}/api/cleanup/spike-test`, {
            headers: headers,
            timeout: '30s',
        });
        
        if (cleanupResponse.status === 200) {
            console.log('Spike test data cleaned up successfully');
        }
        
        // Generate recovery report
        const reportResponse = http.get(`${BASE_URL}/api/reports/spike-test-recovery`, {
            timeout: '10s',
        });
        
        if (reportResponse.status === 200) {
            console.log('Spike test recovery report generated');
        }
    }
    
    console.log('Review circuit breaker triggers and auto-scaling events in the results');
}