package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zaplink_http_requests_total",
			Help: "Total number of HTTP requests processed by ZapLink.",
		},
		[]string{"method", "route", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "zaplink_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	LinksCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "zaplink_links_created_total",
			Help: "Total number of short links created.",
		},
	)

	RedirectsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "zaplink_redirects_total",
			Help: "Total number of redirects served.",
		},
	)

	ClicksTrackedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "zaplink_clicks_tracked_total",
			Help: "Total number of clicks persisted.",
		},
	)
)

func Register() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		LinksCreatedTotal,
		RedirectsTotal,
		ClicksTrackedTotal,
	)
}
