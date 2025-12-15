package middleware

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/behzade/httprpc"
)

// ReverseProxyConfig configures the reverse proxy middleware/handler.
// Target is required.
type ReverseProxyConfig struct {
	Target *url.URL

	// MatchPrefix, if set, limits proxying to paths with this prefix.
	// When used with the middleware form, requests that don't match fall through to next.
	MatchPrefix string

	// StripPrefix removes the given prefix from the request path before forwarding.
	StripPrefix string

	// PreserveHost keeps the incoming Host header instead of rewriting to the target's host.
	PreserveHost bool

	// ErrorHandler handles proxy errors. If nil, a 502 Bad Gateway is returned.
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// ReverseProxyHandler returns an http.Handler that proxies requests to the target.
func ReverseProxyHandler(cfg ReverseProxyConfig) http.Handler {
	if cfg.Target == nil {
		panic("middleware.ReverseProxyHandler: Target is required")
	}

	proxy := httputil.NewSingleHostReverseProxy(cfg.Target)
	origDirector := proxy.Director

	strip := cfg.StripPrefix
	preserveHost := cfg.PreserveHost
	proxy.Director = func(req *http.Request) {
		origHost := req.Host
		origDirector(req)

		if strip != "" && strings.HasPrefix(req.URL.Path, strip) {
			req.URL.Path = ensureLeadingSlash(strings.TrimPrefix(req.URL.Path, strip))
		}
		if preserveHost {
			req.Host = origHost
		}
	}

	if cfg.ErrorHandler != nil {
		proxy.ErrorHandler = cfg.ErrorHandler
	} else {
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		}
	}

	return proxy
}

// ReverseProxy returns a middleware that proxies matching requests to the target.
// If MatchPrefix is set and the path doesn't match, the request is passed to the next handler.
func ReverseProxy(cfg ReverseProxyConfig) httprpc.Middleware {
	handler := ReverseProxyHandler(cfg)
	matchPrefix := cfg.MatchPrefix

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if matchPrefix != "" && !strings.HasPrefix(r.URL.Path, matchPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
}

func ensureLeadingSlash(p string) string {
	if p == "" || strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}
