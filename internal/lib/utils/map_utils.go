package utils

import (
	"fmt"
	"strings"
)

// TODO: generalise (through generics and maybe a passed function), this
// towards usefulness as a general-purpose tool, and stop using in
// LookupTypedStringInParams()

// GetStringFromMap extracts a string value from a map with proper validation
func GetStringFromMap(m map[string]any, key string) (
	string, // s
	any, // provided
	error, // err
) {
	value, exists := m[key]
	if !exists {
		return "", nil, fmt.Errorf("missing %s", key)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", value, fmt.Errorf("invalid %s: expected string", key)
	}

	strValue = strings.TrimSpace(strValue)

	if strValue == "" {
		return "", value, fmt.Errorf("empty %s", key)
	}

	return strValue, value, nil
}

// MapKeys returns a slice of the keys in a map
func MapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// MapValues returns a slice of the values in a map
func MapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}
