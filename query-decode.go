package httprpc

import (
	"encoding"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func decodeQueryParams[Req any](r *http.Request) (Req, error) {
	var req Req

	values := r.URL.Query()
	if len(values) == 0 {
		return req, nil
	}

	rv := reflect.ValueOf(&req).Elem()
	rt := rv.Type()
	if rv.Kind() != reflect.Struct {
		return req, fmt.Errorf("decode query: request type %s must be a struct", rt.Kind())
	}

	for i := range rt.NumField() {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous && deref(field.Type).Kind() == reflect.Struct {
			// Skip embedded structs for now; keep behavior aligned with TS generator.
			continue
		}

		name, skip, err := queryFieldName(rt, field)
		if err != nil {
			return req, err
		}
		if skip {
			continue
		}

		vals, ok := values[name]
		if !ok {
			continue
		}

		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}
		if err := setFromStrings(fv, vals); err != nil {
			return req, fmt.Errorf("decode query %s: %w", name, err)
		}
	}

	return req, nil
}

func setFromStrings(v reflect.Value, vals []string) error {
	if !v.CanSet() {
		return fmt.Errorf("field is not settable")
	}
	if len(vals) == 0 {
		return nil
	}

	// TextUnmarshaler support.
	if v.CanAddr() {
		unmarhshaller, ok := v.Addr().Interface().(encoding.TextUnmarshaler)
		if !ok {
			return fmt.Errorf("type %s does not implement TextUnmarshaler", v.Type())
		}
		if err := unmarhshaller.UnmarshalText([]byte(vals[0])); err != nil {
			return fmt.Errorf("unmarshal text: %w", err)
		}
		return nil
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		return setFromStrings(v.Elem(), vals)
	case reflect.String:
		v.SetString(vals[0])
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(vals[0])
		if err != nil {
			return fmt.Errorf("parse bool: %w", err)
		}
		v.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(vals[0], 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse int: %w", err)
		}
		v.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err := strconv.ParseUint(vals[0], 10, v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse uint: %w", err)
		}
		v.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(vals[0], v.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse float: %w", err)
		}
		v.SetFloat(f)
		return nil
	case reflect.Slice:
		// Special-case []byte for convenience.
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes([]byte(vals[0]))
			return nil
		}
		s := reflect.MakeSlice(v.Type(), len(vals), len(vals))
		for i := range vals {
			if err := setFromStrings(s.Index(i), []string{vals[i]}); err != nil {
				return err
			}
		}
		v.Set(s)
		return nil
	case reflect.Struct:
		if v.Type().PkgPath() == "time" && v.Type().Name() == "Time" {
			tm, err := time.Parse(time.RFC3339, vals[0])
			if err != nil {
				return fmt.Errorf("parse time: %w", err)
			}
			v.Set(reflect.ValueOf(tm))
			return nil
		}
		return fmt.Errorf("unsupported struct type %s", v.Type())
	default:
		return fmt.Errorf("unsupported kind %s", v.Kind())
	}
}

func queryFieldName(owner reflect.Type, f reflect.StructField) (name string, skip bool, err error) {
	if v, found, shouldSkip := tagName(f, "query"); found {
		if shouldSkip {
			return "", true, nil
		}
		if v != "" {
			return v, false, nil
		}
	}

	if v, found, shouldSkip := tagName(f, "json"); found {
		if shouldSkip {
			return "", true, nil
		}
		if v != "" {
			return v, false, nil
		}
		// If json tag is present but empty, fall back to snake-case to mirror encoding/json default.
	}

	fallback := toSnakeCase(f.Name)
	if fallback == "" {
		fallback = f.Name
	}
	if fallback == "" {
		return "", false, fmt.Errorf("%s.%s: empty field name", owner.Name(), f.Name)
	}
	return fallback, false, nil
}

func tagName(f reflect.StructField, key string) (name string, found, skip bool) {
	tag, ok := f.Tag.Lookup(key)
	if !ok {
		return "", false, false
	}
	if tag == "-" {
		return "", true, true
	}
	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		name = parts[0]
	}
	return name, true, false
}
