package httprpc

import (
	"net/http"
	"time"
)

type ServerOption interface {
	apply(*http.Server)
}

type serverOptionFunc func(*http.Server)

func (f serverOptionFunc) apply(s *http.Server) { f(s) }

func ReadHeaderTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.ReadHeaderTimeout = d })
}

func ReadTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.ReadTimeout = d })
}

func WriteTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.WriteTimeout = d })
}

func IdleTimeout(d time.Duration) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.IdleTimeout = d })
}

func MaxHeaderBytes(n int) ServerOption {
	return serverOptionFunc(func(s *http.Server) { s.MaxHeaderBytes = n })
}

// Server returns a configured http.Server using Router.Handler().
func (r *Router) Server(addr string, opts ...ServerOption) *http.Server {
	s := &http.Server{
		Addr:              addr,
		Handler:           r.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(s)
		}
	}
	return s
}
