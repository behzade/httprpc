package httprpc

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	_ "embed"
)

const (
	rootModule  = "root"
	unknownType = "unknown"
	dirPerm     = 0o755
	filePerm    = 0o644
)

// TSGenOptions configures TypeScript generation.
type TSGenOptions struct {
	PackageName string
	ClientName  string
	// SkipPathSegments skips leading path segments when choosing a module file.
	// Example: for "/v1/users/list" and SkipPathSegments=1, the module is "users".
	SkipPathSegments int
}

func (o TSGenOptions) withDefaults() TSGenOptions {
	if o.PackageName == "" {
		o.PackageName = "httprpc"
	}
	if o.ClientName == "" {
		o.ClientName = "Client"
	}
	return o
}

type tsEndpointModel struct {
	Method     string
	Path       string
	MethodName string
	ReqType    string
	ResType    string
	Consumes   string
	Produces   string
	HasBody    bool
	HasParams  bool
}

type tsModel struct {
	PackageName string
	ClientName  string
	Endpoints   []tsEndpointModel
	TypeDefs    []string
}

//go:embed templates/ts/client.tmpl
var tsClientTemplate string

//go:embed templates/ts/base.tmpl
var tsBaseTemplate string

//go:embed templates/ts/module.tmpl
var tsModuleTemplate string

//go:embed templates/ts/index.tmpl
var tsIndexTemplate string

