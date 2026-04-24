package tools

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// bigIntFieldNames enumerates JSON property names whose runtime values are
// 64-bit Synthetix identifiers (subaccount IDs, order IDs, position IDs,
// trade IDs, etc.) that exceed JavaScript's Number.MAX_SAFE_INTEGER (2^53).
//
// These values are emitted and consumed as JSON strings via the ",string"
// encoding/json tag option so downstream clients that parse JSON into
// IEEE-754 doubles (all JSON parsers in JavaScript and many in other
// languages) do not round the low-order digits. Preserving the exact
// identifier is required to look up orders, positions and subaccounts
// on the upstream services.
//
// The auto-generated JSON schema reflects Go int64/uint64 fields as
// "type": "integer", which breaks MCP output validation once we emit them
// as strings. rewriteBigIntFields() walks the schema and switches these
// properties to "type": "string" with a digits-only pattern so the
// published schema matches what tools actually return.
var bigIntFieldNames = map[string]struct{}{
	"id":              {},
	"masterAccountId": {},
	"positionId":      {},
	"subAccountId":    {},
	"venueId":         {},
}

const bigIntPattern = `^-?[0-9]+$`

// schemaForInput generates the JSON schema for T and post-processes ID
// fields so they accept either a JSON integer (legacy/simple clients, and
// small values that fit in a double) or a JSON string of digits
// (precision-preserving path for JS/BigInt clients). The Go input type
// keeps its int64 field and Go's json decoder will accept the integer
// form; string-form callers should send values that fit in int64 so the
// standard decoder continues to succeed.
func schemaForInput[T any]() (*jsonschema.Schema, error) {
	return buildSchema[T](rewriteBigIntFieldsForInput)
}

// schemaForOutput generates the JSON schema for T and post-processes ID
// fields so they are typed as JSON strings. Go output structs marshal
// these fields via the ",string" encoding/json tag to preserve precision
// when JSON numbers would otherwise lose low-order digits on the wire.
func schemaForOutput[T any]() (*jsonschema.Schema, error) {
	return buildSchema[T](rewriteBigIntFieldsForOutput)
}

func buildSchema[T any](rewrite func(*jsonschema.Schema)) (*jsonschema.Schema, error) {
	rt := reflect.TypeFor[T]()
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	schema, err := jsonschema.ForType(rt, &jsonschema.ForOptions{})
	if err != nil {
		return nil, fmt.Errorf("reflect schema for %s: %w", rt, err)
	}
	rewrite(schema)
	return schema, nil
}

// rewriteBigIntFieldsForOutput rewrites properties in bigIntFieldNames to
// digit-strings (no alternative integer form). The MCP validator enforces
// this exactly, matching the ",string"-tagged marshal output.
func rewriteBigIntFieldsForOutput(s *jsonschema.Schema) {
	walkBigIntFields(s, convertToBigIntString)
}

// rewriteBigIntFieldsForInput rewrites properties in bigIntFieldNames to
// accept either a JSON integer or a digit-string, so existing clients that
// pass these values as numbers (e.g. small subaccount IDs in tests and
// scripts) keep working, while precision-sensitive clients can send the
// same identifier as a string to avoid IEEE-754 rounding.
func rewriteBigIntFieldsForInput(s *jsonschema.Schema) {
	walkBigIntFields(s, convertToBigIntOneOf)
}

func walkBigIntFields(s *jsonschema.Schema, convert func(*jsonschema.Schema)) {
	if s == nil {
		return
	}
	for name, prop := range s.Properties {
		if _, match := bigIntFieldNames[name]; match && isIntegerSchema(prop) {
			convert(prop)
		}
		walkBigIntFields(prop, convert)
	}
	if s.Items != nil {
		walkBigIntFields(s.Items, convert)
	}
	if s.AdditionalProperties != nil {
		walkBigIntFields(s.AdditionalProperties, convert)
	}
	for _, sub := range s.OneOf {
		walkBigIntFields(sub, convert)
	}
	for _, sub := range s.AnyOf {
		walkBigIntFields(sub, convert)
	}
	for _, sub := range s.AllOf {
		walkBigIntFields(sub, convert)
	}
	for _, def := range s.Defs {
		walkBigIntFields(def, convert)
	}
}

func isIntegerSchema(s *jsonschema.Schema) bool {
	if s == nil {
		return false
	}
	if s.Type == "integer" {
		return true
	}
	for _, t := range s.Types {
		if t == "integer" {
			return true
		}
	}
	return false
}

func convertToBigIntString(s *jsonschema.Schema) {
	s.Type = "string"
	s.Types = nil
	s.Minimum = nil
	s.Maximum = nil
	s.ExclusiveMinimum = nil
	s.ExclusiveMaximum = nil
	s.MultipleOf = nil
	s.Pattern = bigIntPattern
}

// convertToBigIntOneOf rewrites the schema in place so the property
// accepts either a JSON integer or a digit-string. Because we clear the
// top-level "type" and populate OneOf, the published schema advertises
// both forms to clients and to downstream validators.
func convertToBigIntOneOf(s *jsonschema.Schema) {
	description := s.Description
	*s = jsonschema.Schema{
		Description: description,
		OneOf: []*jsonschema.Schema{
			{Type: "integer"},
			{Type: "string", Pattern: bigIntPattern},
		},
	}
}
