package config

import (
	"strings"
)

const (
	pairSeparator     string = ","
	keyValueSeparator string = "="
)

func ParseUniqueStrings(
	input string,
	separator string,
) (
	r []string,
	err error,
) {

	r = make([]string, 0, len(input)/10)

	m := make(map[string]bool, len(input)/10)

	splits := strings.Split(input, separator)

	for _, s := range splits {
		s = strings.TrimSpace(s)

		if s != "" {

			if _, exists := m[s]; exists {

			} else {

				r = append(r, s)
				m[s] = true
			}
		}
	}

	return
}
