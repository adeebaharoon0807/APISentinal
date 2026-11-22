package middleware

import "net/http"

// Middleware wraps a handler with extra behaviour.
type Middleware func(http.Handler) http.Handler

// Chain wraps h so that mws[0] is the outermost layer (it runs first).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
