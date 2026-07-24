package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

type designerState struct {
	mu          sync.Mutex
	registry    *cardcomponent.Registry
	document    cardcomponent.Document
	gameState   GameState
	library     []cardcomponent.LibraryItem
	lastApplied *cardcomponent.LibraryItem
}

func newDesignerState(registry *cardcomponent.Registry) *designerState {
	return &designerState{registry: registry, document: cardcomponent.MustDefaultDocument(registry), gameState: initialGameState(), library: cloneLibrary(registry.Presets())}
}

func (s *designerState) snapshot() (cardcomponent.Document, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneValue(s.document), cloneLibrary(s.library)
}
func (s *designerState) interactiveSnapshot() (cardcomponent.Document, GameState, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gameState = syncGameStateWithDocument(s.registry, normalizeGameState(s.gameState), s.document)
	return cloneValue(s.document), cloneValue(s.gameState), cloneLibrary(s.library)
}
func (s *designerState) reset() (cardcomponent.Document, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.document = cardcomponent.MustDefaultDocument(s.registry)
	s.gameState = initialGameState()
	s.lastApplied = nil
	return cloneValue(s.document), cloneLibrary(s.library)
}

func (s *designerState) apply(raw json.RawMessage, componentID string) (cardcomponent.Document, json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	normalized, item, kind, err := applyGeneratedConfigToDocumentForComponent(s.registry, raw, &s.document, componentID)
	_ = kind
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	s.lastApplied = &item
	s.gameState = syncGameStateWithDocument(s.registry, s.gameState, s.document)
	return cloneValue(s.document), normalized, nil
}

func (s *designerState) tap(target, zone string, x, y float64) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	componentID, trait := canonicalTapComponent(s.document, target, zone)
	return s.interactLocked(componentID, trait, interactionShortTap, x, y)
}

func (s *designerState) interact(componentID, trait, interaction string, _, _ float64) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interactLocked(componentID, trait, interaction, 0, 0)
}

func (s *designerState) interactLocked(componentID, trait, interaction string, _, _ float64) (tapResult, error) {
	if strings.TrimSpace(componentID) == "" {
		componentID = cardcomponent.DefaultRootID
	}
	node := findNodeByIDPtr(&s.document.Root, componentID)
	if node == nil {
		return tapResult{}, fmt.Errorf("component %q is not available", componentID)
	}
	s.gameState = syncGameStateWithDocument(s.registry, s.gameState, s.document)
	s.gameState.SelectedComponentID = node.ID
	progress := s.gameState.ComponentProgress[node.ID]
	if interaction == interactionLongPress {
		progress.OverlayOpened = true
		progress.OverlayUnlocked = true
		s.gameState.ComponentProgress[node.ID] = progress
	}
	return s.result(node.ID, node.ComponentKind, ""), nil
}

func (s *designerState) randomizeComponent(componentID, trait, scope string) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := findNodeByIDPtr(&s.document.Root, strings.TrimSpace(componentID))
	if node == nil {
		return tapResult{}, fmt.Errorf("component %q is not available", componentID)
	}
	definition, _ := s.registry.Lookup(node.ComponentKind)
	generation, ok := definition.Generation()
	if !ok || !generation.SupportsRandom() {
		return tapResult{}, cardcomponent.NewUnsupportedOperationError(node.ComponentKind, "random generation")
	}
	raw, issues := generation.Random(time.Now().UnixNano(), 1)
	if len(issues) > 0 {
		return tapResult{}, fmt.Errorf("random generation at %s: %s", issues[0].Path, issues[0].Message)
	}
	normalized, item, _, err := applyGeneratedConfigToDocumentForComponent(s.registry, raw, &s.document, node.ID)
	if err != nil {
		return tapResult{}, err
	}
	s.lastApplied = &item
	result := s.result(node.ID, node.ComponentKind, "")
	result.appliedConfig = normalized
	result.events = []CardEvent{{Type: "configApplied", ComponentID: node.ID, ComponentKind: node.ComponentKind, Trait: trait}}
	_ = scope
	return result, nil
}

