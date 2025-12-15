package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/behzade/httprpc"
)

type ctxKey int

const requestIDKey ctxKey = iota

// RequestID injects a request ID into the context and response headers.
// If headerName is empty, "X-Request-ID" is used.
// If the incoming request already has the header, it is propagated.
func RequestID(headerName string) httprpc.Middleware {
	if headerName == "" {
		headerName = "X-Request-ID"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(headerName)
			if id == "" {
				var b [16]byte
				if _, err := rand.Read(b[:]); err == nil {
					id = hex.EncodeToString(b[:])
				}
			}
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			w.Header().Set(headerName, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestIDFromContext returns the request ID set by RequestID middleware, if present.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}
