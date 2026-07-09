package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/design"
)

const (
	KindWorld = "world"
	KindItem  = "item"
	KindClue  = "clue"

	DoorUnlockedFlag = "doorUnlocked"

	BlankControllerCardID     = "blank-controller"
	SliderComponentCardID     = "slider-component"
	RegulatorControllerCardID = "generator-regulator-controller"
	GeneratorPoweredFlag      = "generatorPowered"
)

type Card struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Tags        []string               `json:"tags,omitempty"`
	Collectible bool                   `json:"collectible"`
	Collected   bool                   `json:"collected,omitempty"`
	State       map[string]any         `json:"state,omitempty"`
	Document    cardcomponent.Document `json:"document"`
}

type Snapshot struct {
	WorldDeck                []Card          `json:"worldDeck"`
	ActiveWorldCard          Card            `json:"activeWorldCard"`
	ActiveWorldCardID        string          `json:"activeWorldCardId"`
	ActiveIndex              int             `json:"activeIndex"`
	ActiveEditingComponentID string          `json:"activeEditingComponentId,omitempty"`
	Library                  []Card          `json:"library"`
	EditSession              *EditSession    `json:"editSession,omitempty"`
	SolvedFlags              map[string]bool `json:"solvedFlags"`
	Message                  string          `json:"message,omitempty"`
}

type EditSession struct {
	TargetCardID                string   `json:"targetCardId"`
	DraftCard                   Card     `json:"draftCard"`
	PendingConsumedComponentIDs []string `json:"pendingConsumedComponentIds,omitempty"`
	SelectedComponentID         string   `json:"selectedComponentId,omitempty"`
}

type Session struct {
	mu                       sync.Mutex
	deckDefinition           DeckDefinition
	cardDefinitions          map[string]CardDefinition
	documentVariants         map[string]map[string]cardcomponent.Document
	loadedDecks              map[string]bool
	useRules                 []UseRuleDefinition
	worldDeck                []Card
	activeIndex              int
	activeEditingComponentID string
	library                  []Card
	editSession              *EditSession
	solvedFlags              map[string]bool
	lastMessage              string
}

func NewSession() *Session {
	session, err := NewSessionFromEmbeddedDeck()
	if err != nil {
		panic(err)
	}
	return session
}

func NewSessionFromEmbeddedDeck() (*Session, error) {
	definition, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		return nil, err
	}
	return NewSessionFromDeck(definition)
}

func NewSessionFromDeck(definition DeckDefinition) (*Session, error) {
	if err := ValidateDeckDefinition(definition); err != nil {
		return nil, err
	}
	definition = cloneValue(definition)
	worldDeck, documentVariants, cardDefinitions, activeIndex, err := materializeDeck(definition)
	if err != nil {
		return nil, err
	}
	return &Session{
		deckDefinition:   definition,
		cardDefinitions:  cardDefinitions,
		documentVariants: documentVariants,
		loadedDecks:      map[string]bool{definition.ID: true},
		useRules:         cloneValue(definition.UseRules),
		worldDeck:        worldDeck,
		activeIndex:      activeIndex,
		library:          nil,
		solvedFlags:      cloneValue(definition.InitialSolvedFlags),
		lastMessage:      definition.InitialMessage,
	}, nil
}

func (s *Session) Snapshot() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.snapshotLocked()
}

func (s *Session) Reset() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next, err := NewSessionFromDeck(s.deckDefinition)
	if err != nil {
		return Snapshot{}, err
	}
	s.cardDefinitions = next.cardDefinitions
	s.documentVariants = next.documentVariants
	s.loadedDecks = next.loadedDecks
	s.useRules = next.useRules
	s.worldDeck = next.worldDeck
	s.activeIndex = next.activeIndex
	s.activeEditingComponentID = next.activeEditingComponentID
	s.library = next.library
	s.editSession = next.editSession
	s.solvedFlags = next.solvedFlags
	s.lastMessage = next.lastMessage
	return s.snapshotLocked()
}

func (s *Session) Cycle(direction string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.worldDeck) == 0 {
		return Snapshot{}, fmt.Errorf("world deck is empty")
	}
	switch strings.TrimSpace(direction) {
	case "previous", "prev", "back":
		s.activeIndex--
	case "", "next":
		s.activeIndex++
	default:
		return Snapshot{}, fmt.Errorf("direction must be next or previous")
	}
	if s.activeIndex < 0 {
		s.activeIndex = len(s.worldDeck) - 1
	}
	if s.activeIndex >= len(s.worldDeck) {
		s.activeIndex = 0
	}
	s.activeEditingComponentID = ""
	s.lastMessage = "The next card slides into view."
	return s.snapshotLocked()
}

