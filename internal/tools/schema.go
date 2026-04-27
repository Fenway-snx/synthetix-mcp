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
// fields so they are advertised as digit strings. The runtime FlexInt64
// decoder still accepts JSON numbers where those are sent, but publishing
// a single string shape avoids confusing MCP clients that reject oneOf-heavy
// tool schemas while preserving precision for JavaScript callers.
func schemaForInput[T any]() (*jsonschema.Schema, error) {
	return buildSchema[T](rewriteBigIntFieldsForInput)
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
	stripPermissiveAdditionalProperties(schema)
	return schema, nil
}

// rewriteBigIntFieldsForInput rewrites properties in bigIntFieldNames to
// digit strings. Keep this intentionally simpler than oneOf(integer|string):
// Claude Code has been observed to connect to resources while dropping all
// tools when the input schemas contain oneOf for common ID fields.
func rewriteBigIntFieldsForInput(s *jsonschema.Schema) {
	walkBigIntFields(s, convertToBigIntString)
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

func stripPermissiveAdditionalProperties(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	if s.AdditionalProperties != nil && reflect.DeepEqual(*s.AdditionalProperties, jsonschema.Schema{}) {
		s.AdditionalProperties = nil
	}
	for _, prop := range s.Properties {
		stripPermissiveAdditionalProperties(prop)
	}
	if s.Items != nil {
		stripPermissiveAdditionalProperties(s.Items)
	}
	for _, sub := range s.OneOf {
		stripPermissiveAdditionalProperties(sub)
	}
	for _, sub := range s.AnyOf {
		stripPermissiveAdditionalProperties(sub)
	}
	for _, sub := range s.AllOf {
		stripPermissiveAdditionalProperties(sub)
	}
	for _, def := range s.Defs {
		stripPermissiveAdditionalProperties(def)
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
