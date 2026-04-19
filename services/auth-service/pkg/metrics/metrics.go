package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	AuthFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_failures_total",
		Help: "The total number of failed authentication attempts",
	})

	ActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "auth_active_sessions_gauge",
		Help: "Approximate number of active user sessions",
	})
)
