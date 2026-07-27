package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
)

func TestSessionRevisionLifecycle(t *testing.T) {
	session := newCommandTestSession(t, nil)

	first := mustViewResult(t, session)
	second := mustViewResult(t, session)
	if first.Revision != 0 || second.Revision != 0 || len(first.Events) != 0 {
		t.Fatalf("initial views = %#v %#v", first, second)
	}

	cycled := mustExecuteResult(t, session, CycleCardCommand{Direction: "next"})
	if cycled.Revision != 1 {
		t.Fatalf("cycle revision = %d", cycled.Revision)
	}

	if _, err := session.Execute(CollectCardCommand{CardID: "missing"}); err == nil || !IsInvalidCommand(err) {
		t.Fatalf("invalid command error = %v", err)
	}
	if revision := mustViewResult(t, session).Revision; revision != 1 {
		t.Fatalf("revision after invalid command = %d", revision)
	}

	collected := mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	rejected := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "source", TargetCardID: "target"})
	if collected.Revision != 2 || rejected.Revision != 3 {
		t.Fatalf("command revisions = %d, %d", collected.Revision, rejected.Revision)
	}
	if !hasEventType(rejected.Events, EventActionRejected) {
		t.Fatalf("rejected events = %#v", rejected.Events)
	}

	reset := mustExecuteResult(t, session, ResetCommand{})
	if reset.Revision != 4 || !hasEventType(reset.Events, EventSessionReset) {
		t.Fatalf("reset result = %#v", reset)
	}
}

func TestExecuteRollsBackMutationEventsAndRevisionOnFailure(t *testing.T) {
	session := newCommandTestSession(t, nil)
	mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	target := session.instances["target"]
	if target.State == nil {
		target.State = map[string]any{}
	}
	target.State["integer"] = 7
	session.instances["target"] = target
	session.rules = []RuleDefinition{{
		ID:      "forced-failure",
		Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
		Effects: []RuleEffect{
			SetFlagEffect("partiallyMutated", true),
			LoadDeckEffect("missing-deck"),
		},
	}}
	before := mustViewResult(t, session)

	result, err := session.Execute(PlayCardCommand{SourceCardID: "source", TargetCardID: "target"})
	if err == nil || IsInvalidCommand(err) {
		t.Fatalf("execution error = %v", err)
	}
	if result.Revision != 0 || len(result.Events) != 0 || len(result.Snapshot.WorldDeck) != 0 {
		t.Fatalf("failed result = %#v", result)
	}
	after := mustViewResult(t, session)
	if after.Revision != before.Revision || !reflect.DeepEqual(after.Snapshot, before.Snapshot) {
		t.Fatalf("rollback mismatch:\nbefore=%#v\nafter=%#v", before, after)
	}
	if value, ok := session.instances["target"].State["integer"].(int); !ok || value != 7 {
		t.Fatalf("rollback changed concrete state value: %#v", session.instances["target"].State["integer"])
	}
}

func TestCollectAndPlayEventsAreOrderedAndSemantic(t *testing.T) {
	session := newCommandTestSession(t, []RuleEffect{
		SetFlagEffect("opened", true),
		SetCardStateEffect("target", "open", true),
		RemoveCardTagsEffect("target", []string{"closed"}),
		SetDocumentVariantEffect("target", "changed"),
		SetMessageEffect("Opened."),
	})

	collected := mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	assertEventTypes(t, collected.Events, EventCardMoved, EventCardCollected, EventMessage)

	played := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "source", TargetCardID: "target"})
	assertEventTypes(t, played.Events,
		EventCardPlayed,
		EventFlagChanged,
		EventCardStateChanged,
		EventCardTagsRemoved,
		EventCardVariantChanged,
		EventMessage,
		EventCardMoved,
		EventCardConsumed,
		EventRuleResolved,
	)
	if played.Snapshot.Message != "Opened." {
		t.Fatalf("snapshot message = %q", played.Snapshot.Message)
	}
	lastMessage := ""
	for index, event := range played.Events {
		if event.Sequence != index {
			t.Fatalf("event %d sequence = %d", index, event.Sequence)
		}
		if event.Type == EventMessage {
			lastMessage = event.Message
		}
	}
	if lastMessage != played.Snapshot.Message {
		t.Fatalf("last message event = %q, snapshot message = %q", lastMessage, played.Snapshot.Message)
	}
}

func TestUnmatchedAndMatchedFailedPlayConsumptionEvents(t *testing.T) {
	required := 7
	session := newCommandTestSessionWithCondition(t, required)
	mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})

	unmatched := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "source", TargetCardID: "other"})
	if hasEventType(unmatched.Events, EventCardConsumed) {
		t.Fatalf("unmatched events = %#v", unmatched.Events)
	}

	failed := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "source", TargetCardID: "target"})
	if !hasEventType(failed.Events, EventCardConsumed) || !hasEventType(failed.Events, EventActionRejected) {
		t.Fatalf("failed events = %#v", failed.Events)
	}
}

