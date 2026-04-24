package whitelist

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	snx_lib_db_redis "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/redis"
	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
)

func Test_WhitelistArbitrator_CaseInsensitiveLookup(t *testing.T) {
	t.Parallel()

	initialPermissions := PermissionsMap{
		"0xAbCd": true,
	}

	arbitrator, err := NewWhitelistArbitrator(
		snx_lib_logging_doubles.NewStubLogger(),
		context.Background(),
		&snx_lib_db_redis.SnxClient{},
		"ignored",
		nil,
		initialPermissions,
		time.Minute,
	)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	result, err := arbitrator.CanOrdersBePlacedFor("0xabcD")
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.True(t, result, "lookup should succeed despite mixed casing")

	result, err = arbitrator.CanOrdersBePlacedFor("0xFFFF")
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.False(t, result)
}
