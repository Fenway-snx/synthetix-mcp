// Package testhelpers provides PostgreSQL test utilities shared across all services.
// It is intended to be imported only from _test.go files.
package testhelpers

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// dbCounter generates unique database name suffixes. Using an atomic counter
// (rather than time.Now() or snx_lib_utils_time.Now()) avoids collisions both
// when tests run faster than nanosecond resolution and when a test freezes the
// global time provider via SetTimeProvider.
var dbCounter atomic.Int64

// DBOpener opens a *sql.DB for the given DSN. Use with NewDBWithOpener when
// the default postgres driver needs replacing (e.g. custom pgx type codecs).
type DBOpener func(dsn string) (*sql.DB, error)

// DSN builds a PostgreSQL DSN for dbName, reading host/port/credentials from
// the standard POSTGRES_* environment variables (defaulting to localhost:5432
// with user postgres / password postgres / sslmode disable).
func DSN(dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOrDefault("POSTGRES_HOST", "localhost"),
		envOrDefault("POSTGRES_PORT", "5432"),
		envOrDefault("POSTGRES_USER", "postgres"),
		envOrDefault("POSTGRES_PASSWORD", "postgres"),
		dbName,
		envOrDefault("POSTGRES_SSL_MODE", "disable"),
	)
}

// NewDB creates a temporary PostgreSQL database scoped to t. The database is
// created before the test and dropped (WITH FORCE) after it completes.
// models is passed directly to GORM AutoMigrate; pass none to skip migration.
func NewDB(t *testing.T, dbNamePrefix string, models ...any) *gorm.DB {
	t.Helper()
	return newDB(t, dbNamePrefix, nil, models...)
}

// NewDBWithOpener is like NewDB but uses opener to obtain the underlying
// *sql.DB instead of the default postgres driver. Use this when you need
// non-default driver configuration (e.g. registering custom pgx type codecs).
func NewDBWithOpener(t *testing.T, dbNamePrefix string, opener DBOpener, models ...any) *gorm.DB {
	t.Helper()
	return newDB(t, dbNamePrefix, opener, models...)
}

func newDB(t *testing.T, dbNamePrefix string, opener DBOpener, models ...any) *gorm.DB {
	t.Helper()
	testDBName := fmt.Sprintf("%s_%d", dbNamePrefix, dbCounter.Add(1))

	adminDB, err := gorm.Open(postgres.Open(DSN("postgres")), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	quotedName := quoteIdent(testDBName)
	require.NoError(t, adminDB.Exec("CREATE DATABASE "+quotedName).Error)
	t.Cleanup(func() {
		adminDB.Exec("DROP DATABASE IF EXISTS " + quotedName + " WITH (FORCE)")
		if sqlDB, dbErr := adminDB.DB(); dbErr == nil {
			sqlDB.Close()
		}
	})

	var db *gorm.DB
	if opener != nil {
		rawDB, openErr := opener(DSN(testDBName))
		require.NoError(t, openErr, "open test db: %s", openErr)
		t.Cleanup(func() { rawDB.Close() })
		db, err = gorm.Open(postgres.New(postgres.Config{Conn: rawDB}), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
	} else {
		db, err = gorm.Open(postgres.Open(DSN(testDBName)), &gorm.Config{
			Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		})
		t.Cleanup(func() {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				sqlDB.Close()
			}
		})
	}
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	if len(models) > 0 {
		require.NoError(t, db.AutoMigrate(models...))
	}
	return db
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// quoteIdent wraps a PostgreSQL identifier in double-quotes and escapes any
// embedded double-quotes by doubling them, per the SQL standard. This prevents
// identifiers containing hyphens, spaces, keywords, or other special characters
// from causing syntax errors or injection in CREATE/DROP DATABASE statements.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}