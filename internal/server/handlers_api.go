package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jaimekershawbrown/organisation-leaderboard/internal/leaderboard"
)

const dateLayout = "2006-01-02"

// handleAPILeaderboard is the JSON endpoint the filter UI calls. Query
// params: org (required), repo/user (repeatable, e.g. ?repo=a&repo=b),
// since/until (YYYY-MM-DD). Adding a new filter dimension means adding a
// query param here and a matching field on leaderboard.Query.
func (s *Server) handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	org := q.Get("org")
	if org == "" {
		writeJSONError(w, http.StatusBadRequest, "org is required")
		return
	}

	query := leaderboard.Query{
		Org:   org,
		Repos: q["repo"],
		Users: q["user"],
	}

	if v := q.Get("since"); v != "" {
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid since date, want YYYY-MM-DD")
			return
		}
		query.Since = t
	}
	if v := q.Get("until"); v != "" {
		t, err := time.Parse(dateLayout, v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid until date, want YYYY-MM-DD")
			return
		}
		// Make "until" inclusive of the whole day the user picked.
		query.Until = t.Add(24*time.Hour - time.Nanosecond)
	}

	rows, err := leaderboard.Compute(r.Context(), s.gh, query)
	if err != nil {
		log.Printf("api leaderboard: %v", err)
		writeJSONError(w, http.StatusBadGateway, friendlyGitHubError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rows); err != nil {
		log.Printf("encode leaderboard response: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
