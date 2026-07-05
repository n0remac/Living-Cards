# Living Card

Living Card is a Go web app for experimenting with cards as structured data. The current first-screen experience is a small world-deck game: the server renders cards, the browser cycles through the deck, collectible cards move into a library, and library cards can be played onto world cards.

The app also retains a backend-owned draft-card designer API and hidden overlay for prompt-generated or manually edited component fragments. Both the world game and designer state are process-local, mutex-protected, and reset when the server process exits.

Latest reviewed state: current working tree with chained fuse/switch puzzle packs.

## Current Scope

- The visible UI is the world-deck game stage.
- The seeded world deck and later puzzle packs are embedded pure JSON under `internal/game/decks/`.
- Deck data includes card metadata, document variants, initial flags, initial active card, initial message, and declarative use rules.
- The seeded deck has a locked door, inventory label, collectible bent iron key, faded photograph, and sleeping switch.
- Playing the collected key onto the locked door is data-driven: a rule matches source/target/flags, sets `doorUnlocked`, updates door state/tags, swaps the door document variant, loads the fuse-room pack, and sets the message.
- The fuse room adds clue/red herring cards and a collectible glass fuse. Playing the fuse onto the sleeping switch powers/flips the switch, swaps its document variant, and loads the generator-room pack.
- Game API responses include both card data and server-rendered preview HTML. Browser JavaScript does not own card state.
- The draft-card designer remains available through `/api/draft-card/*`, but the current page does not expose a normal visible button to open the designer overlay.
- No real database dependency has been added yet. The deck loader validates JSON data, materializes runtime session state, and merges embedded packs so a database source can later replace the embedded JSON source.

## Runtime Flow

- `main.go` loads config, optionally rebuilds the TypeScript bundle when `DEV_MODE=true`, creates one Ollama client, and registers the web server.
- `GET /` serves the world-deck stage rendered with GoDom.
- `GET /assets/app.js` and `GET /assets/app.js.map` serve the committed frontend bundle from `web/dist`.
- `GET /api/game/session` returns the current world-deck session with rendered card previews.
- `POST /api/game/reset` resets the world-deck session from the loaded deck definition.
- `POST /api/game/cycle` moves the active world card forward or backward.
- `POST /api/game/collect` collects a collectible active or specified card into the library.
- `POST /api/game/play-card` plays a collected library card onto a target world card and evaluates declarative deck rules.
- `GET /api/draft-card` returns the current backend-owned draft card document.
- `GET /api/draft-card/rendered` returns the draft document, server-rendered preview HTML, and library items.
- `GET /api/draft-card/interactive` returns draft document state, game progress, overlays, components, and library items for the hidden designer/tapping workflow.
- `POST /api/draft-card/reset` resets draft document and draft progress while preserving the in-memory design library.
- `POST /api/draft-card/tap`, `/interact`, `/control-change`, and `/randomize-component` drive the retained draft-card component progression system.
- `POST /api/draft-card/components` adds a textarea, shape, or image component with a default or validated fragment.
- `POST /api/draft-card/fragments/background`, `/border`, `/textarea`, and `/image` ask Ollama for AI-generated fragments.
- `POST /api/draft-card/apply-fragment` validates and applies generated or manually edited fragments for background, border, textarea, shape, or image targets.
- `GET /api/draft-card/library?target=...`, `POST /api/draft-card/library/save-applied`, and `POST /api/draft-card/library/apply` manage in-memory design presets and saved fragments.

## World Deck Model

- `internal/game/deck.go` defines `DeckDefinition`, `CardDefinition`, `UseRuleDefinition`, card matchers, and rule effects.
- `LoadEmbeddedSeededWorldDeck` reads the seed deck; `LoadEmbeddedDeck` reads any embedded deck pack by id.
- `NewSessionFromDeck` materializes runtime `Card` values from deck definitions and keeps document variants available for rule effects.
- `loadDeck` effects append cards from another embedded JSON file, merge rules/document variants/initial flags, focus the loaded pack’s initial active card, and are idempotent per session.
- Validation rejects empty decks, duplicate card ids, missing initial active cards, missing initial document variants, mismatched document `card_id` values, unknown rule card references, unsupported effect types, invalid deck ids, and invalid document-variant references.
- Pack validation can resolve rule references against cards already loaded in the current session, which lets `fuse_room` target the original `sleeping-switch`.
- Supported rule effects are `setFlag`, `setCardState`, `removeCardTags`, `setDocumentVariant`, `setMessage`, and `loadDeck`.
- The runtime session API shape is intentionally stable: `Card`, `Snapshot`, `Collect`, `Cycle`, and rendered web responses still use the same public fields.

## Draft Card Model

- `card.Document` is a tree of `card.Node` values.
- The root card fragment stores card padding and shell shadow.
- Each child component stores target-specific JSON as `json.RawMessage`; JSON object fragments can be stored directly in document/deck JSON.
- Registered component types are card, background, border, textarea, shape, and image.
- Background and border contribute card shell styles.
- Textarea contributes an absolutely positioned text layer.
- Shape contributes an absolutely positioned SVG layer.
- Image contributes an absolutely positioned safe embedded-image layer.
- Render failures are treated as server errors for full preview reads and bad-request errors when applying fragments or interactions.

