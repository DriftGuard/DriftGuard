package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/internal/database"
	"github.com/DriftGuard/core/internal/detector"
	"github.com/DriftGuard/core/internal/git"
	"github.com/DriftGuard/core/internal/kubernetes"
	"github.com/DriftGuard/core/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type DriftController struct {
	config    *config.Config
	db        *database.Database
	logger    *zap.Logger
	k8sClient *kubernetes.Client
	gitClient *git.GitClient
	detector  *detector.DriftDetector
	stopCh    chan struct{}
	isRunning bool
}

func NewDriftController(cfg *config.Config, db *database.Database, logger *zap.Logger) (*DriftController, error) {
	k8sClient, err := kubernetes.NewClient(&cfg.Kubernetes, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	gitClient, err := git.NewGitClient(&cfg.Git, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git client: %w", err)
	}

	detector := detector.NewDriftDetector(logger)

	return &DriftController{
		config:    cfg,
		db:        db,
		logger:    logger,
		k8sClient: k8sClient,
		gitClient: gitClient,
		detector:  detector,
		stopCh:    make(chan struct{}),
	}, nil
}

func (c *DriftController) Start(ctx context.Context) error {
	if c.isRunning {
		c.logger.Warn("DriftController is already running")
		return nil
	}

	c.logger.Info("Starting DriftController")

	if err := c.gitClient.Clone(); err != nil {
		c.logger.Error("Failed to clone Git repository", zap.Error(err))
		return fmt.Errorf("failed to clone Git repository: %w", err)
	}

	c.isRunning = true
	go c.runDriftAnalysis(ctx)

	c.logger.Info("DriftController started successfully")
	return nil
}

func (c *DriftController) Stop() error {
	if !c.isRunning {
		c.logger.Warn("DriftController is not running")
		return nil
	}

	c.logger.Info("Stopping DriftController")
	c.isRunning = false
	close(c.stopCh)

	if err := c.gitClient.Cleanup(); err != nil {
		c.logger.Warn("Failed to cleanup Git client", zap.Error(err))
	}

	c.logger.Info("DriftController stopped successfully")
	return nil
}

func (c *DriftController) runDriftAnalysis(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
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

func (c *DriftController) TriggerManualAnalysis() {
	c.logger.Info("Manual drift analysis triggered")
	c.performDriftAnalysis()
}

func (c *DriftController) performDriftAnalysis() {
	c.logger.Debug("Performing drift analysis")

	if err := c.gitClient.Pull(); err != nil {
		c.logger.Error("Failed to pull latest changes from Git", zap.Error(err))
		return
	}

	namespaces, err := c.k8sClient.GetNamespaces()
	if err != nil {
		c.logger.Error("Failed to get namespaces", zap.Error(err))
		return
	}

	resourceTypes := c.config.Kubernetes.Resources
	if len(resourceTypes) == 0 {
		resourceTypes = []string{"deployments", "services", "configmaps", "secrets"}
	}

	c.logger.Info("Starting drift analysis",
		zap.Strings("namespaces", namespaces),
		zap.Strings("resource_types", resourceTypes))

	for _, namespace := range namespaces {
		for _, resourceType := range resourceTypes {
			kind := c.normalizeResourceKind(resourceType)
			c.analyzeResources(namespace, kind)
		}
	}

	c.logger.Info("Drift analysis completed")
}

func (c *DriftController) analyzeResources(namespace, resourceType string) {
	c.logger.Debug("Analyzing resources",
		zap.String("namespace", namespace),
		zap.String("resource_type", resourceType))

	liveResources, err := c.k8sClient.ListResources(resourceType, namespace)
	if err != nil {
		c.logger.Error("Failed to list live resources",
			zap.String("namespace", namespace),
			zap.String("resource_type", resourceType),
			zap.Error(err))
		return
	}

	for _, liveResource := range liveResources {
		c.analyzeSingleResource(liveResource, resourceType, namespace)
	}
}

func (c *DriftController) analyzeSingleResource(liveResource map[string]interface{}, kind, namespace string) {
	metadata := getMapValue(liveResource, "metadata")
	if metadata == nil {
		c.logger.Warn("Resource missing metadata", zap.String("kind", kind), zap.String("namespace", namespace))
		return
	}

	name := getStringValue(metadata, "name")
	if name == "" {
		c.logger.Warn("Resource missing name", zap.String("kind", kind), zap.String("namespace", namespace))
		return
	}

	resourceID := fmt.Sprintf("%s:%s:%s", kind, namespace, name)
	c.logger.Debug("Analyzing resource", zap.String("resource_id", resourceID))

	desiredResource, err := c.gitClient.GetManifestForResource(kind, namespace, name)
	if err != nil {
		c.logger.Warn("Failed to get desired state from Git",
			zap.String("resource_id", resourceID),
			zap.Error(err))
		desiredResource = nil
	}

	var driftResult *models.DriftResult
	if desiredResource != nil {
		driftResult = c.detector.DetectDrift(resourceID, liveResource, desiredResource)
	} else {
		driftResult = &models.DriftResult{
			ResourceID:    resourceID,
			Detected:      false,
			Changes:       []models.DriftChange{},
			LastEvaluated: time.Now(),
			LiveState:     liveResource,
			DesiredState:  nil,
		}
	}

	// Get existing record to determine state transitions
	existingRecord, err := c.db.GetDriftRecord(resourceID)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		c.logger.Error("Failed to get existing drift record",
			zap.String("resource_id", resourceID),
			zap.Error(err))
	}

	driftRecord := &models.DriftRecord{
		ID:            uuid.New(),
		ResourceID:    resourceID,
		Kind:          kind,
		Namespace:     namespace,
		Name:          name,
		DriftDetected: driftResult.Detected,
		DriftDetails:  driftResult.Changes,
		DesiredState:  desiredResource,
		LiveState:     liveResource,
		LastUpdated:   time.Now(),
		CreatedAt:     time.Now(),
	}

	// Handle state transitions and logging
	c.handleDriftStateTransition(existingRecord, driftRecord)

	if err := c.db.SaveDriftRecord(driftRecord); err != nil {
		c.logger.Error("Failed to save drift record",
			zap.String("resource_id", resourceID),
			zap.Error(err))
		return
	}
}

// handleDriftStateTransition handles the logic for drift state changes and logging
func (c *DriftController) handleDriftStateTransition(existingRecord *models.DriftRecord, newRecord *models.DriftRecord) {
	kind := newRecord.Kind
	name := newRecord.Name

	if existingRecord == nil {
		// New resource being tracked
		if newRecord.DriftDetected {
			newRecord.DriftStatus = models.DriftStatusActive
			now := time.Now()
			newRecord.FirstDetected = &now

			// Log drift detection
			c.logDriftDetected(kind, name, newRecord.DriftDetails)
		} else {
			newRecord.DriftStatus = models.DriftStatusNone
		}
		return
	}

	// Existing resource - check for state transitions
	if existingRecord.DriftStatus == models.DriftStatusActive && !newRecord.DriftDetected {
		// Drift resolved
		newRecord.DriftStatus = models.DriftStatusResolved
		now := time.Now()
		newRecord.ResolvedAt = &now
		newRecord.ResolutionMessage = "Drift resolved. Configuration now matches Git."
		newRecord.FirstDetected = existingRecord.FirstDetected

		// Log drift resolution
		c.logDriftResolved(kind, name)

	} else if newRecord.DriftDetected {
		// Drift is active
		newRecord.DriftStatus = models.DriftStatusActive

		if existingRecord.DriftStatus != models.DriftStatusActive {
			// First time detecting drift for this resource
			now := time.Now()
			newRecord.FirstDetected = &now

			// Log drift detection
			c.logDriftDetected(kind, name, newRecord.DriftDetails)
		} else {
			// Continue existing drift
			newRecord.FirstDetected = existingRecord.FirstDetected

			// Log drift continuation if there are new changes
			if len(newRecord.DriftDetails) != len(existingRecord.DriftDetails) {
				c.logDriftContinued(kind, name, newRecord.DriftDetails)
			}
		}
	} else {
		// No drift
		newRecord.DriftStatus = models.DriftStatusNone
	}
}

// logDriftDetected logs when drift is first detected
func (c *DriftController) logDriftDetected(kind, name string, changes []models.DriftChange) {
	c.logger.Warn("⚠️ Drift detected",
		zap.String("resource", fmt.Sprintf("%s/%s", kind, name)),
		zap.Int("changes_count", len(changes)))

	// Log individual field changes
	for _, change := range changes {
		c.logger.Info("⚠️ Drift detected in resource",
			zap.String("resource", fmt.Sprintf("%s/%s", kind, name)),
			zap.String("field", change.Field),
			zap.Any("from", change.From),
			zap.Any("to", change.To),
			zap.String("type", change.Type),
			zap.String("severity", change.Severity))
	}
}

// logDriftResolved logs when drift is resolved
func (c *DriftController) logDriftResolved(kind, name string) {
	c.logger.Info("✅ Drift resolved",
		zap.String("resource", fmt.Sprintf("%s/%s", kind, name)),
		zap.String("message", "Configuration now matches Git desired state"))
}

// logDriftContinued logs when drift continues with new changes
func (c *DriftController) logDriftContinued(kind, name string, changes []models.DriftChange) {
	c.logger.Warn("⚠️ Drift continued",
		zap.String("resource", fmt.Sprintf("%s/%s", kind, name)),
		zap.Int("changes_count", len(changes)))

	// Log new field changes
	for _, change := range changes {
		c.logger.Info("⚠️ Drift continued in resource",
			zap.String("resource", fmt.Sprintf("%s/%s", kind, name)),
			zap.String("field", change.Field),
			zap.Any("from", change.From),
			zap.Any("to", change.To),
			zap.String("type", change.Type),
			zap.String("severity", change.Severity))
	}
}

func (c *DriftController) GetDriftRecord(resourceID string) (*models.DriftRecord, error) {
	return c.db.GetDriftRecord(resourceID)
}

func (c *DriftController) GetDriftRecords(namespace string, driftDetected *bool) ([]*models.DriftRecord, error) {
	return c.db.GetDriftRecords(namespace, driftDetected, nil)
}

func (c *DriftController) GetDriftRecordsByStatus(namespace string, driftStatus *models.DriftStatus) ([]*models.DriftRecord, error) {
	return c.db.GetDriftRecords(namespace, nil, driftStatus)
}

func (c *DriftController) GetActiveDrifts() ([]*models.DriftRecord, error) {
	return c.db.GetActiveDrifts()
}

func (c *DriftController) GetResolvedDrifts() ([]*models.DriftRecord, error) {
	return c.db.GetResolvedDrifts()
}

func (c *DriftController) GetDriftStatistics() (map[string]interface{}, error) {
	return c.db.GetDriftStatistics()
}

func (c *DriftController) normalizeResourceKind(resourceType string) string {
	kindMappings := map[string]string{
		"deployments": "Deployment",
		"services":    "Service",
		"configmaps":  "ConfigMap",
		"secrets":     "Secret",
		"pods":        "Pod",
		"ingresses":   "Ingress",
		"jobs":        "Job",
		"cronjobs":    "CronJob",
	}

	if kind, exists := kindMappings[resourceType]; exists {
		return kind
	}
	return resourceType
}

func getStringValue(data map[string]interface{}, path string) string {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		if val, ok := current[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
			if mapVal, ok := val.(map[string]interface{}); ok {
				current = mapVal
			} else {
				return ""
			}
		} else {
			return ""
		}
	}

	return ""
}

func getMapValue(data map[string]interface{}, path string) map[string]interface{} {
	keys := strings.Split(path, ".")
	current := data

	for _, key := range keys {
		if val, ok := current[key]; ok {
			if mapVal, ok := val.(map[string]interface{}); ok {
				current = mapVal
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}
