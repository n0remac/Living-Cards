# Living Card

Living Card is a Go web app for experimenting with cards as structured, server-rendered data. The visible app is a small world-deck game: cycle through cards, collect useful cards into a library, and play library cards onto world cards.

The current seeded deck is pure JSON at `internal/game/decks/seeded_world.json`. Runtime code validates that data and materializes a process-local session from it, which keeps the deck format ready for a future database source.

## What Is In The App

- A visible world-deck game served at `/`.
- Server-rendered card previews built from `card.Document` JSON.
- A data-driven seeded deck with document variants and declarative use rules.
- A retained draft-card designer API for generating and applying background, border, textarea, shape, and image fragments.
- In-memory state only; restarting the server resets the game and draft card state.

## Requirements

- Go 1.26.1 or compatible with the version in `go.mod`.
- Node/npm are only needed when rebuilding the TypeScript bundle in development.
- Ollama is optional unless you use AI fragment generation routes.

## Run Locally

```sh
make restart
```

The app listens on `http://127.0.0.1:8090` by default. Useful commands:

```sh
make status
make logs
make stop
```

Run tests with:

```sh
go test ./...
```

## Configuration

Defaults:

- `WEB_ADDR=127.0.0.1:8090`
- `OLLAMA_BASE_URL=http://127.0.0.1:11434`
- `OLLAMA_CHAT_MODEL=qwen2.5:3b-instruct`
- `REQUEST_TIMEOUT_SECONDS=45`
- `DEV_MODE=false`

The `Makefile` runs with `DEV_MODE=true` by default so the frontend bundle is rebuilt during local restarts.

## Main APIs

World deck:

- `GET /api/game/session`
- `POST /api/game/reset`
- `POST /api/game/cycle`
- `POST /api/game/collect`
- `POST /api/game/play-card`

Draft card/designer:

- `GET /api/draft-card`
- `GET /api/draft-card/rendered`
- `GET /api/draft-card/interactive`
- `POST /api/draft-card/apply-fragment`
- `POST /api/draft-card/components`
- `POST /api/draft-card/fragments/background`
- `POST /api/draft-card/fragments/border`
- `POST /api/draft-card/fragments/textarea`
- `POST /api/draft-card/fragments/image`

See `project.md` for the full architecture notes and current known gaps.
