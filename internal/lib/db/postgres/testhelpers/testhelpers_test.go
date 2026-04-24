package testhelpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// DSN
// ---------------------------------------------------------------------------

func Test_DSN_defaults(t *testing.T) {
	// Clear any overrides so the hardcoded defaults are exercised.
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_SSL_MODE", "")

	dsn := DSN("mydb")
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5432")
	assert.Contains(t, dsn, "user=postgres")
	assert.Contains(t, dsn, "password=postgres")
	assert.Contains(t, dsn, "dbname=mydb")
	assert.Contains(t, dsn, "sslmode=disable")
}

func Test_DSN_env_overrides(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "db.example.com")
	t.Setenv("POSTGRES_PORT", "5433")
	t.Setenv("POSTGRES_USER", "admin")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_SSL_MODE", "require")

	dsn := DSN("testdb")
	assert.Contains(t, dsn, "host=db.example.com")
	assert.Contains(t, dsn, "port=5433")
	assert.Contains(t, dsn, "user=admin")
	assert.Contains(t, dsn, "password=secret")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=require")
}

// ---------------------------------------------------------------------------
// quoteIdent
// ---------------------------------------------------------------------------

func Test_quoteIdent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"plain name", "mydb", `"mydb"`},
		{"name with hyphen", "my-db", `"my-db"`},
		{"name with space", "my db", `"my db"`},
		{"SQL keyword", "select", `"select"`},
		{"embedded double quote", `say "hi"`, `"say ""hi"""`},
		{"multiple double quotes", `a"b"c`, `"a""b""c"`},
		{"empty string", "", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, quoteIdent(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// NewDB — smoke test (requires a running PostgreSQL instance)
// ---------------------------------------------------------------------------

func Test_NewDB_smoke(t *testing.T) {
	db := NewDB(t, "test_helpers_smoke")
	require.NotNil(t, db)
	sqlDB, err := db.DB()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NoError(t, sqlDB.Ping(), "expected database to be reachable after NewDB")
}