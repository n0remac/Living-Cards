package appheader

import (
	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/web/component"
)

const MountID = "app-header"

type Component struct{}

func (Component) MountID() string {
	return MountID
}

func (Component) Render() *Node {
	return View()
}

func View() *Node {
	return Header(
		Id(MountID),
		Class("border-b border-[var(--app-border-strong)] bg-[var(--app-surface)]/95 backdrop-blur"),
		Div(
			Class("mx-auto flex w-full max-w-none flex-wrap items-center gap-3 px-4 py-4 md:px-6"),
			Div(
				Class("mr-auto flex min-w-[14rem] flex-col"),
				H1(Class("text-2xl font-semibold tracking-tight text-[var(--app-fg)]"), T("Living Card")),
				P(Class("text-sm text-[var(--app-fg-muted)]"), T("A persistent AI entity with memory, retrieval, and local inference.")),
			),
			Button(
				Id("reload-cards-btn"),
				Type("button"),
				Class(component.SecondaryButtonClass("sm")),
				T("Reload Cards"),
			),
		),
	)
}
