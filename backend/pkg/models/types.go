package models

import (
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceChange represents a change to a Kubernetes resource
type ResourceChange struct {
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
	Changes   []string  `json:"changes" bson:"changes"`
	User      string    `json:"user" bson:"user"`
	Source    string    `json:"source" bson:"source"`
	EventType string    `json:"event_type" bson:"event_type"` // "created", "updated", "deleted"
}

// OptimizedResourceSnapshot represents an optimized snapshot with change history
type OptimizedResourceSnapshot struct {
	ID           uuid.UUID              `json:"id" bson:"id"`
	Kind         string                 `json:"kind" bson:"kind"`
	Namespace    string                 `json:"namespace" bson:"namespace"`
	Name         string                 `json:"name" bson:"name"`
	CurrentState map[string]interface{} `json:"current_state" bson:"current_state"`
	UpdateLog    []ResourceChange       `json:"update_log" bson:"update_log"`
	CreatedAt    time.Time              `json:"created_at" bson:"created_at"`
	LastUpdated  time.Time              `json:"last_updated" bson:"last_updated"`
}

// KubernetesResource represents a Kubernetes resource with metadata
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
