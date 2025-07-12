package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/DriftGuard/core/internal/config"
	"github.com/DriftGuard/core/internal/controller"
	"github.com/DriftGuard/core/internal/health"
	"github.com/DriftGuard/core/internal/metrics"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	router     *gin.Engine
	config     config.ServerConfig
	controller *controller.DriftController
	logger     *zap.Logger
	server     *http.Server
	healthSvc  *health.HealthService
	metrics    *metrics.Metrics
}

func New(cfg config.ServerConfig, controller *controller.DriftController, logger *zap.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

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

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

func (s *Server) Start(port int) error {
	s.logger.Info("Starting HTTP server", zap.Int("port", port))

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Failed to start HTTP server", zap.Error(err))
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx interface{}) error {
	if s.server != nil {
		return s.server.Shutdown(ctx.(context.Context))
	}
	return nil
}

func (s *Server) setupMiddleware() {
	s.router.Use(gin.Recovery())
	s.router.Use(s.loggingMiddleware())
	s.router.Use(s.metricsMiddleware())
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.getHealth)
	s.router.GET("/ready", s.getReady)
	s.router.GET("/metrics", s.getMetrics)

	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/health", s.getHealth)
		v1.GET("/drift-records", s.getDriftRecords)
		v1.GET("/drift-records/:resourceId", s.getDriftRecord)
		v1.GET("/drift-records/active", s.getActiveDrifts)
		v1.GET("/drift-records/resolved", s.getResolvedDrifts)
		v1.GET("/statistics", s.getStatistics)
		v1.POST("/analyze", s.triggerAnalysis)
	}
}

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

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		_ = time.Since(start)

		s.metrics.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})).ServeHTTP(c.Writer, c.Request)
	}
}

func (s *Server) getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Service is running",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) getReady(c *gin.Context) {
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

func (s *Server) getDriftRecords(c *gin.Context) {
	namespace := c.Query("namespace")
	var driftDetected *bool

	if driftStr := c.Query("drift_detected"); driftStr != "" {
		detected := driftStr == "true"
		driftDetected = &detected
	}

	records, err := s.controller.GetDriftRecords(namespace, driftDetected)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve drift records",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drift_records": records,
		"count":         len(records),
	})
}

func (s *Server) getDriftRecord(c *gin.Context) {
	resourceID := c.Param("resourceId")

	record, err := s.controller.GetDriftRecord(resourceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Drift record not found",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (s *Server) getActiveDrifts(c *gin.Context) {
	records, err := s.controller.GetActiveDrifts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve active drift records",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drift_records": records,
		"count":         len(records),
		"status":        "active",
	})
}

func (s *Server) getResolvedDrifts(c *gin.Context) {
	records, err := s.controller.GetResolvedDrifts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve resolved drift records",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"drift_records": records,
		"count":         len(records),
		"status":        "resolved",
	})
}

func (s *Server) getStatistics(c *gin.Context) {
	statistics, err := s.controller.GetDriftStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve statistics",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"statistics": statistics,
	})
}

func (s *Server) triggerAnalysis(c *gin.Context) {
	go func() {
		s.logger.Info("Manual drift analysis triggered via API")
		s.controller.TriggerManualAnalysis()
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Drift analysis triggered successfully",
		"status":  "started",
	})
}
