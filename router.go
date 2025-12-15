package httprpc

import (
	"net/http"
	"sync"
)

// Router is the main router for handling HTTP requests.
type Router struct {
	*EndpointGroup

	tsGenMu  sync.Mutex
	tsGenCfg *TSClientGenConfig
}

// New creates a new Router.
func New() *Router {
	eg := &EndpointGroup{}
	eg.root = eg
	return &Router{
		EndpointGroup: eg,
	}
}

// MiddlewareOption configures middleware options.
type MiddlewareOption interface {
	apply(*MiddlewareWithPriority)
}

// Use adds a middleware to the router. Priority -> Higher means earlier execution.
func (r *Router) Use(middleware Middleware, middlewareOpts ...MiddlewareOption) {
	r.EndpointGroup.Use(middleware, middlewareOpts...)
}

// Handler builds and returns an http.Handler for the registered endpoints.
// It returns an error if routes are invalid (e.g., duplicate method+path).
func (r *Router) Handler() (http.Handler, error) {
	return r.buildHandler()
}

// HandlerMust returns the handler or panics if building the handler fails.
func (r *Router) HandlerMust() http.Handler {
	h, err := r.buildHandler()
	if err != nil {
		panic(err)
	}
	return h
}
