package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Asserts that f panics with a value whose fmt.Sprint representation
// contains substr. This is useful when the panic value carries a
// file+line+function prefix that should not be matched literally.
func AssertPanicsContaining(t *testing.T, substr string, f func()) {
	t.Helper()

	var recovered any

	func() {
		defer func() { recovered = recover() }()
		f()
	}()

	require.NotNil(t, recovered, "expected a panic but none occurred")
	assert.Contains(t, fmt.Sprint(recovered), substr)
}
