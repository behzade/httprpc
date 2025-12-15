package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecoverMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{AddSource: false}))

	h := Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := strings.TrimSpace(rec.Body.String()); body == "" {
		t.Fatalf("expected body, got empty")
	}
	if log := buf.String(); !strings.Contains(log, "panic recovered") {
		t.Fatalf("expected panic log, got %q", log)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/path", http.NoBody)

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{AddSource: false}))

	h := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	log := buf.String()
	if !strings.Contains(log, "method=GET") || !strings.Contains(log, "path=/path") || !strings.Contains(log, "status=201") {
		t.Fatalf("unexpected log: %q", log)
	}
}

func TestRequestSizeLimit(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", 5)))

	h := RequestSizeLimit(3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
	}))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	h := Timeout(10 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deadline, ok := r.Context().Deadline(); !ok || deadline.IsZero() {
			t.Fatalf("expected deadline to be set")
		}
	}))
	h.ServeHTTP(rec, req)
}

func TestRequestIDMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)

	h := RequestID("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := RequestIDFromContext(r.Context())
		if !ok || id == "" {
			t.Fatalf("expected request id in context")
		}
		if got := w.Header().Get("X-Request-ID"); got == "" {
			t.Fatalf("expected response header set")
		}
	}))
	h.ServeHTTP(rec, req)
}

func TestCORSMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", http.NoBody)

	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"X-Test"},
		ExposeHeaders:    []string{"X-Expose"},
		AllowCredentials: true,
		MaxAgeSeconds:    3600,
	}
	h := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not reach handler on OPTIONS")
	}))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected allow origin %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET" {
		t.Fatalf("unexpected allow methods %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "X-Test" {
		t.Fatalf("unexpected allow headers %q", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "X-Expose" {
		t.Fatalf("unexpected expose headers %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected allow credentials true, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Fatalf("unexpected max age %q", got)
	}
}
