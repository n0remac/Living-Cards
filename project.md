# Living Card

Living Card is a Go web app for designing one editable card document. The current product surface is a draft-card designer: users choose a component target, ask Ollama for a safe JSON fragment, review or edit that fragment, then apply it to a server-rendered card preview.

The latest architecture removed the older card, chat, memory, profile, embedding, and patch-proposal flows. State is backend-owned, but still process-local and in-memory.

## Current Scope

- One draft card document is active per running server process.
- The card document has a `card` root with `background`, `border`, and `textarea` child components.
- AI output is limited to declarative JSON fragments for those component targets.
- The server validates, normalizes, and applies fragments before rendering preview HTML.
- The design library is backend-owned and seeded with component presets. Saved items live only in memory for the life of the process.
- JavaScript does not own card state. It coordinates events, API calls, editable generated JSON, status text, and preview replacement.

## Runtime Flow

- `main.go` loads config, optionally rebuilds the frontend when `DEV_MODE=true`, creates one Ollama client, and registers the web server.
- `GET /` serves the designer page rendered with GoDom.
- `GET /assets/app.js` and `GET /assets/app.js.map` serve the bundled TypeScript assets from `web/dist`.
- `GET /api/draft-card` returns the current backend-owned draft card document.
- `GET /api/draft-card/rendered` returns the current draft document, GoDom-rendered preview HTML, and library items.
- `POST /api/draft-card/reset` resets the draft document to defaults and clears the last applied fragment.
- `POST /api/draft-card/fragments/background` generates a background fragment.
- `POST /api/draft-card/fragments/border` generates a border fragment.
- `POST /api/draft-card/fragments/textarea` generates a text-area fragment.
- `POST /api/draft-card/apply-fragment` validates a generated or manually edited fragment, applies the normalized fragment to the draft document, and returns updated preview HTML.
- `GET /api/draft-card/library?target=...` returns presets and saved library items, optionally filtered by target.
- `POST /api/draft-card/library/save-applied` saves the last applied fragment into the process-local library unless an identical item already exists.
- `POST /api/draft-card/library/apply` applies a library item to the current draft.

When the frontend requests an update instead of a new generation, the backend can pass the current applied fragment JSON and component ID back into the fragment prompt.

## Backend Layout

- `internal/config` owns environment config and validation.
- `internal/ollama` owns the Ollama chat client.
- `internal/fragment` owns the generic generation pipeline: prompt calls, strict JSON decoding, normalization, structured validation issues, one repair attempt, safe inline CSS helpers, and raw-output error reporting.
- `internal/components/card` owns the card document model, component registry, default document, library item shape, and GoDom preview shell rendering.
- `internal/components/background` owns background fragment schema, prompts, validation, defaults, render contribution, and presets.
- `internal/components/border` owns border fragment schema, prompts, validation, defaults, render contribution, and presets.
- `internal/components/textarea` owns text-area fragment schema, prompts, validation, defaults, render layer, and presets.
- `internal/web` owns HTTP routing, page rendering, preview rendering, component registration, and the mutex-protected process-local designer state.
- `internal/web/components/appheader` owns the small reusable app header.
- `internal/webbuild` builds the TypeScript bundle with esbuild.

## Component Model

- `card.Document` is a tree of `card.Node` values.
- Each leaf component stores its target-specific fragment as `json.RawMessage`.
- A component `Definition` contributes shell styles or preview layers through the card registry.
- Background and border components contribute card shell styles.
- Text area contributes an absolutely positioned text layer.
- Render failures are treated as server errors for full preview reads and bad-request errors when applying a fragment.

## Fragment Safety

- The model must return one strict JSON object with `target`, `description`, and `fragment`.
- Unknown JSON fields are rejected during fragment decoding.
- Targets are restricted to `background`, `border`, and `textarea`.
- Colors must be hex, `rgb(...)`, `rgba(...)`, `hsl(...)`, or `hsla(...)`.
- Inline CSS is allowlisted per component and rejects markers such as raw angle brackets, braces, `url(`, `javascript:`, `expression(`, `@import`, `position`, and `content`.
- Numeric fields such as border width/radius and text size are normalized with bounded ranges.
- Invalid model output includes structured issues and, when available, the raw response so the frontend can show an editable recovery path.

## Frontend Layout

- `web/src/app.ts` bootstraps the app header and designer.
- `web/src/api.ts` contains the HTTP client functions and fragment error parsing.
- `web/src/types.ts` mirrors the API payload shapes.
- `web/src/designer/controller.ts` wires form submission, generate/update/apply/reset/save/library actions, status messages, busy states, and DOM rendering.
- `web/src/designer/fragments.ts` parses editable generated JSON and formats validation issues.
- `web/src/designer/document.ts` replaces the server-rendered preview node after validating the returned HTML shape.

The generated bundle is committed under `web/dist`.

## Development

- Run tests with `go test ./...`.
- Build and restart the app with `make restart`.
- Stop the app with `make stop`.
- Tail logs with `make logs`.

Default config values:

- `WEB_ADDR=127.0.0.1:8090`
- `OLLAMA_BASE_URL=http://127.0.0.1:11434`
- `OLLAMA_CHAT_MODEL=qwen2.5:3b-instruct`
- `REQUEST_TIMEOUT_SECONDS=45`
- `DEV_MODE=false`

`Makefile` defaults `DEV_MODE=true` for local restarts so the frontend bundle is rebuilt at startup.
