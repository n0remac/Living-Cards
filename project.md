# Living Card Project Architecture

Living Card is a Go web app that serves AI-powered "cards". A card is a static Go-defined character/entity with metadata, personality rules, and a list of UI component instances. The server renders the page shell and card component canvas with GoDom, while TypeScript hydrates dynamic behavior in the browser. Ollama provides chat, summaries, embeddings, profile-fact extraction, and patch proposal generation. SQLite stores memories and profile facts; Qdrant stores vector embeddings for semantic retrieval.

## High-Level Runtime

The application starts in `main.go`, loads config, optionally builds the frontend bundle, creates stores/services, registers web routes, and starts an HTTP server.

```go
cardStore, err := cards.NewStaticStore()
ollamaClient := ollama.NewClient(cfg.OllamaBaseURL, cfg.RequestTimeout)
memoryStore, err := memory.NewStore(cfg.MemoryDBPath, index, cfg.OllamaEmbeddingModel)
web.Register(mux, web.Dependencies{
	Cards: cardStore, Memory: memoryStore, Chat: service,
	Profile: profileStore, Patch: ollamaClient,
})
```

The current request loop is:

```text
GET /
  -> internal/web/page.go renders the app shell
  -> /assets/app.js bootstraps frontend state

GET /api/cards
  -> returns static cards from internal/cards

GET /api/cards/{cardID}/canvas
  -> registry renders card component instances as GoDom HTML

POST /api/cards/{cardID}/components/{componentID}/actions/send
  -> chat-form component delegates to chat.Service
  -> chat.Service retrieves memories/profile, calls Ollama, enqueues post-chat processing

POST /api/components/{componentType}/patch-proposals
  -> reads component ContextFiles and asks Ollama for a reviewable patch proposal
```

## Core Concepts

### Static Cards

Cards are no longer loaded from JSON. They are static Go values in `internal/cards/store.go`.

```go
type Card struct {
	CardID      string              `json:"card_id"`
	Name        string              `json:"name"`
	Personality Personality         `json:"personality"`
	Components  []ComponentInstance `json:"components"`
}
```

Each card declares component instances:

```go
func DefaultComponents() []ComponentInstance {
	return []ComponentInstance{{
		ID: "chat-form", Type: "chat-form", Props: map[string]any{},
	}}
}
```

### Component Registry

Components are registered by type. The registry knows how to render a component, dispatch component actions, and expose context files for AI patch proposals.

```go
type Definition struct {
	Type              string
	Render            func(cards.ComponentInstance) *Node
	HandleAction      ActionHandler
	ContextFiles      []string
	ClientInitializer string
}
```

Rendering wraps each component with metadata for frontend hydration:

```go
Div(
	Attr("data-component-id", instance.ID),
	Attr("data-component-type", instance.Type),
	Attr("data-client-initializer", definition.ClientInitializer),
	definition.Render(instance),
)
```

### Component-Local Code

Each component keeps related GoDom, API/action handling, TypeScript, and type files together.

```text
internal/web/components/chatform/
  view.go       GoDom markup
  handlers.go   component definition and action handler
  client.ts     browser behavior
  types.ts      frontend type exports
```

### Ollama Patch Proposals

Patch proposals are review-only. The server reads only files listed in the selected component's `ContextFiles`, sends them to Ollama with the user instruction, and returns the proposed patch text.

```go
proposal, err := s.client.Chat(ctx, s.model, []ollama.ChatMessage{
	{Role: "system", Content: patchSystemPrompt()},
	{Role: "user", Content: patchUserPrompt(componentType, instruction, contextBody)},
})
```

## Backend Files

### `main.go`

Application entrypoint. It wires config, static cards, Ollama, Qdrant-backed embeddings, SQLite-backed memory/profile stores, the background chat processor, the chat service, and web routes.

```go
processor := chat.NewBackgroundProcessor(chat.ProcessorConfig{
	Memory: memoryStore, Profile: profileStore,
	Ollama: ollamaClient, ChatModel: cfg.OllamaChatModel,
})
```

### `internal/config/config.go`

