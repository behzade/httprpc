package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/behzade/httprpc"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rw *responseRecorder) WriteHeader(status int) {
	if rw.status == 0 {
		rw.status = status
	}
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseRecorder) Write(p []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(p)
	rw.bytes += n
	return n, err
}

// Logging logs request/response metadata using slog.
// If logger is nil, slog.Default is used.
func Logging(logger *slog.Logger) httprpc.Middleware {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &responseRecorder{ResponseWriter: w}

			next.ServeHTTP(rec, r)

			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"duration", time.Since(start),
				"bytes_written", rec.bytes,
			)
		})
	}
}
