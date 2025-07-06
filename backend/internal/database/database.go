package database

import (
	"context"
	"fmt"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/pkg/models"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database represents a MongoDB-backed database layer
type Database struct {
	client   *mongo.Client
	database *mongo.Database
	config   config.DatabaseConfig
}

// DriftStats represents drift statistics for reporting
type DriftStats struct {
	TotalDriftEvents      int64                    `json:"total_drift_events"`
	OpenDriftEvents       int64                    `json:"open_drift_events"`
	ResolvedDriftEvents   int64                    `json:"resolved_drift_events"`
	DriftEventsByType     map[string]int64         `json:"drift_events_by_type"`
	DriftEventsBySeverity map[string]int64         `json:"drift_events_by_severity"`
	AverageDriftScore     float64                  `json:"average_drift_score"`
	TopDriftResources     []map[string]interface{} `json:"top_drift_resources"`
	TimeRange             time.Duration            `json:"time_range"`
	Environment           string                   `json:"environment"`
}

// DatabaseMetrics represents database performance metrics
type DatabaseMetrics struct {
	ActiveConnections int64     `json:"active_connections"`
	TotalOperations   int64     `json:"total_operations"`
	AverageLatency    float64   `json:"average_latency_ms"`
	ErrorRate         float64   `json:"error_rate"`
	LastHealthCheck   time.Time `json:"last_health_check"`
}

// New creates a new MongoDB connection and returns a Database instance
func New(cfg config.DatabaseConfig) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var uri string
	if cfg.User != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.DBName,
		)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d/%s",
			cfg.Host,
			cfg.Port,
			cfg.DBName,
		)
	}

	// Configure connection pooling and retry logic
	clientOpts := options.Client().ApplyURI(uri)
	clientOpts.SetMaxPoolSize(100)
	clientOpts.SetMinPoolSize(5)
	clientOpts.SetMaxConnIdleTime(30 * time.Second)
	clientOpts.SetRetryWrites(true)
	clientOpts.SetRetryReads(true)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test connection with retry logic
	if err := pingWithRetry(client, 3); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.DBName)

	// Create indexes for better performance
	if err := createIndexes(db); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &Database{
		client:   client,
		database: db,
		config:   cfg,
	}, nil
}

// pingWithRetry attempts to ping MongoDB with retry logic
func pingWithRetry(client *mongo.Client, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := client.Ping(ctx, nil)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return lastErr
}

// createIndexes creates necessary indexes for performance
func createIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Indexes for configuration_snapshots
	snapshotsColl := db.Collection("configuration_snapshots")
	_, err := snapshotsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "environment", Value: 1},
			{Key: "namespace", Value: 1},
			{Key: "resource_type", Value: 1},
			{Key: "created_at", Value: -1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create snapshots index: %w", err)
	}

	// Indexes for drift_events
	eventsColl := db.Collection("drift_events")
	_, err = eventsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "environment", Value: 1},
			{Key: "namespace", Value: 1},
			{Key: "status", Value: 1},
			{Key: "detected_at", Value: -1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create events index: %w", err)
	}

	// Indexes for environments
	envColl := db.Collection("environments")
	_, err = envColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create environments index: %w", err)
	}

	return nil
}

// Close disconnects the MongoDB client
func (d *Database) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return d.client.Disconnect(ctx)
}

// HealthCheck performs a health check on the database connection
func (d *Database) HealthCheck(ctx context.Context) error {
	return d.client.Ping(ctx, nil)
}

// GetMetrics returns database performance metrics
func (d *Database) GetMetrics() (*DatabaseMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get server status for metrics
	cmd := map[string]interface{}{"serverStatus": 1}
	var result map[string]interface{}
	err := d.database.RunCommand(ctx, cmd).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get server status: %w", err)
	}

	metrics := &DatabaseMetrics{
		LastHealthCheck: time.Now(),
	}

	// Extract connection metrics
	if connections, ok := result["connections"].(map[string]interface{}); ok {
		if active, ok := connections["active"].(int32); ok {
			metrics.ActiveConnections = int64(active)
		}
	}

	// Extract operation metrics
	if opcounters, ok := result["opcounters"].(map[string]interface{}); ok {
		var totalOps int64
		for _, count := range opcounters {
			if countInt, ok := count.(int32); ok {
				totalOps += int64(countInt)
			}
		}
		metrics.TotalOperations = totalOps
	}

	return metrics, nil
}

// SaveSnapshot stores a new configuration snapshot
func (d *Database) SaveSnapshot(snapshot *models.ConfigurationSnapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("configuration_snapshots")
	_, err := coll.InsertOne(ctx, snapshot)
	return err
}

