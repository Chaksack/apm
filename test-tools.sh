#!/bin/bash

# Test the /tools endpoint
echo "Testing /tools endpoint..."
curl -v http://localhost:8080/tools/

# Test the /tools/:tool endpoint
echo -e "\n\nTesting /tools/prometheus endpoint..."
curl -v http://localhost:8080/tools/prometheus