Reads environment variables and validates runtime settings. Important values include Ollama URL/models, Qdrant URL/prefix, SQLite path, web address, timeout, and dev mode.

```go
cfg := Config{
	OllamaBaseURL: strings.TrimRight(readEnvOrDefault("OLLAMA_BASE_URL", defaultOllamaBaseURL), "/"),
	WebAddr:       strings.TrimSpace(readEnvOrDefault("WEB_ADDR", defaultWebAddr)),
	DevMode:       devMode,
}
```

### `internal/config/config_test.go`

Tests config parsing for `DEV_MODE`, including enabled, disabled, default, and invalid values.

### `internal/cards/store.go`

Defines card domain types, the static default catalog, validation, sorting, lookup, and clone safety.

```go
func NewStaticStore() (*Store, error) {
	return NewStore(DefaultCatalog())
}
```

### `internal/cards/store_test.go`

Tests static card loading, name sorting, invalid cards, duplicate component IDs, cloned component props, and the default catalog.

### `internal/chat/errors.go`

Defines shared chat errors, including card-not-found behavior.

```go
var ErrCardNotFound = errors.New("card not found")
```

### `internal/chat/service.go`

Core synchronous chat pipeline. It validates a chat request, loads the selected card, retrieves user profile context and semantic memories, builds prompts, calls Ollama, and returns the assistant response.

```go
retrieved, err := s.memory.Search(callCtx, userID, cardID, message, s.topK)
systemPrompt, userPrompt := BuildPrompt(card, message, userProfile, retrieved)
reply, err := s.ollama.Chat(callCtx, s.chatModel, []ollama.ChatMessage{
	{Role: "system", Content: systemPrompt},
	{Role: "user", Content: userPrompt},
})
```

### `internal/chat/processor.go`

Asynchronous post-chat processor. It summarizes chat turns into retrievable memories and extracts stable user profile facts.

```go
processor.wg.Add(1)
go processor.run()
```

### `internal/chat/service_test.go`

Tests prompt construction, memory persistence behavior, Ollama failure handling, semantic retrieval smoke behavior, and card-not-found errors.

### `internal/embedding/index.go`

Qdrant vector index adapter backed by Ollama embeddings. It creates/validates Qdrant collections, upserts documents, and searches vectors with filters.

```go
vector, err := i.ollama.Embed(ctx, model, query)
hits, err := i.qdrant.searchPoints(ctx, collectionName, vector, topK, filters)
```

### `internal/embedding/index_test.go`

Tests embedding index setup, collection naming, Qdrant collection behavior, point upserts, search behavior, and error cases with fake transports.

### `internal/memory/store.go`

SQLite memory store plus vector-index integration. It saves summaries to Qdrant first, then stores full memory rows in SQLite. It can list recent card memories and perform semantic search.

```go
collectionName, err := s.index.UpsertDocuments(ctx, s.embeddingModel, []embedding.Document{document})
```

### `internal/memory/store_test.go`

Tests memory saving, listing, search, validation, and vector-index integration using a fake vector index.

### `internal/profile/store.go`

SQLite user profile and fact store. It manages users, fact candidates/accepted facts, profile summaries, fact parsing, and resets.

```go
if update.Confidence >= acceptedConfidenceCutoff {
	status = StatusAccepted
}
```

### `internal/profile/store_test.go`

Tests profile creation, fact update parsing/application, summary rebuilding, reset behavior, sanitization, and validation.

### `internal/ollama/client.go`

Low-level Ollama HTTP client. It supports chat, detailed chat diagnostics, embeddings, detailed embedding diagnostics, and model-list parsing via `ollama list`.

```go
req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
```

### `internal/ollama/client_test.go`

Tests Ollama chat/embed request generation, response parsing, status/error handling, and model list parsing.

### `internal/ollama/models_test.go`

Tests Ollama model listing behavior, command errors, parsing, and timeout/error paths.

## Web Host Files

### `internal/web/handlers.go`

Registers all HTTP routes and owns host-level routing. It creates the component registry, serves frontend assets, card APIs, card canvas fragments, component patch proposal APIs, memory APIs, and profile APIs.

