package wizard

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Store persists wizard State across requests within a session.
type Store interface {
	Get(id string) (*State, bool)
	Save(s *State)
	Delete(id string)
}

// MemoryStore is an in-memory, concurrency-safe Store. Appropriate for this
// single-local-user, single-process tool; state does not need to survive a
// restart.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]*State
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]*State)}
}

func (m *MemoryStore) Get(id string) (*State, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.data[id]
	return s, ok
}

func (m *MemoryStore) Save(s *State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[s.ID] = s
}

func (m *MemoryStore) Delete(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, id)
}

// SweepExpired removes sessions older than maxAge (relative to now). Intended
// to be called periodically from a background goroutine so long-running
// processes don't accumulate abandoned sessions.
func (m *MemoryStore) SweepExpired(now time.Time, maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.data {
		if now.Sub(s.CreatedAt) > maxAge {
			delete(m.data, id)
		}
	}
}

// StartSweeper runs SweepExpired on interval until ctx is canceled, removing
// sessions older than maxAge. Intended to be started once from main so a
// long-running process doesn't accumulate abandoned sessions.
func (m *MemoryStore) StartSweeper(ctx context.Context, interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				m.SweepExpired(now, maxAge)
			}
		}
	}()
}

// NewSessionID generates a random session identifier suitable for a cookie
// value.
func NewSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
