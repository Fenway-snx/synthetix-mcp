package marshal

import (
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
)

var (
	byteSliceType        = reflect.TypeOf([]byte(nil))
	jsonMarshalerIfcType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerIfcType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

// Best-effort marshaling of any Go value as JSON,
//
// silently omitting struct fields and
// map entries whose types cannot be represented in JSON (channels,
// functions, complex numbers, unsafe pointers). Non-finite floats
// (NaN, ±Inf) are encoded as null. Cyclic pointer references are
// broken by encoding the cycle point as null.
//
// Struct tags ("json") are respected: field renaming, "-" for skip,
// and "omitempty" all work as expected. Types implementing
// json.Marshaler or encoding.TextMarshaler are delegated to their
// custom methods without decomposition.
//
// Intended for diagnostic/DLQ payloads where best-effort serialization
// of arbitrary values is more useful than a hard failure.
func SafeMarshalJSON(v any) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	} else {

		sanitized := sanitize(reflect.ValueOf(v), make(map[uintptr]bool))

		return json.Marshal(sanitized)
	}
}

func collectStructFields(v reflect.Value, t reflect.Type, seen map[uintptr]bool, out map[string]any) {
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fv := v.Field(i)
		tag := field.Tag.Get("json")

		name, opts, skip := parseJSONTag(tag, field.Name)
		if skip {
			continue
		}

		// Promote fields from embedded structs that lack an explicit JSON name
		if field.Anonymous && !hasExplicitName(tag) {
			embedded := fv
			for embedded.Kind() == reflect.Ptr {
				if embedded.IsNil() {
					break
				}
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct && !implementsMarshaler(embedded) {
				collectStructFields(embedded, embedded.Type(), seen, out)
				continue
			}
		}

		if isUnmarshalableKind(fv.Kind()) {
			continue
		}

		val := sanitize(fv, seen)

		if opts.omitempty && isEmpty(val) {
			continue
		}

		out[name] = val
	}
}

// Returns the value in a form that json.Marshal will dispatch to the
// custom MarshalJSON / MarshalText method. If the method is on the
// pointer receiver and the value is not addressable, we box it.
func extractMarshalerValue(v reflect.Value) any {
	t := v.Type()
	if t.Implements(jsonMarshalerIfcType) || t.Implements(textMarshalerIfcType) {
		return v.Interface()
	}
	if v.CanAddr() {
		return v.Addr().Interface()
	}
	ptr := reflect.New(t)
	ptr.Elem().Set(v)
	return ptr.Interface()
}

func implementsMarshaler(v reflect.Value) bool {
	t := v.Type()
	if t.Implements(jsonMarshalerIfcType) || t.Implements(textMarshalerIfcType) {
		return true
	}
	pt := reflect.PointerTo(t)
	return pt.Implements(jsonMarshalerIfcType) || pt.Implements(textMarshalerIfcType)
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case bool:
		return !val
	case int64:
		return val == 0
	case uint64:
		return val == 0
	case float64:
		return val == 0
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	case []byte:
		return len(val) == 0
	}
	return false
}

func isUnmarshalableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Chan, reflect.Func, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return true
	}
	return false
}

type tagOptions struct {
	omitempty bool
}

func parseJSONTag(tag string, fieldName string) (string, tagOptions, bool) {
	if tag == "-" {
		return "", tagOptions{}, true
	}

	name := fieldName
	var opts tagOptions

	if tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			name = parts[0]
		}
		for _, opt := range parts[1:] {
			if opt == "omitempty" {
				opts.omitempty = true
			}
		}
	}

	return name, opts, false
}

func hasExplicitName(tag string) bool {
	if tag == "" || tag == "-" {
		return false
	}
	return strings.Split(tag, ",")[0] != ""
}

func resolveKind(v reflect.Value) reflect.Kind {
	if v.Kind() == reflect.Interface && !v.IsNil() {
		return v.Elem().Kind()
	}
	return v.Kind()
}

func sanitize(v reflect.Value, seen map[uintptr]bool) any {
	if !v.IsValid() {
		return nil
	}

	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		if v.Kind() == reflect.Ptr {
			ptr := v.Pointer()
			if seen[ptr] {
				return nil
			}
			seen[ptr] = true
			defer delete(seen, ptr)
		}
		v = v.Elem()
	}

	if implementsMarshaler(v) {
		return extractMarshalerValue(v)
	}

	switch v.Kind() {
	case reflect.Struct:
		return sanitizeStruct(v, seen)

	case reflect.Map:
		return sanitizeMap(v, seen)

	case reflect.Slice:
		if v.Type() == byteSliceType {
			if v.IsNil() {
				return nil
			}
			return v.Bytes()
		}
		if v.IsNil() {
			return nil
		}
		return sanitizeSequence(v, seen)

	case reflect.Array:
		return sanitizeSequence(v, seen)

	case reflect.Bool:
		return v.Bool()

	case reflect.String:
		return v.String()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()

	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
		return f

	default:
		return nil
	}
}

func sanitizeMap(v reflect.Value, seen map[uintptr]bool) any {
	if v.IsNil() {
		return nil
	}
	out := make(map[string]any, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		mv := iter.Value()
		if isUnmarshalableKind(resolveKind(mv)) {
			continue
		}
		key := fmt.Sprint(iter.Key().Interface())
		out[key] = sanitize(mv, seen)
	}
	return out
}

func sanitizeSequence(v reflect.Value, seen map[uintptr]bool) any {
	n := v.Len()
	out := make([]any, 0, n)
	for i := range n {
		out = append(out, sanitize(v.Index(i), seen))
	}
	return out
}

func sanitizeStruct(v reflect.Value, seen map[uintptr]bool) any {
	out := make(map[string]any, v.NumField())
	collectStructFields(v, v.Type(), seen, out)
	return out
}
