package component

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/chat"
	"github.com/n0remac/Living-Card/internal/memory"
	"github.com/n0remac/Living-Card/internal/ollama"
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

type PatchClient interface {
	Chat(ctx context.Context, model string, messages []ollama.ChatMessage) (string, error)
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
	Patch   PatchClient

	PatchModel  string
	ProjectRoot string
}

type Component interface {
	MountID() string
	Render() *Node
}

type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux, deps Dependencies)
}

type ActionHandler func(w http.ResponseWriter, r *http.Request, deps Dependencies, card cards.Card, instance cards.ComponentInstance, action string) bool

type Definition struct {
	Type              string
	Render            func(cards.ComponentInstance) *Node
	HandleAction      ActionHandler
	ContextFiles      []string
	ClientInitializer string
}

type Registry struct {
	definitions map[string]Definition
}

func NewRegistry(definitions ...Definition) (*Registry, error) {
	registry := &Registry{
		definitions: make(map[string]Definition, len(definitions)),
	}
	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func MustNewRegistry(definitions ...Definition) *Registry {
	registry, err := NewRegistry(definitions...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *Registry) Register(definition Definition) error {
	if r == nil {
		return fmt.Errorf("component registry is not initialized")
	}
	definition.Type = strings.TrimSpace(definition.Type)
	definition.ClientInitializer = strings.TrimSpace(definition.ClientInitializer)
	if definition.Type == "" {
		return fmt.Errorf("component type is required")
	}
	if _, exists := r.definitions[definition.Type]; exists {
		return fmt.Errorf("duplicate component type %q", definition.Type)
	}
	definition.ContextFiles = sanitizeContextFiles(definition.ContextFiles)
	r.definitions[definition.Type] = definition
	return nil
}

func (r *Registry) Definition(componentType string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	definition, ok := r.definitions[strings.TrimSpace(componentType)]
	return definition, ok
}

func (r *Registry) ContextFiles(componentType string) ([]string, bool) {
	definition, ok := r.Definition(componentType)
	if !ok {
		return nil, false
	}
	return append([]string(nil), definition.ContextFiles...), true
}

func (r *Registry) RenderCard(card cards.Card) (*Node, error) {
	nodes := make([]*Node, 0, len(card.Components))
	for _, instance := range card.Components {
		definition, ok := r.Definition(instance.Type)
		if !ok {
			return nil, fmt.Errorf("component type %q is not registered", instance.Type)
		}
		if definition.Render == nil {
			return nil, fmt.Errorf("component type %q is not renderable", instance.Type)
		}
		nodes = append(nodes, Div(
			Attr("data-component-id", instance.ID),
			Attr("data-component-type", instance.Type),
			Attr("data-client-initializer", definition.ClientInitializer),
			definition.Render(instance),
		))
	}
	return Div(nodes...), nil
}

func (r *Registry) HandleCardResource(w http.ResponseWriter, req *http.Request, deps Dependencies, card cards.Card, parts []string) bool {
	if len(parts) != 5 || parts[1] != "components" || parts[3] != "actions" {
		return false
	}
	instance, ok := findComponent(card.Components, parts[2])
	if !ok {
		http.Error(w, "component not found", http.StatusNotFound)
		return true
	}
	definition, ok := r.Definition(instance.Type)
	if !ok {
		http.Error(w, "component type is not registered", http.StatusInternalServerError)
		return true
	}
	if definition.HandleAction == nil || !definition.HandleAction(w, req, deps, card, instance, parts[4]) {
		http.NotFound(w, req)
		return true
	}
	return true
}

func findComponent(components []cards.ComponentInstance, componentID string) (cards.ComponentInstance, bool) {
	componentID = strings.TrimSpace(componentID)
	for _, component := range components {
		if component.ID == componentID {
			return component, true
		}
	}
	return cards.ComponentInstance{}, false
}

func sanitizeContextFiles(input []string) []string {
	out := make([]string, 0, len(input))
	for _, item := range input {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
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
