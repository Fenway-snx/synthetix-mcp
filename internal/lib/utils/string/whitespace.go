package string

func HasLeadingOrTrailingWhitespace[T ~string](s T) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] <= ' ' || s[len(s)-1] <= ' '
}
