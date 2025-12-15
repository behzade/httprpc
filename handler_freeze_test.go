package httprpc

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestRegisterAfterHandlerLogs(t *testing.T) {
	r := New()
	RegisterHandler(r.EndpointGroup, GET(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/early"))

	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	_, _ = r.Handler()

	// Try to register after building handler (should log error instead of panic)
	RegisterHandler(r.EndpointGroup, GET(func(context.Context, struct{}) (struct{}, error) {
		return struct{}{}, nil
	}, "/late"))

	// Check that error was logged
	logOutput := buf.String()
	if !strings.Contains(logOutput, "cannot register handlers after handler is built") {
		t.Errorf("expected error log about registration after handler build, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "method=GET") {
		t.Errorf("expected method in log output, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "path=/late") {
		t.Errorf("expected path in log output, got: %s", logOutput)
	}
}
