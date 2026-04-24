package postgres

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
	"gorm.io/gorm"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

var (
	errWriterConfigRequired = errors.New("writer config is required")
)

// DBRequirement declares which database handles a service needs.
// Composed via bitwise OR: RequireWriter | RequireReader.
type DBRequirement int

const (
	RequireWriter DBRequirement = 1 << iota
	RequireReader
	RequireBoth = RequireWriter | RequireReader
)

func (r DBRequirement) needsWriter() bool { return r&RequireWriter != 0 }
func (r DBRequirement) needsReader() bool { return r&RequireReader != 0 }

// ReaderWriterConfig holds connection config for primary (writer) and replica (reader) nodes.
// Which fields are required depends on the DBRequirement passed to LoadReaderWriterConfig.
type ReaderWriterConfig struct {
	Writer *Config
	Reader *Config
}

// ReaderWriter manages distinct *gorm.DB handles for writer and reader connections.
// Only the handles matching the declared DBRequirement are opened.
type ReaderWriter struct {
	writer *gorm.DB
	reader *gorm.DB
}

// LoadReaderWriterConfig loads database config from Viper keys and validates that
// the handles required by req are present. Writer config comes from postgres.*,
// reader config from postgres.reader.*. Returns an error if a required config is
// missing or invalid.
func LoadReaderWriterConfig(v *viper.Viper, req DBRequirement) (*ReaderWriterConfig, error) {
	cfg := &ReaderWriterConfig{}

	if req.needsWriter() {
		writerCfg, err := LoadConfig(v)
		if err != nil {
			return nil, fmt.Errorf("failed to load writer postgres config: %w", err)
		}
		cfg.Writer = writerCfg
	}

	if req.needsReader() {
		readerPrefix := "postgres.reader"
		if v.GetString(readerPrefix+".host") == "" {
			return nil, fmt.Errorf("reader postgres config is required: postgres.reader.host is not set")
		}

		readerCfg := &Config{
			Host:            v.GetString(readerPrefix + ".host"),
			Port:            v.GetInt(readerPrefix + ".port"),
			User:            v.GetString(readerPrefix + ".user"),
			Password:        v.GetString(readerPrefix + ".password"),
			DB:              v.GetString(readerPrefix + ".db"),
			SSLMode:         v.GetString(readerPrefix + ".ssl_mode"),
			GormLogLevel:    v.GetString(readerPrefix + ".gorm_log_level"),
			MaxOpenConns:    v.GetInt(readerPrefix + ".max_open_conns"),
			MaxIdleConns:    v.GetInt(readerPrefix + ".max_idle_conns"),
			ConnMaxLifetime: v.GetDuration(readerPrefix + ".conn_max_lifetime"),
			ConnMaxIdleTime: v.GetDuration(readerPrefix + ".conn_max_idle_time"),
		}

		if err := readerCfg.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate reader postgres config: %w", err)
		}
		cfg.Reader = readerCfg
	}

	return cfg, nil
}

// NewReaderWriter opens database connections for the handles declared by req.
// Returns an error if a required config is nil or the connection fails.
func NewReaderWriter(logger snx_lib_logging.Logger, cfg ReaderWriterConfig, req DBRequirement) (*ReaderWriter, error) {
	rw := &ReaderWriter{}

	if req.needsWriter() {
		if cfg.Writer == nil {
			return nil, errWriterConfigRequired
		}
		writerDB, err := NewClient(logger, *cfg.Writer)
		if err != nil {
			return nil, fmt.Errorf("failed to create writer connection: %w", err)
		}
		rw.writer = writerDB
	}

	if req.needsReader() {
		if cfg.Reader == nil {
			return nil, fmt.Errorf("reader config is required")
		}
		readerDB, err := NewClient(logger, *cfg.Reader)
		if err != nil {
			if rw.writer != nil {
				closeDB(rw.writer)
			}
			return nil, fmt.Errorf("failed to create reader connection: %w", err)
		}
		rw.reader = readerDB
	}

	return rw, nil
}

// Writer returns the primary (read-write) database handle.
func (rw *ReaderWriter) Writer() *gorm.DB {
	return rw.writer
}

// Reader returns the replica (read-only) database handle.
func (rw *ReaderWriter) Reader() *gorm.DB {
	return rw.reader
}

// Close closes all open connections.
func (rw *ReaderWriter) Close() error {
	var readerErr, writerErr error

	if rw.reader != nil {
		if err := closeDB(rw.reader); err != nil {
			readerErr = fmt.Errorf("failed to close reader connection: %w", err)
		}
	}

	if rw.writer != nil {
		if err := closeDB(rw.writer); err != nil {
			writerErr = fmt.Errorf("failed to close writer connection: %w", err)
		}
	}

	return errors.Join(readerErr, writerErr)
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