func (s *Session) Collect(cardID string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cardID = strings.TrimSpace(cardID)
	if cardID == "" && len(s.worldDeck) > 0 {
		cardID = s.worldDeck[s.activeIndex].ID
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	card := s.worldDeck[index]
	if !card.Collectible {
		return Snapshot{}, fmt.Errorf("%s cannot be collected", card.Name)
	}
	if card.Collected {
		s.lastMessage = card.Name + " is already in your library."
		return s.snapshotLocked()
	}
	card.Collected = true
	card.Collectible = false
	s.worldDeck[index] = card
	libraryCard := cloneValue(card)
	libraryCard.Collectible = false
	libraryCard.Collected = true
	s.library = append(s.library, libraryCard)
	s.activeEditingComponentID = ""
	s.lastMessage = card.Name + " moved into your library."
	return s.snapshotLocked()
}

func (s *Session) UseCard(sourceCardID, targetCardID string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceCardID = strings.TrimSpace(sourceCardID)
	targetCardID = strings.TrimSpace(targetCardID)
	source := s.libraryCard(sourceCardID)
	if source == nil {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", sourceCardID)
	}
	if targetCardID == "" && len(s.worldDeck) > 0 {
		targetCardID = s.worldDeck[s.activeIndex].ID
	}
	targetIndex := s.worldCardIndex(targetCardID)
	if targetIndex < 0 {
		return Snapshot{}, fmt.Errorf("target card %q is not in the world deck", targetCardID)
	}
	target := s.worldDeck[targetIndex]
	for _, rule := range s.useRules {
		if !s.ruleBaseMatches(rule, *source, target) {
			continue
		}
		if !sourceComponentConditionsMatch(rule.SourceComponentConditions, source.Document) {
			if err := s.applyRuleFailureEffects(rule, *source, target); err != nil {
				return Snapshot{}, err
			}
			if strings.TrimSpace(rule.FailureMessage) != "" {
				s.lastMessage = rule.FailureMessage
				return s.snapshotLocked()
			}
			continue
		}
		if err := s.applyRuleEffects(rule, *source, target); err != nil {
			return Snapshot{}, err
		}
		s.activeEditingComponentID = ""
		return s.snapshotLocked()
	}
	s.lastMessage = "Nothing on this card responds to " + source.Name + "."
	return s.snapshotLocked()
}

func (s *Session) SelectWorldComponent(cardID, componentID, componentKind string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, node, err := s.worldComponentNode(cardID, componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.requireWorldComponentEditable(node.ComponentKind); err != nil {
		return Snapshot{}, err
	}
	s.activeIndex = index
	s.activeEditingComponentID = node.ID
	s.lastMessage = componentEditLabel(node.ComponentKind) + " edit controls opened."
	return s.snapshotLocked()
}

func (s *Session) ApplyWorldComponentControl(cardID, componentID, componentKind, control string, value json.RawMessage) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, node, err := s.worldComponentNode(cardID, componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.requireWorldComponentEditable(node.ComponentKind); err != nil {
		return Snapshot{}, err
	}
	if err := applyGameComponentControl(node, strings.TrimSpace(control), value); err != nil {
		return Snapshot{}, err
	}
	s.activeIndex = index
	s.activeEditingComponentID = node.ID
	powered, err := s.powerGeneratorIfTuned(index, node.ID)
	if err != nil {
		return Snapshot{}, err
	}
	if !powered {
		s.lastMessage = componentEditLabel(node.ComponentKind) + " updated."
	}
	return s.snapshotLocked()
}

func (s *Session) StartEdit(cardID string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cardID = strings.TrimSpace(cardID)
	index := s.libraryCardIndex(cardID)
	if index < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", cardID)
	}
	card := s.library[index]
	if !stateBool(card.State, "editable") {
		return Snapshot{}, fmt.Errorf("%s cannot be edited", card.Name)
	}
	s.editSession = &EditSession{
		TargetCardID: card.ID,
		DraftCard:    cloneValue(card),
	}
	s.activeEditingComponentID = ""
	s.lastMessage = "Editing " + card.Name + "."
	return s.snapshotLocked()
}

func (s *Session) InstallEditComponent(componentCardID string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	componentCardID = strings.TrimSpace(componentCardID)
	componentIndex := s.libraryCardIndex(componentCardID)
	if componentIndex < 0 {
		return Snapshot{}, fmt.Errorf("component card %q is not in your library", componentCardID)
	}
	if componentCardID == s.editSession.TargetCardID {
		return Snapshot{}, fmt.Errorf("a card cannot install itself")
	}
	if stringInSlice(s.editSession.PendingConsumedComponentIDs, componentCardID) {
		return Snapshot{}, fmt.Errorf("%s is already pending for this edit", s.library[componentIndex].Name)
	}

	component := s.library[componentIndex]
	componentKind := stateString(component.State, "componentKind")
	switch componentKind {
	case slider.Kind:
		part, err := sliderConfigFromComponentCard(component)
		if err != nil {
			return Snapshot{}, err
		}
		nodeID := nextComponentNodeID(s.editSession.DraftCard.Document, preferredSliderNodeID(s.editSession.TargetCardID))
		s.editSession.DraftCard.Document.Root.Children = append(s.editSession.DraftCard.Document.Root.Children, cardcomponent.Node{
			ID:            nodeID,
			ComponentKind: slider.Kind,
			Config:        mustRaw(part),
		})
		s.editSession.SelectedComponentID = nodeID
	case border.Kind:
		part, err := borderConfigFromComponentCard(component)
		if err != nil {
			return Snapshot{}, err
		}
		node := findNodeByKindPtr(&s.editSession.DraftCard.Document.Root, border.Kind)
		if node == nil {
			nodeID := nextComponentNodeID(s.editSession.DraftCard.Document, s.editSession.TargetCardID+"-border")
			s.editSession.DraftCard.Document.Root.Children = append(s.editSession.DraftCard.Document.Root.Children, cardcomponent.Node{
				ID:            nodeID,
				ComponentKind: border.Kind,
				Config:        mustRaw(part),
			})
			s.editSession.SelectedComponentID = nodeID
		} else {
			node.Config = mustRaw(part)
			s.editSession.SelectedComponentID = node.ID
		}
	default:
		if componentKind == "" {
			return Snapshot{}, fmt.Errorf("%s is not a component card", component.Name)
		}
		return Snapshot{}, fmt.Errorf("component kind %q is not supported yet", componentKind)
	}

	s.editSession.PendingConsumedComponentIDs = append(s.editSession.PendingConsumedComponentIDs, componentCardID)
	s.lastMessage = component.Name + " added to the draft."
	return s.snapshotLocked()
}

func (s *Session) SelectEditComponent(componentID, componentKind string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := s.editComponentNode(componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	s.editSession.SelectedComponentID = node.ID
	s.lastMessage = componentEditLabel(node.ComponentKind) + " edit controls opened."
	return s.snapshotLocked()
}

func (s *Session) ApplyEditControl(componentID, control string, value json.RawMessage) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := s.editComponentNode(componentID, "")
	if err != nil {
		return Snapshot{}, err
	}
	control = strings.TrimSpace(control)
	if err := applyGameComponentControl(node, control, value); err != nil {
		return Snapshot{}, err
	}
	s.editSession.SelectedComponentID = node.ID
	s.lastMessage = fmt.Sprintf("%s %s updated.", s.editSession.DraftCard.Name, control)
	return s.snapshotLocked()
}

func (s *Session) ApplyLibraryComponentControl(cardID, componentID, componentKind, control string, value json.RawMessage) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cardID = strings.TrimSpace(cardID)
	index := s.libraryCardIndex(cardID)
	if index < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", cardID)
	}
	root := &s.library[index].Document.Root
	node, err := componentNode(root, componentID, componentKind, s.library[index].Name)
	if err != nil {
		return Snapshot{}, err
	}
	if strings.TrimSpace(node.ComponentKind) != slider.Kind {
		return Snapshot{}, fmt.Errorf("component kind %q does not support library controls", node.ComponentKind)
	}
	if err := applyGameComponentControl(node, strings.TrimSpace(control), value); err != nil {
		return Snapshot{}, err
	}
	s.lastMessage = componentEditLabel(node.ComponentKind) + " updated in " + s.library[index].Name + "."
	return s.snapshotLocked()
}

