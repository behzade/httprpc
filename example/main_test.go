package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSPAHandler(t *testing.T) {
	staticFS, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		t.Fatalf("failed to get static fs: %v", err)
	}
	handler := spaHandler(staticFS)

	tests := []struct {
		name string
		path string
	}{
		{name: "root", path: "/"},
		{name: "fallback", path: "/missing"},
		{name: "direct index", path: "/index.html"},
		{name: "static file", path: "/favicon.ico"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (Location: %q)", rec.Code, rec.Header().Get("Location"))
			}
			if loc := rec.Header().Get("Location"); loc != "" {
				t.Fatalf("unexpected redirect to %q", loc)
			}
		})
	}
}
