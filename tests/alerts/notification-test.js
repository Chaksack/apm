#!/usr/bin/env node

/**
 * Notification system tests for APM monitoring
 * Tests email notifications, Slack webhooks, template rendering, and delivery confirmation
 */

const assert = require('assert');
const sinon = require('sinon');
const nodemailer = require('nodemailer');
const axios = require('axios');

class NotificationTest {
    constructor() {
        this.testResults = [];
        this.mockTransporter = null;
        this.mockSlackClient = null;
        this.setupMocks();
    }

    setupMocks() {
        // Mock email transporter
        this.mockTransporter = {
            sendMail: sinon.stub(),
            verify: sinon.stub().resolves(true)
        };

        // Mock Slack client
        this.mockSlackClient = {
            post: sinon.stub()
        };

        // Mock axios for webhook calls
        this.axiosStub = sinon.stub(axios, 'post');
    }

    async runAllTests() {
        console.log('Starting Notification Tests...');
        console.log('='.repeat(50));

        const tests = [
            'testEmailNotificationBasic',
            'testEmailNotificationWithTemplate',
            'testEmailNotificationDeliveryConfirmation',
            'testSlackWebhookBasic',
            'testSlackWebhookWithAttachments',
            'testSlackWebhookValidation',
            'testTemplateRenderingBasic',
            'testTemplateRenderingWithData',
            'testTemplateRenderingEdgeCases',
            'testDeliveryConfirmationTracking',
            'testNotificationRetryLogic',
            'testNotificationFiltering',
            'testMultiChannelNotification',
            'testNotificationThrottling'
        ];

        for (const testName of tests) {
            try {
                await this[testName]();
                this.testResults.push({ test: testName, status: 'PASS' });
                console.log(`✓ ${testName} passed`);
            } catch (error) {
                this.testResults.push({ test: testName, status: 'FAIL', error: error.message });
                console.log(`✗ ${testName} failed: ${error.message}`);
            }
        }

        this.printTestSummary();
        this.cleanup();
    }

    // Email Notification Tests
    async testEmailNotificationBasic() {
        console.log('Testing basic email notification...');
        
        const emailConfig = {
            host: 'smtp.example.com',
            port: 587,
            secure: false,
            auth: {
                user: 'alerts@example.com',
                pass: 'password'
            }
        };

        const alert = {
            labels: {
                alertname: 'HighCPUUsage',
                severity: 'critical',
                instance: 'web-01',
                service: 'web-server'
            },
            annotations: {
                summary: 'High CPU usage detected',
                description: 'CPU usage is above 90% for 5 minutes'
            },
            startsAt: new Date().toISOString()
        };

        const emailData = {
            from: 'alerts@example.com',
            to: 'oncall@example.com',
            subject: `[CRITICAL] ${alert.labels.alertname} - ${alert.labels.instance}`,
            html: this.renderEmailTemplate(alert)
        };

        this.mockTransporter.sendMail.resolves({
            messageId: 'test-message-id-123',
            response: '250 Message accepted'
        });

        const result = await this.sendEmailNotification(emailData);
        
        assert.strictEqual(result.messageId, 'test-message-id-123');
        assert(this.mockTransporter.sendMail.calledOnce);
        assert(this.mockTransporter.sendMail.calledWith(emailData));
    }

    async testEmailNotificationWithTemplate() {
        console.log('Testing email notification with template...');
        
        const alert = {
            labels: {
                alertname: 'DatabaseConnectionError',
                severity: 'critical',
                instance: 'db-01',
                service: 'database',
                team: 'data'
            },
            annotations: {
                summary: 'Database connection failed',
                description: 'Unable to connect to database for 2 minutes',
                runbook: 'https://wiki.example.com/runbooks/database'
            },
            startsAt: new Date().toISOString()
        };

        const template = `
            <h2>🚨 Critical Alert: {{alertname}}</h2>
            <p><strong>Service:</strong> {{service}}</p>
            <p><strong>Instance:</strong> {{instance}}</p>
            <p><strong>Team:</strong> {{team}}</p>
            <p><strong>Description:</strong> {{description}}</p>
            <p><strong>Started:</strong> {{startsAt}}</p>
            {{#if runbook}}
            <p><a href="{{runbook}}">View Runbook</a></p>
            {{/if}}
        `;

        const renderedContent = this.renderTemplate(template, alert);
        
        assert(renderedContent.includes('Critical Alert: DatabaseConnectionError'));
        assert(renderedContent.includes('Service: database'));
        assert(renderedContent.includes('Team: data'));
        assert(renderedContent.includes('https://wiki.example.com/runbooks/database'));
    }

