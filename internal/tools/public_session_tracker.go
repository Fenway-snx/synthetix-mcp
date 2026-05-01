package tools

import (
	"sync"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
)

// Tracks first-seen times for unauthenticated sessions.
// Authenticated sessions persist their own creation timestamps.
type PublicSessionTracker struct {
	mu        sync.RWMutex
	createdAt map[string]int64 // session ID → createdAt unix millis
}

func NewPublicSessionTracker() *PublicSessionTracker {
	return &PublicSessionTracker{createdAt: map[string]int64{}}
}

// Records and returns the first-seen timestamp for a session.
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
