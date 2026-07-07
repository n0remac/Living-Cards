# Living Card

Living Card is a Go web app for experimenting with cards as structured, server-rendered data. The visible app is a small world-deck game: cycle through cards, collect useful cards into a library, build simple card-based tools, and play library cards onto world cards.

The current game content is pure JSON under `internal/game/decks/`. Runtime code validates that data, materializes a process-local session from it, and can load additional deck packs from declarative rule effects. This keeps the deck format ready for a future database source.

## What Is In The App

- A visible world-deck game served at `/`.
- Server-rendered card previews built from `card.Document` JSON, including background, border, textarea, shape, image, and slider components.
- Data-driven deck packs with document variants, declarative use rules, slider-value source conditions, and chained `loadDeck` effects.
- A seeded puzzle path where opening the door loads the fuse room, using the fuse on the switch loads the generator room, and a tuned regulator controller powers the generator.
- A full-screen draft edit workflow where editable library cards can accept consumed component cards, expose HTML/CSS-style controls, and save back into the library.
- A retained draft-card designer API for generating and applying background, border, textarea, shape, and image configs.
- In-memory state only; restarting the server resets the game and draft card state.

## Requirements

- Go 1.26.1 or compatible with the version in `go.mod`.
- Node/npm are only needed when rebuilding the TypeScript bundle in development.
- Ollama is optional unless you use AI config generation routes.

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
- `POST /api/game/edit/start`
- `POST /api/game/edit/install-component`
- `POST /api/game/edit/control-change`
- `POST /api/game/edit/save`
- `POST /api/game/edit/cancel`
- `POST /api/game/save-controller`

Draft card/designer:

- `GET /api/draft-card`
- `GET /api/draft-card/rendered`
- `GET /api/draft-card/interactive`
- `POST /api/draft-card/apply-config`
- `POST /api/draft-card/components`
- `POST /api/draft-card/configs/background`
- `POST /api/draft-card/configs/border`
- `POST /api/draft-card/configs/textarea`
- `POST /api/draft-card/configs/image`

See `project.md` for the full architecture notes and current known gaps.