func (s *designerState) applyControlChange(componentID, trait, control string, value json.RawMessage) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := findNodeByIDPtr(&s.document.Root, strings.TrimSpace(componentID))
	if node == nil {
		return tapResult{}, fmt.Errorf("component %q is not available", componentID)
	}
	if err := s.applyNodeControl(node, control, value); err != nil {
		return tapResult{}, err
	}
	result := s.result(node.ID, node.ComponentKind, control)
	result.events = []CardEvent{{Type: "controlChanged", ComponentID: node.ID, ComponentKind: node.ComponentKind, Trait: trait, Control: control}}
	return result, nil
}

func (s *designerState) applyNodeControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	definition, ok := s.registry.Lookup(node.ComponentKind)
	if !ok {
		return fmt.Errorf("component kind %q is not registered", node.ComponentKind)
	}
	config, issues := definition.ApplyControl(node.Config, strings.TrimSpace(control), value)
	if len(issues) > 0 {
		return fmt.Errorf("control %q at %s: %s", control, issues[0].Path, issues[0].Message)
	}
	node.Config = config
	s.gameState = syncGameStateWithDocument(s.registry, s.gameState, s.document)
	return nil
}

func (s *designerState) addComponent(componentKind string, rawConfig json.RawMessage) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	definition, ok := s.registry.Lookup(strings.TrimSpace(componentKind))
	if !ok {
		return tapResult{}, fmt.Errorf("component kind %q is not registered", componentKind)
	}
	if definition.Structure() != cardcomponent.StructureLeaf {
		return tapResult{}, fmt.Errorf("component kind %q cannot be added as a leaf", componentKind)
	}
	config, issues := definition.CanonicalizeConfig(cardcomponent.RawConfig{Present: len(rawConfig) > 0, Value: rawConfig})
	if len(issues) > 0 {
		return tapResult{}, fmt.Errorf("invalid %s config at %s: %s", componentKind, issues[0].Path, issues[0].Message)
	}
	id := nextComponentID(s.document, componentKind)
	s.document.Root.Children = append(s.document.Root.Children, cardcomponent.Node{ID: id, ComponentKind: componentKind, Config: config})
	s.gameState = syncGameStateWithDocument(s.registry, s.gameState, s.document)
	s.gameState.SelectedComponentID = id
	result := s.result(id, componentKind, "")
	result.events = []CardEvent{{Type: "componentAdded", ComponentID: id, ComponentKind: componentKind, Message: definition.Label() + " added"}}
	return result, nil
}

func (s *designerState) result(componentID, kind, control string) tapResult {
	s.gameState = syncGameStateWithDocument(s.registry, normalizeGameState(s.gameState), s.document)
	return tapResult{document: cloneValue(s.document), gameState: cloneValue(s.gameState), library: cloneLibrary(s.library), overlay: buildOverlay(s.registry, s.document, s.gameState, componentID), events: []CardEvent{{Type: "configApplied", ComponentID: componentID, ComponentKind: kind, Control: control}}}
}

func (s *designerState) applyLibraryItem(id string) (cardcomponent.Document, json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, candidate := range s.library {
		if candidate.ID != id {
			continue
		}
		raw, err := generatedRawFromLibraryItem(candidate)
		if err != nil {
			return cardcomponent.Document{}, nil, err
		}
		normalized, item, _, err := applyGeneratedConfigToDocumentForComponent(s.registry, raw, &s.document, "")
		if err != nil {
			return cardcomponent.Document{}, nil, err
		}
		s.lastApplied = &item
		return cloneValue(s.document), normalized, nil
	}
	return cardcomponent.Document{}, nil, fmt.Errorf("library item %q was not found", id)
}
func (s *designerState) saveLastApplied() (cardcomponent.LibraryItem, []cardcomponent.LibraryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastApplied == nil {
		return cardcomponent.LibraryItem{}, nil, fmt.Errorf("no applied config is available to save")
	}
	item := cloneValue(*s.lastApplied)
	item.Saved = true
	if item.ID == "" || strings.HasPrefix(item.ID, "applied-") {
		item.ID = "saved-" + item.ComponentKind + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if item.Name == "" {
		item.Name = item.Description
	}
	for _, candidate := range s.library {
		if candidate.ComponentKind == item.ComponentKind && string(candidate.Config) == string(item.Config) {
			return cloneValue(candidate), cloneLibrary(s.library), nil
		}
	}
	s.library = append([]cardcomponent.LibraryItem{item}, s.library...)
	return cloneValue(item), cloneLibrary(s.library), nil
}
func (s *designerState) libraryForComponentKind(kind string) []cardcomponent.LibraryItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return cloneLibrary(s.library)
	}
	var out []cardcomponent.LibraryItem
	for _, item := range s.library {
		if item.ComponentKind == kind {
			out = append(out, cloneValue(item))
		}
	}
	return out
}
func (s *designerState) currentConfig(kind string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	node := findNodeByKind(s.document.Root, kind)
	if node == nil {
		return "", ""
	}
	raw, _ := json.MarshalIndent(schema.GeneratedConfigEnvelope{ComponentKind: kind, Description: "Current applied config", Config: node.Config}, "", "  ")
	return string(raw), node.ID
}

