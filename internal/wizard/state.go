// Package wizard holds the per-user, per-session state of the multi-step
// org -> repo -> leaderboard flow. State lives server-side, keyed by a
// cookie, since this is a single-local-user tool and the state (full repo
// lists) is too large to comfortably round-trip in a client-side cookie.
package wizard

import (
	"time"

	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/metrics"
	"github.com/jaimekb/org-leaderboard/internal/pipeline"
)

// Step identifies where a session is in the wizard flow. Handlers are
// server-authoritative about step order: a request that's ahead of the
// session's current step is redirected back rather than trusted.
type Step int

const (
	StepOrgSelect Step = iota
	StepRepoSelect
	StepRunning
	StepDone
)

// State is one user's progress through the wizard.
type State struct {
	ID   string
	Step Step

	SelectedOrgs []string

	// AvailableRepos caches the repos fetched for each selected org, keyed
	// by org login, so the repo-select page doesn't refetch on back/forward.
	AvailableRepos map[string][]githubapi.Repo
	// SelectedRepos holds the chosen repo names per org.
	SelectedRepos map[string][]string

	// Running is true while a pipeline.Run for this session is in flight, so
	// the run handler doesn't kick off a second concurrent run on page refresh.
	Running      bool
	Result       *metrics.LeaderboardResult
	RepoStatuses []pipeline.RepoStatus
	Err          error

	CreatedAt time.Time
}

// NewState creates a fresh session at the first wizard step.
func NewState(id string, now time.Time) *State {
	return &State{
		ID:             id,
		Step:           StepOrgSelect,
		AvailableRepos: map[string][]githubapi.Repo{},
		SelectedRepos:  map[string][]string{},
		CreatedAt:      now,
	}
}
