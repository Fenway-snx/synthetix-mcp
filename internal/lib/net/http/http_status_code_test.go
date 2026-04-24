package http

import (
	net_http "net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_HTTPStatusCode_VALUES(t *testing.T) {

	tests := []struct {
		httpStatusCode HTTPStatusCode
		nh_value       int
		int_value      int
	}{
		// 2xx
		{
			HTTPStatusCode_200_OK,
			net_http.StatusOK,
			200,
		},
		// 4xx
		{
			HTTPStatusCode_400_BadRequest,
			net_http.StatusBadRequest,
			400,
		},
		{
			HTTPStatusCode_401_Unauthorized,
			net_http.StatusUnauthorized,
			401,
		},
		{
			HTTPStatusCode_403_Forbidden,
			net_http.StatusForbidden,
			403,
		},
		{
			HTTPStatusCode_404_NotFound,
			net_http.StatusNotFound,
			404,
		},
		{
			HTTPStatusCode_409_Conflict,
			net_http.StatusConflict,
			409,
		},
		// 5xx
		{
			HTTPStatusCode_500_InternalServerError,
			net_http.StatusInternalServerError,
			500,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, int(tt.httpStatusCode), tt.nh_value)
		assert.Equal(t, int(tt.httpStatusCode), tt.int_value)
	}
}
