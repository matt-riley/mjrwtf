// Package metrics provides Prometheus metrics for the application.
// It includes HTTP request metrics, URL tracking metrics, and Go runtime metrics.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "mjrwtf"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// Registry is the Prometheus registry for this metrics instance
	Registry *prometheus.Registry

	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Business metrics
	URLClicksTotal  *prometheus.CounterVec
	URLsActiveTotal prometheus.Gauge

	// Redirect click recording (async worker pool) metrics
	RedirectClickQueueDepth          prometheus.Gauge
	RedirectClickDroppedTotal        prometheus.Counter
	RedirectClickRecordFailuresTotal prometheus.Counter
}

// New creates and registers all Prometheus metrics with a new registry
func New() *Metrics {
	registry := prometheus.NewRegistry()

	// Register Go runtime metrics
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	urlClicksTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "url_clicks_total",
			Help:      "Total number of URL redirect clicks",
		},
		[]string{"short_code"},
	)

	urlsActiveTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "urls_active_total",
			Help:      "Total number of active shortened URLs",
		},
	)

	redirectClickQueueDepth := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "redirect_click_queue_depth",
			Help:      "Current depth of the async redirect click recording queue",
		},
	)

	redirectClickDroppedTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "redirect_click_dropped_total",
			Help:      "Total number of redirect click recording tasks dropped (queue full or shutdown)",
		},
	)

	redirectClickRecordFailuresTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "redirect_click_record_failures_total",
			Help:      "Total number of failures while recording redirect click analytics",
		},
	)

	// Register all custom metrics
	registry.MustRegister(httpRequestsTotal)
	registry.MustRegister(httpRequestDuration)
	registry.MustRegister(urlClicksTotal)
	registry.MustRegister(urlsActiveTotal)
	registry.MustRegister(redirectClickQueueDepth)
	registry.MustRegister(redirectClickDroppedTotal)
	registry.MustRegister(redirectClickRecordFailuresTotal)

	return &Metrics{
		Registry:                         registry,
		HTTPRequestsTotal:                httpRequestsTotal,
		HTTPRequestDuration:              httpRequestDuration,
		URLClicksTotal:                   urlClicksTotal,
		URLsActiveTotal:                  urlsActiveTotal,
		RedirectClickQueueDepth:          redirectClickQueueDepth,
		RedirectClickDroppedTotal:        redirectClickDroppedTotal,
		RedirectClickRecordFailuresTotal: redirectClickRecordFailuresTotal,
	}
}

// Handler returns an HTTP handler for the metrics endpoint
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

// RecordHTTPRequest records an HTTP request metric
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration float64) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration)
}

// RecordURLClick records a URL click metric.
//
// Privacy note: Prometheus label values are typically treated as low-sensitivity but are often broadly accessible.
// To avoid leaking per-short-code activity and to prevent high-cardinality labels, this metric is aggregated.
func (m *Metrics) RecordURLClick(_ string) {
	m.URLClicksTotal.WithLabelValues("all").Inc()
}

// SetActiveURLs sets the current number of active URLs.
// Can be called periodically by querying the repository for total count.
func (m *Metrics) SetActiveURLs(count float64) {
	m.URLsActiveTotal.Set(count)
}

// IncrementActiveURLs increments the active URLs counter.
// Call this when creating a new URL.
func (m *Metrics) IncrementActiveURLs() {
	m.URLsActiveTotal.Inc()
}

// DecrementActiveURLs decrements the active URLs counter.
// Call this when deleting a URL.
func (m *Metrics) DecrementActiveURLs() {
	m.URLsActiveTotal.Dec()
}
