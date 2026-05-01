package utils

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errLogTagsContainsEmptySegmentCheckForTrailingOrDoubleCommas = errors.New("log_tags contains an empty segment (check for trailing or double commas)")
	errLogTagsMustNotBeEmptyAtLeastOneKeyValuePairRequired       = errors.New("log_tags must not be empty: at least one key:value pair is required")
)

// ValidateLogTags checks that raw is a non-empty, well-formed
// comma-separated list of "key:value" pairs. Returns an error describing the
// first violation found.
func ValidateLogTags(raw string) error {
	if raw == "" {
		return errLogTagsMustNotBeEmptyAtLeastOneKeyValuePairRequired
	}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			return errLogTagsContainsEmptySegmentCheckForTrailingOrDoubleCommas
		}
		idx := strings.IndexByte(pair, ':')
		if idx < 0 {
			return fmt.Errorf("log_tags: %q is missing ':' separator", pair)
		}
		if idx == 0 {
			return fmt.Errorf("log_tags: %q has an empty key", pair)
		}
	}
	return nil
}

// Converts validated comma-separated log tags into key/value pairs.
func ParseLogTags(raw string) []any {
	if raw == "" {
		return nil
	}
	var kvs []any
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		idx := strings.IndexByte(pair, ':')
		kvs = append(kvs, pair[:idx], pair[idx+1:])
	}
	return kvs
}
