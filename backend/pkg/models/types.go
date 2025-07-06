package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigurationSnapshot represents a snapshot of a Kubernetes resource configuration
type ConfigurationSnapshot struct {
	ID           uuid.UUID       `json:"id" bson:"id"`
	Environment  string          `json:"environment" bson:"environment"`
	Namespace    string          `json:"namespace" bson:"namespace"`
	ResourceType string          `json:"resource_type" bson:"resource_type"`
	ResourceName string          `json:"resource_name" bson:"resource_name"`
	GitCommit    string          `json:"git_commit" bson:"git_commit"`
	GitBranch    string          `json:"git_branch" bson:"git_branch"`
	LiveState    json.RawMessage `json:"live_state" bson:"live_state"`
	DesiredState json.RawMessage `json:"desired_state" bson:"desired_state"`
	DriftScore   float64         `json:"drift_score" bson:"drift_score"`
	CreatedAt    time.Time       `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" bson:"updated_at"`
}

// DriftEvent represents a detected configuration drift event
type DriftEvent struct {
	ID              uuid.UUID       `json:"id" bson:"id"`
	SnapshotID      uuid.UUID       `json:"snapshot_id" bson:"snapshot_id"`
	Environment     string          `json:"environment" bson:"environment"`
	Namespace       string          `json:"namespace" bson:"namespace"`
	ResourceType    string          `json:"resource_type" bson:"resource_type"`
	ResourceName    string          `json:"resource_name" bson:"resource_name"`
	DriftType       DriftType       `json:"drift_type" bson:"drift_type"`
	Severity        Severity        `json:"severity" bson:"severity"`
	Description     string          `json:"description" bson:"description"`
	Details         json.RawMessage `json:"details" bson:"details"`
	Remediation     string          `json:"remediation" bson:"remediation"`
	Status          EventStatus     `json:"status" bson:"status"`
	DetectedAt      time.Time       `json:"detected_at" bson:"detected_at"`
	ResolvedAt      *time.Time      `json:"resolved_at" bson:"resolved_at"`
	ResolvedBy      *string         `json:"resolved_by" bson:"resolved_by"`
	ResolutionNotes *string         `json:"resolution_notes" bson:"resolution_notes"`
}

// DriftType represents the type of configuration drift
type DriftType string

const (
	DriftTypeMissing   DriftType = "missing"
	DriftTypeExtra     DriftType = "extra"
	DriftTypeModified  DriftType = "modified"
	DriftTypeOutOfSync DriftType = "out_of_sync"
	DriftTypeAnomaly   DriftType = "anomaly"
)

// Severity represents the severity level of a drift event
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// EventStatus represents the status of a drift event
type EventStatus string

const (
	EventStatusOpen     EventStatus = "open"
	EventStatusResolved EventStatus = "resolved"
	EventStatusIgnored  EventStatus = "ignored"
	EventStatusPending  EventStatus = "pending"
)

// KubernetesResource represents a Kubernetes resource with metadata
type KubernetesResource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   metav1.ObjectMeta      `json:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Status     map[string]interface{} `json:"status,omitempty"`
}

// GitRepository represents a Git repository configuration
type GitRepository struct {
	URL      string `json:"url" yaml:"url"`
	Branch   string `json:"branch" yaml:"branch"`
	Path     string `json:"path" yaml:"path"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	SSHKey   string `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"`
}

// Environment represents a deployment environment
type Environment struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	GitRepo     GitRepository     `json:"git_repo" yaml:"git_repo"`
	K8sContext  string            `json:"k8s_context" yaml:"k8s_context"`
}

// DriftAnalysis represents the result of drift analysis
type DriftAnalysis struct {
	SnapshotID  uuid.UUID       `json:"snapshot_id"`
	DriftScore  float64         `json:"drift_score"`
	Severity    Severity        `json:"severity"`
	DriftType   DriftType       `json:"drift_type"`
	Description string          `json:"description"`
	Details     json.RawMessage `json:"details"`
	Remediation string          `json:"remediation"`
	Confidence  float64         `json:"confidence"`
}

// MCPRequest represents a request to the Model Context Protocol
type MCPRequest struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Data        json.RawMessage `json:"data"`
	Environment string          `json:"environment"`
	Timestamp   time.Time       `json:"timestamp"`
}

// MCPResponse represents a response from the Model Context Protocol
type MCPResponse struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	Error     *string         `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}
