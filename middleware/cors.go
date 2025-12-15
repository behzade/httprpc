// Package middleware provides HTTP middleware utilities for the httprpc framework.
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/behzade/httprpc"
)

// CORSConfig configures CORS behavior.
// For production, adjust the allowed origins and methods to your needs.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

// CORS returns middleware applying the provided CORSConfig.
func CORS(cfg CORSConfig) httprpc.Middleware {
	allowOrigins := strings.Join(defaultIfEmpty(cfg.AllowedOrigins, []string{"*"}), ", ")
	allowMethods := strings.Join(defaultIfEmpty(cfg.AllowedMethods, []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}), ", ")
	allowHeaders := strings.Join(defaultIfEmpty(cfg.AllowedHeaders, []string{"Content-Type", "Authorization"}), ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")
	maxAge := cfg.MaxAgeSeconds

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowOrigins)
			w.Header().Set("Access-Control-Allow-Methods", allowMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			if exposeHeaders != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
			}
			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if maxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func defaultIfEmpty(in, fallback []string) []string {
	if len(in) == 0 {
		return fallback
	}
	return in
}
