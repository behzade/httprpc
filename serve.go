package httprpc

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

func collectMiddlewares(group *EndpointGroup) []*MiddlewareWithPriority {
	if group == nil {
		return nil
	}

	var out []*MiddlewareWithPriority
	for g := group; g != nil; g = g.parent {
		out = append(out, g.Middlewares...)
	}
	return out
}

func applyMiddlewares(h http.Handler, middlewares []*MiddlewareWithPriority) http.Handler {
	if len(middlewares) == 0 {
		return h
	}

	ordered := append([]*MiddlewareWithPriority(nil), middlewares...)
	sort.SliceStable(ordered, func(i, j int) bool {
		// Higher priority runs earlier (wraps outer), so we apply it later.
		return ordered[i].Priority < ordered[j].Priority
	})

	for _, mw := range ordered {
		if mw == nil || mw.Middleware == nil {
			continue
		}
		h = mw.Middleware(h)
	}
	return h
}

// Handler returns an http.Handler that dispatches to registered endpoints.
// Current behavior is exact match on r.URL.Path (no templating).
func (r *Router) buildHandler() (http.Handler, error) {
	type methods struct {
		byMethod map[string]http.Handler
		allow    string
	}

	root := r.EndpointGroup
	if root != nil && root.root != nil {
		root = root.root
	}
	if root != nil {
		root.sealed = true
	}

	var fallback http.Handler
	if r.fallback != nil {
		fallback = applyMiddlewares(r.fallback, collectMiddlewares(root))
	}

	byPath := make(map[string]*methods, len(r.Handlers))
	for _, e := range r.Handlers {
		if e == nil {
			continue
		}

		m := byPath[e.Path]
		if m == nil {
			m = &methods{byMethod: map[string]http.Handler{}}
			byPath[e.Path] = m
		}
		if _, exists := m.byMethod[e.Method]; exists {
			return nil, fmt.Errorf("duplicate route: %s %s", e.Method, e.Path)
		}

		h := e.Handler
		if h == nil {
			h = http.NotFoundHandler()
		}
		h = applyMiddlewares(h, collectMiddlewares(e.Group))
		m.byMethod[e.Method] = h
	}

	for _, m := range byPath {
		if len(m.byMethod) == 0 {
			continue
		}
		methods := make([]string, 0, len(m.byMethod))
		for method := range m.byMethod {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		m.allow = strings.Join(methods, ", ")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		m := byPath[req.URL.Path]
		if m == nil {
			if fallback != nil {
				fallback.ServeHTTP(w, req)
				return
			}
			http.NotFound(w, req)
			return
		}
		h := m.byMethod[req.Method]
		if h == nil {
			if m.allow != "" {
				w.Header().Set("Allow", m.allow)
			}
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		h.ServeHTTP(w, req)
	}), nil
}

// Describe returns endpoint metadata suitable for generators.
func (r *Router) Describe() []EndpointDescription {
	out := make([]EndpointDescription, 0, len(r.Metas))
	for _, m := range r.Metas {
		if m == nil {
			continue
		}
		out = append(out, EndpointDescription{
			Method:   m.Method,
			Path:     m.Path,
			Req:      typeRef(m.Req),
			Res:      typeRef(m.Res),
			Consumes: append([]string(nil), m.Consumes...),
			Produces: append([]string(nil), m.Produces...),
		})
	}
	return out
}