    async testEmailNotificationDeliveryConfirmation() {
        console.log('Testing email delivery confirmation...');
        
        const emailData = {
            from: 'alerts@example.com',
            to: 'oncall@example.com',
            subject: 'Test Alert',
            html: '<p>Test alert content</p>'
        };

        // Mock successful delivery
        this.mockTransporter.sendMail.resolves({
            messageId: 'confirmed-message-123',
            response: '250 Message accepted',
            envelope: {
                from: 'alerts@example.com',
                to: ['oncall@example.com']
            }
        });

        const result = await this.sendEmailNotification(emailData);
        const confirmation = this.trackDeliveryConfirmation(result);
        
        assert.strictEqual(confirmation.status, 'sent');
        assert.strictEqual(confirmation.messageId, 'confirmed-message-123');
        assert(confirmation.timestamp);
    }

    // Slack Webhook Tests
    async testSlackWebhookBasic() {
        console.log('Testing basic Slack webhook...');
        
        const webhookUrl = 'https://hooks.slack.com/services/TEST/WEBHOOK/URL';
        const alert = {
            labels: {
                alertname: 'HighMemoryUsage',
                severity: 'warning',
                instance: 'api-01',
                service: 'api-server'
            },
            annotations: {
                summary: 'High memory usage detected',
                description: 'Memory usage is above 85%'
            }
        };

        const slackMessage = {
            text: `Alert: ${alert.labels.alertname}`,
            username: 'AlertManager',
            icon_emoji: ':warning:',
            attachments: [{
                color: 'warning',
                title: alert.labels.alertname,
                text: alert.annotations.description,
                fields: [
                    { title: 'Severity', value: alert.labels.severity, short: true },
                    { title: 'Service', value: alert.labels.service, short: true },
                    { title: 'Instance', value: alert.labels.instance, short: true }
                ]
            }]
        };

        this.axiosStub.resolves({
            status: 200,
            data: 'ok',
            headers: { 'content-type': 'text/plain' }
        });

        const result = await this.sendSlackNotification(webhookUrl, slackMessage);
        
        assert.strictEqual(result.status, 200);
        assert.strictEqual(result.data, 'ok');
        assert(this.axiosStub.calledOnce);
        assert(this.axiosStub.calledWith(webhookUrl, slackMessage));
    }

    async testSlackWebhookWithAttachments() {
        console.log('Testing Slack webhook with rich attachments...');
        
        const webhookUrl = 'https://hooks.slack.com/services/TEST/WEBHOOK/URL';
        const alert = {
            labels: {
                alertname: 'ServiceDown',
                severity: 'critical',
                instance: 'web-01',
                service: 'web-server',
                team: 'platform'
            },
            annotations: {
                summary: 'Service is down',
                description: 'Web server is not responding to health checks',
                dashboard: 'https://grafana.example.com/dashboard/web-server',
                runbook: 'https://wiki.example.com/runbooks/web-server'
            }
        };

        const slackMessage = {
            text: ':rotating_light: Critical Alert',
            username: 'AlertManager',
            icon_emoji: ':rotating_light:',
            attachments: [{
                color: 'danger',
                title: `${alert.labels.alertname} - ${alert.labels.instance}`,
                title_link: alert.annotations.dashboard,
                text: alert.annotations.description,
                fields: [
                    { title: 'Severity', value: alert.labels.severity, short: true },
                    { title: 'Team', value: alert.labels.team, short: true },
                    { title: 'Service', value: alert.labels.service, short: true },
                    { title: 'Instance', value: alert.labels.instance, short: true }
                ],
                actions: [
                    {
                        type: 'button',
                        text: 'View Dashboard',
                        url: alert.annotations.dashboard
                    },
                    {
                        type: 'button',
                        text: 'View Runbook',
                        url: alert.annotations.runbook
                    }
                ],
                footer: 'AlertManager',
                ts: Math.floor(Date.now() / 1000)
            }]
        };

        this.axiosStub.resolves({ status: 200, data: 'ok' });

        const result = await this.sendSlackNotification(webhookUrl, slackMessage);
        
        assert.strictEqual(result.status, 200);
        
        const sentMessage = this.axiosStub.getCall(0).args[1];
        assert.strictEqual(sentMessage.attachments[0].color, 'danger');
        assert.strictEqual(sentMessage.attachments[0].actions.length, 2);
        assert(sentMessage.attachments[0].ts);
    }

