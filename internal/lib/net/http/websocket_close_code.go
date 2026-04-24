package http

/*
References:
	https://websocket.org/reference/close-codes/
*/

// Strong type that represents a WebSocket connection Close Code.
type WebSocketCloseCode int

/*
1000|Normal Closure|Connection fulfilled its purpose|Standard graceful shutdown
1001|Going Away|Server/browser is shutting down|Server maintenance, page navigation
1002|Protocol Error|Protocol violation detected|Invalid frame, bad data format
1003|Unsupported Data|Received incompatible data type|Wrong message format received
1004|Reserved|Reserved for future use|Do not use
1005|No Status Received|Expected status code was absent|Internal use only (cannot send)
1006|Abnormal Closure|Connection lost unexpectedly|Internal use only (cannot send)
1007|Invalid Payload Data|Message data doesn’t match type|Malformed UTF-8, invalid JSON
1008|Policy Violation|Generic policy violation|Rate limiting, message size exceeded
1009|Message Too Big|Message exceeds size limits|Payload too large
1010|Mandatory Extension|Required extension not supported|Extension negotiation failed
1011|Internal Error|Server encountered unexpected error|Server-side exception
1012|Service Restart|Server is restarting|Planned restart
1013|Try Again Later|Temporary server overload|Server temporarily unavailable
1014|Bad Gateway|Gateway/proxy received invalid response|Proxy server issues
1015|TLS Handshake|TLS/SSL handshake failure|Internal use only (cannot send)
*/

// Standard Close Codes
const (
	WebSocketCloseCode_1000_NormalClosure      WebSocketCloseCode = 1000
	WebSocketCloseCode_1001_GoingAway          WebSocketCloseCode = 1001
	WebSocketCloseCode_1002_ProtocolError      WebSocketCloseCode = 1002
	WebSocketCloseCode_1003_UnsupportedData    WebSocketCloseCode = 1003
	WebSocketCloseCode_1004_Reserved           WebSocketCloseCode = 1004
	WebSocketCloseCode_1005_NoStatusReceived   WebSocketCloseCode = 1005
	WebSocketCloseCode_1006_AbnormalClosure    WebSocketCloseCode = 1006
	WebSocketCloseCode_1007_InvalidPayloadData WebSocketCloseCode = 1007
	WebSocketCloseCode_1008_PolicyViolation    WebSocketCloseCode = 1008
	WebSocketCloseCode_1009_MessageTooBig      WebSocketCloseCode = 1009
	WebSocketCloseCode_1010_MandatoryExtension WebSocketCloseCode = 1010
	WebSocketCloseCode_1011_InternalError      WebSocketCloseCode = 1011
	WebSocketCloseCode_1012_ServiceRestart     WebSocketCloseCode = 1012
	WebSocketCloseCode_1013_TryAgainLater      WebSocketCloseCode = 1013
	WebSocketCloseCode_1014_BadGateway         WebSocketCloseCode = 1014
	WebSocketCloseCode_1015_TLSHandshake       WebSocketCloseCode = 1015
)

// Indicates whether code is in Application-specific domain.
func (wscc WebSocketCloseCode) IsApplication() bool {
	v := int(wscc)

	if v < 4_000 {
		return false
	}

	if v > 4_999 {
		return false
	}

	return true
}

// Indicates whether code is in Framework-specific domain.
func (wscc WebSocketCloseCode) IsFramework() bool {
	v := int(wscc)

	if v < 3_000 {
		return false
	}

	if v > 3_999 {
		return false
	}

	return true
}

func (wscc WebSocketCloseCode) IsReserved() bool {
	v := int(wscc)

	if v < 1_000 {
		// NOTE: by inference, negative values are reserved

		return true
	}

	if v > 4_999 {
		// by inference, negative values are reserved

		return true
	}

	return false
}

// Indicates whether code is in Standard domain.
func (wscc WebSocketCloseCode) IsStandard() bool {
	v := int(wscc)

	if v < 1_000 {
		return false
	}

	if v > 2_999 {
		return false
	}

	return true
}

func (wscc WebSocketCloseCode) HasDescription() (description string, exists bool) {

	switch wscc {
	case WebSocketCloseCode_1000_NormalClosure:
		return "Connection fulfilled its purpose", true
	case WebSocketCloseCode_1001_GoingAway:
		return "Server/browser is shutting down", true
	case WebSocketCloseCode_1002_ProtocolError:
		return "Protocol violation detected", true
	case WebSocketCloseCode_1003_UnsupportedData:
		return "Received incompatible data type", true
	case WebSocketCloseCode_1004_Reserved:
		return "Reserved for future use", true
	case WebSocketCloseCode_1005_NoStatusReceived:
		return "Expected status code was absent", true
	case WebSocketCloseCode_1006_AbnormalClosure:
		return "Connection lost unexpectedly", true
	case WebSocketCloseCode_1007_InvalidPayloadData:
		return "Message data doesn’t match type", true
	case WebSocketCloseCode_1008_PolicyViolation:
		return "Generic policy violation", true
	case WebSocketCloseCode_1009_MessageTooBig:
		return "Message exceeds size limits", true
	case WebSocketCloseCode_1010_MandatoryExtension:
		return "Required extension not supported", true
	case WebSocketCloseCode_1011_InternalError:
		return "Server encountered unexpected error", true
	case WebSocketCloseCode_1012_ServiceRestart:
		return "Server is restarting", true
	case WebSocketCloseCode_1013_TryAgainLater:
		return "Temporary server overload", true
	case WebSocketCloseCode_1014_BadGateway:
		return "Gateway/proxy received invalid response", true
	case WebSocketCloseCode_1015_TLSHandshake:
		return "TLS/SSL handshake failure", true
	default:
		return "", false
	}
}
