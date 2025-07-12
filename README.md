# DriftGuard

**Intelligent GitOps Configuration Drift Detection Engine**

<img src="DriftGaurd.png" alt="DriftGuard" width="400" height="auto">

DriftGuard is an intelligent GitOps configuration drift detection platform that continuously monitors Kubernetes resources for configuration drift by comparing live cluster states against desired states stored in Git repositories. It features enhanced state tracking, automatic resolution detection, and comprehensive drift analysis.

## 🚀 Key Features

- **🔍 Enhanced Drift Detection**: Continuously monitors Kubernetes resources with intelligent drift classification
- **✅ Drift Resolution Detection**: Automatically detects when drift is resolved and configuration matches Git again
- **📊 State Management**: Tracks drift status (active/resolved/none) with timestamps and resolution messages
- **🔗 Git Integration**: Compares live state against Git-stored manifests with hash-based state tracking
- **🏷️ Intelligent Classification**: Automatically classifies drift types (Scaling, VersionChange, ResourceChange, etc.)
- **⚠️ Severity Assessment**: Assigns severity levels (low, medium, high) to detected drifts
- **💾 MongoDB Storage**: Persistent storage of drift records with enhanced state tracking
- **🌐 REST API**: Comprehensive HTTP endpoints for querying drift records, status filtering, and statistics
- **📈 Metrics & Monitoring**: Prometheus metrics and health checks
- **📝 Enhanced Logging**: Detailed logging with emojis for drift detection and resolution events

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │   Git Repo      │    │   DriftGuard    │
│   API Server    │    │   (Desired      │    │   Backend       │
│   (Live State)  │◄──►│   State)        │◄──►│   (Controller)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   MongoDB       │    │   REST API      │
                       │   (Enhanced     │    │   (HTTP         │
                       │   Drift Records)│    │   Endpoints)    │
                       └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Prometheus    │    │   Health        │
                       │   Metrics       │    │   Checks        │
                       └─────────────────┘    └─────────────────┘
```

## 📋 Prerequisites

- **Go 1.21+**
- **MongoDB 4.4+**
- **Kubernetes cluster** (local or remote)
- **Git repository** with Kubernetes manifests
- **kubectl** configured to access your cluster

## 🛠️ Installation & Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd DriftGuard
```

### 2. Start MongoDB

```bash
# Create data directory
mkdir -p ~/data/db

# Start MongoDB (macOS/Linux)
sudo mongod --dbpath=~/data/db

# Or using Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest
```

### 3. Configure DriftGuard

Update the configuration file (`backend/configs/config.yaml`):

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

database:
  host: localhost
  port: 27017
  dbname: driftguard
  timeout: 10s

kubernetes:
  config_path: ""  # Leave empty for default kubeconfig
  context: ""
  namespaces: ["driftguard"]
  resources: ["deployments", "services", "configmaps", "secrets"]
  skip_system_namespaces: true

git:
  repository_url: "Add your Repo"
  default_branch: main
  clone_timeout: 5m
  pull_timeout: 2m

drift_detection:
  interval: 30s
  enable_periodic: true
```

### 4. Create Test Git Repository

```bash
# Create a dummy Git repository for testing
mkdir -p dummy-k8s-repo
cd dummy-k8s-repo
git init

# Create test Kubernetes manifests
cat > test-deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-app
  namespace: driftguard
  labels:
    app: nginx
    environment: test
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.23
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        env:
        - name: NGINX_ENV
          value: "production"
        - name: LOG_LEVEL
          value: "info"
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  namespace: driftguard
  labels:
    app: nginx
spec:
  selector:
    app: nginx
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  type: ClusterIP
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-config
  namespace: driftguard
data:
  nginx.conf: |
    server {
        listen 80;
        server_name localhost;
        
        location / {
            root   /usr/share/nginx/html;
            index  index.html index.htm;
        }
        
        error_page   500 502 503 504  /50x.html;
        location = /50x.html {
            root   /usr/share/nginx/html;
        }
    }
  environment: "production"
  version: "1.0.0"
EOF

