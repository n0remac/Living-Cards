# Living Card

Living Card is a Go web app for one process-local editable card document. The current primary surface is an interactive card stage: the user taps card zones to mutate the card, earns XP, levels up, and unlocks simple color controls. An AI JSON fragment designer still exists as a hidden overlay/API workflow for prompt-generated or manually edited fragments.

The current architecture has removed the older multi-card, chat, memory, profile, embedding, and patch-proposal product flows. State is backend-owned, mutex-protected, and in-memory for the life of the server process.

Latest reviewed commit: `704461e Tighten up card taping screen`.

## Current Scope

- One draft card document is active per running server process.
- The draft document has a `card` root with `background`, `border`, and `textarea` child components.
- The stage is tap-first: short taps apply random generated fragments for unlocked tap targets.
- Random tap generation currently supports `background` and `border`.
- `textarea` remains part of the card document, preset library, AI generation, validation, and rendering paths, but it is locked in the tap loop and has no random tap generator.
- Game progress tracks total taps, XP, level, unlocked targets, unlocked modes, and per-target progress.
- Simple color controls unlock at level 5 for background and border.
- The designer library is backend-owned and seeded with background, border, and textarea presets. Saved items live only in memory.
- Frontend JavaScript does not own card state. It sends interactions to the backend, replaces server-rendered preview HTML, updates HUD/notifications, and manages transient UI state.

## Runtime Flow

- `main.go` loads config, optionally rebuilds the TypeScript bundle when `DEV_MODE=true`, creates one Ollama client, and registers the web server.
- `GET /` serves the interactive stage page rendered with GoDom.
- `GET /assets/app.js` and `GET /assets/app.js.map` serve the committed frontend bundle from `web/dist`.
- `GET /api/draft-card` returns the current backend-owned draft card document.
- `GET /api/draft-card/rendered` returns the current draft document, server-rendered preview HTML, and library items.
- `GET /api/draft-card/interactive` returns the current draft document, game state, rendered preview HTML, available targets, and library items.
- `POST /api/draft-card/reset` resets the draft document, game progress, and last applied fragment; the in-memory library is preserved.
- `POST /api/draft-card/tap` applies a random fragment for an unlocked tap target and returns updated game events.
- `POST /api/draft-card/control-change` applies an unlocked background or border color/gradient control.
- `POST /api/draft-card/fragments/background` asks Ollama for a background fragment.
- `POST /api/draft-card/fragments/border` asks Ollama for a border fragment.
- `POST /api/draft-card/fragments/textarea` asks Ollama for a text-area fragment.
- `POST /api/draft-card/apply-fragment` validates a generated or manually edited fragment, applies the normalized fragment to the draft document, and returns updated preview HTML.
- `GET /api/draft-card/library?target=...` returns presets and saved library items, optionally filtered by target.
- `POST /api/draft-card/library/save-applied` saves the last applied fragment unless an identical item already exists.
- `POST /api/draft-card/library/apply` applies a library item to the current draft.

When the frontend requests an AI update instead of a new generation, the backend can pass the current applied fragment JSON and component ID back into the fragment prompt.

## Game Rules

- Initial level is 1 with `background` and `border` unlocked.
- Each successful tap grants 1 XP.
- Global level is `XP / 5 + 1`.
- Per-target level is based on taps to that target: `target taps / 3 + 1`.
- Level 5 unlocks `simpleControls` for background and border.
- Locked but known targets return an `invalidAction` event without mutating state.
- Unknown targets return a bad request.
- Color controls validate colors with the same safe color rules used by generated fragments.
- Gradient controls require both a primary and secondary safe color. A zero angle defaults to 135 degrees.

## Backend Layout

