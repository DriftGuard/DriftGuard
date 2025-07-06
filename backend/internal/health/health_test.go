package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// mockHealthChecker implements HealthChecker for testing
type mockHealthChecker struct {
	name   string
	status Status
	msg    string
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Check(ctx context.Context) (Status, string, map[string]interface{}) {
	return m.status, m.msg, map[string]interface{}{"test": "data"}
}

func TestHealthService(t *testing.T) {
	logger := zap.NewNop()
	hs := NewHealthService(logger)

	// Test registering components
	healthyComp := &mockHealthChecker{
		name:   "healthy-component",
		status: StatusHealthy,
		msg:    "All good",
	}

	unhealthyComp := &mockHealthChecker{
		name:   "unhealthy-component",
		status: StatusUnhealthy,
		msg:    "Something is wrong",
	}

	hs.RegisterComponent(healthyComp)
	hs.RegisterComponent(unhealthyComp)

	// Test CheckAll
	components := hs.CheckAll(context.Background())
	if len(components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(components))
	}

	// Test GetOverallStatus
	status, _ := hs.GetOverallStatus(context.Background())
	if status != StatusUnhealthy {
		t.Errorf("Expected status %s, got %s", StatusUnhealthy, status)
	}

	// Test HTTP handler
	req := httptest.NewRequest("GET", "/health?readiness=true", nil)
	w := httptest.NewRecorder()
	hs.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHealthServiceWithOnlyHealthyComponents(t *testing.T) {
	logger := zap.NewNop()
	hs := NewHealthService(logger)

	healthyComp := &mockHealthChecker{
		name:   "healthy-component",
		status: StatusHealthy,
		msg:    "All good",
	}

	hs.RegisterComponent(healthyComp)

	status, _ := hs.GetOverallStatus(context.Background())
	if status != StatusHealthy {
		t.Errorf("Expected status %s, got %s", StatusHealthy, status)
	}

	// Test HTTP handler for healthy state
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	hs.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}
