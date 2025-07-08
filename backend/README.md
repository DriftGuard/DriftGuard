# DriftGuard Backend

DriftGuard is a Kubernetes drift detection system that monitors your cluster resources and compares them against your desired state stored in Git repositories.

## Current Implementation Status

### Implemented Features

- **Configuration Management**: Complete YAML-based configuration system with validation
- **Database Layer**: MongoDB integration with connection pooling, indexing, and metrics
- **Kubernetes Watcher**: Real Kubernetes API integration with selective resource monitoring
- **HTTP Server**: REST API with health checks, metrics, and middleware
- **Metrics & Monitoring**: Prometheus metrics for all components
- **Lifecycle Management**: Graceful startup/shutdown with component health checks
- **Resource Monitoring**: Selective monitoring of configured Kubernetes resources
- **Snapshot Storage**: Efficient snapshot saving with throttling and filtering
- **Health Checks**: Built-in health monitoring for all components

### Partially Implemented Features

- **Controller**: Basic structure in place, needs drift detection logic
- **Git Integration**: Structure defined, needs implementation
- **MCP Client**: Structure defined, needs AI/ML integration
- **Logger**: Basic structure, needs configurable implementation

### TODO Features

- **Drift Detection Logic**: Compare live state with Git desired state
- **Git Repository Integration**: Clone, pull, and compare with Git repos
- **AI/ML Integration**: MCP client for intelligent drift analysis
- **Notification System**: Alerting and notification mechanisms
- **Remediation**: Automatic drift remediation capabilities
- **Advanced Logging**: Configurable logging with correlation IDs

## Features

- **Selective Resource Monitoring**: Only monitor resources you explicitly configure
- **Git Integration**: Compare live state with desired state from Git repositories (structure ready)
- **Drift Detection**: Identify configuration drift in real-time (framework ready)
- **Database Storage**: MongoDB-backed storage for snapshots and drift events
- **REST API**: HTTP API for querying drift information
- **Metrics**: Prometheus metrics for monitoring system health
- **Health Checks**: Built-in health monitoring for all components
- **Efficiency Controls**: Configurable snapshot intervals and resource filtering

## Quick Start

### Prerequisites

- Go 1.24+
- MongoDB (local or remote)
- Kubernetes cluster access
- kubectl configured

### 1. Configure Resources

Edit `configs/config.yaml` to specify which resources to monitor:

```yaml
kubernetes:
  resources: [
    "deployments",     # Application deployments
    "services",        # Network services
    "configmaps",      # Configuration data
    "secrets"          # Sensitive data
  ]
  namespaces: ["your-namespace"]  # Specify namespaces to monitor
  enable_snapshots: true
  snapshot_interval: 5m
```

### 2. Configure Database

Update the database configuration in `configs/config.yaml`:

```yaml
database:
  host: localhost
  port: 27017
  dbname: driftguard
  user: ""  # Add if authentication is required
  password: ""  # Add if authentication is required
```

### 3. Start the Application

```bash
cd backend
go run cmd/controller/main.go
```

### 4. Verify Health

```bash
# Check application health
curl http://localhost:8080/health

# Check readiness
curl http://localhost:8080/ready

# View metrics
curl http://localhost:8080/metrics
```

## Testing the Implementation

### 1. Test Health Endpoints

```bash
# Test health check
curl -v http://localhost:8080/health

# Test readiness probe
curl -v http://localhost:8080/ready

# Expected response: {"status":"healthy","components":{...}}
```

### 2. Test Metrics Endpoint

```bash
# View Prometheus metrics
curl http://localhost:8080/metrics

# Look for metrics like:
# - driftguard_app_uptime_seconds
# - driftguard_k8s_resources_watched
# - driftguard_http_requests_total
```

### 3. Test Kubernetes Resource Monitoring

Create a test deployment to verify resource monitoring:

```bash
# Create a test namespace
kubectl create namespace test-driftguard

# Create a test deployment
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: test-driftguard
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
EOF

# Check logs for resource monitoring
# You should see logs like:
# "Setting up Deployment watcher"
# "Deployment added" or "Deployment updated"
```

### 4. Test Database Operations

```bash
# Connect to MongoDB and check collections
mongosh driftguard

# List collections
show collections

# Check for snapshots
db.configuration_snapshots.find().limit(5)

# Check for drift events
db.drift_events.find().limit(5)
```

### 5. Test Configuration Validation

```bash
# Test with invalid config
cp configs/config.yaml configs/test-config.yaml
# Edit test-config.yaml to remove required fields

# Run with invalid config
go run cmd/controller/main.go -config configs/test-config.yaml
# Should show validation errors
```

### 6. Test Resource Filtering

```bash
# Update config to only monitor specific resources
# Edit configs/config.yaml to only include "deployments"

# Restart the application
# Check logs - should only show deployment watchers
```

## Configuration

### Resource Monitoring

DriftGuard only monitors resources that are explicitly configured. See [Resource Monitoring Configuration](RESOURCE_MONITORING.md) for detailed configuration options.

