package session

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	snx_lib_utils_time "github.com/Fenway-snx/synthetix-mcp/internal/lib/utils/time"
	"github.com/Fenway-snx/synthetix-mcp/internal/guardrails"
)

var ErrSessionNotFound = errors.New("session not found")

type AuthMode string

const (
	AuthModeAuthenticated AuthMode = "authenticated"
	AuthModePublic        AuthMode = "public"
)

type State struct {
	AgentGuardrails *guardrails.Config `json:"agentGuardrails,omitempty"`
	AuthMode        AuthMode           `json:"authMode"`
	CreatedAt       int64              `json:"createdAt"`
	ExpiresAt       int64              `json:"expiresAt"`
	LastActivityAt  int64              `json:"lastActivityAt"`
	// SubAccountId is encoded as a JSON string (not number) because
	// Synthetix composite subaccount IDs can exceed 2^53 and would lose
	// precision if rendered as IEEE-754 doubles by any JavaScript-side
	// tooling (including the MCP client SDK, which decodes numbers via
	// JSON.parse by default).
	SubAccountID  int64  `json:"subAccountId,string"`
	WalletAddress string `json:"walletAddress"`
}

// Accepts subAccountId as either a JSON string (current wire form) or a
// bare JSON number so legacy payloads still decode cleanly.
func (s *State) UnmarshalJSON(data []byte) error {
	type alias State
	aux := struct {
		SubAccountID json.RawMessage `json:"subAccountId"`
		*alias
	}{
		alias: (*alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	raw := bytes.TrimSpace(aux.SubAccountID)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	if raw[0] == '"' {
		var str string
		if err := json.Unmarshal(raw, &str); err != nil {
			return fmt.Errorf("decode subAccountId string: %w", err)
		}
		n, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return fmt.Errorf("parse subAccountId %q: %w", str, err)
		}
		s.SubAccountID = n
		return nil
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err != nil {
		return fmt.Errorf("decode subAccountId number: %w", err)
	}
	s.SubAccountID = n
	return nil
}

type Store interface {
	Delete(ctx context.Context, sessionID string) error
	DeleteIfExists(ctx context.Context, sessionID string) (bool, error)
	Get(ctx context.Context, sessionID string) (*State, error)
	Save(ctx context.Context, sessionID string, state *State, ttl time.Duration) error
	Touch(ctx context.Context, sessionID string, ttl time.Duration) error
}

// In-process, TTL-aware session store for the standalone image.
// Concurrent-safe.
type MemoryStore struct {
	mu    sync.Mutex
	items map[string]memoryEntry
}

type memoryEntry struct {
	state     State
	expiresAt time.Time
}

type persistedSessions struct {
	Version  int                       `json:"version"`
	Sessions map[string]persistedEntry `json:"sessions"`
}

type persistedEntry struct {
	State     State `json:"state"`
	ExpiresAt int64 `json:"expiresAt"`
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: map[string]memoryEntry{}}
}

func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, fmt.Errorf("session store path is required")
	}
	store := &FileStore{
		path:  path,
		items: map[string]memoryEntry{},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *MemoryStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, sessionID)
	return nil
}

// Returns true only if a key was actually removed so callers can avoid
// double-cleanup races.
func (s *MemoryStore) DeleteIfExists(_ context.Context, sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[sessionID]
	if ok {
		delete(s.items, sessionID)
	}
	return ok, nil
}

// Live (non-expired) entry count.
func (s *MemoryStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := snx_lib_utils_time.Now()
	s.reapLocked(now)
	return len(s.items), nil
}

func (s *MemoryStore) Get(_ context.Context, sessionID string) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if snx_lib_utils_time.Now().After(entry.expiresAt) {
		delete(s.items, sessionID)
		return nil, ErrSessionNotFound
	}
	copy := entry.state
	return &copy, nil
}

func (s *MemoryStore) Save(_ context.Context, sessionID string, state *State, ttl time.Duration) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[sessionID] = memoryEntry{
		state:     *state,
		expiresAt: snx_lib_utils_time.Now().Add(ttl),
	}
	return nil
}

func (s *MemoryStore) Touch(_ context.Context, sessionID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	now := snx_lib_utils_time.Now()
	if now.After(entry.expiresAt) {
		delete(s.items, sessionID)
		return ErrSessionNotFound
	}
	ApplyTouchTimes(&entry.state, ttl, now)
	entry.expiresAt = now.Add(ttl)
	s.items[sessionID] = entry
	return nil
}

