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

	StageDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "worker_stage_duration_seconds",
		Help:    "Duration of each rendering stage",
		Buckets: prometheus.DefBuckets,
	}, []string{"stage"}) // stage: download, render, upload

	QueueWaitTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "worker_queue_wait_duration_seconds",
		Help:    "Time spent in the queue before processing",
		Buckets: prometheus.DefBuckets,
	})
)
