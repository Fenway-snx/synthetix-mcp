# Logger

A structured logging package based on the Cosmos SDK's logging interface, using zerolog as the underlying implementation.

## Features

- Structured logging with key-value pairs
- Multiple log levels (Debug, Info, Warn, Error)
- JSON and console output formats
- Color support for console output
- Stack trace support for errors
- Customizable time format
- Filtering capabilities
- Hook system for custom logging behavior

## Usage

### Basic Usage

```go
import "github.com/Synthetixio/v4-offchain-lib/logger"

func main() {
    // Initialize the logger (call this once at startup)
    logger.Init(true) // true for debug mode

    // Log messages with key-value pairs
    logger.Info("Processing order", "order_id", "123", "amount", 100)
    logger.Error("Failed to process order", "error", err, "order_id", "123")
}
```

### Creating a Child Logger

```go
// Create a child logger with context
childLogger := logger.With("module", "matching")
childLogger.Info("Order matched", "price", 1000)
```

### Custom Configuration

```go
import (
    "os"
    "github.com/rs/zerolog"
    "github.com/Synthetixio/v4-offchain-lib/logger"
)

func main() {
    // Create a custom logger with specific configuration
    customLogger := logger.NewLogger(os.Stderr,
        logger.WithOutputJSON(true),
        logger.WithTimeFormat("2006-01-02 15:04:05"),
        logger.WithLevel(zerolog.DebugLevel),
        logger.WithStackTrace(true),
    )

    customLogger.Info("Custom logger message", "key", "value")
}
```

### Available Options

- `WithOutputJSON(bool)` - Configure JSON output format
- `WithColor(bool)` - Enable/disable color output
- `WithTimeFormat(string)` - Set custom time format
- `WithLevel(zerolog.Level)` - Set log level
- `WithStackTrace(bool)` - Enable/disable stack traces
- `WithFilter(func(level zerolog.Level) bool)` - Add custom filter
- `WithHooks(...zerolog.Hook)` - Add custom hooks

## Dependencies

- github.com/rs/zerolog
- github.com/pkg/errors

## License

MIT 