## Draft Progression Rules

- Initial draft global level is 1 with only the `card` component type unlocked and `card-root` selected.
- Each XP-bearing interaction grants 1 XP. Global level is `total XP / 5 + 1`.
- Component level is `component XP / 3 + 1`.
- Card-root traits start with `background` and `border`; `shadow` unlocks at global level 2; `padding` unlocks at global level 4.
- Textarea unlocks at global level 3. Shape unlocks at global level 7 and creates `shape-1` if missing from the draft document.
- Image components are recognized once present in the draft document and have their own controls/progress.
- Component overlays unlock at component level 3.
- Long press grants interaction XP. The first successful overlay open also records an additional interaction bonus.
- Short taps mutate and grant XP while randomization is enabled. If `preventRandomizing` is enabled for a component, short taps select the component without mutating or granting XP.
- Control changes grant XP only when they actually change document or component-progress state.
- Locked known components or traits return an `invalidAction` event without mutating state.
- Unknown component IDs, targets, controls, or interactions return a bad request.

## Backend Layout

- `internal/config` owns environment config and validation.
- `internal/ollama` owns the Ollama chat client used by fragment generation. Embedding/model-list helpers remain, but the current app path uses chat only.
- `internal/fragment` owns strict JSON fragment generation, normalization, validation issues, repair attempts, and safe inline CSS helpers.
- `internal/components/card` owns the document model, root fragment, registry, default document, library item shape, and GoDom preview shell rendering.
- `internal/components/background`, `border`, `textarea`, `shape`, and `image` own component fragment schemas, validation, defaults, rendering, random generation, and AI prompt specs where implemented.
- `internal/game` owns the data-driven world-deck model and process-local game session.
- `internal/web` owns HTTP routing, page rendering, preview rendering, component registration, draft designer state, and world-game response rendering.
- `internal/webbuild` builds the TypeScript bundle with esbuild.
- `web/src/game/GameController.ts` owns the visible world-deck browser interactions.
- `web/src/designer/*` and `web/src/stage/*` retain the hidden draft-card designer/tapping client flows.

## Fragment Safety

- AI output must be one strict JSON object with `target`, `description`, and `fragment`.
- Unknown JSON fields are rejected during strict fragment decoding.
- Fragment apply targets are restricted to `background`, `border`, `textarea`, `shape`, and `image`.
- Active AI generation routes cover `background`, `border`, `textarea`, and `image`.
- Colors must be hex, `rgb(...)`, `rgba(...)`, `hsl(...)`, or `hsla(...)`.
- Inline CSS is allowlisted per component and rejects markers such as raw angle brackets, braces, `url(`, `javascript:`, `expression(`, `@import`, `position`, and `content`.
- Image sources must be embedded PNG, JPEG, WebP, or GIF data URLs; SVG, external URLs, HTML, and JavaScript are rejected.
- Numeric fields such as border width/radius, text size, shape size, image size, and component position are normalized with bounded ranges.
- Invalid model output includes structured issues and, when available, the raw response so the frontend can show an editable recovery path.

## Recent Changes Reviewed

- Current working tree adds `loadDeck`, idempotent embedded pack loading, `fuse_room.json`, `generator_room.json`, and tests for the chained fuse/switch puzzle.
- `1f5aa3e` moved the world deck out of Go constructors and into embedded pure JSON.
- `1f5aa3e` added typed deck definitions, deck validation, `NewSessionFromDeck`, document variants, and declarative use-rule effects.
- `1f5aa3e` simplified game-card rendering around the existing `card.Document` renderer and added tests for loaded deck data and invalid deck fixtures.
- `68f2c84` introduced the visible world-deck game format, `/api/game/*` endpoints, collectible library cards, drag/drop play-card interactions, and image components.
- `68f2c84` added image fragment schema/rendering/validation and expanded draft-card component APIs to include image.
- `30b4f38` moved draft component controls into a static edge overlay with fixed top/side/bottom slots.

## Known Gaps

- There is still no persistent database. Deck definitions are database-ready data, but the active content source is embedded JSON and session state is in memory.
- The visible world-deck game is still a short linear puzzle path; the generator room is loaded as the next area but does not yet have its own deeper mechanics.
- The designer overlay is still rendered and wired, but the current page does not render a visible `designer-toggle-btn`, so there is no normal UI path to open it.
- Shape fragments can be applied manually and generated randomly, but there is still no `/api/draft-card/fragments/shape` AI generation route.
- The seeded design library does not include shape or image presets.
- The retained `web/src/stage/colorControls.ts` module and some older draft-card API helpers are unused by the visible world-deck UI.
- `docker-compose.yml` still includes Qdrant from older flows, but the current app does not use a vector database.

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
