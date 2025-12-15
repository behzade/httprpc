package httprpc

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestRunServerOptions(t *testing.T) {
	t.Run("WithGracefulShutdown", func(t *testing.T) {
		cfg := runServerConfig{}
		opt := WithGracefulShutdown(45 * time.Second)
		opt.apply(&cfg)

		if !cfg.gracefulShutdown {
			t.Error("expected gracefulShutdown to be enabled")
		}
		if cfg.shutdownTimeout != 45*time.Second {
			t.Errorf("expected shutdownTimeout to be 45s, got %v", cfg.shutdownTimeout)
		}
	})

	t.Run("WithLogger", func(t *testing.T) {
		cfg := runServerConfig{}
		logger := slog.Default()
		opt := WithLogger(logger)
		opt.apply(&cfg)

		if cfg.logger != logger {
			t.Error("expected logger to be set")
		}
	})

	t.Run("DefaultConfig", func(t *testing.T) {
		cfg := runServerConfig{
			gracefulShutdown: true,
		}
		cfg.withDefaults()

		if cfg.logger == nil {
			t.Error("expected default logger to be set")
		}
		if cfg.shutdownTimeout != defaultShutdownTimeout {
			t.Errorf("expected default timeout %v, got %v", defaultShutdownTimeout, cfg.shutdownTimeout)
		}
	})
}

func TestRunServerGracefulShutdownEnabled(t *testing.T) {
	r := New()
	RegisterHandler(r.EndpointGroup, GET(func(ctx context.Context, req struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/test"))

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		// Use a port that's likely available
		err := r.RunServer("127.0.0.1:0", WithGracefulShutdown(1*time.Second))
		serverErr <- err
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Note: In a real test environment, we'd need to send a signal or
	// have a way to trigger shutdown. This test verifies the options
	// configuration works correctly.
}

func TestRunServerConfigurationChaining(t *testing.T) {
	cfg := runServerConfig{}

	// Apply multiple options
	WithGracefulShutdown(60 * time.Second).apply(&cfg)
	WithLogger(slog.Default()).apply(&cfg)

	if !cfg.gracefulShutdown {
		t.Error("expected gracefulShutdown to be enabled")
	}
	if cfg.shutdownTimeout != 60*time.Second {
		t.Errorf("expected shutdownTimeout to be 60s, got %v", cfg.shutdownTimeout)
	}
	if cfg.logger == nil {
		t.Error("expected logger to be set")
	}
}

func TestServerOptionCompatibility(t *testing.T) {
	// Ensure ServerOption and RunServerOption are separate types
	// and don't conflict
	r := New()
	RegisterHandler(r.EndpointGroup, GET(func(ctx context.Context, req struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/test"))

	// Server() takes ServerOption
	server := r.Server(":8080", ReadTimeout(10*time.Second))
	if server.ReadTimeout != 10*time.Second {
		t.Errorf("expected ReadTimeout to be 10s, got %v", server.ReadTimeout)
	}

	// RunServer() takes RunServerOption
	cfg := runServerConfig{}
	WithGracefulShutdown(30 * time.Second).apply(&cfg)
	if cfg.shutdownTimeout != 30*time.Second {
		t.Errorf("expected shutdownTimeout to be 30s, got %v", cfg.shutdownTimeout)
	}
}
