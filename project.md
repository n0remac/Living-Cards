# Living Card architecture

Living Card is a Go web app for experimenting with collectible cards as structured, server-rendered data. The visible app is a small world-deck game. Card and deck state is process-local, while the source content is strict embedded JSON under `internal/game/decks/`.

## Component model

`internal/components/card` owns the component language runtime:

- `Document`, `Node`, and `ComponentTemplate`.
- `TypedDefinition[T]` and its erased runtime `Definition`.
- The immutable, ordered `Registry`.
- Strict config, template, node, and document codecs.
- Rendering, controls, properties, roles, presets, generation metadata, and installation policies.

`internal/components/schema` owns shared validation issues, typed property values, safe CSS helpers, and generation envelopes. It does not depend on component implementations or application packages.

Each leaf package owns its config type and one `Definition()` containing all supported backend mechanics. `internal/components/catalog` is the only production catalog composition point. It explicitly registers:

```text
card
background
border
text
shape
image
slider
text_input
button
```

`heading`, `form`, `stack`, and `grid` are reserved for later work. The current `card` definition is the sole root; every other registered component is a leaf.

Adding a backend component requires only:

1. Implementing its typed `Definition()` in a leaf package.
2. Adding that definition once to `internal/components/catalog`.

Every applicable backend consumer then discovers it through the erased registry API. Game, web, rendering, editing, installation, inspection, presets, and generation do not import or decode concrete leaf config types.

## Dependency direction

```text
components/schema
      ↑
components/card
      ↑
leaf component packages
      ↑
components/catalog

design → components/card + components/schema
game/web → components/card; catalog injected at composition boundaries
```

There is no global registration, `init()` registration, or secondary component registry. `main.go` builds one catalog and injects it into the web and game composition roots.

## Strict component JSON

Component-language fields and config/control IDs use snake case. A node uses `component_kind`; generated config uses the same field. Legacy `textarea`, `textinput`, camel-case config fields, and control aliases are rejected.

Config decoding has these semantics:

```text
config omitted → apply typed defaults
config: {}      → apply typed defaults
config: null    → error
unknown field   → error
explicit zero   → preserve, then validate
explicit ""     → preserve, then validate
invalid range   → error, never clamp
enum casing     → exact canonical value required
```

Decoding begins with a fresh typed default, overlays explicitly authored fields, performs non-lossy normalization such as permitted whitespace trimming, validates, and emits canonical JSON. Normalizers may not repair or replace invalid authored values.

Documents reject unknown fields, duplicate or invalid component IDs, unknown kinds, nested roots, and children on leaves. Validation still walks invalid subtrees so errors are not hidden behind a structural failure. Rendering currently remains intentionally direct-child-only; recursive containers are future work.

## Capabilities

Registration does not imply that every component supports every optional operation.

- All definitions provide defaults, validation, and rendering.
- Definitions with controls are editable through both designer and game editor paths.
- Typed properties expose primitive `string`, `number`, or `bool` values without concrete config assertions. Slider rules read `value` this way.
- `text_input` and `button` declare form-field and form-submitter roles separately from their readable properties.
- Installation is declared as `append` or `replace_kind`.
- Presets and random generation are ordered definition metadata.
- AI metadata is active only for background, border, text, and image. Shape retains random generation but has no AI route.

Unsupported optional operations return an explicit unsupported issue or `card.ErrUnsupportedOperation`; consumers do not silently fall back.

## Component templates and installation

Installable component cards store a single canonical template under `component_template`:

```json
{
  "component_template": {
    "component_kind": "slider",
    "component_id": "optional-explicit-id",
    "config": { "value": 50 }
  }
}
```

Installation strictly decodes the template, resolves the definition, canonicalizes its typed config, validates or allocates the ID, applies the declared policy, and inserts the node. Generated IDs use `<card-id>-<kebab-kind>`, followed by `-2`, `-3`, and so on. Explicit IDs are preserved exactly and rejected on collision. `replace_kind` inserts when absent, replaces one match while retaining its ID by default, and rejects ambiguous multiple matches.

## Generation boundary

`internal/design` is transport-oriented and non-generic. It receives an erased `card.Definition`, asks the model for a strict envelope, and delegates config decoding and canonicalization back to the definition:

```text
LLM raw response
→ strict generated envelope
→ exact registered kind
→ erased typed config codec
→ canonical generated envelope
```

The registry has no dependency on Ollama, HTTP handlers, or model clients. Repair prompts consume shared `schema.Issue` values.

## World deck runtime

The game loads the seeded world plus fuse-room, generator-room, and archive-terminal packs from embedded JSON. Deck decoding rejects unknown fields and runs every card document through the component registry. One strictly tagged rule model handles card plays, form submissions, and world-component updates. Rules can inspect typed registry properties, discover form participants through roles, install component templates, and render cards without leaf-package imports.

Rules retain loaded declaration order: a newly loaded deck appends its rules after the current session rules. Each internal signal resolves at most one rule, applies its effects sequentially, and queues component-install follow-ups after the effect list. Queued signals resolve cards from current session state while retaining a canonical snapshot of the component that caused an update. Processing is capped at 32 signals so an accidental component-rule loop fails the command transactionally.

Collecting a card moves it from the world deck into the library. A play against a matching target consumes the library card after its success or failure effects complete. Component cards are editable bases: editing installs their own template into the draft, and saving converts the base into an editable controller while consuming any added component cards.

The main game endpoints are under `/api/game/*`. Draft/designer endpoints are under `/api/draft-card/*`. The frontend bundle is built from `web/src/app.ts` by `internal/webbuild` and committed under `web/dist/`.

### Game mutation boundary

The world-deck runtime exposes two session operations: `View` for a
non-mutating read at the current revision and `Execute` for typed commands.
HTTP routes retain their resource-oriented paths, strictly decode their
existing request shapes, construct commands, and return:

```text
HTTP request
→ typed command
→ Session.Execute
→ state mutation
→ ordered semantic events
→ revisioned domain snapshot
→ rendered HTTP snapshot
```

Before dispatch, `Execute` clones every mutable gameplay field and runs the
command against that working state while holding the session lock. Invalid
commands, internal rule failures, and domain snapshot failures restore the
previous state, discard events, and leave the revision unchanged. A valid
command increments the revision once even when game conditions reject the
action. Reset restores the seeded gameplay state without restarting the
revision sequence.

Events are command-local and transient. They are emitted at mutation time,
including declarative rule effects, and use contiguous sequence numbers
beginning at zero. The diagnostic `ruleResolved` event is available to tests,
logs, and the frontend but does not require visible presentation. Events are
not an event store and do not support replay or undo. The web renderer runs after
the domain transaction commits; a response-rendering or connection failure
cannot safely roll back an already committed session.

## Deferred work

- Recursive rendering and real container child policies.
- Heading, form, stack, and grid components.
- Document versioning and migrations.
- Generated, committed TypeScript types and frontend metadata from the Go catalog.
- Persistent database-backed documents and sessions.

## Development

Run all backend and frontend build tests with:

```sh
go test ./...
```

Run locally with `make restart`; use `make status`, `make logs`, and `make stop` for process management.
