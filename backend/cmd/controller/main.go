package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/internal/controller"
	"github.com/DriftGuard/core/internal/database"
	"github.com/DriftGuard/core/internal/lifecycle"
	"github.com/DriftGuard/core/internal/metrics"
	"github.com/DriftGuard/core/internal/server"
	"go.uber.org/zap"
)

// Application version information
const (
	Version   = "0.1.0"
	BuildDate = "2024-01-01"
	GitCommit = "development"
)

var (
	configFile      = flag.String("config", "configs/config.yaml", "Path to configuration file")
	port            = flag.Int("port", 8080, "Port to run the server on")
	logLevel        = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	metricsPort     = flag.Int("metrics-port", 9090, "Port for metrics endpoint")
	enableProfiling = flag.Bool("enable-profiling", false, "Enable pprof profiling")
	dryRun          = flag.Bool("dry-run", false, "Run in dry-run mode")
	showVersion     = flag.Bool("version", false, "Show version information")
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n%s\n", r, debug.Stack())
			os.Exit(2)
		}
	}()

	flag.Parse()

	// Show version and exit if requested
	if *showVersion {
		fmt.Printf("DriftGuard Controller v%s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
		fmt.Printf("Git Commit: %s\n", GitCommit)
		os.Exit(0)
	}

	fmt.Println("Starting DriftGuard Controller...")
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Git Commit: %s\n", GitCommit)

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting DriftGuard Controller",
		zap.String("version", Version),
		zap.String("build_date", BuildDate),
		zap.String("git_commit", GitCommit))

	// Initialize metrics
	metrics := metrics.NewMetrics(logger)
	metrics.SetAppStartTime()
	metrics.SetAppVersion(Version, GitCommit, BuildDate)

	// Load configuration
	logger.Info("Loading configuration", zap.String("config_file", *configFile))
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Error("Failed to load configuration", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully")

	// Initialize lifecycle manager
	lifecycleMgr := lifecycle.NewLifecycleManager(logger)

	// Initialize database
	logger.Info("Initializing database")
	db, err := database.New(cfg.Database)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database initialized successfully")


	// Initialize drift controller
	logger.Info("Initializing drift controller")
	driftController := controller.NewDriftController(cfg, db, logger)
	logger.Info("Drift controller initialized successfully")

	// Initialize HTTP server
	logger.Info("Initializing HTTP server")
	httpServer := server.New(cfg.Server, driftController, logger)
	logger.Info("HTTP server initialized successfully")

	// Register components with lifecycle manager
	lifecycleMgr.RegisterComponent(&lifecycleComponent{
		name:   "database",
		start:  func(ctx context.Context) error { return nil },
		stop:   func(ctx context.Context) error { return db.Close() },
		health: func(ctx context.Context) error { return db.HealthCheck(ctx) },
	})

	lifecycleMgr.RegisterComponent(&lifecycleComponent{
		name:   "drift-controller",
		start:  func(ctx context.Context) error { return driftController.Start(ctx) },
		stop:   func(ctx context.Context) error { return driftController.Stop() },
		health: func(ctx context.Context) error { return nil },
	})

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start application lifecycle
	logger.Info("Starting application lifecycle")
	if err := lifecycleMgr.Start(ctx); err != nil {
		logger.Error("Failed to start application lifecycle", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to start application lifecycle: %v\n", err)
		os.Exit(1)
	}

	// Start HTTP server
	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", *port))
		if err := httpServer.Start(*port); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server", zap.Error(err))
			fmt.Fprintf(os.Stderr, "Failed to start HTTP server: %v\n", err)
			cancel()
		}
	}()

	// Start metrics update goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics.UpdateUptime(lifecycleMgr.GetStartTime())
			}
		}
	}()

	logger.Info("DriftGuard Controller is running",
		zap.Int("port", *port),
		zap.Duration("uptime", lifecycleMgr.GetUptime()))

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down DriftGuard Controller...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop lifecycle manager
	if err := lifecycleMgr.Stop(shutdownCtx); err != nil {
		logger.Error("Error during lifecycle shutdown", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Error during lifecycle shutdown: %v\n", err)
	}

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Error during server shutdown: %v\n", err)
	}

	logger.Info("DriftGuard Controller stopped successfully")
}

// lifecycleComponent implements the lifecycle.Component interface
type lifecycleComponent struct {
	name   string
	start  func(context.Context) error
	stop   func(context.Context) error
	health func(context.Context) error
}

func (lc *lifecycleComponent) Name() string {
	return lc.name
}

func (lc *lifecycleComponent) Start(ctx context.Context) error {
	return lc.start(ctx)
}

func (lc *lifecycleComponent) Stop(ctx context.Context) error {
	return lc.stop(ctx)
}

func (lc *lifecycleComponent) Health(ctx context.Context) error {
	return lc.health(ctx)
}
