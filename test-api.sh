#!/bin/bash

# Test the /api/v1/status endpoint
echo "Testing /api/v1/status endpoint..."
curl -v http://localhost:8081/api/v1/status

# Test the /health endpoint for comparison
echo -e "\n\nTesting /health endpoint..."
curl -v http://localhost:8081/health
