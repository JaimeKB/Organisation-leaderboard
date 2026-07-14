package httpserver

import (
	"net/http"
	"net/url"
	"sort"

	"github.com/jaimekb/org-leaderboard/internal/wizard"
)

type orgRow struct {
	Login    string
	Selected bool
}

type orgViewData struct {
	baseData
	Orgs []orgRow
	Err  string
}

func (s *Server) ShowOrgs(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())

	orgs, err := s.client.ListUserOrgs(r.Context())
	if err != nil {
		s.templates.render(w, "orgs", orgViewData{
			baseData: baseData{Step: "orgs"},
			Err:      "failed to load organizations: " + err.Error(),
		})
		return
	}

	selected := make(map[string]bool, len(state.SelectedOrgs))
	for _, o := range state.SelectedOrgs {
		selected[o] = true
	}

	rows := make([]orgRow, 0, len(orgs))
	for _, o := range orgs {
		rows = append(rows, orgRow{Login: o.Login, Selected: selected[o.Login]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Login < rows[j].Login })

	data := orgViewData{baseData: baseData{Step: "orgs"}, Orgs: rows, Err: r.URL.Query().Get("err")}
	s.templates.render(w, "orgs", data)
}

func (s *Server) SubmitOrgs(w http.ResponseWriter, r *http.Request) {
	state := stateFromContext(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	selected := r.Form["org"]
	if len(selected) == 0 {
		http.Redirect(w, r, "/orgs?err="+url.QueryEscape("select at least one organization"), http.StatusSeeOther)
		return
	}

	next := *state
	next.SelectedOrgs = selected
	next.Step = wizard.StepRepoSelect
	s.store.Save(&next)

	http.Redirect(w, r, "/repos", http.StatusSeeOther)
}
