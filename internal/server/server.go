// Package server wires the GitHub client and leaderboard aggregation up to
// an HTTP server: server-rendered pages for browsing, plus a small JSON API
// the filter UI calls into.
package server

import (
	"context"
	"embed"
	"html/template"
	"net/http"
	"time"

	"github.com/jaimekershawbrown/organisation-leaderboard/internal/githubapi"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	gh        *githubapi.Client
	hasToken  bool
	templates *template.Template
	http      *http.Server
}

var templateFuncs = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
}

// New builds the server. hasToken indicates whether a GITHUB_PAT was
// configured; when false, page handlers short-circuit with a setup message
// instead of hitting the GitHub API and surfacing a raw 401.
func New(addr string, gh *githubapi.Client, hasToken bool) (*Server, error) {
	tmpl, err := template.New("").Funcs(templateFuncs).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	s := &Server{gh: gh, hasToken: hasToken, templates: tmpl}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /org/{org}", s.handleOrgDashboard)
	mux.HandleFunc("GET /api/leaderboard", s.handleAPILeaderboard)
	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	s.http = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s, nil
}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
