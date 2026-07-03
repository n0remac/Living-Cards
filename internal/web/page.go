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
				stageHUDView(),
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
		preview,
	)
}

func stageHUDView() *Node {
	return Div(
		Class("pointer-events-none fixed inset-x-0 top-0 z-20 flex items-start justify-between gap-3 p-3 sm:p-5"),
		Div(
			Class("pointer-events-auto flex min-w-0 flex-col gap-2 rounded-md border border-[var(--app-border)] bg-[var(--app-surface)]/90 px-3 py-2 shadow-lg backdrop-blur"),
			Div(
				Class("flex items-center gap-3"),
				H1(Class("text-sm font-semibold text-[var(--app-fg)]"), T("Living Card")),
				Span(Id("card-level"), Class("rounded-full bg-emerald-400/15 px-2 py-0.5 text-xs font-semibold text-emerald-100"), T("Lv 1")),
			),
			Div(
				Class("h-1.5 w-44 max-w-[42vw] overflow-hidden rounded-full bg-black/30"),
				Div(Id("card-xp-bar"), Class("h-full w-0 rounded-full bg-emerald-300 transition-[width] duration-300")),
			),
			Div(
				Class("flex gap-3 text-[0.7rem] font-medium uppercase text-[var(--app-fg-soft)]"),
				Span(Id("card-xp"), T("0 XP")),
				Span(Id("card-taps"), T("0 taps")),
			),
		),
		Div(
			Class("pointer-events-auto flex gap-2"),
			Button(
				Id("designer-toggle-btn"),
				Type("button"),
				Class(uiSecondaryButtonClass("sm")),
				T("Designer"),
			),
			Button(
				Id("reset-draft-btn"),
				Type("button"),
				Class(uiSecondaryButtonClass("sm")),
				T("Reset"),
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
  padding: 5.5rem 1rem 4rem;
  perspective: 1000px;
  touch-action: manipulation;
}

.stage-card-wrap #draft-card-preview {
  width: min(76vw, 28rem, calc((100dvh - 10rem) * 5 / 7));
  max-width: min(76vw, 28rem, calc((100dvh - 10rem) * 5 / 7));
  cursor: pointer;
  transform-origin: center;
  will-change: transform;
  color: var(--app-fg);
}

.stage-toast {
  position: absolute;
  left: 50%;
  top: 18%;
  transform: translateX(-50%);
  max-width: min(28rem, calc(100vw - 2rem));
}

.designer-open #designer-overlay {
  display: block;
}
`
}
