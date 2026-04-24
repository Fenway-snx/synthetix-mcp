package test

// Returns a pointer to a copy of v. Intended for tests that need `*T`
// fields where taking the address of a composite literal or cast is
// awkward or illegal (e.g. inside struct literals).
func MakePointerOf[T any](v T) *T {
	return &v
}
