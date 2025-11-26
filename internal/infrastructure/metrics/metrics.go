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

	// Register all custom metrics
	registry.MustRegister(httpRequestsTotal)
	registry.MustRegister(httpRequestDuration)
	registry.MustRegister(urlClicksTotal)
	registry.MustRegister(urlsActiveTotal)

	return &Metrics{
		Registry:            registry,
		HTTPRequestsTotal:   httpRequestsTotal,
		HTTPRequestDuration: httpRequestDuration,
		URLClicksTotal:      urlClicksTotal,
		URLsActiveTotal:     urlsActiveTotal,
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

// RecordURLClick records a URL click metric
func (m *Metrics) RecordURLClick(shortCode string) {
	m.URLClicksTotal.WithLabelValues(shortCode).Inc()
}

// SetActiveURLs sets the current number of active URLs
func (m *Metrics) SetActiveURLs(count float64) {
	m.URLsActiveTotal.Set(count)
}

// IncrementActiveURLs increments the active URLs counter
func (m *Metrics) IncrementActiveURLs() {
	m.URLsActiveTotal.Inc()
}

// DecrementActiveURLs decrements the active URLs counter
func (m *Metrics) DecrementActiveURLs() {
	m.URLsActiveTotal.Dec()
}