// GetSnapshots retrieves snapshots for a specific environment and namespace
func (d *Database) GetSnapshots(env, namespace string) ([]*models.ConfigurationSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("configuration_snapshots")
	filter := map[string]interface{}{"environment": env, "namespace": namespace}
	cur, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []*models.ConfigurationSnapshot
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// SaveDriftEvent stores a new drift event
func (d *Database) SaveDriftEvent(event *models.DriftEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_events")
	_, err := coll.InsertOne(ctx, event)
	return err
}

// GetDriftEvents retrieves drift events with optional filtering
func (d *Database) GetDriftEvents(filters map[string]interface{}) ([]*models.DriftEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_events")
	cur, err := coll.Find(ctx, filters)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []*models.DriftEvent
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateDriftEventStatus updates the status of a drift event
func (d *Database) UpdateDriftEventStatus(id uuid.UUID, status models.EventStatus) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_events")
	filter := map[string]interface{}{"id": id}
	update := map[string]interface{}{
		"$set": map[string]interface{}{
			"status":      status,
			"resolved_at": time.Now(),
		},
	}
	_, err := coll.UpdateOne(ctx, filter, update)
	return err
}

// GetEnvironmentByName retrieves an environment by name
func (d *Database) GetEnvironmentByName(name string) (*models.Environment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("environments")
	filter := map[string]interface{}{"name": name}
	var env models.Environment
	err := coll.FindOne(ctx, filter).Decode(&env)
	if err != nil {
		return nil, err
	}
	return &env, nil
}

// SaveEnvironment stores or updates an environment configuration
func (d *Database) SaveEnvironment(env *models.Environment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("environments")
	filter := map[string]interface{}{"name": env.Name}
	update := map[string]interface{}{"$set": env}
	_, err := coll.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// GetDriftStatistics returns drift statistics for reporting
func (d *Database) GetDriftStatistics(env string, timeRange time.Duration) (*DriftStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_events")

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.Add(-timeRange)

	// Base filter
	filter := map[string]interface{}{
		"environment": env,
		"detected_at": map[string]interface{}{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	// Get total drift events
	totalCount, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count total drift events: %w", err)
	}

	// Get open drift events
	openFilter := map[string]interface{}{
		"environment": env,
		"detected_at": map[string]interface{}{
			"$gte": startTime,
			"$lte": endTime,
		},
		"status": models.EventStatusOpen,
	}
	openCount, err := coll.CountDocuments(ctx, openFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count open drift events: %w", err)
	}

	// Get resolved drift events
	resolvedFilter := map[string]interface{}{
		"environment": env,
		"detected_at": map[string]interface{}{
			"$gte": startTime,
			"$lte": endTime,
		},
		"status": models.EventStatusResolved,
	}
	resolvedCount, err := coll.CountDocuments(ctx, resolvedFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count resolved drift events: %w", err)
	}

	// Get drift events by type
	typePipeline := []map[string]interface{}{
		{"$match": filter},
		{"$group": map[string]interface{}{
			"_id":   "$drift_type",
			"count": map[string]interface{}{"$sum": 1},
		}},
	}
	typeCursor, err := coll.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate drift events by type: %w", err)
	}
	defer typeCursor.Close(ctx)

	driftEventsByType := make(map[string]int64)
	for typeCursor.Next(ctx) {
		var result map[string]interface{}
		if err := typeCursor.Decode(&result); err != nil {
			continue
		}
		if driftType, ok := result["_id"].(string); ok {
			if count, ok := result["count"].(int32); ok {
				driftEventsByType[driftType] = int64(count)
			}
		}
	}

	// Get drift events by severity
	severityPipeline := []map[string]interface{}{
		{"$match": filter},
		{"$group": map[string]interface{}{
			"_id":   "$severity",
			"count": map[string]interface{}{"$sum": 1},
		}},
	}
	severityCursor, err := coll.Aggregate(ctx, severityPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate drift events by severity: %w", err)
	}
	defer severityCursor.Close(ctx)

	driftEventsBySeverity := make(map[string]int64)
	for severityCursor.Next(ctx) {
		var result map[string]interface{}
		if err := severityCursor.Decode(&result); err != nil {
			continue
		}
		if severity, ok := result["_id"].(string); ok {
			if count, ok := result["count"].(int32); ok {
				driftEventsBySeverity[severity] = int64(count)
			}
		}
	}

	// Get average drift score
	scorePipeline := []map[string]interface{}{
		{"$match": filter},
		{"$group": map[string]interface{}{
			"_id":      nil,
			"avgScore": map[string]interface{}{"$avg": "$drift_score"},
		}},
	}
	scoreCursor, err := coll.Aggregate(ctx, scorePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate average drift score: %w", err)
	}
	defer scoreCursor.Close(ctx)

	var avgScore float64
	if scoreCursor.Next(ctx) {
		var result map[string]interface{}
		if err := scoreCursor.Decode(&result); err == nil {
			if score, ok := result["avgScore"].(float64); ok {
				avgScore = score
			}
		}
	}

	// Get top drift resources
	topResourcesPipeline := []map[string]interface{}{
		{"$match": filter},
		{"$group": map[string]interface{}{
			"_id": map[string]interface{}{
				"resource_type": "$resource_type",
				"resource_name": "$resource_name",
			},
			"count":    map[string]interface{}{"$sum": 1},
			"avgScore": map[string]interface{}{"$avg": "$drift_score"},
		}},
		{"$sort": map[string]interface{}{"count": -1}},
		{"$limit": 10},
	}
	topResourcesCursor, err := coll.Aggregate(ctx, topResourcesPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get top drift resources: %w", err)
	}
	defer topResourcesCursor.Close(ctx)

	var topDriftResources []map[string]interface{}
	for topResourcesCursor.Next(ctx) {
		var result map[string]interface{}
		if err := topResourcesCursor.Decode(&result); err != nil {
			continue
		}
		topDriftResources = append(topDriftResources, result)
	}

	return &DriftStats{
		TotalDriftEvents:      totalCount,
		OpenDriftEvents:       openCount,
		ResolvedDriftEvents:   resolvedCount,
		DriftEventsByType:     driftEventsByType,
		DriftEventsBySeverity: driftEventsBySeverity,
		AverageDriftScore:     avgScore,
		TopDriftResources:     topDriftResources,
		TimeRange:             timeRange,
		Environment:           env,
	}, nil
}
