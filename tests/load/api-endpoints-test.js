import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const payloadSizeMetric = new Trend('payload_size');
const apiCallsCounter = new Counter('api_calls_total');
const endpointErrors = new Counter('endpoint_errors');

// API endpoints test configuration
export const options = {
    stages: [
        { duration: '1m', target: 10 },   // Warm up
        { duration: '5m', target: 25 },   // Normal load
        { duration: '2m', target: 40 },   // Increased load
        { duration: '3m', target: 40 },   // Sustain load
        { duration: '2m', target: 0 },    // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<1000'], // 95% under 1s
        http_req_failed: ['rate<0.05'],    // Less than 5% failure rate
        errors: ['rate<0.05'],             // Custom error rate under 5%
        api_calls_total: ['count>1000'],   // At least 1000 API calls
    },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';

// Test data with different payload sizes
const testData = {
    small: {
        user: {
            name: 'John',
            email: 'john@test.com',
            age: 25,
        },
        product: {
            name: 'Test Product',
            price: 19.99,
            category: 'test',
        },
    },
    medium: {
        user: {
            name: 'Jane Smith',
            email: 'jane.smith@example.com',
            age: 30,
            bio: 'Medium sized bio with some details about the user.',
            preferences: {
                theme: 'dark',
                language: 'en',
                notifications: true,
            },
            address: {
                street: '123 Main St',
                city: 'Anytown',
                state: 'CA',
                zip: '12345',
            },
        },
        product: {
            name: 'Medium Product with Longer Name',
            description: 'This is a medium-sized product description with more details about the product features and benefits.',
            price: 49.99,
            category: 'electronics',
            tags: ['popular', 'new', 'featured'],
            specifications: {
                weight: '1.5kg',
                dimensions: '10x20x30cm',
                color: 'black',
            },
        },
    },
    large: {
        user: {
            name: 'Robert Johnson',
            email: 'robert.johnson@example.com',
            age: 35,
            bio: 'Large bio with extensive details about the user, their background, interests, and professional experience. This user has been with the platform for several years and has built up a comprehensive profile.',
            preferences: {
                theme: 'light',
                language: 'en',
                notifications: true,
                privacy: 'public',
                newsletter: true,
            },
            address: {
                street: '456 Oak Avenue, Apartment 2B',
                city: 'Metropolitan City',
                state: 'NY',
                zip: '54321',
                country: 'USA',
            },
            socialProfiles: {
                twitter: '@robertj',
                linkedin: 'robert-johnson',
                github: 'rjohnson',
            },
            activityHistory: Array.from({ length: 50 }, (_, i) => ({
                action: `Action ${i}`,
                timestamp: new Date(Date.now() - i * 86400000).toISOString(),
                details: `Details for action ${i} with additional context`,
            })),
        },
        product: {
            name: 'Large Product with Comprehensive Details and Extended Name',
            description: 'This is a large product with extensive description including detailed specifications, features, benefits, usage instructions, warranty information, and customer reviews. The product has been thoroughly tested and comes with full documentation.',
            price: 199.99,
            category: 'electronics',
            tags: ['premium', 'featured', 'bestseller', 'new', 'recommended'],
            specifications: {
                weight: '2.5kg',
                dimensions: '15x25x35cm',
                color: 'silver',
                material: 'aluminum',
                warranty: '2 years',
                certifications: ['CE', 'FCC', 'RoHS'],
            },
            features: Array.from({ length: 20 }, (_, i) => `Feature ${i + 1} with detailed description`),
            reviews: Array.from({ length: 30 }, (_, i) => ({
                id: i + 1,
                rating: Math.floor(Math.random() * 5) + 1,
                comment: `Review comment ${i + 1} with detailed feedback about the product`,
                author: `User ${i + 1}`,
                date: new Date(Date.now() - i * 86400000).toISOString(),
            })),
        },
    },
};

// Headers for requests
const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
};

// Authentication token for protected endpoints
let authToken = null;

