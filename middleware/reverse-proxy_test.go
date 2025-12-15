package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestReverseProxyHandler_StripsPrefixAndRewritesHost(t *testing.T) {
	var gotPath, gotHost string
	upstream := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))

	targetURL, _ := url.Parse(upstream.URL)
	handler := ReverseProxyHandler(ReverseProxyConfig{
		Target:      targetURL,
		StripPrefix: "/ui",
	})

	req := httptest.NewRequest(http.MethodGet, "/ui/static/app.js", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if gotPath != "/static/app.js" {
		t.Fatalf("expected path /static/app.js, got %q", gotPath)
	}
	if gotHost != targetURL.Host {
		t.Fatalf("expected host %q, got %q", targetURL.Host, gotHost)
	}
}

func TestReverseProxyMiddleware_MatchPrefixFallsThrough(t *testing.T) {
	upstream := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	targetURL, _ := url.Parse(upstream.URL)
	proxyMW := ReverseProxy(ReverseProxyConfig{
		Target:      targetURL,
		MatchPrefix: "/proxy",
		StripPrefix: "/proxy",
	})

	nextCalled := false
	handler := proxyMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/other", http.NoBody)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTeapot {
		t.Fatalf("expected fallback next status 418, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatalf("expected next handler to be called when prefix does not match")
	}

	// matching path should proxy (next not called)
	nextCalled = false
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/proxy/hello", http.NoBody)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected proxied status 200, got %d", rec.Code)
	}
	if nextCalled {
		t.Fatalf("expected proxy to short-circuit next handler")
	}
}

func newTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("unable to open listener: %v", err)
	}
	srv := httptest.NewUnstartedServer(h)
	srv.Listener = ln
	srv.Start()
	t.Cleanup(srv.Close)
	return srv
}
