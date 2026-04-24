package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

type Config struct {
	Host     string // config: "host"
	Port     int    // config: "port"
	User     string // config: "user"
	Password string // config: "password"
	DB       string // config: "db"
	SSLMode  string // config: "ssl_mode"
	// GORM Configuration
	GormLogLevel string // config: "gorm_log_level"
	// Connection pool settings (required)
	MaxOpenConns    int           // config: "max_open_conns"
	MaxIdleConns    int           // config: "max_idle_conns"
	ConnMaxLifetime time.Duration // config: "conn_max_lifetime"
	ConnMaxIdleTime time.Duration // config: "conn_max_idle_time"
}

// LoadConfig loads PostgreSQL configuration from Viper with the given key prefix
// For example, if keyPrefix is "postgres", it will look for postgres.host, postgres.port, etc.
// Returns an error if required fields are missing or invalid.
func LoadConfig(v *viper.Viper) (*Config, error) {
	keyPrefix := "postgres"
	cfg := &Config{
		Host:            v.GetString(keyPrefix + ".host"),
		Port:            v.GetInt(keyPrefix + ".port"),
		User:            v.GetString(keyPrefix + ".user"),
		Password:        v.GetString(keyPrefix + ".password"),
		DB:              v.GetString(keyPrefix + ".db"),
		SSLMode:         v.GetString(keyPrefix + ".ssl_mode"),
		GormLogLevel:    v.GetString(keyPrefix + ".gorm_log_level"),
		MaxOpenConns:    v.GetInt(keyPrefix + ".max_open_conns"),
		MaxIdleConns:    v.GetInt(keyPrefix + ".max_idle_conns"),
		ConnMaxLifetime: v.GetDuration(keyPrefix + ".conn_max_lifetime"),
		ConnMaxIdleTime: v.GetDuration(keyPrefix + ".conn_max_idle_time"),
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid postgres config: %w", err)
	}

	return cfg, nil
}

// Validate validates the PostgreSQL configuration
func (c *Config) Validate() error {
	var errs []string

	if c.Host == "" {
		errs = append(errs, "host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		errs = append(errs, fmt.Sprintf("port must be between 1 and 65535, got %d", c.Port))
	}

	if c.User == "" {
		errs = append(errs, "user is required")
	}

	if c.Password == "" {
		errs = append(errs, "password is required")
	}

	if c.DB == "" {
		errs = append(errs, "db is required")
	}

	if c.SSLMode == "" {
		errs = append(errs, "ssl_mode is required")
	}

	if c.GormLogLevel == "" {
		errs = append(errs, "gorm_log_level is required")
	}

	// Connection pool settings are required
	if c.MaxOpenConns <= 0 {
		errs = append(errs, "max_open_conns must be greater than 0")
	}

	if c.MaxIdleConns <= 0 {
		errs = append(errs, "max_idle_conns must be greater than 0")
	}

	if c.MaxOpenConns > 0 && c.MaxIdleConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		errs = append(errs, fmt.Sprintf("max_idle_conns (%d) must be <= max_open_conns (%d)", c.MaxIdleConns, c.MaxOpenConns))
	}

	if c.ConnMaxLifetime <= 0 {
		errs = append(errs, "conn_max_lifetime must be greater than 0")
	}

	if c.ConnMaxIdleTime <= 0 {
		errs = append(errs, "conn_max_idle_time must be greater than 0")
	}

	if len(errs) > 0 {
		return fmt.Errorf("postgres config validation failed: %s", strings.Join(errs, ";\n"))
	}

	return nil
}

// CustomGormLogger integrates GORM with our existing logger
type CustomGormLogger struct {
	LogLevel gormlogger.LogLevel
	logger   snx_lib_logging.Logger
}

func NewCustomGormLogger(
	logger snx_lib_logging.Logger,
	logLevel gormlogger.LogLevel,
) *CustomGormLogger {
	return &CustomGormLogger{
		LogLevel: logLevel,
		logger:   logger,
	}
}

// LogMode sets the log level for GORM
func (l *CustomGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info level messages
func (l *CustomGormLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= gormlogger.Info {
		l.logger.Info("GORM", "message", msg, "data", data)
	}
}

// Warn logs warn level messages
func (l *CustomGormLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= gormlogger.Warn {
		l.logger.Warn("GORM", "message", msg, "data", data)
	}
}

// Error logs error level messages
func (l *CustomGormLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= gormlogger.Error {
		l.logger.Error("GORM", "message", msg, "data", data)
	}
}

// Trace logs SQL queries and execution time
func (l *CustomGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := snx_lib_utils_time.Since(begin)
	sql, rows := fc()

	if err != nil && l.LogLevel >= gormlogger.Error {
		l.logger.Error("GORM SQL Error",
			"sql", sql,
			"rows_affected", rows,
			"elapsed", elapsed.String(),
			"error", err.Error())
	} else if l.LogLevel >= gormlogger.Info {
		l.logger.Info("GORM SQL",
			"sql", sql,
			"rows_affected", rows,
			"elapsed", elapsed.String())
	}
}

// getGormLogLevel converts string log level to GORM log level
func getGormLogLevel(level string) gormlogger.LogLevel {
	switch strings.ToLower(level) {
	case "silent":
		return gormlogger.Silent
	case "error":
		return gormlogger.Error
	case "warn":
		return gormlogger.Warn
	case "info":
		return gormlogger.Info
	default:
		return gormlogger.Silent
	}
}

func (c *Config) GetPostgresDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DB, c.SSLMode)
}

