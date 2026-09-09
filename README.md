# WNDRTECH
Fan project for a Neuroscape online client à la Talishar/Jinteki

# WNDRTECH Match Server

An online match server for the [Neuroscape TCG](https://www.neuroscapetcg.com/),
following in the footsteps of community projects like [Jinteki.net](https://jinteki.net)
(Netrunner) or [Talishar](https://talishar.net/) (Flesh and Blood): play online,
in the browser, with any deck you own.

> **Project status:** prototype (phase P0). The server runs and handles
> WebSocket sessions, but no rules engine is implemented yet.
> See the [roadmap](#roadmap).

## ⚠️ Unaffiliated community project

This project is a **fan project**, developed independently and without any
affiliation with Neuroscape TCG, its publishers or distributors.

- No official assets (card art, logos, card text) are used without explicit
  permission.
- Card definitions are descriptive rules data, for interoperability and
  preservation purposes, similar to what [Scryfall](https://scryfall.com/)
  does for MTG or NetrunnerDB for Netrunner.
- This project is **not** monetized and never will be.

For any question or report: open an issue on this repo.

## Architecture

**Authoritative** match server in Go:

Founding principles:

- **The client is never trusted.** Everything goes through the server, which
  validates actions and redistributes filtered views (hidden information
  masked per side).
- **Command/event model.** Every action is a serializable command; the event
  log enables replays, spectating, and regression replay.
- **Single-threaded actor per match**: a game's state is only mutated by its
  dedicated goroutine — no mutexes in game logic, no data races.

## Stack

| Component  | Choice                                    |
|------------|-------------------------------------------|
| Language   | Go ≥ 1.22                                  |
| Real-time  | WebSocket (gorilla/websocket)              |
| Testing    | testify + rapid (property-based)           |
| Lint       | golangci-lint, gosec, govulncheck          |
| Infra      | AWS ECS Fargate, ALB, Terraform            |
| CI/CD      | GitHub Actions (tests + lint + deploy)     |

## Getting Started

Prerequisites: Go ≥ 1.22, Docker, golangci-lint, make.

```bash
git clone <repo>
cd matchserver
make help       # lists all targets

make run        # starts the server on :8080
make check      # build + tests (-race) + lint — exact equivalent of CI
make test       # tests only
make cover      # HTML coverage (cover.html)
