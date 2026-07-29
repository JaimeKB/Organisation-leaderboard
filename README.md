# Organisation Leaderboard

A local web app that shows the GitHub organisations and repositories you have
access to, and a commit leaderboard per org — filterable by repo, member, and
date range.

## Setup

1. Create a **classic** Personal Access Token at
   [github.com/settings/tokens](https://github.com/settings/tokens) with the
   `read:org` and `repo` scopes. This is created under your own account —
   you don't need to own or administer an org, just be a member of it.
   (A small number of orgs restrict even classic PAT access; if every org
   comes back empty, check with your org's admin.)
2. Copy `.env.example` to `.env` and paste the token in as `GITHUB_PAT`.

## Run

```sh
make local
```

This builds the Docker image, starts the server, waits for it to become
healthy, and opens `http://localhost:8080` in your browser automatically.
Press **Ctrl+C** to stop — it gracefully shuts down the server and tears
down the container/network.

If `.env` is missing, `make local` creates it from `.env.example` and asks
you to fill in `GITHUB_PAT` before rerunning.

`make down` tears down the containers manually, if needed.

## How the leaderboard is computed

For the selected org, repos, and date range, the app fetches commits on each
repo's default branch (via the GitHub commits API, paginated) and tallies
them per author. Commits not linked to a GitHub account fall back to the raw
git author name/email. Default date range is the last 90 days.

## Extending the filters

Everything the leaderboard understands lives in one place:
`internal/leaderboard/query.go` (`Query` struct) and
`internal/leaderboard/leaderboard.go` (`Compute`). To add a new filter
dimension (e.g. branch, file path, exclude bots, minimum commit count):

1. Add a field to `Query`.
2. Add matching logic in `Compute`/`addCommits`.
3. Add a query param for it in `internal/server/handlers_api.go`
   (`handleAPILeaderboard`).
4. Add a form control for it in `internal/server/templates/org.html` and
   read it in `internal/server/static/app.js`.

The JSON API (`GET /api/leaderboard?org=...&repo=...&user=...&since=...&until=...`)
is query-param driven, so new filters are additive and backward compatible.

## Project layout

```
cmd/server/            entrypoint, graceful shutdown
internal/config/       env config loading
internal/githubapi/    GitHub REST API client (orgs, repos, commits)
internal/leaderboard/  filterable commit aggregation (pure, unit tested)
internal/server/       HTTP routes, HTML templates, static JS/CSS
```

## Development

```sh
go build ./...
go vet ./...
go test ./...
```
