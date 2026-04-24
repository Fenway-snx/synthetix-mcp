package cors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsOriginAllowed(t *testing.T) {

	t.Run("empty", func(t *testing.T) {

		r := IsOriginAllowed("")

		assert.False(t, r)
	})

	t.Run("allowed (and should be)", func(t *testing.T) {

		inputs := []string{
			"http://localhost",
			"http://localhost:80",
			"https://localhost",
			"https://localhost:443",
			"http://127.0.0.1",
			"http://127.0.0.1:80",
			"https://127.0.0.1",
			"https://127.0.0.1:443",
			"https://snxdev.io",
			"https://abc.snxdev.io",
			"https://synthetix.io",
			"https://abc.synthetix.io",
		}

		for i, input := range inputs {
			isOriginAllowed := IsOriginAllowed(input)

			assert.True(t, isOriginAllowed, "#%d value '%s' was expected to be allowed", i, input)
		}
	})

	t.Run("not allowed (and should not be)", func(t *testing.T) {

		inputs := []string{
			"http",
			"http://",
			"https",
			"https://",
			"http://someserver",
			"http://someserver/abc",
			"http://someserver:8000",
			"http://someserver:8000/abc",
			"http://128.0.0.1",
			"http://128.0.0.1:80",
			"http://128.0.0.1:8000",
			"https://128.0.0.1",
			"https://128.0.0.1:80",
			"https://128.0.0.1:8000",
			"httpx://localhost",
			"httpx://localhost:80",
			"httpx://localhost:8000",
			"httpsx://localhost",
			"httpsx://localhost:80",
			"httpsx://localhost:8000",
			"httpx://127.0.0.1",
			"httpx://127.0.0.1:80",
			"httpx://127.0.0.1:8000",
			"httpsx://127.0.0.1",
			"httpsx://127.0.0.1:80",
			"httpsx://127.0.0.1:8000",
			"http://snxdev.io",
			"http://abc.snxdev.io",
			"http://synthetix.io",
			"http://abc.synthetix.io",
			"http://snxdev.io:443",
			"http://abc.snxdev.io:443",
			"http://synthetix.io:443",
			"http://abc.synthetix.io:443",
		}

		for i, input := range inputs {
			isOriginAllowed := IsOriginAllowed(input)

			assert.False(t, isOriginAllowed, "#%d value '%s' was expected to be disallowed", i, input)
		}
	})

	t.Run("allowed (but should not be)", func(t *testing.T) {

		inputs := []string{
			"http://localhost:8000",
			"https://localhost:8000",
			"http://127.0.0.1:8000",
			"https://127.0.0.1:8000",

			// "http://localhost/abc", // This should be valid ...
			"http://localhost:80/abc",   // ... if this is valid or ...
			"http://localhost:8000/abc", // ... if this is valid.

			"https://someserver:8000/abc.synthetix.io",
		}

		for i, input := range inputs {
			isOriginAllowed := IsOriginAllowed(input)

			assert.True(t, isOriginAllowed, "#%d value '%s' was expected to be allowed", i, input)
		}
	})

	t.Run("not allowed (but should be)", func(t *testing.T) {

		inputs := []string{
			"https://snxdev.io:443",
			"https://abc.snxdev.io:443",
			"https://synthetix.io:443",
			"https://abc.synthetix.io:443",
		}

		for i, input := range inputs {
			isOriginAllowed := IsOriginAllowed(input)

			assert.False(t, isOriginAllowed, "#%d value '%s' was expected to be disallowed", i, input)
		}
	})
}