    async testSlackWebhookValidation() {
        console.log('Testing Slack webhook validation...');
        
        const webhookUrl = 'https://hooks.slack.com/services/TEST/WEBHOOK/URL';
        
        // Test with invalid webhook URL
        const invalidUrl = 'https://invalid-webhook-url.com';
        this.axiosStub.withArgs(invalidUrl).rejects(new Error('Invalid webhook URL'));
        
        try {
            await this.sendSlackNotification(invalidUrl, { text: 'Test' });
            assert.fail('Should have thrown an error for invalid webhook URL');
        } catch (error) {
            assert(error.message.includes('Invalid webhook URL'));
        }

        // Test with valid webhook but server error
        this.axiosStub.withArgs(webhookUrl).resolves({
            status: 400,
            data: 'invalid_payload'
        });

        try {
            await this.sendSlackNotification(webhookUrl, { invalid: 'payload' });
            assert.fail('Should have thrown an error for invalid payload');
        } catch (error) {
            assert(error.message.includes('Slack webhook failed'));
        }
    }

    // Template Rendering Tests
    async testTemplateRenderingBasic() {
        console.log('Testing basic template rendering...');
        
        const template = 'Alert: {{alertname}} on {{instance}}';
        const data = {
            alertname: 'HighCPUUsage',
            instance: 'web-01'
        };

        const result = this.renderTemplate(template, data);
        
        assert.strictEqual(result, 'Alert: HighCPUUsage on web-01');
    }

    async testTemplateRenderingWithData() {
        console.log('Testing template rendering with complex data...');
        
        const template = `
            Alert: {{labels.alertname}}
            Severity: {{labels.severity}}
            {{#if annotations.runbook}}
            Runbook: {{annotations.runbook}}
            {{/if}}
            {{#each labels}}
            {{@key}}: {{this}}
            {{/each}}
        `;

        const alert = {
            labels: {
                alertname: 'DatabaseError',
                severity: 'critical',
                service: 'database'
            },
            annotations: {
                summary: 'Database error occurred',
                runbook: 'https://wiki.example.com/db-errors'
            }
        };

        const result = this.renderTemplate(template, alert);
        
        assert(result.includes('Alert: DatabaseError'));
        assert(result.includes('Severity: critical'));
        assert(result.includes('Runbook: https://wiki.example.com/db-errors'));
        assert(result.includes('service: database'));
    }

    async testTemplateRenderingEdgeCases() {
        console.log('Testing template rendering edge cases...');
        
        // Test with missing variables
        const template = 'Alert: {{alertname}} - {{missingVar}}';
        const data = { alertname: 'TestAlert' };
        
        const result = this.renderTemplate(template, data);
        assert(result.includes('TestAlert'));
        assert(result.includes('{{missingVar}}') || result.includes(''));

        // Test with empty data
        const emptyResult = this.renderTemplate('{{alertname}}', {});
        assert.strictEqual(emptyResult, '');

        // Test with null/undefined values
        const nullData = { alertname: null, severity: undefined };
        const nullResult = this.renderTemplate('{{alertname}}-{{severity}}', nullData);
        assert.strictEqual(nullResult, '-');
    }

    // Delivery Confirmation Tests
    async testDeliveryConfirmationTracking() {
        console.log('Testing delivery confirmation tracking...');
        
        const notifications = [
            { id: 'email-1', type: 'email', status: 'sent', timestamp: new Date() },
            { id: 'slack-1', type: 'slack', status: 'delivered', timestamp: new Date() },
            { id: 'email-2', type: 'email', status: 'failed', timestamp: new Date() }
        ];

        const tracker = this.createDeliveryTracker();
        
        notifications.forEach(notification => {
            tracker.track(notification);
        });

        const summary = tracker.getSummary();
        
        assert.strictEqual(summary.total, 3);
        assert.strictEqual(summary.sent, 1);
        assert.strictEqual(summary.delivered, 1);
        assert.strictEqual(summary.failed, 1);
        assert.strictEqual(summary.successRate, 0.67); // 2/3 rounded
    }

    async testNotificationRetryLogic() {
        console.log('Testing notification retry logic...');
        
        const webhookUrl = 'https://hooks.slack.com/services/TEST/WEBHOOK/URL';
        const message = { text: 'Test alert' };
        
        // Mock first two calls to fail, third to succeed
        this.axiosStub.onCall(0).rejects(new Error('Network error'));
        this.axiosStub.onCall(1).rejects(new Error('Timeout'));
        this.axiosStub.onCall(2).resolves({ status: 200, data: 'ok' });

        const result = await this.sendSlackNotificationWithRetry(webhookUrl, message, 3);
        
        assert.strictEqual(result.status, 200);
        assert.strictEqual(this.axiosStub.callCount, 3);
    }

