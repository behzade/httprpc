package httprpc

import "net/http"

type Endpoint[Req, Res any] struct {
	Handler Handler[Req, Res]
	Path    string
	Method  string
}

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
}

type EndpointGroup struct {
	Prefix      string
	Handlers    []*endpoint
	Middlewares []*MiddlewareWithPriority
}

func (eg *EndpointGroup) Group(prefix string) *EndpointGroup {
	return &EndpointGroup{
		Prefix:      eg.Prefix + prefix, // should we handle trailing slashes?
		Middlewares: []*MiddlewareWithPriority{},
	}
}

func RegisterHandler[Req any, Res any](codec Codec[Req, Res], eg *EndpointGroup, in Endpoint[Req, Res]) {
	eg.Handlers = append(eg.Handlers, &endpoint{
		Path:    eg.Prefix + in.Path,
		Method:  in.Method,
		Handler: adaptHandler(codec, in.Handler),
	})
}
