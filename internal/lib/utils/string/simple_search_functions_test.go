package string

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func Benchmark_ByteIsASCIILeadingWordChar(b *testing.B) {

	b.ResetTimer()
	for i := 0; i != b.N; i++ {
		for j := 0; j != 256; j++ {
			c := byte(i)

			_ = ByteIsASCIILeadingWordChar(c)
		}
	}
}

func Benchmark_ByteIsASCIIWordChar(b *testing.B) {

	b.ResetTimer()
	for i := 0; i != b.N; i++ {
		for j := 0; j != 256; j++ {
			c := byte(i)

			_ = ByteIsASCIIWordChar(c)
		}
	}
}

func Test_ByteIsASCIILeadingWordChar(t *testing.T) {

	for i := 0; i != 256; i++ {
		c := byte(i)
		r := rune(c)

		b := ByteIsASCIILeadingWordChar(c)

		if '_' == c {
			assert.True(t, b, "underscore is a leading word character")
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			assert.True(t, b, "letter '%c' is a leading word character", rune(c))
		} else if unicode.IsDigit(rune(c)) {
			assert.False(t, b, "digit '%c' is not a leading word character", rune(c))
		} else {
			assert.False(t, b, "any other value is not a leading word character")
		}
	}
}

func Test_ByteIsASCIIWordChar(t *testing.T) {

	for i := 0; i != 256; i++ {
		c := byte(i)
		r := rune(c)

		b := ByteIsASCIIWordChar(c)

		if '_' == c {
			assert.True(t, b, "underscore is a word character")
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			assert.True(t, b, "letter '%c' is a word character", rune(c))
		} else if unicode.IsDigit(rune(c)) {
			assert.True(t, b, "digit '%c' is a word character", rune(c))
		} else {
			assert.False(t, b, "any other value is not a word character")
		}
	}
}
