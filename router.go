package httprpc

import "sync"

// Router is the main router for handling HTTP requests.
type Router struct {
	*EndpointGroup

	tsGenMu       sync.Mutex
	tsGenLastDir  string
	tsGenLastHash string
	tsGenCfg      *TSClientGenConfig
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
