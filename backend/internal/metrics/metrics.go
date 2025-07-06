package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Metrics holds all the Prometheus metrics
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec

	// Drift detection metrics
	driftEventsTotal       *prometheus.CounterVec
	driftDetectionDuration *prometheus.HistogramVec
	driftScoreGauge        *prometheus.GaugeVec

	// Kubernetes metrics
	k8sResourcesWatched *prometheus.GaugeVec
	k8sEventsProcessed  *prometheus.CounterVec

	// Database metrics
	dbConnectionsActive *prometheus.GaugeVec
	dbQueryDuration     *prometheus.HistogramVec

	// Application metrics
	appStartTime prometheus.Gauge
	appUptime    prometheus.Gauge
	appVersion   *prometheus.GaugeVec

	// Registry for all metrics
	registry *prometheus.Registry
	logger   *zap.Logger
}

// NewMetrics creates a new metrics instance
func NewMetrics(logger *zap.Logger) *Metrics {
	registry := prometheus.NewRegistry()

	m := &Metrics{
		registry: registry,
		logger:   logger,
	}

	m.initializeMetrics()
	m.registerMetrics()

	return m
}

// initializeMetrics creates all the metric definitions
func (m *Metrics) initializeMetrics() {
	// HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "driftguard_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "driftguard_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpRequestsInFlight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "driftguard_http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
		[]string{"method", "endpoint"},
	)

	// Drift detection metrics
	m.driftEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "driftguard_drift_events_total",
			Help: "Total number of drift events detected",
		},
		[]string{"severity", "drift_type", "environment"},
	)

	m.driftDetectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "driftguard_drift_detection_duration_seconds",
			Help:    "Time taken to detect drift in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"resource_type", "environment"},
	)

	m.driftScoreGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "driftguard_drift_score",
			Help: "Current drift score for resources",
		},
		[]string{"resource_type", "resource_name", "environment"},
	)

	// Kubernetes metrics
	m.k8sResourcesWatched = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "driftguard_k8s_resources_watched",
			Help: "Number of Kubernetes resources being watched",
		},
		[]string{"resource_type", "namespace"},
	)

	m.k8sEventsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "driftguard_k8s_events_processed_total",
			Help: "Total number of Kubernetes events processed",
		},
		[]string{"event_type", "resource_type"},
	)

	// Database metrics
	m.dbConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "driftguard_db_connections_active",
			Help: "Number of active database connections",
		},
		[]string{"database"},
	)

	m.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "driftguard_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// Application metrics
	m.appStartTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "driftguard_app_start_time_seconds",
			Help: "Application start time in seconds since epoch",
		},
	)

	m.appUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "driftguard_app_uptime_seconds",
			Help: "Application uptime in seconds",
		},
	)

	m.appVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "driftguard_app_version_info",
			Help: "Application version information",
		},
		[]string{"version", "commit", "build_date"},
	)
}

// registerMetrics registers all metrics with the registry
func (m *Metrics) registerMetrics() {
	metrics := []prometheus.Collector{
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestsInFlight,
		m.driftEventsTotal,
		m.driftDetectionDuration,
		m.driftScoreGauge,
		m.k8sResourcesWatched,
		m.k8sEventsProcessed,
		m.dbConnectionsActive,
		m.dbQueryDuration,
		m.appStartTime,
		m.appUptime,
		m.appVersion,
	}

	for _, metric := range metrics {
		if err := m.registry.Register(metric); err != nil {
			m.logger.Error("Failed to register metric", zap.Error(err))
		}
	}
}

// SetAppStartTime sets the application start time
func (m *Metrics) SetAppStartTime() {
	m.appStartTime.Set(float64(time.Now().Unix()))
}

// SetAppVersion sets the application version information
func (m *Metrics) SetAppVersion(version, commit, buildDate string) {
	m.appVersion.WithLabelValues(version, commit, buildDate).Set(1)
}

// UpdateUptime updates the application uptime metric
func (m *Metrics) UpdateUptime(startTime time.Time) {
	uptime := time.Since(startTime).Seconds()
	m.appUptime.Set(uptime)
}

// HTTP middleware for request metrics
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		m.httpRequestsInFlight.WithLabelValues(r.Method, r.URL.Path).Inc()
		defer m.httpRequestsInFlight.WithLabelValues(r.Method, r.URL.Path).Dec()

		// Create a response writer that captures the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		m.httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		m.httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, string(rw.statusCode)).Inc()
	})
}

// RecordDriftEvent records a drift event
func (m *Metrics) RecordDriftEvent(severity, driftType, environment string) {
	m.driftEventsTotal.WithLabelValues(severity, driftType, environment).Inc()
}

// RecordDriftDetectionDuration records the time taken to detect drift
func (m *Metrics) RecordDriftDetectionDuration(resourceType, environment string, duration time.Duration) {
	m.driftDetectionDuration.WithLabelValues(resourceType, environment).Observe(duration.Seconds())
}

// SetDriftScore sets the drift score for a resource
func (m *Metrics) SetDriftScore(resourceType, resourceName, environment string, score float64) {
	m.driftScoreGauge.WithLabelValues(resourceType, resourceName, environment).Set(score)
}

// SetK8sResourcesWatched sets the number of resources being watched
func (m *Metrics) SetK8sResourcesWatched(resourceType, namespace string, count float64) {
	m.k8sResourcesWatched.WithLabelValues(resourceType, namespace).Set(count)
}

// RecordK8sEvent records a processed Kubernetes event
func (m *Metrics) RecordK8sEvent(eventType, resourceType string) {
	m.k8sEventsProcessed.WithLabelValues(eventType, resourceType).Inc()
}

// SetDBConnections sets the number of active database connections
func (m *Metrics) SetDBConnections(database string, count float64) {
	m.dbConnectionsActive.WithLabelValues(database).Set(count)
}

// RecordDBQueryDuration records the duration of a database query
func (m *Metrics) RecordDBQueryDuration(operation, table string, duration time.Duration) {
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// Handler returns the Prometheus metrics handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