### Supported Resource Types

- `deployments` - Application deployments
- `services` - Network services  
- `configmaps` - Configuration data
- `secrets` - Sensitive data
- `pods` - Individual pods (high frequency)
- `ingresses` - Ingress rules
- `persistentvolumeclaims` - Storage claims
- `serviceaccounts` - Service accounts
- `networkpolicies` - Network policies

### Configuration Examples

**Minimal Production Setup**:
```yaml
kubernetes:
  resources: ["deployments", "services", "configmaps", "secrets"]
  namespaces: ["production"]
  enable_snapshots: true
  snapshot_interval: 10m
```

**Development Environment**:
```yaml
kubernetes:
  resources: ["deployments", "services", "configmaps", "pods"]
  namespaces: ["development"]
  enable_snapshots: true
  snapshot_interval: 5m
```

**Efficiency-Focused**:
```yaml
kubernetes:
  resources: ["deployments", "services"]
  namespaces: ["production"]
  enable_snapshots: true
  snapshot_interval: 30m
  skip_system_namespaces: true
  skip_frequent_pods: true
```

## API Endpoints

### Health & Monitoring
- `GET /health` - Health check
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### Data Endpoints (Structure Ready)
- `GET /api/v1/snapshots` - List configuration snapshots
- `GET /api/v1/drifts` - List drift events
- `GET /api/v1/environments` - List environments
- `GET /api/v1/statistics` - Get drift statistics

## Architecture

### Implemented Components

- **Configuration**: Complete YAML-based config with validation
- **Database**: MongoDB with connection pooling and indexing
- **Kubernetes Watcher**: Real API integration with selective monitoring
- **HTTP Server**: REST API with middleware and health checks
- **Metrics**: Prometheus metrics for all components
- **Lifecycle Manager**: Graceful startup/shutdown
- **Health Service**: Component health monitoring

### Data Flow (Current)

1. **Configuration Loading**: YAML config loaded and validated
2. **Database Connection**: MongoDB connection established with indexes
3. **Kubernetes Watcher**: Resource watchers started for configured resources
4. **Resource Monitoring**: Live resource changes captured and stored
5. **Snapshot Storage**: Resource states saved to MongoDB with throttling
6. **Health Monitoring**: All components monitored for health
7. **Metrics Collection**: Prometheus metrics exposed via HTTP

### Data Flow (Future - TODO)

1. **Git Integration**: Clone and monitor Git repositories
2. **Drift Detection**: Compare live state with Git desired state
3. **Event Generation**: Create drift events for differences
4. **AI Analysis**: Use MCP for intelligent drift analysis
5. **Notifications**: Send alerts for drift events
6. **Remediation**: Automatic drift correction

## Development

### Prerequisites

- Go 1.24+
- MongoDB
- Kubernetes cluster access

### Building

```bash
go build -o driftguard cmd/controller/main.go
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test -v ./internal/health

# Run with coverage
go test -cover ./...
```

### Running Tests

```bash
# Test health service
go test -v ./internal/health

# Test lifecycle manager
go test -v ./internal/lifecycle

# Test configuration loading
go test -v ./internal/config
```

## Performance

### Resource Usage Optimization

1. **Selective Monitoring**: Only monitor essential resources
2. **Snapshot Intervals**: Configure appropriate snapshot intervals (5m-30m)
3. **Namespace Filtering**: Monitor only specific namespaces
4. **System Resource Exclusion**: Skip system namespaces and frequent pods

### Monitoring Recommendations

- **Production**: Monitor `deployments`, `services`, `configmaps`, `secrets`
- **Development**: Add `pods` for debugging
- **Security**: Include `serviceaccounts` and `networkpolicies`
- **Storage**: Add `persistentvolumeclaims` if using persistent storage

## Troubleshooting

### Common Issues

1. **MongoDB Connection Failed**: Check database host, port, and credentials
2. **Kubernetes Access Denied**: Verify kubeconfig and RBAC permissions
3. **Resource Not Monitored**: Check if resource type is in configuration
4. **High Resource Usage**: Reduce monitored resources or increase intervals

### Logs

Check application logs for detailed information:

```bash
# Look for resource monitoring logs
grep "Setting up.*watcher" logs/driftguard.log

# Check for configuration errors
grep "resource type.*not configured" logs/driftguard.log

# Check database connection
grep "Failed to connect to MongoDB" logs/driftguard.log
```

### Health Checks

```bash
# Check overall health
curl http://localhost:8080/health

# Check specific components
curl http://localhost:8080/health | jq '.components'

# Check metrics for issues
curl http://localhost:8080/metrics | grep error
```

## Efficiency Options

See [EFFICIENCY_OPTIONS.md](EFFICIENCY_OPTIONS.md) for detailed information on:
- Snapshot throttling and intervals
- Resource filtering strategies
- Database cleanup approaches
- Storage optimization techniques

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## License

See [LICENSE.md](../LICENSE.md) for license information.
