package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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

func printBanner() {
	blue := "\033[1;34m"
	cyan := "\033[1;36m"
	green := "\033[1;32m"
	reset := "\033[0m"
	banner := `
 ____       _  __ _    ____                     _ 
|  _ \ _ __(_)/ _| |_ / ___|_   _  __ _ _ __ __| |
| | | | '__| | |_| __| |  _| | | |/ _` + "`" + ` | '__/ _` + "`" + ` |
| |_| | |  | |  _| |_| |_| | |_| | (_| | | | (_| |
|____/|_|  |_|_|  \__|\____|\__,_|\__,_|_|  \__,_|
`
	fmt.Println(blue + banner + reset)
	fmt.Println(cyan + "DriftGuard - Intelligent GitOps Drift Detection" + reset)
	fmt.Printf("%sVersion:%s   %s\n", green, reset, Version)
	fmt.Printf("%sBuild:%s     %s\n", green, reset, BuildDate)
	fmt.Printf("%sCommit:%s    %s\n", green, reset, GitCommit)
	fmt.Printf("%sGo:%s         %s\n", green, reset, runtime.Version())
	fmt.Printf("%sOS/Arch:%s    %s/%s\n", green, reset, runtime.GOOS, runtime.GOARCH)
	fmt.Println()
}

func main() {
	printBanner()
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n%s\n", r, debug.Stack())
			os.Exit(2)
		}
	}()

	flag.Parse()

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

	metrics := metrics.NewMetrics(logger)
	metrics.SetAppStartTime()
	metrics.SetAppVersion(Version, GitCommit, BuildDate)

	logger.Info("Loading configuration", zap.String("config_file", *configFile))
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Error("Failed to load configuration", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded successfully")

	lifecycleMgr := lifecycle.NewLifecycleManager(logger)

	logger.Info("Initializing database")
	db, err := database.New(cfg.Database)
	if err != nil {
		logger.Error("Failed to initialize database", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database initialized successfully")

	logger.Info("Initializing drift controller")
	driftController, err := controller.NewDriftController(cfg, db, logger)
	if err != nil {
		logger.Error("Failed to initialize drift controller", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to initialize drift controller: %v\n", err)
		os.Exit(1)
	}
	logger.Info("Drift controller initialized successfully")

	logger.Info("Initializing HTTP server")
	httpServer := server.New(cfg.Server, driftController, logger)
	logger.Info("HTTP server initialized successfully")

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

	lifecycleMgr.RegisterComponent(&lifecycleComponent{
		name:   "http-server",
		start:  func(ctx context.Context) error { return nil },
		stop:   func(ctx context.Context) error { return httpServer.Shutdown(ctx) },
		health: func(ctx context.Context) error { return nil },
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("Starting application lifecycle")
	if err := lifecycleMgr.Start(ctx); err != nil {
		logger.Error("Failed to start application lifecycle", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Failed to start application lifecycle: %v\n", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("Starting HTTP server", zap.Int("port", *port))
		if err := httpServer.Start(*port); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start HTTP server", zap.Error(err))
			fmt.Fprintf(os.Stderr, "Failed to start HTTP server: %v\n", err)
			cancel()
		}
	}()

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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down DriftGuard Controller...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := lifecycleMgr.Stop(shutdownCtx); err != nil {
		logger.Error("Error during lifecycle shutdown", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Error during lifecycle shutdown: %v\n", err)
	}

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Error during server shutdown", zap.Error(err))
		fmt.Fprintf(os.Stderr, "Error during server shutdown: %v\n", err)
	}

	logger.Info("DriftGuard Controller stopped successfully")
}

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
