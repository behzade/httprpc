package middleware

import (
	"log/slog"
	"net/http"

	"github.com/behzade/httprpc"
)

// Recover returns middleware that recovers from panics and writes a 500 response.
// If logger is nil, slog.Default is used.
func Recover(logger *slog.Logger) httprpc.Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered", "panic", rec, "method", r.Method, "path", r.URL.Path)
					w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
