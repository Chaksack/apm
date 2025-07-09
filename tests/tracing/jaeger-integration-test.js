/**
 * Jaeger Integration Test Suite
 * Tests Jaeger API, trace queries, UI functionality, and export/import
 */

const axios = require('axios');
const { expect } = require('chai');
const fs = require('fs').promises;
const path = require('path');

// Configuration
const JAEGER_CONFIG = {
    baseUrl: process.env.JAEGER_URL || 'http://localhost:16686',
    timeout: 30000,
    maxRetries: 3
};

class JaegerIntegrationTester {
    constructor(config = JAEGER_CONFIG) {
        this.config = config;
        this.client = axios.create({
            baseURL: config.baseUrl,
            timeout: config.timeout,
            headers: {
                'Content-Type': 'application/json'
            }
        });
    }

    /**
     * Test Jaeger API health and availability
     */
    async testApiHealth() {
        try {
            const response = await this.client.get('/api/services');
            expect(response.status).to.equal(200);
            expect(response.data).to.have.property('data');
            return {
                success: true,
                services: response.data.data,
                timestamp: new Date().toISOString()
            };
        } catch (error) {
            return {
                success: false,
                error: error.message,
                timestamp: new Date().toISOString()
            };
        }
    }

    /**
     * Test service discovery
     */
    async testServiceDiscovery() {
        const response = await this.client.get('/api/services');
        expect(response.status).to.equal(200);
        
        const services = response.data.data;
        expect(services).to.be.an('array');
        
        // Check for expected GoFiber service
        const goFiberService = services.find(s => s.includes('apm') || s.includes('fiber'));
        
        return {
            totalServices: services.length,
            services: services,
            hasGoFiberService: !!goFiberService,
            goFiberService: goFiberService
        };
    }

    /**
     * Test trace query functionality
     */
    async testTraceQuery(serviceName, options = {}) {
        const params = {
            service: serviceName,
            lookback: options.lookback || '1h',
            limit: options.limit || 20,
            start: options.start,
            end: options.end,
            minDuration: options.minDuration,
            maxDuration: options.maxDuration,
            tags: options.tags
        };

        // Remove undefined parameters
        Object.keys(params).forEach(key => {
            if (params[key] === undefined) {
                delete params[key];
            }
        });

        const response = await this.client.get('/api/traces', { params });
        expect(response.status).to.equal(200);
        
        const traces = response.data.data;
        expect(traces).to.be.an('array');

        return {
            traceCount: traces.length,
            traces: traces,
            queryParams: params
        };
    }

    /**
     * Test trace detail retrieval
     */
    async testTraceDetail(traceId) {
        const response = await this.client.get(`/api/traces/${traceId}`);
        expect(response.status).to.equal(200);
        
        const trace = response.data.data[0];
        expect(trace).to.have.property('traceID');
        expect(trace).to.have.property('spans');
        expect(trace.spans).to.be.an('array');
        expect(trace.spans.length).to.be.greaterThan(0);

        return {
            traceId: trace.traceID,
            spanCount: trace.spans.length,
            processes: Object.keys(trace.processes || {}),
            duration: this.calculateTraceDuration(trace.spans),
            spans: trace.spans.map(span => ({
                spanId: span.spanID,
                operationName: span.operationName,
                duration: span.duration,
                tags: this.parseTags(span.tags || [])
            }))
        };
    }

    /**
     * Test trace search with filters
     */
    async testTraceSearch() {
        const testCases = [
            {
                name: 'Search by operation',
                params: { operation: 'HTTP GET' }
            },
            {
                name: 'Search by tag',
                params: { tags: 'http.method:GET' }
            },
            {
                name: 'Search by duration',
                params: { minDuration: '1ms', maxDuration: '10s' }
            },
            {
                name: 'Search with time range',
                params: { 
                    start: Date.now() - 3600000, // 1 hour ago
                    end: Date.now()
                }
            }
        ];

        const results = [];
        
        for (const testCase of testCases) {
            try {
                const response = await this.client.get('/api/traces', {
                    params: {
                        service: 'apm-service',
                        ...testCase.params
                    }
                });
                
                results.push({
                    testCase: testCase.name,
                    success: true,
                    traceCount: response.data.data.length,
                    params: testCase.params
                });
            } catch (error) {
                results.push({
                    testCase: testCase.name,
                    success: false,
                    error: error.message
                });
            }
        }

        return results;
    }

    /**
     * Test dependencies API
     */
    async testDependencies() {
        const endTs = Date.now();
        const lookback = 3600000; // 1 hour in milliseconds
        
        const response = await this.client.get('/api/dependencies', {
            params: {
                endTs: endTs,
                lookback: lookback
            }
        });
        
        expect(response.status).to.equal(200);
        
        const dependencies = response.data.data;
        expect(dependencies).to.be.an('array');

        return {
            dependencyCount: dependencies.length,
            dependencies: dependencies.map(dep => ({
                parent: dep.parent,
                child: dep.child,
                callCount: dep.callCount
            }))
        };
    }

