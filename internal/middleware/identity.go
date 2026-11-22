package middleware

import "net/http"

// UserIDHeader carries the authenticated user's ID from the gateway to backends.
// Backends may trust it only because the gateway always removes any copy sent by the client.
const UserIDHeader = "X-User-ID"

// StripIdentityHeaders removes identity headers supplied by the client, so that only
// the gateway can set them after successful authentication. It must run before auth.
func StripIdentityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(UserIDHeader)
		next.ServeHTTP(w, r)
	})
}
