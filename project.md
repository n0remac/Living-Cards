# Living Card

Living Card is now a draft-card designer. The app renders one editable card document and uses Ollama to generate safe JSON fragments for component verticals.

## Runtime Flow

- `main.go` loads config, optionally rebuilds the frontend in dev mode, creates one Ollama client, and registers the web server.
- `GET /` serves the designer page.
- `GET /api/draft-card` returns the current backend-owned draft card document.
- `GET /api/draft-card/rendered` returns the current draft document, GoDom-rendered preview HTML, and design library items.
- `POST /api/draft-card/reset` resets the backend-owned draft document and clears the current applied fragment.
- `POST /api/draft-card/fragments/background` generates a background fragment.
- `POST /api/draft-card/fragments/border` generates a border fragment.
- `POST /api/draft-card/fragments/textarea` generates a text-area fragment.
- `POST /api/draft-card/apply-fragment` validates a generated or edited fragment, applies the normalized fragment to the backend-owned draft document, and returns updated GoDom-rendered preview HTML.
- `GET /api/draft-card/library?target=...` returns backend-owned library presets and saved items.
- `POST /api/draft-card/library/save-applied` saves the last applied fragment into the backend-owned library.
- `POST /api/draft-card/library/apply` applies a backend-owned library item to the current draft.

The app no longer exposes static card, chat, memory, profile, or patch-proposal routes.

## Backend Layout

- `internal/fragment` owns the generic generation pipeline: prompt calls, strict JSON decoding, structured validation issues, one repair attempt, and inspectable raw-output errors.
- `internal/components/card` owns the card document model and composition shell.
- `internal/components/background` owns background fragment schema, prompts, validation, defaults, render contribution, and preset data.
- `internal/components/border` owns border fragment schema, prompts, validation, defaults, render contribution, and preset data.
- `internal/components/textarea` owns text-area fragment schema, prompts, validation, defaults, render contribution, and preset data.
- `internal/web` owns HTTP routing, page rendering, and process-local draft/library state.
- `internal/webbuild` builds the TypeScript bundle with esbuild.

## Frontend Layout

- `web/src/app.ts` is a small bootstrap for the app header and designer.
- `web/src/designer/controller.ts` wires events, fetch calls, status text, and DOM updates.
- `web/src/designer/document.ts` swaps server-rendered preview HTML.
- `web/src/designer/fragments.ts` handles generic generated-fragment JSON parsing and API generation dispatch.

The browser no longer owns draft card state or design library state. JavaScript only coordinates user actions and applies server responses to the DOM.

AI-generated output is limited to declarative JSON fragments and safe inline CSS declarations. JavaScript remains handwritten.
