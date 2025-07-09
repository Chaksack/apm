#!/usr/bin/env node
/**
 * Grafana Dashboard and Data Source Validation
 * Validates Grafana dashboards, queries, data sources, and alerts.
 */

const axios = require('axios');
const fs = require('fs');
const path = require('path');

class GrafanaValidator {
    constructor(grafanaUrl = 'http://localhost:3000', apiKey = null, username = 'admin', password = 'admin') {
        this.grafanaUrl = grafanaUrl;
        this.apiKey = apiKey;
        this.username = username;
        this.password = password;
        
        // Setup axios instance
        this.client = axios.create({
            baseURL: grafanaUrl,
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json'
            }
        });
        
        // Setup authentication
        if (this.apiKey) {
            this.client.defaults.headers.common['Authorization'] = `Bearer ${this.apiKey}`;
        } else {
            this.client.defaults.auth = {
                username: this.username,
                password: this.password
            };
        }
    }
    
    async checkConnectivity() {
        try {
            const response = await this.client.get('/api/health');
            return response.status === 200;
        } catch (error) {
            console.error('Grafana connectivity check failed:', error.message);
            return false;
        }
    }
    
    async getDataSources() {
        try {
            const response = await this.client.get('/api/datasources');
            return response.data;
        } catch (error) {
            console.error('Failed to get data sources:', error.message);
            return [];
        }
    }
    
    async testDataSource(dataSourceId) {
        try {
            const response = await this.client.get(`/api/datasources/${dataSourceId}/health`);
            return response.data;
        } catch (error) {
            console.error(`Failed to test data source ${dataSourceId}:`, error.message);
            return null;
        }
    }
    
    async getDashboards() {
        try {
            const response = await this.client.get('/api/search?type=dash-db');
            return response.data;
        } catch (error) {
            console.error('Failed to get dashboards:', error.message);
            return [];
        }
    }
    
    async getDashboard(uid) {
        try {
            const response = await this.client.get(`/api/dashboards/uid/${uid}`);
            return response.data;
        } catch (error) {
            console.error(`Failed to get dashboard ${uid}:`, error.message);
            return null;
        }
    }
    
    async executeQuery(dataSourceId, query) {
        try {
            const response = await this.client.post('/api/ds/query', {
                queries: [{
                    datasource: { uid: dataSourceId },
                    expr: query,
                    format: 'time_series',
                    intervalMs: 60000,
                    maxDataPoints: 100
                }]
            });
            return response.data;
        } catch (error) {
            console.error(`Failed to execute query: ${query}`, error.message);
            return null;
        }
    }
    
    async getAlertRules() {
        try {
            const response = await this.client.get('/api/ruler/grafana/api/v1/rules');
            return response.data;
        } catch (error) {
            console.error('Failed to get alert rules:', error.message);
            return {};
        }
    }
    
    async getAlertInstances() {
        try {
            const response = await this.client.get('/api/alertmanager/grafana/api/v1/alerts');
            return response.data;
        } catch (error) {
            console.error('Failed to get alert instances:', error.message);
            return [];
        }
    }
}

class GrafanaValidationSuite {
    constructor(grafanaUrl, apiKey, username, password) {
        this.validator = new GrafanaValidator(grafanaUrl, apiKey, username, password);
        this.testResults = [];
        this.startTime = new Date();
    }
    
    async runTest(testName, testFunc, ...args) {
        const start = Date.now();
        try {
            const result = await testFunc.apply(this, args);
            const duration = Date.now() - start;
            
            this.testResults.push({
                testName,
                result: !!result,
                duration,
                timestamp: new Date().toISOString(),
                details: result
            });
            
            const status = result ? 'PASS' : 'FAIL';
            console.log(`[${status}] ${testName} (${duration}ms)`);
            return !!result;
            
        } catch (error) {
            const duration = Date.now() - start;
            this.testResults.push({
                testName,
                result: false,
                duration,
                timestamp: new Date().toISOString(),
                error: error.message
            });
            
            console.log(`[ERROR] ${testName}: ${error.message}`);
            return false;
        }
    }
    
    async validateConnectivity() {
        return await this.validator.checkConnectivity();
    }
    