```go
registry := component.MustNewRegistry(
	appheader.Definition(),
	chatform.Definition(),
)
```

Important routes:

```text
GET  /
GET  /assets/app.js
GET  /api/cards
GET  /api/cards/{cardID}
GET  /api/cards/{cardID}/canvas
GET  /api/cards/{cardID}/memories?user_id=...
POST /api/cards/{cardID}/components/{componentID}/actions/{action}
POST /api/components/{componentType}/patch-proposals
GET  /api/users/{userID}/profile
DELETE /api/users/{userID}/profile
```

### `internal/web/page.go`

Renders the server-side page shell. It includes the app header, card list mount, dynamic `card-canvas` mount, debug panels, and script tag for `/assets/app.js`.

```go
Div(Id("card-canvas"))
Script(Type("module"), Src("/assets/app.js"))
```

### `internal/web/ui.go`

Shared page head, Tailwind/browser setup, CSS variables, and legacy shared UI class helpers.

```go
Script(Src("https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"))
Style(T(sharedPageCSS() + extraCSS))
```

### `internal/web/handlers_test.go`

Tests all major web behavior: card APIs, canvas rendering, component action dispatch, old route removal, profile APIs, asset serving, patch proposal endpoint, and page mounts.

## Component Registry And Patch Files

### `internal/web/component/component.go`

Defines web dependency interfaces, component definitions, action dispatch, registry rendering, context-file metadata, and shared UI class helpers for components.

```go
func (r *Registry) HandleCardResource(w http.ResponseWriter, req *http.Request, deps Dependencies, card cards.Card, parts []string) bool {
	instance, ok := findComponent(card.Components, parts[2])
	definition, ok := r.Definition(instance.Type)
	return definition.HandleAction(w, req, deps, card, instance, parts[4])
}
```

### `internal/web/component/patch.go`

Builds safe component context bundles and asks Ollama for patch proposals. It prevents absolute paths and parent-directory escapes.

```go
if filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
	return "", fmt.Errorf("invalid context file path %q", name)
}
```

### `internal/web/component/component_test.go`

Tests context file cloning, registry action dispatch, ordered component rendering, wrapper metadata, patch context file inclusion, unknown component errors, and empty instruction errors.

## Web Components

### `internal/web/components/appheader/view.go`

GoDom view for the top app header. It renders the title, subtitle, and reload button.

```go
Button(
	Id("reload-cards-btn"),
	Type("button"),
	Class(component.SecondaryButtonClass("sm")),
	T("Reload Cards"),
)
```

### `internal/web/components/appheader/handlers.go`

Defines the `app-header` component registry entry, context files, and client initializer name. It has no action routes yet.

```go
return component.Definition{
	Type: "app-header",
	ClientInitializer: "initAppHeader",
	ContextFiles: []string{...},
}
```

### `internal/web/components/appheader/client.ts`

Browser initializer for the header. It binds the reload button to the app's `loadCards` callback.

```ts
reload.addEventListener("click", () => {
  void deps.loadCards();
});
```

### `internal/web/components/chatform/view.go`

GoDom view for the conversation component. It renders status, card metadata, transcript, textarea, and send button.

```go
Form(
	Id("chat-form"),
	TextArea(Id("chat-input"), Name("message")),
	Button(Id("send-btn"), Type("submit"), T("Send")),
)
```

### `internal/web/components/chatform/handlers.go`

Defines the `chat-form` registry entry and handles the `send` action for:

```text
POST /api/cards/{cardID}/components/chat-form/actions/send
```

It decodes `{ user_id, message }`, calls `deps.Chat.Chat`, and returns `chat.Result`.

### `internal/web/components/chatform/client.ts`

Browser behavior for the chat form. It owns submit handling, transcript rendering, card metadata rendering, and status updates.

```ts
const payload = await chatDeps.sendChat(cardId, chatDeps.getUserId(), message);
chatDeps.appendTranscript(cardId, {
  user: message,
  assistant: payload.assistant_response || "",
});
```

### `internal/web/components/chatform/types.ts`

