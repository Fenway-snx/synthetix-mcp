package http

import net_http "net/http"

// Strong type that represents an HTTP status code.
type HTTPStatusCode int

// 2xx
const (
	HTTPStatusCode_200_OK HTTPStatusCode = net_http.StatusOK
)

// 4xx
const (
	HTTPStatusCode_400_BadRequest            HTTPStatusCode = net_http.StatusBadRequest
	HTTPStatusCode_401_Unauthorized          HTTPStatusCode = net_http.StatusUnauthorized
	HTTPStatusCode_403_Forbidden             HTTPStatusCode = net_http.StatusForbidden
	HTTPStatusCode_404_NotFound              HTTPStatusCode = net_http.StatusNotFound
	HTTPStatusCode_409_Conflict              HTTPStatusCode = net_http.StatusConflict
	HTTPStatusCode_429_StatusTooManyRequests HTTPStatusCode = net_http.StatusTooManyRequests
)

// 5xx
const (
	HTTPStatusCode_500_InternalServerError      HTTPStatusCode = net_http.StatusInternalServerError
	HTTPStatusCode_501_StatusNotImplemented     HTTPStatusCode = net_http.StatusNotImplemented
	HTTPStatusCode_503_StatusServiceUnavailable HTTPStatusCode = net_http.StatusServiceUnavailable
)
