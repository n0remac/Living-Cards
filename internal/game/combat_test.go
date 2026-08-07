package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestAttackCreatureEffectDecodingAndValidation(t *testing.T) {
	var effect RuleEffect
	if err := json.Unmarshal([]byte(`{"kind":"attackCreature"}`), &effect); err != nil {
		t.Fatal(err)
	}
	if effect.Kind() != EffectAttackCreature {
		t.Fatalf("kind = %q", effect.Kind())
	}
	if err := json.Unmarshal([]byte(`{"kind":"attackCreature","power":3}`), &effect); err == nil {
		t.Fatal("attackCreature accepted an unknown field")
	}

	registry := catalog.MustNew()
	definition := combatTestDeck(6, []int{3})
	definition.Rules[0].Trigger = FormSubmittedTrigger(CardMatcherDefinition{ID: "target"}, "combat-form")
	if err := ValidateDeckDefinition(registry, definition); err == nil || !strings.Contains(err.Error(), "only valid for cardPlayed") {
		t.Fatalf("attack validation error = %v", err)
	}
	definition = combatTestDeck(6, []int{3})
	definition.Rules[0].Trigger = FormSubmittedTrigger(CardMatcherDefinition{ID: "target"}, "combat-form")
	definition.Rules[0].Effects = []RuleEffect{SetMessageEffect("unused")}
	if err := ValidateDeckDefinition(registry, definition); err == nil || !strings.Contains(err.Error(), "retainSource is only valid") {
		t.Fatalf("retain validation error = %v", err)
	}
}

