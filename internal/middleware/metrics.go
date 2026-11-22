package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics records request counts and latency per route.
type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	prefixes []string
}

func NewMetrics(reg prometheus.Registerer, prefixes []string) *Metrics {
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sentinel_http_requests_total",
			Help: "Requests handled by the gateway.",
		}, []string{"route", "method", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "sentinel_http_request_duration_seconds",
			Help:    "Time from request received to response written.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"route", "method"}),
		prefixes: prefixes,
	}
	reg.MustRegister(m.requests, m.duration)
	return m
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		route := m.routeLabel(r.URL.Path)
		method := normalizeMethod(r.Method)
		m.requests.WithLabelValues(route, method, strconv.Itoa(rec.status)).Inc()
		m.duration.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
	})
}

// routeLabel maps a raw path to its configured route prefix, keeping label cardinality bounded.
func (m *Metrics) routeLabel(path string) string {
	for _, p := range m.prefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return p
		}
	}
	return "unmatched"
}

// normalizeMethod returns the method if it is a standard HTTP method, otherwise "OTHER",
// so clients cannot create unlimited label values with made-up methods.
func normalizeMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return method
	default:
		return "OTHER"
	}
}
