package postgres

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	snx_lib_logging_doubles "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/doubles"
	snx_lib_db_testhelpers "github.com/Fenway-snx/synthetix-mcp/internal/lib/db/postgres/testhelpers"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return snx_lib_db_testhelpers.NewDB(t, "test_readerwriter")
}

func setViperWriterConfig(v *viper.Viper) {
	v.Set("postgres.host", "localhost")
	v.Set("postgres.port", 5432)
	v.Set("postgres.user", "test_user")
	v.Set("postgres.password", "test_pass")
	v.Set("postgres.db", "test_db")
	v.Set("postgres.ssl_mode", "disable")
	v.Set("postgres.gorm_log_level", "silent")
	v.Set("postgres.max_open_conns", 10)
	v.Set("postgres.max_idle_conns", 5)
	v.Set("postgres.conn_max_lifetime", "1h")
	v.Set("postgres.conn_max_idle_time", "30m")
}

func setViperReaderConfig(v *viper.Viper) {
	v.Set("postgres.reader.host", "replica.local")
	v.Set("postgres.reader.port", 5433)
	v.Set("postgres.reader.user", "reader_user")
	v.Set("postgres.reader.password", "reader_pass")
	v.Set("postgres.reader.db", "test_db")
	v.Set("postgres.reader.ssl_mode", "disable")
	v.Set("postgres.reader.gorm_log_level", "silent")
	v.Set("postgres.reader.max_open_conns", 20)
	v.Set("postgres.reader.max_idle_conns", 10)
	v.Set("postgres.reader.conn_max_lifetime", "1h")
	v.Set("postgres.reader.conn_max_idle_time", "30m")
}

// --- DBRequirement helpers ---

func Test_DBRequirement_NEEDS_WRITER_AND_READER_FLAGS(t *testing.T) {
	assert.True(t, RequireWriter.needsWriter())
	assert.False(t, RequireWriter.needsReader())

	assert.False(t, RequireReader.needsWriter())
	assert.True(t, RequireReader.needsReader())

	assert.True(t, RequireBoth.needsWriter())
	assert.True(t, RequireBoth.needsReader())
}

// --- LoadReaderWriterConfig ---

func Test_LoadReaderWriterConfig_REQUIRE_BOTH(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v)
	setViperReaderConfig(v)

	cfg, err := LoadReaderWriterConfig(v, RequireBoth)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, cfg.Writer)
	require.NotNil(t, cfg.Reader)
	assert.Equal(t, "localhost", cfg.Writer.Host)
	assert.Equal(t, "replica.local", cfg.Reader.Host)
	assert.Equal(t, 5433, cfg.Reader.Port)
}

func Test_LoadReaderWriterConfig_REQUIRE_BOTH_MISSING_READER(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v)

	_, err := LoadReaderWriterConfig(v, RequireBoth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reader postgres config is required")
}

func Test_LoadReaderWriterConfig_REQUIRE_BOTH_MISSING_WRITER(t *testing.T) {
	v := viper.New()
	setViperReaderConfig(v)

	_, err := LoadReaderWriterConfig(v, RequireBoth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load writer postgres config")
}

func Test_LoadReaderWriterConfig_REQUIRE_WRITER_ONLY(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v)

	cfg, err := LoadReaderWriterConfig(v, RequireWriter)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, cfg.Writer)
	assert.Nil(t, cfg.Reader)
	assert.Equal(t, "localhost", cfg.Writer.Host)
}

