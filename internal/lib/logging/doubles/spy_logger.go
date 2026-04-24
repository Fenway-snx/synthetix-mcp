package doubles

import (
	"strings"
	"sync"

	snx_lib_logging "github.com/Fenway-snx/synthetix-mcp/internal/lib/logging"
)

// Severity of a recorded log entry.
type Level string

const (
	LevelDebug Level = "debug"
	LevelError Level = "error"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
)

// A single recorded log call.
type Entry struct {
	Level   Level
	Message string
	KeyVals []any
}

// Shared, mutex-protected store so that child loggers produced by With()
// write to the same slice as the root.
type entryStore struct {
	mu      sync.Mutex
	entries []Entry
}

func (s *entryStore) append(e Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, e)
}

func (s *entryStore) snapshot() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *entryStore) reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = s.entries[:0]
}

// A [snx_lib_logging.Logger] that records every call for later
// inspection. Safe for concurrent use.
type SpyLogger struct {
	store   *entryStore
	context []any
}

var _ snx_lib_logging.Logger = (*SpyLogger)(nil)

// Returns a new spy logger that records all calls.
func NewSpyLogger() *SpyLogger {
	return &SpyLogger{store: &entryStore{}}
}

func (l *SpyLogger) record(level Level, msg string, keyVals []any) {
	merged := make([]any, 0, len(l.context)+len(keyVals))
	merged = append(merged, l.context...)
	merged = append(merged, keyVals...)
	l.store.append(Entry{Level: level, Message: msg, KeyVals: merged})
}

func (l *SpyLogger) Debug(msg string, keyVals ...any) {
	l.record(LevelDebug, msg, keyVals)
}

func (l *SpyLogger) Info(msg string, keyVals ...any) {
	l.record(LevelInfo, msg, keyVals)
}

func (l *SpyLogger) Warn(msg string, keyVals ...any) {
	l.record(LevelWarn, msg, keyVals)
}

func (l *SpyLogger) Error(msg string, keyVals ...any) {
	l.record(LevelError, msg, keyVals)
}

// Returns a child spy that shares the same entry store but prepends the
// given context to every subsequent log call.
func (l *SpyLogger) With(keyVals ...any) snx_lib_logging.Logger {
	ctx := make([]any, 0, len(l.context)+len(keyVals))
	ctx = append(ctx, l.context...)
	ctx = append(ctx, keyVals...)
	return &SpyLogger{store: l.store, context: ctx}
}

// Returns a snapshot of all recorded entries across this logger
// and any children produced by With().
func (l *SpyLogger) Entries() []Entry {
	return l.store.snapshot()
}

// Returns the message strings from recorded entries, optionally filtered to
// the given levels. With no arguments, returns all messages.
func (l *SpyLogger) Messages(levels ...Level) []string {
	entries := l.store.snapshot()
	if len(levels) == 0 {
		msgs := make([]string, len(entries))
		for i, e := range entries {
			msgs[i] = e.Message
		}
		return msgs
	}
	allow := make(map[Level]struct{}, len(levels))
	for _, lv := range levels {
		allow[lv] = struct{}{}
	}
	var msgs []string
	for _, e := range entries {
		if _, ok := allow[e.Level]; ok {
			msgs = append(msgs, e.Message)
		}
	}
	return msgs
}

// Returns true if any recorded entry matches the given level and contains
// msgSubstring in its message.
func (l *SpyLogger) HasEntry(level Level, msgSubstring string) bool {
	for _, e := range l.store.snapshot() {
		if e.Level == level && strings.Contains(e.Message, msgSubstring) {
			return true
		}
	}
	return false
}

// Clears all recorded entries.
func (l *SpyLogger) Reset() {
	l.store.reset()
}