- `internal/config` owns environment config and validation.
- `internal/ollama` owns the Ollama chat client used by fragment generation. Embedding and model-list helpers remain in this package, but the current app path uses chat.
- `internal/fragment` owns the generic generation pipeline: prompt calls, strict JSON decoding, normalization, structured validation issues, one repair attempt, safe inline CSS helpers, and raw-output error reporting.
- `internal/components/card` owns the card document model, component registry, default document, library item shape, and GoDom preview shell rendering.
- `internal/components/background` owns background fragment schema, prompts, validation, defaults, render contribution, seeded presets, and random tap fragments.
- `internal/components/border` owns border fragment schema, prompts, validation, defaults, render contribution, seeded presets, and random tap fragments.
- `internal/components/textarea` owns text-area fragment schema, prompts, validation, defaults, render layer, and seeded presets.
- `internal/web` owns HTTP routing, page rendering, preview rendering, component registration, game state, designer state, and mutex-protected process-local state.
- `internal/webbuild` builds the TypeScript bundle with esbuild.
- `internal/web/components/appheader` is retained but not mounted by the current stage page.

## Component Model

- `card.Document` is a tree of `card.Node` values.
- Each leaf component stores its target-specific fragment as `json.RawMessage`.
- A component `Definition` contributes shell styles or preview layers through the card registry.
- Background and border components contribute card shell styles.
- Text area contributes an absolutely positioned text layer with `data-component-type="textarea"` for hit testing.
- Render failures are treated as server errors for full preview reads and bad-request errors when applying a fragment.

## Fragment Safety

- AI output must be one strict JSON object with `target`, `description`, and `fragment`.
- Unknown JSON fields are rejected during fragment decoding.
- Targets are restricted to `background`, `border`, and `textarea`.
- Colors must be hex, `rgb(...)`, `rgba(...)`, `hsl(...)`, or `hsla(...)`.
- Inline CSS is allowlisted per component and rejects markers such as raw angle brackets, braces, `url(`, `javascript:`, `expression(`, `@import`, `position`, and `content`.
- Numeric fields such as border width/radius and text size are normalized with bounded ranges.
- Invalid model output includes structured issues and, when available, the raw response so the frontend can show an editable recovery path.

## Frontend Layout

- `web/src/app.ts` boots the designer controller and the interactive stage controller.
- `web/src/api.ts` contains HTTP client functions and fragment error parsing.
- `web/src/types.ts` mirrors API payload shapes, game state, card events, and fragment/library contracts.
- `web/src/stage/StageController.ts` loads interactive state, binds pointer input, handles taps/long-presses, updates the HUD, refreshes previews, and coordinates reset.
- `web/src/stage/hitTesting.ts` maps pointer events to `border`, `background`, or `textarea` hit zones.
- `web/src/stage/cardMotion.ts` owns tap and invalid-tap animations.
- `web/src/stage/colorControls.ts` renders the unlocked color/gradient popover.
- `web/src/stage/overlays.ts` owns the notification queue and notification history panel.
- `web/src/designer/controller.ts` wires the hidden designer overlay: AI generate/update/apply, manual JSON edits, reset, library apply, save, and status messages.
- `web/src/designer/fragments.ts` parses editable generated JSON and formats validation issues.
- `web/src/designer/document.ts` replaces the server-rendered preview node after validating the returned HTML shape.

The generated bundle is committed under `web/dist`.

## Recent Changes Reviewed

- The page now renders a full-screen interactive stage instead of the older header/sidebar designer layout.
- A compact HUD shows level, XP, taps, and notification history above the card.
- Taps use pointer hit testing and card animations before/after backend mutation.
- The latest commit added long-press color controls, gradient support, and the `/api/draft-card/control-change` route.
- Notifications moved from temporary toast elements to a queue/history UI in the HUD.
- Tests now cover interactive stage rendering, tap progression, level-up events, simple control unlocks, color/gradient controls, reset behavior, AI fragment generation/repair, and library flows.

## Known Gaps

- The designer overlay is still rendered and wired, but the current page does not render a visible `designer-toggle-btn`, so there is no normal UI path to open it.
- `textarea` is declared as a known tap target, but it is not initially unlocked and has no random tap fragment generator.
- All document, game, and library state is process-local and is lost when the server exits.

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
