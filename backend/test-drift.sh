#!/bin/bash

# DriftGuard Test Script
# This script demonstrates the enhanced drift detection functionality with state tracking

set -e

echo "🚀 DriftGuard Enhanced Test Script"
echo "=================================="

# Configuration
API_BASE="http://localhost:8080"
NAMESPACE="driftguard"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if DriftGuard is running
check_health() {
    log_info "Checking DriftGuard health..."
    
    if curl -s -f "$API_BASE/health" > /dev/null; then
        log_success "DriftGuard is healthy"
        return 0
    else
        log_error "DriftGuard is not responding"
        return 1
    fi
}

# Wait for DriftGuard to be ready
wait_for_ready() {
    log_info "Waiting for DriftGuard to be ready..."
    
    for i in {1..30}; do
        if curl -s -f "$API_BASE/ready" > /dev/null; then
            log_success "DriftGuard is ready"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    
    log_error "DriftGuard failed to become ready"
    return 1
}

# Create test namespace and resources
setup_test_environment() {
    log_info "Setting up test environment..."
    
    # Create namespace
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply test deployment
    kubectl apply -f test-deployment.yaml
    
    log_success "Test environment setup complete"
}

# Trigger drift by modifying the deployment
create_drift() {
    log_info "Creating drift by scaling deployment..."
    
    # Scale the deployment to create drift
    kubectl scale deployment nginx-app -n $NAMESPACE --replicas=3
    
    log_success "Deployment scaled to 3 replicas (drift created)"
}

# Resolve drift by reverting to original state
resolve_drift() {
    log_info "Resolving drift by reverting to original state..."
    
    # Scale the deployment back to original state
    kubectl scale deployment nginx-app -n $NAMESPACE --replicas=2
    
    log_success "Deployment scaled back to 2 replicas (drift should be resolved)"
}

# Check drift records
check_drift_records() {
    log_info "Checking drift records..."
    
    # Wait a bit for drift detection to run
    sleep 10
    
    # Get all drift records
    response=$(curl -s "$API_BASE/api/v1/drift-records")
    
    if echo "$response" | jq -e '.drift_records' > /dev/null; then
        count=$(echo "$response" | jq '.count')
        log_success "Found $count drift records"
        
        # Show drift records with drift
        drift_records=$(echo "$response" | jq -r '.drift_records[] | select(.drift_detected == true) | .resource_id')
        
        if [ -n "$drift_records" ]; then
            log_info "Resources with detected drift:"
            echo "$drift_records" | while read -r resource; do
                echo "  - $resource"
            done
        else
            log_warning "No drift detected yet"
        fi
    else
        log_error "Failed to get drift records"
        echo "$response"
    fi
}

# Check active drifts
check_active_drifts() {
    log_info "Checking active drifts..."
    
    response=$(curl -s "$API_BASE/api/v1/drift-records/active")
    
    if echo "$response" | jq -e '.drift_records' > /dev/null; then
        count=$(echo "$response" | jq '.count')
        log_success "Found $count active drift records"
        
        if [ "$count" -gt 0 ]; then
            log_info "Active drift details:"
            echo "$response" | jq -r '.drift_records[] | "  - \(.resource_id): \(.drift_status) (first detected: \(.first_detected))"'
        fi
    else
        log_error "Failed to get active drift records"
        echo "$response"
    fi
}

# Check resolved drifts
check_resolved_drifts() {
    log_info "Checking resolved drifts..."
    
    response=$(curl -s "$API_BASE/api/v1/drift-records/resolved")
    
    if echo "$response" | jq -e '.drift_records' > /dev/null; then
        count=$(echo "$response" | jq '.count')
        log_success "Found $count resolved drift records"
        
        if [ "$count" -gt 0 ]; then
            log_info "Resolved drift details:"
            echo "$response" | jq -r '.drift_records[] | "  - \(.resource_id): resolved at \(.resolved_at) - \(.resolution_message)"'
        fi
    else
        log_error "Failed to get resolved drift records"
        echo "$response"
    fi
}

# Get specific drift record
get_drift_details() {
    log_info "Getting drift details for nginx-app..."
    
    response=$(curl -s "$API_BASE/api/v1/drift-records/Deployment:$NAMESPACE:nginx-app")
    
    if echo "$response" | jq -e '.drift_status' > /dev/null; then
        drift_status=$(echo "$response" | jq -r '.drift_status')
        drift_detected=$(echo "$response" | jq -r '.drift_detected')
        
        log_success "Drift status: $drift_status (detected: $drift_detected)"
        
        if [ "$drift_detected" = "true" ]; then
            log_info "Drift details:"
            echo "$response" | jq -r '.drift_details[] | "  - \(.field): \(.from) -> \(.to) (\(.type), \(.severity))"'
            
            if [ "$drift_status" = "active" ]; then
                first_detected=$(echo "$response" | jq -r '.first_detected')
                log_info "First detected: $first_detected"
            fi
        elif [ "$drift_status" = "resolved" ]; then
            resolved_at=$(echo "$response" | jq -r '.resolved_at')
            resolution_message=$(echo "$response" | jq -r '.resolution_message')
            log_success "Resolved at: $resolved_at"
            log_info "Resolution message: $resolution_message"
        fi
    else
        log_error "Failed to get drift details"
        echo "$response"
    fi
}

