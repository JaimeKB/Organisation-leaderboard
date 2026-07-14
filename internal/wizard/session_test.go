package wizard

import (
	"testing"
	"time"
)

func TestMemoryStore_SaveGetDelete(t *testing.T) {
	store := NewMemoryStore()

	if _, ok := store.Get("missing"); ok {
		t.Fatalf("Get(missing) ok = true, want false")
	}

	s := NewState("session-1", time.Now())
	s.SelectedOrgs = []string{"my-org"}
	store.Save(s)

	got, ok := store.Get("session-1")
	if !ok {
		t.Fatalf("Get(session-1) ok = false, want true")
	}
	if len(got.SelectedOrgs) != 1 || got.SelectedOrgs[0] != "my-org" {
		t.Errorf("Get(session-1).SelectedOrgs = %v, want [my-org]", got.SelectedOrgs)
	}

	store.Delete("session-1")
	if _, ok := store.Get("session-1"); ok {
		t.Fatalf("Get(session-1) after Delete ok = true, want false")
	}
}

func TestMemoryStore_SweepExpired(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()

	fresh := NewState("fresh", now)
	stale := NewState("stale", now.Add(-3*time.Hour))
	store.Save(fresh)
	store.Save(stale)

	store.SweepExpired(now, 2*time.Hour)

	if _, ok := store.Get("fresh"); !ok {
		t.Errorf("fresh session was swept, want kept")
	}
	if _, ok := store.Get("stale"); ok {
		t.Errorf("stale session was kept, want swept")
	}
}

func TestNewSessionID_UniqueAndNonEmpty(t *testing.T) {
	a, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID() error = %v", err)
	}
	b, err := NewSessionID()
	if err != nil {
		t.Fatalf("NewSessionID() error = %v", err)
	}
	if a == "" || b == "" {
		t.Fatalf("NewSessionID() returned empty string")
	}
	if a == b {
		t.Errorf("NewSessionID() returned duplicate IDs: %q", a)
	}
}
