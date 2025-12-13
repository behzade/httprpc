package httprpc

type Router struct {
	*EndpointGroup
}

func NewRouter() *Router {
	eg := &EndpointGroup{}
	eg.root = eg
	return &Router{
		eg,
	}
}

type MiddlewareOption interface {
	apply(*MiddlewareWithPriority)
}

// Use adds a middleware to the router. Priority -> Higher means earlier execution.
func (r *Router) Use(middleware Middleware, middlewareOpts ...MiddlewareOption) {
	r.EndpointGroup.Use(middleware, middlewareOpts...)
}
