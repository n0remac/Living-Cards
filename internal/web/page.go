package web

import (
	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/web/components/appheader"
	"github.com/n0remac/Living-Card/internal/web/components/chatform"
)

func Page() *Node {
	return Html(
		Attr("data-theme", "dark"),
		pageHead("Living Card", pageCSS()),
		Body(
			Class("min-h-screen bg-[var(--app-bg)] text-[var(--app-fg)]"),
			Div(
				Class("flex min-h-screen flex-col"),
				appheader.View(),
				Main(
					Class("flex-1 px-4 pb-4 pt-3 md:px-6"),
					Div(
						Class("grid gap-4 xl:grid-cols-[18rem_minmax(0,1.2fr)_minmax(0,0.9fr)]"),
						Aside(
							Class("rounded-3xl border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm backdrop-blur"),
							H2(Class("text-sm font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Cards")),
							Div(Id("card-list"), Class("mt-4 flex max-h-[calc(100vh-12rem)] flex-col gap-2 overflow-y-auto pr-1")),
						),
						chatform.View(),
						Section(
							Class("rounded-3xl border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm backdrop-blur"),
							H2(Class("text-sm font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Debug")),
							Div(
								Class("mt-4 space-y-4"),
								Div(
									Class("rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"),
									H3(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Retrieved Memories")),
									Div(Id("retrieved-memories"), Class("mt-3 space-y-3 text-sm text-[var(--app-fg-muted)]"), T("No retrieval yet.")),
								),
								Div(
									Class("rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"),
									H3(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Stored Summary")),
									Div(Id("stored-summary"), Class("mt-3 text-sm text-[var(--app-fg-muted)]"), T("No interaction stored yet.")),
								),
								Div(
									Class("rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4"),
									H3(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Recent Memories")),
									Div(Id("recent-memories"), Class("mt-3 space-y-3 text-sm text-[var(--app-fg-muted)]"), T("No card selected.")),
								),
							),
						),
					),
				),
			),
			Script(Type("module"), Src("/assets/app.js")),
		),
	)
}

func pageCSS() string {
	return `
.card-item.is-active {
  border-color: rgba(34, 211, 238, 0.68);
  background: rgba(8, 145, 178, 0.14);
}
`
}