func TestGeneratorTuningEmitsDeckLoaded(t *testing.T) {
	registry := catalog.MustNew()
	definition, err := LoadEmbeddedDeck(registry, GeneratorDeckDefinition)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	mustExecuteResult(t, session, CollectCardCommand{CardID: "slider-component"})
	started := mustExecuteResult(t, session, StartEditingCommand{CardID: "slider-component"})
	sliderID := started.Snapshot.EditSession.SelectedComponentID
	mustExecuteResult(t, session, SaveEditCommand{})
	mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "slider-component", TargetCardID: "generator-panel"})

	tuned := mustExecuteResult(t, session, ChangeWorldComponentCommand{
		CardID: "generator-panel", ComponentID: sliderID, ComponentKind: card.KindSlider,
		Control: "value", Value: json.RawMessage(`73`),
	})
	if !hasEventType(tuned.Events, EventDeckLoaded) {
		t.Fatalf("tuning events = %#v", tuned.Events)
	}
}

func TestSubmitFormEventsDoNotExposeFieldValues(t *testing.T) {
	registry := catalog.MustNew()
	document := simpleDocument("terminal")
	document.Root.Children = []card.Node{
		{ID: "password", ComponentKind: card.KindTextInput, Config: json.RawMessage(`{"form_id":"login","name":"password"}`)},
		{ID: "submit", ComponentKind: card.KindButton, Config: json.RawMessage(`{"form_id":"login"}`)},
	}
	definition := DeckDefinition{
		ID: "form-command-test", Name: "Form Command Test", InitialActiveCardID: "terminal",
		Cards: []CardDefinition{{
			ID: "terminal", Name: "Terminal", Kind: KindWorld, InitialDocument: "default",
			Documents: map[string]card.Document{"default": document},
		}},
		Rules: []RuleDefinition{{
			ID:      "accept-login",
			Trigger: FormSubmittedTrigger(CardMatcherDefinition{ID: "terminal"}, "login"),
			Conditions: []RuleCondition{
				FormFieldEqualsCondition("password", "nightjar", false, false),
			},
			Effects: []RuleEffect{SetMessageEffect("Accepted.")},
		}},
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	result := mustExecuteResult(t, session, SubmitFormCommand{
		CardID: "terminal", FormID: "login", Fields: map[string]string{"password": "nightjar"},
	})
	assertEventTypes(t, result.Events, EventFormSubmitted, EventMessage, EventRuleResolved)
	raw, err := json.Marshal(result.Events)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || containsJSONText(raw, "nightjar") {
		t.Fatalf("events leaked submitted value: %s", raw)
	}
}

func newCommandTestSession(t *testing.T, effects []RuleEffect) *Session {
	t.Helper()
	registry := catalog.MustNew()
	targetDefault := simpleDocument("target")
	targetChanged := simpleDocument("target")
	definition := DeckDefinition{
		ID: "command-test", Name: "Command Test", InitialActiveCardID: "target", InitialMessage: "Ready.",
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, Tags: []string{"closed"}, State: map[string]any{}, InitialDocument: "default", Documents: map[string]card.Document{"default": targetDefault, "changed": targetChanged}},
			{ID: "other", Name: "Other", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("other")}},
			{ID: "source", Name: "Source", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": sliderDocument("source", 3)}},
		},
	}
	if effects != nil {
		effects = append(effects, MoveCardEffect(SignaledInstance(RuleCardSource), ZoneDiscard))
		definition.Rules = []RuleDefinition{{
			ID:      "command-rule",
			Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
			Effects: effects,
		}}
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func newCommandTestSessionWithCondition(t *testing.T, required int) *Session {
	t.Helper()
	session := newCommandTestSession(t, nil)
	session.rules = []RuleDefinition{{
		ID:      "conditional",
		Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
		Conditions: []RuleCondition{
			ComponentPropertyEqualsCondition(RuleCardSource, "", card.KindSlider, "", "value", NumberRuleValue(float64(required))),
		},
		Effects: []RuleEffect{
			SetMessageEffect("Matched."),
			MoveCardEffect(SignaledInstance(RuleCardSource), ZoneDiscard),
		},
		ElseEffects: []RuleEffect{
			SetMessageEffect("Wrong value."),
			MoveCardEffect(SignaledInstance(RuleCardSource), ZoneDiscard),
		},
	}}
	return session
}

func mustViewResult(t *testing.T, session *Session) Result {
	t.Helper()
	result, err := session.View()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mustExecuteResult(t *testing.T, session *Session, command Command) Result {
	t.Helper()
	result, err := session.Execute(command)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func hasEventType(events []Event, eventType EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func assertEventTypes(t *testing.T, events []Event, expected ...EventType) {
	t.Helper()
	actual := make([]EventType, len(events))
	for index, event := range events {
		actual[index] = event.Type
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("event types = %v, want %v", actual, expected)
	}
}

func containsJSONText(raw []byte, value string) bool {
	for index := 0; index+len(value) <= len(raw); index++ {
		if string(raw[index:index+len(value)]) == value {
			return true
		}
	}
	return false
}
