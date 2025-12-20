package httprpc

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

type metaTag struct {
	name      string
	omitempty bool
	found     bool
	skip      bool
}

func decodeRequestMeta[Meta any](r *http.Request) (Meta, error) {
	var meta Meta
	mv := reflect.ValueOf(&meta).Elem()
	mt := mv.Type()
	if mv.Kind() != reflect.Struct {
		return meta, fmt.Errorf("decode meta: request meta type %s must be a struct", mt.Kind())
	}

	pathParams, _ := r.Context().Value(pathParamsKey{}).(map[string]string)

	for i := range mt.NumField() {
		field := mt.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			continue
		}

		pathTag, err := parseMetaTag(mt, field, "path", true)
		if err != nil {
			return meta, err
		}
		headerTag, err := parseMetaTag(mt, field, "header", false)
		if err != nil {
			return meta, err
		}
		if pathTag.found && headerTag.found {
			return meta, fmt.Errorf("%s.%s: cannot use both path and header tags", metaOwnerName(mt), field.Name)
		}
		if pathTag.found {
			if pathTag.skip {
				continue
			}
			val, ok := pathParams[pathTag.name]
			if !ok {
				return meta, fmt.Errorf("missing path param %q", pathTag.name)
			}
			fv := mv.Field(i)
			if !fv.CanSet() {
				continue
			}
			if err := setFromStrings(fv, []string{val}); err != nil {
				return meta, fmt.Errorf("decode path %s: %w", pathTag.name, err)
			}
			continue
		}
		if headerTag.found {
			if headerTag.skip {
				continue
			}
			vals := r.Header.Values(headerTag.name)
			if len(vals) == 0 {
				if headerTag.omitempty {
					continue
				}
				return meta, fmt.Errorf("missing header %q", headerTag.name)
			}
			fv := mv.Field(i)
			if !fv.CanSet() {
				continue
			}
			if err := setFromStrings(fv, vals); err != nil {
				return meta, fmt.Errorf("decode header %s: %w", headerTag.name, err)
			}
		}
	}

	return meta, nil
}

func validateMetaType(meta reflect.Type, path string) error {
	if meta == nil {
		return nil
	}
	meta = deref(meta)
	if meta.Kind() != reflect.Struct {
		return fmt.Errorf("request meta type %s must be a struct", meta.Kind())
	}

	pattern, err := parseRoutePattern(path)
	if err != nil {
		return err
	}
	pathParams := map[string]struct{}{}
	for _, name := range pattern.params {
		pathParams[name] = struct{}{}
	}

	seenPath := map[string]struct{}{}
	seenHeader := map[string]struct{}{}

	for i := range meta.NumField() {
		field := meta.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			continue
		}

		pathTag, err := parseMetaTag(meta, field, "path", true)
		if err != nil {
			return err
		}
		headerTag, err := parseMetaTag(meta, field, "header", false)
		if err != nil {
			return err
		}
		if pathTag.found && headerTag.found {
			return fmt.Errorf("%s.%s: cannot use both path and header tags", metaOwnerName(meta), field.Name)
		}
		if pathTag.found {
			if pathTag.skip {
				continue
			}
			if _, ok := pathParams[pathTag.name]; !ok {
				return fmt.Errorf("path tag %q does not match route %s", pathTag.name, path)
			}
			if _, ok := seenPath[pathTag.name]; ok {
				return fmt.Errorf("path tag %q is used more than once", pathTag.name)
			}
			seenPath[pathTag.name] = struct{}{}
		}
		if headerTag.found {
			if headerTag.skip {
				continue
			}
			if _, ok := seenHeader[headerTag.name]; ok {
				return fmt.Errorf("header tag %q is used more than once", headerTag.name)
			}
			seenHeader[headerTag.name] = struct{}{}
		}
	}

	return nil
}

func parseMetaTag(owner reflect.Type, f reflect.StructField, key string, requireSnakeCase bool) (metaTag, error) {
	tag, ok := f.Tag.Lookup(key)
	if !ok {
		return metaTag{}, nil
	}
	if tag == "-" {
		return metaTag{found: true, skip: true}, nil
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return metaTag{}, fmt.Errorf("%s.%s: %s tag must specify a name", metaOwnerName(owner), f.Name, key)
	}
	omit := false
	for _, p := range parts[1:] {
		if p == "omitempty" {
			omit = true
		}
	}
	if requireSnakeCase && !isSnakeCase(name) {
		return metaTag{}, fmt.Errorf("%s.%s: %s tag %q must be snake_case", metaOwnerName(owner), f.Name, key, name)
	}
	return metaTag{name: name, omitempty: omit, found: true}, nil
}

func metaOwnerName(owner reflect.Type) string {
	if owner == nil {
		return ""
	}
	if owner.Name() != "" {
		return owner.Name()
	}
	if owner.String() != "" {
		return owner.String()
	}
	return "meta"
}
