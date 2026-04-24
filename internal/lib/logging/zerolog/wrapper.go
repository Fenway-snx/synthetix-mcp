package zerolog

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
	rs_zerolog "github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
	snx_lib_logging_utils "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging/utils"
)

func init() {
	rs_zerolog.InterfaceMarshalFunc = func(i any) ([]byte, error) {
		switch v := i.(type) {
		case json.Marshaler:
			return json.Marshal(i)
		case encoding.TextMarshaler:
			return json.Marshal(i)
		case fmt.Stringer:
			return json.Marshal(v.String())
		default:
			return json.Marshal(i)
		}
	}
}

type wrapper struct {
	*rs_zerolog.Logger
}

var _ snx_lib_logging.Logger = wrapper{}

// NewLogger returns a new logger that writes to the given destination.
func NewLogger(dst io.Writer, options ...Option) snx_lib_logging.Logger {
	logCfg := defaultConfig
	for _, opt := range options {
		opt(&logCfg)
	}

	output := dst
	if !logCfg.OutputJSON {
		output = rs_zerolog.ConsoleWriter{
			Out:        dst,
			NoColor:    !logCfg.Color,
			TimeFormat: logCfg.TimeFormat,
		}
	}

	if logCfg.Filter != nil {
		output = NewFilterWriter(output, logCfg.Filter)
	}

	logger := rs_zerolog.New(output)
	if logCfg.StackTrace {
		rs_zerolog.ErrorStackMarshaler = func(err error) any {
			return pkgerrors.MarshalStack(errors.WithStack(err))
		}

		logger = logger.With().Stack().Logger()
	}

	if logCfg.TimeFormat != "" {
		logger = logger.With().Timestamp().Logger()
	}

	if logCfg.Level != rs_zerolog.NoLevel {
		logger = logger.Level(logCfg.Level)
	}

	logger = logger.Hook(logCfg.Hooks...)

	return wrapper{&logger}
}

func (w wrapper) Debug(msg string, keyVals ...any) {
	ev := w.Logger.Debug()
	dispatchEv(ev, msg, keyVals...)
}

func (w wrapper) Info(msg string, keyVals ...any) {
	ev := w.Logger.Info()
	dispatchEv(ev, msg, keyVals...)
}

func (w wrapper) Warn(msg string, keyVals ...any) {
	ev := w.Logger.Warn()
	dispatchEv(ev, msg, keyVals...)
}

func (w wrapper) Error(msg string, keyVals ...any) {
	ev := w.Logger.Error()
	dispatchEv(ev, msg, keyVals...)
}

func (w wrapper) With(keyVals ...any) snx_lib_logging.Logger {
	logger := w.Logger.With().Fields(snx_lib_logging_utils.StringifyAllBigInts(keyVals, 0)).Logger()
	return wrapper{&logger}
}

// Defers the potentially expensive conversions carried out by
// StringifyAllBigInts until it is known that events are being
// emitted at the given severity level.
func dispatchEv(ev *rs_zerolog.Event, msg string, keyVals ...any) {
	if ev != nil {
		ev.Fields(snx_lib_logging_utils.StringifyAllBigInts(keyVals, 0)).Msg(msg)
	}
}
