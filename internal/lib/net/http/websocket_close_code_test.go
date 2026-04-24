package http

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func Test_WebSocketCloseCode_WELL_KNOWN_STANDARD_VALUES(t *testing.T) {

	tests := []struct {
		WebSocketCloseCode WebSocketCloseCode
		int_value          int
		gows_value         int
		s                  string
	}{
		{
			WebSocketCloseCode_1000_NormalClosure,
			1000,
			websocket.CloseNormalClosure,
			"Connection fulfilled its purpose",
		},
		{
			WebSocketCloseCode_1001_GoingAway,
			1001,
			websocket.CloseGoingAway,
			"Server/browser is shutting down",
		},
		{
			WebSocketCloseCode_1002_ProtocolError,
			1002,
			websocket.CloseProtocolError,
			"Protocol violation detected",
		},
		{
			WebSocketCloseCode_1003_UnsupportedData,
			1003,
			websocket.CloseUnsupportedData,
			"Received incompatible data type",
		},
		{
			WebSocketCloseCode_1004_Reserved,
			1004,
			-1,
			"Reserved for future use",
		},
		{
			WebSocketCloseCode_1005_NoStatusReceived,
			1005,
			websocket.CloseNoStatusReceived,
			"Expected status code was absent",
		},
		{
			WebSocketCloseCode_1006_AbnormalClosure,
			1006,
			websocket.CloseAbnormalClosure,
			"Connection lost unexpectedly",
		},
		{
			WebSocketCloseCode_1007_InvalidPayloadData,
			1007,
			websocket.CloseInvalidFramePayloadData,
			"Message data doesn’t match type",
		},
		{
			WebSocketCloseCode_1008_PolicyViolation,
			1008,
			websocket.ClosePolicyViolation,
			"Generic policy violation",
		},
		{
			WebSocketCloseCode_1009_MessageTooBig,
			1009,
			websocket.CloseMessageTooBig,
			"Message exceeds size limits",
		},
		{
			WebSocketCloseCode_1010_MandatoryExtension,
			1010,
			websocket.CloseMandatoryExtension,
			"Required extension not supported",
		},
		{
			WebSocketCloseCode_1011_InternalError,
			1011,
			websocket.CloseInternalServerErr,
			"Server encountered unexpected error",
		},
		{
			WebSocketCloseCode_1012_ServiceRestart,
			1012,
			websocket.CloseServiceRestart,
			"Server is restarting",
		},
		{
			WebSocketCloseCode_1013_TryAgainLater,
			1013,
			websocket.CloseTryAgainLater,
			"Temporary server overload",
		},
		{
			WebSocketCloseCode_1014_BadGateway,
			1014,
			-1,
			"Gateway/proxy received invalid response",
		},
		{
			WebSocketCloseCode_1015_TLSHandshake,
			1015,
			websocket.CloseTLSHandshake,
			"TLS/SSL handshake failure",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, int(tt.WebSocketCloseCode), tt.int_value)

		if -1 != tt.gows_value {
			assert.Equal(t, int(tt.WebSocketCloseCode), tt.gows_value)
		}

		s, known := tt.WebSocketCloseCode.HasDescription()
		assert.True(t, known)
		assert.Equal(t, tt.s, s)
	}
}
