package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand/v2"
	"net/http"
	"time"
)

func main() {
	name := flag.String("name", "orders", "resource name, also the URL prefix (orders, users, ...)")
	addr := flag.String("addr", ":8081", "listen address")
	delay := flag.Duration("delay", 0, "artificial delay added to every request (e.g. 2s)")
	errRate := flag.Float64("error-rate", 0, "fraction of requests that fail with 500 (0.0 to 1.0)")
	ownerCheck := flag.Bool("owner-check", false, "only let a user read resources they own (uses X-User-ID set by the gateway)")
	flag.Parse()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /"+*name+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s rid=%s xff=%q user=%q auth_header_present=%t", *name, r.Method, r.URL.Path,
			r.Header.Get("X-Request-ID"), r.Header.Get("X-Forwarded-For"), r.Header.Get("X-User-ID"), r.Header.Get("Authorization") != "")

		if *ownerCheck {
			owner, ok := owners[r.PathValue("id")]
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
				return
			}
			// No X-User-ID means the caller authenticated as an app (API key mode), not as a user.
			if user := r.Header.Get("X-User-ID"); user != "" && user != owner {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":       r.PathValue("id"),
			"service":  *name,
			"instance": *addr,
		})
	})

	mux.HandleFunc("POST /"+*name, func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s remote=%s", *name, r.Method, r.URL.Path, r.RemoteAddr)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		body["id"] = rand.IntN(1_000_000)
		body["service"] = *name
		writeJSON(w, http.StatusCreated, body)
	})

	handler := withFaults(mux, *delay, *errRate)

	log.Printf("[%s] listening on %s (delay=%s error-rate=%.2f)", *name, *addr, *delay, *errRate)
	log.Fatal(http.ListenAndServe(*addr, handler))
}

// owners stands in for a database table: which user owns which resource ID.
var owners = map[string]string{
	"1": "user-42",
	"2": "user-42",
	"3": "user-7",
}

// withFaults is middleware: it wraps a handler and injects delay and random failures.
func withFaults(next http.Handler, delay time.Duration, errRate float64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				log.Printf("caller gave up while we were delaying: %v", r.Context().Err())
				return
			}
		}
		if rand.Float64() < errRate {
			http.Error(w, "injected failure", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
