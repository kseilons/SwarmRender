package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobs_created_total",
		Help: "The total number of created render jobs",
	})
)
