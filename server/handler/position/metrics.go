package position

import "github.com/prometheus/client_golang/prometheus"

type metrics struct {
	Requests   prometheus.Counter
	Errors     prometheus.Counter
	Latency    prometheus.Histogram
	BgRequests prometheus.Counter
	BgErrors   prometheus.Counter
	BgLatency  prometheus.Histogram
}

func (m *metrics) valid() bool {
	return m.Requests != nil && m.Errors != nil && m.Latency != nil &&
		m.BgRequests != nil && m.BgErrors != nil && m.BgLatency != nil
}

func newMetrics() *metrics {
	m := &metrics{
		Requests: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "traintracker",
			Subsystem: "fg",
			Name:      "requests",
		}),
		Errors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "traintracker",
			Subsystem: "fg",
			Name:      "errors",
		}),
		Latency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "traintracker",
			Subsystem: "fg",
			Name:      "latency",
			Buckets:   prometheus.DefBuckets,
		}),
		BgRequests: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "traintracker",
			Subsystem: "bg",
			Name:      "requests",
		}),
		BgErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "traintracker",
			Subsystem: "bg",
			Name:      "errors",
		}),
		BgLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "traintracker",
			Subsystem: "bg",
			Name:      "latency",
			Buckets:   []float64{0.5, 1, 2, 3, 5, 7.5, 10, 12.5, 15},
		}),
	}

	prometheus.MustRegister(m.Requests, m.Errors, m.Latency, m.BgRequests, m.BgErrors, m.BgLatency)

	return m
}
