package component

import (
	"context"
	"net/http"

	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/chat"
	"github.com/n0remac/Living-Card/internal/memory"
	"github.com/n0remac/Living-Card/internal/profile"
)

type CardStore interface {
	List() []cards.Card
	Get(cardID string) (cards.Card, bool)
}

type MemoryStore interface {
	ListByCard(ctx context.Context, userID, cardID string, limit int) ([]memory.Memory, error)
}

type ChatService interface {
	Chat(ctx context.Context, request chat.Request) (chat.Result, error)
}

type ProfileStore interface {
	Get(ctx context.Context, userID string) (profile.Profile, error)
	Reset(ctx context.Context, userID string) error
}

type Dependencies struct {
	Cards   CardStore
	Memory  MemoryStore
	Chat    ChatService
	Profile ProfileStore
}

type Component interface {
	MountID() string
	Render() *Node
}

type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux, deps Dependencies)
}

func PrimaryButtonClass(size string) string {
	return "inline-flex items-center justify-center rounded-xl border border-cyan-300/30 bg-cyan-400 px-4 font-semibold text-slate-950 shadow-sm transition duration-150 hover:-translate-y-0.5 hover:bg-cyan-300 focus:outline-none focus:ring-4 focus:ring-cyan-400/25 disabled:cursor-not-allowed disabled:opacity-60 " + buttonSizeClass(size)
}

func SecondaryButtonClass(size string) string {
	return "inline-flex items-center justify-center rounded-xl border border-[var(--app-border-strong)] bg-[var(--app-panel)] px-4 font-medium text-[var(--app-fg)] shadow-sm transition duration-150 hover:-translate-y-0.5 hover:border-cyan-400/50 hover:bg-cyan-500/10 focus:outline-none focus:ring-4 focus:ring-cyan-400/15 disabled:cursor-not-allowed disabled:opacity-60 " + buttonSizeClass(size)
}

func InputClass() string {
	return "w-full rounded-2xl border border-[var(--app-border-strong)] bg-[var(--app-panel)] px-3 py-2.5 text-sm text-[var(--app-fg)] shadow-sm outline-none transition placeholder:text-[var(--app-fg-soft)] focus:border-cyan-400/70 focus:ring-4 focus:ring-cyan-400/15"
}

func buttonSizeClass(size string) string {
	switch size {
	case "xs":
		return "h-8 px-3 text-[0.68rem] uppercase tracking-[0.18em]"
	case "sm":
		return "h-9 px-3.5 text-sm"
	default:
		return "h-10 px-4 text-sm"
	}
}
