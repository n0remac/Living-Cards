# Living Card

Living Card is a Go web app for one process-local editable card document. The primary surface is an interactive card stage: the user taps or long-presses card components, earns XP, unlocks component traits and controls, and edits the rendered card through server-owned state. A hidden AI JSON fragment designer still exists as an overlay/API workflow for prompt-generated or manually edited fragments.

The older multi-card, chat, memory, profile, embedding, and patch-proposal product flows have been removed. State is backend-owned, mutex-protected, and in-memory for the life of the server process.

Latest reviewed commit: `30b4f38 static editing controls in overlay`.

## Current Scope

- One draft card document is active per running server process.
- The default document has a `card` root with `background`, `border`, and `textarea` children.
- A `shape` child is appended when the shape component unlocks at global level 7.
- Known component IDs are `card-root`, `textarea-main`, and `shape-1`.
- The stage is component-first: short taps randomize unlocked traits while randomization is enabled; long presses request an editable component overlay; dragging moves text and shape components through the position control.
- Card-root randomization covers background, border, shadow, and padding as those traits unlock.
- Textarea and shape have their own random fragment generators, component progress, overlays, and controls.
- Game progress tracks total XP, global level, unlocked component types, selected component, and per-component progress. Legacy target fields remain in API responses for compatibility with earlier background/border flows.
- Component overlays unlock at component level 3. The latest UI renders overlay controls into fixed edge slots around the stage instead of pointer-anchored floating panels.
- The designer library is backend-owned and seeded with background, border, and textarea presets. Saved items live only in memory.
- Frontend JavaScript does not own card state. It sends interactions to the backend, replaces server-rendered preview HTML, updates HUD/notifications, and manages transient pointer/UI state.

## Runtime Flow

- `main.go` loads config, optionally rebuilds the TypeScript bundle when `DEV_MODE=true`, creates one Ollama client, and registers the web server.
- `GET /` serves the interactive stage page rendered with GoDom.
- `GET /assets/app.js` and `GET /assets/app.js.map` serve the committed frontend bundle from `web/dist`.
- `GET /api/draft-card` returns the current backend-owned draft card document.
- `GET /api/draft-card/rendered` returns the current draft document, server-rendered preview HTML, and library items.
- `GET /api/draft-card/interactive` returns the current draft document, game state, rendered preview HTML, available legacy targets, available component descriptors, the selected component overlay when unlocked, and library items.
- `POST /api/draft-card/reset` resets the draft document, game progress, and last applied fragment; the in-memory library is preserved.
- `POST /api/draft-card/tap` is the legacy target/zone interaction endpoint. It maps background, border, textarea, and shape targets to component interactions.
- `POST /api/draft-card/interact` applies a component interaction such as `shortTap` or `longPress`.
- `POST /api/draft-card/control-change` applies generic component controls. It also retains the older target color/gradient payload shape for background and border controls.
- `POST /api/draft-card/randomize-component` randomizes a component trait directly.
- `POST /api/draft-card/fragments/background` asks Ollama for a background fragment.
- `POST /api/draft-card/fragments/border` asks Ollama for a border fragment.
- `POST /api/draft-card/fragments/textarea` asks Ollama for a text-area fragment.
- `POST /api/draft-card/apply-fragment` validates a generated or manually edited fragment, applies the normalized fragment to the draft document, and returns updated preview HTML. The apply path accepts background, border, textarea, and shape targets.
- `GET /api/draft-card/library?target=...` returns presets and saved library items, optionally filtered by target.
- `POST /api/draft-card/library/save-applied` saves the last applied fragment unless an identical item already exists.
- `POST /api/draft-card/library/apply` applies a library item to the current draft.

When the frontend requests an AI update instead of a new generation, the backend can pass the current applied fragment JSON and component ID back into the fragment prompt.

## Game Rules

- Initial global level is 1 with only the `card` component type unlocked and `card-root` selected.
- Each XP-bearing interaction grants 1 XP. Global level is `total XP / 5 + 1`.
- Component level is `component XP / 3 + 1`.
- Card-root traits start with `background` and `border`; `shadow` unlocks at global level 2; `padding` unlocks at global level 4.
- Textarea unlocks at global level 3. Shape unlocks at global level 7 and creates `shape-1` if it is missing from the document.
- Component overlays unlock at component level 3.
- Long press grants interaction XP. The first successful overlay open also records an additional interaction bonus and marks the overlay as opened.
- Short taps mutate and grant XP while randomization is enabled. If `preventRandomizing` is enabled for a component, short taps select the component without mutating or granting XP.
- Control changes grant XP only when they actually change document or component-progress state.
- Card-root controls include `preventRandomizing` at component level 3, then background/border color and border width/radius controls at global level 5. Shadow and padding controls are included once the global control gate is open.
- Text controls unlock progressively from component level 3: content, background, border, x/y/position, then typography, padding, and border shape controls at later global or component levels.
- Shape controls unlock progressively from component level 3: shape, fill, x/y/position, then border, size, rotation, and shadow controls at later global or component levels.
- Locked known components or traits return an `invalidAction` event without mutating state.
- Unknown component IDs, targets, controls, or interactions return a bad request.
- Color controls validate colors with the same safe color rules used by generated fragments.
- Gradient controls require both a primary and secondary safe color. A zero angle defaults to 135 degrees.

