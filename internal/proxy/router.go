package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/adeeba-bipl/api-sentinel/internal/config"
	"github.com/adeeba-bipl/api-sentinel/internal/middleware"
)

// NewRouter builds one reverse proxy per route and registers it by path prefix.
// All proxies share a single Transport, i.e. one connection pool.
func NewRouter(routes []config.Route, logger *slog.Logger) (http.Handler, error) {
	mux := http.NewServeMux()
	transport := newTransport()

	for _, rt := range routes {
		target, err := url.Parse(rt.Upstream)
		if err != nil {
			return nil, fmt.Errorf("route %q: %w", rt.Prefix, err)
		}

		p := &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				pr.SetXForwarded()
			},
			Transport:    transport,
			ErrorHandler: errorHandler(logger, rt.Upstream),
		}

		mux.Handle(rt.Prefix, p)     // exact: POST /orders
		mux.Handle(rt.Prefix+"/", p) // subtree: GET /orders/5
	}
	return mux, nil
}

func newTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second, // connect timeout: give up on a TCP handshake after 2s
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,              // idle connections kept across all backends
		MaxIdleConnsPerHost:   32,               // idle connections kept per backend (default is 2)
		IdleConnTimeout:       90 * time.Second, // close pooled connections unused for 90s
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func errorHandler(logger *slog.Logger, upstream string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		reqID := middleware.RequestIDFrom(r.Context())

		if errors.Is(err, context.Canceled) {
			logger.Info("client closed request", "request_id", reqID, "upstream", upstream)
			w.WriteHeader(499)
			return
		}

		status := http.StatusBadGateway
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			status = http.StatusGatewayTimeout
		}

		logger.Warn("upstream request failed",
			"request_id", reqID,
			"upstream", upstream,
			"status", status,
			"error", err.Error(),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{
			"error":      http.StatusText(status),
			"request_id": reqID,
		})
	}
}
