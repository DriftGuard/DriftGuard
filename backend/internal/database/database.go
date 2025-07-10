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

// SaveOptimizedSnapshot saves or updates an optimized resource snapshot with change history
func (d *Database) SaveOptimizedSnapshot(snapshot *models.OptimizedResourceSnapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate required fields to prevent null values in unique index
	if snapshot.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if snapshot.Kind == "" {
		return fmt.Errorf("kind cannot be empty")
	}
	if snapshot.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	coll := d.database.Collection("optimized_snapshots")

	// Check if resource already exists
	filter := bson.M{
		"namespace": snapshot.Namespace,
		"kind":      snapshot.Kind,
		"name":      snapshot.Name,
	}

	var existingSnapshot models.OptimizedResourceSnapshot
	err := coll.FindOne(ctx, filter).Decode(&existingSnapshot)

	if err != nil {
		// Resource doesn't exist, create new snapshot
		snapshot.ID = uuid.New()
		snapshot.CreatedAt = time.Now()
		snapshot.LastUpdated = time.Now()
		_, err = coll.InsertOne(ctx, snapshot)
		return err
	}

	// Resource exists, update it with new change
	update := bson.M{
		"$set": bson.M{
			"current_state": snapshot.CurrentState,
			"last_updated":  time.Now(),
		},
		"$push": bson.M{
			"update_log": snapshot.UpdateLog[0], // Add the latest change
		},
	}

	_, err = coll.UpdateOne(ctx, filter, update)
	return err
}

// GetOptimizedSnapshot retrieves an optimized snapshot for a specific resource
func (d *Database) GetOptimizedSnapshot(namespace, kind, name string) (*models.OptimizedResourceSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("optimized_snapshots")
	filter := bson.M{
		"namespace": namespace,
		"kind":      kind,
		"name":      name,
	}

	var snapshot models.OptimizedResourceSnapshot
	err := coll.FindOne(ctx, filter).Decode(&snapshot)
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// GetOptimizedSnapshots retrieves all optimized snapshots for a namespace
func (d *Database) GetOptimizedSnapshots(namespace string) ([]*models.OptimizedResourceSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := d.database.Collection("optimized_snapshots")
	filter := bson.M{"namespace": namespace}
	cur, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []*models.OptimizedResourceSnapshot
	if err := cur.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