func TestAttackCreatureRequiresComponentsSumsPowerAndClampsHealth(t *testing.T) {
	t.Run("missing source creature", func(t *testing.T) {
		session := newCombatTestSession(t, 6, []int{3})
		session.library[0].Document.Root.Children = removeNodesByKind(session.library[0].Document.Root.Children, card.KindCreature)
		if _, _, err := session.attackCreature(CardPlayedSignal{SourceCardID: "source", TargetCardID: "target"}); err == nil || !strings.Contains(err.Error(), "no creature") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("missing attack", func(t *testing.T) {
		session := newCombatTestSession(t, 6, []int{3})
		session.library[0].Document.Root.Children = removeNodesByKind(session.library[0].Document.Root.Children, card.KindAttack)
		if _, _, err := session.attackCreature(CardPlayedSignal{SourceCardID: "source", TargetCardID: "target"}); err == nil || !strings.Contains(err.Error(), "no attack") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("missing target creature", func(t *testing.T) {
		session := newCombatTestSession(t, 6, []int{3})
		session.worldDeck[0].Document.Root.Children = removeNodesByKind(session.worldDeck[0].Document.Root.Children, card.KindCreature)
		if _, _, err := session.attackCreature(CardPlayedSignal{SourceCardID: "source", TargetCardID: "target"}); err == nil || !strings.Contains(err.Error(), "no creature") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("multiple attacks", func(t *testing.T) {
		session := newCombatTestSession(t, 6, []int{2, 3})
		payload, signal, err := session.attackCreature(CardPlayedSignal{SourceCardID: "source", TargetCardID: "target"})
		if err != nil {
			t.Fatal(err)
		}
		if payload.Attack != 5 || payload.PreviousHealth != 6 || payload.Health != 1 || signal.ComponentKind != card.KindCreature {
			t.Fatalf("payload = %#v signal = %#v", payload, signal)
		}
		if got := creatureHealth(t, session, "target"); got != 1 {
			t.Fatalf("health = %d", got)
		}
	})
	t.Run("clamps at zero", func(t *testing.T) {
		session := newCombatTestSession(t, 6, []int{9})
		payload, _, err := session.attackCreature(CardPlayedSignal{SourceCardID: "source", TargetCardID: "target"})
		if err != nil || payload.Health != 0 || creatureHealth(t, session, "target") != 0 {
			t.Fatalf("payload = %#v health = %d error = %v", payload, creatureHealth(t, session, "target"), err)
		}
	})
}

func TestFailedCombatCommandRollsBackHealth(t *testing.T) {
	session := newCombatTestSession(t, 6, []int{3})
	session.rules[0].Effects = append(session.rules[0].Effects, SetDocumentVariantEffect("target", "missing"))
	if _, err := session.Execute(PlayCardCommand{SourceCardID: "source", TargetCardID: "target"}); err == nil {
		t.Fatal("invalid follow-on effect succeeded")
	}
	if got := creatureHealth(t, session, "target"); got != 6 {
		t.Fatalf("rolled-back health = %d", got)
	}
	if session.libraryCardIndex("source") < 0 {
		t.Fatal("failed command removed source")
	}
}

func TestArchiveGuardianFullPathRetainsCreatureAndOrdersEvents(t *testing.T) {
	registry := catalog.MustNew()
	definition, err := LoadEmbeddedDeck(registry, ArchiveGuardianDeckDefinition)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	mustExecuteResult(t, session, CollectCardCommand{CardID: "moss-stalker"})

	failed := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "moss-stalker", TargetCardID: "archive-guardian"})
	if !containsCard(failed.Snapshot.Library, "moss-stalker") || hasEventType(failed.Events, EventCardConsumed) || failed.Snapshot.Message != "The creature needs an attack component before it can challenge the guardian." {
		t.Fatalf("failed attack = %#v", failed)
	}

	mustExecuteResult(t, session, CollectCardCommand{CardID: "crystal-claws"})
	mustExecuteResult(t, session, StartEditingCommand{CardID: "moss-stalker"})
	mustExecuteResult(t, session, InstallEditComponentCommand{ComponentCardID: "crystal-claws"})
	saved := mustExecuteResult(t, session, SaveEditCommand{})
	if containsCard(saved.Snapshot.Library, "crystal-claws") || findRuleComponent(saved.Snapshot.Library[0].Document, card.KindAttack, "") == nil {
		t.Fatalf("equipped library = %#v", saved.Snapshot.Library)
	}

	first := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "moss-stalker", TargetCardID: "archive-guardian"})
	if creatureHealth(t, session, "archive-guardian") != 3 || !containsCard(first.Snapshot.Library, "moss-stalker") || hasEventType(first.Events, EventCardConsumed) {
		t.Fatalf("first attack = %#v", first)
	}
	assertCombatEventOrder(t, first.Events, 6, 3)

	second := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "moss-stalker", TargetCardID: "archive-guardian"})
	if !second.Snapshot.SolvedFlags["guardianDefeated"] || !containsCard(second.Snapshot.Library, "moss-stalker") || hasEventType(second.Events, EventCardConsumed) {
		t.Fatalf("second attack = %#v", second)
	}
	guardian := findCard(second.Snapshot.WorldDeck, "archive-guardian")
	if guardian == nil || hasTag(*guardian, "hostile") || !stateBool(guardian.State, "defeated") || creatureHealth(t, session, "archive-guardian") != 0 {
		t.Fatalf("guardian = %#v", guardian)
	}
	assertCombatEventOrder(t, second.Events, 3, 0)
	if second.Snapshot.Message != "The Archive Guardian shatters. The sealed stacks are yours to explore." {
		t.Fatalf("message = %q", second.Snapshot.Message)
	}
}

func TestLegacyCardPlayStillConsumesSource(t *testing.T) {
	definition := combatTestDeck(6, []int{3})
	definition.Rules[0].RetainSource = false
	definition.Rules[0].Effects = []RuleEffect{SetMessageEffect("legacy resolved")}
	session, err := NewSessionFromDeck(catalog.MustNew(), definition)
	if err != nil {
		t.Fatal(err)
	}
	mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	result := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "source", TargetCardID: "target"})
	if containsCard(result.Snapshot.Library, "source") || !hasEventType(result.Events, EventCardConsumed) {
		t.Fatalf("legacy result = %#v", result)
	}
}

