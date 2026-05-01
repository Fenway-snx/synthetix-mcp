package auth

import (
	"sync"
	"time"

	snx_lib_auth "github.com/Fenway-snx/synthetix-mcp/internal/lib/auth"
)

// Single-process nonce store for the standalone image. Per-address
// keyspace keeps memory bounded per wallet.
type memoryNonceStore struct {
	mu   sync.Mutex
	seen map[string]map[snx_lib_auth.Nonce]time.Time
}

func newMemoryNonceStore() snx_lib_auth.NonceStore {
	return &memoryNonceStore{seen: map[string]map[snx_lib_auth.Nonce]time.Time{}}
}

func (s *memoryNonceStore) IsNonceUsed(address string, nonce snx_lib_auth.Nonce) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.seen[address]; ok {
		if _, used := m[nonce]; used {
			return true, nil
		}
	}
	return false, nil
}

func (s *memoryNonceStore) ReserveNonce(address string, nonce snx_lib_auth.Nonce) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.seen[address]
	if !ok {
		m = map[snx_lib_auth.Nonce]time.Time{}
		s.seen[address] = m
	}
	if _, used := m[nonce]; used {
		return false, nil
	}
	m[nonce] = time.Now()
	return true, nil
}

// Drops entries older than maxAge so per-address maps stay bounded.
func (s *memoryNonceStore) CleanupExpiredNonces(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	s.mu.Lock()
	defer s.mu.Unlock()
	for addr, m := range s.seen {
		for n, at := range m {
			if at.Before(cutoff) {
				delete(m, n)
			}
		}
		if len(m) == 0 {
			delete(s.seen, addr)
		}
	}
	return nil
}
