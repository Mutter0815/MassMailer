package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	APIRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "api_http_requests_total", Help: "HTTP requests"},
		[]string{"method", "path", "status"},
	)
	APIRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	PublishedJobsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "api_published_jobs_total", Help: "Jobs published to queue"},
	)

	WorkerJobsConsumed = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "worker_jobs_consumed_total", Help: "Jobs consumed"},
	)
	WorkerJobsSent = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "worker_jobs_sent_total", Help: "Jobs sent successfully"},
	)
	WorkerJobsFailed = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "worker_jobs_failed_total", Help: "Jobs failed"},
	)
	WorkerJobRetries = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "worker_job_retries_total", Help: "Retries performed"},
	)
	WorkerProcessDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "worker_job_process_duration_seconds",
			Help:    "Time spent processing a job",
			Buckets: prometheus.DefBuckets,
		},
	)
)

func init() {
	prometheus.MustRegister(
		APIRequestsTotal, APIRequestDuration, PublishedJobsTotal,
		WorkerJobsConsumed, WorkerJobsSent, WorkerJobsFailed, WorkerJobRetries, WorkerProcessDuration,
	)
}

func Handler() http.Handler { return promhttp.Handler() }
