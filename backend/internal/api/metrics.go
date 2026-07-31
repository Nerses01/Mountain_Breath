package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metrics owns the Prometheus registry and every instrument the API exports.
type metrics struct {
	registry *prometheus.Registry

	httpRequests  *prometheus.CounterVec
	httpDuration  *prometheus.HistogramVec
	ordersCreated prometheus.Counter
}

func newMetrics(extra ...prometheus.Collector) *metrics {
	reg := prometheus.NewRegistry()

	// Standard runtime collectors: goroutines, GC, heap, process CPU/fds.
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	for _, c := range extra {
		reg.MustRegister(c)
	}

	return &metrics{
		registry: reg,

		// Counter: only ever goes up; rate() turns it into requests/second.
		httpRequests: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "mb_http_requests_total",
			Help: "HTTP requests processed, by method, route pattern and status.",
		}, []string{"method", "route", "status"}),

		// Histogram: counts observations into latency buckets — lets
		// Prometheus compute p95/p99 across instances.
		httpDuration: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "mb_http_request_duration_seconds",
			Help:    "HTTP request duration, by method and route pattern.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
		}, []string{"method", "route"}),

		// A business metric: infrastructure graphs say "the server is fine",
		// this one says "the shop is selling".
		ordersCreated: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "mb_orders_created_total",
			Help: "Orders successfully placed.",
		}),
	}
}

// metricsMiddleware records every request. IMPORTANT: the route label uses
// chi's route PATTERN (/api/v1/products/{slug}) — never the raw URL path.
// Labeling by raw path would create one metric series per product slug
// (unbounded label cardinality), which slowly kills a Prometheus server.
func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched" // 404s: one shared series, same cardinality rule
		}
		s.metrics.httpRequests.WithLabelValues(r.Method, route, strconv.Itoa(ww.Status())).Inc()
		s.metrics.httpDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}
