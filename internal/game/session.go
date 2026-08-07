package game

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const (
	KindWorld = "world"
	KindItem  = "item"
	KindClue  = "clue"
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
	revision                 uint64
	registry                 *cardcomponent.Registry
	deckDefinition           DeckDefinition
	cardDefinitions          map[string]CardDefinition
	documentVariants         map[string]map[string]cardcomponent.Document
	loadedDecks              map[string]bool
	rules                    []RuleDefinition
	worldDeck                []Card
	activeIndex              int
	activeEditingComponentID string
	library                  []Card
	editSession              *EditSession
	solvedFlags              map[string]bool
	lastMessage              string
}

func NewSession(registry *cardcomponent.Registry) *Session {
	session, err := NewSessionFromEmbeddedDeck(registry)
	if err != nil {
		panic(err)
	}
	return session
}

func NewSessionFromEmbeddedDeck(registry *cardcomponent.Registry) (*Session, error) {
	definition, err := LoadEmbeddedSeededWorldDeck(registry)
	if err != nil {
		return nil, err
	}
	return NewSessionFromDeck(registry, definition)
}

func NewSessionFromDeck(registry *cardcomponent.Registry, definition DeckDefinition) (*Session, error) {
	if registry == nil {
		return nil, fmt.Errorf("component registry is not initialized")
	}
	if err := ValidateDeckDefinition(registry, definition); err != nil {
		return nil, err
	}
	definition = cloneValue(definition)
	worldDeck, documentVariants, cardDefinitions, activeIndex, err := materializeDeck(definition)
	if err != nil {
		return nil, err
	}
	return &Session{
		registry:         registry,
		deckDefinition:   definition,
		cardDefinitions:  cardDefinitions,
		documentVariants: documentVariants,
		loadedDecks:      map[string]bool{definition.ID: true},
		rules:            cloneValue(definition.Rules),
		worldDeck:        worldDeck,
		activeIndex:      activeIndex,
		library:          nil,
		solvedFlags:      cloneValue(definition.InitialSolvedFlags),
		lastMessage:      definition.InitialMessage,
	}, nil
}

func (s *Session) resetLocked(events *eventCollector) (Snapshot, error) {
	next, err := NewSessionFromDeck(s.registry, s.deckDefinition)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	s.cardDefinitions = next.cardDefinitions
	s.documentVariants = next.documentVariants
	s.loadedDecks = next.loadedDecks
	s.rules = next.rules
	s.worldDeck = next.worldDeck
	s.activeIndex = next.activeIndex
	s.activeEditingComponentID = next.activeEditingComponentID
	s.library = next.library
	s.editSession = next.editSession
	s.solvedFlags = next.solvedFlags
	s.lastMessage = next.lastMessage
	events.emit(EventSessionReset, SessionResetPayload{})
	events.message(s.lastMessage)
	return s.commandSnapshotLocked()
}