// GenTS writes a TypeScript client based on registered endpoint metadata.
//
// Intended usage with go:generate:
//
//	//go:generate go run ./cmd/gen
//
// where ./cmd/gen constructs your router and calls router.GenTS(...).
func (r *Router) GenTS(w io.Writer, opts TSGenOptions) error {
	opts = opts.withDefaults()
	meta := r.Metas

	types := collectTypes(meta)
	typeNames := assignTypeNames(types)

	orderedTypes := orderedByName(typeNames)
	typeDefs := make([]string, 0, len(orderedTypes))
	for _, t := range orderedTypes {
		name := typeNames[t]
		def, err := tsTypeDef(t, name, typeNames)
		if err != nil {
			return err
		}
		if def != "" {
			typeDefs = append(typeDefs, def)
		}
	}

	endpoints := make([]tsEndpointModel, 0, len(meta))
	for _, m := range meta {
		if m == nil {
			continue
		}
		reqType := typeNames[deref(m.Req)]
		resType := typeNames[deref(m.Res)]
		hasBody := endpointHasBody(m.Method, m.Req)
		hasParams := endpointHasParams(m.Req)

		endpoints = append(endpoints, tsEndpointModel{
			Method:     strings.ToUpper(m.Method),
			Path:       m.Path,
			MethodName: endpointMethodName(m.Method, m.Path),
			ReqType:    reqType,
			ResType:    resType,
			Consumes:   firstOr(m.Consumes),
			Produces:   firstOr(m.Produces),
			HasBody:    hasBody,
			HasParams:  hasParams,
		})
	}
	sort.SliceStable(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	tmpl, err := template.New("ts").
		Funcs(template.FuncMap{"quote": strconv.Quote}).
		Parse(tsClientTemplate)
	if err != nil {
		return fmt.Errorf("parse client template: %w", err)
	}

	model := tsModel{
		PackageName: opts.PackageName,
		ClientName:  opts.ClientName,
		Endpoints:   endpoints,
		TypeDefs:    typeDefs,
	}

	var buf bytes.Buffer
	if execErr := tmpl.Execute(&buf, model); execErr != nil {
		return fmt.Errorf("execute client template: %w", execErr)
	}
	if _, err = io.Copy(w, &buf); err != nil {
		return fmt.Errorf("copy output: %w", err)
	}
	return nil
}

// GenTSDir writes a multi-file TypeScript client into dir, split by path segment.
// It overwrites the generated files it creates.
func (r *Router) GenTSDir(dir string, opts TSGenOptions) error {
	opts = opts.withDefaults()
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Group endpoints by module segment.
	modules := map[string][]*EndpointMeta{}
	for _, m := range r.Metas {
		if m == nil {
			continue
		}
		key := moduleKey(m.Path, opts.SkipPathSegments)
		modules[key] = append(modules[key], m)
	}

	moduleKeys := make([]string, 0, len(modules))
	for k := range modules {
		moduleKeys = append(moduleKeys, k)
	}
	sort.Strings(moduleKeys)

	baseTmpl, err := template.New("base").Funcs(template.FuncMap{"quote": strconv.Quote}).Parse(tsBaseTemplate)
	if err != nil {
		return fmt.Errorf("parse base template: %w", err)
	}
	moduleTmpl, err := template.New("module").Funcs(template.FuncMap{"quote": strconv.Quote}).Parse(tsModuleTemplate)
	if err != nil {
		return fmt.Errorf("parse module template: %w", err)
	}
	indexTmpl, err := template.New("index").Funcs(template.FuncMap{"quote": strconv.Quote}).Parse(tsIndexTemplate)
	if err != nil {
		return fmt.Errorf("parse index template: %w", err)
	}

	// base.ts
	if err := writeTemplate(filepath.Join(dir, "base.ts"), baseTmpl, tsModel{
		PackageName: opts.PackageName,
		ClientName:  opts.ClientName,
	}); err != nil {
		return err
	}

	type indexModule struct {
		Key       string
		File      string
		ClassName string
		PropName  string
	}
	var indexModules []indexModule

	// <module>.ts
	for _, key := range moduleKeys {
		metas := modules[key]
		types := collectTypes(metas)
		typeNames := assignTypeNames(types)
		orderedTypes := orderedByName(typeNames)

		typeDefs := make([]string, 0, len(orderedTypes))
		for _, t := range orderedTypes {
			name := typeNames[t]
			def, err := tsTypeDef(t, name, typeNames)
			if err != nil {
				return err
			}
			if def != "" {
				typeDefs = append(typeDefs, def)
			}
		}

		endpoints := make([]tsEndpointModel, 0, len(metas))
		for _, m := range metas {
			reqType := typeNames[deref(m.Req)]
			resType := typeNames[deref(m.Res)]
			hasBody := endpointHasBody(m.Method, m.Req)
			hasParams := endpointHasParams(m.Req)
			endpoints = append(endpoints, tsEndpointModel{
				Method:     strings.ToUpper(m.Method),
				Path:       m.Path,
				MethodName: endpointMethodName(m.Method, m.Path),
				ReqType:    reqType,
				ResType:    resType,
				Consumes:   firstOr(m.Consumes),
				Produces:   firstOr(m.Produces),
				HasBody:    hasBody,
				HasParams:  hasParams,
			})
		}
		sort.SliceStable(endpoints, func(i, j int) bool {
			if endpoints[i].Path == endpoints[j].Path {
				return endpoints[i].Method < endpoints[j].Method
			}
			return endpoints[i].Path < endpoints[j].Path
		})

		model := tsModel{
			PackageName: opts.PackageName,
			ClientName:  moduleClientClassName(key),
			Endpoints:   endpoints,
			TypeDefs:    typeDefs,
		}

		file := moduleFileName(key) + ".ts"
		if err := writeTemplate(filepath.Join(dir, file), moduleTmpl, model); err != nil {
			return err
		}

		indexModules = append(indexModules, indexModule{
			Key:       key,
			File:      strings.TrimSuffix(file, ".ts"),
			ClassName: moduleClientClassName(key),
			PropName:  modulePropName(key),
		})
	}

	// index.ts
	var buf bytes.Buffer
	if err := indexTmpl.Execute(&buf, map[string]any{
		"PackageName": opts.PackageName,
		"ClientName":  opts.ClientName,
		"Modules":     indexModules,
	}); err != nil {
		return fmt.Errorf("execute index template: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.ts"), buf.Bytes(), filePerm); err != nil {
		return fmt.Errorf("write index file: %w", err)
	}

	return nil
}

func firstOr(in []string) string {
	if len(in) == 0 || in[0] == "" {
		return "application/json"
	}
	return in[0]
}

func moduleKey(path string, skip int) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return rootModule
	}
	parts := strings.Split(path, "/")
	for len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if skip > 0 && skip < len(parts) {
		parts = parts[skip:]
	}
	if len(parts) == 0 || parts[0] == "" {
		return rootModule
	}
	return parts[0]
}

