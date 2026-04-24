package hooks

import (
	rs_zerolog "github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// Automatically adds OpenTelemetry trace context to all log entries when a
// valid span exists in the log event context.
//
// Intentionally dormant until registered via WithHooks().
// trace.SpanFromContext is a single context.Value() lookup with no locks,
// so per-log-line overhead is negligible.
type OTelTraceHook struct{}

// Implements zerolog.Hook. Extracts the trace and span IDs from the log event
// context and adds them as structured fields for log-trace correlation in
// Grafana/Tempo.
func (h OTelTraceHook) Run(e *rs_zerolog.Event, level rs_zerolog.Level, msg string) {
	ctx := e.GetCtx()
	if ctx == nil {
		return
	}

	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return
	}

	e.Str("otel.trace_id", sc.TraceID().String())
	e.Str("otel.span_id", sc.SpanID().String())
}

// Creates a new OpenTelemetry trace hook for zerolog.
func NewOTelTraceHook() rs_zerolog.Hook {
	return OTelTraceHook{}
}