func (s *Session) SaveEdit() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	targetIndex := s.libraryCardIndex(s.editSession.TargetCardID)
	if targetIndex < 0 {
		return Snapshot{}, fmt.Errorf("target card %q is not in your library", s.editSession.TargetCardID)
	}

	card := cloneValue(s.editSession.DraftCard)
	card.ID = s.editSession.TargetCardID
	card.Collectible = false
	card.Collected = true
	card.Document.CardID = card.ID
	if card.State == nil {
		card.State = map[string]any{}
	}
	card.State["editable"] = true

	installedKinds := map[string]bool{}
	if findNodeByKind(card.Document.Root, slider.Kind) != nil {
		installedKinds[slider.Kind] = true
		card.Tags = appendStringOnce(card.Tags, "controller")
		card.Tags = appendStringOnce(card.Tags, "slider-controller")
		card.State["built"] = true
		if card.ID == BlankControllerCardID {
			card.Name = "Regulator Controller"
			card.Document.Name = "Regulator Controller"
		}
	}
	if findNodeByKind(card.Document.Root, border.Kind) != nil {
		installedKinds[border.Kind] = true
	}
	for kind := range installedKinds {
		card.State["installedComponents"] = appendStateStringOnce(card.State["installedComponents"], kind)
	}

	s.library[targetIndex] = card
	pending := map[string]bool{}
	for _, cardID := range s.editSession.PendingConsumedComponentIDs {
		pending[cardID] = true
	}
	if len(pending) > 0 {
		next := make([]Card, 0, len(s.library))
		for _, candidate := range s.library {
			if pending[candidate.ID] && candidate.ID != card.ID {
				continue
			}
			next = append(next, candidate)
		}
		s.library = next
	}
	s.lastMessage = card.Name + " saved to your library."
	s.editSession = nil
	return s.snapshotLocked()
}

