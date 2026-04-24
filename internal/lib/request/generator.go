package request

import (
	"github.com/google/uuid"
)

// Maximum allowed length for client-provided request IDs.
// Prevents abuse via excessively large IDs that could bloat logs or responses.
const MaxClientRequestIdLength = 255

// A strong type representing a server-generated internal request ID.
// It is always a UUID v4 and should never be constructed from client input.
// Use NewRequestID() to generate a new one.
type RequestId string

// Returns the string representation of the RequestId.
func (r RequestId) String() string {
	return string(r)
}

// Generator defines the interface for generating unique request IDs
type Generator interface {
	NewRequestID() RequestId
}

// UUIDGenerator generates UUID v4 request IDs
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUID-based request ID generator
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// NewRequestID generates a new UUID v4 request ID
func (g *UUIDGenerator) NewRequestID() RequestId {
	return RequestId(uuid.New().String())
}

// DefaultGenerator is the default request ID generator
var DefaultGenerator = NewUUIDGenerator()

// NewRequestID generates a new request ID using the default generator
func NewRequestID() RequestId {
	return DefaultGenerator.NewRequestID()
}