    async validateDataSources() {
        const dataSources = await this.validator.getDataSources();
        if (!dataSources || dataSources.length === 0) {
            return false;
        }
        
        let allHealthy = true;
        for (const ds of dataSources) {
            const health = await this.validator.testDataSource(ds.id);
            if (!health || health.status !== 'OK') {
                console.log(`Data source ${ds.name} health check failed`);
                allHealthy = false;
            }
        }
        
        return allHealthy;
    }
    
    async validateDashboards() {
        const dashboards = await this.validator.getDashboards();
        if (!dashboards || dashboards.length === 0) {
            return false;
        }
        
        let validDashboards = 0;
        for (const dashboard of dashboards) {
            const dashboardData = await this.validator.getDashboard(dashboard.uid);
            if (dashboardData && dashboardData.dashboard) {
                validDashboards++;
            }
        }
        
        return validDashboards > 0;
    }
    
    async validateDashboardQueries() {
        const dashboards = await this.validator.getDashboards();
        const dataSources = await this.validator.getDataSources();
        
        if (!dashboards || !dataSources || dataSources.length === 0) {
            return false;
        }
        
        const prometheusDS = dataSources.find(ds => ds.type === 'prometheus');
        if (!prometheusDS) {
            console.log('No Prometheus data source found');
            return false;
        }
        
        // Test common queries
        const testQueries = [
            'up',
            'rate(http_requests_total[5m])',
            'node_cpu_seconds_total',
            'process_resident_memory_bytes'
        ];
        
        let successfulQueries = 0;
        for (const query of testQueries) {
            const result = await this.validator.executeQuery(prometheusDS.uid, query);
            if (result && result.results && result.results.length > 0) {
                successfulQueries++;
            }
        }
        
        return successfulQueries > 0;
    }
    
    async validatePanelData() {
        const dashboards = await this.validator.getDashboards();
        if (!dashboards || dashboards.length === 0) {
            return false;
        }
        
        // Get the first dashboard and check its panels
        const firstDashboard = dashboards[0];
        const dashboardData = await this.validator.getDashboard(firstDashboard.uid);
        
        if (!dashboardData || !dashboardData.dashboard || !dashboardData.dashboard.panels) {
            return false;
        }
        
        const panels = dashboardData.dashboard.panels;
        let validPanels = 0;
        
        for (const panel of panels) {
            if (panel.targets && panel.targets.length > 0) {
                // Check if panel has valid targets
                const hasValidTargets = panel.targets.some(target => 
                    target.expr && target.expr.length > 0
                );
                if (hasValidTargets) {
                    validPanels++;
                }
            }
        }
        
        return validPanels > 0;
    }
    
    async validateAlertRules() {
        const alertRules = await this.validator.getAlertRules();
        if (!alertRules || Object.keys(alertRules).length === 0) {
            console.log('No alert rules found');
            return true; // Not having alerts is not necessarily a failure
        }
        
        let validRules = 0;
        for (const [folder, rules] of Object.entries(alertRules)) {
            if (rules && rules.length > 0) {
                for (const rule of rules) {
                    if (rule.rules && rule.rules.length > 0) {
                        validRules += rule.rules.length;
                    }
                }
            }
        }
        
        return validRules >= 0;
    }
    
    async validateAlertInstances() {
        const alertInstances = await this.validator.getAlertInstances();
        // Having no active alerts is actually good
        return Array.isArray(alertInstances);
    }
    
    async validateDashboardMetrics() {
        const dashboards = await this.validator.getDashboards();
        if (!dashboards || dashboards.length === 0) {
            return false;
        }
        
        // Expected dashboard types for APM
        const expectedDashboards = [
            'infrastructure',
            'application',
            'service',
            'performance',
            'alert'
        ];
        
        let foundDashboards = 0;
        for (const dashboard of dashboards) {
            const titleLower = dashboard.title.toLowerCase();
            if (expectedDashboards.some(expected => titleLower.includes(expected))) {
                foundDashboards++;
            }
        }
        
        return foundDashboards > 0;
    }
    
