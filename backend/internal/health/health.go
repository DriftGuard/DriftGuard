package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Component represents a health-checkable component
type Component struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	LastCheck time.Time              `json:"last_check"`
}

// HealthChecker defines the interface for health-checkable components
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) (Status, string, map[string]interface{})
}

// HealthService manages health checks for all components
type HealthService struct {
	components map[string]HealthChecker
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewHealthService creates a new health service
func NewHealthService(logger *zap.Logger) *HealthService {
	return &HealthService{
		components: make(map[string]HealthChecker),
		logger:     logger,
	}
}

// RegisterComponent adds a component to be monitored
func (h *HealthService) RegisterComponent(checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.components[checker.Name()] = checker
	h.logger.Info("Registered health check component", zap.String("component", checker.Name()))
}

// CheckAll performs health checks on all registered components
func (h *HealthService) CheckAll(ctx context.Context) map[string]Component {
	h.mu.RLock()
	defer h.mu.RUnlock()

	results := make(map[string]Component)

	for name, checker := range h.components {
		status, message, details := checker.Check(ctx)
		results[name] = Component{
			Name:      name,
			Status:    status,
			Message:   message,
			Details:   details,
			LastCheck: time.Now(),
		}
	}

	return results
}

// GetOverallStatus determines the overall health status
func (h *HealthService) GetOverallStatus(ctx context.Context) (Status, string) {
	components := h.CheckAll(ctx)

	unhealthyCount := 0
	degradedCount := 0

	for _, comp := range components {
		switch comp.Status {
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		}
	}

	if unhealthyCount > 0 {
		return StatusUnhealthy, fmt.Sprintf("%d components unhealthy", unhealthyCount)
	}
	if degradedCount > 0 {
		return StatusDegraded, fmt.Sprintf("%d components degraded", degradedCount)
	}
	return StatusHealthy, "all components healthy"
}

// ServeHTTP handles HTTP health check requests
func (h *HealthService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if this is a readiness probe
	isReadiness := r.URL.Query().Get("readiness") == "true"

	var status Status
	var message string

	if isReadiness {
		// For readiness, check all components
		status, message = h.GetOverallStatus(ctx)
	} else {
		// For liveness, just check if the service is running
		status = StatusHealthy
		message = "service is running"
	}

	// Set appropriate HTTP status code
	var httpStatus int
	switch status {
	case StatusHealthy:
		httpStatus = http.StatusOK
	case StatusDegraded:
		httpStatus = http.StatusOK // Still OK but degraded
	case StatusUnhealthy:
		httpStatus = http.StatusServiceUnavailable
	}

	// Prepare response
	response := map[string]interface{}{
		"status":  status,
		"message": message,
		"time":    time.Now().UTC(),
	}

	if isReadiness {
		response["components"] = h.CheckAll(ctx)
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	// Encode response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