func (s *Session) cycleLocked(direction string, events *eventCollector) (Snapshot, error) {
	if len(s.worldDeck) == 0 {
		return Snapshot{}, fmt.Errorf("world deck is empty")
	}
	previousCardID := s.worldDeck[s.activeIndex].ID
	normalizedDirection := strings.TrimSpace(direction)
	switch normalizedDirection {
	case "previous", "prev", "back":
		s.activeIndex--
	case "", "next":
		s.activeIndex++
		normalizedDirection = "next"
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
	events.emit(EventCardCycled, CardCycledPayload{
		Direction:      normalizedDirection,
		PreviousCardID: previousCardID,
		ActiveCardID:   s.worldDeck[s.activeIndex].ID,
	})
	s.setMessageLocked("The next card slides into view.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) collectLocked(cardID string, events *eventCollector) (Snapshot, error) {
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
	libraryCard := cloneValue(card)
	libraryCard.Collectible = false
	libraryCard.Collected = true
	s.library = append(s.library, libraryCard)
	s.worldDeck = append(s.worldDeck[:index], s.worldDeck[index+1:]...)
	switch {
	case index < s.activeIndex:
		s.activeIndex--
	case s.activeIndex >= len(s.worldDeck):
		s.activeIndex = 0
	}
	s.activeEditingComponentID = ""
	events.emit(EventCardCollected, CardCollectedPayload{
		CardID:             card.ID,
		PreviousWorldIndex: index,
		ActiveCardID:       s.worldDeck[s.activeIndex].ID,
	})
	s.setMessageLocked(card.Name+" moved into your library.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) playCardLocked(sourceCardID, targetCardID string, events *eventCollector) (Snapshot, error) {
	sourceCardID = strings.TrimSpace(sourceCardID)
	targetCardID = strings.TrimSpace(targetCardID)
	sourceIndex := s.libraryCardIndex(sourceCardID)
	if sourceIndex < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", sourceCardID)
	}
	source := cloneValue(s.library[sourceIndex])
	if targetCardID == "" && len(s.worldDeck) > 0 {
		targetCardID = s.worldDeck[s.activeIndex].ID
	}
	targetIndex := s.worldCardIndex(targetCardID)
	if targetIndex < 0 {
		return Snapshot{}, fmt.Errorf("target card %q is not in the world deck", targetCardID)
	}
	target := s.worldDeck[targetIndex]
	played := &CardPlayedPayload{SourceCardID: source.ID, TargetCardID: target.ID}
	events.emit(EventCardPlayed, played)
	resolution, err := s.runRules(CardPlayedSignal{SourceCardID: source.ID, TargetCardID: target.ID}, events)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	switch resolution.Outcome {
	case RuleOutcomeSuccess:
		played.Outcome = "resolved"
		if !resolution.RetainSource {
			s.removeLibraryCard(source.ID)
			events.emit(EventCardConsumed, CardConsumedPayload{CardID: source.ID})
		}
		s.activeEditingComponentID = ""
		s.ensureMessageEventLocked(events)
		return s.commandSnapshotLocked()
	case RuleOutcomeConditionsFailed:
		played.Outcome = "conditionsFailed"
		events.emit(EventActionRejected, ActionRejectedPayload{Action: "playCard", Outcome: played.Outcome})
		if !resolution.RetainSource {
			s.removeLibraryCard(source.ID)
			events.emit(EventCardConsumed, CardConsumedPayload{CardID: source.ID})
		}
		s.activeEditingComponentID = ""
		if !events.hasMessage() {
			s.setMessageLocked(source.Name+" was played, but its conditions were not met.", events)
		}
		return s.commandSnapshotLocked()
	case RuleOutcomeNoMatch:
		played.Outcome = "noMatchingRule"
		events.emit(EventActionRejected, ActionRejectedPayload{Action: "playCard", Outcome: played.Outcome})
		s.setMessageLocked("Nothing on this card responds to "+source.Name+".", events)
		return s.commandSnapshotLocked()
	default:
		return Snapshot{}, failExecution(fmt.Errorf("unsupported rule outcome %q", resolution.Outcome))
	}
}

func (s *Session) submitFormLocked(cardID, formID string, fields map[string]string, events *eventCollector) (Snapshot, error) {
	cardID = strings.TrimSpace(cardID)
	formID = strings.TrimSpace(formID)
	if cardID == "" || formID == "" {
		return Snapshot{}, fmt.Errorf("cardId and formId are required")
	}
	if len(fields) > 16 {
		return Snapshot{}, fmt.Errorf("form may contain at most 16 fields")
	}
	for name, value := range fields {
		if !cardcomponent.ValidComponentID(name) {
			return Snapshot{}, fmt.Errorf("form field name %q is invalid", name)
		}
		if len([]rune(value)) > 128 {
			return Snapshot{}, fmt.Errorf("form field %q must be at most 128 characters", name)
		}
	}
	targetIndex := s.worldCardIndex(cardID)
	if targetIndex < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	target := s.worldDeck[targetIndex]
	requiredFields, acceptsForm := s.formRuleFieldNames(target, formID)
	if !acceptsForm {
		return Snapshot{}, fmt.Errorf("form %q does not accept submissions for %s", formID, target.Name)
	}
	if !documentHasSubmitButton(s.registry, target.Document, formID) ||
		!documentHasNamedFormFields(s.registry, target.Document, formID, requiredFields) {
		return Snapshot{}, fmt.Errorf("form %q is not mounted on %s", formID, target.Name)
	}
	events.emit(EventFormSubmitted, FormSubmittedPayload{CardID: cardID, FormID: formID})
	worldCardCount := len(s.worldDeck)
	resolution, err := s.runRules(FormSubmittedSignal{CardID: cardID, FormID: formID, Fields: cloneValue(fields)}, events)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	if len(s.worldDeck) == worldCardCount {
		s.activeIndex = targetIndex
	}
	switch resolution.Outcome {
	case RuleOutcomeSuccess:
		s.activeEditingComponentID = ""
		s.ensureMessageEventLocked(events)
		return s.commandSnapshotLocked()
	case RuleOutcomeConditionsFailed:
		events.emit(EventActionRejected, ActionRejectedPayload{Action: "submitForm", Outcome: "conditionsFailed"})
		s.ensureMessageEventLocked(events)
		return s.commandSnapshotLocked()
	case RuleOutcomeNoMatch:
		return Snapshot{}, failExecution(fmt.Errorf("form %q stopped accepting submissions for %s", formID, target.Name))
	default:
		return Snapshot{}, failExecution(fmt.Errorf("unsupported rule outcome %q", resolution.Outcome))
	}
}

func (s *Session) selectWorldComponentLocked(cardID, componentID, componentKind string, events *eventCollector) (Snapshot, error) {
	index, node, err := s.worldComponentNode(cardID, componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.requireWorldComponentEditable(node.ComponentKind); err != nil {
		return Snapshot{}, err
	}
	s.activeIndex = index
	s.activeEditingComponentID = node.ID
	events.emit(EventComponentSelected, ComponentPayload{
		CardID:        s.worldDeck[index].ID,
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Scope:         "world",
	})
	s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" edit controls opened.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) changeWorldComponentLocked(cardID, componentID, componentKind, control string, value json.RawMessage, events *eventCollector) (Snapshot, error) {
	index, node, err := s.worldComponentNode(cardID, componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.requireWorldComponentEditable(node.ComponentKind); err != nil {
		return Snapshot{}, err
	}
	control = strings.TrimSpace(control)
	if err := applyGameComponentControl(s.registry, node, control, value); err != nil {
		return Snapshot{}, err
	}
	s.activeIndex = index
	s.activeEditingComponentID = node.ID
	events.emit(EventComponentChanged, ComponentPayload{
		CardID:        s.worldDeck[index].ID,
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Control:       control,
		Scope:         "world",
	})
	_, err = s.runRules(ComponentUpdatedSignal{
		CardID: s.worldDeck[index].ID, ComponentID: node.ID,
		ComponentKind: node.ComponentKind, Component: cloneValue(*node),
	}, events)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	if !events.hasMessage() {
		s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" updated.", events)
	}
	return s.commandSnapshotLocked()
}

func (s *Session) startEditingLocked(cardID string, events *eventCollector) (Snapshot, error) {
	cardID = strings.TrimSpace(cardID)
	index := s.libraryCardIndex(cardID)
	if index < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", cardID)
	}
	card := s.library[index]
	_, isComponentCard := card.State[componentTemplateStateKey]
	if !stateBool(card.State, "editable") && !isComponentCard {
		return Snapshot{}, fmt.Errorf("%s cannot be edited", card.Name)
	}
	draft := cloneValue(card)
	selectedComponentID := ""
	if isComponentCard {
		template, err := componentTemplateFromCard(s.registry, card)
		if err != nil {
			return Snapshot{}, err
		}
		document, node, err := s.registry.InstallTemplate(draft.Document, template)
		if err != nil {
			return Snapshot{}, err
		}
		draft.Document = document
		selectedComponentID = node.ID
	}
	s.editSession = &EditSession{
		TargetCardID:        card.ID,
		DraftCard:           draft,
		SelectedComponentID: selectedComponentID,
	}
	s.activeEditingComponentID = ""
	events.emit(EventEditStarted, EditPayload{CardID: card.ID})
	s.setMessageLocked("Editing "+card.Name+".", events)
	return s.commandSnapshotLocked()
}

func (s *Session) installEditComponentLocked(componentCardID string, events *eventCollector) (Snapshot, error) {
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
	template, err := componentTemplateFromCard(s.registry, component)
	if err != nil {
		return Snapshot{}, err
	}
	document, node, err := s.registry.InstallTemplate(s.editSession.DraftCard.Document, template)
	if err != nil {
		return Snapshot{}, err
	}
	s.editSession.DraftCard.Document = document
	s.editSession.SelectedComponentID = node.ID

	s.editSession.PendingConsumedComponentIDs = append(s.editSession.PendingConsumedComponentIDs, componentCardID)
	events.emit(EventEditComponentInstalled, EditComponentInstalledPayload{
		CardID:          s.editSession.TargetCardID,
		ComponentCardID: componentCardID,
		ComponentID:     node.ID,
		ComponentKind:   node.ComponentKind,
	})
	s.setMessageLocked(component.Name+" added to the draft.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) selectEditComponentLocked(componentID, componentKind string, events *eventCollector) (Snapshot, error) {
	node, err := s.editComponentNode(componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	s.editSession.SelectedComponentID = node.ID
	events.emit(EventComponentSelected, ComponentPayload{
		CardID:        s.editSession.TargetCardID,
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Scope:         "edit",
	})
	s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" edit controls opened.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) changeEditComponentLocked(componentID, control string, value json.RawMessage, events *eventCollector) (Snapshot, error) {
	node, err := s.editComponentNode(componentID, "")
	if err != nil {
		return Snapshot{}, err
	}
	control = strings.TrimSpace(control)
	if err := applyGameComponentControl(s.registry, node, control, value); err != nil {
		return Snapshot{}, err
	}
	s.editSession.SelectedComponentID = node.ID
	events.emit(EventComponentChanged, ComponentPayload{
		CardID:        s.editSession.TargetCardID,
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Control:       control,
		Scope:         "edit",
	})
	s.setMessageLocked(fmt.Sprintf("%s %s updated.", s.editSession.DraftCard.Name, control), events)
	return s.commandSnapshotLocked()
}

func (s *Session) changeLibraryComponentLocked(cardID, componentID, componentKind, control string, value json.RawMessage, events *eventCollector) (Snapshot, error) {
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
	control = strings.TrimSpace(control)
	if err := applyGameComponentControl(s.registry, node, control, value); err != nil {
		return Snapshot{}, err
	}
	events.emit(EventComponentChanged, ComponentPayload{
		CardID:        s.library[index].ID,
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Control:       control,
		Scope:         "library",
	})
	s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" updated in "+s.library[index].Name+".", events)
	return s.commandSnapshotLocked()
}

func (s *Session) saveEditLocked(events *eventCollector) (Snapshot, error) {
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
	for _, value := range appendStateStringOnce(card.State["installedComponents"], "") {
		installedKinds[value] = true
	}
	if _, isComponentCard := card.State[componentTemplateStateKey]; isComponentCard {
		template, err := componentTemplateFromCard(s.registry, card)
		if err != nil {
			return Snapshot{}, err
		}
		installedKinds[template.ComponentKind] = true
		delete(card.State, componentTemplateStateKey)
		card.Tags = removeComponentCardTags(card.Tags)
	}
	for _, componentCardID := range s.editSession.PendingConsumedComponentIDs {
		if componentCard := s.libraryCard(componentCardID); componentCard != nil {
			if template, err := componentTemplateFromCard(s.registry, *componentCard); err == nil {
				installedKinds[template.ComponentKind] = true
			}
		}
	}
	kinds := make([]string, 0, len(installedKinds))
	for kind := range installedKinds {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		card.State["installedComponents"] = appendStateStringOnce(card.State["installedComponents"], kind)
		card.Tags = appendStringOnce(card.Tags, kind+"-controller")
	}
	if len(installedKinds) > 0 {
		card.Tags = appendStringOnce(card.Tags, "controller")
		card.State["built"] = true
	}

	s.library[targetIndex] = card
	events.emit(EventEditSaved, EditPayload{CardID: card.ID})
	pending := map[string]bool{}
	for _, cardID := range s.editSession.PendingConsumedComponentIDs {
		pending[cardID] = true
	}
	if len(pending) > 0 {
		next := make([]Card, 0, len(s.library))
		for _, candidate := range s.library {
			if pending[candidate.ID] && candidate.ID != card.ID {
				events.emit(EventCardConsumed, CardConsumedPayload{CardID: candidate.ID})
				continue
			}
			next = append(next, candidate)
		}
		s.library = next
	}
	s.setMessageLocked(card.Name+" saved to your library.", events)
	s.editSession = nil
	return s.commandSnapshotLocked()
}

func (s *Session) cancelEditLocked(events *eventCollector) (Snapshot, error) {
	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	cardID := s.editSession.TargetCardID
	cardName := s.editSession.DraftCard.Name
	s.editSession = nil
	events.emit(EventEditCanceled, EditPayload{CardID: cardID})
	s.setMessageLocked("Canceled editing "+cardName+".", events)
	return s.commandSnapshotLocked()
}

func (s *Session) setMessageLocked(message string, events *eventCollector) {
	s.lastMessage = message
	events.message(message)
}

func (s *Session) ensureMessageEventLocked(events *eventCollector) {
	if !events.hasMessage() {
		events.message(s.lastMessage)
	}
}

func (s *Session) commandSnapshotLocked() (Snapshot, error) {
	snapshot, err := s.snapshotLocked()
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	return snapshot, nil
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
	node, err := componentNode(&s.editSession.DraftCard.Document.Root, componentID, componentKind, s.editSession.DraftCard.Name)
	if err != nil {
		return nil, err
	}
	if definition, ok := s.registry.Lookup(node.ComponentKind); ok {
		if _, installable := definition.Install(); installable {
			return node, nil
		}
	}
	return nil, fmt.Errorf("component kind %q does not support edit controls", node.ComponentKind)
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
	if !worldComponentKindEditable(s.registry, componentKind) {
		return cardcomponent.NewUnsupportedOperationError(componentKind, "editing")
	}
	return nil
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

func (s *Session) removeLibraryCard(cardID string) bool {
	index := s.libraryCardIndex(cardID)
	if index < 0 {
		return false
	}
	s.library = append(s.library[:index], s.library[index+1:]...)
	return true
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

func (s *Session) formRuleFieldNames(target Card, formID string) ([]string, bool) {
	seen := map[string]bool{}
	var names []string
	accepts := false
	for _, rule := range s.rules {
		trigger := rule.Trigger.formSubmitted
		if rule.Trigger.kind != TriggerFormSubmitted || trigger == nil ||
			trigger.FormID != formID || !cardMatches(target, trigger.Target) {
			continue
		}
		accepts = true
		for _, condition := range rule.Conditions {
			if condition.kind != ConditionFormFieldEquals || condition.formFieldEquals == nil {
				continue
			}
			name := condition.formFieldEquals.Name
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names, accepts
}

func documentHasSubmitButton(registry *cardcomponent.Registry, document cardcomponent.Document, formID string) bool {
	found := false
	visitNodes(document.Root, func(node cardcomponent.Node) {
		if found {
			return
		}
		definition, ok := registry.Lookup(node.ComponentKind)
		if !ok || !definition.HasRole(cardcomponent.RoleFormSubmitter) {
			return
		}
		value, present, issues := definition.ReadProperty(node.Config, "form_id")
		found = len(issues) == 0 && present && value.Kind == schema.PropertyString && value.String == formID
	})
	return found
}

func documentHasNamedFormFields(registry *cardcomponent.Registry, document cardcomponent.Document, formID string, names []string) bool {
	available := map[string]bool{}
	visitNodes(document.Root, func(node cardcomponent.Node) {
		definition, ok := registry.Lookup(node.ComponentKind)
		if !ok || !definition.HasRole(cardcomponent.RoleFormField) {
			return
		}
		form, present, issues := definition.ReadProperty(node.Config, "form_id")
		if len(issues) > 0 || !present || form.Kind != schema.PropertyString || form.String != formID {
			return
		}
		name, present, issues := definition.ReadProperty(node.Config, "name")
		if len(issues) == 0 && present && name.Kind == schema.PropertyString {
			available[name.String] = true
		}
	})
	for _, name := range names {
		if !available[name] {
			return false
		}
	}
	return true
}

func visitNodes(node cardcomponent.Node, visit func(cardcomponent.Node)) {
	visit(node)
	for _, child := range node.Children {
		visitNodes(child, visit)
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

func (s *Session) loadDeck(deckID string) (bool, error) {
	deckID = strings.TrimSpace(deckID)
	if s.loadedDecks[deckID] {
		return false, nil
	}
	definition, err := LoadEmbeddedDeck(s.registry, deckID)
	if err != nil {
		return false, err
	}
	existingRuleIDs := make(map[string]bool, len(s.rules))
	for _, rule := range s.rules {
		existingRuleIDs[rule.ID] = true
	}
	if err := ValidateDeckPackDefinition(s.registry, definition, s.cardDefinitions, existingRuleIDs); err != nil {
		return false, err
	}
	worldDeck, documentVariants, cardDefinitions, activeIndex, err := materializeDeck(definition)
	if err != nil {
		return false, err
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
	s.rules = append(s.rules, cloneValue(definition.Rules)...)
	if s.loadedDecks == nil {
		s.loadedDecks = map[string]bool{}
	}
	s.loadedDecks[definition.ID] = true
	s.activeIndex = startIndex + activeIndex
	s.activeEditingComponentID = ""
	return true, nil
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

func removeComponentCardTags(values []string) []string {
	out := values[:0]
	for _, candidate := range values {
		if candidate == "component" || strings.HasSuffix(candidate, "-component") {
			continue
		}
		out = append(out, candidate)
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
