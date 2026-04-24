package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_WebSocketCloseCodeCounts(t *testing.T) {

	t.Run("no events", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		s := wsccc.String()

		assert.Equal(t, "{}", s)
	})

	t.Run("1 standard event", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		wsccc.Push(WebSocketCloseCode_1000_NormalClosure)

		s := wsccc.String()

		assert.Equal(t, "{ 1000:1 }", s)
	})

	t.Run("3 standard events", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		wsccc.Push(WebSocketCloseCode_1005_NoStatusReceived)
		wsccc.Push(WebSocketCloseCode_1000_NormalClosure)
		wsccc.Push(WebSocketCloseCode_1005_NoStatusReceived)

		s := wsccc.String()

		assert.Equal(t, "{ 1000:1, 1005:2 }", s)
	})

	t.Run("1 reserved event", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		wsccc.Push(WebSocketCloseCode(123))

		s := wsccc.String()

		assert.Equal(t, "{ reserved:1 }", s)
	})

	t.Run("1 reserved event and 12 standard events", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		wsccc.Push(WebSocketCloseCode_1008_PolicyViolation)
		wsccc.Push(WebSocketCloseCode_1003_UnsupportedData)
		wsccc.Push(WebSocketCloseCode_1003_UnsupportedData)
		wsccc.Push(WebSocketCloseCode_1000_NormalClosure)
		wsccc.Push(WebSocketCloseCode_1008_PolicyViolation)
		wsccc.Push(WebSocketCloseCode_1000_NormalClosure)
		wsccc.Push(WebSocketCloseCode_1015_TLSHandshake)
		wsccc.Push(WebSocketCloseCode_1012_ServiceRestart)
		wsccc.Push(WebSocketCloseCode(999))
		wsccc.Push(WebSocketCloseCode_1009_MessageTooBig)
		wsccc.Push(WebSocketCloseCode_1000_NormalClosure)
		wsccc.Push(WebSocketCloseCode_1006_AbnormalClosure)
		wsccc.Push(WebSocketCloseCode_1008_PolicyViolation)

		s := wsccc.String()

		assert.Equal(t, "{ 1000:3, 1003:2, 1006:1, 1008:3, 1009:1, 1012:1, 1015:1; reserved:1 }", s)
	})

	t.Run("1 custom event", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		wsccc.Push(WebSocketCloseCode(3_001))

		s := wsccc.String()

		assert.Equal(t, "{ 3001:1 }", s)
	})

	t.Run("big mix of values event", func(t *testing.T) {

		wsccc := WebSocketCloseCodeCounts{}

		wsccc.Push(WebSocketCloseCode(1_012))
		wsccc.Push(WebSocketCloseCode(10_000))
		wsccc.Push(WebSocketCloseCode(-1))
		wsccc.Push(WebSocketCloseCode(1_006))
		wsccc.Push(WebSocketCloseCode(3_001))
		wsccc.Push(WebSocketCloseCode(999))
		wsccc.Push(WebSocketCloseCode(3_001))
		wsccc.Push(WebSocketCloseCode(4_999))
		wsccc.Push(WebSocketCloseCode(1_006))

		s := wsccc.String()

		assert.Equal(t, "{ 1006:2, 1012:1, 3001:2, 4999:1; reserved:3 }", s)
	})
}
