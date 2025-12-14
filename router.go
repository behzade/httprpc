package httprpc

import "sync"

type Router struct {
	*EndpointGroup

	tsGenMu       sync.Mutex
	tsGenLastDir  string
	tsGenLastHash string
	tsGenCfg      *TSClientGenConfig
}

func New() *Router {
	eg := &EndpointGroup{}
	eg.root = eg
	return &Router{
		EndpointGroup: eg,
	}
}

type MiddlewareOption interface {
	apply(*MiddlewareWithPriority)
}

// Use adds a middleware to the router. Priority -> Higher means earlier execution.
func (r *Router) Use(middleware Middleware, middlewareOpts ...MiddlewareOption) {
	r.EndpointGroup.Use(middleware, middlewareOpts...)
}
