package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/jaimekb/org-leaderboard/internal/metrics"
	"github.com/jaimekb/org-leaderboard/internal/pipeline"
	"github.com/jaimekb/org-leaderboard/internal/wizard"
)

// pipelineTimeout bounds a single leaderboard run so a stuck GitHub call
// can't hang a session forever.
const pipelineTimeout = 5 * time.Minute

func (s *Server) ShowLeaderboardRun(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())
	if state.Step < wizard.StepRunning {
		http.Redirect(w, r, "/orgs", http.StatusSeeOther)
		return
	}

	if state.Result == nil && !state.Running {
		next := *state
		next.Running = true
		s.store.Save(&next)
		go s.runPipeline(next.ID)
	}

	s.templates.render(w, "loading", baseData{Step: "leaderboard"})
}

// runPipeline executes the fetch/aggregate pipeline for session id in the
// background and persists the result. It re-fetches the session immediately
// before saving so it doesn't clobber concurrent unrelated field changes.
func (s *Server) runPipeline(id string) {
	state, ok := s.store.Get(id)
	if !ok {
		return
	}

	var repos []pipeline.RepoRef
	for org, names := range state.SelectedRepos {
		for _, name := range names {
			repos = append(repos, pipeline.RepoRef{Owner: org, Name: name})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), pipelineTimeout)
	defer cancel()

	result, err := pipeline.Run(ctx, s.client, repos, metrics.Enabled(), pipeline.Options{
		Concurrency: s.concurrency,
		Window:      s.window,
	})

	latest, ok := s.store.Get(id)
	if !ok {
		return
	}
	next := *latest
	next.Running = false
	next.Step = wizard.StepDone
	if err != nil {
		next.Err = err
	} else {
		lb := result.Leaderboard
		next.Result = &lb
		next.RepoStatuses = result.RepoStatuses
	}
	s.store.Save(&next)
}

func (s *Server) LeaderboardStatus(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())
	if state.Result != nil || state.Err != nil {
		w.Header().Set("HX-Redirect", "/leaderboard")
		w.WriteHeader(http.StatusOK)
		return
	}
	s.templates.renderFragment(w, "loading", "status-pending", nil)
}

type leaderboardViewData struct {
	baseData
	Metrics      []metrics.Metric
	Rows         []metrics.ContributorResult
	SortKey      string
	PendingRepos []string
	FailedRepos  []string
}

func (s *Server) ShowLeaderboard(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())
	if state.Err != nil {
		http.Error(w, "leaderboard run failed: "+state.Err.Error(), http.StatusInternalServerError)
		return
	}
	if state.Result == nil {
		http.Redirect(w, r, "/leaderboard/run", http.StatusSeeOther)
		return
	}

	sortKey := r.URL.Query().Get("sort")
	if _, ok := metrics.Lookup(sortKey); !ok {
		if len(state.Result.Metrics) > 0 {
			sortKey = state.Result.Metrics[0].Key()
		}
	}

	var pending, failed []string
	for _, rs := range state.RepoStatuses {
		switch {
		case rs.Pending:
			pending = append(pending, rs.Repo.FullName())
		case rs.Err != nil:
			failed = append(failed, rs.Repo.FullName())
		}
	}

	data := leaderboardViewData{
		baseData:     baseData{Step: "leaderboard"},
		Metrics:      state.Result.Metrics,
		Rows:         state.Result.SortedBy(sortKey),
		SortKey:      sortKey,
		PendingRepos: pending,
		FailedRepos:  failed,
	}
	s.templates.render(w, "leaderboard", data)
}
