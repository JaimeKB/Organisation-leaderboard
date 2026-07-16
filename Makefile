SHELL := /bin/bash
.ONESHELL:

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	OPEN_CMD := open
else ifeq ($(UNAME_S),Linux)
	OPEN_CMD := xdg-open
else
	OPEN_CMD := start
endif

-include .env
export

.PHONY: local
local: check-env
	@trap 'docker compose down' EXIT
	docker compose up --build -d
	echo "Waiting for the server to become healthy..."
	until curl -sf "http://localhost:$${PORT:-8080}/healthz" >/dev/null 2>&1; do sleep 0.3; done
	$(OPEN_CMD) "http://localhost:$${PORT:-8080}" >/dev/null 2>&1 || true
	echo "Serving at http://localhost:$${PORT:-8080} — press Ctrl+C to stop."
	docker compose logs -f

.PHONY: down
down:
	docker compose down

.PHONY: check-env
check-env:
	@if [ ! -f .env ]; then \
		echo "No .env found — copying .env.example. Add your GITHUB_PAT before running 'make local' again."; \
		cp .env.example .env; \
		exit 1; \
	fi
