package main

import (
	"context"
	"testing"

	"github.com/DriftGuard/core/internal/lifecycle"
	"github.com/DriftGuard/core/internal/logger"
)

func TestLifecycleManager(t *testing.T) {
	logger := logger.New()
	defer logger.Sync()

	lm := lifecycle.NewLifecycleManager(logger)

	// Test component registration
	component := &lifecycleComponent{
		name:   "test-component",
		start:  func(ctx context.Context) error { return nil },
		stop:   func(ctx context.Context) error { return nil },
		health: func(ctx context.Context) error { return nil },
	}

	lm.RegisterComponent(component)

	// Test starting lifecycle
	ctx := context.Background()
	err := lm.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start lifecycle: %v", err)
	}

	// Verify status
	if lm.GetStatus() != lifecycle.StatusRunning {
		t.Errorf("Expected status %s, got %s", lifecycle.StatusRunning, lm.GetStatus())
	}

	// Test health check
	err = lm.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test stopping lifecycle
	err = lm.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop lifecycle: %v", err)
	}

	// Verify status
	if lm.GetStatus() != lifecycle.StatusStopped {
		t.Errorf("Expected status %s, got %s", lifecycle.StatusStopped, lm.GetStatus())
	}
}

func TestLifecycleComponent(t *testing.T) {
	lc := &lifecycleComponent{
		name:   "test",
		start:  func(ctx context.Context) error { return nil },
		stop:   func(ctx context.Context) error { return nil },
		health: func(ctx context.Context) error { return nil },
	}

	if lc.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", lc.Name())
	}

	ctx := context.Background()
	if err := lc.Start(ctx); err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if err := lc.Health(ctx); err != nil {
		t.Errorf("Health failed: %v", err)
	}

	if err := lc.Stop(ctx); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}
