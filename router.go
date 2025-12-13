package httprpc

type Router struct {
	*EndpointGroup
}

func NewRouter() *Router {
	return &Router{
		&EndpointGroup{},
	}
}

type MiddlewareOption interface {
	apply(*MiddlewareWithPriority)
}

// Use adds a middleware to the entire router. Priorty -> Higher means earlier execution.
func (r *Router) Use(middleware Middleware, middlewareOpts ...MiddlewareOption) {
	out := &MiddlewareWithPriority{
		Middleware: middleware,
	}

	for _, opt := range middlewareOpts {
		opt.apply(out)
	}
	r.Middlewares = append(r.Middlewares, out)
}