// API endpoint definitions
const endpoints = {
    auth: {
        login: '/api/auth/login',
        register: '/api/auth/register',
        logout: '/api/auth/logout',
        refresh: '/api/auth/refresh',
        forgot: '/api/auth/forgot-password',
        reset: '/api/auth/reset-password',
    },
    users: {
        list: '/api/users',
        create: '/api/users',
        get: '/api/users/:id',
        update: '/api/users/:id',
        delete: '/api/users/:id',
        profile: '/api/users/profile',
        search: '/api/users/search',
    },
    products: {
        list: '/api/products',
        create: '/api/products',
        get: '/api/products/:id',
        update: '/api/products/:id',
        delete: '/api/products/:id',
        search: '/api/products/search',
        categories: '/api/categories',
        featured: '/api/products/featured',
    },
    orders: {
        list: '/api/orders',
        create: '/api/orders',
        get: '/api/orders/:id',
        update: '/api/orders/:id',
        cancel: '/api/orders/:id/cancel',
        history: '/api/orders/history',
    },
    admin: {
        stats: '/api/admin/stats',
        users: '/api/admin/users',
        reports: '/api/admin/reports',
        settings: '/api/admin/settings',
    },
    misc: {
        health: '/health',
        metrics: '/metrics',
        version: '/api/version',
        status: '/api/status',
    },
};

