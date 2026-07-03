package web

import (
	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/web/components/appheader"
)

func Page() *Node {
	return Html(
		Attr("data-theme", "dark"),
		pageHead("Living Card", pageCSS()),
		Body(
			Class("min-h-screen bg-[var(--app-bg)] text-[var(--app-fg)]"),
			Div(
				Class("flex min-h-screen flex-col"),
				appheader.View(uiSecondaryButtonClass("sm")),
				Main(
					Class("flex-1 px-4 py-4 md:px-6"),
					cardWorkspaceView(),
				),
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
	return Section(
		Id("card-workspace"),
		Class("grid min-h-[calc(100vh-7.5rem)] gap-4 lg:grid-cols-[minmax(0,1.25fr)_24rem]"),
		Div(
			Class("flex min-h-[26rem] items-center justify-center rounded-lg border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm"),
			preview,
		),
		Aside(
			Class("rounded-lg border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm"),
			Div(
				Class("flex items-start justify-between gap-3"),
				Div(
					H2(Class("text-sm font-semibold uppercase text-[var(--app-fg-soft)]"), T("Designer")),
					P(Class("mt-1 text-sm text-[var(--app-fg-muted)]"), T("Generate a component fragment, review it, then apply it to the card.")),
				),
				Button(
					Id("designer-reset-btn"),
					Type("button"),
					Class(uiSecondaryButtonClass("xs")),
					T("Reset"),
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
		),
	)
}

func pageCSS() string {
	return `
#draft-card-preview {
  color: var(--app-fg);
}
`
}
