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

type Snapshot struct {
	WorldDeck                []CardSnapshot     `json:"worldDeck"`
	ActiveWorldCard          CardSnapshot       `json:"activeWorldCard"`
	ActiveWorldCardID        CardInstanceID     `json:"activeWorldCardId"`
	ActiveIndex              int                `json:"activeIndex"`
	ActiveEditingComponentID string             `json:"activeEditingComponentId,omitempty"`
	Library                  []CardSnapshot     `json:"library"`
	EditSession              *EditSession       `json:"editSession,omitempty"`
	Encounter                *EncounterSnapshot `json:"encounter,omitempty"`
	SolvedFlags              map[string]bool    `json:"solvedFlags"`
	Message                  string             `json:"message,omitempty"`
}

type EditSession struct {
	TargetCardID                CardInstanceID   `json:"targetCardId"`
	DraftCard                   CardSnapshot     `json:"draftCard"`
	PendingConsumedComponentIDs []CardInstanceID `json:"pendingConsumedComponentIds,omitempty"`
	SelectedComponentID         string           `json:"selectedComponentId,omitempty"`
}

type editSessionState struct {
	TargetInstanceID           CardInstanceID
	DraftInstance              CardInstance
	PendingConsumedInstanceIDs []CardInstanceID
	SelectedComponentID        string
}

type Session struct {
	mu                       sync.Mutex
	revision                 uint64
	registry                 *cardcomponent.Registry
	deckDefinition           DeckDefinition
	cardDefinitions          map[CardDefinitionID]CardDefinition
	loadedDecks              map[string]bool
	rules                    []RuleDefinition
	instances                map[CardInstanceID]CardInstance
	zones                    ZoneState
	activeSceneCardID        CardInstanceID
	activeEditingComponentID string
	editSession              *editSessionState
	encounter                *EncounterState
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
	materialized, err := materializeDeck(definition)
	if err != nil {
		return nil, err
	}
	session := &Session{
		registry:          registry,
		deckDefinition:    definition,
		cardDefinitions:   materialized.definitions,
		loadedDecks:       map[string]bool{definition.ID: true},
		rules:             cloneValue(definition.Rules),
		instances:         materialized.instances,
		zones:             materialized.zones,
		activeSceneCardID: materialized.activeID,
		encounter:         cloneValue(definition.InitialEncounter),
		solvedFlags:       cloneValue(definition.InitialSolvedFlags),
		lastMessage:       definition.InitialMessage,
	}
	if err := session.validateZoneState(); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Session) resetLocked(events *eventCollector) (Snapshot, error) {
	next, err := NewSessionFromDeck(s.registry, s.deckDefinition)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	s.cardDefinitions = next.cardDefinitions
	s.loadedDecks = next.loadedDecks
	s.rules = next.rules
	s.instances = next.instances
	s.zones = next.zones
	s.activeSceneCardID = next.activeSceneCardID
	s.activeEditingComponentID = next.activeEditingComponentID
	s.editSession = next.editSession
	s.encounter = next.encounter
	s.solvedFlags = next.solvedFlags
	s.lastMessage = next.lastMessage
	events.emit(EventSessionReset, SessionResetPayload{})
	events.message(s.lastMessage)
	return s.commandSnapshotLocked()
}

