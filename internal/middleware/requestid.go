package middleware

import (
	"context"
	"crypto/rand"
	"net/http"
)

const RequestIDHeader = "X-Request-ID"

type ctxKey struct{}

// RequestID gives every request a fresh unique ID, forwards it to the backend,
// returns it to the client, and stores it in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := rand.Text()

		r.Header.Set(RequestIDHeader, id)   // the proxy forwards this to the backend
		w.Header().Set(RequestIDHeader, id) // the client sees it in the response

		ctx := context.WithValue(r.Context(), ctxKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFrom returns the request ID stored in ctx, or "" if there is none.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
