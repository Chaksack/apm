import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');

// Test configuration
export const options = {
    stages: [
        { duration: '2m', target: 10 },   // Ramp up to 10 users
        { duration: '5m', target: 10 },   // Stay at 10 users
        { duration: '2m', target: 20 },   // Ramp up to 20 users
        { duration: '5m', target: 20 },   // Stay at 20 users
        { duration: '3m', target: 0 },    // Ramp down to 0 users
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
        http_req_failed: ['rate<0.1'],     // Error rate under 10%
        errors: ['rate<0.1'],              // Custom error rate under 10%
        http_reqs: ['rate>10'],            // At least 10 requests per second
    },
};

// Base URL configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';

// Test data
const testData = {
    users: [
        { name: 'John Doe', email: 'john@example.com', age: 30 },
        { name: 'Jane Smith', email: 'jane@example.com', age: 25 },
        { name: 'Bob Johnson', email: 'bob@example.com', age: 35 },
    ],
    products: [
        { name: 'Product A', price: 29.99, category: 'electronics' },
        { name: 'Product B', price: 19.99, category: 'books' },
        { name: 'Product C', price: 49.99, category: 'clothing' },
    ],
};

// Headers for requests
const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
};

export default function () {
    // Health check scenario
    group('Health Check', () => {
        const response = http.get(`${BASE_URL}/health`);
        
        check(response, {
            'health check status is 200': (r) => r.status === 200,
            'health check response time < 100ms': (r) => r.timings.duration < 100,
        });
        
        errorRate.add(response.status !== 200);
        responseTime.add(response.timings.duration);
    });

    sleep(1);

    // User management scenario
    group('User Management', () => {
        const userData = testData.users[Math.floor(Math.random() * testData.users.length)];
        
        // Create user
        const createResponse = http.post(`${BASE_URL}/api/users`, JSON.stringify(userData), {
            headers: headers,
        });
        
        check(createResponse, {
            'user creation status is 201': (r) => r.status === 201,
            'user creation response time < 200ms': (r) => r.timings.duration < 200,
            'user creation returns user data': (r) => r.json() && r.json().id,
        });
        
        errorRate.add(createResponse.status !== 201);
        responseTime.add(createResponse.timings.duration);
        
        // Get user list
        const listResponse = http.get(`${BASE_URL}/api/users`);
        
        check(listResponse, {
            'user list status is 200': (r) => r.status === 200,
            'user list response time < 300ms': (r) => r.timings.duration < 300,
            'user list returns array': (r) => Array.isArray(r.json()),
        });
        
        errorRate.add(listResponse.status !== 200);
        responseTime.add(listResponse.timings.duration);
        
        // If user was created, get specific user
        if (createResponse.status === 201) {
            const userId = createResponse.json().id;
            const getUserResponse = http.get(`${BASE_URL}/api/users/${userId}`);
            
            check(getUserResponse, {
                'get user status is 200': (r) => r.status === 200,
                'get user response time < 150ms': (r) => r.timings.duration < 150,
                'get user returns correct data': (r) => r.json() && r.json().id === userId,
            });
            
            errorRate.add(getUserResponse.status !== 200);
            responseTime.add(getUserResponse.timings.duration);
        }
    });

    sleep(1);

    // Product catalog scenario
    group('Product Catalog', () => {
        // Get all products
        const productsResponse = http.get(`${BASE_URL}/api/products`);
        
        check(productsResponse, {
            'products list status is 200': (r) => r.status === 200,
            'products list response time < 400ms': (r) => r.timings.duration < 400,
            'products list returns data': (r) => r.json(),
        });
        
        errorRate.add(productsResponse.status !== 200);
        responseTime.add(productsResponse.timings.duration);
        
        // Search products
        const searchResponse = http.get(`${BASE_URL}/api/products/search?q=electronics`);
        
        check(searchResponse, {
            'product search status is 200': (r) => r.status === 200,
            'product search response time < 500ms': (r) => r.timings.duration < 500,
        });
        
        errorRate.add(searchResponse.status !== 200);
        responseTime.add(searchResponse.timings.duration);
        
        // Get product categories
        const categoriesResponse = http.get(`${BASE_URL}/api/categories`);
        
        check(categoriesResponse, {
            'categories status is 200': (r) => r.status === 200,
            'categories response time < 200ms': (r) => r.timings.duration < 200,
        });
        
        errorRate.add(categoriesResponse.status !== 200);
        responseTime.add(categoriesResponse.timings.duration);
    });

    sleep(1);

    // Authentication scenario
    group('Authentication', () => {
        const loginData = {
            email: 'test@example.com',
            password: 'testpassword123',
        };
        
        const loginResponse = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify(loginData), {
            headers: headers,
        });
        
        check(loginResponse, {
            'login attempt processed': (r) => r.status === 200 || r.status === 401,
            'login response time < 300ms': (r) => r.timings.duration < 300,
        });
        
        errorRate.add(loginResponse.status >= 500);
        responseTime.add(loginResponse.timings.duration);
        
        // If login successful, test protected endpoint
        if (loginResponse.status === 200) {
            const token = loginResponse.json().token;
            const protectedResponse = http.get(`${BASE_URL}/api/protected`, {
                headers: {
                    ...headers,
                    Authorization: `Bearer ${token}`,
                },
            });
            
            check(protectedResponse, {
                'protected endpoint status is 200': (r) => r.status === 200,
                'protected endpoint response time < 200ms': (r) => r.timings.duration < 200,
            });
            
            errorRate.add(protectedResponse.status !== 200);
            responseTime.add(protectedResponse.timings.duration);
        }
    });

    sleep(1);
}

// Setup function - runs once before all VUs
export function setup() {
    console.log('Setting up load test environment...');
    
    // Test if the service is available
    const healthResponse = http.get(`${BASE_URL}/health`);
    if (healthResponse.status !== 200) {
        console.error(`Service not available at ${BASE_URL}, status: ${healthResponse.status}`);
        return null;
    }
    
    console.log('Service is available, starting load test...');
    return { serviceAvailable: true };
}

// Teardown function - runs once after all VUs
export function teardown(data) {
    console.log('Load test completed');
    
    if (data && data.serviceAvailable) {
        console.log('Cleaning up test data...');
        // Add cleanup logic here if needed
    }
}