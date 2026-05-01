package tools

import (
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
)

// JSON property names whose values are 64-bit identifiers.
// String encoding preserves precision for JavaScript clients.
// Schema rewriting advertises that wire shape instead of integers.
var bigIntFieldNames = map[string]struct{}{
	"id":              {},
	"masterAccountId": {},
	"positionId":      {},
	"subAccountId":    {},
	"venueId":         {},
}

const bigIntPattern = `^-?[0-9]+$`

// Generates input schema and advertises ID fields as digit strings.
// Runtime decoding still accepts JSON numbers where supported.
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

// Rewrites configured ID fields to digit strings.
// This avoids oneOf-heavy schemas that some MCP clients reject.
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
