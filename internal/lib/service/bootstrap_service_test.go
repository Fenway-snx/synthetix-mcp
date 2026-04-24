package service

import (
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	snx_lib_runtime "github.com/Fenway-snx/synthetix-mcp/internal/lib/runtime"
)

type testConfig struct {
	ServiceConfigCommon
}

func (testConfig) CanonicalString() string { return "local" }

func (testConfig) IsLocal() bool       { return true }
func (testConfig) IsDevelopment() bool { return false }
func (testConfig) IsStaging() bool     { return false }
func (testConfig) IsProduction() bool  { return false }
func (testConfig) IsCustom() bool      { return false }

func Test_LogLevel(t *testing.T) {
	c := ServiceConfigCommon{logLevel: "debug"}
	assert.Equal(t, "debug", c.LogLevel())
}

func Test_LogOutputJSON(t *testing.T) {
	c := ServiceConfigCommon{logOutputJSON: true}
	assert.True(t, c.LogOutputJSON())
}

func Test_LogTags(t *testing.T) {
	c := ServiceConfigCommon{logTags: "env:prod,region:ap-northeast-1"}
	assert.Equal(t, "env:prod,region:ap-northeast-1", c.LogTags())
}

func Test_CreateBootstrapLogger_WithLogTags(t *testing.T) {
	logger, cfg, err := createBootstrapLogger(func() (*testConfig, error) {
		return &testConfig{ServiceConfigCommon{logLevel: "info", logOutputJSON: true, logTags: "env:prod,region:ap-northeast-1"}}, nil
	})
	assert.NotNil(t, logger)
	assert.NotNil(t, cfg)
	assert.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
}

func Test_CreateBootstrapLogger_ConfigSuccess(t *testing.T) {
	logger, cfg, err := createBootstrapLogger(func() (*testConfig, error) {
		return &testConfig{ServiceConfigCommon{logLevel: "warn", logOutputJSON: true, logTags: "env:local"}}, nil
	})
	assert.NotNil(t, logger)
	assert.NotNil(t, cfg)
	assert.NoError(t, err, "createBootstrapLogger")
}

func Test_CreateBootstrapLogger_InvalidLogTags(t *testing.T) {
	logger, cfg, err := createBootstrapLogger(func() (*testConfig, error) {
		return &testConfig{ServiceConfigCommon{logLevel: "info", logOutputJSON: true, logTags: "badtag"}}, nil
	})
	assert.NotNil(t, logger)
	assert.NotNil(t, cfg)
	assert.ErrorContains(t, err, "missing ':'")
}

func Test_CreateBootstrapLogger_ConfigError(t *testing.T) {
	logger, cfg, err := createBootstrapLogger(func() (*testConfig, error) {
		return nil, errors.New("fail")
	})
	assert.NotNil(t, logger)
	assert.Nil(t, cfg)
	assert.Error(t, err)
}

func Test_BootstrapService_Normal(t *testing.T) {
	called := false
	BootstrapService("test-svc",
		func() (*testConfig, error) {
			return &testConfig{ServiceConfigCommon{logLevel: "info", logOutputJSON: true, logTags: "env:local"}}, nil
		},
		func(dc snx_lib_runtime.DiagnosticContext, ec snx_lib_runtime.ExecutionContext, _ *testConfig) {
			called = true
			ec.CancelFunc()()
		},
	)
	assert.True(t, called)
}

func Test_BootstrapService_panic(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^Test_BootstrapService_panic_SUBPROCESS$")
	cmd.Env = append(os.Environ(), "RUN_PANIC_TEST=1")
	err := cmd.Run()
	assert.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	assert.True(t, ok)
	assert.Equal(t, 1, exitErr.ExitCode())
}

func Test_BootstrapService_panic_SUBPROCESS(t *testing.T) {
	if os.Getenv("RUN_PANIC_TEST") != "1" {
		return
	}
	BootstrapService("panic-svc",
		func() (*testConfig, error) {
			return &testConfig{ServiceConfigCommon{logLevel: "info", logOutputJSON: true, logTags: "env:local"}}, nil
		},
		func(_ snx_lib_runtime.DiagnosticContext, _ snx_lib_runtime.ExecutionContext, _ *testConfig) {
			panic("test panic")
		},
	)
}
