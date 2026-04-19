package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	FramesRendered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "worker_frames_rendered_total",
		Help: "The total number of rendered frames",
	})

	ActiveJobs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "worker_jobs_active",
		Help: "The number of active rendering jobs",
	})
)