func moduleFileName(key string) string {
	if key == "" {
		return rootModule
	}
	return strings.ToLower(sanitizeIdent(key))
}

func moduleClientClassName(key string) string {
	if key == "" || key == rootModule {
		return "RootClient"
	}
	s := sanitizeIdent(key)
	if s == "" {
		return "RootClient"
	}
	return strings.ToUpper(s[:1]) + s[1:] + "Client"
}

func modulePropName(key string) string {
	if key == "" || key == rootModule {
		return rootModule
	}
	s := sanitizeIdent(key)
	if s == "" {
		return rootModule
	}
	return lowerFirst(s)
}

func writeTemplate(path string, tmpl *template.Template, model any) error {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, model); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), filePerm); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

func collectTypes(metas []*EndpointMeta) []reflect.Type {
	seen := map[reflect.Type]bool{}
	var out []reflect.Type
	var visit func(t reflect.Type)
	visit = func(t reflect.Type) {
		if t == nil {
			return
		}
		t = deref(t)
		if t.Kind() == reflect.Pointer {
			return
		}
		if seen[t] {
			return
		}
		seen[t] = true

		switch t.Kind() {
		case reflect.Struct:
			out = append(out, t)
			for i := range t.NumField() {
				f := t.Field(i)
				if !f.IsExported() {
					continue
				}
				visit(f.Type)
			}
		case reflect.Slice, reflect.Array:
			visit(t.Elem())
		case reflect.Map:
			visit(t.Elem())
		default:
			// do nothing for other types
		}
	}

	for _, m := range metas {
		if m == nil {
			continue
		}
		visit(m.Req)
		visit(m.Res)
	}
	return out
}

func assignTypeNames(types []reflect.Type) map[reflect.Type]string {
	out := map[reflect.Type]string{}
	used := map[string]int{}

	for _, t := range types {
		base := t.Name()
		if base == "" {
			base = "Anon"
		}
		base = sanitizeIdent(base)
		n := used[base]
		used[base] = n + 1
		if n == 0 {
			out[t] = base
		} else {
			out[t] = fmt.Sprintf("%s%d", base, n+1)
		}
	}

	if _, ok := out[reflect.TypeFor[struct{}]()]; !ok {
		out[reflect.TypeFor[struct{}]()] = "Empty"
	}
	return out
}

func orderedByName(typeNames map[reflect.Type]string) []reflect.Type {
	var types []reflect.Type
	for t := range typeNames {
		if t == nil {
			continue
		}
		if deref(t).Kind() != reflect.Struct {
			continue
		}
		types = append(types, t)
	}
	sort.SliceStable(types, func(i, j int) bool {
		return typeNames[types[i]] < typeNames[types[j]]
	})
	return types
}

func endpointMethodName(method, path string) string {
	name := strings.ToLower(method) + "_" + strings.Trim(path, "/")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, "{", "")
	name = strings.ReplaceAll(name, "}", "")
	name = sanitizeIdent(name)
	if name == "" {
		return "call"
	}
	return name
}

func sanitizeIdent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		ok := r == '_' || unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r))
		if !ok {
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if unicode.IsDigit(rune(out[0])) {
		return "_" + out
	}
	return out
}

func deref(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func tsTypeDef(t reflect.Type, name string, typeNames map[reflect.Type]string) (string, error) {
	t = deref(t)
	if t.Kind() != reflect.Struct {
		return "", nil
	}

	// special-case empty struct
	if t.NumField() == 0 {
		return fmt.Sprintf("export type %s = Record<string, never>", name), nil
	}

	var b strings.Builder
	b.WriteString("export interface ")
	b.WriteString(name)
	b.WriteString(" {\n")

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous && deref(f.Type).Kind() == reflect.Struct {
			// Skip embedding for now; generator can be improved to merge fields.
			continue
		}

		jsonName, omit, skip, err := requiredSnakeCaseJSONFieldName(t, f)
		if err != nil {
			return "", err
		}
		if skip {
			continue
		}

		tsType := tsTypeExpr(f.Type, typeNames)
		optional := omit
		b.WriteString("  ")
		b.WriteString(jsonName)
		if optional {
			b.WriteString("?")
		}
		b.WriteString(": ")
		b.WriteString(tsType)
		b.WriteString("\n")
	}

	b.WriteString("}")
	return b.String(), nil
}

