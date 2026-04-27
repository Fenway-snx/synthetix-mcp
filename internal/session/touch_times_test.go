package session

import (
	"testing"
	"time"
)

func TestApplyTouchTimesUsesSingleNowBase(t *testing.T) {
	now := time.Unix(1_700_000_000, 123_000_000).UTC()
	ttl := 30 * time.Minute
	state := &State{}

	ApplyTouchTimes(state, ttl, now)

	if state.LastActivityAt != now.UnixMilli() {
		t.Fatalf("expected LastActivityAt %d, got %d", now.UnixMilli(), state.LastActivityAt)
	}
	if state.ExpiresAt != now.Add(ttl).UnixMilli() {
		t.Fatalf("expected ExpiresAt %d, got %d", now.Add(ttl).UnixMilli(), state.ExpiresAt)
	}
	if state.ExpiresAt-state.LastActivityAt != ttl.Milliseconds() {
		t.Fatalf("expected expires delta %dms, got %dms", ttl.Milliseconds(), state.ExpiresAt-state.LastActivityAt)
	}
}