func generatedRawFromLibraryItem(item cardcomponent.LibraryItem) (json.RawMessage, error) {
	return json.Marshal(schema.GeneratedConfigEnvelope{ComponentKind: item.ComponentKind, Description: item.Description, Config: item.Config})
}
func nextComponentID(document cardcomponent.Document, kind string) string {
	base := document.CardID + "-" + strings.ReplaceAll(kind, "_", "-")
	if findNodeByID(document.Root, base) == nil {
		return base
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d", base, index)
		if findNodeByID(document.Root, candidate) == nil {
			return candidate
		}
	}
}
func cloneLibrary(items []cardcomponent.LibraryItem) []cardcomponent.LibraryItem {
	out := make([]cardcomponent.LibraryItem, len(items))
	for index, item := range items {
		out[index] = cloneValue(item)
	}
	return out
}
func cloneValue[T any](value T) T {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}
func findNodeByKind(node cardcomponent.Node, target string) *cardcomponent.Node {
	if node.ComponentKind == target {
		copy := cloneValue(node)
		return &copy
	}
	for _, child := range node.Children {
		if found := findNodeByKind(child, target); found != nil {
			return found
		}
	}
	return nil
}
func findNodeByKindPtr(node *cardcomponent.Node, target string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ComponentKind == target {
		return node
	}
	for index := range node.Children {
		if found := findNodeByKindPtr(&node.Children[index], target); found != nil {
			return found
		}
	}
	return nil
}
func findNodeByID(node cardcomponent.Node, id string) *cardcomponent.Node {
	if node.ID == id {
		copy := cloneValue(node)
		return &copy
	}
	for _, child := range node.Children {
		if found := findNodeByID(child, id); found != nil {
			return found
		}
	}
	return nil
}
func findNodeByIDPtr(node *cardcomponent.Node, id string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ID == id {
		return node
	}
	for index := range node.Children {
		if found := findNodeByIDPtr(&node.Children[index], id); found != nil {
			return found
		}
	}
	return nil
}
func componentExistsInDocument(document cardcomponent.Document, id string) bool {
	return findNodeByID(document.Root, strings.TrimSpace(id)) != nil
}
func syncGameStateWithDocument(registry *cardcomponent.Registry, state GameState, document cardcomponent.Document) GameState {
	state = normalizeGameState(state)
	var visit func(cardcomponent.Node)
	visit = func(node cardcomponent.Node) {
		if _, ok := registry.Lookup(node.ComponentKind); ok {
			progress := state.ComponentProgress[node.ID]
			progress.ComponentID = node.ID
			progress.ComponentKind = node.ComponentKind
			definition, _ := registry.Lookup(node.ComponentKind)
			progress.UnlockedControls = definition.ControlIDs()
			controls, _ := definition.Controls(node.Config)
			progress.UnlockedTraits = nil
			seen := map[string]bool{}
			for _, control := range controls {
				if control.Trait != "" && !seen[control.Trait] {
					progress.UnlockedTraits = append(progress.UnlockedTraits, control.Trait)
					seen[control.Trait] = true
				}
			}
			progress.OverlayUnlocked = true
			state.ComponentProgress[node.ID] = progress
			state.UnlockedComponentKinds = appendStringOnce(state.UnlockedComponentKinds, node.ComponentKind)
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document.Root)
	return normalizeGameState(state)
}
func documentSignature(document cardcomponent.Document) string {
	raw, _ := json.Marshal(document)
	return string(raw)
}
