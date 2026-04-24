package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func Test_AdminHTTPServer_StartRegistersHealthEndpoint(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	cfg := &Config{Port: getFreePort(t)}
	server := NewAdminHTTPServer(logger, cfg)

	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Shutdown(context.Background())
	})

	response := waitForGet(t, fmt.Sprintf("http://127.0.0.1:%d/admin/health", cfg.Port))
	require.NotNil(t, response)
	defer response.Body.Close()

	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func Test_AdminHTTPServer_RegisterRouteBeforeStart(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	cfg := &Config{Port: getFreePort(t)}
	server := NewAdminHTTPServer(logger, cfg)

	server.RegisterRoute("/admin/custom", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Shutdown(context.Background())
	})

	response := waitForGet(t, fmt.Sprintf("http://127.0.0.1:%d/admin/custom", cfg.Port))
	require.NotNil(t, response)
	defer response.Body.Close()

	assert.Equal(t, http.StatusNoContent, response.StatusCode)
}

func Test_AdminHTTPServer_ShutdownStopsServer(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()
	cfg := &Config{Port: getFreePort(t)}
	server := NewAdminHTTPServer(logger, cfg)

	require.NoError(t, server.Start())
	require.NoError(t, server.Shutdown(context.Background()))
}

func getFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	defer listener.Close()

	tcpAddress, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return tcpAddress.Port
}

func waitForGet(t *testing.T, url string) *http.Response {
	t.Helper()

	var response *http.Response
	var err error

	for range 50 {
		response, err = http.Get(url)
		if err == nil {
			return response
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	return nil
}
