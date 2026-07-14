package httpserver

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/jaimekb/org-leaderboard/internal/githubapi"
	"github.com/jaimekb/org-leaderboard/internal/wizard"
)

type repoRow struct {
	Org      string
	Name     string
	Private  bool
	Selected bool
	Labels   []string
}

type repoOrgGroup struct {
	Org   string
	Repos []repoRow
}

type repoViewData struct {
	baseData
	Groups []repoOrgGroup
	Err    string
}

func (s *Server) ShowRepos(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())
	if state.Step < wizard.StepRepoSelect {
		http.Redirect(w, r, "/orgs", http.StatusSeeOther)
		return
	}

	// Work on a copy: fetch-and-cache repos for any selected org not already
	// in AvailableRepos, then persist once via store.Save (copy-on-write —
	// see stateFromContext's contract).
	next := *state
	availableRepos := make(map[string][]githubapi.Repo, len(next.AvailableRepos))
	for k, v := range next.AvailableRepos {
		availableRepos[k] = v
	}
	next.AvailableRepos = availableRepos

	changed := false
	for _, org := range next.SelectedOrgs {
		if _, ok := availableRepos[org]; ok {
			continue
		}
		repos, err := s.client.ListOrgRepos(r.Context(), org)
		if err != nil {
			s.templates.render(w, "repos", repoViewData{
				baseData: baseData{Step: "repos"},
				Err:      "failed to load repositories for " + org + ": " + err.Error(),
			})
			return
		}
		availableRepos[org] = repos
		changed = true
	}

	if changed {
		s.store.Save(&next)
		state = &next
	}

	selected := make(map[string]map[string]bool, len(state.SelectedRepos))
	for org, names := range state.SelectedRepos {
		set := make(map[string]bool, len(names))
		for _, n := range names {
			set[n] = true
		}
		selected[org] = set
	}

	groups := make([]repoOrgGroup, 0, len(state.SelectedOrgs))
	for _, org := range state.SelectedOrgs {
		repos := state.AvailableRepos[org]
		rows := make([]repoRow, 0, len(repos))
		for _, repo := range repos {
			rows = append(rows, repoRow{
				Org:      org,
				Name:     repo.Name,
				Private:  repo.Private,
				Selected: selected[org][repo.Name],
				Labels:   s.labelConfig.LabelsFor(org, repo.Name),
			})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
		groups = append(groups, repoOrgGroup{Org: org, Repos: rows})
	}

	data := repoViewData{baseData: baseData{Step: "repos"}, Groups: groups, Err: r.URL.Query().Get("err")}
	s.templates.render(w, "repos", data)
}

func (s *Server) SubmitRepos(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	selectedRepos := make(map[string][]string)
	for _, val := range r.Form["repo"] {
		org, name, ok := strings.Cut(val, "|")
		if !ok {
			continue
		}
		selectedRepos[org] = append(selectedRepos[org], name)
	}
	if len(selectedRepos) == 0 {
		http.Redirect(w, r, "/repos?err="+url.QueryEscape("select at least one repository"), http.StatusSeeOther)
		return
	}

	next := *state
	next.SelectedRepos = selectedRepos
	next.Step = wizard.StepRunning
	next.Running = false
	next.Result = nil
	next.RepoStatuses = nil
	next.Err = nil
	s.store.Save(&next)

	http.Redirect(w, r, "/leaderboard/run", http.StatusSeeOther)
}