## Backend Layout

- `internal/config` owns environment config and validation.
- `internal/ollama` owns the Ollama chat client used by fragment generation. Embedding and model-list helpers remain in this package, but the current app path uses chat.
- `internal/fragment` owns the generic generation pipeline: prompt calls, strict JSON decoding, normalization, structured validation issues, one repair attempt, safe inline CSS helpers, and raw-output error reporting.
- `internal/components/card` owns the card document model, root fragment, component registry, default document, library item shape, and GoDom preview shell rendering.
- `internal/components/background` owns background fragment schema, prompts, validation, defaults, render contribution, seeded presets, and random fragments.
- `internal/components/border` owns border fragment schema, prompts, validation, defaults, render contribution, seeded presets, and random fragments.
- `internal/components/textarea` owns text-area fragment schema, prompts, validation, defaults, render layer, seeded presets, and random fragments.
- `internal/components/shape` owns shape fragment schema, prompts, validation, defaults, SVG layer rendering, and random fragments.
- `internal/web` owns HTTP routing, page rendering, preview rendering, component registration, game state, designer state, and mutex-protected process-local state.
- `internal/webbuild` builds the TypeScript bundle with esbuild.
- `internal/web/components/appheader` is retained but not mounted by the current stage page.

## Component Model

- `card.Document` is a tree of `card.Node` values.
- The root card fragment currently stores card padding and shell shadow.
- Each leaf component stores its target-specific fragment as `json.RawMessage`.
- A component `Definition` contributes shell styles or preview layers through the card registry.
- Background and border components contribute card shell styles.
- Textarea contributes an absolutely positioned text layer with `data-component-id` and `data-component-type="textarea"` for hit testing and dragging.
- Shape contributes an absolutely positioned SVG layer with `data-component-id` and `data-component-type="shape"` for hit testing and dragging.
- Render failures are treated as server errors for full preview reads and bad-request errors when applying a fragment or interaction.

## Fragment Safety

- AI output must be one strict JSON object with `target`, `description`, and `fragment`.
- Unknown JSON fields are rejected during fragment decoding.
- Fragment apply targets are restricted to `background`, `border`, `textarea`, and `shape`.
- The active AI generation routes currently cover `background`, `border`, and `textarea`.
- Colors must be hex, `rgb(...)`, `rgba(...)`, `hsl(...)`, or `hsla(...)`.
- Inline CSS is allowlisted per component and rejects markers such as raw angle brackets, braces, `url(`, `javascript:`, `expression(`, `@import`, `position`, and `content`.
- Numeric fields such as border width/radius, text size, shape size, and component position are normalized with bounded ranges.
- Invalid model output includes structured issues and, when available, the raw response so the frontend can show an editable recovery path.

## Frontend Layout

- `web/src/app.ts` boots the designer controller and the interactive stage controller.
- `web/src/api.ts` contains HTTP client functions and fragment error parsing.
- `web/src/types.ts` mirrors API payload shapes, game state, component progress, overlays, controls, card events, and fragment/library contracts.
- `web/src/stage/StageController.ts` loads interactive state, binds pointer input, handles taps/long-presses/drags, updates the HUD, refreshes previews, coordinates reset, and opens component overlays.
- `web/src/stage/hitTesting.ts` maps pointer events to card-root background/border traits or to text/shape components.
- `web/src/stage/cardMotion.ts` owns tap and invalid-tap animations.
- `web/src/stage/componentControls.ts` renders the active fixed edge-control overlay, splits controls into left/right rails, and manages overlay close/open state.
- `web/src/stage/overlays.ts` owns the notification queue and notification history panel; while controls are open, notifications can also write into the edge-control status line.
- `web/src/stage/colorControls.ts` is a retained older pointer-anchored color popover module and is not imported by the current stage.
- `web/src/designer/controller.ts` wires the hidden designer overlay: AI generate/update/apply, manual JSON edits, reset, library apply, save, and status messages.
- `web/src/designer/fragments.ts` parses editable generated JSON and formats validation issues.
- `web/src/designer/document.ts` replaces the server-rendered preview node after validating the returned HTML shape.

The generated bundle is committed under `web/dist`.

## Recent Changes Reviewed

- `30b4f38` moved component controls into a static edge overlay with top, left, right, and bottom slots.
- The stage now hides the HUD and reset button while component controls are open.
- Component controls split range/color controls to the right rail and other controls to the left rail.
- Overlay close/open state is now tracked on `document.body` with `stage-controls-open`.
- Pointer handling closes the overlay before a new press, suppresses accidental taps after close, and still allows drag-to-move for text and shape components.
- Notifications can mirror into `stage-edge-controls-status` while controls are open.
- Page-rendering tests now assert the edge-control overlay scaffold.

## Known Gaps

- The designer overlay is still rendered and wired, but the current page does not render a visible `designer-toggle-btn`, so there is no normal UI path to open it.
- Shape fragments can be applied manually and generated randomly, but there is no `/api/draft-card/fragments/shape` AI generation route or visible designer target for shape.
- The seeded library does not include shape presets.
- The retained `web/src/stage/colorControls.ts` module and some older API helpers are unused by the active stage UI.
- Edge-control status can be overwritten when an overlay rerenders after a control response, so some unlock/level notifications may only remain visible in notification history.
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