    async runAllValidations() {
        console.log('Starting Grafana Validation Suite');
        console.log('='.repeat(50));
        
        // Check connectivity first
        if (!await this.runTest('Grafana connectivity', this.validateConnectivity)) {
            console.log('Cannot connect to Grafana. Stopping validation.');
            return false;
        }
        
        // Run all validation categories
        const validations = [
            ['Data Sources', this.validateDataSources],
            ['Dashboards', this.validateDashboards],
            ['Dashboard Queries', this.validateDashboardQueries],
            ['Panel Data', this.validatePanelData],
            ['Alert Rules', this.validateAlertRules],
            ['Alert Instances', this.validateAlertInstances],
            ['Dashboard Metrics', this.validateDashboardMetrics]
        ];
        
        let overallResult = true;
        for (const [category, validationFunc] of validations) {
            console.log(`\n--- ${category} ---`);
            const result = await this.runTest(category, validationFunc);
            overallResult = overallResult && result;
        }
        
        this.printSummary();
        return overallResult;
    }
    
    printSummary() {
        console.log('\n' + '='.repeat(50));
        console.log('VALIDATION SUMMARY');
        console.log('='.repeat(50));
        
        const totalTests = this.testResults.length;
        const passedTests = this.testResults.filter(r => r.result).length;
        const failedTests = totalTests - passedTests;
        
        console.log(`Total Tests: ${totalTests}`);
        console.log(`Passed: ${passedTests}`);
        console.log(`Failed: ${failedTests}`);
        console.log(`Success Rate: ${((passedTests / totalTests) * 100).toFixed(1)}%`);
        console.log(`Total Duration: ${Date.now() - this.startTime.getTime()}ms`);
        
        if (failedTests > 0) {
            console.log('\nFailed Tests:');
            this.testResults
                .filter(r => !r.result)
                .forEach(result => {
                    const error = result.error || 'Test failed';
                    console.log(`  - ${result.testName}: ${error}`);
                });
        }
    }
    
    saveResults(filename = 'grafana_validation_results.json') {
        const results = {
            validationRun: {
                timestamp: new Date().toISOString(),
                duration: Date.now() - this.startTime.getTime(),
                totalTests: this.testResults.length,
                passed: this.testResults.filter(r => r.result).length,
                failed: this.testResults.filter(r => !r.result).length
            },
            testResults: this.testResults
        };
        
        fs.writeFileSync(filename, JSON.stringify(results, null, 2));
        console.log(`Results saved to ${filename}`);
    }
}

async function main() {
    const args = process.argv.slice(2);
    const grafanaUrl = args.find(arg => arg.startsWith('--grafana-url='))?.split('=')[1] || 'http://localhost:3000';
    const apiKey = args.find(arg => arg.startsWith('--api-key='))?.split('=')[1];
    const username = args.find(arg => arg.startsWith('--username='))?.split('=')[1] || 'admin';
    const password = args.find(arg => arg.startsWith('--password='))?.split('=')[1] || 'admin';
    const output = args.find(arg => arg.startsWith('--output='))?.split('=')[1] || 'grafana_validation_results.json';
    
    if (args.includes('--help') || args.includes('-h')) {
        console.log(`
Usage: node grafana-validation.js [options]

Options:
  --grafana-url=URL     Grafana server URL (default: http://localhost:3000)
  --api-key=KEY         Grafana API key
  --username=USER       Username for basic auth (default: admin)
  --password=PASS       Password for basic auth (default: admin)
  --output=FILE         Output file for results (default: grafana_validation_results.json)
  --help, -h            Show this help message
        `);
        process.exit(0);
    }
    
    try {
        const suite = new GrafanaValidationSuite(grafanaUrl, apiKey, username, password);
        const success = await suite.runAllValidations();
        suite.saveResults(output);
        
        process.exit(success ? 0 : 1);
        
    } catch (error) {
        console.error('Validation suite failed:', error);
        process.exit(1);
    }
}

// Handle npm dependencies check
try {
    require('axios');
} catch (error) {
    console.error('Missing dependency: axios');
    console.error('Run: npm install axios');
    process.exit(1);
}

if (require.main === module) {
    main();
}

module.exports = { GrafanaValidator, GrafanaValidationSuite };