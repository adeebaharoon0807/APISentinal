package middleware

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	expectedHash := sha256.Sum256([]byte("test-secret"))
	called := false
	headerForwarded := false
	h := APIKeyAuth(expectedHash)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		headerForwarded = r.Header.Get(APIKeyHeader) != ""
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(APIKeyHeader, "test-secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
	if headerForwarded {
		t.Fatal("API key header was forwarded to the downstream handler")
	}
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	testAPIKeyAuthRejects(t, "")
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	testAPIKeyAuthRejects(t, "wrong-secret")
}

func TestAPIKeyAuth_OversizedKey(t *testing.T) {
	testAPIKeyAuthRejects(t, string(make([]byte, MaxAPIKeyLength+1)))
}

func TestAPIKeyAuth_DuplicateHeaders(t *testing.T) {
	expectedHash := sha256.Sum256([]byte("test-secret"))
	called := false
	h := APIKeyAuth(expectedHash)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add(APIKeyHeader, "test-secret")
	req.Header.Add(APIKeyHeader, "test-secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("next handler was called for a request with duplicate API key headers")
	}
	if rec.Body.String() != "unauthorized\n" {
		t.Fatalf("body = %q, want generic unauthorized response", rec.Body.String())
	}
}

func testAPIKeyAuthRejects(t *testing.T, presentedKey string) {
	t.Helper()

	expectedHash := sha256.Sum256([]byte("test-secret"))
	called := false
	h := APIKeyAuth(expectedHash)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if presentedKey != "" {
		req.Header.Set(APIKeyHeader, presentedKey)
	}
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("next handler was called for an unauthorized request")
	}
	if rec.Body.String() != "unauthorized\n" {
		t.Fatalf("body = %q, want generic unauthorized response", rec.Body.String())
	}
}

func TestNewAPIKeyAuthFromEnv_EmptyConfiguration(t *testing.T) {
	t.Setenv(APIKeyEnv, "")

	if _, err := NewAPIKeyAuthFromEnv(); err == nil {
		t.Fatal("NewAPIKeyAuth() returned nil error for empty configuration")
	}
}

func TestNewAPIKeyAuthFromEnv_UnsetConfiguration(t *testing.T) {
	previous, wasSet := os.LookupEnv(APIKeyEnv)
	if err := os.Unsetenv(APIKeyEnv); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(APIKeyEnv, previous)
			return
		}
		_ = os.Unsetenv(APIKeyEnv)
	})

	if _, err := NewAPIKeyAuthFromEnv(); err == nil {
		t.Fatal("NewAPIKeyAuthFromEnv() returned nil error for unset configuration")
	}
}

func TestNewAPIKeyAuthFromEnv_WhitespaceOnlyConfiguration(t *testing.T) {
	t.Setenv(APIKeyEnv, " \t\n ")

	if _, err := NewAPIKeyAuthFromEnv(); err == nil {
		t.Fatal("NewAPIKeyAuthFromEnv() returned nil error for whitespace-only configuration")
	}
}

func TestNewAPIKeyAuthFromEnv_ValidConfiguration(t *testing.T) {
	t.Setenv(APIKeyEnv, "configured-secret")

	auth, err := NewAPIKeyAuthFromEnv()
	if err != nil {
		t.Fatalf("NewAPIKeyAuthFromEnv() error = %v", err)
	}

	called := false
	h := auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(APIKeyHeader, "configured-secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}
