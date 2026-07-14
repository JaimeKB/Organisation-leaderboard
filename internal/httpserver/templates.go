package httpserver

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/jaimekb/org-leaderboard/web"
)

var pageNames = []string{"orgs", "repos", "loading", "leaderboard"}

var templateFuncs = template.FuncMap{
	"inc": func(i int) int { return i + 1 },
}

// templateSet holds one parsed {layout + page} template per page, avoiding
// the "content" template-name collision that would occur if every page's
// {{define "content"}} block were parsed into a single shared template set.
type templateSet map[string]*template.Template

func loadTemplates() (templateSet, error) {
	ts := make(templateSet, len(pageNames))
	for _, name := range pageNames {
		t, err := template.New("layout.html").Funcs(templateFuncs).ParseFS(
			web.TemplatesFS, "templates/layout.html", "templates/"+name+".html",
		)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", name, err)
		}
		ts[name] = t
	}
	return ts, nil
}

// baseData is embedded by every page's view-data struct to drive the layout's
// step indicator.
type baseData struct {
	Step string
}

func (ts templateSet) render(w http.ResponseWriter, page string, data any) {
	t, ok := ts[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// renderFragment executes a named block within page's template set without
// the surrounding layout — used for the htmx-polled loading fragment.
func (ts templateSet) renderFragment(w http.ResponseWriter, page, blockName string, data any) {
	t, ok := ts[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, blockName, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
