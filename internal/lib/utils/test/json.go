package test

import (
	"encoding/json"
)

// Helper function to marshal JSON and panic on error.
func MustMarshalJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
