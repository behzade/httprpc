package httprpc

import "reflect"

// EndpointMeta contains metadata about an endpoint.
type EndpointMeta struct {
	Method string
	Path   string

	Req  reflect.Type
	Meta reflect.Type
	Res  reflect.Type

	Consumes []string
	Produces []string
}

// TypeRef represents a reference to a Go type.
type TypeRef struct {
	String  string
	Name    string
	PkgPath string
}

// EndpointDescription describes an endpoint for TypeScript generation.
type EndpointDescription struct {
	Method string
	Path   string

	Req  TypeRef
	Meta TypeRef
	Res  TypeRef

	Consumes []string
	Produces []string
}

func typeRef(t reflect.Type) TypeRef {
	if t == nil {
		return TypeRef{}
	}
	return TypeRef{
		String:  t.String(),
		Name:    t.Name(),
		PkgPath: t.PkgPath(),
	}
}