func (s *Session) cycleLocked(direction string, events *eventCollector) (Snapshot, error) {
	scene := s.zones[ZoneScene]
	if len(scene) == 0 {
		return Snapshot{}, fmt.Errorf("world deck is empty")
	}
	activeIndex := s.zoneIndex(ZoneScene, s.activeSceneCardID)
	if activeIndex < 0 {
		return Snapshot{}, failExecution(fmt.Errorf("active scene card %q is not in the scene", s.activeSceneCardID))
	}
	previousCardID := s.activeSceneCardID
	normalizedDirection := strings.TrimSpace(direction)
	switch normalizedDirection {
	case "previous", "prev", "back":
		activeIndex--
	case "", "next":
		activeIndex++
		normalizedDirection = "next"
	default:
		return Snapshot{}, fmt.Errorf("direction must be next or previous")
	}
	if activeIndex < 0 {
		activeIndex = len(scene) - 1
	}
	if activeIndex >= len(scene) {
		activeIndex = 0
	}
	s.activeSceneCardID = scene[activeIndex]
	s.activeEditingComponentID = ""
	events.emit(EventCardCycled, CardCycledPayload{
		Direction:      normalizedDirection,
		PreviousCardID: string(previousCardID),
		ActiveCardID:   string(s.activeSceneCardID),
	})
	s.setMessageLocked("The next card slides into view.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) collectLocked(cardID string, events *eventCollector) (Snapshot, error) {
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		cardID = string(s.activeSceneCardID)
	}
	instanceID := CardInstanceID(cardID)
	index := s.zoneIndex(ZoneScene, instanceID)
	if index < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	instance, _ := s.instance(instanceID)
	definition, ok := s.cardDefinitions[instance.DefinitionID]
	if !ok {
		return Snapshot{}, failExecution(fmt.Errorf("card instance %q references missing definition %q", instanceID, instance.DefinitionID))
	}
	if !definition.Collectible {
		return Snapshot{}, fmt.Errorf("%s cannot be collected", definition.Name)
	}
	move, err := s.moveCard(instanceID, ZoneScene, ZoneLibrary)
	if err != nil {
		return Snapshot{}, err
	}
	events.emit(EventCardMoved, CardMovedPayloadFromMove(move))
	s.activeEditingComponentID = ""
	events.emit(EventCardCollected, CardCollectedPayload{
		CardID:             cardID,
		PreviousWorldIndex: index,
		ActiveCardID:       string(s.activeSceneCardID),
	})
	s.setMessageLocked(definition.Name+" moved into your library.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) playCardLocked(sourceCardID, targetCardID string, events *eventCollector) (Snapshot, error) {
	sourceCardID = strings.TrimSpace(sourceCardID)
	targetCardID = strings.TrimSpace(targetCardID)
	sourceInstanceID := CardInstanceID(sourceCardID)
	if s.zoneIndex(ZoneLibrary, sourceInstanceID) < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", sourceCardID)
	}
	source, _ := s.instance(sourceInstanceID)
	sourceDefinition := s.cardDefinitions[source.DefinitionID]
	if targetCardID == "" {
		targetCardID = string(s.activeSceneCardID)
	}
	targetInstanceID := CardInstanceID(targetCardID)
	if s.zoneIndex(ZoneScene, targetInstanceID) < 0 {
		return Snapshot{}, fmt.Errorf("target card %q is not in the world deck", targetCardID)
	}
	played := &CardPlayedPayload{SourceCardID: sourceCardID, TargetCardID: targetCardID}
	events.emit(EventCardPlayed, played)
	resolution, err := s.runRules(CardPlayedSignal{SourceInstanceID: sourceInstanceID, TargetInstanceID: targetInstanceID}, events)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	switch resolution.Outcome {
	case RuleOutcomeSuccess:
		played.Outcome = "resolved"
		if err := s.advanceEncounter(events); err != nil {
			return Snapshot{}, failExecution(err)
		}
		s.activeEditingComponentID = ""
		s.ensureMessageEventLocked(events)
		return s.commandSnapshotLocked()
	case RuleOutcomeConditionsFailed:
		played.Outcome = "conditionsFailed"
		events.emit(EventActionRejected, ActionRejectedPayload{Action: "playCard", Outcome: played.Outcome})
		s.activeEditingComponentID = ""
		if err := s.advanceEncounter(events); err != nil {
			return Snapshot{}, failExecution(err)
		}
		if !events.hasMessage() {
			s.setMessageLocked(sourceDefinition.Name+" was played, but its conditions were not met.", events)
		}
		return s.commandSnapshotLocked()
	case RuleOutcomeNoMatch:
		played.Outcome = "noMatchingRule"
		events.emit(EventActionRejected, ActionRejectedPayload{Action: "playCard", Outcome: played.Outcome})
		if err := s.advanceEncounter(events); err != nil {
			return Snapshot{}, failExecution(err)
		}
		s.setMessageLocked("Nothing on this card responds to "+sourceDefinition.Name+".", events)
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
	instanceID := CardInstanceID(cardID)
	targetIndex := s.zoneIndex(ZoneScene, instanceID)
	if targetIndex < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	target, _ := s.instance(instanceID)
	targetDefinition := s.cardDefinitions[target.DefinitionID]
	requiredFields, acceptsForm := s.formRuleFieldNames(target, formID)
	if !acceptsForm {
		return Snapshot{}, fmt.Errorf("form %q does not accept submissions for %s", formID, targetDefinition.Name)
	}
	if !documentHasSubmitButton(s.registry, target.Document, formID) ||
		!documentHasNamedFormFields(s.registry, target.Document, formID, requiredFields) {
		return Snapshot{}, fmt.Errorf("form %q is not mounted on %s", formID, targetDefinition.Name)
	}
	events.emit(EventFormSubmitted, FormSubmittedPayload{CardID: cardID, FormID: formID})
	s.activeSceneCardID = instanceID
	resolution, err := s.runRules(FormSubmittedSignal{InstanceID: instanceID, FormID: formID, Fields: cloneValue(fields)}, events)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	switch resolution.Outcome {
	case RuleOutcomeSuccess:
		if err := s.advanceEncounter(events); err != nil {
			return Snapshot{}, failExecution(err)
		}
		s.activeEditingComponentID = ""
		s.ensureMessageEventLocked(events)
		return s.commandSnapshotLocked()
	case RuleOutcomeConditionsFailed:
		events.emit(EventActionRejected, ActionRejectedPayload{Action: "submitForm", Outcome: "conditionsFailed"})
		if err := s.advanceEncounter(events); err != nil {
			return Snapshot{}, failExecution(err)
		}
		s.ensureMessageEventLocked(events)
		return s.commandSnapshotLocked()
	case RuleOutcomeNoMatch:
		return Snapshot{}, failExecution(fmt.Errorf("form %q stopped accepting submissions for %s", formID, targetDefinition.Name))
	default:
		return Snapshot{}, failExecution(fmt.Errorf("unsupported rule outcome %q", resolution.Outcome))
	}
}

func (s *Session) selectWorldComponentLocked(cardID, componentID, componentKind string, events *eventCollector) (Snapshot, error) {
	instance, node, err := s.worldComponentNode(cardID, componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.requireWorldComponentEditable(node.ComponentKind); err != nil {
		return Snapshot{}, err
	}
	s.activeSceneCardID = instance.InstanceID
	s.activeEditingComponentID = node.ID
	events.emit(EventComponentSelected, ComponentPayload{
		CardID:        string(instance.InstanceID),
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Scope:         "world",
	})
	s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" edit controls opened.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) changeWorldComponentLocked(cardID, componentID, componentKind, control string, value json.RawMessage, events *eventCollector) (Snapshot, error) {
	instance, node, err := s.worldComponentNode(cardID, componentID, componentKind)
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
	s.instances[instance.InstanceID] = instance
	s.activeSceneCardID = instance.InstanceID
	s.activeEditingComponentID = node.ID
	events.emit(EventComponentChanged, ComponentPayload{
		CardID:        string(instance.InstanceID),
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Control:       control,
		Scope:         "world",
	})
	_, err = s.runRules(ComponentUpdatedSignal{
		InstanceID: instance.InstanceID, ComponentID: node.ID,
		ComponentKind: node.ComponentKind, Component: cloneValue(*node),
	}, events)
	if err != nil {
		return Snapshot{}, failExecution(err)
	}
	if err := s.advanceEncounter(events); err != nil {
		return Snapshot{}, failExecution(err)
	}
	if !events.hasMessage() {
		s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" updated.", events)
	}
	return s.commandSnapshotLocked()
}

func (s *Session) startEditingLocked(cardID string, events *eventCollector) (Snapshot, error) {
	cardID = strings.TrimSpace(cardID)
	instanceID := CardInstanceID(cardID)
	if s.zoneIndex(ZoneLibrary, instanceID) < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", cardID)
	}
	instance, _ := s.instance(instanceID)
	definition := s.cardDefinitions[instance.DefinitionID]
	_, isComponentCard := instance.State[componentTemplateStateKey]
	if !stateBool(instance.State, "editable") && !isComponentCard {
		return Snapshot{}, fmt.Errorf("%s cannot be edited", definition.Name)
	}
	draft := cloneValue(instance)
	selectedComponentID := ""
	if isComponentCard {
		template, err := componentTemplateFromCard(s.registry, instance)
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
	s.editSession = &editSessionState{
		TargetInstanceID:    instanceID,
		DraftInstance:       draft,
		SelectedComponentID: selectedComponentID,
	}
	s.activeEditingComponentID = ""
	events.emit(EventEditStarted, EditPayload{CardID: cardID})
	s.setMessageLocked("Editing "+definition.Name+".", events)
	return s.commandSnapshotLocked()
}

func (s *Session) installEditComponentLocked(componentCardID string, events *eventCollector) (Snapshot, error) {
	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	componentCardID = strings.TrimSpace(componentCardID)
	componentInstanceID := CardInstanceID(componentCardID)
	if s.zoneIndex(ZoneLibrary, componentInstanceID) < 0 {
		return Snapshot{}, fmt.Errorf("component card %q is not in your library", componentCardID)
	}
	if componentInstanceID == s.editSession.TargetInstanceID {
		return Snapshot{}, fmt.Errorf("a card cannot install itself")
	}
	if instanceIDInSlice(s.editSession.PendingConsumedInstanceIDs, componentInstanceID) {
		component, _ := s.instance(componentInstanceID)
		return Snapshot{}, fmt.Errorf("%s is already pending for this edit", s.cardDefinitions[component.DefinitionID].Name)
	}

	component, _ := s.instance(componentInstanceID)
	template, err := componentTemplateFromCard(s.registry, component)
	if err != nil {
		return Snapshot{}, err
	}
	document, node, err := s.registry.InstallTemplate(s.editSession.DraftInstance.Document, template)
	if err != nil {
		return Snapshot{}, err
	}
	s.editSession.DraftInstance.Document = document
	s.editSession.SelectedComponentID = node.ID

	s.editSession.PendingConsumedInstanceIDs = append(s.editSession.PendingConsumedInstanceIDs, componentInstanceID)
	events.emit(EventEditComponentInstalled, EditComponentInstalledPayload{
		CardID:          string(s.editSession.TargetInstanceID),
		ComponentCardID: componentCardID,
		ComponentID:     node.ID,
		ComponentKind:   node.ComponentKind,
	})
	s.setMessageLocked(s.cardDefinitions[component.DefinitionID].Name+" added to the draft.", events)
	return s.commandSnapshotLocked()
}

func (s *Session) selectEditComponentLocked(componentID, componentKind string, events *eventCollector) (Snapshot, error) {
	node, err := s.editComponentNode(componentID, componentKind)
	if err != nil {
		return Snapshot{}, err
	}
	s.editSession.SelectedComponentID = node.ID
	events.emit(EventComponentSelected, ComponentPayload{
		CardID:        string(s.editSession.TargetInstanceID),
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
		CardID:        string(s.editSession.TargetInstanceID),
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Control:       control,
		Scope:         "edit",
	})
	definition := s.cardDefinitions[s.editSession.DraftInstance.DefinitionID]
	s.setMessageLocked(fmt.Sprintf("%s %s updated.", definition.Name, control), events)
	return s.commandSnapshotLocked()
}

func (s *Session) changeLibraryComponentLocked(cardID, componentID, componentKind, control string, value json.RawMessage, events *eventCollector) (Snapshot, error) {
	cardID = strings.TrimSpace(cardID)
	instanceID := CardInstanceID(cardID)
	if s.zoneIndex(ZoneLibrary, instanceID) < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", cardID)
	}
	instance, _ := s.instance(instanceID)
	definition := s.cardDefinitions[instance.DefinitionID]
	root := &instance.Document.Root
	node, err := componentNode(root, componentID, componentKind, definition.Name)
	if err != nil {
		return Snapshot{}, err
	}
	control = strings.TrimSpace(control)
	if err := applyGameComponentControl(s.registry, node, control, value); err != nil {
		return Snapshot{}, err
	}
	s.instances[instanceID] = instance
	events.emit(EventComponentChanged, ComponentPayload{
		CardID:        cardID,
		ComponentID:   node.ID,
		ComponentKind: node.ComponentKind,
		Control:       control,
		Scope:         "library",
	})
	s.setMessageLocked(componentEditLabel(s.registry, node.ComponentKind)+" updated in "+definition.Name+".", events)
	return s.commandSnapshotLocked()
}