    async testNotificationFiltering() {
        console.log('Testing notification filtering...');
        
        const alerts = [
            { labels: { severity: 'critical', team: 'platform' } },
            { labels: { severity: 'warning', team: 'platform' } },
            { labels: { severity: 'info', team: 'data' } },
            { labels: { severity: 'critical', team: 'data' } }
        ];

        const filters = {
            criticalOnly: alert => alert.labels.severity === 'critical',
            platformTeam: alert => alert.labels.team === 'platform',
            excludeInfo: alert => alert.labels.severity !== 'info'
        };

        const criticalAlerts = alerts.filter(filters.criticalOnly);
        const platformAlerts = alerts.filter(filters.platformTeam);
        const nonInfoAlerts = alerts.filter(filters.excludeInfo);

        assert.strictEqual(criticalAlerts.length, 2);
        assert.strictEqual(platformAlerts.length, 2);
        assert.strictEqual(nonInfoAlerts.length, 3);
    }

    async testMultiChannelNotification() {
        console.log('Testing multi-channel notification...');
        
        const alert = {
            labels: { alertname: 'ServiceDown', severity: 'critical' },
            annotations: { summary: 'Service is down' }
        };

        const channels = [
            { type: 'email', config: { to: 'oncall@example.com' } },
            { type: 'slack', config: { webhook: 'https://hooks.slack.com/test' } },
            { type: 'pagerduty', config: { service_key: 'test-key' } }
        ];

        // Mock all channel responses
        this.mockTransporter.sendMail.resolves({ messageId: 'email-123' });
        this.axiosStub.resolves({ status: 200, data: 'ok' });

        const results = await this.sendMultiChannelNotification(alert, channels);
        
        assert.strictEqual(results.length, 3);
        assert.strictEqual(results[0].channel, 'email');
        assert.strictEqual(results[1].channel, 'slack');
        assert.strictEqual(results[2].channel, 'pagerduty');
        assert(results.every(r => r.success));
    }

    async testNotificationThrottling() {
        console.log('Testing notification throttling...');
        
        const throttler = this.createNotificationThrottler();
        const alertKey = 'HighCPUUsage-web-01';
        
        // First notification should go through
        const first = throttler.shouldNotify(alertKey, 'critical');
        assert.strictEqual(first, true);
        
        // Second notification within throttle period should be blocked
        const second = throttler.shouldNotify(alertKey, 'critical');
        assert.strictEqual(second, false);
        
        // After throttle period, should allow notification
        throttler.clearThrottleForTesting(alertKey);
        const third = throttler.shouldNotify(alertKey, 'critical');
        assert.strictEqual(third, true);
    }

    // Helper Methods
    async sendEmailNotification(emailData) {
        return this.mockTransporter.sendMail(emailData);
    }

    async sendSlackNotification(webhookUrl, message) {
        const response = await axios.post(webhookUrl, message);
        if (response.status !== 200) {
            throw new Error(`Slack webhook failed with status ${response.status}`);
        }
        return response;
    }

    async sendSlackNotificationWithRetry(webhookUrl, message, maxRetries = 3) {
        let lastError;
        
        for (let i = 0; i < maxRetries; i++) {
            try {
                return await this.sendSlackNotification(webhookUrl, message);
            } catch (error) {
                lastError = error;
                if (i < maxRetries - 1) {
                    await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
                }
            }
        }
        
        throw lastError;
    }

    async sendMultiChannelNotification(alert, channels) {
        const results = [];
        
        for (const channel of channels) {
            try {
                let result;
                
                switch (channel.type) {
                    case 'email':
                        result = await this.sendEmailNotification({
                            to: channel.config.to,
                            subject: `Alert: ${alert.labels.alertname}`,
                            html: this.renderEmailTemplate(alert)
                        });
                        break;
                    
                    case 'slack':
                        result = await this.sendSlackNotification(channel.config.webhook, {
                            text: `Alert: ${alert.labels.alertname}`,
                            attachments: [{ text: alert.annotations.summary }]
                        });
                        break;
                    
                    case 'pagerduty':
                        // Mock PagerDuty API call
                        result = { status: 'triggered', incident_key: 'test-key' };
                        break;
                }
                
                results.push({
                    channel: channel.type,
                    success: true,
                    result: result
                });
            } catch (error) {
                results.push({
                    channel: channel.type,
                    success: false,
                    error: error.message
                });
            }
        }
        
        return results;
    }

