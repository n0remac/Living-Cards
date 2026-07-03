package appheader

import (
	. "github.com/n0remac/GoDom/html"
)

const MountID = "app-header"

func View(resetButtonClass ...string) *Node {
	buttonClass := "inline-flex h-9 items-center justify-center rounded-xl border border-[var(--app-border-strong)] bg-[var(--app-panel)] px-3.5 text-sm font-medium text-[var(--app-fg)] shadow-sm transition duration-150 hover:-translate-y-0.5 hover:border-cyan-400/50 hover:bg-cyan-500/10 focus:outline-none focus:ring-4 focus:ring-cyan-400/15 disabled:cursor-not-allowed disabled:opacity-60"
	if len(resetButtonClass) > 0 && resetButtonClass[0] != "" {
		buttonClass = resetButtonClass[0]
	}
	return Header(
		Id(MountID),
		Class("border-b border-[var(--app-border-strong)] bg-[var(--app-surface)]/95 backdrop-blur"),
		Div(
			Class("mx-auto flex w-full max-w-none flex-wrap items-center gap-3 px-4 py-4 md:px-6"),
			Div(
				Class("mr-auto flex min-w-[14rem] flex-col"),
				H1(Class("text-2xl font-semibold tracking-tight text-[var(--app-fg)]"), T("Living Card")),
				P(Class("text-sm text-[var(--app-fg-muted)]"), T("Design an editable AI-powered card from structured style fragments.")),
			),
			Button(
				Id("reset-draft-btn"),
				Type("button"),
				Class(buttonClass),
				T("Reset Draft"),
			),
		),
	)
}