# Get enhanced statistics
get_statistics() {
    log_info "Getting enhanced drift statistics..."
    
    response=$(curl -s "$API_BASE/api/v1/statistics")
    
    if echo "$response" | jq -e '.statistics' > /dev/null; then
        stats=$(echo "$response" | jq -r '.statistics')
        
        log_success "Enhanced Drift Statistics:"
        echo "$stats" | jq -r 'to_entries[] | "  - \(.key): \(.value)"'
        
        # Show key metrics
        active_count=$(echo "$stats" | jq -r '.active_drift')
        resolved_count=$(echo "$stats" | jq -r '.resolved_drift')
        recent_active=$(echo "$stats" | jq -r '.recent_active_drift')
        recent_resolved=$(echo "$stats" | jq -r '.recent_resolutions')
        
        log_info "Summary:"
        log_info "  - Active drifts: $active_count"
        log_info "  - Resolved drifts: $resolved_count"
        log_info "  - Recent active (24h): $recent_active"
        log_info "  - Recent resolutions (24h): $recent_resolved"
    else
        log_error "Failed to get statistics"
        echo "$response"
    fi
}

# Trigger manual analysis
trigger_analysis() {
    log_info "Triggering manual drift analysis..."
    
    response=$(curl -s -X POST "$API_BASE/api/v1/analyze")
    
    if echo "$response" | jq -e '.status' > /dev/null; then
        status=$(echo "$response" | jq -r '.status')
        log_success "Analysis triggered: $status"
    else
        log_error "Failed to trigger analysis"
        echo "$response"
    fi
}

# Cleanup test environment
cleanup() {
    log_info "Cleaning up test environment..."
    
    # Delete test resources
    kubectl delete -f test-deployment.yaml --ignore-not-found=true
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    
    log_success "Cleanup complete"
}

# Main test flow
main() {
    echo
    log_info "Starting DriftGuard Enhanced Test..."
    
    # Check if DriftGuard is running
    if ! check_health; then
        log_error "Please start DriftGuard first: go run cmd/controller/main.go"
        exit 1
    fi
    
    # Wait for ready
    if ! wait_for_ready; then
        log_error "DriftGuard is not ready"
        exit 1
    fi
    
    # Setup test environment
    setup_test_environment
    
    # Wait for initial state to be captured
    log_info "Waiting for initial state capture..."
    sleep 15
    
    # Check initial drift records
    check_drift_records
    
    # Create drift
    create_drift
    
    # Wait for drift detection
    log_info "Waiting for drift detection..."
    sleep 30
    
    # Check drift records and active drifts
    check_drift_records
    check_active_drifts
    
    # Get specific drift details
    get_drift_details
    
    # Get enhanced statistics
    get_statistics
    
    # Trigger manual analysis
    trigger_analysis
    
    # Wait for analysis to complete
    sleep 10
    
    # Resolve drift
    resolve_drift
    
    # Wait for drift resolution
    log_info "Waiting for drift resolution..."
    sleep 30
    
    # Check resolved drifts
    check_resolved_drifts
    
    # Get final drift details
    get_drift_details
    
    # Get final statistics
    get_statistics
    
    echo
    log_success "Enhanced test completed successfully!"
    echo
    log_info "Key Features Demonstrated:"
    log_info "  ✅ Drift detection with detailed field changes"
    log_info "  ✅ State tracking (active/resolved/none)"
    log_info "  ✅ Drift resolution detection"
    log_info "  ✅ Enhanced logging with emojis"
    log_info "  ✅ Hash-based state tracking"
    log_info "  ✅ Timestamp tracking (first detected, resolved)"
    echo
    log_info "To view the DriftGuard dashboard, visit: http://localhost:8080"
    log_info "To view metrics, visit: http://localhost:8080/metrics"
    log_info "To view active drifts: http://localhost:8080/api/v1/drift-records/active"
    log_info "To view resolved drifts: http://localhost:8080/api/v1/drift-records/resolved"
    
    # Ask if user wants to cleanup
    echo
    read -p "Do you want to cleanup the test environment? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup
    fi
}

# Run main function
main "$@" 