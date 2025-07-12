package database

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/pkg/models"
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

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.DBName)

	// Create indexes for optimized snapshots
	if err := createIndexes(db); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &Database{
		client:   client,
		database: db,
		config:   cfg,
	}, nil
}

// createIndexes creates necessary indexes for performance
func createIndexes(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Indexes for optimized_snapshots
	optimizedSnapshotsColl := db.Collection("optimized_snapshots")
	_, err := optimizedSnapshotsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "namespace", Value: 1},
			{Key: "kind", Value: 1},
			{Key: "name", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create optimized snapshots index: %w", err)
	}

	// Indexes for drift_records
	driftRecordsColl := db.Collection("drift_records")
	_, err = driftRecordsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "resource_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("failed to create drift records index: %w", err)
	}

	// Index for drift detection queries
	_, err = driftRecordsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "drift_status", Value: 1},
			{Key: "last_updated", Value: -1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create drift status index: %w", err)
	}

	// Index for drift resolution queries
	_, err = driftRecordsColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "drift_detected", Value: 1},
			{Key: "last_updated", Value: -1},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create drift detection index: %w", err)
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

// generateHash generates a SHA256 hash of the desired state for tracking
func (d *Database) generateHash(state map[string]interface{}) string {
	if state == nil {
		return ""
	}

	data, err := json.Marshal(state)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash[:])
}

// SaveDriftRecord saves or updates a drift record with enhanced state tracking
func (d *Database) SaveDriftRecord(record *models.DriftRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate required fields
	if record.ResourceID == "" {
		return fmt.Errorf("resource_id cannot be empty")
	}

	coll := d.database.Collection("drift_records")

	// Get existing record to determine state transitions
	existingRecord, err := d.GetDriftRecord(record.ResourceID)

	// Set timestamps
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.LastUpdated = time.Now()

	// Generate hash for desired state
	if record.DesiredState != nil {
		record.LastKnownGoodHash = d.generateHash(record.DesiredState)
	}

	// Handle state transitions
	if existingRecord != nil {
		// Check if drift was resolved
		if existingRecord.DriftStatus == models.DriftStatusActive && !record.DriftDetected {
			record.DriftStatus = models.DriftStatusResolved
			now := time.Now()
			record.ResolvedAt = &now
			record.ResolutionMessage = "Drift resolved. Configuration now matches Git."
		} else if record.DriftDetected {
			// Drift is active
			record.DriftStatus = models.DriftStatusActive
			if existingRecord.DriftStatus != models.DriftStatusActive {
				// First time detecting drift
				now := time.Now()
				record.FirstDetected = &now
			} else {
				// Continue existing drift
				record.FirstDetected = existingRecord.FirstDetected
			}
		} else {
			// No drift
			record.DriftStatus = models.DriftStatusNone
		}
	} else {
		// New record
		if record.DriftDetected {
			record.DriftStatus = models.DriftStatusActive
			now := time.Now()
			record.FirstDetected = &now
		} else {
			record.DriftStatus = models.DriftStatusNone
		}
	}

	// Use upsert to update existing record or create new one
	filter := bson.M{"resource_id": record.ResourceID}
	update := bson.M{
		"$set": bson.M{
			"kind":                 record.Kind,
			"namespace":            record.Namespace,
			"name":                 record.Name,
			"drift_detected":       record.DriftDetected,
			"drift_status":         record.DriftStatus,
			"drift_details":        record.DriftDetails,
			"desired_state":        record.DesiredState,
			"live_state":           record.LiveState,
			"last_known_good_hash": record.LastKnownGoodHash,
			"last_updated":         record.LastUpdated,
		},
		"$setOnInsert": bson.M{
			"_id":        record.ID,
			"created_at": record.CreatedAt,
		},
	}

	// Add conditional updates for drift state
	if record.FirstDetected != nil {
		update["$set"].(bson.M)["first_detected"] = record.FirstDetected
	}
	if record.ResolvedAt != nil {
		update["$set"].(bson.M)["resolved_at"] = record.ResolvedAt
	}
	if record.ResolutionMessage != "" {
		update["$set"].(bson.M)["resolution_message"] = record.ResolutionMessage
	}

	opts := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetDriftRecord retrieves a drift record for a specific resource
func (d *Database) GetDriftRecord(resourceID string) (*models.DriftRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_records")
	filter := bson.M{"resource_id": resourceID}

	var record models.DriftRecord
	err := coll.FindOne(ctx, filter).Decode(&record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetDriftRecords retrieves drift records with optional filtering
func (d *Database) GetDriftRecords(namespace string, driftDetected *bool, driftStatus *models.DriftStatus) ([]*models.DriftRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_records")
	filter := bson.M{}

	if namespace != "" {
		filter["namespace"] = namespace
	}
	if driftDetected != nil {
		filter["drift_detected"] = *driftDetected
	}
	if driftStatus != nil {
		filter["drift_status"] = *driftStatus
	}

	cur, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []*models.DriftRecord
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// GetActiveDrifts retrieves all currently active drift records
func (d *Database) GetActiveDrifts() ([]*models.DriftRecord, error) {
	activeStatus := models.DriftStatusActive
	return d.GetDriftRecords("", nil, &activeStatus)
}

// GetResolvedDrifts retrieves all resolved drift records
func (d *Database) GetResolvedDrifts() ([]*models.DriftRecord, error) {
	resolvedStatus := models.DriftStatusResolved
	return d.GetDriftRecords("", nil, &resolvedStatus)
}

// GetDriftStatistics returns enhanced statistics about drift detection
func (d *Database) GetDriftStatistics() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("drift_records")

	// Total records
	totalCount, err := coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// Records with active drift
	activeDriftCount, err := coll.CountDocuments(ctx, bson.M{"drift_status": models.DriftStatusActive})
	if err != nil {
		return nil, err
	}

	// Records with resolved drift
	resolvedDriftCount, err := coll.CountDocuments(ctx, bson.M{"drift_status": models.DriftStatusResolved})
	if err != nil {
		return nil, err
	}

	// Records without drift
	noDriftCount, err := coll.CountDocuments(ctx, bson.M{"drift_status": models.DriftStatusNone})
	if err != nil {
		return nil, err
	}

	// Recent active drift (last 24 hours)
	yesterday := time.Now().Add(-24 * time.Hour)
	recentActiveDriftCount, err := coll.CountDocuments(ctx, bson.M{
		"drift_status": models.DriftStatusActive,
		"last_updated": bson.M{"$gte": yesterday},
	})
	if err != nil {
		return nil, err
	}

	// Recent resolutions (last 24 hours)
	recentResolvedCount, err := coll.CountDocuments(ctx, bson.M{
		"drift_status": models.DriftStatusResolved,
		"resolved_at":  bson.M{"$gte": yesterday},
	})
	if err != nil {
		return nil, err
	}

	// Calculate percentages
	var activePercentage, resolvedPercentage float64
	if totalCount > 0 {
		activePercentage = float64(activeDriftCount) / float64(totalCount) * 100
		resolvedPercentage = float64(resolvedDriftCount) / float64(totalCount) * 100
	}

	return map[string]interface{}{
		"total_records":       totalCount,
		"active_drift":        activeDriftCount,
		"resolved_drift":      resolvedDriftCount,
		"no_drift":            noDriftCount,
		"recent_active_drift": recentActiveDriftCount,
		"recent_resolutions":  recentResolvedCount,
		"active_percentage":   activePercentage,
		"resolved_percentage": resolvedPercentage,
	}, nil
}
