package httprpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
	defaultShutdownTimeout   = 30 * time.Second
)

// ServerOption configures server options.
type ServerOption interface {
	apply(*http.Server)
}

// ReadHeaderTimeout sets the ReadHeaderTimeout for the server.
func ReadHeaderTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.ReadHeaderTimeout = d })
}

// ReadTimeout sets the ReadTimeout for the server.
func ReadTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.ReadTimeout = d })
}

// WriteTimeout sets the WriteTimeout for the server.
func WriteTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.WriteTimeout = d })
}

// IdleTimeout sets the IdleTimeout for the server.
func IdleTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.IdleTimeout = d })
}

// MaxHeaderBytes sets the MaxHeaderBytes for the server.
func MaxHeaderBytes(n int) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.MaxHeaderBytes = n })
}

type serverOptionFunc func(*http.Server)

func (f serverOptionFunc) apply(s *http.Server) { f(s) }

// Server returns a configured http.Server using Router.Handler().
func (r *Router) Server(addr string, opts ...ServerOption) *http.Server {
	handler := r.HandlerMust()

	s := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(s)
		}
	}
	return s
}

// RunServerOption configures RunServer behavior.
type RunServerOption interface {
	apply(*runServerConfig)
}

type runServerConfig struct {
	gracefulShutdown bool
	shutdownTimeout  time.Duration
	logger           *slog.Logger
}

func (c *runServerConfig) withDefaults() {
	if c.logger == nil {
		c.logger = slog.Default()
	}
	if c.shutdownTimeout <= 0 {
		c.shutdownTimeout = defaultShutdownTimeout
	}
}

type runServerOptionFunc func(*runServerConfig)

func (f runServerOptionFunc) apply(c *runServerConfig) { f(c) }

// WithGracefulShutdown enables graceful shutdown on SIGINT/SIGTERM signals.
// This is the default behavior. Use this option to explicitly enable it or
// to customize the shutdown timeout.
func WithGracefulShutdown(timeout time.Duration) RunServerOption {
	return runServerOptionFunc(func(c *runServerConfig) {
		c.gracefulShutdown = true
		c.shutdownTimeout = timeout
	})
}

// WithLogger sets a custom logger for server lifecycle events.
// If not provided, slog.Default() is used.
func WithLogger(logger *slog.Logger) RunServerOption {
	return runServerOptionFunc(func(c *runServerConfig) {
		c.logger = logger
	})
}

// RunServer runs the HTTP server. By default, it enables graceful shutdown
// with a 30-second timeout. It blocks until the server is shut down.
//
// Options:
//   - WithGracefulShutdown(timeout): Enable graceful shutdown (default: enabled with 30s timeout)
//   - WithLogger(logger): Custom logger for server events (default: slog.Default())
//
// Example:
//
//	// Simple usage with defaults (graceful shutdown enabled)
//	r.RunServer(":8080")
//
//	// Custom timeout
//	r.RunServer(":8080", httprpc.WithGracefulShutdown(60*time.Second))
//
//	// Custom logger
//	r.RunServer(":8080", httprpc.WithLogger(myLogger))
func (r *Router) RunServer(addr string, opts ...RunServerOption) error {
	cfg := runServerConfig{
		gracefulShutdown: true, // default to graceful shutdown
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&cfg)
		}
	}
	cfg.withDefaults()

	server := r.Server(addr)

	if !cfg.gracefulShutdown {
		cfg.logger.Info("starting http server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}

	// Channel to receive server errors
	serverErrors := make(chan error, 1)

	// Start server in goroutine
	go func() {
		cfg.logger.Info("starting http server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		cfg.logger.Info("received shutdown signal", "signal", sig.String())
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
	defer cancel()

	cfg.logger.Info("shutting down server gracefully", "timeout", cfg.shutdownTimeout)
	if err := server.Shutdown(ctx); err != nil {
		cfg.logger.Error("server shutdown failed", "error", err)
		return fmt.Errorf("server shutdown: %w", err)
	}

	cfg.logger.Info("server stopped")
	return nil
}