export default function () {
    // Authentication flow
    group('Authentication Endpoints', () => {
        // Register new user
        const registerData = {
            ...testData.small.user,
            email: `test.${Math.random().toString(36).substring(7)}@example.com`,
            password: 'testpassword123',
        };
        
        const registerResponse = http.post(`${BASE_URL}${endpoints.auth.register}`, JSON.stringify(registerData), {
            headers: headers,
        });
        
        check(registerResponse, {
            'user registration status': (r) => r.status === 201 || r.status === 200,
            'user registration response time': (r) => r.timings.duration < 500,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(registerResponse.status >= 400);
        responseTime.add(registerResponse.timings.duration);
        payloadSizeMetric.add(JSON.stringify(registerData).length);
        
        // Login
        const loginData = {
            email: registerData.email,
            password: registerData.password,
        };
        
        const loginResponse = http.post(`${BASE_URL}${endpoints.auth.login}`, JSON.stringify(loginData), {
            headers: headers,
        });
        
        check(loginResponse, {
            'login status': (r) => r.status === 200 || r.status === 401,
            'login response time': (r) => r.timings.duration < 300,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(loginResponse.status >= 400 && loginResponse.status !== 401);
        responseTime.add(loginResponse.timings.duration);
        
        if (loginResponse.status === 200) {
            try {
                authToken = loginResponse.json().token;
            } catch (e) {
                // Token might be in a different format
            }
        }
    });

    sleep(0.5);

    // User management endpoints with different payload sizes
    group('User Management Endpoints', () => {
        const payloadSizes = ['small', 'medium', 'large'];
        const payloadSize = payloadSizes[Math.floor(Math.random() * payloadSizes.length)];
        const userData = testData[payloadSize].user;
        
        // Create user
        const createResponse = http.post(`${BASE_URL}${endpoints.users.create}`, JSON.stringify(userData), {
            headers: headers,
        });
        
        check(createResponse, {
            [`user creation ${payloadSize} payload`]: (r) => r.status === 201 || r.status === 200,
            [`user creation ${payloadSize} response time`]: (r) => r.timings.duration < 1000,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(createResponse.status >= 400);
        responseTime.add(createResponse.timings.duration);
        payloadSizeMetric.add(JSON.stringify(userData).length);
        
        // Get user list
        const listResponse = http.get(`${BASE_URL}${endpoints.users.list}?limit=50&offset=0`);
        
        check(listResponse, {
            'user list status': (r) => r.status === 200,
            'user list response time': (r) => r.timings.duration < 500,
            'user list returns data': (r) => r.body.length > 0,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(listResponse.status !== 200);
        responseTime.add(listResponse.timings.duration);
        
        // Search users
        const searchResponse = http.get(`${BASE_URL}${endpoints.users.search}?q=test&limit=10`);
        
        check(searchResponse, {
            'user search status': (r) => r.status === 200,
            'user search response time': (r) => r.timings.duration < 600,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(searchResponse.status !== 200);
        responseTime.add(searchResponse.timings.duration);
        
        // Get specific user (if creation was successful)
        if (createResponse.status === 201 || createResponse.status === 200) {
            try {
                const userId = createResponse.json().id;
                const getUserResponse = http.get(`${BASE_URL}${endpoints.users.get.replace(':id', userId)}`);
                
                check(getUserResponse, {
                    'get user status': (r) => r.status === 200,
                    'get user response time': (r) => r.timings.duration < 300,
                });
                
                apiCallsCounter.add(1);
                errorRate.add(getUserResponse.status !== 200);
                responseTime.add(getUserResponse.timings.duration);
            } catch (e) {
                // User ID might not be available
            }
        }
    });

    sleep(0.5);

    // Product management endpoints
    group('Product Management Endpoints', () => {
        const payloadSizes = ['small', 'medium', 'large'];
        const payloadSize = payloadSizes[Math.floor(Math.random() * payloadSizes.length)];
        const productData = testData[payloadSize].product;
        
        // Create product
        const createResponse = http.post(`${BASE_URL}${endpoints.products.create}`, JSON.stringify(productData), {
            headers: authToken ? { ...headers, Authorization: `Bearer ${authToken}` } : headers,
        });
        
        check(createResponse, {
            [`product creation ${payloadSize} payload`]: (r) => r.status === 201 || r.status === 200 || r.status === 401,
            [`product creation ${payloadSize} response time`]: (r) => r.timings.duration < 1000,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(createResponse.status >= 400 && createResponse.status !== 401);
        responseTime.add(createResponse.timings.duration);
        payloadSizeMetric.add(JSON.stringify(productData).length);
        
        // Get product list
        const listResponse = http.get(`${BASE_URL}${endpoints.products.list}?limit=50&page=1`);
        
        check(listResponse, {
            'product list status': (r) => r.status === 200,
            'product list response time': (r) => r.timings.duration < 600,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(listResponse.status !== 200);
        responseTime.add(listResponse.timings.duration);
        
        // Search products
        const searchResponse = http.get(`${BASE_URL}${endpoints.products.search}?q=test&category=electronics&limit=20`);
        
        check(searchResponse, {
            'product search status': (r) => r.status === 200,
            'product search response time': (r) => r.timings.duration < 800,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(searchResponse.status !== 200);
        responseTime.add(searchResponse.timings.duration);
        
        // Get categories
        const categoriesResponse = http.get(`${BASE_URL}${endpoints.products.categories}`);
        
        check(categoriesResponse, {
            'categories status': (r) => r.status === 200,
            'categories response time': (r) => r.timings.duration < 300,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(categoriesResponse.status !== 200);
        responseTime.add(categoriesResponse.timings.duration);
        
        // Get featured products
        const featuredResponse = http.get(`${BASE_URL}${endpoints.products.featured}`);
        
        check(featuredResponse, {
            'featured products status': (r) => r.status === 200,
            'featured products response time': (r) => r.timings.duration < 400,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(featuredResponse.status !== 200);
        responseTime.add(featuredResponse.timings.duration);
    });

    sleep(0.5);

    // Order management endpoints
    group('Order Management Endpoints', () => {
        const orderData = {
            userId: Math.floor(Math.random() * 100) + 1,
            items: [
                {
                    productId: Math.floor(Math.random() * 50) + 1,
                    quantity: Math.floor(Math.random() * 3) + 1,
                    price: Math.floor(Math.random() * 100) + 10,
                },
            ],
            totalAmount: Math.floor(Math.random() * 200) + 50,
            shippingAddress: {
                street: '123 Test St',
                city: 'Test City',
                state: 'TS',
                zip: '12345',
            },
        };
        
        // Create order
        const createResponse = http.post(`${BASE_URL}${endpoints.orders.create}`, JSON.stringify(orderData), {
            headers: authToken ? { ...headers, Authorization: `Bearer ${authToken}` } : headers,
        });
        
        check(createResponse, {
            'order creation status': (r) => r.status === 201 || r.status === 200 || r.status === 401,
            'order creation response time': (r) => r.timings.duration < 1000,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(createResponse.status >= 400 && createResponse.status !== 401);
        responseTime.add(createResponse.timings.duration);
        
        // Get order list
        const listResponse = http.get(`${BASE_URL}${endpoints.orders.list}?limit=20`, {
            headers: authToken ? { ...headers, Authorization: `Bearer ${authToken}` } : headers,
        });
        
        check(listResponse, {
            'order list status': (r) => r.status === 200 || r.status === 401,
            'order list response time': (r) => r.timings.duration < 500,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(listResponse.status >= 400 && listResponse.status !== 401);
        responseTime.add(listResponse.timings.duration);
        
        // Get order history
        const historyResponse = http.get(`${BASE_URL}${endpoints.orders.history}?limit=50`, {
            headers: authToken ? { ...headers, Authorization: `Bearer ${authToken}` } : headers,
        });
        
        check(historyResponse, {
            'order history status': (r) => r.status === 200 || r.status === 401,
            'order history response time': (r) => r.timings.duration < 600,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(historyResponse.status >= 400 && historyResponse.status !== 401);
        responseTime.add(historyResponse.timings.duration);
    });

    sleep(0.5);

    // Error scenario testing
    group('Error Scenario Testing', () => {
        // Test 404 errors
        const notFoundResponse = http.get(`${BASE_URL}/api/nonexistent-endpoint`);
        
        check(notFoundResponse, {
            'not found handled correctly': (r) => r.status === 404,
            'not found response time': (r) => r.timings.duration < 200,
        });
        
        apiCallsCounter.add(1);
        // Don't count 404 as error for this test
        responseTime.add(notFoundResponse.timings.duration);
        
        // Test invalid JSON
        const invalidJsonResponse = http.post(`${BASE_URL}${endpoints.users.create}`, 'invalid-json', {
            headers: headers,
        });
        
        check(invalidJsonResponse, {
            'invalid JSON handled': (r) => r.status === 400,
            'invalid JSON response time': (r) => r.timings.duration < 200,
        });
        
        apiCallsCounter.add(1);
        // Don't count 400 as error for this test
        responseTime.add(invalidJsonResponse.timings.duration);
        
        // Test large payload (if supported)
        const largePayload = {
            data: 'x'.repeat(1024 * 1024), // 1MB of data
        };
        
        const largePayloadResponse = http.post(`${BASE_URL}/api/large-payload-test`, JSON.stringify(largePayload), {
            headers: headers,
            timeout: '30s',
        });
        
        check(largePayloadResponse, {
            'large payload handled': (r) => r.status !== 0,
            'large payload response time': (r) => r.timings.duration < 30000,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(largePayloadResponse.status >= 500);
        responseTime.add(largePayloadResponse.timings.duration);
    });

    sleep(0.5);

    // System endpoints
    group('System Endpoints', () => {
        // Health check
        const healthResponse = http.get(`${BASE_URL}${endpoints.misc.health}`);
        
        check(healthResponse, {
            'health check status': (r) => r.status === 200,
            'health check response time': (r) => r.timings.duration < 100,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(healthResponse.status !== 200);
        responseTime.add(healthResponse.timings.duration);
        
        // Metrics (if available)
        const metricsResponse = http.get(`${BASE_URL}${endpoints.misc.metrics}`);
        
        check(metricsResponse, {
            'metrics endpoint accessible': (r) => r.status === 200 || r.status === 401 || r.status === 404,
            'metrics response time': (r) => r.timings.duration < 500,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(metricsResponse.status >= 500);
        responseTime.add(metricsResponse.timings.duration);
        
        // Version info
        const versionResponse = http.get(`${BASE_URL}${endpoints.misc.version}`);
        
        check(versionResponse, {
            'version endpoint accessible': (r) => r.status === 200 || r.status === 404,
            'version response time': (r) => r.timings.duration < 200,
        });
        
        apiCallsCounter.add(1);
        errorRate.add(versionResponse.status >= 500);
        responseTime.add(versionResponse.timings.duration);
    });

    sleep(0.5);
}

// Setup function
export function setup() {
    console.log('Setting up API endpoints test...');
    
    // Verify service is available
    const healthResponse = http.get(`${BASE_URL}/health`);
    if (healthResponse.status !== 200) {
        console.error(`Service not available at ${BASE_URL}`);
        return null;
    }
    
    console.log('Starting comprehensive API endpoints test');
    return { serviceAvailable: true };
}

// Teardown function
export function teardown(data) {
    console.log('API endpoints test completed');
    
    if (data && data.serviceAvailable) {
        console.log('Cleaning up test data...');
        
        // Clean up test data
        const cleanupResponse = http.delete(`${BASE_URL}/api/cleanup/api-test`, {
            headers: headers,
            timeout: '30s',
        });
        
        if (cleanupResponse.status === 200) {
            console.log('API test data cleaned up successfully');
        }
    }
    
    console.log('Review API endpoint performance metrics and error rates');
}