package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "containerlease_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "containerlease_http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	provisionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "containerlease_provision_duration_seconds",
		Help:    "Duration of container provisioning attempts",
		Buckets: prometheus.DefBuckets,
	}, []string{"result"})

	cleanupOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "containerlease_cleanup_operations_total",
		Help: "Count of cleanup operations by source and result",
	}, []string{"source", "result"})

	activeContainers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "containerlease_active_containers",
		Help: "Number of running containers (logical state)",
	})

	// Phase 2: Self-Healing and Chaos Monkey metrics
	containerRestarts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "containerlease_container_restarts_total",
		Help: "Count of container restart attempts",
	}, []string{"result"})

	containerFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "containerlease_container_failures_total",
		Help: "Count of container failures before restart/termination",
	}, []string{"reason"})

	chaosMonkeyKills = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "containerlease_chaos_monkey_kills_total",
		Help: "Count of containers killed by chaos monkey for resilience testing",
	}, []string{"operation"})
)

// ObserveHTTPRequest records an HTTP request metric
func ObserveHTTPRequest(method, path, status string, duration time.Duration) {
	httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	httpRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
}

// ObserveProvision records the duration of a provisioning attempt with a result label.
func ObserveProvision(result string, duration time.Duration) {
	provisionDuration.WithLabelValues(result).Observe(duration.Seconds())
}

// ObserveCleanup increments the cleanup counter for the given source and result.
func ObserveCleanup(source, result string) {
	cleanupOperations.WithLabelValues(source, result).Inc()
}

// IncrementActive increments the active container gauge.
func IncrementActive() {
	activeContainers.Inc()
}

// DecrementActive decrements the active container gauge.
func DecrementActive() {
	activeContainers.Dec()
}

// SetActive sets the active container gauge to a specific count.
func SetActive(count int) {
	if count < 0 {
		count = 0
	}
	activeContainers.Set(float64(count))
}

// ObserveContainerRestart records a container restart attempt (Phase 2: Self-Healing)
func ObserveContainerRestart(result string) {
	containerRestarts.WithLabelValues(result).Inc()
}

// ObserveContainerFailure records a container failure (Phase 2: Self-Healing)
func ObserveContainerFailure(reason string) {
	containerFailures.WithLabelValues(reason).Inc()
}

// ObserveChaosMoney records chaos monkey operations (Phase 2)
func ObserveChaosMoney(operation string, count int) {
	switch operation {
	case "injection":
		// Just track that an injection occurred
		chaosMonkeyKills.WithLabelValues("injection_check").Inc()
	case "kill":
		// Count each kill
		for i := 0; i < count; i++ {
			chaosMonkeyKills.WithLabelValues("container_killed").Inc()
		}
	}
}