// Lazy eviction of expired entries; no background reaper so the
// store has no lifecycle to tear down.
func (s *MemoryStore) reapLocked(now time.Time) {
	for id, entry := range s.items {
		if now.After(entry.expiresAt) {
			delete(s.items, id)
		}
	}
}

// FileStore is the same TTL-aware store as MemoryStore, persisted as JSON
// after each mutation. It stores only MCP session bindings/guardrails; it does
// not store private keys, signatures, or per-request nonces.
type FileStore struct {
	mu    sync.Mutex
	path  string
	items map[string]memoryEntry
}

func (s *FileStore) Delete(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, sessionID)
	return s.persistLocked()
}

func (s *FileStore) DeleteIfExists(_ context.Context, sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[sessionID]
	if ok {
		delete(s.items, sessionID)
		if err := s.persistLocked(); err != nil {
			return false, err
		}
	}
	return ok, nil
}

func (s *FileStore) Count(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := snx_lib_utils_time.Now()
	if s.reapLocked(now) {
		if err := s.persistLocked(); err != nil {
			return 0, err
		}
	}
	return len(s.items), nil
}

func (s *FileStore) Get(_ context.Context, sessionID string) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	if snx_lib_utils_time.Now().After(entry.expiresAt) {
		delete(s.items, sessionID)
		_ = s.persistLocked()
		return nil, ErrSessionNotFound
	}
	copy := entry.state
	return &copy, nil
}

func (s *FileStore) Save(_ context.Context, sessionID string, state *State, ttl time.Duration) error {
	if state == nil {
		return fmt.Errorf("session state is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[sessionID] = memoryEntry{
		state:     *state,
		expiresAt: snx_lib_utils_time.Now().Add(ttl),
	}
	return s.persistLocked()
}

func (s *FileStore) Touch(_ context.Context, sessionID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.items[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	now := snx_lib_utils_time.Now()
	if now.After(entry.expiresAt) {
		delete(s.items, sessionID)
		_ = s.persistLocked()
		return ErrSessionNotFound
	}
	ApplyTouchTimes(&entry.state, ttl, now)
	entry.expiresAt = now.Add(ttl)
	s.items[sessionID] = entry
	return s.persistLocked()
}

func (s *FileStore) Path() string {
	return s.path
}

func (s *FileStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read session store %s: %w", s.path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var persisted persistedSessions
	if err := json.Unmarshal(raw, &persisted); err != nil {
		return fmt.Errorf("decode session store %s: %w", s.path, err)
	}
	if persisted.Sessions != nil {
		s.items = map[string]memoryEntry{}
		for id, entry := range persisted.Sessions {
			s.items[id] = memoryEntry{
				state:     entry.State,
				expiresAt: time.UnixMilli(entry.ExpiresAt),
			}
		}
	}
	now := snx_lib_utils_time.Now()
	if s.reapLocked(now) {
		if err := s.persistLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create session store directory: %w", err)
	}
	sessions := make(map[string]persistedEntry, len(s.items))
	for id, entry := range s.items {
		sessions[id] = persistedEntry{
			State:     entry.state,
			ExpiresAt: entry.expiresAt.UnixMilli(),
		}
	}
	body, err := json.MarshalIndent(persistedSessions{
		Version:  1,
		Sessions: sessions,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session store: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".sessions-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary session store: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary session store: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary session store: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary session store: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace session store: %w", err)
	}
	return nil
}

func (s *FileStore) reapLocked(now time.Time) bool {
	changed := false
	for id, entry := range s.items {
		if now.After(entry.expiresAt) {
			delete(s.items, id)
			changed = true
		}
	}
	return changed
}

// ApplyTouchTimes updates state timestamps to reflect a TTL bump.
// Used both by MemoryStore.Touch and by callers that already hold a
// freshly loaded *State and want to reflect the TTL bump locally
// without an extra Get.
func ApplyTouchTimes(state *State, ttl time.Duration, now time.Time) {
	if state == nil {
		return
	}
	state.LastActivityAt = now.UnixMilli()
	state.ExpiresAt = now.Add(ttl).UnixMilli()
}
