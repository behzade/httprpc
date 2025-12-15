package middleware

import (
	"net/http"

	"github.com/behzade/httprpc"
)

// RequestSizeLimit limits the readable request body to maxBytes using http.MaxBytesReader.
func RequestSizeLimit(maxBytes int64) httprpc.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes > 0 && r.Body != nil && r.Body != http.NoBody {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