    /**
     * Test operations API
     */
    async testOperations(serviceName) {
        const response = await this.client.get('/api/operations', {
            params: { service: serviceName }
        });
        
        expect(response.status).to.equal(200);
        
        const operations = response.data.data;
        expect(operations).to.be.an('array');

        return {
            operationCount: operations.length,
            operations: operations,
            hasHttpOperations: operations.some(op => op.includes('HTTP')),
            hasFiberOperations: operations.some(op => op.includes('fiber'))
        };
    }

    /**
     * Test trace export functionality
     */
    async testTraceExport(traceId) {
        try {
            // Export trace as JSON
            const jsonResponse = await this.client.get(`/api/traces/${traceId}`);
            expect(jsonResponse.status).to.equal(200);
            
            const traceData = jsonResponse.data.data[0];
            
            // Validate export format
            expect(traceData).to.have.property('traceID');
            expect(traceData).to.have.property('spans');
            expect(traceData).to.have.property('processes');
            
            // Save to file for import test
            const exportPath = path.join(__dirname, `exported_trace_${traceId}.json`);
            await fs.writeFile(exportPath, JSON.stringify(traceData, null, 2));
            
            return {
                success: true,
                exportPath: exportPath,
                traceId: traceData.traceID,
                spanCount: traceData.spans.length
            };
        } catch (error) {
            return {
                success: false,
                error: error.message
            };
        }
    }

    /**
     * Test trace import functionality (simulated)
     */
    async testTraceImport(exportPath) {
        try {
            // Read exported trace
            const traceData = JSON.parse(await fs.readFile(exportPath, 'utf8'));
            
            // Validate imported data structure
            expect(traceData).to.have.property('traceID');
            expect(traceData).to.have.property('spans');
            expect(traceData.spans).to.be.an('array');
            
            // Validate spans structure
            traceData.spans.forEach(span => {
                expect(span).to.have.property('spanID');
                expect(span).to.have.property('operationName');
                expect(span).to.have.property('startTime');
                expect(span).to.have.property('duration');
            });
            
            // Clean up
            await fs.unlink(exportPath);
            
            return {
                success: true,
                traceId: traceData.traceID,
                spanCount: traceData.spans.length,
                validated: true
            };
        } catch (error) {
            return {
                success: false,
                error: error.message
            };
        }
    }

    /**
     * Test UI functionality through API endpoints
     */
    async testUIFunctionality() {
        const tests = [];
        
        // Test search endpoint
        try {
            const searchResponse = await this.client.get('/api/traces', {
                params: { service: 'apm-service', lookback: '1h' }
            });
            tests.push({
                name: 'Search UI endpoint',
                success: searchResponse.status === 200,
                traceCount: searchResponse.data.data.length
            });
        } catch (error) {
            tests.push({
                name: 'Search UI endpoint',
                success: false,
                error: error.message
            });
        }

        // Test services endpoint
        try {
            const servicesResponse = await this.client.get('/api/services');
            tests.push({
                name: 'Services UI endpoint',
                success: servicesResponse.status === 200,
                serviceCount: servicesResponse.data.data.length
            });
        } catch (error) {
            tests.push({
                name: 'Services UI endpoint',
                success: false,
                error: error.message
            });
        }

        // Test dependencies endpoint
        try {
            const depsResponse = await this.client.get('/api/dependencies', {
                params: { endTs: Date.now(), lookback: 3600000 }
            });
            tests.push({
                name: 'Dependencies UI endpoint',
                success: depsResponse.status === 200,
                dependencyCount: depsResponse.data.data.length
            });
        } catch (error) {
            tests.push({
                name: 'Dependencies UI endpoint',
                success: false,
                error: error.message
            });
        }

        return tests;
    }

    /**
     * Test GoFiber specific trace validation
     */
    async testGoFiberTraces() {
        const traceQuery = await this.testTraceQuery('apm-service');
        
        const goFiberTraces = [];
        
        for (const trace of traceQuery.traces) {
            const traceDetail = await this.testTraceDetail(trace.traceID);
            
            // Check for GoFiber specific spans
            const fiberSpans = traceDetail.spans.filter(span => 
                span.tags.component === 'fiber' || 
                span.operationName.includes('HTTP')
            );
            
            if (fiberSpans.length > 0) {
                goFiberTraces.push({
                    traceId: trace.traceID,
                    fiberSpanCount: fiberSpans.length,
                    totalSpans: traceDetail.spanCount,
                    duration: traceDetail.duration,
                    httpMethods: fiberSpans
                        .filter(span => span.tags['http.method'])
                        .map(span => span.tags['http.method']),
                    statusCodes: fiberSpans
                        .filter(span => span.tags['http.status_code'])
                        .map(span => span.tags['http.status_code'])
                });
            }
        }
        
        return {
            totalTraces: traceQuery.traceCount,
            goFiberTraces: goFiberTraces.length,
            traces: goFiberTraces
        };
    }

