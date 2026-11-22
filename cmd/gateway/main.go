package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/adeeba-bipl/api-sentinel/internal/config"
	"github.com/adeeba-bipl/api-sentinel/internal/middleware"
	"github.com/adeeba-bipl/api-sentinel/internal/proxy"
	"github.com/adeeba-bipl/api-sentinel/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	configPath := flag.String("config", "configs/gateway.json", "path to gateway config")
	adminAddr := flag.String("admin-addr", ":9091", "address for the admin server (/metrics)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	prefixes := make([]string, 0, len(cfg.Routes))
	for _, rt := range cfg.Routes {
		prefixes = append(prefixes, rt.Prefix)
	}
	metrics := middleware.NewMetrics(reg, prefixes)

	go serveAdmin(*adminAddr, reg)

	router, err := proxy.NewRouter(cfg.Routes, logger)
	if err != nil {
		log.Fatal(err)
	}

	authMode := os.Getenv(middleware.AuthModeEnv)
	if authMode == "" {
		authMode = middleware.AuthModeAPIKey
	}

	var auth middleware.Middleware
	switch authMode {
	case middleware.AuthModeAPIKey:
		auth, err = middleware.NewAPIKeyAuthFromEnv()
	case middleware.AuthModeJWT:
		auth, err = middleware.NewJWTAuthFromEnv()
	default:
		err = fmt.Errorf("unsupported authentication mode %q", authMode)
	}
	if err != nil {
		log.Fatal(err)
	}

	handler := middleware.Chain(router,
		middleware.RequestID,
		middleware.Logging(logger),
		metrics.Middleware,
		middleware.Recover(logger),
		middleware.StripIdentityHeaders,
		auth,
	)

	log.Printf("loaded %d routes", len(cfg.Routes))
	if err := server.Run(cfg.ListenAddr, handler); err != nil {
		log.Fatal(err)
	}
}

// serveAdmin exposes internal endpoints (/metrics) on a separate port that is not
// published to clients and is not behind client authentication.
func serveAdmin(addr string, reg *prometheus.Registry) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("admin listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("admin server: %v", err)
	}
}