func hasJSONBody(req reflect.Type) bool {
	req = deref(req)
	if req.Kind() == reflect.Struct && req.NumField() == 0 {
		return false
	}
	return true
}

func endpointHasBody(method string, req reflect.Type) bool {
	if strings.EqualFold(method, http.MethodGet) {
		return false
	}
	return hasJSONBody(deref(req))
}

func endpointHasParams(req reflect.Type) bool {
	req = deref(req)
	if req == nil {
		return false
	}
	if req.Kind() == reflect.Struct {
		return req.NumField() > 0
	}
	return true
}

func tsTypeExpr(t reflect.Type, typeNames map[reflect.Type]string) string {
	if t == nil {
		return unknownType
	}
	switch t.Kind() {
	case reflect.Pointer:
		return tsTypeExpr(t.Elem(), typeNames) + " | null"
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return tsTypeExpr(t.Elem(), typeNames) + "[]"
	case reflect.Map:
		return "Record<string, " + tsTypeExpr(t.Elem(), typeNames) + ">"
	case reflect.Struct:
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return "string"
		}
		if name, ok := typeNames[t]; ok {
			return name
		}
		// fallback for unnamed structs or missed registration
		return "Record<string, " + unknownType + ">"
	case reflect.Interface:
		return unknownType
	default:
		return unknownType
	}
}

func requiredSnakeCaseJSONFieldName(owner reflect.Type, f reflect.StructField) (name string, omitempty, skip bool, err error) {
	tag, ok := f.Tag.Lookup("json")
	if !ok {
		return "", false, false, fmt.Errorf("%s.%s: missing json tag (suggest %q)", owner.Name(), f.Name, toSnakeCase(f.Name))
	}
	if tag == "-" {
		return "", false, true, nil
	}
	if tag == "" || strings.HasPrefix(tag, ",") {
		return "", false, false, fmt.Errorf("%s.%s: json tag must specify an explicit snake_case name (suggest %q)", owner.Name(), f.Name, toSnakeCase(f.Name))
	}
	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		name = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omitempty = true
		}
	}
	if !isSnakeCase(name) {
		return "", false, false, fmt.Errorf("%s.%s: json tag %q must be snake_case (suggest %q)", owner.Name(), f.Name, name, toSnakeCase(name))
	}
	return name, omitempty, false, nil
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func isSnakeCase(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		case r == '_' && i > 0:
		default:
			return false
		}
	}
	return s[0] >= 'a' && s[0] <= 'z'
}

func toSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var out []rune
	var prev rune
	var wroteUnderscore bool

	rs := []rune(s)
	for i, r := range rs {
		if r == '_' || r == '-' || unicode.IsSpace(r) {
			if len(out) > 0 && !wroteUnderscore {
				out = append(out, '_')
				wroteUnderscore = true
			}
			prev = r
			continue
		}

		isUpper := unicode.IsUpper(r)
		isLower := unicode.IsLower(r)
		isDigit := unicode.IsDigit(r)

		if isUpper {
			nextLower := false
			if i+1 < len(rs) {
				nextLower = unicode.IsLower(rs[i+1])
			}
			if len(out) > 0 && !wroteUnderscore {
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || nextLower {
					out = append(out, '_')
				}
			}
			r = unicode.ToLower(r)
			wroteUnderscore = false
			out = append(out, r)
		} else if isLower || isDigit {
			wroteUnderscore = false
			out = append(out, unicode.ToLower(r))
		}

		prev = r
	}

	for len(out) > 0 && out[0] == '_' {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == '_' {
		out = out[:len(out)-1]
	}
	return string(out)
}
