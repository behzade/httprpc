package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/behzade/httprpc"
)

// Timeout sets a context timeout for each request. If d is zero or negative, it is a no-op.
func Timeout(d time.Duration) httprpc.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if d <= 0 {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
