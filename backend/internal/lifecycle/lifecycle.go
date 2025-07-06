package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Component represents a lifecycle-managed component
type Component interface {
	Name() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
}

// LifecycleManager manages the application lifecycle
type LifecycleManager struct {
	components []Component
	logger     *zap.Logger
	startTime  time.Time
	mu         sync.RWMutex
	status     Status
}

// Status represents the application status
type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusStopped  Status = "stopped"
	StatusError    Status = "error"
)

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(logger *zap.Logger) *LifecycleManager {
	return &LifecycleManager{
		components: make([]Component, 0),
		logger:     logger,
		status:     StatusStopped,
	}
}

// RegisterComponent adds a component to be managed
func (lm *LifecycleManager) RegisterComponent(component Component) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.components = append(lm.components, component)
	lm.logger.Info("Registered lifecycle component", zap.String("component", component.Name()))
}

// Start starts all components in order
func (lm *LifecycleManager) Start(ctx context.Context) error {
	lm.mu.Lock()
	if lm.status == StatusStarting || lm.status == StatusRunning {
		lm.mu.Unlock()
		return fmt.Errorf("lifecycle manager is already %s", lm.status)
	}
	lm.status = StatusStarting
	lm.startTime = time.Now()
	lm.mu.Unlock()

	lm.logger.Info("Starting application lifecycle", zap.Time("start_time", lm.startTime))

	// Start components in order
	for _, component := range lm.components {
		lm.logger.Info("Starting component", zap.String("component", component.Name()))

		if err := component.Start(ctx); err != nil {
			lm.logger.Error("Failed to start component",
				zap.String("component", component.Name()),
				zap.Error(err))

			// Try to stop already started components
			lm.stopComponents(ctx, component.Name())
			lm.setStatus(StatusError)
			return fmt.Errorf("failed to start component %s: %w", component.Name(), err)
		}

		lm.logger.Info("Component started successfully", zap.String("component", component.Name()))
	}

	lm.setStatus(StatusRunning)
	lm.logger.Info("Application lifecycle started successfully",
		zap.Duration("startup_duration", time.Since(lm.startTime)))

	return nil
}

// Stop stops all components in reverse order
func (lm *LifecycleManager) Stop(ctx context.Context) error {
	lm.mu.Lock()
	if lm.status == StatusStopping || lm.status == StatusStopped {
		lm.mu.Unlock()
		return fmt.Errorf("lifecycle manager is already %s", lm.status)
	}
	lm.status = StatusStopping
	lm.mu.Unlock()

	lm.logger.Info("Stopping application lifecycle")

	// Stop components in reverse order
	for i := len(lm.components) - 1; i >= 0; i-- {
		component := lm.components[i]
		lm.logger.Info("Stopping component", zap.String("component", component.Name()))

		if err := component.Stop(ctx); err != nil {
			lm.logger.Error("Failed to stop component",
				zap.String("component", component.Name()),
				zap.Error(err))
			// Continue stopping other components
		} else {
			lm.logger.Info("Component stopped successfully", zap.String("component", component.Name()))
		}
	}

	lm.setStatus(StatusStopped)
	lm.logger.Info("Application lifecycle stopped successfully")

	return nil
}

// Health checks the health of all components
func (lm *LifecycleManager) Health(ctx context.Context) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if lm.status != StatusRunning {
		return fmt.Errorf("application is not running, status: %s", lm.status)
	}

	var errors []error
	for _, component := range lm.components {
		if err := component.Health(ctx); err != nil {
			errors = append(errors, fmt.Errorf("component %s: %w", component.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("health check failed: %v", errors)
	}

	return nil
}

// GetStatus returns the current application status
func (lm *LifecycleManager) GetStatus() Status {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.status
}

// GetUptime returns the application uptime
func (lm *LifecycleManager) GetUptime() time.Duration {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if lm.status == StatusStopped || lm.status == StatusError {
		return 0
	}

	return time.Since(lm.startTime)
}

// GetStartTime returns the application start time
func (lm *LifecycleManager) GetStartTime() time.Time {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.startTime
}

// stopComponents stops components up to the given component name
func (lm *LifecycleManager) stopComponents(ctx context.Context, stopAt string) {
	for i := len(lm.components) - 1; i >= 0; i-- {
		component := lm.components[i]
		if component.Name() == stopAt {
			break
		}

		lm.logger.Info("Stopping component during startup failure",
			zap.String("component", component.Name()))

		if err := component.Stop(ctx); err != nil {
			lm.logger.Error("Failed to stop component during startup failure",
				zap.String("component", component.Name()),
				zap.Error(err))
		}
	}
}

// setStatus sets the application status
func (lm *LifecycleManager) setStatus(status Status) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.status = status
}