func NewClient(
	logger snx_lib_logging.Logger,
	cfg Config,
) (*gorm.DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Create custom GORM logger with configurable log level
	customLogger := NewCustomGormLogger(logger, getGormLogLevel(cfg.GormLogLevel))

	// Initialize PostgreSQL connection with GORM
	dsn := cfg.GetPostgresDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:      customLogger,
		QueryFields: true,
	})
	if err != nil {
		// If the database does not exist, attempt to create it, then retry
		if isDatabaseNotExistError(err) {
			if errEnsure := ensureDatabaseExists(cfg, customLogger); errEnsure != nil {
				return nil, fmt.Errorf("failed to ensure database exists: %w (original connection error: %v)", errEnsure, err)
			}
			// Retry opening the target database
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger:      customLogger,
				QueryFields: true,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to open database connection after creating database: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to open database connection: %w", err)
		}
	}
	logger.Debug("Connected to database")

	// Test database connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		// Some drivers (e.g., pgx) surface the 3D000 error on first Ping rather than Open
		if isDatabaseNotExistError(err) {
			if errEnsure := ensureDatabaseExists(cfg, customLogger); errEnsure != nil {
				return nil, fmt.Errorf("failed to ensure database exists: %w (original ping error: %v)", errEnsure, err)
			}

			// Close the existing (failed) sql.DB before retrying
			_ = sqlDB.Close()

			// Re-open the target database now that it exists
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
				Logger:      customLogger,
				QueryFields: true,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to open database connection after creating database: %w", err)
			}

			// Re-acquire and ping the new connection
			sqlDB, err = db.DB()
			if err != nil {
				return nil, fmt.Errorf("failed to get underlying sql.DB after creating database: %w", err)
			}
			if err := sqlDB.Ping(); err != nil {
				return nil, fmt.Errorf("failed to ping database after creating database: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to ping database: %w", err)
		}
	}

	// Configure connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := RegisterPoolMetrics(sqlDB, cfg.DB, cfg.Host); err != nil {
		logger.Warn("Failed to register database pool metrics", "error", err)
	}

	return db, nil
}

// isDatabaseNotExistError checks error messages for the PostgreSQL SQLSTATE 3D000
// or typical phrases indicating that the database does not exist.
func isDatabaseNotExistError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.InvalidCatalogName
	}
	return false
}

// isDuplicateDatabaseError checks for PostgreSQL SQLSTATE 42P04 (duplicate_database)
// which can occur if multiple instances concurrently attempt to CREATE DATABASE.
func isDuplicateDatabaseError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.DuplicateDatabase
	}
	return false
}

// ensureDatabaseExists connects to the default "postgres" database and
// creates the target database if it does not already exist.
func ensureDatabaseExists(cfg Config, gormLogger gormlogger.Interface) error {
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, "postgres", cfg.SSLMode)

	adminDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return fmt.Errorf("failed to open admin connection to postgres database: %w", err)
	}

	// Ensure we close the temporary admin connection
	adminSQL, err := adminDB.DB()
	if err != nil {
		return fmt.Errorf("failed to obtain admin sql.DB: %w", err)
	}
	defer adminSQL.Close()

	// Check existence via pg_database using COUNT to avoid ErrRecordNotFound
	var count int
	if err := adminDB.Raw("SELECT COUNT(1) FROM pg_database WHERE datname = ?", cfg.DB).Scan(&count).Error; err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}
	if count > 0 {
		return nil
	}

	// Safely quote the database name for CREATE DATABASE
	sanitizedName := strings.ReplaceAll(cfg.DB, "\"", "\"\"")
	createStmt := fmt.Sprintf("CREATE DATABASE \"%s\"", sanitizedName)
	if err := adminDB.Exec(createStmt).Error; err != nil {
		// If another instance created the database concurrently, treat as success
		if isDuplicateDatabaseError(err) {
			return nil
		}
		return fmt.Errorf("failed to create database %q: %w", cfg.DB, err)
	}

	return nil
}
