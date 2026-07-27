package game

type EventType string

const (
	EventSessionReset           EventType = "sessionReset"
	EventCardCycled             EventType = "cardCycled"
	EventCardCollected          EventType = "cardCollected"
	EventCardInstantiated       EventType = "cardInstantiated"
	EventCardMoved              EventType = "cardMoved"
	EventCardPlayed             EventType = "cardPlayed"
	EventCardConsumed           EventType = "cardConsumed"
	EventFormSubmitted          EventType = "formSubmitted"
	EventComponentSelected      EventType = "componentSelected"
	EventComponentChanged       EventType = "componentChanged"
	EventEditStarted            EventType = "editStarted"
	EventEditComponentInstalled EventType = "editComponentInstalled"
	EventEditSaved              EventType = "editSaved"
	EventEditCanceled           EventType = "editCanceled"
	EventFlagChanged            EventType = "flagChanged"
	EventCardStateChanged       EventType = "cardStateChanged"
	EventCardTagsRemoved        EventType = "cardTagsRemoved"
	EventCardVariantChanged     EventType = "cardVariantChanged"
	EventComponentMounted       EventType = "componentMounted"
	EventDeckLoaded             EventType = "deckLoaded"
	EventRuleResolved           EventType = "ruleResolved"
	EventActionRejected         EventType = "actionRejected"
	EventEncounterStarted       EventType = "encounterStarted"
	EventEncounterPhaseChanged  EventType = "encounterPhaseChanged"
	EventEncounterResolved      EventType = "encounterResolved"
	EventActorResourceChanged   EventType = "actorResourceChanged"
	EventMessage                EventType = "message"
)

type Event struct {
	Sequence int       `json:"sequence"`
	Type     EventType `json:"type"`
	Message  string    `json:"message,omitempty"`
	Payload  any       `json:"payload,omitempty"`
}

type SessionResetPayload struct{}

type CardCycledPayload struct {
	Direction      string `json:"direction"`
	PreviousCardID string `json:"previousCardId"`
	ActiveCardID   string `json:"activeCardId"`
}

type CardCollectedPayload struct {
	CardID             string `json:"cardId"`
	PreviousWorldIndex int    `json:"previousWorldIndex"`
	ActiveCardID       string `json:"activeCardId"`
}

type CardInstantiatedPayload struct {
	InstanceID   string `json:"instanceId"`
	DefinitionID string `json:"definitionId"`
	Zone         string `json:"zone"`
}

type CardMovedPayload struct {
	InstanceID string `json:"instanceId"`
	From       string `json:"from"`
	To         string `json:"to"`
	FromIndex  int    `json:"fromIndex"`
	ToIndex    int    `json:"toIndex"`
}

func CardMovedPayloadFromMove(move CardMove) CardMovedPayload {
	return CardMovedPayload{
		InstanceID: string(move.InstanceID),
		From:       string(move.From),
		To:         string(move.To),
		FromIndex:  move.FromIndex,
		ToIndex:    move.ToIndex,
	}
}

type CardPlayedPayload struct {
	SourceCardID string `json:"sourceCardId"`
	TargetCardID string `json:"targetCardId"`
	Outcome      string `json:"outcome"`
}

type CardConsumedPayload struct {
	CardID string `json:"cardId"`
}

type FormSubmittedPayload struct {
	CardID string `json:"cardId"`
	FormID string `json:"formId"`
}

type ComponentPayload struct {
	CardID        string `json:"cardId"`
	ComponentID   string `json:"componentId"`
	ComponentKind string `json:"componentKind"`
	Control       string `json:"control,omitempty"`
	Scope         string `json:"scope"`
}

type EditPayload struct {
	CardID string `json:"cardId"`
}

type EditComponentInstalledPayload struct {
	CardID          string `json:"cardId"`
	ComponentCardID string `json:"componentCardId"`
	ComponentID     string `json:"componentId"`
	ComponentKind   string `json:"componentKind"`
}

type FlagChangedPayload struct {
	Flag  string `json:"flag"`
	Value bool   `json:"value"`
}

type CardStateChangedPayload struct {
	CardID string `json:"cardId"`
	Key    string `json:"key"`
	Value  any    `json:"value"`
}

type CardTagsRemovedPayload struct {
	CardID string   `json:"cardId"`
	Tags   []string `json:"tags"`
}

type CardVariantChangedPayload struct {
	CardID  string `json:"cardId"`
	Variant string `json:"variant"`
}

type ComponentMountedPayload struct {
	SourceCardID  string `json:"sourceCardId"`
	TargetCardID  string `json:"targetCardId"`
	ComponentID   string `json:"componentId"`
	ComponentKind string `json:"componentKind"`
}

type DeckLoadedPayload struct {
	DeckID string `json:"deckId"`
}

type RuleResolvedPayload struct {
	RuleID      string `json:"ruleId"`
	TriggerKind string `json:"triggerKind"`
	Outcome     string `json:"outcome"`
}

type ActionRejectedPayload struct {
	Action  string `json:"action"`
	Outcome string `json:"outcome"`
}

type EncounterStartedPayload struct {
	EncounterID string `json:"encounterId"`
	Phase       string `json:"phase"`
}

type EncounterPhaseChangedPayload struct {
	EncounterID   string `json:"encounterId"`
	PreviousPhase string `json:"previousPhase"`
	Phase         string `json:"phase"`
}

type EncounterResolvedPayload struct {
	EncounterID string `json:"encounterId"`
	Outcome     string `json:"outcome"`
}

type ActorResourceChangedPayload struct {
	InstanceID string `json:"instanceId"`
	Resource   string `json:"resource"`
	Previous   int    `json:"previous"`
	Current    int    `json:"current"`
}

type eventCollector struct {
	events []Event
}

func (c *eventCollector) emit(eventType EventType, payload any) {
	c.events = append(c.events, Event{Sequence: len(c.events), Type: eventType, Payload: payload})
}

func (c *eventCollector) message(message string) {
	c.events = append(c.events, Event{Sequence: len(c.events), Type: EventMessage, Message: message})
}

func (c *eventCollector) hasMessage() bool {
	for _, event := range c.events {
		if event.Type == EventMessage {
			return true
		}
	}
	return false
}
