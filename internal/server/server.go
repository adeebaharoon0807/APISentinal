package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// Run starts the HTTP server and blocks until SIGINT/SIGTERM arrives,
// then shuts down gracefully, letting in-flight requests finish.
func Run(addr string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,  // client must send all headers within 5s
		ReadTimeout:       15 * time.Second, // ...and the whole request (incl. body) within 15s
		WriteTimeout:      30 * time.Second, // we must finish writing the response within 30s
		IdleTimeout:       60 * time.Second, // keep-alive connections idle longer than 60s are closed
	}

	// ctx is cancelled when the process receives Ctrl+C (SIGINT) or SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("sentinel listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	case <-ctx.Done():
		log.Println("shutdown signal received, draining in-flight requests...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Println("shutdown complete")
	return nil
}
