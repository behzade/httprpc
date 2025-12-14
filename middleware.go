package httprpc

type middlewarePriority int

func (p middlewarePriority) apply(out *MiddlewareWithPriority) {
	out.Priority = int(p)
}

// Priority sets middleware ordering. Higher priority runs earlier (wraps outer).
// For equal priority, later-registered middleware wraps outer.
func Priority(priority int) MiddlewareOption {
	return middlewarePriority(priority)
}

// Use adds a middleware to this group (and its sub-groups).
func (eg *EndpointGroup) Use(middleware Middleware, middlewareOpts ...MiddlewareOption) {
	out := &MiddlewareWithPriority{
		Middleware: middleware,
	}

	for _, opt := range middlewareOpts {
		opt.apply(out)
	}
	eg.Middlewares = append(eg.Middlewares, out)
}
