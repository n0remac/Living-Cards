package web

import (
	"encoding/json"
	"strings"
	"time"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

const (
	componentKindCard      = cardcomponent.Kind
	componentCardRoot      = cardcomponent.DefaultRootID
	traitShadow            = "shadow"
	traitPadding           = "padding"
	traitTypography        = "typography"
	traitFill              = "fill"
	traitPosition          = "position"
	traitSize              = "size"
	interactionShortTap    = "shortTap"
	interactionLongPress   = "longPress"
	editModeRandom         = "random"
	editModeSimpleControls = "simpleControls"
	xpPerInteraction       = 1
	xpPerLevel             = 5
	componentXPPerLevel    = 3
)

type GameState struct {
	TotalXP                int                              `json:"totalXp"`
	GlobalLevel            int                              `json:"globalLevel"`
	TotalInteractions      int                              `json:"totalInteractions"`
	UnlockedComponentKinds []string                         `json:"unlockedComponentKinds"`
	SelectedComponentID    string                           `json:"selectedComponentId,omitempty"`
	ComponentProgress      map[string]ComponentProgress     `json:"componentProgress"`
	TapCount               int                              `json:"tapCount"`
	Level                  int                              `json:"level"`
	XP                     int                              `json:"xp"`
	UnlockedConfigKinds    []string                         `json:"unlockedConfigKinds"`
	UnlockedModes          []string                         `json:"unlockedModes"`
	ComponentKindProgress  map[string]ComponentKindProgress `json:"componentKindProgress"`
}
type ComponentProgress struct {
	ComponentID      string   `json:"componentId"`
	ComponentKind    string   `json:"componentKind"`
	XP               int      `json:"xp"`
	Level            int      `json:"level"`
	Interactions     int      `json:"interactions"`
	RandomTapEnabled bool     `json:"randomTapEnabled"`
	OverlayUnlocked  bool     `json:"overlayUnlocked"`
	OverlayOpened    bool     `json:"overlayOpened"`
	UnlockedTraits   []string `json:"unlockedTraits"`
	UnlockedControls []string `json:"unlockedControls"`
}
type ComponentKindProgress struct {
	Taps          int      `json:"taps"`
	Level         int      `json:"level"`
	UnlockedModes []string `json:"unlockedModes"`
}
type ComponentDescriptor struct {
	ComponentID   string   `json:"componentId"`
	ComponentKind string   `json:"componentKind"`
	Label         string   `json:"label"`
	Traits        []string `json:"traits"`
}
type ComponentOverlay struct {
	ComponentID      string              `json:"componentId"`
	ComponentKind    string              `json:"componentKind"`
	Title            string              `json:"title"`
	RandomizeEnabled bool                `json:"randomizeEnabled"`
	Controls         []ControlDescriptor `json:"controls"`
}
type ControlDescriptor = cardcomponent.ControlDescriptor
type ControlOption = cardcomponent.ControlOption
type CardEvent struct {
	Type          string `json:"type"`
	ComponentKind string `json:"componentKind,omitempty"`
	ComponentID   string `json:"componentId,omitempty"`
	Trait         string `json:"trait,omitempty"`
	Control       string `json:"control,omitempty"`
	Amount        int    `json:"amount,omitempty"`
	Level         int    `json:"level,omitempty"`
	Mode          string `json:"mode,omitempty"`
	Message       string `json:"message,omitempty"`
}
type tapResult struct {
	document      cardcomponent.Document
	gameState     GameState
	appliedConfig json.RawMessage
	library       []cardcomponent.LibraryItem
	events        []CardEvent
	overlay       *ComponentOverlay
}

func initialGameState() GameState {
	return normalizeGameState(GameState{GlobalLevel: 1, UnlockedComponentKinds: []string{componentKindCard}, SelectedComponentID: componentCardRoot, ComponentProgress: map[string]ComponentProgress{componentCardRoot: {ComponentID: componentCardRoot, ComponentKind: componentKindCard, Level: 1, OverlayUnlocked: true}}})
}
func normalizeGameState(state GameState) GameState {
	if state.TotalXP < 0 {
		state.TotalXP = 0
	}
	state.GlobalLevel = state.TotalXP/xpPerLevel + 1
	if state.GlobalLevel < 1 {
		state.GlobalLevel = 1
	}
	if state.ComponentProgress == nil {
		state.ComponentProgress = map[string]ComponentProgress{}
	}
	for id, progress := range state.ComponentProgress {
		progress.ComponentID = id
		if progress.Level < 1 {
			progress.Level = progress.XP/componentXPPerLevel + 1
		}
		state.ComponentProgress[id] = progress
	}
	if state.SelectedComponentID == "" {
		state.SelectedComponentID = componentCardRoot
	}
	state.XP = state.TotalXP
	state.Level = state.GlobalLevel
	state.TapCount = state.TotalInteractions
	state.UnlockedConfigKinds = append([]string(nil), state.UnlockedComponentKinds...)
	state.UnlockedModes = []string{editModeRandom, editModeSimpleControls}
	state.ComponentKindProgress = map[string]ComponentKindProgress{}
	for _, progress := range state.ComponentProgress {
		current := state.ComponentKindProgress[progress.ComponentKind]
		current.Taps += progress.Interactions
		if progress.Level > current.Level {
			current.Level = progress.Level
		}
		current.UnlockedModes = append([]string(nil), state.UnlockedModes...)
		state.ComponentKindProgress[progress.ComponentKind] = current
	}
	return state
}
func ensureComponentProgress(state GameState, id, kind string) GameState {
	state = normalizeGameState(state)
	progress := state.ComponentProgress[id]
	progress.ComponentID = id
	progress.ComponentKind = kind
	if progress.Level < 1 {
		progress.Level = 1
	}
	state.ComponentProgress[id] = progress
	state.UnlockedComponentKinds = appendStringOnce(state.UnlockedComponentKinds, kind)
	return state
}
func canonicalTapComponent(document cardcomponent.Document, target, zone string) (string, string) {
	target = strings.TrimSpace(target)
	zone = strings.TrimSpace(zone)
	if node := findNodeByID(document.Root, target); node != nil {
		return node.ID, zone
	}
	if node := findNodeByKind(document.Root, target); node != nil {
		return node.ID, zone
	}
	return target, zone
}
func canonicalTapTarget(target, zone string) string {
	target = strings.TrimSpace(target)
	if target != "" {
		return target
	}
	return strings.TrimSpace(zone)
}
func isKnownTapTarget(target string) bool { return strings.TrimSpace(target) != "" }
func isKnownComponentID(id string) bool   { return strings.TrimSpace(id) != "" }

func advanceInteraction(state GameState, componentID string, amount int) (GameState, []CardEvent) {
	state = normalizeGameState(state)
	if amount < 1 {
		amount = 1
	}
	progress := state.ComponentProgress[componentID]
	before := progress.Level
	progress.XP += amount
	progress.Interactions += amount
	progress.Level = progress.XP/componentXPPerLevel + 1
	state.ComponentProgress[componentID] = progress
	state.TotalXP += amount
	state.TotalInteractions += amount
	events := []CardEvent{{Type: "xpGained", ComponentID: componentID, ComponentKind: progress.ComponentKind, Amount: amount}}
	if progress.Level > before {
		events = append(events, CardEvent{Type: "componentLevelUp", ComponentID: componentID, ComponentKind: progress.ComponentKind, Level: progress.Level})
	}
	return normalizeGameState(state), events
}
func tapSeed(state GameState, componentID string) int64 {
	return time.Now().UnixNano() + int64(state.TotalInteractions) + int64(len(componentID))
}

func availableComponents(registry *cardcomponent.Registry, state GameState, document cardcomponent.Document) []ComponentDescriptor {
	state = syncGameStateWithDocument(registry, state, document)
	var out []ComponentDescriptor
	var visit func(cardcomponent.Node)
	visit = func(node cardcomponent.Node) {
		definition, ok := registry.Lookup(node.ComponentKind)
		if ok {
			progress := state.ComponentProgress[node.ID]
			out = append(out, ComponentDescriptor{ComponentID: node.ID, ComponentKind: node.ComponentKind, Label: definition.Label(), Traits: append([]string(nil), progress.UnlockedTraits...)})
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document.Root)
	return out
}
func buildOverlay(registry *cardcomponent.Registry, document cardcomponent.Document, state GameState, componentID string) *ComponentOverlay {
	node := findNodeByID(document.Root, strings.TrimSpace(componentID))
	if node == nil {
		return nil
	}
	definition, ok := registry.Lookup(node.ComponentKind)
	if !ok {
		return nil
	}
	controls, issues := definition.Controls(node.Config)
	if len(issues) > 0 {
		return nil
	}
	generation, hasGeneration := definition.Generation()
	return &ComponentOverlay{ComponentID: node.ID, ComponentKind: node.ComponentKind, Title: definition.Label(), RandomizeEnabled: hasGeneration && generation.SupportsRandom(), Controls: controls}
}

func appendStringOnce(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || stringInSlice(values, value) {
		return values
	}
	return append(values, value)
}
func stringInSlice(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func componentTitle(kind string) string {
	words := strings.ReplaceAll(kind, "_", " ")
	if words == "" {
		return "Component"
	}
	return strings.ToUpper(words[:1]) + words[1:]
}
func componentLabel(kind string) string { return componentTitle(kind) }
