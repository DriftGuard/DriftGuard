# Resource Monitoring Configuration

DriftGuard allows you to configure which Kubernetes resources to monitor for drift detection. By default, only resources explicitly listed in the configuration file will be monitored.

## Configuration

The `kubernetes.resources` field in your `config.yaml` file controls which resource types are monitored:

```yaml
kubernetes:
  resources: [
    "deployments",     # Application deployments
    "services",        # Network services
    "configmaps",      # Configuration data
    "secrets"          # Sensitive data
  ]
```

## Supported Resource Types

The following resource types can be configured for monitoring:

| Resource Type | Description | Use Case |
|---------------|-------------|----------|
| `deployments` | Application deployments | Core application drift detection |
| `services` | Network services | Service configuration drift |
| `configmaps` | Configuration data | Config drift detection |
| `secrets` | Sensitive data | Secret management drift |
| `pods` | Individual pods | Pod-level drift (high frequency) |
| `ingresses` | Ingress rules | Network routing drift |
| `persistentvolumeclaims` | Storage claims | Storage configuration drift |
| `serviceaccounts` | Service accounts | RBAC drift detection |
| `networkpolicies` | Network policies | Network security drift |

## Configuration Examples

### Minimal Monitoring (Recommended for Production)
Monitor only essential resources to reduce overhead:

```yaml
kubernetes:
  resources: [
    "deployments",
    "services", 
    "configmaps",
    "secrets"
  ]
```

### Comprehensive Monitoring
Monitor all resource types for complete drift detection:

```yaml
kubernetes:
  resources: [
    "deployments",
    "services",
    "configmaps", 
    "secrets",
    "ingresses",
    "persistentvolumeclaims",
    "serviceaccounts",
    "networkpolicies"
  ]
```

### Development Environment
Monitor frequently changing resources for development:

```yaml
kubernetes:
  resources: [
    "deployments",
    "services",
    "configmaps",
    "pods"  # Include pods for development debugging
  ]
```

### Security-Focused Monitoring
Focus on security-related resources:

```yaml
kubernetes:
  resources: [
    "secrets",
    "serviceaccounts", 
    "networkpolicies",
    "configmaps"  # May contain security configs
  ]
```

## Performance Considerations

### High-Frequency Resources
Some resources change very frequently and may generate excessive events:

- **Pods**: Change frequently due to scaling, restarts, updates
- **Services**: May change during deployments
- **ConfigMaps**: Change when configuration is updated

### Low-Frequency Resources  
These resources change less frequently and are good for monitoring:

- **Deployments**: Core application configuration
- **Secrets**: Security credentials
- **Ingresses**: Network routing rules
- **NetworkPolicies**: Security policies

## Best Practices

1. **Start Minimal**: Begin with essential resources (`deployments`, `services`, `configmaps`, `secrets`)

2. **Add Gradually**: Add more resource types based on your specific needs

3. **Monitor Storage**: If using persistent storage, include `persistentvolumeclaims`

4. **Security Focus**: Always monitor `secrets` and `serviceaccounts` for security drift

5. **Network Security**: Include `networkpolicies` and `ingresses` for network security drift

6. **Avoid Pods in Production**: Pods change too frequently for effective drift detection

## Configuration Validation

DriftGuard validates your resource configuration on startup:

- Invalid resource types will cause startup errors
- Empty resource list will prevent monitoring
- Resource names are case-insensitive

## Runtime Behavior

- Only configured resources will be monitored
- API calls for unconfigured resources will return errors
- Resource watchers are only started for configured types
- Database snapshots are only saved for monitored resources

## Troubleshooting

### Resource Not Monitored
If a resource type is not being monitored:

1. Check if it's listed in `kubernetes.resources`
2. Verify the resource name spelling (case-insensitive)
3. Restart DriftGuard after configuration changes

### High Resource Usage
If monitoring is consuming too many resources:

1. Reduce the number of monitored resource types
2. Increase `snapshot_interval` for less frequent snapshots
3. Enable `skip_frequent_pods` if monitoring pods
4. Use `skip_system_namespaces` to exclude system resources

### Missing Drift Detection
If drift is not being detected:

1. Ensure the resource type is in the configuration
2. Check that the resource is in a monitored namespace
3. Verify the resource has the expected labels
4. Check logs for any monitoring errors 