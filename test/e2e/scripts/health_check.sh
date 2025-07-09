#!/bin/bash

# Health check script for APM services

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Services to check
declare -A SERVICES=(
    ["Prometheus"]="http://localhost:9090/-/ready"
    ["Grafana"]="http://localhost:3000/api/health"
    ["Loki"]="http://localhost:3100/ready"
    ["Jaeger"]="http://localhost:16686"
    ["AlertManager"]="http://localhost:9093/-/ready"
)

# Function to check service health
check_service() {
    local name=$1
    local url=$2
    local max_attempts=30
    local attempt=1
    
    echo -n "Checking ${name}..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "${url}" > /dev/null 2>&1; then
            echo -e " ${GREEN}✓${NC}"
            return 0
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            echo -e " ${RED}✗${NC}"
            return 1
        fi
        
        sleep 2
        ((attempt++))
    done
}

# Main health check
main() {
    echo "Performing health checks on APM services..."
    echo "=========================================="
    
    local failed=0
    
    for service in "${!SERVICES[@]}"; do
        if ! check_service "$service" "${SERVICES[$service]}"; then
            ((failed++))
        fi
    done
    
    echo "=========================================="
    
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}All services are healthy!${NC}"
        exit 0
    else
        echo -e "${RED}${failed} service(s) failed health check${NC}"
        exit 1
    fi
}

main "$@"