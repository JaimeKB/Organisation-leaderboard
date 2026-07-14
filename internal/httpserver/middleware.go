package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/jaimekb/org-leaderboard/internal/wizard"
)

type contextKey int

const stateContextKey contextKey = iota

const sessionCookieName = "session_id"

// withSession ensures every request has a wizard session: it reads the
// session cookie, loads the matching State from store, or creates a fresh
// one (and sets the cookie) if missing/unknown. The State is injected into
// the request context for handlers to read via stateFromContext.
func withSession(store wizard.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var state *wizard.State

		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			if s, ok := store.Get(cookie.Value); ok {
				state = s
			}
		}

		if state == nil {
			id, err := wizard.NewSessionID()
			if err != nil {
				http.Error(w, "failed to create session", http.StatusInternalServerError)
				return
			}
			state = wizard.NewState(id, time.Now())
			store.Save(state)
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    id,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		ctx := context.WithValue(r.Context(), stateContextKey, state)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// stateFromContext returns the current request's wizard session state.
// Callers must treat the returned *State as read-only: to persist a change,
// build a modified copy and pass it to Store.Save (see handlers_*.go).
func stateFromContext(ctx context.Context) *wizard.State {
	state, _ := ctx.Value(stateContextKey).(*wizard.State)
	return state
}
