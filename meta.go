package httprpc

import "reflect"

type EndpointMeta struct {
	Method string
	Path   string

	Req reflect.Type
	Res reflect.Type

	Consumes []string
	Produces []string
}

type TypeRef struct {
	String  string
	Name    string
	PkgPath string
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

type EndpointDescription struct {
	Method string
	Path   string

	Req TypeRef
	Res TypeRef

	Consumes []string
	Produces []string
}