func (s *Session) saveEditLocked(events *eventCollector) (Snapshot, error) {
	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	targetID := s.editSession.TargetInstanceID
	if s.zoneIndex(ZoneLibrary, targetID) < 0 {
		return Snapshot{}, fmt.Errorf("target card %q is not in your library", targetID)
	}

	instance := cloneValue(s.editSession.DraftInstance)
	instance.InstanceID = targetID
	instance.Document.CardID = string(targetID)
	if instance.State == nil {
		instance.State = map[string]any{}
	}
	instance.State["editable"] = true

	installedKinds := map[string]bool{}
	for _, value := range appendStateStringOnce(instance.State["installedComponents"], "") {
		installedKinds[value] = true
	}
	if _, isComponentCard := instance.State[componentTemplateStateKey]; isComponentCard {
		template, err := componentTemplateFromCard(s.registry, instance)
		if err != nil {
			return Snapshot{}, err
		}
		installedKinds[template.ComponentKind] = true
		delete(instance.State, componentTemplateStateKey)
		instance.Tags = removeComponentCardTags(instance.Tags)
	}
	for _, componentID := range s.editSession.PendingConsumedInstanceIDs {
		if component, ok := s.instance(componentID); ok {
			if template, err := componentTemplateFromCard(s.registry, component); err == nil {
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
		instance.State["installedComponents"] = appendStateStringOnce(instance.State["installedComponents"], kind)
		instance.Tags = appendStringOnce(instance.Tags, kind+"-controller")
	}
	if len(installedKinds) > 0 {
		instance.Tags = appendStringOnce(instance.Tags, "controller")
		instance.State["built"] = true
	}

	s.instances[targetID] = instance
	events.emit(EventEditSaved, EditPayload{CardID: string(targetID)})
	for _, componentID := range s.editSession.PendingConsumedInstanceIDs {
		if componentID == targetID {
			continue
		}
		move, err := s.moveCard(componentID, ZoneLibrary, ZoneDiscard)
		if err != nil {
			return Snapshot{}, err
		}
		events.emit(EventCardMoved, CardMovedPayloadFromMove(move))
		events.emit(EventCardConsumed, CardConsumedPayload{CardID: string(componentID)})
	}
	s.setMessageLocked(s.cardDefinitions[instance.DefinitionID].Name+" saved to your library.", events)
	s.editSession = nil
	return s.commandSnapshotLocked()
}

func (s *Session) cancelEditLocked(events *eventCollector) (Snapshot, error) {
	if s.editSession == nil {
		return Snapshot{}, fmt.Errorf("start editing a card first")
	}
	cardID := s.editSession.TargetInstanceID
	cardName := s.cardDefinitions[s.editSession.DraftInstance.DefinitionID].Name
	s.editSession = nil
	events.emit(EventEditCanceled, EditPayload{CardID: string(cardID)})
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
	if err := s.validateZoneState(); err != nil {
		return Snapshot{}, err
	}
	worldDeck, err := s.zoneSnapshots(ZoneScene)
	if err != nil {
		return Snapshot{}, err
	}
	library, err := s.zoneSnapshots(ZoneLibrary)
	if err != nil {
		return Snapshot{}, err
	}
	activeIndex := s.zoneIndex(ZoneScene, s.activeSceneCardID)
	activeCard, err := s.cardSnapshot(s.activeSceneCardID, ZoneScene)
	if err != nil {
		return Snapshot{}, err
	}
	var editSession *EditSession
	if s.editSession != nil {
		draft := s.editSession.DraftInstance
		definition := s.cardDefinitions[draft.DefinitionID]
		editSession = &EditSession{
			TargetCardID:                s.editSession.TargetInstanceID,
			DraftCard:                   CardSnapshot{ID: draft.InstanceID, Name: definition.Name, Kind: definition.Kind, Tags: cloneValue(draft.Tags), Collected: true, State: cloneValue(draft.State), Document: cloneValue(draft.Document)},
			PendingConsumedComponentIDs: cloneValue(s.editSession.PendingConsumedInstanceIDs),
			SelectedComponentID:         s.editSession.SelectedComponentID,
		}
	}
	var encounter *EncounterSnapshot
	if s.encounter != nil {
		participants := make([]EncounterParticipantSnapshot, 0, len(s.encounter.Participants))
		for _, participant := range s.encounter.Participants {
			zone, ok := s.instanceZone(participant.InstanceID)
			if !ok {
				return Snapshot{}, fmt.Errorf("encounter participant %q has invalid zone membership", participant.InstanceID)
			}
			card, err := s.cardSnapshot(participant.InstanceID, zone)
			if err != nil {
				return Snapshot{}, err
			}
			participants = append(participants, EncounterParticipantSnapshot{Role: participant.Role, Card: card})
		}
		encounter = &EncounterSnapshot{
			ID: s.encounter.ID, Phase: s.encounter.Phase, Participants: participants,
			Pressure: s.encounter.Pressure, MaxPressure: s.encounter.MaxPressure, Outcome: s.encounter.Outcome,
		}
	}
	return Snapshot{
		WorldDeck:                worldDeck,
		ActiveWorldCard:          activeCard,
		ActiveWorldCardID:        s.activeSceneCardID,
		ActiveIndex:              activeIndex,
		ActiveEditingComponentID: s.activeEditingComponentID,
		Library:                  library,
		EditSession:              editSession,
		Encounter:                encounter,
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
	definition := s.cardDefinitions[s.editSession.DraftInstance.DefinitionID]
	node, err := componentNode(&s.editSession.DraftInstance.Document.Root, componentID, componentKind, definition.Name)
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

func (s *Session) worldComponentNode(cardID, componentID, componentKind string) (CardInstance, *cardcomponent.Node, error) {
	cardID = strings.TrimSpace(cardID)
	componentID = strings.TrimSpace(componentID)
	componentKind = strings.TrimSpace(componentKind)
	if cardID == "" {
		cardID = string(s.activeSceneCardID)
	}
	instanceID := CardInstanceID(cardID)
	if s.zoneIndex(ZoneScene, instanceID) < 0 {
		return CardInstance{}, nil, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	instance, _ := s.instance(instanceID)
	definition := s.cardDefinitions[instance.DefinitionID]
	node, err := componentNode(&instance.Document.Root, componentID, componentKind, definition.Name)
	if err != nil {
		return CardInstance{}, nil, err
	}
	return instance, node, nil
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

func (s *Session) formRuleFieldNames(target CardInstance, formID string) ([]string, bool) {
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

func cardMatches(card CardInstance, matcher CardMatcherDefinition) bool {
	if strings.TrimSpace(matcher.ID) != "" && card.DefinitionID != CardDefinitionID(matcher.ID) {
		return false
	}
	for _, tag := range matcher.Tags {
		if !hasTag(card, tag) {
			return false
		}
	}
	return true
}

func (s *Session) loadDeck(deckID string, events *eventCollector) (bool, error) {
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
	existingInstances := make(map[CardInstanceID]CardDefinitionID, len(s.instances))
	for instanceID, instance := range s.instances {
		existingInstances[instanceID] = instance.DefinitionID
	}
	if err := ValidateDeckPackDefinition(s.registry, definition, s.cardDefinitions, existingInstances, existingRuleIDs); err != nil {
		return false, err
	}
	materialized, err := materializeDeck(definition)
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
	if s.cardDefinitions == nil {
		s.cardDefinitions = map[CardDefinitionID]CardDefinition{}
	}
	for cardID, card := range materialized.definitions {
		s.cardDefinitions[cardID] = card
	}
	for _, spec := range initialInstanceSpecs(definition) {
		instance := materialized.instances[spec.InstanceID]
		s.instances[spec.InstanceID] = instance
		zone := spec.Zone
		if zone == "" {
			zone = ZoneScene
		}
		s.zones[zone] = append(s.zones[zone], spec.InstanceID)
		events.emit(EventCardInstantiated, CardInstantiatedPayload{
			InstanceID: string(spec.InstanceID), DefinitionID: string(spec.DefinitionID), Zone: string(zone),
		})
	}
	s.rules = append(s.rules, cloneValue(definition.Rules)...)
	if s.loadedDecks == nil {
		s.loadedDecks = map[string]bool{}
	}
	s.loadedDecks[definition.ID] = true
	s.activeSceneCardID = materialized.activeID
	s.activeEditingComponentID = ""
	return true, nil
}

func hasTag(card CardInstance, tag string) bool {
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
