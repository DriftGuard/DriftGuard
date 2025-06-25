# Contributing to DriftGuard

Thank you for your interest in contributing to DriftGuard! This document provides guidelines and information for contributors to help you get started and ensure your contributions align with the project's goals.

## Table of Contents

- [Project Overview](#project-overview)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Code Style and Standards](#code-style-and-standards)
- [Architecture Guidelines](#architecture-guidelines)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting](#issue-reporting)
- [Community Guidelines](#community-guidelines)

## Project Overview

DriftGuard is an intelligent GitOps configuration drift detection platform that combines real-time monitoring, AI-powered analysis, and automated remediation. The project addresses the critical challenge of configuration drift in modern DevOps operations.

### Key Components

- **Backend Infrastructure (Go)**: Core drift detection engine, Kubernetes integration, and real-time monitoring
- **AI/ML Components (Python)**: Machine learning models for drift classification and risk assessment
- **MCP Integration Layer**: Model Context Protocol for seamless AI-backend communication
- **Web Dashboard**: User interface for monitoring and managing drift detection

### Technology Stack

- **Go 1.21+**: Backend infrastructure and Kubernetes integration
- **Python 3.11+**: AI/ML components and data analysis
- **Kubernetes**: Target platform for drift detection
- **PostgreSQL**: Primary database for configuration and drift data
- **NATS/Kafka**: Real-time event streaming
- **FastAPI**: Python API framework for ML services
- **Gin**: Go web framework for REST APIs

## Getting Started

### Prerequisites

1. **Go 1.21 or later**
   ```bash
   # Install Go
   go version
   ```

2. **Python 3.11 or later**
   ```bash
   # Install Python
   python3 --version
   ```

3. **Kubernetes Cluster** (for testing)
   ```bash
   # Local development with minikube
   minikube start
   ```

4. **Docker**
   ```bash
   # Install Docker
   docker --version
   ```

5. **Git**
   ```bash
   # Install Git
   git --version
   ```

### Setting Up the Development Environment

1. **Fork and Clone the Repository**
   ```bash
   # Fork the repository on GitHub
   # Then clone your fork
   git clone https://github.com/your-username/DriftGuard.git
   cd DriftGuard
   ```

2. **Set Up Go Environment**
   ```bash
   # Initialize Go modules
   go mod init github.com/driftguard/core
   go mod tidy
   ```

3. **Set Up Python Environment**
   ```bash
   # Create virtual environment
   python3 -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   
   # Install dependencies
   pip install -r requirements.txt
   ```

4. **Configure Development Tools**
   ```bash
   # Install pre-commit hooks
   pre-commit install
   
   # Install development dependencies
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

## Development Environment

### Project Structure

```
DriftGuard/
├── cmd/                    # Application entry points
│   ├── controller/        # Main controller binary
│   ├── agent/            # Kubernetes agent
│   └── api/              # REST API server
├── internal/              # Private application code
│   ├── detector/         # Core drift detection logic
│   ├── analyzer/         # Configuration analysis
│   ├── watcher/          # Kubernetes resource watchers
│   └── storage/          # Database and storage layer
├── pkg/                   # Public libraries
│   ├── client/           # Go SDK for external use
│   ├── models/           # Shared data models
│   └── utils/            # Utility functions
├── ai/                    # Python AI/ML components
│   ├── models/           # ML models and training
│   ├── analysis/         # Drift analysis algorithms
│   └── api/              # FastAPI ML service
├── web/                   # Frontend dashboard
│   ├── src/              # React/TypeScript source
│   ├── public/           # Static assets
│   └── package.json      # Node.js dependencies
├── deploy/                # Deployment manifests
│   ├── kubernetes/       # K8s manifests
│   ├── helm/             # Helm charts
│   └── docker/           # Docker configurations
├── docs/                  # Documentation
├── tests/                 # Test files
└── scripts/               # Build and deployment scripts
```

### Local Development Setup

1. **Start Local Kubernetes Cluster**
   ```bash
   # Using minikube
   minikube start --cpus 4 --memory 8192
   
   # Or using kind
   kind create cluster --name driftguard
   ```

2. **Deploy Dependencies**
   ```bash
   # Deploy PostgreSQL
   kubectl apply -f deploy/kubernetes/postgresql.yaml
   
   # Deploy NATS
   kubectl apply -f deploy/kubernetes/nats.yaml
   ```

3. **Run Development Services**
   ```bash
   # Start Go backend
   go run cmd/controller/main.go --config=configs/dev.yaml
   
   # Start Python AI service
   cd ai && python -m uvicorn api.main:app --reload --port 8000
   
   # Start web dashboard
   cd web && npm run dev
   ```

## Code Style and Standards

### Go Code Standards

1. **Formatting**
   ```bash
   # Use gofmt for formatting
   gofmt -w .
   
   # Use goimports for import organization
   goimports -w .
   ```

2. **Linting**
   ```bash
   # Run golangci-lint
   golangci-lint run
   ```

3. **Code Style Guidelines**
   - Use meaningful variable and function names
   - Add comments for exported functions and types
   - Follow Go naming conventions
   - Use context.Context for cancellation and timeouts
   - Handle errors explicitly, don't ignore them

4. **Example Go Code**
   ```go
   // Package detector provides drift detection functionality
   package detector
   
   import (
       "context"
       "fmt"
       "time"
   )
   
   // DriftDetector monitors Kubernetes resources for configuration drift
   type DriftDetector struct {
       client    kubernetes.Interface
       analyzer  *Analyzer
       stopCh    chan struct{}
   }
   
   // NewDriftDetector creates a new drift detector instance
   func NewDriftDetector(client kubernetes.Interface, analyzer *Analyzer) *DriftDetector {
       return &DriftDetector{
           client:   client,
           analyzer: analyzer,
           stopCh:   make(chan struct{}),
       }
   }
   
   // Start begins monitoring for configuration drift
   func (d *DriftDetector) Start(ctx context.Context) error {
       // Implementation here
       return nil
   }
   ```

### Python Code Standards

1. **Formatting**
   ```bash
   # Use black for formatting
   black ai/
   
   # Use isort for import sorting
   isort ai/
   ```

2. **Linting**
   ```bash
   # Run flake8
   flake8 ai/
   
   # Run mypy for type checking
   mypy ai/
   ```

3. **Code Style Guidelines**
   - Follow PEP 8 style guide
   - Use type hints for all function parameters and return values
   - Add docstrings for all classes and functions
   - Use async/await for I/O operations
   - Handle exceptions appropriately

4. **Example Python Code**
   ```python
   """Drift analysis module for configuration drift detection."""
   
   from typing import Dict, List, Optional
   import asyncio
   from dataclasses import dataclass
   
   import numpy as np
   from sklearn.ensemble import IsolationForest
   
   
   @dataclass
   class DriftResult:
       """Result of drift analysis."""
       severity: str
       confidence: float
       details: Dict[str, any]
       timestamp: str
   
   
   class DriftAnalyzer:
       """Analyzes configuration drift using machine learning."""
   
       def __init__(self, model_path: Optional[str] = None):
           """Initialize the drift analyzer.
           
           Args:
               model_path: Path to pre-trained model file
           """
           self.model = self._load_model(model_path)
           self.anomaly_detector = IsolationForest(contamination=0.1)
   
       async def analyze_drift(
           self, 
           current_config: Dict[str, any], 
           desired_config: Dict[str, any]
       ) -> DriftResult:
           """Analyze configuration drift between current and desired states.
           
           Args:
               current_config: Current configuration state
               desired_config: Desired configuration state
           
           Returns:
               DriftResult with analysis details
           """
           # Implementation here
           pass
   ```

## Architecture Guidelines

### Design Principles

1. **Separation of Concerns**
   - Keep Go backend focused on infrastructure and real-time operations
   - Use Python for AI/ML and data analysis tasks
   - Maintain clear boundaries between components

2. **MCP Integration**
   - Use Model Context Protocol for Go-Python communication
   - Maintain context across service boundaries
   - Ensure scalable and extensible architecture

3. **Kubernetes Native**
   - Design as Kubernetes-native applications
   - Use Kubernetes APIs and patterns
   - Support operator pattern for complex deployments

4. **Observability**
   - Implement comprehensive logging
   - Use structured logging with correlation IDs
   - Add metrics and tracing for all operations

### Component Guidelines

1. **Go Backend Components**
   - Use dependency injection for testability
   - Implement interfaces for external dependencies
   - Use context for cancellation and timeouts
   - Handle graceful shutdown

2. **Python AI Components**
   - Design for async operations
   - Use dependency injection for ML models
   - Implement proper error handling and retries
   - Cache model predictions when appropriate

3. **API Design**
   - Follow RESTful principles
   - Use consistent error responses
   - Implement proper authentication and authorization
   - Version APIs appropriately

## Testing Guidelines

### Go Testing

1. **Unit Tests**
   ```bash
   # Run unit tests
   go test ./...
   
   # Run with coverage
   go test -cover ./...
   ```

2. **Integration Tests**
   ```bash
   # Run integration tests
   go test -tags=integration ./...
   ```

3. **Test Guidelines**
   - Write tests for all exported functions
   - Use table-driven tests for multiple scenarios
   - Mock external dependencies
   - Test error conditions

### Python Testing

1. **Unit Tests**
   ```bash
   # Run unit tests
   pytest ai/tests/
   
   # Run with coverage
   pytest --cov=ai ai/tests/
   ```

2. **Integration Tests**
   ```bash
   # Run integration tests
   pytest ai/tests/integration/
   ```

3. **Test Guidelines**
   - Use pytest fixtures for test setup
   - Mock external services and APIs
   - Test async functions with asyncio
   - Use parameterized tests for multiple scenarios

### End-to-End Testing

1. **Kubernetes Testing**
   ```bash
   # Run E2E tests
   go test -tags=e2e ./tests/e2e/
   ```

2. **Test Environment**
   - Use testcontainers for external dependencies
   - Create isolated test namespaces
   - Clean up resources after tests

## Pull Request Process

### Before Submitting a PR

1. **Ensure Code Quality**
   ```bash
   # Run all checks
   make check
   
   # Run tests
   make test
   
   # Build the project
   make build
   ```

2. **Update Documentation**
   - Update relevant documentation
   - Add comments for new functionality
   - Update README if needed

3. **Create Meaningful Commits**
   ```bash
   # Use conventional commit format
   git commit -m "feat: add drift classification model"
   git commit -m "fix: resolve memory leak in watcher"
   git commit -m "docs: update API documentation"
   ```

### PR Guidelines

1. **PR Title and Description**
   - Use clear, descriptive titles
   - Provide detailed description of changes
   - Include issue numbers if applicable
   - Add screenshots for UI changes

2. **Code Review Checklist**
   - [ ] Code follows style guidelines
   - [ ] Tests are included and passing
   - [ ] Documentation is updated
   - [ ] No breaking changes (or clearly documented)
   - [ ] Performance impact considered
   - [ ] Security implications reviewed

3. **Review Process**
   - At least one approval required
   - Address all review comments
   - Update PR based on feedback
   - Squash commits before merging

### Commit Message Format

Use conventional commit format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test changes
- `chore`: Build/tooling changes

## Issue Reporting

### Bug Reports

When reporting bugs, please include:

1. **Environment Information**
   - Operating system and version
   - Go version
   - Python version
   - Kubernetes version
   - DriftGuard version

2. **Steps to Reproduce**
   - Clear, step-by-step instructions
   - Minimal example if possible
   - Expected vs actual behavior

3. **Additional Information**
   - Logs and error messages
   - Screenshots if applicable
   - Related issues or PRs

### Feature Requests

When requesting features, please include:

1. **Problem Description**
   - What problem does this solve?
   - Current workarounds
   - Impact on users

2. **Proposed Solution**
   - How should it work?
   - Alternative approaches considered
   - Implementation suggestions

3. **Additional Context**
   - Use cases and examples
   - Priority level
   - Related features

## Community Guidelines

### Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please:

- Be respectful and considerate of others
- Use inclusive language
- Focus on constructive feedback
- Help others learn and grow

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and discussions
- **Pull Requests**: Code contributions and reviews

### Getting Help

1. **Check Documentation**
   - README.md for project overview
   - IMPLEMENTATION.md for technical details
   - HOSTING.md for deployment information

2. **Search Existing Issues**
   - Check if your question has been answered
   - Look for similar bug reports

3. **Ask Questions**
   - Use GitHub Discussions for general questions
   - Be specific and provide context
   - Include relevant code and error messages

### Recognition

Contributors will be recognized in:

- Project README contributors section
- Release notes for significant contributions
- GitHub contributors graph

## Development Workflow

### Branch Strategy

1. **Main Branch**
   - `main`: Production-ready code
   - Protected branch requiring PR reviews

2. **Development Branches**
   - `develop`: Integration branch for features
   - `feature/*`: Individual feature branches
   - `hotfix/*`: Critical bug fixes

3. **Release Process**
   ```bash
   # Create release branch
   git checkout -b release/v1.0.0
   
   # Update version and changelog
   # Create PR to main
   # Tag release after merge
   git tag v1.0.0
   git push origin v1.0.0
   ```

### Release Process

1. **Version Management**
   - Use semantic versioning (MAJOR.MINOR.PATCH)
   - Update version in all relevant files
   - Generate changelog from commits

2. **Release Checklist**
   - [ ] All tests passing
   - [ ] Documentation updated
   - [ ] Version numbers updated
   - [ ] Changelog generated
   - [ ] Release notes written
   - [ ] Docker images built and pushed
   - [ ] Helm charts updated

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [Python Documentation](https://docs.python.org/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [GitHub Flow](https://guides.github.com/introduction/flow/)
- [Conventional Commits](https://www.conventionalcommits.org/)

Thank you for contributing to DriftGuard! Your contributions help make GitOps configuration drift detection more reliable and accessible for the entire DevOps community. 