# Commit the manifests
git add test-deployment.yaml
git commit -m "Initial Kubernetes manifests"
```

### 5. Create Kubernetes Namespace

```bash
kubectl create namespace driftguard
```

### 6. Start DriftGuard

```bash
cd backend
go run cmd/controller/main.go --config=configs/config.yaml
```

## 🧪 Comprehensive Testing Guide

### Step 1: Verify DriftGuard is Running

```bash
# Check if DriftGuard is running
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","message":"Service is running","time":"2025-01-15T10:30:00Z"}
```

### Step 2: Deploy Initial Resources

```bash
# Apply the manifests from Git
kubectl apply -f ../dummy-k8s-repo/test-deployment.yaml

# Verify resources are created
kubectl get all -n driftguard
```

### Step 3: Trigger Initial Drift Analysis

```bash
# Trigger manual drift analysis
curl -X POST http://localhost:8080/api/v1/analyze

# Expected response:
# {"message":"Drift analysis triggered successfully","status":"started"}
```

### Step 4: Check Initial State

```bash
# Wait a few seconds, then check drift records
sleep 5
curl http://localhost:8080/api/v1/drift-records

# You should see records with drift_detected: false initially
```

### Step 5: Create Drift by Scaling Deployment

```bash
# Scale the deployment to create drift
kubectl scale deployment nginx-app -n driftguard --replicas=3

# Verify the change
kubectl get deployment nginx-app -n driftguard
```

### Step 6: Detect Drift

```bash
# Trigger drift analysis again
curl -X POST http://localhost:8080/api/v1/analyze

# Wait and check for drift detection
sleep 5
curl http://localhost:8080/api/v1/drift-records/active

# You should now see active drift records
```

### Step 7: Resolve Drift

```bash
# Scale back to original state
kubectl scale deployment nginx-app -n driftguard --replicas=2

# Or re-apply the Git manifest
kubectl apply -f ../dummy-k8s-repo/test-deployment.yaml

# Trigger analysis again
curl -X POST http://localhost:8080/api/v1/analyze

# Wait and check for resolution
sleep 5
curl http://localhost:8080/api/v1/drift-records/resolved
```

### Step 8: Test Different Types of Drift

```bash
# Test image change drift
kubectl set image deployment/nginx-app nginx=nginx:1.24 -n driftguard

# Test environment variable drift
kubectl set env deployment/nginx-app NGINX_ENV=staging -n driftguard

# Test resource drift
kubectl patch deployment nginx-app -n driftguard -p '{"spec":{"template":{"spec":{"containers":[{"name":"nginx","resources":{"requests":{"memory":"128Mi"}}}]}}}}'

# Trigger analysis after each change
curl -X POST http://localhost:8080/api/v1/analyze
```

### Step 9: Check Statistics

```bash
# Get comprehensive statistics
curl http://localhost:8080/api/v1/statistics

# Expected response includes:
# - total_records
# - active_drift
# - resolved_drift
# - no_drift
# - active_percentage
# - resolved_percentage
```

### Step 10: Test API Filtering

```bash
# Filter by namespace
curl "http://localhost:8080/api/v1/drift-records?namespace=driftguard"

# Filter by drift status
curl "http://localhost:8080/api/v1/drift-records?drift_detected=true"

# Get specific resource drift
curl http://localhost:8080/api/v1/drift-records/Deployment:driftguard:nginx-app
```

## 🌐 API Reference

### Health & Monitoring Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Basic health check |
| `GET` | `/ready` | Readiness check |
| `GET` | `/metrics` | Prometheus metrics |

### Drift Detection API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/drift-records` | List all drift records |
| `GET` | `/api/v1/drift-records/:resourceId` | Get specific drift record |
| `GET` | `/api/v1/drift-records/active` | Get active drift records |
| `GET` | `/api/v1/drift-records/resolved` | Get resolved drift records |
| `GET` | `/api/v1/statistics` | Get drift statistics |
| `POST` | `/api/v1/analyze` | Trigger manual drift analysis |

### Query Parameters

**List Drift Records:**
```bash
# Get all drift records
curl http://localhost:8080/api/v1/drift-records

# Filter by namespace
curl "http://localhost:8080/api/v1/drift-records?namespace=driftguard"

# Filter by drift status
curl "http://localhost:8080/api/v1/drift-records?drift_detected=true"
```

