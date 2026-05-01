package session

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyTouchTimesUpdatesActivityAndExpiry(t *testing.T) {
	state := &State{}
	now := time.Unix(1704067200, 0).UTC()

	ApplyTouchTimes(state, 5*time.Minute, now)

	if state.LastActivityAt != now.UnixMilli() {
		t.Fatalf("expected last activity %d, got %d", now.UnixMilli(), state.LastActivityAt)
	}
	if state.ExpiresAt != now.Add(5*time.Minute).UnixMilli() {
		t.Fatalf("expected expiry %d, got %d", now.Add(5*time.Minute).UnixMilli(), state.ExpiresAt)
	}
}

func TestApplyTouchTimesIgnoresNilState(t *testing.T) {
	ApplyTouchTimes(nil, time.Minute, time.Unix(1704067200, 0).UTC())
}

// Confirms legacy numeric subaccount IDs still load during rolling deploys.
func TestStateUnmarshalAcceptsLegacyBareNumberSubAccountID(t *testing.T) {
	legacy := []byte(`{"authMode":"authenticated","createdAt":1,"expiresAt":2,"lastActivityAt":3,"subAccountId":77,"walletAddress":"0xabc"}`)

	var state State
	if err := json.Unmarshal(legacy, &state); err != nil {
		t.Fatalf("unmarshal legacy session: %v", err)
	}
	if state.SubAccountID != 77 {
		t.Fatalf("expected subAccountId 77, got %d", state.SubAccountID)
	}
	if state.WalletAddress != "0xabc" {
		t.Fatalf("expected wallet preserved, got %q", state.WalletAddress)
	}
}

// The current wire format encodes subAccountId as a JSON string to
// survive Lua/cjson round-trips at int64 precision. Both decoding the
// string form and round-tripping it must succeed without loss.
func TestStateRoundTripPreservesLargeSubAccountID(t *testing.T) {
	const huge int64 = 9007199254740993 // 2^53 + 1, lossy as a float64.
	original := State{
		AuthMode:       AuthModeAuthenticated,
		CreatedAt:      1,
		ExpiresAt:      2,
		LastActivityAt: 3,
		SubAccountID:   huge,
		WalletAddress:  "0xabc",
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"subAccountId":"9007199254740993"`) {
		t.Fatalf("expected subAccountId encoded as JSON string, got %s", encoded)
	}

	var decoded State
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.SubAccountID != huge {
		t.Fatalf("expected %d after round-trip, got %d", huge, decoded.SubAccountID)
	}
}

func TestStateUnmarshalRejectsMalformedSubAccountID(t *testing.T) {
	bad := []byte(`{"subAccountId":"not-a-number"}`)
	var state State
	if err := json.Unmarshal(bad, &state); err == nil {
		t.Fatalf("expected error decoding malformed subAccountId, got state %+v", state)
	}
}

func TestStateUnmarshalToleratesMissingSubAccountID(t *testing.T) {
	noSub := []byte(`{"authMode":"public","createdAt":1}`)
	var state State
	if err := json.Unmarshal(noSub, &state); err != nil {
		t.Fatalf("unmarshal without subAccountId: %v", err)
	}
	if state.SubAccountID != 0 {
		t.Fatalf("expected zero subAccountId when key absent, got %d", state.SubAccountID)
	}
	if state.AuthMode != AuthModePublic {
		t.Fatalf("expected other fields preserved, got %+v", state)
	}
}

func TestFileStorePersistsSessionAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	ctx := context.Background()
	ttl := time.Hour
	original := &State{
		AuthMode:       AuthModeAuthenticated,
		CreatedAt:      1,
		ExpiresAt:      2,
		LastActivityAt: 3,
		SubAccountID:   77,
		WalletAddress:  "0xabc",
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	if err := store.Save(ctx, "session-1", original, ttl); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("reload file store: %v", err)
	}
	got, err := reloaded.Get(ctx, "session-1")
	if err != nil {
		t.Fatalf("get reloaded session: %v", err)
	}
	if got.SubAccountID != 77 || got.WalletAddress != "0xabc" || got.AuthMode != AuthModeAuthenticated {
		t.Fatalf("unexpected reloaded session: %+v", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat store: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 permissions, got %v", info.Mode().Perm())
	}
}

func TestFileStoreDropsExpiredSessionsOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	expired := persistedSessions{
		Version: 1,
		Sessions: map[string]persistedEntry{
			"old": {
				State: State{
					AuthMode:      AuthModeAuthenticated,
					SubAccountID:  77,
					WalletAddress: "0xabc",
				},
				ExpiresAt: time.Now().Add(-time.Hour).UnixMilli(),
			},
		},
	}
	body, err := json.Marshal(expired)
	if err != nil {
		t.Fatalf("marshal expired store: %v", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write expired store: %v", err)
	}

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	_, err = store.Get(context.Background(), "old")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected expired session not found, got %v", err)
	}
	count, err := store.Count(context.Background())
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected expired session to be reaped, got count %d", count)
	}
}
