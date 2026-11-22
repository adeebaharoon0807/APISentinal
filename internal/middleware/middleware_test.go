package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecover_PanicBecomes500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	panicky := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	h := Recover(logger)(panicky)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestRequestID(t *testing.T) {
	var headerSeen, ctxSeen string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: save r.Header.Get(RequestIDHeader) into headerSeen
		headerSeen = r.Header.Get(RequestIDHeader)
		// TODO: save RequestIDFrom(r.Context()) into ctxSeen
		ctxSeen = RequestIDFrom(r.Context())
	})
	h := RequestID(next)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.Header.Set(RequestIDHeader, "attacker-chosen")
	h.ServeHTTP(rec, req)

	got := rec.Header().Get(RequestIDHeader)
	// TODO: t.Fatalf if got is empty
	if got == "" {
		t.Fatalf("request id is empty")
	}
	// TODO: t.Fatalf if got == "attacker-chosen"   (client value must be replaced)
	if got == "attacker-chosen" {
		t.Fatalf("request id is attacker-chosen")
	}
	// TODO: t.Fatalf if headerSeen != got           (backend gets the same ID)
	if headerSeen != got {
		t.Fatalf("header seen is not equal to request id")
	}
	// TODO: t.Fatalf if ctxSeen != got              (context carries the same ID)
	if ctxSeen != got {
		t.Fatalf("context seen is not equal to request id")
	}

}
