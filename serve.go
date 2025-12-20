package httprpc

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type routeMethods struct {
	byMethod map[string]http.Handler
	allow    string
}

type pathParamMatch struct {
	values map[string]string
}

func (m pathParamMatch) ok() bool {
	return m.values != nil
}

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

type routePattern struct {
	path     string
	shape    string
	segments []string
	params   []string
	methods  *routeMethods
}

func parseRoutePattern(path string) (*routePattern, error) {
	path = normalizeRoutePath(path)
	if path == "/" {
		return &routePattern{path: path, shape: path, segments: []string{""}}, nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	params := make([]string, 0, len(parts))
	shapeParts := make([]string, 0, len(parts))
	seenParams := map[string]struct{}{}
	for _, part := range parts {
		if strings.Contains(part, "{") || strings.Contains(part, "}") {
			return nil, fmt.Errorf("invalid path segment %q: use :name for params", part)
		}
		if strings.HasPrefix(part, ":") {
			name := strings.TrimPrefix(part, ":")
			if name == "" {
				return nil, fmt.Errorf("invalid path segment %q: missing param name", part)
			}
			if strings.Contains(name, ":") {
				return nil, fmt.Errorf("invalid path segment %q: unexpected ':'", part)
			}
			if !isSnakeCase(name) {
				return nil, fmt.Errorf("invalid path param %q: must be snake_case", name)
			}
			if _, ok := seenParams[name]; ok {
				return nil, fmt.Errorf("duplicate path param %q in %s", name, path)
			}
			seenParams[name] = struct{}{}
			params = append(params, name)
			shapeParts = append(shapeParts, ":")
			continue
		}
		if strings.Contains(part, ":") {
			return nil, fmt.Errorf("invalid path segment %q: use :name for params", part)
		}
		shapeParts = append(shapeParts, part)
	}
	shape := "/" + strings.Join(shapeParts, "/")
	return &routePattern{
		path:     path,
		shape:    shape,
		segments: parts,
		params:   params,
	}, nil
}

func (p *routePattern) match(path string) pathParamMatch {
	if p.path == path {
		return pathParamMatch{values: map[string]string{}}
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(p.segments) != len(parts) {
		return pathParamMatch{}
	}
	values := make(map[string]string, len(p.params))
	for i, seg := range p.segments {
		if name, ok := pathParamName(seg); ok {
			values[name] = parts[i]
			continue
		}
		if seg != parts[i] {
			return pathParamMatch{}
		}
	}
	return pathParamMatch{values: values}
}

func pathParamName(segment string) (string, bool) {
	const minPathParamSegmentLen = 2
	if len(segment) < minPathParamSegmentLen {
		return "", false
	}
	if !strings.HasPrefix(segment, ":") {
		return "", false
	}
	name := strings.TrimSpace(segment[1:])
	if name == "" {
		return "", false
	}
	return name, true
}

// Handler returns an http.Handler that dispatches to registered endpoints.
// Supports exact matches and "/:param" path segments.
func (r *Router) buildHandler() (http.Handler, error) {
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

	staticRoutes := make(map[string]*routeMethods, len(r.Handlers))
	var patternRoutes []*routePattern
	patternByShape := map[string]*routePattern{}
	for _, e := range r.Handlers {
		if e == nil {
			continue
		}

		pattern, err := parseRoutePattern(e.Path)
		if err != nil {
			return nil, fmt.Errorf("invalid route %s %s: %w", e.Method, e.Path, err)
		}

		var m *routeMethods
		if len(pattern.params) > 0 {
			existing := patternByShape[pattern.shape]
			if existing != nil && existing.path != pattern.path {
				return nil, fmt.Errorf("ambiguous route: %s conflicts with %s", pattern.path, existing.path)
			}
			if existing == nil {
				patternByShape[pattern.shape] = pattern
				patternRoutes = append(patternRoutes, pattern)
			} else {
				pattern = existing
			}
			m = pattern.methods
		} else {
			m = staticRoutes[pattern.path]
		}
		if m == nil {
			m = &routeMethods{byMethod: map[string]http.Handler{}}
			if len(pattern.params) > 0 {
				pattern.methods = m
			} else {
				staticRoutes[pattern.path] = m
			}
		}
		if _, exists := m.byMethod[e.Method]; exists {
			return nil, fmt.Errorf("duplicate route: %s %s", e.Method, pattern.path)
		}

		h := e.Handler
		if h == nil {
			h = http.NotFoundHandler()
		}
		h = applyMiddlewares(h, collectMiddlewares(e.Group))
		m.byMethod[e.Method] = h
	}

	for _, m := range staticRoutes {
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
	for _, p := range patternRoutes {
		if p == nil || p.methods == nil || len(p.methods.byMethod) == 0 {
			continue
		}
		methods := make([]string, 0, len(p.methods.byMethod))
		for method := range p.methods.byMethod {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		p.methods.allow = strings.Join(methods, ", ")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestPath := normalizeRequestPath(req.URL.Path)

		m := staticRoutes[requestPath]
		if m != nil {
			h := m.byMethod[req.Method]
			if h == nil {
				if m.allow != "" {
					w.Header().Set("Allow", m.allow)
				}
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			h.ServeHTTP(w, req)
			return
		}

		for _, p := range patternRoutes {
			if p == nil || p.methods == nil {
				continue
			}
			match := p.match(requestPath)
			if !match.ok() {
				continue
			}
			h := p.methods.byMethod[req.Method]
			if h == nil {
				if p.methods.allow != "" {
					w.Header().Set("Allow", p.methods.allow)
				}
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
				return
			}
			req = withPathParams(req, match.values)
			h.ServeHTTP(w, req)
			return
		}

		if fallback != nil {
			fallback.ServeHTTP(w, req)
			return
		}
		http.NotFound(w, req)
	}), nil
}

func normalizeRoutePath(path string) string {
	path = "/" + strings.Trim(path, "/")
	if path == "/" {
		return "/"
	}
	return path
}

func normalizeRequestPath(path string) string {
	path = "/" + strings.Trim(path, "/")
	if path == "/" {
		return "/"
	}
	return path
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
			Meta:     typeRef(m.Meta),
			Res:      typeRef(m.Res),
			Consumes: append([]string(nil), m.Consumes...),
			Produces: append([]string(nil), m.Produces...),
		})
	}
	return out
}
