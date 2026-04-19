package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	StitchingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "stitching_duration_seconds",
		Help:    "Duration of the stitching process",
		Buckets: prometheus.DefBuckets,
	})

	CompressionRatio = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "stitching_compression_ratio",
		Help: "Ratio of input frames size to output video size",
	})
)
