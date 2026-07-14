APP_NAME := org-leaderboard
IMAGE    := $(APP_NAME):local
PORT     ?= 8080

.PHONY: build run test vet docker-build local

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

vet:
	go vet ./...

docker-build:
	docker build -t $(IMAGE) .

## local builds the Docker image and runs it, reading GITHUB_PAT from .env
## (copy .env.example to .env and fill in a token first) and publishing the
## app on http://localhost:$(PORT). configs/ is mounted so the label config
## file (used by a future feature) persists across runs.
local: docker-build
	mkdir -p configs
	docker run --rm -it \
		--env-file .env \
		-p $(PORT):8080 \
		-v "$(CURDIR)/configs:/app/configs" \
		$(IMAGE)