func Test_LoadReaderWriterConfig_REQUIRE_WRITER_ONLY_MISSING_WRITER(t *testing.T) {
	v := viper.New()

	_, err := LoadReaderWriterConfig(v, RequireWriter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load writer postgres config")
}

func Test_LoadReaderWriterConfig_REQUIRE_READER_ONLY(t *testing.T) {
	v := viper.New()
	setViperReaderConfig(v)

	cfg, err := LoadReaderWriterConfig(v, RequireReader)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	assert.Nil(t, cfg.Writer)
	require.NotNil(t, cfg.Reader)
	assert.Equal(t, "replica.local", cfg.Reader.Host)
}

func Test_LoadReaderWriterConfig_REQUIRE_READER_ONLY_MISSING_READER(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v) // writer present but reader missing

	_, err := LoadReaderWriterConfig(v, RequireReader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reader postgres config is required")
}

func Test_LoadReaderWriterConfig_REQUIRE_READER_ONLY_EMPTY_HOST(t *testing.T) {
	v := viper.New()
	v.Set("postgres.reader.host", "")
	v.Set("postgres.reader.port", 5433)

	_, err := LoadReaderWriterConfig(v, RequireReader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reader postgres config is required")
}

func Test_LoadReaderWriterConfig_INVALID_READER(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v)
	v.Set("postgres.reader.host", "replica.local")
	v.Set("postgres.reader.port", -1)

	_, err := LoadReaderWriterConfig(v, RequireBoth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate reader postgres config")
}

func Test_LoadReaderWriterConfig_READER_MISSING_PASSWORD(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v)
	v.Set("postgres.reader.host", "replica.local")
	v.Set("postgres.reader.port", 5433)
	v.Set("postgres.reader.user", "ro_user")
	v.Set("postgres.reader.db", "test_db")
	v.Set("postgres.reader.ssl_mode", "disable")
	v.Set("postgres.reader.gorm_log_level", "silent")
	v.Set("postgres.reader.max_open_conns", 10)
	v.Set("postgres.reader.max_idle_conns", 5)
	v.Set("postgres.reader.conn_max_lifetime", "1h")
	v.Set("postgres.reader.conn_max_idle_time", "30m")

	_, err := LoadReaderWriterConfig(v, RequireBoth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate reader postgres config")
}

func Test_LoadReaderWriterConfig_REQUIRE_READER_POOL_FIELDS(t *testing.T) {
	v := viper.New()
	setViperReaderConfig(v)

	v.Set("postgres.reader.host", "replica.local")
	v.Set("postgres.reader.port", 5433)
	v.Set("postgres.reader.user", "ro_user")
	v.Set("postgres.reader.password", "ro_pass")
	v.Set("postgres.reader.db", "replica_db")
	v.Set("postgres.reader.ssl_mode", "require")
	v.Set("postgres.reader.gorm_log_level", "warn")
	v.Set("postgres.reader.max_open_conns", 50)
	v.Set("postgres.reader.max_idle_conns", 25)
	v.Set("postgres.reader.conn_max_lifetime", "2h")
	v.Set("postgres.reader.conn_max_idle_time", "45m")

	cfg, err := LoadReaderWriterConfig(v, RequireReader)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, cfg.Reader)

	assert.Equal(t, 50, cfg.Reader.MaxOpenConns)
	assert.Equal(t, 25, cfg.Reader.MaxIdleConns)
	assert.Equal(t, 2*time.Hour, cfg.Reader.ConnMaxLifetime)
	assert.Equal(t, 45*time.Minute, cfg.Reader.ConnMaxIdleTime)
}

// --- NewReaderWriter ---

func Test_NewReaderWriter_NIL_WRITER_CONFIG(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	_, err := NewReaderWriter(logger, ReaderWriterConfig{}, RequireWriter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writer config is required")
}

func Test_NewReaderWriter_NIL_READER_CONFIG(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	_, err := NewReaderWriter(logger, ReaderWriterConfig{}, RequireReader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reader config is required")
}

func Test_NewReaderWriter_REQUIRE_BOTH_NIL_WRITER(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	cfg := ReaderWriterConfig{
		Reader: &Config{
			Host:            "replica.local",
			Port:            5433,
			User:            "test",
			Password:        "test",
			DB:              "test",
			SSLMode:         "disable",
			GormLogLevel:    "silent",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 1,
			ConnMaxIdleTime: 1,
		},
	}

	_, err := NewReaderWriter(logger, cfg, RequireBoth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writer config is required")
}

func Test_NewReaderWriter_REQUIRE_BOTH_NIL_BOTH(t *testing.T) {
	logger := snx_lib_logging_doubles.NewStubLogger()

	_, err := NewReaderWriter(logger, ReaderWriterConfig{}, RequireBoth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writer config is required")
}

// --- ReaderWriter handles ---

func Test_ReaderWriter_BOTH_HANDLES(t *testing.T) {
	writerDB := openTestDB(t)
	readerDB := openTestDB(t)

	rw := &ReaderWriter{writer: writerDB, reader: readerDB}

	assert.Equal(t, writerDB, rw.Writer())
	assert.Equal(t, readerDB, rw.Reader())
	assert.NotEqual(t, rw.Writer(), rw.Reader())
}

func Test_ReaderWriter_WRITER_ONLY(t *testing.T) {
	writerDB := openTestDB(t)

	rw := &ReaderWriter{writer: writerDB}

	assert.Equal(t, writerDB, rw.Writer())
	assert.Nil(t, rw.Reader(), "Reader() should return nil when not configured")
}

func Test_ReaderWriter_READER_ONLY(t *testing.T) {
	readerDB := openTestDB(t)

	rw := &ReaderWriter{reader: readerDB}

	assert.Nil(t, rw.Writer(), "Writer() should return nil when not configured")
	assert.Equal(t, readerDB, rw.Reader())
}

// --- Close ---

func Test_ReaderWriter_Close_BOTH(t *testing.T) {
	writerDB := openTestDB(t)
	readerDB := openTestDB(t)
	rw := &ReaderWriter{writer: writerDB, reader: readerDB}

	err := rw.Close()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	sqlWriter, _ := writerDB.DB()
	assert.Error(t, sqlWriter.Ping(), "writer connection should be closed")

	sqlReader, _ := readerDB.DB()
	assert.Error(t, sqlReader.Ping(), "reader connection should be closed")
}

func Test_ReaderWriter_Close_WRITER_ONLY(t *testing.T) {
	writerDB := openTestDB(t)
	rw := &ReaderWriter{writer: writerDB}

	err := rw.Close()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	sqlDB, _ := writerDB.DB()
	assert.Error(t, sqlDB.Ping(), "writer connection should be closed")
}

func Test_ReaderWriter_Close_READER_ONLY(t *testing.T) {
	readerDB := openTestDB(t)
	rw := &ReaderWriter{reader: readerDB}

	err := rw.Close()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	sqlDB, _ := readerDB.DB()
	assert.Error(t, sqlDB.Ping(), "reader connection should be closed")
}

// --- closeDB ---

func Test_closeDB(t *testing.T) {
	t.Run("succeeds on open connection", func(t *testing.T) {
		db := openTestDB(t)
		err := closeDB(db)
		require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

		sqlDB, _ := db.DB()
		assert.Error(t, sqlDB.Ping(), "should not be pingable after close")
	})
}

func Test_ReaderWriter_Close_WRITER_STILL_CLOSES_WHEN_READER_IS_NIL(t *testing.T) {
	writerDB := openTestDB(t)
	rw := &ReaderWriter{writer: writerDB, reader: nil}

	err := rw.Close()
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)

	sqlWriter, _ := writerDB.DB()
	assert.Error(t, sqlWriter.Ping(), "writer should be closed")
}

func Test_ReaderWriter_READER_RETURNS_READER_WHEN_SET(t *testing.T) {
	writerDB := openTestDB(t)
	readerDB := openTestDB(t)

	rw := &ReaderWriter{writer: writerDB, reader: readerDB}

	assert.Equal(t, readerDB, rw.Reader())
	assert.NotEqual(t, rw.Writer(), rw.Reader())
}

func Test_LoadReaderWriterConfig_REQUIRE_BOTH_READER_POOL_FIELDS(t *testing.T) {
	v := viper.New()
	setViperWriterConfig(v)

	v.Set("postgres.reader.host", "replica.local")
	v.Set("postgres.reader.port", 5433)
	v.Set("postgres.reader.user", "ro_user")
	v.Set("postgres.reader.password", "ro_pass")
	v.Set("postgres.reader.db", "replica_db")
	v.Set("postgres.reader.ssl_mode", "require")
	v.Set("postgres.reader.gorm_log_level", "warn")
	v.Set("postgres.reader.max_open_conns", 50)
	v.Set("postgres.reader.max_idle_conns", 25)
	v.Set("postgres.reader.conn_max_lifetime", "2h")
	v.Set("postgres.reader.conn_max_idle_time", "45m")

	cfg, err := LoadReaderWriterConfig(v, RequireBoth)
	require.NoError(t, err, "expected `err` to be `nil`, but it was '%[1]s' (%[1]T)", err)
	require.NotNil(t, cfg.Reader)

	assert.Equal(t, 50, cfg.Reader.MaxOpenConns)
	assert.Equal(t, 25, cfg.Reader.MaxIdleConns)
	assert.Equal(t, 2*time.Hour, cfg.Reader.ConnMaxLifetime)
	assert.Equal(t, 45*time.Minute, cfg.Reader.ConnMaxIdleTime)
}
