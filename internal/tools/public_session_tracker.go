package tools

import (
	"sync"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// PublicSessionTracker records the first time a public (unauthenticated)
// MCP session ID is observed by a tool or resource handler, so get_session
// can report a real createdAt timestamp instead of a misleading zero.
//
// The MCP SDK owns the Mcp-Session-Id lifecycle and does not surface a
// "session opened" hook we can use; this tracker is the thinnest viable
// substitute. Public sessions are ephemeral (they carry no server-side
// state) and the tracker is intentionally in-memory — a process restart
// simply resets first-seen, which is correct semantically.
//
// Authenticated sessions persist their own CreatedAt in session.State and
// bypass this tracker.
type PublicSessionTracker struct {
	mu        sync.RWMutex
	createdAt map[string]int64 // session ID → createdAt unix millis
}

func NewPublicSessionTracker() *PublicSessionTracker {
	return &PublicSessionTracker{createdAt: map[string]int64{}}
}

// Observe records the current time as the createdAt for sessionID if it
// has not been seen before, and returns the (possibly pre-existing)
// createdAt millis for the session.
func (p *PublicSessionTracker) Observe(sessionID string) int64 {
	if p == nil || sessionID == "" {
		return 0
	}
	p.mu.RLock()
	if ts, ok := p.createdAt[sessionID]; ok {
		p.mu.RUnlock()
		return ts
	}
	p.mu.RUnlock()

	now := snx_lib_utils_time.Now().UnixMilli()
	p.mu.Lock()
	defer p.mu.Unlock()
	if ts, ok := p.createdAt[sessionID]; ok {
		return ts
	}
	p.createdAt[sessionID] = now
	return now
}

// Drops the tracked createdAt for sessionID, e.g. when the session
// is closed or upgraded to authenticated (which carries its own CreatedAt).
func (p *PublicSessionTracker) Forget(sessionID string) {
	if p == nil || sessionID == "" {
		return
	}
	p.mu.Lock()
	delete(p.createdAt, sessionID)
	p.mu.Unlock()
}
