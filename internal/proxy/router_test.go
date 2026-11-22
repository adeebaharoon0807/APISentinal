package proxy

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adeeba-bipl/api-sentinel/internal/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRouter_UpstreamDownReturns502(t *testing.T) {
	// Start a real server, remember its address, then shut it down:
	// now that address is guaranteed to refuse connections.
	upstream := httptest.NewServer(http.NotFoundHandler())
	deadURL := upstream.URL
	upstream.Close()

	router, err := NewRouter([]config.Route{{Prefix: "/orders", Upstream: deadURL}}, discardLogger())
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders/1", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Bad Gateway") {
		t.Fatalf("body = %q, want JSON error", rec.Body.String())
	}
}

func TestRouter_ProxiesToUpstream(t *testing.T) {
	var gotPath, gotXFF string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotXFF = r.Header.Get("X-Forwarded-For")
		io.WriteString(w, "hello from upstream")
	}))
	defer upstream.Close()

	router, err := NewRouter([]config.Route{{Prefix: "/orders", Upstream: upstream.URL}}, discardLogger())
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/orders/42", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "hello from upstream" {
		t.Fatalf("body = %q, want %q", got, "hello from upstream")
	}
	if gotPath != "/orders/42" {
		t.Fatalf("upstream saw path %q, want %q", gotPath, "/orders/42")
	}
	if gotXFF != "192.0.2.1" {
		t.Fatalf("upstream saw X-Forwarded-For %q, want %q", gotXFF, "192.0.2.1")
	}
}
