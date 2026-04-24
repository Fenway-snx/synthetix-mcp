package session

import (
	"encoding/json"
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

// Sessions written before SubAccountID was tagged ,string contain
// subAccountId as a bare JSON number. Loading those after the change
// must succeed; otherwise a rolling deploy would force every connected
// trader to re-authenticate.
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