Re-exports shared frontend types so the chat form component can keep type usage local.

```ts
export type { Card, ChatResult, Memory, SearchResult, TranscriptItem } from "../../../../web/src/types";
```

## Frontend Files

### `web/src/app.ts`

Frontend bootstrap. It loads cards, renders the card selector, fetches selected card metadata, fetches the server-rendered canvas fragment, hydrates component behavior, and updates debug panels.

```ts
canvas.innerHTML = await fetchCardCanvas(cardId);
hydrateCardCanvas(canvas, hydrationDeps());
```

### `web/src/api.ts`

Typed fetch helpers for cards, canvas HTML, memories, and chat send actions.

```ts
export async function fetchCardCanvas(cardId: string): Promise<string> {
  const response = await fetch("/api/cards/" + encodeURIComponent(cardId) + "/canvas", { cache: "no-store" });
  return await response.text();
}
```

### `web/src/components.ts`

Client-side component hydration registry. It maps server-rendered `data-component-type` values to TypeScript initializers.

```ts
const initializers: Record<string, Initializer> = {
  "chat-form": initChatForm,
};
```

### `web/src/dom.ts`

Small DOM helpers for element lookup and HTML escaping.

```ts
export function byID<T extends HTMLElement>(id: string): T | null {
  return document.getElementById(id) as T | null;
}
```

### `web/src/state.ts`

In-memory browser state. Tracks loaded cards, selected card, local user ID, and local transcripts.

```ts
export const livingCardState: LivingCardState = {
  cards: [], selectedCardId: "", userId: "local-user", transcripts: {},
};
```

### `web/src/types.ts`

Frontend API/domain TypeScript interfaces for cards, component instances, memories, search results, chat results, and transcript rows.

```ts
export interface ComponentInstance {
  id: string;
  type: string;
  props?: Record<string, unknown>;
}
```

### `web/dist/app.js`

Generated JavaScript bundle produced by `internal/webbuild`. It is served at `/assets/app.js`. Do not edit by hand.

### `web/dist/app.js.map`

Generated source map for `web/dist/app.js`. It is served at `/assets/app.js.map`. Do not edit by hand.

## Build And Infrastructure Files

### `internal/webbuild/build.go`

Uses esbuild's Go API to bundle `web/src/app.ts` into `web/dist/app.js` with a linked sourcemap.

```go
api.Build(api.BuildOptions{
	EntryPoints: []string{"web/src/app.ts"},
	Bundle: true,
	Outfile: "web/dist/app.js",
	Format: api.FormatESModule,
})
```

### `internal/webbuild/build_test.go`

Runs the frontend build and verifies `web/dist/app.js` and `web/dist/app.js.map` exist.

### `docker-compose.yml`

Defines local Ollama and Qdrant services.

```yaml
services:
  ollama:
    image: ollama/ollama:latest
  qdrant:
    image: qdrant/qdrant:latest
```

### `go.mod`

Go module definition and direct dependencies:

```go
require (
	github.com/evanw/esbuild v0.28.1
	github.com/n0remac/GoDom ...
	modernc.org/sqlite v1.50.0
)
```

### `go.sum`

Go dependency checksums. It is managed by Go tooling.

## Tests By Package

Most packages have focused unit tests:

```text
internal/cards          static card catalog and clone safety
internal/chat           prompt building and chat/memory behavior
internal/config         env parsing
internal/embedding      Qdrant/Ollama embedding adapter
internal/memory         SQLite memory plus vector index
internal/ollama         HTTP client and model list parsing
internal/profile        facts/profile summary store
internal/web            HTTP routes, canvas, assets, patch proposals
internal/web/component  registry rendering and patch context behavior
internal/webbuild       frontend bundle generation
```

Recommended verification command:

```sh
go test ./...
```

## Important Design Constraints

- Cards are static Go code, not JSON files.
- The current dynamic canvas is server-rendered with GoDom.
- Browser code hydrates server-rendered component HTML.
- Component API actions are routed through the registry.
- Ollama patch proposals are review-only and do not write files.
- Component context for AI edits is limited to each component's `ContextFiles`.