func combatTestDeck(targetHealth int, powers []int) DeckDefinition {
	return DeckDefinition{
		ID: "combat-test", Name: "Combat Test", InitialActiveCardID: "target",
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, Tags: []string{"hostile"}, InitialDocument: "default", Documents: map[string]card.Document{"default": combatDocument("target", &targetHealth, nil)}},
			{ID: "source", Name: "Source", Kind: KindItem, Tags: []string{"friendly-creature"}, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": combatDocument("source", intPointer(4), powers)}},
		},
		Rules: []RuleDefinition{{
			ID: "combat-test-attack", Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
			RetainSource: true, Effects: []RuleEffect{AttackCreatureEffect()},
		}},
	}
}

func combatDocument(id string, health *int, powers []int) card.Document {
	document := card.Document{CardID: id, Name: id, Root: card.Node{ID: id + "-root", ComponentKind: card.Kind}}
	if health != nil {
		document.Root.Children = append(document.Root.Children, card.Node{ID: id + "-creature", ComponentKind: card.KindCreature, Config: json.RawMessage(fmt.Sprintf(`{"health":%d,"max_health":%d}`, *health, maxInt(*health, 1)))})
	}
	for index, power := range powers {
		document.Root.Children = append(document.Root.Children, card.Node{ID: fmt.Sprintf("%s-attack-%d", id, index), ComponentKind: card.KindAttack, Config: json.RawMessage(fmt.Sprintf(`{"label":"Attack %d","power":%d}`, index+1, power))})
	}
	return document
}

func newCombatTestSession(t *testing.T, health int, powers []int) *Session {
	t.Helper()
	session, err := NewSessionFromDeck(catalog.MustNew(), combatTestDeck(health, powers))
	if err != nil {
		t.Fatal(err)
	}
	mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	return session
}

func creatureHealth(t *testing.T, session *Session, cardID string) int {
	t.Helper()
	index := session.worldCardIndex(cardID)
	if index < 0 {
		t.Fatalf("card %q is not in world", cardID)
	}
	node := findRuleComponent(session.worldDeck[index].Document, card.KindCreature, "")
	if node == nil {
		t.Fatalf("card %q has no creature", cardID)
	}
	definition, _ := session.registry.Lookup(card.KindCreature)
	value, present, issues := definition.ReadProperty(node.Config, "health")
	if len(issues) != 0 || !present || value.Kind != schema.PropertyNumber {
		t.Fatalf("health property = %#v, %v, %#v", value, present, issues)
	}
	return int(value.Number)
}

func assertCombatEventOrder(t *testing.T, events []Event, previousHealth, health int) {
	t.Helper()
	if len(events) < 3 || events[0].Type != EventCardPlayed || events[1].Type != EventCreatureAttacked || events[2].Type != EventRuleResolved {
		t.Fatalf("event order = %#v", events)
	}
	payload, ok := events[1].Payload.(CreatureAttackedPayload)
	if !ok || payload.Attack != 3 || payload.PreviousHealth != previousHealth || payload.Health != health {
		t.Fatalf("attack payload = %#v", events[1].Payload)
	}
	raw, err := json.Marshal(events[1])
	if err != nil || !strings.Contains(string(raw), `"type":"creatureAttacked"`) || !strings.Contains(string(raw), `"previousHealth":`) {
		t.Fatalf("serialized event = %s error = %v", raw, err)
	}
}

func removeNodesByKind(nodes []card.Node, kind string) []card.Node {
	out := nodes[:0]
	for _, node := range nodes {
		if node.ComponentKind != kind {
			out = append(out, node)
		}
	}
	return out
}

func findCard(cards []Card, id string) *Card {
	for index := range cards {
		if cards[index].ID == id {
			return &cards[index]
		}
	}
	return nil
}

func intPointer(value int) *int { return &value }
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
