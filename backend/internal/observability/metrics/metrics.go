package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	provisionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "provision_duration_seconds",
		Help:    "Duration of container provisioning attempts",
		Buckets: prometheus.DefBuckets,
	}, []string{"result"})

	cleanupOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cleanup_operations_total",
		Help: "Count of cleanup operations by source and result",
	}, []string{"source", "result"})

	activeContainers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_containers",
		Help: "Number of running containers (logical state)",
	})
)

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
