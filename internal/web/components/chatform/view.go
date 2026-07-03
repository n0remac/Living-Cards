package chatform

import (
	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/web/component"
)

const MountID = "chat-form-component"

type Component struct{}

func (Component) MountID() string {
	return MountID
}

func (Component) Render() *Node {
	return View()
}

func View() *Node {
	return Section(
		Id(MountID),
		Class("rounded-3xl border border-[var(--app-border-strong)] bg-[var(--app-surface)] p-4 shadow-sm backdrop-blur"),
		Div(
			Class("flex items-center justify-between gap-3"),
			H2(Class("text-sm font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Conversation")),
			Div(Id("conversation-status"), Class("text-sm text-[var(--app-fg-soft)]"), T("Select a card to begin.")),
		),
		Div(Id("card-meta"), Class("mt-4 rounded-2xl border border-[var(--app-border)] bg-[var(--app-surface-muted)] p-4 text-sm text-[var(--app-fg-muted)]"), T("No card selected.")),
		Div(Id("transcript"), Class("mt-4 space-y-3")),
		Form(
			Id("chat-form"),
			Class("mt-4 space-y-3"),
			Div(
				Class("space-y-2"),
				Label(Class("block text-[0.72rem] font-semibold uppercase tracking-[0.18em] text-[var(--app-fg-soft)]"), T("Message")),
				TextArea(
					Id("chat-input"),
					Name("message"),
					Class(component.InputClass()+" min-h-28 resize-y"),
					Placeholder("Ask the card a question..."),
				),
			),
			Button(
				Id("send-btn"),
				Type("submit"),
				Class(component.PrimaryButtonClass("sm")),
				T("Send"),
			),
		),
	)
}
