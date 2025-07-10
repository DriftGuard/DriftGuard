package controller

import (
	"context"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/internal/database"
	"github.com/DriftGuard/core/pkg/models"
	"go.uber.org/zap"
)

// DriftController coordinates drift detection and management
type DriftController struct {
	config    *config.Config
	db        *database.Database
	logger    *zap.Logger
	stopCh    chan struct{}
	isRunning bool
}

// NewDriftController creates a new drift controller instance
func NewDriftController(cfg *config.Config, db *database.Database, logger *zap.Logger) *DriftController {
	return &DriftController{
		config:  cfg,
		db:      db,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the drift detection process
func (c *DriftController) Start(ctx context.Context) error {
	if c.isRunning {
		c.logger.Warn("DriftController is already running")
		return nil
	}

	c.logger.Info("Starting DriftController")
	c.isRunning = true

	// Start periodic drift analysis
	go c.runDriftAnalysis(ctx)

	c.logger.Info("DriftController started successfully")
	return nil
}

// Stop gracefully stops the drift controller
func (c *DriftController) Stop() error {
	if !c.isRunning {
		c.logger.Warn("DriftController is not running")
		return nil
	}

	c.logger.Info("Stopping DriftController")
	c.isRunning = false
	close(c.stopCh)

	c.logger.Info("DriftController stopped successfully")
	return nil
}

// runDriftAnalysis runs periodic drift analysis
func (c *DriftController) runDriftAnalysis(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	c.logger.Info("Starting periodic drift analysis")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Drift analysis stopped due to context cancellation")
			return
		case <-c.stopCh:
			c.logger.Info("Drift analysis stopped due to controller shutdown")
			return
		case <-ticker.C:
			c.performDriftAnalysis()
		}
	}
}

// performDriftAnalysis performs a single drift analysis cycle
func (c *DriftController) performDriftAnalysis() {
	c.logger.Debug("Performing drift analysis")

	// Get all snapshots for analysis
	snapshots, err := c.db.GetOptimizedSnapshots("")
	if err != nil {
		c.logger.Error("Failed to get snapshots for analysis", zap.Error(err))
		return
	}

	c.logger.Info("Analyzing snapshots", zap.Int("count", len(snapshots)))

	// For now, just log the analysis
	// In the future, this would compare with Git desired state
	for _, snapshot := range snapshots {
		c.logger.Debug("Analyzing snapshot",
			zap.String("kind", snapshot.Kind),
			zap.String("namespace", snapshot.Namespace),
			zap.String("name", snapshot.Name),
			zap.Int("update_count", len(snapshot.UpdateLog)))
	}
}

// GetSnapshots retrieves snapshots for a namespace
func (c *DriftController) GetSnapshots(namespace string) ([]*models.OptimizedResourceSnapshot, error) {
	return c.db.GetOptimizedSnapshots(namespace)
}

// GetSnapshot retrieves a specific snapshot
func (c *DriftController) GetSnapshot(namespace, kind, name string) (*models.OptimizedResourceSnapshot, error) {
	return c.db.GetOptimizedSnapshot(namespace, kind, name)
}
