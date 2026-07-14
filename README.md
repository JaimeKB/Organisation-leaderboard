# Org Contributor Leaderboard

A local web app for seeing who's been shipping across your GitHub organizations. Pick one or more orgs you belong to, pick repos within them, and get a leaderboard of commit activity over the last 3 months.

## How it works

1. **Organizations** — lists the GitHub orgs your token can see; check the ones you care about.
2. **Repositories** — lists repos in the selected orgs; check the ones to include.
3. **Leaderboard** — fetches contributor stats for the selected repos and ranks contributors by commit count over the last 3 months.

## Setup

### 1. Create a GitHub Personal Access Token

This app authenticates as you, using a PAT — there's no OAuth app to register.

1. GitHub → Settings → Developer settings → [Personal access tokens](https://github.com/settings/tokens).
2. Create a token (classic works simplest) with scopes: **`read:org`** and **`repo`**.
   - `repo` is required even for public repos, because the contributor-stats endpoint used for commit counts requires it.
3. **If you're in a private org with SSO enforced**: after creating the token, GitHub will show an "Authorize" / "Enable SSO" action next to it (or under the org's People/Settings page). You must explicitly authorize the token for that org, or the app simply won't see it or its repos — this is a one-click step on GitHub's side, no config needed here.

### 2. Configure

```
cp .env.example .env
# edit .env and set GITHUB_PAT=<your token>
```

### 3. Run

With Docker (recommended — no local Go install needed):

```
make local
```

Then open http://localhost:8080.

Without Docker:

```
go run ./cmd/server
```

## Project layout

```
cmd/server/          entrypoint: config load, wiring, HTTP server
internal/config/      env + label-config (YAML) loading
internal/githubapi/   the only package that talks to go-github; Client interface for testability
internal/metrics/      pluggable per-contributor metric strategy (commit count today)
internal/wizard/       server-side session state for the org -> repo -> leaderboard flow
internal/pipeline/     bounded-concurrency fetch + aggregate across selected repos
internal/httpserver/   HTTP routes, handlers, html/template rendering
web/                   HTML templates and static assets (embedded into the binary)
configs/               labels.yaml (git-ignored; see labels.example.yaml)
```

## Extending

The architecture has two extension points built in from the start:

- **New metrics** (lines added/deleted, PR count, reviews, ...): implement the `metrics.Metric` interface (`internal/metrics/metric.go`) in a new file and add it to the registry (`internal/metrics/registry.go`). The leaderboard automatically gains a column and a `?sort=` option — no pipeline or handler changes needed.
- **Repo labels/groups**: `internal/config/labels.go` already defines a YAML schema (`configs/labels.yaml`) mapping org/repo to labels, with `LabelsFor`/`ReposByLabel`/`SetLabels` helpers, and the repo-selection page already annotates each repo checkbox with its labels if present. Not yet wired up: a "select by label" step that pre-checks repos instead of picking them by hand every run.

## Testing

```
make test   # or: go test ./...
make vet
```

Everything except the real GitHub API client is tested with no network access (fakes/`httptest`). `internal/githubapi`'s tests run against an `httptest.Server` serving canned GitHub-shaped JSON — no test hits the real GitHub API.
