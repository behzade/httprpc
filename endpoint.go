package httprpc

import (
	"log/slog"
	"net/http"
	"reflect"
)

// Endpoint represents an HTTP endpoint with a handler, path, and method.
type Endpoint[Req, Res any] struct {
	Handler Handler[Req, Res]
	Path    string
	Method  string
}

// EndpointWithMeta represents an HTTP endpoint with typed request metadata.
type EndpointWithMeta[Req, Meta, Res any] struct {
	Handler HandlerWithMeta[Req, Meta, Res]
	Path    string
	Method  string
}

// HandlerMiddleware is a middleware function for typed handlers.
type HandlerMiddleware[Req, Res any] func(next Handler[Req, Res]) Handler[Req, Res]

// HandlerWithMetaMiddleware is a middleware function for typed handlers with metadata.
type HandlerWithMetaMiddleware[Req, Meta, Res any] func(next HandlerWithMeta[Req, Meta, Res]) HandlerWithMeta[Req, Meta, Res]

func newEndpoint[Req, Res any](handler Handler[Req, Res], path, method string) Endpoint[Req, Res] {
	return Endpoint[Req, Res]{
		Handler: handler,
		Path:    path,
		Method:  method,
	}
}

func newEndpointWithMeta[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path, method string) EndpointWithMeta[Req, Meta, Res] {
	return EndpointWithMeta[Req, Meta, Res]{
		Handler: handler,
		Path:    path,
		Method:  method,
	}
}

// GET creates an Endpoint for HTTP GET requests.
func GET[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodGet)
}

// POST creates an Endpoint for HTTP POST requests.
func POST[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodPost)
}

// PUT creates an Endpoint for HTTP PUT requests.
func PUT[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodPut)
}

// DELETE creates an Endpoint for HTTP DELETE requests.
func DELETE[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodDelete)
}

// PATCH creates an Endpoint for HTTP PATCH requests.
func PATCH[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodPatch)
}

// OPTIONS creates an Endpoint for HTTP OPTIONS requests.
func OPTIONS[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodOptions)
}

// HEAD creates an Endpoint for HTTP HEAD requests.
func HEAD[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodHead)
}

// GETM creates an EndpointWithMeta for HTTP GET requests.
func GETM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodGet)
}

// POSTM creates an EndpointWithMeta for HTTP POST requests.
func POSTM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodPost)
}

// PUTM creates an EndpointWithMeta for HTTP PUT requests.
func PUTM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodPut)
}

// DELETEM creates an EndpointWithMeta for HTTP DELETE requests.
func DELETEM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodDelete)
}

// PATCHM creates an EndpointWithMeta for HTTP PATCH requests.
func PATCHM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodPatch)
}

// OPTIONSM creates an EndpointWithMeta for HTTP OPTIONS requests.
func OPTIONSM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodOptions)
}

// HEADM creates an EndpointWithMeta for HTTP HEAD requests.
func HEADM[Req, Meta, Res any](handler HandlerWithMeta[Req, Meta, Res], path string) EndpointWithMeta[Req, Meta, Res] {
	return newEndpointWithMeta(handler, path, http.MethodHead)
}

// Middleware is a function that wraps an http.Handler.
type Middleware func(next http.Handler) http.Handler

// MiddlewareWithPriority associates a middleware with a priority level.
type MiddlewareWithPriority struct {
	Middleware Middleware
	Priority   int
}

type endpoint struct {
	Path    string
	Method  string
	Handler http.Handler
	Group   *EndpointGroup
}

// EndpointGroup groups endpoints with a common prefix and middlewares.
type EndpointGroup struct {
	Prefix      string
	Handlers    []*endpoint
	Middlewares []*MiddlewareWithPriority

	root   *EndpointGroup
	parent *EndpointGroup

	Metas []*EndpointMeta

	sealed bool
}

// Group creates a subgroup with the given prefix.
func (eg *EndpointGroup) Group(prefix string) *EndpointGroup {
	return &EndpointGroup{
		Prefix:      eg.Prefix + prefix, // should we handle trailing slashes?
		Middlewares: []*MiddlewareWithPriority{},
		root:        eg.root,
		parent:      eg,
	}
}

// RegisterOption configures options for registering a handler.
type RegisterOption[Req, Res any] interface {
	apply(*registerOptions[Req, Res])
}

type registerOptions[Req, Res any] struct {
	codec       Codec[Req, Res]
	middlewares []HandlerMiddleware[Req, Res]
}

type registerOptionFunc[Req, Res any] func(*registerOptions[Req, Res])

func (f registerOptionFunc[Req, Res]) apply(o *registerOptions[Req, Res]) { f(o) } //nolint:unused // interface method

// RegisterOptionWithMeta configures options for registering a handler with metadata.
type RegisterOptionWithMeta[Req, Meta, Res any] interface {
	apply(*registerOptionsWithMeta[Req, Meta, Res])
}

type registerOptionsWithMeta[Req, Meta, Res any] struct {
	codec       Codec[Req, Res]
	middlewares []HandlerWithMetaMiddleware[Req, Meta, Res]
}

type registerOptionWithMetaFunc[Req, Meta, Res any] func(*registerOptionsWithMeta[Req, Meta, Res])

//nolint:unused // interface method
func (f registerOptionWithMetaFunc[Req, Meta, Res]) apply(o *registerOptionsWithMeta[Req, Meta, Res]) {
	f(o)
}

// WithCodec sets the codec for the handler.
func WithCodec[Req, Res any](codec Codec[Req, Res]) RegisterOption[Req, Res] {
	return registerOptionFunc[Req, Res](func(o *registerOptions[Req, Res]) { o.codec = codec })
}