**Get Specific Record:**
```bash
# Get drift record for a specific resource
curl http://localhost:8080/api/v1/drift-records/Deployment:driftguard:nginx-app
```

## 📊 Drift Detection Features

### Resource Identification
Resources are identified using the format: `kind:namespace:name`

### Drift Status Tracking
The system tracks drift status with three states:

- **`active`**: Drift is currently detected and active
- **`resolved`**: Drift was previously detected but has been resolved
- **`none`**: No drift detected

### Drift Classification
The system automatically classifies drifts into categories:

- **Scaling**: Replica count changes
- **VersionChange**: Container image changes
- **ResourceChange**: CPU/memory resource changes
- **ConfigurationChange**: Environment variable changes
- **LabelChange**: Label modifications
- **AnnotationChange**: Annotation modifications
- **PortChange**: Port configuration changes
- **VolumeChange**: Volume mount changes
- **SecretChange**: Secret reference changes
- **ConfigMapChange**: ConfigMap reference changes

### Severity Levels

- **High**: Image changes, secret changes, significant scaling (>50%)
- **Medium**: Minor scaling, environment variables, ports, volumes
- **Low**: Labels, annotations, other metadata

## 📈 Enhanced Statistics

The system provides comprehensive drift statistics:

```json
{
  "statistics": {
    "total_records": 25,
    "active_drift": 3,
    "resolved_drift": 15,
    "no_drift": 7,
    "recent_active_drift": 1,
    "recent_resolutions": 2,
    "active_percentage": 12.0,
    "resolved_percentage": 60.0
  }
}
```

## 🔧 Configuration Options

### Kubernetes Configuration
- `config_path`: Path to kubeconfig file
- `context`: Kubernetes context to use
- `namespaces`: List of namespaces to monitor
- `resources`: List of resource types to monitor
- `skip_system_namespaces`: Skip system namespaces

### Git Configuration
- `repository_url`: Git repository URL
- `default_branch`: Branch to monitor
- `clone_timeout`: Timeout for repository cloning
- `pull_timeout`: Timeout for pulling updates

### Drift Detection Configuration
- `interval`: Periodic analysis interval
- `enable_periodic`: Enable periodic drift detection

## 🐛 Troubleshooting

### Common Issues

1. **MongoDB Connection Failed**
   ```bash
   # Check if MongoDB is running
   ps aux | grep mongod
   
   # Start MongoDB if not running
   sudo mongod --dbpath=~/data/db
   ```

2. **Kubernetes Connection Failed**
   ```bash
   # Check kubectl configuration
   kubectl config current-context
   
   # Test cluster connection
   kubectl get nodes
   ```

3. **Git Repository Access Failed**
   ```bash
   # Check Git repository URL in config
   # Ensure repository is accessible
   git ls-remote <repository-url>
   ```

4. **Port Already in Use**
   ```bash
   # Check if port 8080 is in use
   lsof -i :8080
   
   # Kill process or change port in config
   ```

### Log Analysis

DriftGuard uses structured logging with emojis:
- 🚀 System startup
- ⚠️ Drift detected
- ✅ Drift resolved
- 🔄 Drift continued
- 📊 Analysis completed

## 🚀 Production Deployment

### Docker Deployment

```bash
# Build the container
docker build -t driftguard:latest backend/

# Run with environment variables
docker run -d \
  -p 8080:8080 \
  -e MONGODB_HOST=your-mongodb-host \
  -e KUBECONFIG=/path/to/kubeconfig \
  driftguard:latest
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driftguard
  namespace: driftguard
spec:
  replicas: 1
  selector:
    matchLabels:
      app: driftguard
  template:
    metadata:
      labels:
        app: driftguard
    spec:
      containers:
      - name: driftguard
        image: driftguard:latest
        ports:
        - containerPort: 8080
        env:
        - name: MONGODB_HOST
          value: "mongodb-service"
        volumeMounts:
        - name: kubeconfig
          mountPath: /root/.kube
          readOnly: true
      volumes:
      - name: kubeconfig
        secret:
          secretName: driftguard-kubeconfig
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE.md) file for details.

## 📞 Support

For questions, feature requests, or contributions, please open an issue in this repository.