func (s *Session) CancelEdit() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	cardName := s.editSession.DraftCard.Name
	s.editSession = nil
	s.lastMessage = "Canceled editing " + cardName + "."
	return s.snapshotLocked()
}

func (s *Session) SaveController(templateCardID string, document cardcomponent.Document) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	templateCardID = strings.TrimSpace(templateCardID)
	if templateCardID != BlankControllerCardID {
		return Snapshot{}, fmt.Errorf("templateCardId must be %q", BlankControllerCardID)
	}
	if s.libraryCard(BlankControllerCardID) == nil {
		return Snapshot{}, fmt.Errorf("blank controller must be in your library")
	}
	if s.libraryCard(SliderComponentCardID) == nil {
		return Snapshot{}, fmt.Errorf("slider component must be in your library")
	}
	sliderPart, err := validatedSliderFromDocument(document)
	if err != nil {
		return Snapshot{}, err
	}
	controller := Card{
		ID:          RegulatorControllerCardID,
		Name:        "Regulator Controller",
		Kind:        KindItem,
		Tags:        []string{"item", "controller", "slider-controller"},
		Collectible: false,
		Collected:   true,
		State: map[string]any{
			"created":  true,
			"template": BlankControllerCardID,
		},
		Document: regulatorControllerDocument(sliderPart),
	}
	for index := range s.library {
		if s.library[index].ID == RegulatorControllerCardID {
			s.library[index] = controller
			s.lastMessage = "Regulator Controller updated in your library."
			return s.snapshotLocked()
		}
	}
	s.library = append(s.library, controller)
	s.lastMessage = "Regulator Controller saved to your library."
	return s.snapshotLocked()
}

func (s *Session) snapshotLocked() (Snapshot, error) {
	if len(s.worldDeck) == 0 {
		return Snapshot{}, fmt.Errorf("world deck is empty")
	}
	if s.activeIndex < 0 || s.activeIndex >= len(s.worldDeck) {
		s.activeIndex = 0
	}
	worldDeck := cloneCards(s.worldDeck)
	library := cloneCards(s.library)
	var editSession *EditSession
	if s.editSession != nil {
		edit := cloneValue(*s.editSession)
		editSession = &edit
	}
	return Snapshot{
		WorldDeck:                worldDeck,
		ActiveWorldCard:          cloneValue(s.worldDeck[s.activeIndex]),
		ActiveWorldCardID:        s.worldDeck[s.activeIndex].ID,
		ActiveIndex:              s.activeIndex,
		ActiveEditingComponentID: s.activeEditingComponentID,
		Library:                  library,
		EditSession:              editSession,
		SolvedFlags:              cloneValue(s.solvedFlags),
		Message:                  s.lastMessage,
	}, nil
}

