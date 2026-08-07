package game

type EventType string

const (
	EventSessionReset           EventType = "sessionReset"
	EventCardCycled             EventType = "cardCycled"
	EventCardCollected          EventType = "cardCollected"
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
	EventCreatureAttacked       EventType = "creatureAttacked"
	EventDeckLoaded             EventType = "deckLoaded"
	EventRuleResolved           EventType = "ruleResolved"
	EventActionRejected         EventType = "actionRejected"
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

type CreatureAttackedPayload struct {
	SourceCardID   string `json:"sourceCardId"`
	TargetCardID   string `json:"targetCardId"`
	Attack         int    `json:"attack"`
	PreviousHealth int    `json:"previousHealth"`
	Health         int    `json:"health"`
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
