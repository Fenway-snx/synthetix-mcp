package utils

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	maxRecursionDepth = 25

	maxSafeInteger = 9_007_199_254_740_991
)

// StringifyAllBigInts transforms big integers to strings in key-value
// pairs. Returns a new slice to avoid mutating the caller's data.
func StringifyAllBigInts(keyVals []any, recursionDepth int) []any {
	result := make([]any, len(keyVals))
	copy(result, keyVals)
	for i := 1; i < len(result); i += 2 {
		result[i] = stringifyBigInts(result[i], recursionDepth+1)
	}
	return result
}

// stringifyBigInts recursively converts uint64 values > maxSafeInteger
// to strings. This ensures Datadog can properly ingest large integers
// that would otherwise lose precision in JavaScript's number handling.
func stringifyBigInts(v any, recursionDepth int) any {
	if v == nil {
		return nil
	}

	if recursionDepth > maxRecursionDepth {
		return v
	}

	if _, ok := v.(fmt.Stringer); ok {
		return fmt.Sprintf("%s", v)
	}

	if _, ok := v.(error); ok {
		return fmt.Sprintf("%v", v)
	}

	if _, ok := v.([]byte); ok {
		return fmt.Sprintf("%s", v)
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Struct:
		return stringifyStructBigInts(rv, recursionDepth+1)
	case reflect.Slice:
		return stringifySliceBigInts(rv, recursionDepth+1)
	case reflect.Map:
		return stringifyMapBigInts(rv, recursionDepth+1)
	case reflect.Int, reflect.Int64:
		val := rv.Int()
		if val > int64(maxSafeInteger) || val < -int64(maxSafeInteger) {
			return fmt.Sprintf("%d", val)
		}
		return v
	case reflect.Uint, reflect.Uint64:
		val := rv.Uint()
		if val > maxSafeInteger {
			return fmt.Sprintf("%d", val)
		}
		return v
	}

	return v
}

func stringifyStructBigInts(rv reflect.Value, recursionDepth int) map[string]any {
	rt := rv.Type()
	result := make(map[string]any, rv.NumField())

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Tag.Get("json")
		if name == "-" {
			continue
		}
		if name != "" {
			name, _, _ = strings.Cut(name, ",")
		}
		if name == "" {
			name = field.Name
		}

		result[name] = stringifyBigInts(rv.Field(i).Interface(), recursionDepth+1)
	}
	return result
}

func stringifySliceBigInts(rv reflect.Value, recursionDepth int) []any {
	result := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result[i] = stringifyBigInts(rv.Index(i).Interface(), recursionDepth+1)
	}
	return result
}

func stringifyMapBigInts(rv reflect.Value, recursionDepth int) map[string]any {
	result := make(map[string]any, rv.Len())
	for _, key := range rv.MapKeys() {
		result[fmt.Sprintf("%v", key.Interface())] = stringifyBigInts(rv.MapIndex(key).Interface(), recursionDepth+1)
	}
	return result
}