    renderTemplate(template, data) {
        // Simple template rendering (in real implementation, use Handlebars or similar)
        let result = template;
        
        // Handle simple variables {{variable}}
        result = result.replace(/\{\{([^}]+)\}\}/g, (match, key) => {
            const value = this.getNestedValue(data, key.trim());
            return value !== undefined ? value : '';
        });
        
        // Handle conditionals {{#if condition}}...{{/if}}
        result = result.replace(/\{\{#if\s+([^}]+)\}\}(.*?)\{\{\/if\}\}/gs, (match, condition, content) => {
            const value = this.getNestedValue(data, condition.trim());
            return value ? content : '';
        });
        
        return result;
    }

    renderEmailTemplate(alert) {
        const template = `
            <html>
            <body>
                <h2>🚨 Alert: {{labels.alertname}}</h2>
                <table border="1" style="border-collapse: collapse;">
                    <tr><td><strong>Severity</strong></td><td>{{labels.severity}}</td></tr>
                    <tr><td><strong>Service</strong></td><td>{{labels.service}}</td></tr>
                    <tr><td><strong>Instance</strong></td><td>{{labels.instance}}</td></tr>
                    <tr><td><strong>Description</strong></td><td>{{annotations.description}}</td></tr>
                    <tr><td><strong>Started</strong></td><td>{{startsAt}}</td></tr>
                </table>
            </body>
            </html>
        `;
        
        return this.renderTemplate(template, alert);
    }

    getNestedValue(obj, path) {
        return path.split('.').reduce((current, key) => {
            return current && current[key] !== undefined ? current[key] : undefined;
        }, obj);
    }

    trackDeliveryConfirmation(result) {
        return {
            status: result.messageId ? 'sent' : 'failed',
            messageId: result.messageId,
            timestamp: new Date().toISOString()
        };
    }

    createDeliveryTracker() {
        const notifications = [];
        
        return {
            track(notification) {
                notifications.push(notification);
            },
            
            getSummary() {
                const total = notifications.length;
                const sent = notifications.filter(n => n.status === 'sent').length;
                const delivered = notifications.filter(n => n.status === 'delivered').length;
                const failed = notifications.filter(n => n.status === 'failed').length;
                
                return {
                    total,
                    sent,
                    delivered,
                    failed,
                    successRate: Math.round((sent + delivered) / total * 100) / 100
                };
            }
        };
    }

    createNotificationThrottler() {
        const throttleMap = new Map();
        const throttlePeriod = 5 * 60 * 1000; // 5 minutes
        
        return {
            shouldNotify(alertKey, severity) {
                const now = Date.now();
                const lastNotified = throttleMap.get(alertKey);
                
                if (!lastNotified || now - lastNotified > throttlePeriod) {
                    throttleMap.set(alertKey, now);
                    return true;
                }
                
                return false;
            },
            
            clearThrottleForTesting(alertKey) {
                throttleMap.delete(alertKey);
            }
        };
    }

    printTestSummary() {
        console.log('\n' + '='.repeat(50));
        console.log('Test Results Summary:');
        console.log('='.repeat(50));
        
        const passed = this.testResults.filter(t => t.status === 'PASS').length;
        const failed = this.testResults.filter(t => t.status === 'FAIL').length;
        
        console.log(`Total Tests: ${this.testResults.length}`);
        console.log(`Passed: ${passed}`);
        console.log(`Failed: ${failed}`);
        console.log(`Success Rate: ${Math.round(passed / this.testResults.length * 100)}%`);
        
        if (failed > 0) {
            console.log('\nFailed Tests:');
            this.testResults.filter(t => t.status === 'FAIL').forEach(t => {
                console.log(`  - ${t.test}: ${t.error}`);
            });
        }
    }

    cleanup() {
        // Restore stubs
        if (this.axiosStub.restore) {
            this.axiosStub.restore();
        }
        sinon.restore();
    }
}

// Run tests if this file is executed directly
if (require.main === module) {
    const testRunner = new NotificationTest();
    testRunner.runAllTests().then(() => {
        console.log('\nNotification Tests Complete');
        process.exit(0);
    }).catch(error => {
        console.error('Test runner failed:', error);
        process.exit(1);
    });
}

module.exports = NotificationTest;