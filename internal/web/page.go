package web

import (
	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

func Page() *Node {
	return Html(
		Attr("data-theme", "dark"),
		pageHead("Living Card", pageCSS()),
		Body(
			Class("bg-[var(--app-bg)] text-[var(--app-fg)]"),
			Main(
				Id("living-card-stage"),
				Class("living-card-stage"),
				Div(Class("living-card-atmosphere")),
				gameStageView(),
				Div(
					Id("stage-overlay-root"),
					Class("stage-overlay-root pointer-events-none fixed inset-0 z-30"),
					Attr("aria-live", "polite"),
					stageEdgeControlsView(),
				),
				controllerBuilderView(),
				designerOverlayView(),
			),
			Script(Type("module"), Src("/assets/app.js")),
		),
	)
}

func gameStageView() *Node {
	return Div(
		Id("card-workspace"),
		Attr("data-game-stage", ""),
		Class("game-stage-shell"),
		Div(
			Class("game-topbar"),
			Div(
				Class("game-title-block"),
				H1(Class("game-title"), T("Living Card")),
				Div(Id("game-progress"), Class("game-progress"), T("0 cards collected")),
			),
			Button(
				Id("reset-draft-btn"),
				Type("button"),
				Class(uiSecondaryButtonClass("sm")),
				T("Reset"),
			),
		),
		Div(
			Class("game-board"),
			Button(Id("game-prev-card"), Type("button"), Class("game-cycle-button"), Attr("aria-label", "Previous card"), T("‹")),
			Div(
				Class("game-active-column"),
				Div(Id("game-world-card"), Class("game-world-card"), Attr("aria-live", "polite"),
					Div(Class("game-loading-card"), T("Loading card...")),
				),
				Div(
					Class("game-action-row"),
					Div(Id("game-status"), Class("game-status"), T("Loading scene...")),
					Button(Id("game-collect-card"), Type("button"), Class(uiPrimaryButtonClass("sm")), T("Collect")),
				),
			),
			Button(Id("game-next-card"), Type("button"), Class("game-cycle-button"), Attr("aria-label", "Next card"), T("›")),
		),
		Div(
			Class("game-library-panel"),
			Div(
				Class("game-library-header"),
				H2(Class("game-library-title"), T("Library")),
				Span(Id("game-library-count"), Class("game-library-count"), T("Empty")),
			),
			Div(Id("game-library-list"), Class("game-library-list"), T("No cards collected.")),
		),
	)
}

func stageHUDView() *Node {
	return Div(
		Id("stage-hud"),
		Class("stage-hud"),
		Div(
			Class("stage-xp-panel"),
			Div(
				Class("stage-xp-stats"),
				Span(Id("card-level"), T("Lv 1")),
				Span(Id("card-xp"), T("0 XP")),
				Span(Id("card-taps"), T("0 taps")),
			),
			Div(
				Class("stage-xp-track"),
				Div(Id("card-xp-bar"), Class("stage-xp-fill")),
			),
		),
		Button(
			Id("stage-notification-section"),
			Type("button"),
			Attr("aria-live", "polite"),
			Class("stage-notification-section"),
			Span(Id("stage-notification-current"), T("No notifications")),
		),
		Div(
			Id("stage-notification-history"),
			Class("stage-notification-history hidden"),
			Div(Class("stage-notification-history-title"), T("Notifications")),
			Div(Id("stage-notification-history-list"), Class("stage-notification-history-list")),
		),
	)
}

func stageResetView() *Node {
	return Div(
		Class("stage-reset pointer-events-none fixed right-3 top-3 z-20 sm:right-5 sm:top-5"),
		Button(
			Id("reset-draft-btn"),
			Type("button"),
			Class(uiSecondaryButtonClass("sm")+" pointer-events-auto"),
			T("Reset"),
		),
	)
}

func stageEdgeControlsView() *Node {
	return Div(
		Id("stage-edge-controls"),
		Class("stage-edge-controls"),
		Div(Id("stage-edge-controls-top"), Class("stage-edge-controls-top")),
		Div(
			Class("stage-edge-controls-sides"),
			Div(Id("stage-edge-controls-left"), Class("stage-edge-controls-rail")),
			Div(Class("stage-edge-controls-spacer")),
			Div(Id("stage-edge-controls-right"), Class("stage-edge-controls-rail")),
		),
		Div(
			Id("stage-edge-controls-bottom"),
			Class("stage-edge-controls-bottom"),
			Div(Id("stage-edge-controls-status"), Class("stage-edge-controls-status"), T("")),
		),
	)
}

func controllerBuilderView() *Node {
	return Div(
		Id("controller-builder-overlay"),
		Class("controller-builder-overlay"),
		Attr("role", "dialog"),
		Attr("aria-modal", "true"),
		Attr("aria-labelledby", "controller-builder-title"),
		Div(
			Class("controller-builder-panel"),
			Div(
				Class("controller-builder-header"),
				H2(Id("controller-builder-title"), Class("controller-builder-title"), T("Regulator Controller")),
				Button(Id("controller-builder-close"), Type("button"), Class(uiSecondaryButtonClass("xs")), T("Close")),
			),
			Div(
				Class("controller-builder-body"),
				Label(Attr("for", "controller-slider-input"), Class("controller-builder-label"), T("Output")),
				Div(
					Class("controller-builder-value-row"),
					Input(
						Id("controller-slider-input"),
						Type("range"),
						Attr("min", "0"),
						Attr("max", "100"),
						Attr("step", "1"),
						Value("50"),
						Class("controller-builder-range"),
					),
					Input(
						Id("controller-slider-number"),
						Type("number"),
						Attr("min", "0"),
						Attr("max", "100"),
						Attr("step", "1"),
						Value("50"),
						Class("controller-builder-number"),
					),
				),
			),
			Div(
				Class("controller-builder-actions"),
				Button(Id("controller-builder-cancel"), Type("button"), Class(uiSecondaryButtonClass("sm")), T("Cancel")),
				Button(Id("controller-builder-save"), Type("button"), Class(uiPrimaryButtonClass("sm")), T("Save Controller")),
			),
		),
	)
}

func designerOverlayView() *Node {
	return Div(
		Id("designer-overlay"),
		Class("fixed inset-0 z-40 hidden bg-black/50 p-3 backdrop-blur-sm sm:p-5"),
		Div(
			Class("ml-auto flex h-full w-full max-w-md flex-col rounded-md border border-[var(--app-border-strong)] bg-[var(--app-surface-muted)] shadow-2xl"),
			Div(
				Class("flex items-start justify-between gap-3 border-b border-[var(--app-border)] p-4"),
				Div(
					H2(Class("text-sm font-semibold uppercase text-[var(--app-fg-soft)]"), T("Designer")),
				),
				Button(
					Id("designer-close-btn"),
					Type("button"),
					Class(uiSecondaryButtonClass("xs")),
					T("Close"),
				),
			),
			Div(
				Class("min-h-0 flex-1 overflow-y-auto p-4"),
				designerControlsView(),
			),
		),
	)
}

func designerControlsView() *Node {
	return Div(
		Div(
			Class("flex justify-end"),
			Button(
				Id("designer-reset-btn"),
				Type("button"),
				Class(uiSecondaryButtonClass("xs")),
				T("Reset Draft"),
			),
		),
		Form(
			Id("card-designer-form"),
			Class("mt-5 space-y-4"),
			Div(
				Class("space-y-2"),
				Label(Attr("for", "fragment-target"), Class("block text-[0.72rem] font-semibold uppercase text-[var(--app-fg-soft)]"), T("Target")),
				Select(
					Id("fragment-target"),
					Name("target"),
					Class(uiInputClass()),
					Option(Value(background.Type), T("Background")),
					Option(Value(border.Type), T("Border")),
					Option(Value(textarea.Type), T("Text area")),
					Option(Value(imagecomponent.Type), T("Image")),
				),
			),
			Div(
				Class("space-y-2"),
				Label(Attr("for", "fragment-instruction"), Class("block text-[0.72rem] font-semibold uppercase text-[var(--app-fg-soft)]"), T("Instruction")),
				TextArea(
					Id("fragment-instruction"),
					Name("instruction"),
					Class(uiInputClass()+" min-h-28 resize-y"),
					Placeholder("Describe the fragment you want..."),
				),
			),
			Div(
				Class("flex flex-wrap gap-2"),
				Button(
					Id("generate-fragment-btn"),
					Type("submit"),
					Name("intent"),
					Value("generate"),
					Class(uiPrimaryButtonClass("sm")),
					T("Generate"),
				),
				Button(
					Id("update-fragment-btn"),
					Type("submit"),
					Name("intent"),
					Value("update"),
					Class(uiSecondaryButtonClass("sm")),
					T("Update"),
				),
				Button(
					Id("apply-fragment-btn"),
					Type("button"),
					Attr("disabled", "disabled"),
					Class(uiSecondaryButtonClass("sm")),
					T("Apply"),
				),
			),
		),
		Div(Id("designer-status"), Class("mt-4 text-sm text-[var(--app-fg-soft)]"), T("Ready.")),
		Div(
			Class("mt-5 space-y-3 border-t border-[var(--app-border)] pt-5"),
			Div(
				Class("flex items-center justify-between gap-3"),
				H3(Class("text-[0.72rem] font-semibold uppercase text-[var(--app-fg-soft)]"), T("Library")),
				Button(
					Id("save-design-btn"),
					Type("button"),
					Attr("disabled", "disabled"),
					Class(uiSecondaryButtonClass("xs")),
					T("Save Applied"),
				),
			),
			Div(Id("design-library-list"), Class("grid gap-2")),
		),
		Div(
			Class("mt-5 space-y-3 border-t border-[var(--app-border)] pt-5"),
			H3(Class("text-[0.72rem] font-semibold uppercase text-[var(--app-fg-soft)]"), T("Add component")),
			Div(
				Class("flex flex-wrap gap-2"),
				Button(Id("add-textarea-component-btn"), Type("button"), Class(uiSecondaryButtonClass("xs")), T("Text")),
				Button(Id("add-shape-component-btn"), Type("button"), Class(uiSecondaryButtonClass("xs")), T("Shape")),
				Label(
					Class(uiSecondaryButtonClass("xs")+" cursor-pointer"),
					T("Image"),
					Input(Id("add-image-component-input"), Type("file"), Attr("accept", "image/png,image/jpeg,image/webp,image/gif"), Class("hidden")),
				),
			),
		),
		Div(
			Class("mt-5 space-y-2"),
			H3(Class("text-[0.72rem] font-semibold uppercase text-[var(--app-fg-soft)]"), T("Generated fragment")),
			P(Id("fragment-description"), Class("text-sm text-[var(--app-fg-muted)]"), T("No generated fragment yet.")),
			TextArea(
				Id("fragment-preview"),
				Attr("spellcheck", "false"),
				Class(uiInputClass()+" min-h-72 resize-y font-mono text-xs leading-5"),
				T("{}"),
			),
		),
	)
}

func pageCSS() string {
	return `
.living-card-stage {
  position: fixed;
  inset: 0;
  width: 100vw;
  height: 100dvh;
  overflow: auto;
  display: block;
  isolation: isolate;
}

.living-card-atmosphere {
  position: absolute;
  inset: 0;
  z-index: -1;
  background:
    radial-gradient(circle at 18% 18%, rgba(16, 185, 129, 0.18), transparent 28%),
    radial-gradient(circle at 82% 72%, rgba(245, 158, 11, 0.14), transparent 30%),
    linear-gradient(145deg, #111111 0%, #1f2933 48%, #271c30 100%);
}

.game-stage-shell {
  min-height: 100dvh;
  width: min(100%, 76rem);
  margin: 0 auto;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 1rem;
  padding: 1rem;
}

.game-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.game-title-block {
  min-width: 0;
}

.game-title {
  margin: 0;
  font-size: clamp(1.1rem, 2.5vw, 1.7rem);
  font-weight: 800;
  letter-spacing: 0;
}

.game-progress {
  margin-top: 0.2rem;
  color: rgba(244, 244, 245, 0.62);
  font-size: 0.82rem;
  font-weight: 650;
}

.game-board {
  min-height: 0;
  display: grid;
  grid-template-columns: 3rem minmax(0, 1fr) 3rem;
  gap: 1rem;
  align-items: center;
}

.game-active-column {
  min-width: 0;
  display: grid;
  justify-items: center;
  gap: 0.75rem;
}

.game-world-card {
  width: min(86vw, 27rem, calc((100dvh - 14rem) * 5 / 7));
}

.game-world-card > [data-card-id] {
  width: 100%;
  max-width: none;
  color: var(--app-fg);
}

.game-loading-card {
  aspect-ratio: 5 / 7;
  display: grid;
  place-items: center;
  border: 1px solid rgba(255,255,255,0.16);
  border-radius: 1rem;
  background: rgba(15,23,42,0.72);
  color: rgba(244,244,245,0.7);
  font-weight: 700;
}

.game-cycle-button {
  width: 3rem;
  height: 3rem;
  border-radius: 0.45rem;
  border: 1px solid rgba(255,255,255,0.18);
  background: rgba(9,9,11,0.66);
  color: rgba(244,244,245,0.9);
  font-size: 2rem;
  font-weight: 800;
  line-height: 1;
  cursor: pointer;
}

.game-cycle-button:hover {
  border-color: rgba(52,211,153,0.45);
  background: rgba(16,185,129,0.16);
}

.game-action-row {
  width: min(86vw, 27rem);
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 0.75rem;
  align-items: center;
}

.game-status {
  min-height: 2.25rem;
  display: flex;
  align-items: center;
  border: 1px solid rgba(255,255,255,0.12);
  border-radius: 0.45rem;
  background: rgba(9,9,11,0.62);
  padding: 0.5rem 0.7rem;
  color: rgba(244,244,245,0.78);
  font-size: 0.84rem;
  font-weight: 650;
}

.game-library-panel {
  display: grid;
  gap: 0.65rem;
  border-top: 1px solid rgba(255,255,255,0.12);
  padding-top: 0.85rem;
}

.game-library-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.game-library-title {
  margin: 0;
  font-size: 0.78rem;
  font-weight: 800;
  text-transform: uppercase;
  color: rgba(244,244,245,0.72);
}

.game-library-count {
  color: rgba(244,244,245,0.52);
  font-size: 0.76rem;
  font-weight: 700;
}

.game-library-list {
  min-height: 7rem;
  display: flex;
  gap: 0.75rem;
  overflow-x: auto;
  padding-bottom: 0.25rem;
  color: rgba(244,244,245,0.52);
  font-size: 0.85rem;
}

.game-library-card {
  flex: 0 0 8rem;
  display: grid;
  gap: 0.35rem;
  border: 0;
  background: transparent;
  color: inherit;
  padding: 0;
  text-align: left;
  cursor: grab;
}

.game-library-card:active {
  cursor: grabbing;
}

.game-library-card [data-card-id] {
  width: 8rem;
  max-width: none;
  pointer-events: none;
}

.game-library-card-name {
  color: rgba(244,244,245,0.82);
  font-size: 0.76rem;
  font-weight: 700;
}

.game-library-build {
  min-height: 2rem;
  border-radius: 0.45rem;
  border: 1px solid rgba(125, 211, 252, 0.36);
  background: rgba(14, 165, 233, 0.14);
  color: rgba(224, 242, 254, 0.95);
  font-size: 0.72rem;
  font-weight: 800;
  cursor: pointer;
}

.game-library-build:disabled {
  cursor: not-allowed;
  opacity: 0.52;
}

.controller-builder-overlay {
  position: fixed;
  inset: 0;
  z-index: 45;
  display: none;
  place-items: center;
  padding: 1rem;
  background: rgba(0,0,0,0.58);
  backdrop-filter: blur(10px);
}

.controller-builder-open .controller-builder-overlay {
  display: grid;
}

.controller-builder-panel {
  width: min(100%, 28rem);
  display: grid;
  gap: 0;
  overflow: hidden;
  border: 1px solid rgba(212,212,216,0.28);
  border-radius: 0.5rem;
  background: rgba(24,24,27,0.98);
  box-shadow: 0 1.5rem 4rem rgba(0,0,0,0.46);
}

.controller-builder-header,
.controller-builder-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 1rem;
}

.controller-builder-header {
  border-bottom: 1px solid rgba(255,255,255,0.12);
}

.controller-builder-title {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 800;
  letter-spacing: 0;
}

.controller-builder-body {
  display: grid;
  gap: 0.8rem;
  padding: 1rem;
}

.controller-builder-label {
  color: rgba(244,244,245,0.72);
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
}

.controller-builder-value-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 5rem;
  gap: 0.75rem;
  align-items: center;
}

.controller-builder-range {
  width: 100%;
  accent-color: #7dd3fc;
}

.controller-builder-number {
  width: 100%;
  height: 2.5rem;
  border: 1px solid rgba(212,212,216,0.26);
  border-radius: 0.45rem;
  background: rgba(9,9,11,0.7);
  color: rgba(244,244,245,0.95);
  font-size: 1rem;
  font-weight: 800;
  text-align: center;
}

.controller-builder-actions {
  justify-content: flex-end;
  border-top: 1px solid rgba(255,255,255,0.12);
}

@media (max-width: 42rem) {
  .game-stage-shell {
    padding: 0.75rem;
  }

  .game-board {
    grid-template-columns: 2.5rem minmax(0, 1fr) 2.5rem;
    gap: 0.5rem;
  }

  .game-cycle-button {
    width: 2.5rem;
    height: 2.5rem;
  }

  .game-action-row {
    grid-template-columns: 1fr;
  }
}

.stage-card-wrap {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  padding: 1rem;
  perspective: 1000px;
  touch-action: manipulation;
}

.stage-controls-open .stage-hud,
.stage-controls-open .stage-reset {
  visibility: hidden;
  opacity: 0;
  pointer-events: none;
}

.stage-card-stack {
  width: min(94vw, 34rem, calc((100dvh - 6.5rem) * 5 / 7));
  display: grid;
  gap: 0;
}

.stage-hud {
  position: relative;
  z-index: 2;
  width: 100%;
  display: grid;
  gap: 0.45rem;
  pointer-events: none;
}

.stage-xp-panel {
  display: grid;
  gap: 0.35rem;
  width: 100%;
  padding: 0 0 0.45rem;
}

.stage-xp-stats {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  color: rgba(244, 244, 245, 0.78);
  font-size: 0.72rem;
  font-weight: 650;
  text-transform: uppercase;
}

.stage-xp-track {
  height: 0.2rem;
  width: 100%;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.16);
}

.stage-xp-fill {
  height: 100%;
  width: 0;
  border-radius: inherit;
  background: #34d399;
  transition: width 220ms ease;
}

.stage-notification-section {
  pointer-events: auto;
  min-height: 2rem;
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 0.45rem;
  background: rgba(9, 9, 11, 0.58);
  padding: 0.45rem 0.6rem;
  color: rgba(244, 244, 245, 0.84);
  font-size: 0.76rem;
  font-weight: 650;
  text-align: left;
  backdrop-filter: blur(10px);
}

.stage-notification-section[data-tone="info"] {
  border-color: rgba(52, 211, 153, 0.28);
  color: rgba(236, 253, 245, 0.94);
}

.stage-notification-section[data-tone="error"] {
  border-color: rgba(251, 191, 36, 0.3);
  color: rgba(254, 243, 199, 0.95);
}

.stage-notification-section[data-tone="empty"] {
  color: rgba(244, 244, 245, 0.44);
}

.stage-notification-history {
  pointer-events: auto;
  position: absolute;
  top: calc(100% + 0.4rem);
  left: 0;
  right: 0;
  z-index: 5;
  max-height: min(16rem, 38dvh);
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 0.45rem;
  background: rgba(12, 12, 14, 0.94);
  color: rgba(244, 244, 245, 0.88);
  box-shadow: 0 1.25rem 3rem rgba(0, 0, 0, 0.36);
  backdrop-filter: blur(16px);
}

.stage-notification-history-title {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0.55rem 0.7rem;
  font-size: 0.68rem;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
  color: rgba(244, 244, 245, 0.62);
}

.stage-notification-history-list {
  display: grid;
  max-height: calc(min(16rem, 38dvh) - 2rem);
  overflow-y: auto;
}

.stage-notification-history-item {
  padding: 0.6rem 0.7rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 0.8rem;
  line-height: 1.3;
}

.stage-notification-history-item[data-tone="error"] {
  color: rgba(254, 243, 199, 0.95);
}

.stage-notification-history-empty {
  padding: 0.7rem;
  color: rgba(244, 244, 245, 0.5);
  font-size: 0.8rem;
}

.stage-card-wrap #draft-card-preview {
  width: 100%;
  max-width: none;
  cursor: pointer;
  transform-origin: center;
  will-change: transform;
  color: var(--app-fg);
}

.stage-edge-controls {
  --stage-card-width: min(94vw, 34rem, calc((100dvh - 6.5rem) * 5 / 7));
  position: absolute;
  inset: 0;
  display: none;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 0.75rem;
  padding: 0.75rem;
  pointer-events: none;
}

.stage-controls-open .stage-edge-controls {
  display: grid;
}

.stage-edge-controls-top,
.stage-edge-controls-bottom {
  pointer-events: auto;
  width: var(--stage-card-width);
  max-width: calc(100vw - 1.5rem);
  justify-self: center;
}

.stage-edge-controls-sides {
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(12rem, 1fr) var(--stage-card-width) minmax(12rem, 1fr);
  gap: 0.75rem;
  align-items: center;
}

.stage-edge-controls-spacer {
  pointer-events: none;
}

.stage-edge-controls-rail {
  pointer-events: auto;
  display: grid;
  gap: 0.65rem;
  align-content: start;
  max-height: min(34rem, calc(100dvh - 8rem));
  overflow-y: auto;
  scrollbar-width: thin;
}

.stage-edge-panel {
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 0.45rem;
  background: rgba(9, 9, 11, 0.76);
  color: rgba(244, 244, 245, 0.9);
  box-shadow: 0 1rem 2.4rem rgba(0, 0, 0, 0.32);
  backdrop-filter: blur(16px);
}

.stage-edge-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.55rem 0.65rem;
}

.stage-edge-title {
  min-width: 0;
  font-size: 0.78rem;
  font-weight: 700;
  color: rgba(244, 244, 245, 0.92);
}

.stage-edge-subtitle {
  margin-top: 0.1rem;
  font-size: 0.68rem;
  font-weight: 650;
  color: rgba(244, 244, 245, 0.52);
  text-transform: uppercase;
}

.stage-edge-controls-group {
  display: grid;
  gap: 0.6rem;
  padding: 0.65rem;
}

.stage-edge-controls-empty {
  padding: 0.7rem;
  font-size: 0.78rem;
  color: rgba(244, 244, 245, 0.58);
}

.stage-edge-controls-status {
  min-height: 2rem;
  padding: 0.5rem 0.65rem;
  font-size: 0.76rem;
  font-weight: 650;
  color: rgba(244, 244, 245, 0.66);
}

.stage-edge-controls-status[data-tone="error"] {
  color: rgba(254, 243, 199, 0.95);
}

@media (max-width: 58rem) {
  .stage-edge-controls {
    padding: 0.55rem;
    gap: 0.5rem;
  }

  .stage-edge-controls-sides {
    align-self: end;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 0.5rem;
  }

  .stage-edge-controls-spacer {
    display: none;
  }

  .stage-edge-controls-rail {
    max-height: min(14rem, 30dvh);
  }
}

.designer-open #designer-overlay {
  display: block;
}
`
}
