package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "route", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})

	httpRequestsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Number of HTTP requests currently being processed.",
	})

	httpRequestErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "Total number of HTTP requests that resulted in an error (status >= 400).",
	}, []string{"method", "route", "status"})
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, httpRequestsInFlight, httpRequestErrorsTotal)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		httpRequestsInFlight.Inc()
		start := time.Now()

		next.ServeHTTP(sw, r)

		httpRequestsInFlight.Dec()
		duration := time.Since(start).Seconds()

		route := r.URL.Path
		if rc := chi.RouteContext(r.Context()); rc != nil {
			if pattern := rc.RoutePattern(); pattern != "" {
				route = pattern
			}
		}

		status := strconv.Itoa(sw.status)
		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(duration)

		if sw.status >= 400 {
			httpRequestErrorsTotal.WithLabelValues(r.Method, route, status).Inc()
		}
	})
}
