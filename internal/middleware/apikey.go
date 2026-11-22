package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"
)

const (
	APIKeyHeader    = "X-API-Key"
	APIKeyEnv       = "API_SENTINEL_API_KEY"
	MaxAPIKeyLength = 4096
)

// NewAPIKeyAuthFromEnv creates API key authentication using the configured
// server-side key. The key is read once at startup and never logged.
func NewAPIKeyAuthFromEnv() (Middleware, error) {
	expectedKey, ok := os.LookupEnv(APIKeyEnv)
	if !ok || strings.TrimSpace(expectedKey) == "" {
		return nil, errors.New("API key authentication requires " + APIKeyEnv)
	}

	expectedHash := sha256.Sum256([]byte(expectedKey))
	return APIKeyAuth(expectedHash), nil
}

func APIKeyAuth(expectedHash [sha256.Size]byte) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			values := r.Header.Values(APIKeyHeader)
			if len(values) != 1 || len(values[0]) > MaxAPIKeyLength {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			presentedHash := sha256.Sum256([]byte(values[0]))
			if subtle.ConstantTimeCompare(expectedHash[:], presentedHash[:]) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			r.Header.Del(APIKeyHeader)
			next.ServeHTTP(w, r)
		})
	}
}
