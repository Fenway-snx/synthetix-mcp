package constants

import "errors"

// API Service error constants

// Validation errors
var (
	ErrSymbolsNameEmpty = errors.New(`"symbols" must be a non empty string`)
	ErrSymbolNameEmpty  = errors.New(`"symbol" must be a non empty string`)
)
