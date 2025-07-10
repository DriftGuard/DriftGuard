# DriftGuard

**Kubernetes + Terraform GitOps Drift Detection Engine**

<img src="DriftGaurd.png" alt="DriftGuard" width="400" height="auto">

DriftGuard is a modular and extensible backend system that detects configuration drift between the live state of your infrastructure (Kubernetes or AWS) and the desired state stored in Git repositories or Terraform state files.

Built for real-time drift detection in Kubernetes, designed to scale for Terraform & AWS infrastructure, with native support for GitOps, Prometheus, and future AI integrations.

## Overview

Modern infrastructure is GitOps-driven, but challenges persist:

- Manual changes in production environments continue to occur
- Drift between Git repositories and reality causes operational incidents
- Kubernetes and Terraform lack unified drift detection solutions

DriftGuard addresses these challenges by detecting and alerting when your production environment drifts from what Git/Infrastructure-as-Code defines as the desired state.

## Key Features

| Layer            | Feature                                | Status           |
| ---------------- | -------------------------------------- | ---------------- |
| Kubernetes       | Real-time resource drift detection     | Implemented      |
| Git              | Compare against GitOps desired state   | Work in Progress |
| Terraform        | `.tfstate` parsing and drift analysis  | Work in Progress |
| AWS              | Live state querying via AWS SDK        | Planned          |
| Metrics          | Prometheus metrics for observability   | Implemented      |
| API              | RESTful API with health and statistics | Implemented      |
| Notifications    | Slack, Email, Webhooks via Notifier    | Planned          |
| Auto-Remediation | GitHub PRs for drift fixes             | Planned          |
| AI Risk Engine   | Drift severity and confidence scoring  | Future AI Phase  |

## Architecture Overview

```
           ┌─────────────────────────────┐
           │       Git Repository        │◄──┐
           └─────────────────────────────┘   │
                    ▲                        │
                    │                        ▼
┌────────────┐ ┌─────────────────────────────┐
│ Kubernetes │──►│ DriftGuard Watcher │────┐
└────────────┘ └─────────────────────────────┘ │
                          ▼
        ┌────────────────────┐
        │ Drift Detection Core│
        └────────────────────┘
                          │
┌────────────────────┬───────────────────┴──────────────┬────────────────────┐
▼                    ▼                                   ▼                    ▼
MongoDB Snapshot DB  Prometheus Metrics  Notifier System (Planned)  Remediation Engine (Planned)
```

## Repository Structure

```
backend/
├── cmd/
│   └── controller/             # Main application entrypoint
├── configs/                    # YAML-based configuration
├── internal/
│   ├── aws/                    # AWS SDK live state fetchers (planned)
│   ├── config/                 # Application config loader & validator
│   ├── controller/             # Orchestrates drift detection
│   ├── database/               # MongoDB integration
│   ├── git/                    # Git desired-state loader (WIP)
│   ├── health/                 # Readiness and liveness checks
│   ├── lifecycle/              # Graceful startup/shutdown
│   ├── logger/                 # Zap-based structured logging
│   ├── mcp/                    # Future AI/ML integration client
│   ├── metrics/                # Prometheus exporters
│   ├── notifier/               # Slack/Email/SNS (Planned)
│   ├── remediator/             # GitHub PR generator (Planned)
│   ├── server/                 # Gin-based HTTP API server
│   ├── terraform/              # Terraform drift module (Planned)
│   └── watcher/                # Kubernetes resource informers
├── pkg/
│   └── models/                 # Shared Go types and DTOs
├── docs/                       # Architecture documentation
└── README.md                   # Project documentation
```

## How It Works

1. **Watches live Kubernetes state** (deployments, services, configmaps, secrets)
2. **Fetches desired state** from Git repositories (Work in Progress)
3. **Fetches AWS infrastructure state** from `.tfstate` files or AWS SDK (Planned)
4. Compares live and desired states to calculate drift scores
5. Stores snapshots and drift events in MongoDB
6. Exposes metrics for Prometheus monitoring
7. Provides REST API for querying drifts and system health

## Implementation Status

### Fully Implemented

- Kubernetes Watcher with informers
- Configuration Management system
- MongoDB Backend integration
- Metrics Exporter for Prometheus
- Lifecycle & Health checks
- REST API (basic endpoints)

### Work In Progress

- Git integration for desired state
- Terraform `.tfstate` parsing
- GitHub remediation framework
- Enhanced logging configuration

### Planned Features

- AWS SDK integration
- Notification system (Slack, Email, Webhook)
- Auto-remediation engine
- AI-based risk scoring and drift analysis

## Use Cases

- **Detect unauthorized changes**: Identify when Kubernetes resources are modified outside of GitOps workflows
- **Infrastructure compliance**: Alert when AWS infrastructure doesn't match Terraform state files
- **Drift pattern analysis**: Track configuration drift patterns across environments (production, staging, development)
- **Automated remediation**: Generate GitHub Pull Requests to correct detected drift
- **Operational visibility**: Provide metrics and dashboards for infrastructure drift monitoring

## Technology Stack

| Tool             | Purpose                                |
| ---------------- | -------------------------------------- |
| **Go**           | Core backend language                  |
| **Kubernetes**   | Live resource monitoring via client-go |
| **MongoDB**      | Snapshot and drift data persistence    |
| **Prometheus**   | Metrics collection and alerting        |
| **Gin**          | Lightweight HTTP API framework         |
| **Terraform**    | Infrastructure-as-Code source of truth |
| **AWS SDK (Go)** | Planned live cloud state integration   |
| **Zap**          | Structured logging framework           |

## Configuration

DriftGuard uses YAML-based configuration files located in the `configs/` directory. Key configuration areas include:

- **Kubernetes**: Cluster connection and resource watching
- **Database**: MongoDB connection settings
- **Metrics**: Prometheus endpoint configuration
- **API**: HTTP server settings
- **Logging**: Structured logging levels and outputs

## Monitoring and Observability

### Prometheus Metrics

- `driftguard_drift_events_total`: Total number of drift events detected
- `driftguard_resources_watched`: Number of Kubernetes resources being monitored
- `driftguard_drift_score`: Current drift score by resource type
- `driftguard_sync_duration_seconds`: Time taken for drift detection cycles

### Health Endpoints

- `/health/ready`: Readiness probe for Kubernetes deployments
- `/health/live`: Liveness probe for container health
- `/metrics`: Prometheus metrics endpoint

## Development Setup

### Prerequisites

- Go 1.21 or higher
- MongoDB instance
- Kubernetes cluster access
- Docker (for containerized deployment)

### Local Development

```bash
# Clone the repository
git clone <repository-url>
cd driftguard

# Install dependencies
go mod download

# Run the application
go run cmd/controller/main.go
```

### Docker Deployment

```bash
# Build the container
docker build -t driftguard:latest .

# Run with docker-compose
docker-compose up -d
```

## Contributing

We welcome contributions from the DevOps and platform engineering community. Please refer to our contribution guidelines for:

- Code style and standards
- Testing requirements
- Pull request process
- Issue reporting

## Roadmap

### Q1 2025

- Complete Git integration for desired state comparison
- Terraform state file parsing implementation
- Enhanced API endpoints for drift querying

### Q2 2025

- AWS SDK integration for live infrastructure state
- Notification system implementation
- Auto-remediation engine development

### Q3 2025

- AI-based drift risk scoring
- Advanced analytics and reporting
- Multi-cloud support expansion

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contact

For questions, feature requests, or contributions, please open an issue in this repository or contact the [maintainers](https://github.com/DriftGuard/DriftGuard/blob/main/.github/CODEOWNERS).

