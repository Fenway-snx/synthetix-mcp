package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"

	snx_lib_api_json "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/json"
	snx_lib_api_types "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/types"
	snx_lib_api_validation_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/api/validation/utils"
	snx_lib_request "github.com/Fenway-snx/synthetix-mcp/internal/lib/request"
)

type ClientRequestId = snx_lib_api_types.ClientRequestId

const (
	RequestIDHeader    = "X-Request-ID"
	clientRequestIdKey = "client_request_id"
	requestIdKey       = "request_id"

	MaxClientRequestIdLength = snx_lib_request.MaxClientRequestIdLength
)

// Generates a server-side internal request ID (UUID) for each request.
// If the client provides an X-Request-ID header, it is stored as the
// client request ID and echoed back in the response; it is never used
// for internal processing.
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawClientId := c.Request().Header.Get(RequestIDHeader)
			validatedClientId, err := snx_lib_api_validation_utils.ValidateClientSideString(rawClientId, MaxClientRequestIdLength, snx_lib_api_validation_utils.ClientSideOption_Trim)
			if err != nil {
				resp := snx_lib_api_json.NewValidationErrorResponse[any](
					"",
					"Request ID failed validation",
					nil,
				)
				return c.JSON(http.StatusBadRequest, resp)
			}

			clientRequestId := ClientRequestId(validatedClientId)

			requestId := snx_lib_request.NewRequestID()

			c.Set(requestIdKey, requestId)
			c.Set(clientRequestIdKey, clientRequestId)

			if clientRequestId != "" {
				c.Response().Header().Set(RequestIDHeader, string(clientRequestId))
			}

			return next(c)
		}
	}
}

// Retrieves the server-generated internal request ID from the context.
func GetRequestId(c echo.Context) snx_lib_request.RequestId {
	if requestId, ok := c.Get(requestIdKey).(snx_lib_request.RequestId); ok {
		return requestId
	}

	return ""
}

// Retrieves the client-provided request ID from the context.
func GetClientRequestId(c echo.Context) ClientRequestId {
	if clientRequestId, ok := c.Get(clientRequestIdKey).(ClientRequestId); ok {
		return clientRequestId
	}

	return ""
}
