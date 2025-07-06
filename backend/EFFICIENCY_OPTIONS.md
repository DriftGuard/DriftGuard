# DriftGuard Efficiency Options

## Database Efficiency Controls

DriftGuard provides several options to control database usage and prevent excessive storage consumption:

### 1. **Snapshot Throttling** (Recommended)
- **`enable_snapshots`**: Enable/disable snapshot saving entirely
- **`snapshot_interval`**: Minimum time between snapshots for the same resource (default: 5m)
- **`skip_system_namespaces`**: Skip kube-system, kube-public, and default namespaces
- **`skip_frequent_pods`**: Skip deployment pods that change frequently

### 2. **Configuration Examples**

#### Minimal Storage Usage
```yaml
kubernetes:
  enable_snapshots: true
  snapshot_interval: 30m  # Only snapshot every 30 minutes
  skip_system_namespaces: true
  skip_frequent_pods: true
```

#### Moderate Storage Usage
```yaml
kubernetes:
  enable_snapshots: true
  snapshot_interval: 10m  # Snapshot every 10 minutes
  skip_system_namespaces: true
  skip_frequent_pods: false  # Include pod changes
```

#### Full Monitoring (High Storage)
```yaml
kubernetes:
  enable_snapshots: true
  snapshot_interval: 1m   # Snapshot every minute
  skip_system_namespaces: false
  skip_frequent_pods: false
```

### 3. **Alternative Approaches**

#### Option A: Event-Only Mode
Disable snapshots and only track drift events:
```yaml
kubernetes:
  enable_snapshots: false  # No snapshots saved
```

#### Option B: Selective Resource Monitoring
Only watch specific resource types:
```yaml
kubernetes:
  resources: ["deployments", "configmaps", "secrets"]  # Skip pods, services
```

#### Option C: Namespace Filtering
Only monitor specific namespaces:
```yaml
kubernetes:
  namespaces: ["production", "staging"]  # Skip test namespaces
```

### 4. **Database Cleanup Strategies**

#### Automatic Cleanup
Add to your deployment:
```yaml
# CronJob to clean old snapshots
apiVersion: batch/v1
kind: CronJob
metadata:
  name: driftguard-cleanup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: mongo:latest
            command:
            - mongosh
            - driftguard
            - --eval
            - "db.configuration_snapshots.deleteMany({created_at: {\$lt: new Date(Date.now() - 30*24*60*60*1000)}})"
```

#### Manual Cleanup Commands
```bash
# Remove snapshots older than 7 days
mongosh driftguard --eval "db.configuration_snapshots.deleteMany({created_at: {\$lt: new Date(Date.now() - 7*24*60*60*1000)}})"

# Remove snapshots for specific resource
mongosh driftguard --eval "db.configuration_snapshots.deleteMany({resource_name: 'web-app'})"

# Remove all snapshots
mongosh driftguard --eval "db.configuration_snapshots.deleteMany({})"
```

### 5. **Storage Estimation**

| Configuration | Snapshots/Day | Storage/Month | Cost/Month* |
|---------------|---------------|---------------|-------------|
| Minimal (30m) | ~48 | ~50MB | $0.01 |
| Moderate (10m) | ~144 | ~150MB | $0.03 |
| Full (1m) | ~1440 | ~1.5GB | $0.30 |

*Based on MongoDB Atlas pricing

### 6. **Best Practices**

1. **Start with Minimal Configuration**: Use 30-minute intervals initially
2. **Monitor Storage Usage**: Check database size regularly
3. **Use Namespace Filtering**: Only monitor production namespaces
4. **Implement Cleanup**: Set up automatic cleanup of old snapshots
5. **Consider Event-Only Mode**: For cost-sensitive environments

### 7. **Runtime Controls**

You can also control behavior at runtime:

```bash
# Disable snapshots temporarily
curl -X POST http://localhost:8080/api/v1/config/snapshots/disable

# Enable snapshots
curl -X POST http://localhost:8080/api/v1/config/snapshots/enable

# Change snapshot interval
curl -X POST http://localhost:8080/api/v1/config/snapshots/interval \
  -H "Content-Type: application/json" \
  -d '{"interval": "15m"}'
```

### 8. **Monitoring and Alerts**

Set up alerts for:
- Database size exceeding thresholds
- Snapshot creation rate
- Storage cost per day

This ensures you stay within your budget while maintaining effective drift detection. 