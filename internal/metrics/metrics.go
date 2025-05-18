package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	RateRequestsTotal       prometheus.Counter
	ConversionRequestsTotal prometheus.Counter
	HistoricalRequestsTotal prometheus.Counter
}

func NewMetrics() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"path", "method", "status_code"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"path", "method"},
		),

		RateRequestsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "rate_requests_total",
				Help: "Total number of exchange rate requests",
			},
		),

		ConversionRequestsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "conversion_requests_total",
				Help: "Total number of currency conversion requests",
			},
		),

		HistoricalRequestsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "historical_requests_total",
				Help: "Total number of historical exchange rate requests",
			},
		),
	}
}
