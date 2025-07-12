package models

import (
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DriftStatus represents the current status of a drift
type DriftStatus string

const (
	DriftStatusActive   DriftStatus = "active"
	DriftStatusResolved DriftStatus = "resolved"
	DriftStatusNone     DriftStatus = "none"
)

// DriftChange represents a specific field change detected in drift
type DriftChange struct {
	Field    string      `json:"field" bson:"field"`
	From     interface{} `json:"from" bson:"from"`
	To       interface{} `json:"to" bson:"to"`
	Type     string      `json:"type" bson:"type"`
	Severity string      `json:"severity" bson:"severity"`
}

// DriftResult represents the result of a drift detection analysis
type DriftResult struct {
	Detected      bool          `json:"detected" bson:"detected"`
	Changes       []DriftChange `json:"changes" bson:"changes"`
	LastEvaluated time.Time     `json:"last_evaluated" bson:"last_evaluated"`
	ResourceID    string        `json:"resource_id" bson:"resource_id"`
	DesiredState  interface{}   `json:"desired_state" bson:"desired_state"`
	LiveState     interface{}   `json:"live_state" bson:"live_state"`
}

// DriftRecord represents a complete drift record with enhanced state tracking
type DriftRecord struct {
	ID                uuid.UUID              `json:"id" bson:"_id"`
	ResourceID        string                 `json:"resource_id" bson:"resource_id"`
	Kind              string                 `json:"kind" bson:"kind"`
	Namespace         string                 `json:"namespace" bson:"namespace"`
	Name              string                 `json:"name" bson:"name"`
	DriftDetected     bool                   `json:"drift_detected" bson:"drift_detected"`
	DriftStatus       DriftStatus            `json:"drift_status" bson:"drift_status"`
	DriftDetails      []DriftChange          `json:"drift_details" bson:"drift_details"`
	DesiredState      map[string]interface{} `json:"desired_state" bson:"desired_state"`
	LiveState         map[string]interface{} `json:"live_state" bson:"live_state"`
	LastKnownGoodHash string                 `json:"last_known_good_hash" bson:"last_known_good_hash"`
	FirstDetected     *time.Time             `json:"first_detected" bson:"first_detected"`
	ResolvedAt        *time.Time             `json:"resolved_at" bson:"resolved_at"`
	ResolutionMessage string                 `json:"resolution_message" bson:"resolution_message"`
	LastUpdated       time.Time              `json:"last_updated" bson:"last_updated"`
	CreatedAt         time.Time              `json:"created_at" bson:"created_at"`
}

// KubernetesResource represents a Kubernetes resource
type KubernetesResource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   metav1.ObjectMeta      `json:"metadata"`
	Spec       map[string]interface{} `json:"spec,omitempty"`
	Status     map[string]interface{} `json:"status,omitempty"`
}

// Environment represents a deployment environment
type Environment struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	K8sContext  string            `json:"k8s_context" yaml:"k8s_context"`
}