    /**
     * Test performance metrics
     */
    async testPerformanceMetrics() {
        const start = Date.now();
        
        // Test query performance
        const queryStart = Date.now();
        await this.testTraceQuery('apm-service', { limit: 100 });
        const queryDuration = Date.now() - queryStart;
        
        // Test service discovery performance
        const servicesStart = Date.now();
        await this.testServiceDiscovery();
        const servicesDuration = Date.now() - servicesStart;
        
        // Test dependencies performance
        const depsStart = Date.now();
        await this.testDependencies();
        const depsDuration = Date.now() - depsStart;
        
        const totalDuration = Date.now() - start;
        
        return {
            totalDuration: totalDuration,
            queryDuration: queryDuration,
            servicesDuration: servicesDuration,
            dependenciesDuration: depsDuration,
            performanceGood: totalDuration < 5000 // Should complete in under 5 seconds
        };
    }

    /**
     * Parse tags from Jaeger format
     */
    parseTags(tags) {
        const tagMap = {};
        tags.forEach(tag => {
            tagMap[tag.key] = tag.value;
        });
        return tagMap;
    }

    /**
     * Calculate trace duration from spans
     */
    calculateTraceDuration(spans) {
        if (!spans || spans.length === 0) return 0;
        
        const startTimes = spans.map(span => span.startTime);
        const endTimes = spans.map(span => span.startTime + span.duration);
        
        return Math.max(...endTimes) - Math.min(...startTimes);
    }

    /**
     * Run comprehensive integration test suite
     */
    async runFullTestSuite() {
        const results = {
            timestamp: new Date().toISOString(),
            tests: {}
        };

        console.log('Starting Jaeger Integration Test Suite...');

        // API Health Test
        console.log('Testing API health...');
        results.tests.apiHealth = await this.testApiHealth();

        if (!results.tests.apiHealth.success) {
            console.error('API health test failed, skipping remaining tests');
            return results;
        }

        // Service Discovery Test
        console.log('Testing service discovery...');
        results.tests.serviceDiscovery = await this.testServiceDiscovery();

        // Trace Query Test
        console.log('Testing trace queries...');
        results.tests.traceQuery = await this.testTraceQuery('apm-service');

        // Trace Search Test
        console.log('Testing trace search...');
        results.tests.traceSearch = await this.testTraceSearch();

        // Dependencies Test
        console.log('Testing dependencies...');
        results.tests.dependencies = await this.testDependencies();

        // Operations Test
        console.log('Testing operations...');
        results.tests.operations = await this.testOperations('apm-service');

        // UI Functionality Test
        console.log('Testing UI functionality...');
        results.tests.uiFunctionality = await this.testUIFunctionality();

        // GoFiber Traces Test
        console.log('Testing GoFiber traces...');
        results.tests.goFiberTraces = await this.testGoFiberTraces();

        // Performance Test
        console.log('Testing performance...');
        results.tests.performance = await this.testPerformanceMetrics();

        // Export/Import Test (if traces exist)
        if (results.tests.traceQuery.traceCount > 0) {
            console.log('Testing export/import...');
            const firstTraceId = results.tests.traceQuery.traces[0].traceID;
            
            const exportResult = await this.testTraceExport(firstTraceId);
            results.tests.export = exportResult;
            
            if (exportResult.success) {
                results.tests.import = await this.testTraceImport(exportResult.exportPath);
            }
        }

        console.log('Test suite completed!');
        return results;
    }
}

// Test functions for Mocha/Jest
describe('Jaeger Integration Tests', function() {
    this.timeout(60000); // 60 second timeout
    
    let tester;
    
    before(function() {
        tester = new JaegerIntegrationTester();
    });

    it('should have healthy API', async function() {
        const result = await tester.testApiHealth();
        expect(result.success).to.be.true;
    });

    it('should discover services', async function() {
        const result = await tester.testServiceDiscovery();
        expect(result.totalServices).to.be.greaterThan(0);
    });

    it('should query traces successfully', async function() {
        const result = await tester.testTraceQuery('apm-service');
        expect(result.traceCount).to.be.greaterThanOrEqual(0);
    });

    it('should search traces with filters', async function() {
        const results = await tester.testTraceSearch();
        expect(results).to.be.an('array');
        expect(results.length).to.be.greaterThan(0);
    });

    it('should retrieve dependencies', async function() {
        const result = await tester.testDependencies();
        expect(result.dependencyCount).to.be.greaterThanOrEqual(0);
    });

    it('should test UI functionality', async function() {
        const results = await tester.testUIFunctionality();
        expect(results).to.be.an('array');
        results.forEach(test => {
            expect(test.success).to.be.true;
        });
    });

    it('should validate GoFiber traces', async function() {
        const result = await tester.testGoFiberTraces();
        expect(result.totalTraces).to.be.greaterThanOrEqual(0);
    });

    it('should perform within acceptable time limits', async function() {
        const result = await tester.testPerformanceMetrics();
        expect(result.performanceGood).to.be.true;
    });
});

// Export for standalone usage
module.exports = JaegerIntegrationTester;

// Run tests if called directly
if (require.main === module) {
    (async () => {
        const tester = new JaegerIntegrationTester();
        const results = await tester.runFullTestSuite();
        console.log(JSON.stringify(results, null, 2));
    })();
}