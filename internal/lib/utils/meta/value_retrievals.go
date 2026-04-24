package meta

import "fmt"

// Attempts to retrieve the bool value from v if it is of bool type.
func TryGetBoolFromAnyOrFalse(val any) bool {

	switch v := val.(type) {
	case bool:

		return v
	default:

		return false
	}
}

// Attempts to retrieve the string value from v if it is of string type, or
// implements `fmt.Stringer`.
func TryGetStringFromAnyOrEmpty(val any) string {

	switch v := val.(type) {
	case string:

		return v
	case fmt.Stringer:

		return v.String()
	default:

		return ""
	}
}