// WithCodecWithMeta sets the codec for the handler with metadata.
func WithCodecWithMeta[Req, Meta, Res any](codec Codec[Req, Res]) RegisterOptionWithMeta[Req, Meta, Res] {
	return registerOptionWithMetaFunc[Req, Meta, Res](func(o *registerOptionsWithMeta[Req, Meta, Res]) { o.codec = codec })
}

// WithMiddleware adds a middleware to the handler.
func WithMiddleware[Req, Res any](middleware HandlerMiddleware[Req, Res]) RegisterOption[Req, Res] {
	return registerOptionFunc[Req, Res](func(o *registerOptions[Req, Res]) {
		o.middlewares = append(o.middlewares, middleware)
	})
}

// WithMiddlewares adds multiple middlewares to the handler.
func WithMiddlewares[Req, Res any](middlewares ...HandlerMiddleware[Req, Res]) RegisterOption[Req, Res] {
	return registerOptionFunc[Req, Res](func(o *registerOptions[Req, Res]) {
		o.middlewares = append(o.middlewares, middlewares...)
	})
}

// WithMetaMiddleware adds a middleware to the handler with metadata.
func WithMetaMiddleware[Req, Meta, Res any](middleware HandlerWithMetaMiddleware[Req, Meta, Res]) RegisterOptionWithMeta[Req, Meta, Res] {
	return registerOptionWithMetaFunc[Req, Meta, Res](func(o *registerOptionsWithMeta[Req, Meta, Res]) {
		o.middlewares = append(o.middlewares, middleware)
	})
}

// WithMetaMiddlewares adds multiple middlewares to the handler with metadata.
func WithMetaMiddlewares[Req, Meta, Res any](middlewares ...HandlerWithMetaMiddleware[Req, Meta, Res]) RegisterOptionWithMeta[Req, Meta, Res] {
	return registerOptionWithMetaFunc[Req, Meta, Res](func(o *registerOptionsWithMeta[Req, Meta, Res]) {
		o.middlewares = append(o.middlewares, middlewares...)
	})
}

// RegisterHandler registers an endpoint with the endpoint group.
func RegisterHandler[Req, Res any](eg *EndpointGroup, in Endpoint[Req, Res], opts ...RegisterOption[Req, Res]) {
	root := eg.root
	if root == nil {
		root = eg
	}
	if root.sealed {
		slog.Error("cannot register handlers after handler is built", "method", in.Method, "path", in.Path)
		return
	}
	if root.sealed {
		slog.Error("cannot register handlers after handler is built", "method", in.Method, "path", in.Path)
		return
	}

	o := registerOptions[Req, Res]{
		codec: DefaultCodec[Req, Res]{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&o)
		}
	}

	codec := o.codec
	handler := in.Handler
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		mw := o.middlewares[i]
		if mw == nil {
			continue
		}
		handler = mw(handler)
	}

	path := eg.Prefix + in.Path
	root.Handlers = append(root.Handlers, &endpoint{
		Path:    eg.Prefix + in.Path,
		Method:  in.Method,
		Handler: adaptHandler(codec, handler),
		Group:   eg,
	})

	var consumes, produces []string
	if ct, ok := any(codec).(interface {
		Consumes() []string
		Produces() []string
	}); ok {
		consumes = ct.Consumes()
		produces = ct.Produces()
	}

	root.Metas = append(root.Metas, &EndpointMeta{
		Method:   in.Method,
		Path:     path,
		Req:      reflect.TypeFor[Req](),
		Res:      reflect.TypeFor[Res](),
		Consumes: consumes,
		Produces: produces,
	})
}

// RegisterHandlerM registers an endpoint with typed metadata.
func RegisterHandlerM[Req, Meta, Res any](eg *EndpointGroup, in EndpointWithMeta[Req, Meta, Res], opts ...RegisterOptionWithMeta[Req, Meta, Res]) {
	root := eg.root
	if root == nil {
		root = eg
	}
	if root.sealed {
		slog.Error("cannot register handlers after handler is built", "method", in.Method, "path", in.Path)
		return
	}
	if root.sealed {
		slog.Error("cannot register handlers after handler is built", "method", in.Method, "path", in.Path)
		return
	}

	metaType := reflect.TypeFor[Meta]()
	if metaType != nil && deref(metaType).Kind() != reflect.Struct {
		slog.Error("request meta type must be a struct", "method", in.Method, "path", in.Path, "type", metaType.String())
		return
	}

	o := registerOptionsWithMeta[Req, Meta, Res]{
		codec: DefaultCodec[Req, Res]{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&o)
		}
	}

	codec := o.codec
	handler := in.Handler
	for i := len(o.middlewares) - 1; i >= 0; i-- {
		mw := o.middlewares[i]
		if mw == nil {
			continue
		}
		handler = mw(handler)
	}

	path := eg.Prefix + in.Path
	if err := validateMetaType(metaType, path); err != nil {
		slog.Error("invalid request meta", "method", in.Method, "path", path, "error", err)
		return
	}
	root.Handlers = append(root.Handlers, &endpoint{
		Path:    eg.Prefix + in.Path,
		Method:  in.Method,
		Handler: adaptHandlerWithMeta(codec, handler),
		Group:   eg,
	})

	var consumes, produces []string
	if ct, ok := any(codec).(interface {
		Consumes() []string
		Produces() []string
	}); ok {
		consumes = ct.Consumes()
		produces = ct.Produces()
	}

	root.Metas = append(root.Metas, &EndpointMeta{
		Method:   in.Method,
		Path:     path,
		Req:      reflect.TypeFor[Req](),
		Meta:     metaType,
		Res:      reflect.TypeFor[Res](),
		Consumes: consumes,
		Produces: produces,
	})
}
