package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Defines options that moderate the behaviour of client-side string
// validation.
type ClientSideOption int

const ClientSideOption_None ClientSideOption = 0
const (
	ClientSideOption_RejectEmpty ClientSideOption = 1 << iota // Causes empty strings to be rejected
	ClientSideOption_Trim                                     // Causes input strings to be trimmed of leading and trailing space prior to validation
)

var (
	validClientSideOptions = []ClientSideOption{
		ClientSideOption_RejectEmpty,
		ClientSideOption_Trim,
	}
	allBits int
)

func init() {
	for _, opt := range validClientSideOptions {
		allBits |= int(opt)
	}
}

func (cso ClientSideOption) String() string {
	switch cso {
	case ClientSideOption_None:
		return "None"
	case ClientSideOption_RejectEmpty:
		return "RejectEmpty"
	case ClientSideOption_Trim:
		return "Trim"
	default:
		int_cso := int(cso)
		if (int_cso & ^allBits) != 0 {
			return fmt.Sprintf("0x%x", int_cso)
		} else {
			var builder strings.Builder
			for _, opt := range validClientSideOptions {
				int_opt := int(opt)

				if (int_cso & int_opt) != 0 {
					if builder.Len() != 0 {
						builder.WriteByte('|')
					}

					builder.WriteString(opt.String()) // 1-level recursion
				}
			}

			return builder.String()
		}
	}
}

const (
	clientStringRegExp = `^\s*[-.=/+_A-Za-z0-9]+\s*$`
)

var (
	errEmptyInput              = errors.New("empty input")
	errInvalidClientSideString = errors.New("invalid string")
)

var (
	re *regexp.Regexp
)

func init() {
	re = regexp.MustCompile(clientStringRegExp)
}

func _hasOption(bits int, option ClientSideOption) bool {
	option_int := int(option)

	if option_int == 0 {
		return bits == 0
	} else {
		return (bits & option_int) == option_int
	}
}

// Validates a client-side string or string-like object.
//
// Parameters:
//   - input - The input string;
//   - maxLen - The maximum length. No maximum is enforced if <1;
//   - options - 0 or more options that moderate the behaviur of the
//     function;
func ValidateClientSideString[T ~string](
	input T,
	maxLen int,
	options ...ClientSideOption,
) (validatedForm T, err error) {

	var optionBits int
	for _, option := range options {

		optionBits |= int(option)
	}

	s := string(input)

	if _hasOption(optionBits, ClientSideOption_Trim) {
		s = strings.TrimSpace(s)
	}

	if s == "" {
		if _hasOption(optionBits, ClientSideOption_RejectEmpty) {
			err = errEmptyInput
		} else {
			validatedForm = T(s)
		}
	} else {

		if maxLen > 0 && len(s) > maxLen {
			err = fmt.Errorf("input too long (max %d characters)", maxLen)

			return
		}

		if !re.MatchString(s) {
			err = errInvalidClientSideString

			return
		}

		validatedForm = T(s)
	}

	return
}
