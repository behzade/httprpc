package httprpc

import (
	"net/http"
	"sort"
	"strings"
)

func collectMiddlewares(group *EndpointGroup) []*MiddlewareWithPriority {
	if group == nil {
		return nil
	}

	var chain []*EndpointGroup
	for g := group; g != nil; g = g.parent {
		chain = append(chain, g)
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	var out []*MiddlewareWithPriority
	for _, g := range chain {
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
		return ordered[i].Priority > ordered[j].Priority
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
func (r *Router) Handler() http.Handler {
	type methods struct {
		byMethod map[string]http.Handler
		allow    string
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
			panic("duplicate route: " + e.Method + " " + e.Path)
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
	})
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
