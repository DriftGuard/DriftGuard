package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/internal/controller"
	"github.com/DriftGuard/core/internal/health"
	"github.com/DriftGuard/core/internal/metrics"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	router     *gin.Engine
	config     config.ServerConfig
	controller *controller.DriftController
	logger     *zap.Logger
	server     *http.Server
	healthSvc  *health.HealthService
	metrics    *metrics.Metrics
}

// New creates a new HTTP server instance
func New(cfg config.ServerConfig, controller *controller.DriftController, logger *zap.Logger) *Server {
	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Initialize health service and metrics
	healthSvc := health.NewHealthService(logger)
	metrics := metrics.NewMetrics(logger)

	server := &Server{
		router:     router,
		config:     cfg,
		controller: controller,
		logger:     logger,
		healthSvc:  healthSvc,
		metrics:    metrics,
	}

	// Set up middleware
	server.setupMiddleware()

	// Set up routes
	server.setupRoutes()

	return server
}

// Start starts the HTTP server
func (s *Server) Start(port int) error {
	s.logger.Info("Starting HTTP server", zap.Int("port", port))

	// Create HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// Start server
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Failed to start HTTP server", zap.Error(err))
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx interface{}) error {
	if s.server != nil {
		return s.server.Shutdown(ctx.(context.Context))
	}
	return nil
}

// setupMiddleware configures all middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.router.Use(gin.Recovery())

	// Logging middleware
	s.router.Use(s.loggingMiddleware())

	// Metrics middleware
	s.router.Use(s.metricsMiddleware())
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoints
	s.router.GET("/health", s.getHealth)
	s.router.GET("/ready", s.getReady)

	// Metrics endpoint
	s.router.GET("/metrics", s.getMetrics)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/health", s.getHealth)
		v1.GET("/snapshots", s.getSnapshots)
		v1.GET("/snapshots/:id", s.getSnapshot)
		v1.GET("/drifts", s.getDriftEvents)
		v1.GET("/drifts/:id", s.getDriftEvent)
		v1.POST("/drifts/:id/remediate", s.remediateDrift)
		v1.GET("/environments", s.getEnvironments)
		v1.GET("/statistics", s.getStatistics)
	}
}

// loggingMiddleware logs all requests
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		s.logger.Info("HTTP Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// metricsMiddleware records metrics for all requests
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		_ = time.Since(start) // Duration captured by metrics middleware

		// Record metrics using the HTTP middleware
		s.metrics.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This will be handled by the metrics middleware
		})).ServeHTTP(c.Writer, c.Request)
	}
}

// API handler methods

func (s *Server) getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Service is running",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) getReady(c *gin.Context) {
	// Check if all components are ready
	status, message := s.healthSvc.GetOverallStatus(c.Request.Context())
	if status != health.StatusHealthy {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "not ready",
			"message": message,
			"time":    time.Now().UTC(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"message": "All components are ready",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) getMetrics(c *gin.Context) {
	s.metrics.Handler().ServeHTTP(c.Writer, c.Request)
}

func (s *Server) getSnapshots(c *gin.Context) {
	namespace := c.Query("namespace")

	// Get snapshots from database
	snapshots, err := s.controller.GetSnapshots(namespace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve snapshots",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"snapshots": snapshots,
		"count":     len(snapshots),
	})
}

func (s *Server) getSnapshot(c *gin.Context) {
	id := c.Param("id")

	// Parse the ID to extract namespace, kind, and name
	// For now, we'll use a simple format: namespace:kind:name
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid snapshot ID format. Expected: namespace:kind:name",
		})
		return
	}

	namespace, kind, name := parts[0], parts[1], parts[2]

	snapshot, err := s.controller.GetSnapshot(namespace, kind, name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Snapshot not found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func (s *Server) getDriftEvents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Drift events endpoint - not implemented yet",
		"events":  []interface{}{},
	})
}

func (s *Server) getDriftEvent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Drift event endpoint - not implemented yet",
	})
}

func (s *Server) remediateDrift(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Drift remediation endpoint - not implemented yet",
	})
}

func (s *Server) getEnvironments(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":      "Environments endpoint - not implemented yet",
		"environments": []interface{}{},
	})
}

func (s *Server) getStatistics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message":    "Statistics endpoint - not implemented yet",
		"statistics": map[string]interface{}{},
	})
}
