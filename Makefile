# ===== Makefile — Neuroscape Match Server =====
# Prérequis : go >= 1.22, docker, golangci-lint (binaires en PATH)

BINARY     := matchserver
IMAGE      := matchserver
REGISTRY   ?= your-registry.amazonaws.com  # ECR URI
AWS_REGION ?= eu-west-3
TAG        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

GOFILES    := $(shell find . -name '*.go' -not -path './vendor/*')
GOFLAGS    := -trimpath
LDFLAGS    := -s -w -X main.version=$(TAG)

.PHONY: help build test lint fmt tidy run clean docker-build docker-push deploy check ci

help: ## Affiche cette aide
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

## --- Développement ---

build: ## Compile le binaire
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/server

run: ## Lance le serveur en local
	go run ./cmd/server

test: ## Tests unitaires + intégration (race detector actif)
	go test -race -count=1 -timeout 120s ./...

cover: ## Couverture HTML dans cover.html
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt: ## Formate + trie les imports
	gofmt -s -w .
	goimports -local matchserver -w .

tidy: ## Nettoie go.mod
	go mod tidy

## --- Qualité / CI ---

lint: ## go vet + golangci-lint
	go vet ./...
	golangci-lint run

govuln: ## Scan CVE des dépendances
	govulncheck ./...

check: build test lint ## Tout ce que la CI doit bloquer — cible de référence

## --- Docker / AWS ---

docker-build: ## Image multi-stage
	docker build -t $(REGISTRY)/$(IMAGE):$(TAG) .

docker-push: docker-build ## Push vers ECR
	aws ecr get-login-password --region $(AWS_REGION) | docker login --username AWS --password-stdin $(REGISTRY)
	docker push $(REGISTRY)/$(IMAGE):$(TAG)

deploy: docker-push ## Roulement ECS (rollout implicite blue/green)
	aws ecs update-service \
	  --cluster matchserver --service prod \
	  --force-new-deployment --region $(AWS_REGION)

clean:
	rm -rf bin/ coverage.out coverage.html
