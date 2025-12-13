package httprpc

import (
	"net/http"
	"reflect"
)

type Endpoint[Req, Res any] struct {
	Handler Handler[Req, Res]
	Path    string
	Method  string
}

type HandlerMiddleware[Req, Res any] func(next Handler[Req, Res]) Handler[Req, Res]

func newEndpoint[Req, Res any](handler Handler[Req, Res], path string, method string) Endpoint[Req, Res] {
	return Endpoint[Req, Res]{
		Handler: handler,
		Path:    path,
		Method:  method,
	}
}

func GET[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodGet)
}

func POST[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodPost)
}

func PUT[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodPut)
}

func DELETE[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodDelete)
}

func PATCH[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodPatch)
}

func OPTIONS[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodOptions)
}

func HEAD[Req, Res any](handler Handler[Req, Res], path string) Endpoint[Req, Res] {
	return newEndpoint(handler, path, http.MethodHead)
}

type Middleware func(next http.Handler) http.Handler

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

type EndpointGroup struct {
	Prefix      string
	Handlers    []*endpoint
	Middlewares []*MiddlewareWithPriority

	root   *EndpointGroup
	parent *EndpointGroup

	Metas []*EndpointMeta
}

func (eg *EndpointGroup) Group(prefix string) *EndpointGroup {
	return &EndpointGroup{
		Prefix:      eg.Prefix + prefix, // should we handle trailing slashes?
		Middlewares: []*MiddlewareWithPriority{},
		root:        eg.root,
		parent:      eg,
	}
}

type RegisterOption[Req, Res any] interface {
	apply(*registerOptions[Req, Res])
}

type registerOptions[Req, Res any] struct {
	codec       Codec[Req, Res]
	middlewares []HandlerMiddleware[Req, Res]
}

type registerOptionFunc[Req, Res any] func(*registerOptions[Req, Res])

func (f registerOptionFunc[Req, Res]) apply(o *registerOptions[Req, Res]) { f(o) }

func WithCodec[Req, Res any](codec Codec[Req, Res]) RegisterOption[Req, Res] {
	return registerOptionFunc[Req, Res](func(o *registerOptions[Req, Res]) { o.codec = codec })
}

func WithMiddleware[Req, Res any](middleware HandlerMiddleware[Req, Res]) RegisterOption[Req, Res] {
	return registerOptionFunc[Req, Res](func(o *registerOptions[Req, Res]) {
		o.middlewares = append(o.middlewares, middleware)
	})
}

func WithMiddlewares[Req, Res any](middlewares ...HandlerMiddleware[Req, Res]) RegisterOption[Req, Res] {
	return registerOptionFunc[Req, Res](func(o *registerOptions[Req, Res]) {
		o.middlewares = append(o.middlewares, middlewares...)
	})
}

func RegisterHandler[Req any, Res any](eg *EndpointGroup, in Endpoint[Req, Res], opts ...RegisterOption[Req, Res]) {
	root := eg.root
	if root == nil {
		root = eg
	}

	o := registerOptions[Req, Res]{
		codec: JSONCodec[Req, Res]{},
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
