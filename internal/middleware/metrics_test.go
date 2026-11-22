package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRouteLabel(t *testing.T) {
	m := &Metrics{prefixes: []string{"/orders", "/users"}}

	tests := []struct {
		path string
		want string
	}{
		{"/orders", "/orders"},
		{"/orders/5", "/orders"},
		{"/users/7/profile", "/users"},
		{"/ordersXYZ", "unmatched"},
		{"/inventory/1", "unmatched"},
		{"/", "unmatched"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := m.routeLabel(tt.path); got != tt.want {
				t.Fatalf("routeLabel(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestNormalizeMethod(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"GET", "GET"},
		{"DELETE", "DELETE"},
		{"OPTIONS", "OPTIONS"},
		{"FOO", "OTHER"},
		{"get", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := normalizeMethod(tt.method); got != tt.want {
				t.Fatalf("normalizeMethod(%q) = %q, want %q", tt.method, got, tt.want)
			}
		})
	}
}

func TestMetrics_CountsRequestsByRouteNotPath(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg, []string{"/orders"})

	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	for _, path := range []string{"/orders/1", "/orders/2", "/orders/3"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	// Three different paths must produce ONE series, with value 3.
	want := `
# HELP sentinel_http_requests_total Requests handled by the gateway.
# TYPE sentinel_http_requests_total counter
sentinel_http_requests_total{method="GET",route="/orders",status="418"} 3
`
	if err := testutil.CollectAndCompare(m.requests, strings.NewReader(want)); err != nil {
		t.Fatal(err)
	}
}