func (s *Session) editComponentNode(componentID, componentKind string) (*cardcomponent.Node, error) {
	if s.editSession == nil {
		return nil, fmt.Errorf("start editing a card first")
	}
	componentID = strings.TrimSpace(componentID)
	componentKind = strings.TrimSpace(componentKind)
	if componentID == "" && componentKind == "" {
		componentID = strings.TrimSpace(s.editSession.SelectedComponentID)
	}
	if componentID == "" && componentKind == "" {
		if node := findNodeByKind(s.editSession.DraftCard.Document.Root, slider.Kind); node != nil {
			componentID = node.ID
		} else if node := findNodeByKind(s.editSession.DraftCard.Document.Root, border.Kind); node != nil {
			componentID = node.ID
		}
	}
	node, err := componentNode(&s.editSession.DraftCard.Document.Root, componentID, componentKind, s.editSession.DraftCard.Name)
	if err != nil {
		return nil, err
	}
	switch node.ComponentKind {
	case slider.Kind, border.Kind:
		return node, nil
	default:
		return nil, fmt.Errorf("component kind %q does not support edit controls", node.ComponentKind)
	}
}

func (s *Session) worldComponentNode(cardID, componentID, componentKind string) (int, *cardcomponent.Node, error) {
	cardID = strings.TrimSpace(cardID)
	componentID = strings.TrimSpace(componentID)
	componentKind = strings.TrimSpace(componentKind)
	if cardID == "" {
		if len(s.worldDeck) == 0 {
			return -1, nil, fmt.Errorf("world deck is empty")
		}
		cardID = s.worldDeck[s.activeIndex].ID
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return -1, nil, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	node, err := componentNode(&s.worldDeck[index].Document.Root, componentID, componentKind, s.worldDeck[index].Name)
	if err != nil {
		return -1, nil, err
	}
	return index, node, nil
}

func componentNode(root *cardcomponent.Node, componentID, componentKind, cardName string) (*cardcomponent.Node, error) {
	componentID = strings.TrimSpace(componentID)
	componentKind = strings.TrimSpace(componentKind)
	var node *cardcomponent.Node
	if componentID != "" {
		node = findNodeByIDPtr(root, componentID)
	} else if componentKind != "" {
		node = findNodeByKindPtr(root, componentKind)
	}
	if node == nil {
		if componentID != "" {
			return nil, fmt.Errorf("component %q is not on %s", componentID, cardName)
		}
		return nil, fmt.Errorf("%s has no %s component", cardName, componentKind)
	}
	if componentKind != "" && node.ComponentKind != componentKind {
		return nil, fmt.Errorf("component %q is %s, not %s", node.ID, node.ComponentKind, componentKind)
	}
	return node, nil
}

func (s *Session) requireWorldComponentEditable(componentKind string) error {
	componentKind = strings.TrimSpace(componentKind)
	if !worldComponentKindEditable(componentKind) {
		return fmt.Errorf("component kind %q does not support active card editing", componentKind)
	}
	switch componentKind {
	case background.Kind, border.Kind:
		if !s.libraryHasComponentKind(componentKind) {
			return fmt.Errorf("%s controls require finding a %s component card", componentEditLabel(componentKind), componentKind)
		}
	}
	return nil
}

func (s *Session) libraryHasComponentKind(componentKind string) bool {
	componentKind = strings.TrimSpace(componentKind)
	for _, card := range s.library {
		if stateString(card.State, "componentKind") == componentKind {
			return true
		}
	}
	return false
}

func (s *Session) worldCardIndex(cardID string) int {
	for index, card := range s.worldDeck {
		if card.ID == cardID {
			return index
		}
	}
	return -1
}

func (s *Session) libraryCard(cardID string) *Card {
	for index := range s.library {
		if s.library[index].ID == cardID {
			return &s.library[index]
		}
	}
	return nil
}

func (s *Session) libraryCardIndex(cardID string) int {
	for index := range s.library {
		if s.library[index].ID == cardID {
			return index
		}
	}
	return -1
}

func materializeDeck(definition DeckDefinition) ([]Card, map[string]map[string]cardcomponent.Document, map[string]CardDefinition, int, error) {
	worldDeck := make([]Card, 0, len(definition.Cards))
	documentVariants := make(map[string]map[string]cardcomponent.Document, len(definition.Cards))
	cardDefinitions := make(map[string]CardDefinition, len(definition.Cards))
	activeIndex := -1
	for index, card := range definition.Cards {
		document, ok := card.Documents[card.InitialDocument]
		if !ok {
			return nil, nil, nil, 0, fmt.Errorf("card %q initial document variant %q does not exist", card.ID, card.InitialDocument)
		}
		cardDefinitions[card.ID] = cloneValue(card)
		documentVariants[card.ID] = cloneValue(card.Documents)
		worldDeck = append(worldDeck, Card{
			ID:          card.ID,
			Name:        card.Name,
			Kind:        card.Kind,
			Tags:        append([]string(nil), card.Tags...),
			Collectible: card.Collectible,
			State:       cloneValue(card.State),
			Document:    cloneValue(document),
		})
		if card.ID == definition.InitialActiveCardID {
			activeIndex = index
		}
	}
	if activeIndex < 0 {
		return nil, nil, nil, 0, fmt.Errorf("initial active card %q does not exist", definition.InitialActiveCardID)
	}
	return worldDeck, documentVariants, cardDefinitions, activeIndex, nil
}

func (s *Session) ruleBaseMatches(rule UseRuleDefinition, source Card, target Card) bool {
	if !cardMatches(source, rule.Source) || !cardMatches(target, rule.Target) {
		return false
	}
	for flag, value := range rule.FlagConditions {
		if s.solvedFlags[flag] != value {
			return false
		}
	}
	return true
}

func sourceComponentConditionsMatch(conditions []ComponentConditionDefinition, document cardcomponent.Document) bool {
	for _, condition := range conditions {
		switch strings.TrimSpace(condition.ComponentKind) {
		case slider.Kind:
			part, ok := firstSliderConfig(document)
			if !ok || condition.ValueEquals == nil {
				return false
			}
			if part.Value != *condition.ValueEquals {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validatedSliderFromDocument(document cardcomponent.Document) (slider.Config, error) {
	part, ok := firstSliderConfig(document)
	if !ok {
		return slider.Config{}, fmt.Errorf("controller document must include a slider component")
	}
	return part, nil
}

func firstSliderConfig(document cardcomponent.Document) (slider.Config, bool) {
	var out slider.Config
	found := false
	var visit func(cardcomponent.Node)
	visit = func(node cardcomponent.Node) {
		if found {
			return
		}
		if node.ComponentKind == slider.Kind {
			var part slider.Config
			if err := json.Unmarshal(node.Config, &part); err != nil {
				return
			}
			generated := design.GeneratedConfig[slider.Config]{
				ComponentKind: slider.Kind,
				Description:   "Controller slider",
				Config:        part,
			}
			slider.NormalizeGenerated(&generated)
			if issues := slider.ValidateGenerated(generated); len(issues) > 0 {
				return
			}
			out = generated.Config
			found = true
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document.Root)
	return out, found
}

func regulatorControllerDocument(part slider.Config) cardcomponent.Document {
	part = slider.NormalizeConfig(part)
	part.Label = "Output"
	return cardcomponent.Document{
		CardID: RegulatorControllerCardID,
		Name:   "Regulator Controller",
		Root: cardcomponent.Node{
			ID:            RegulatorControllerCardID + "-root",
			ComponentKind: cardcomponent.Kind,
			Config:        cardcomponent.EncodeRootConfig(cardcomponent.RootConfig{PaddingPX: 18, Shadow: "0 24px 60px rgba(8,47,73,0.34)"}),
			Children: []cardcomponent.Node{
				{
					ID:            RegulatorControllerCardID + "-background",
					ComponentKind: background.Kind,
					Config: mustRaw(background.Config{
						BackgroundColor: "#082f49",
						CSS:             "background: linear-gradient(160deg, #082f49 0%, #0f172a 100%);",
					}),
				},
				{
					ID:            RegulatorControllerCardID + "-border",
					ComponentKind: border.Kind,
					Config: mustRaw(border.Config{
						BorderWidthPX:  2,
						BorderRadiusPX: 22,
						BorderColor:    "#7dd3fc",
						CSS:            "border: 2px solid #7dd3fc; border-radius: 22px;",
					}),
				},
				{
					ID:            RegulatorControllerCardID + "-title",
					ComponentKind: textarea.Kind,
					Config: mustRaw(textarea.Config{
						Content:    "REGULATOR",
						FontFamily: "system",
						FontSizePX: 24,
						FontWeight: 800,
						FontStyle:  "normal",
						Color:      "#e0f2fe",
						Align:      "center",
						Position:   "center",
						X:          50,
						Y:          20,
						PaddingPX:  4,
						CSS:        "text-align: center;",
					}),
				},
				{
					ID:            RegulatorControllerCardID + "-hint",
					ComponentKind: textarea.Kind,
					Config: mustRaw(textarea.Config{
						Content:    "Output setpoint",
						FontFamily: "system",
						FontSizePX: 14,
						FontWeight: 700,
						FontStyle:  "normal",
						Color:      "#bae6fd",
						Align:      "center",
						Position:   "center",
						X:          50,
						Y:          40,
						PaddingPX:  4,
						CSS:        "text-align: center;",
					}),
				},
				{
					ID:            "regulator-output-slider",
					ComponentKind: slider.Kind,
					Config:        mustRaw(part),
				},
			},
		},
	}
}

func cardMatches(card Card, matcher CardMatcherDefinition) bool {
	if strings.TrimSpace(matcher.ID) != "" && card.ID != matcher.ID {
		return false
	}
	for _, tag := range matcher.Tags {
		if !hasTag(card, tag) {
			return false
		}
	}
	return true
}

func (s *Session) applyRuleEffects(rule UseRuleDefinition, source Card, target Card) error {
	for _, effect := range rule.Effects {
		switch effect.EffectKind {
		case EffectSetFlag:
			if s.solvedFlags == nil {
				s.solvedFlags = map[string]bool{}
			}
			value, ok := effect.Value.(bool)
			if !ok {
				return fmt.Errorf("%s effect value must be a boolean", EffectSetFlag)
			}
			s.solvedFlags[effect.Flag] = value
		case EffectSetCardState:
			if err := s.updateEffectCard(effect, target, func(card *Card) error {
				if card.State == nil {
					card.State = map[string]any{}
				}
				card.State[effect.Key] = cloneValue(effect.Value)
				return nil
			}); err != nil {
				return err
			}
		case EffectRemoveCardTags:
			if err := s.updateEffectCard(effect, target, func(card *Card) error {
				for _, tag := range effect.Tags {
					card.Tags = removeString(card.Tags, tag)
				}
				return nil
			}); err != nil {
				return err
			}
		case EffectSetDocumentVariant:
			if err := s.updateEffectCard(effect, target, func(card *Card) error {
				variants := s.documentVariants[card.ID]
				document, ok := variants[effect.Variant]
				if !ok {
					return fmt.Errorf("card %q document variant %q does not exist", card.ID, effect.Variant)
				}
				card.Document = cloneValue(document)
				return nil
			}); err != nil {
				return err
			}
		case EffectSetMessage:
			s.lastMessage = effect.Message
		case EffectLoadDeck:
			if err := s.loadDeck(effect.DeckID); err != nil {
				return err
			}
		case EffectCopySourceComponent:
			if err := s.copySourceComponentToEffectCard(effect, source, target); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported effect kind %q", effect.EffectKind)
		}
	}
	return nil
}

func (s *Session) applyRuleFailureEffects(rule UseRuleDefinition, source Card, target Card) error {
	for _, effect := range rule.Effects {
		if effect.EffectKind != EffectCopySourceComponent {
			continue
		}
		if err := s.copySourceComponentToEffectCard(effect, source, target); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) copySourceComponentToEffectCard(effect RuleEffectDefinition, source Card, target Card) error {
	componentKind := strings.TrimSpace(effect.ComponentKind)
	if componentKind != slider.Kind {
		return fmt.Errorf("%s effect currently supports componentKind %q", EffectCopySourceComponent, slider.Kind)
	}
	sourceNode := findNodeByKind(source.Document.Root, componentKind)
	if sourceNode == nil {
		return fmt.Errorf("%s source card %q has no %s component", EffectCopySourceComponent, source.ID, componentKind)
	}
	targetNode := cloneValue(*sourceNode)
	if componentID := strings.TrimSpace(effect.ComponentID); componentID != "" {
		targetNode.ID = componentID
	}
	return s.updateEffectCard(effect, target, func(card *Card) error {
		appendOrReplaceRootChild(&card.Document.Root, targetNode)
		return nil
	})
}

func (s *Session) powerGeneratorIfTuned(cardIndex int, selectedComponentID string) (bool, error) {
	if cardIndex < 0 || cardIndex >= len(s.worldDeck) {
		return false, nil
	}
	card := &s.worldDeck[cardIndex]
	if card.ID != "generator-panel" || s.solvedFlags[GeneratorPoweredFlag] {
		return false, nil
	}
	targetValue, ok := stateInt(card.State, "targetValue")
	if !ok {
		return false, nil
	}
	sliderNode := findNodeByKind(card.Document.Root, slider.Kind)
	if sliderNode == nil {
		return false, nil
	}
	part, err := decodeSliderNode(*sliderNode)
	if err != nil {
		return false, err
	}
	if part.Value != targetValue {
		return false, nil
	}
	mountedSlider := cloneValue(*sliderNode)
	variants := s.documentVariants[card.ID]
	activeDocument, ok := variants["active"]
	if !ok {
		return false, fmt.Errorf("card %q document variant %q does not exist", card.ID, "active")
	}
	card.Document = cloneValue(activeDocument)
	appendOrReplaceRootChild(&card.Document.Root, mountedSlider)
	if card.State == nil {
		card.State = map[string]any{}
	}
	card.State["powered"] = true
	card.State["useful"] = true
	card.Tags = removeString(card.Tags, "inactive")
	if s.solvedFlags == nil {
		s.solvedFlags = map[string]bool{}
	}
	s.solvedFlags[GeneratorPoweredFlag] = true
	if strings.TrimSpace(selectedComponentID) != "" {
		s.activeEditingComponentID = selectedComponentID
	}
	s.lastMessage = "The regulator locks at 73. The generator comes fully online."
	return true, nil
}

func (s *Session) loadDeck(deckID string) error {
	deckID = strings.TrimSpace(deckID)
	if s.loadedDecks[deckID] {
		return nil
	}
	definition, err := LoadEmbeddedDeck(deckID)
	if err != nil {
		return err
	}
	if err := ValidateDeckPackDefinition(definition, s.cardDefinitions); err != nil {
		return err
	}
	worldDeck, documentVariants, cardDefinitions, activeIndex, err := materializeDeck(definition)
	if err != nil {
		return err
	}
	if s.solvedFlags == nil {
		s.solvedFlags = map[string]bool{}
	}
	for flag, value := range definition.InitialSolvedFlags {
		if _, exists := s.solvedFlags[flag]; !exists {
			s.solvedFlags[flag] = value
		}
	}
	startIndex := len(s.worldDeck)
	s.worldDeck = append(s.worldDeck, worldDeck...)
	if s.documentVariants == nil {
		s.documentVariants = map[string]map[string]cardcomponent.Document{}
	}
	for cardID, documents := range documentVariants {
		s.documentVariants[cardID] = documents
	}
	if s.cardDefinitions == nil {
		s.cardDefinitions = map[string]CardDefinition{}
	}
	for cardID, card := range cardDefinitions {
		s.cardDefinitions[cardID] = card
	}
	s.useRules = append(s.useRules, cloneValue(definition.UseRules)...)
	if s.loadedDecks == nil {
		s.loadedDecks = map[string]bool{}
	}
	s.loadedDecks[definition.ID] = true
	s.activeIndex = startIndex + activeIndex
	return nil
}

func (s *Session) updateEffectCard(effect RuleEffectDefinition, target Card, update func(*Card) error) error {
	cardID := effect.CardID
	if strings.TrimSpace(cardID) == "" {
		cardID = target.ID
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return fmt.Errorf("effect %q references card %q outside world deck", effect.EffectKind, cardID)
	}
	card := s.worldDeck[index]
	if err := update(&card); err != nil {
		return err
	}
	s.worldDeck[index] = card
	return nil
}

func hasTag(card Card, tag string) bool {
	for _, candidate := range card.Tags {
		if candidate == tag {
			return true
		}
	}
	return false
}

func removeString(values []string, value string) []string {
	out := values[:0]
	for _, candidate := range values {
		if candidate != value {
			out = append(out, candidate)
		}
	}
	return out
}

func cloneCards(cards []Card) []Card {
	if len(cards) == 0 {
		return nil
	}
	out := make([]Card, len(cards))
	for index, card := range cards {
		out[index] = cloneValue(card)
	}
	return out
}

func mustRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
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
