package cors

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// Checks if the given origin is allowed by CORS policy.
//
// This is the single source of truth for CORS origin validation across all
// services.
func IsOriginAllowed(origin string) bool {
	// Normalize origin to lowercase for case-insensitive domain matching (per RFC)
	origin = strings.ToLower(origin)

	// Allow localhost for development (with or without explicit port)
	if origin == "http://localhost" || strings.HasPrefix(origin, "http://localhost:") {
		return true
	}
	if origin == "http://127.0.0.1" || strings.HasPrefix(origin, "http://127.0.0.1:") {
		return true
	}

	// Allow HTTPS localhost for local development with self-signed certs (with or without explicit port)
	if origin == "https://localhost" || strings.HasPrefix(origin, "https://localhost:") {
		return true
	}
	if origin == "https://127.0.0.1" || strings.HasPrefix(origin, "https://127.0.0.1:") {
		return true
	}

	// Allow any subdomain of snxdev.io (development/staging) - HTTPS only
	if strings.HasPrefix(origin, "https://") && (strings.HasSuffix(origin, ".snxdev.io") || origin == "https://snxdev.io") {
		return true
	}

	// Allow any subdomain of synthetix.io (production) - HTTPS only
	if strings.HasPrefix(origin, "https://") && (strings.HasSuffix(origin, ".synthetix.io") || origin == "https://synthetix.io") {
		return true
	}

	return false
}

// Applies CORS headers to a response if the origin is allowed.
//
// This ensures consistent CORS header application across all services.
func ApplyCORSHeaders(c echo.Context, origin string) {
	if IsOriginAllowed(origin) {
		c.Response().Header().Set("Access-Control-Allow-Origin", origin)
		c.Response().Header().Set("Access-Control-Allow-Methods", "GET, HEAD, PUT, PATCH, POST, DELETE, OPTIONS")
		c.Response().Header().Set("Access-Control-Allow-Headers", "*")
	}
}
