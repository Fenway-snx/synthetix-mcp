package halt

import (
	"net/http"

	snx_lib_runtime_admin "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime/admin"
)

// Wraps an HTTP handler with admin API key authentication.
func AdminAuthMiddleware(apiKey string, next http.Handler) http.Handler {
	return snx_lib_runtime_admin.AdminAuthMiddleware(apiKey, next)
}
