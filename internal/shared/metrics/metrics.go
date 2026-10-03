package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Registry struct{ *prometheus.Registry }

var (
	TickDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "lodeen_tick_duration_seconds",
		Help:    "Duration of a single world tick.",
		Buckets: []float64{0.001, 0.002, 0.005, 0.010, 0.025, 0.050, 0.100},
	})
	SnapshotBytes = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "lodeen_snapshot_bytes",
		Help:    "Size of a snapshot payload in bytes.",
		Buckets: prometheus.ExponentialBuckets(128, 4, 8),
	})
	RTTSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "lodeen_rtt_seconds",
		Help:    "Client round-trip time as measured by ping.",
		Buckets: []float64{0.005, 0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1.0},
	})
)

func New() *Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		TickDuration,
		SnapshotBytes,
		RTTSeconds,
	)
	return &Registry{reg}
}

func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
}
