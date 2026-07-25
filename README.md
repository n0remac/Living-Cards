# Living Card

Living Card is a Go web app for experimenting with cards as structured, server-rendered data. The visible app is a small world-deck game: cycle through cards, collect useful cards into a library, build simple card-based tools, and play library cards onto world cards.

The current game content is pure JSON under `internal/game/decks/`. Runtime code validates that data, materializes a process-local session from it, and can load additional deck packs from declarative rule effects. This keeps the deck format ready for a future database source.

## What Is In The App

- A visible world-deck game served at `/`.
- Server-rendered card previews built from strictly validated `card.Document` JSON, including background, border, text, shape, image, slider, text-input, and button components.
- Data-driven deck packs with one strictly tagged rule model for card plays, form submissions, and world-component updates, plus typed component-property conditions and chained `loadDeck` effects.
- A seeded puzzle path where opening the door loads the fuse room, using the fuse on the switch loads the generator room, and powering the generator reveals an Archive Terminal password puzzle.
- Editable component cards that become controller bases through the full-screen edit workflow. Added component cards remain pending until save and are retained on cancel.
- A registry-driven draft-card designer API for editing every component that declares controls and generating AI configs for background, border, text, and image.
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
npm ci
npm run typecheck
```

## Generate Card Background Images

The `cardimages` developer command combines abandoned fantasy technology,
nature growth, and signs of reawakening into deterministic image prompts, then
uses the OpenAI Images API to download card-background artwork.

Preview prompts without making a billable request:

```sh
go run ./cmd/cardimages -dry-run -count 3 -seed 42
```

Generate an image after setting `OPENAI_API_KEY`:

```sh
go run ./cmd/cardimages -count 1 -seed 42 -quality low
```

The equivalent Make target accepts command arguments through `ARGS`:

```sh
make card-images ARGS="-dry-run -count 3 -seed 42"
```

Generate one tailored prompt and image for every card currently authored under
`internal/game/decks/`:

```sh
make card-images-game ARGS="-seed 42"
```

Preview the complete game-card batch without making any billable requests:

```sh
make card-images-game ARGS="-dry-run -seed 42"
```

Game-card mode discovers every deck JSON automatically. It uses each card's
name, kind, tags, deck setting, and text from all document variants as
narrative context. Outputs use stable card-ID filenames such as
`glass-fuse.webp` and `glass-fuse.json`. The seed controls art-direction
details independently for each card, so adding another card does not change
existing card prompts.

Defaults use `gpt-image-2`, WebP, medium quality, and an exact 5:7 resolution
of `960x1344`. Images and JSON manifests are written to
`web/assets/card-backgrounds/`. Each manifest records the complete prompt,
selected vocabulary, generation settings, seed, and OpenAI request ID.

Use `-tech "glass fuse"` or `-nature "moss,fungi"` to pin prompt elements.
Existing assets are never overwritten unless `-force` is supplied. Run
`go run ./cmd/cardimages -help` for all options.

Generated WebP files are served from `/assets/card-backgrounds/`. Authored deck
backgrounds select them with a validated local `asset_id`; arbitrary URLs and
the adjacent JSON generation manifests are not exposed by the asset route.

## Generated Card Types

The Go component catalog is the source of truth for component kinds, config
shapes, simple field constraints, controls, capabilities, defaults, and
presets. Generate the committed TypeScript contracts and frontend metadata
after changing a component definition:

```sh
go run ./cmd/cardtypes
```

Check for stale generated files without rewriting them:

```sh
go run ./cmd/cardtypes -check
```

The Go test suite performs the same in-process stale-file check. Generated
files live under `web/src/generated/` and should not be edited by hand.

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
- `POST /api/game/submit-form`
- `POST /api/game/component/select`
- `POST /api/game/component/control-change`
- `POST /api/game/edit/start`
- `POST /api/game/edit/install-component`
- `POST /api/game/edit/component/select`
- `POST /api/game/edit/control-change`
- `POST /api/game/library/component/control-change`
- `POST /api/game/edit/save`
- `POST /api/game/edit/cancel`

Every world-deck endpoint returns the same result envelope:

```json
{
  "revision": 4,
  "snapshot": {},
  "events": []
}
```

`GET /api/game/session` does not advance the revision and returns an empty
event list. Each accepted command advances the in-memory session revision
exactly once, including valid actions rejected by the game world. Malformed
or invalid commands do not advance it. Commands execute transactionally:
rule or snapshot failures restore the previous session state and discard the
command's events.

Events are ordered semantic facts for one command. Their `sequence` starts at
zero for each result. The browser ignores stale snapshots and presents a
revision's events at most once; events are not persisted or replayed.

Draft card/designer:

- `GET /api/draft-card`
- `GET /api/draft-card/rendered`
- `GET /api/draft-card/interactive`
- `POST /api/draft-card/apply-config`
- `POST /api/draft-card/components`
- `POST /api/draft-card/configs/background`
- `POST /api/draft-card/configs/border`
- `POST /api/draft-card/configs/text`
- `POST /api/draft-card/configs/image`

See `project.md` for the full architecture notes and current known gaps.
