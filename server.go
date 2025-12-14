package httprpc

import (
	"net/http"
	"time"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
)

// ServerOption configures server options.
type ServerOption interface {
	apply(*http.Server)
}

// ReadHeaderTimeout sets the ReadHeaderTimeout for the server.
func ReadHeaderTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.ReadHeaderTimeout = d })
}

// ReadTimeout sets the ReadTimeout for the server.
func ReadTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.ReadTimeout = d })
}

// WriteTimeout sets the WriteTimeout for the server.
func WriteTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.WriteTimeout = d })
}

// IdleTimeout sets the IdleTimeout for the server.
func IdleTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.IdleTimeout = d })
}

// MaxHeaderBytes sets the MaxHeaderBytes for the server.
func MaxHeaderBytes(n int) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.MaxHeaderBytes = n })
}

type serverOptionFunc func(*http.Server)

func (f serverOptionFunc) apply(s *http.Server) { f(s) }

// Server returns a configured http.Server using Router.Handler().
func (r *Router) Server(addr string, opts ...ServerOption) *http.Server {
	s := &http.Server{
		Addr:              addr,
		Handler:           r.Handler(),
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(s)
		}
	}
	return s
}
