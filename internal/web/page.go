package web

import (
	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
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
				stageResetView(),
				cardWorkspaceView(),
				Div(
					Id("stage-overlay-root"),
					Class("pointer-events-none fixed inset-0 z-30"),
					Attr("aria-live", "polite"),
				),
				designerOverlayView(),
			),
			Script(Type("module"), Src("/assets/app.js")),
		),
	)
}

func cardWorkspaceView() *Node {
	preview, err := cardcomponent.RenderDocument(cardcomponent.DefaultDocument(), cardComponentRegistry())
	if err != nil {
		preview = Div(
			Id("draft-card-preview"),
			Class("relative aspect-[5/7] w-full max-w-md overflow-hidden rounded-3xl border border-red-400/40 p-6 text-red-100"),
			T(err.Error()),
		)
	}
	return Div(
		Id("card-workspace"),
		Attr("data-card-preview-root", ""),
		Class("stage-card-wrap"),
		Div(
			Class("stage-card-stack"),
			stageHUDView(),
			preview,
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
		Class("pointer-events-none fixed right-3 top-3 z-20 sm:right-5 sm:top-5"),
		Button(
			Id("reset-draft-btn"),
			Type("button"),
			Class(uiSecondaryButtonClass("sm")+" pointer-events-auto"),
			T("Reset"),
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
  overflow: hidden;
  display: grid;
  place-items: center;
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

.stage-card-wrap {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  padding: 1rem;
  perspective: 1000px;
  touch-action: manipulation;
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

.designer-open #designer-overlay {
  display: block;
}
`
}
