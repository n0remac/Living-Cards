package game

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestTaggedRuleDefinitionsDecodeStrictly(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		out  any
	}{
		{
			name: "unknown trigger",
			raw:  `{"kind":"timerElapsed"}`,
			out:  &RuleTrigger{},
		},
		{
			name: "trigger field from another kind",
			raw:  `{"kind":"cardPlayed","source":{"id":"source"},"target":{"id":"target"},"formId":"login"}`,
			out:  &RuleTrigger{},
		},
		{
			name: "unknown condition field",
			raw:  `{"kind":"flagEquals","flag":"ready","value":false,"legacy":true}`,
			out:  &RuleCondition{},
		},
		{
			name: "missing condition value",
			raw:  `{"kind":"flagEquals","flag":"ready"}`,
			out:  &RuleCondition{},
		},
		{
			name: "irrelevant effect field",
			raw:  `{"kind":"setMessage","message":"Ready.","flag":"ready"}`,
			out:  &RuleEffect{},
		},
		{
			name: "missing effect value",
			raw:  `{"kind":"setFlag","flag":"ready"}`,
			out:  &RuleEffect{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(test.raw), test.out); err == nil {
				t.Fatalf("json.Unmarshal(%s) succeeded", test.raw)
			}
		})
	}

	original := RuleDefinition{
		ID:      "strict-round-trip",
		Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
		Conditions: []RuleCondition{
			FlagEqualsCondition("ready", false),
			ComponentPropertyEqualsCondition(RuleCardSource, "", card.KindSlider, "", "value", NumberRuleValue(73)),
		},
		Effects:     []RuleEffect{SetFlagEffect("ready", true), SetMessageEffect("Ready.")},
		ElseEffects: []RuleEffect{SetMessageEffect("Not ready.")},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RuleDefinition
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("round trip mismatch:\noriginal=%#v\ndecoded=%#v", original, decoded)
	}
}

func TestRuleValidationUsesPropertyKindsAndTriggerContext(t *testing.T) {
	registry := catalog.MustNew()
	slider, ok := registry.Lookup(card.KindSlider)
	if !ok {
		t.Fatal("slider definition is not registered")
	}
	if kind, ok := slider.PropertyKind("value"); !ok || kind != schema.PropertyNumber {
		t.Fatalf("slider value property = %q, %v", kind, ok)
	}

	base := ruleTestDeck(nil)
	tests := []struct {
		name string
		rule RuleDefinition
		want string
	}{
		{
			name: "property type mismatch",
			rule: RuleDefinition{
				ID:      "wrong-property-type",
				Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
				Conditions: []RuleCondition{
					ComponentPropertyEqualsCondition(RuleCardSource, "", card.KindSlider, "", "value", StringRuleValue("73")),
				},
				Effects: []RuleEffect{SetMessageEffect("matched")},
			},
			want: "is number, not string",
		},
		{
			name: "source outside card play",
			rule: RuleDefinition{
				ID:      "invalid-source",
				Trigger: FormSubmittedTrigger(CardMatcherDefinition{ID: "target"}, "login"),
				Conditions: []RuleCondition{
					ComponentPresentCondition(RuleCardSource, card.KindSlider, ""),
				},
				Effects: []RuleEffect{SetMessageEffect("matched")},
			},
			want: "source reference is only valid",
		},
		{
			name: "trigger component outside update",
			rule: RuleDefinition{
				ID:      "invalid-trigger-component",
				Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
				Conditions: []RuleCondition{
					ComponentPropertyEqualsCondition("", RuleComponentTrigger, card.KindSlider, "", "value", NumberRuleValue(73)),
				},
				Effects: []RuleEffect{SetMessageEffect("matched")},
			},
			want: "trigger component is only valid",
		},
		{
			name: "unknown trigger component",
			rule: RuleDefinition{
				ID:      "unknown-trigger-component",
				Trigger: ComponentUpdatedTrigger(CardMatcherDefinition{ID: "target"}, "oscillator", ""),
				Effects: []RuleEffect{SetMessageEffect("matched")},
			},
			want: "is not registered",
		},
		{
			name: "form field outside form",
			rule: RuleDefinition{
				ID:      "invalid-form-field",
				Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
				Conditions: []RuleCondition{
					FormFieldEqualsCondition("password", "secret", false, true),
				},
				Effects: []RuleEffect{SetMessageEffect("matched")},
			},
			want: "only valid for formSubmitted",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := cloneValue(base)
			definition.Rules = []RuleDefinition{test.rule}
			err := ValidateDeckDefinition(registry, definition)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateDeckDefinition() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRuleIDsRemainUniqueAcrossLoadedPacks(t *testing.T) {
	registry := catalog.MustNew()
	definition := ruleTestDeck([]RuleDefinition{{
		ID:      "shared-rule",
		Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
		Effects: []RuleEffect{SetMessageEffect("matched")},
	}})
	err := ValidateDeckPackDefinition(registry, definition, nil, map[string]bool{"shared-rule": true})
	if err == nil || !strings.Contains(err.Error(), `duplicate rule id "shared-rule"`) {
		t.Fatalf("ValidateDeckPackDefinition() error = %v", err)
	}
}

func TestRuleRuntimeSelectsFirstSuccessAndFirstFallback(t *testing.T) {
	t.Run("first success", func(t *testing.T) {
		session := newRuleRuntimeSession(t, []RuleDefinition{
			{
				ID:      "fails-first",
				Trigger: testCardPlayedTrigger(),
				Conditions: []RuleCondition{
					FlagEqualsCondition("missing", true),
				},
				Effects: []RuleEffect{SetMessageEffect("wrong")},
			},
			{
				ID:      "passes-second",
				Trigger: testCardPlayedTrigger(),
				Conditions: []RuleCondition{
					FlagEqualsCondition("missing", false),
				},
				Effects: []RuleEffect{SetMessageEffect("second")},
			},
			{
				ID:      "also-passes",
				Trigger: testCardPlayedTrigger(),
				Effects: []RuleEffect{SetMessageEffect("third")},
			},
		})
		result := collectAndPlayRuleSource(t, session)
		if result.Snapshot.Message != "second" {
			t.Fatalf("message = %q", result.Snapshot.Message)
		}
		assertResolvedRule(t, result.Events, "passes-second", RuleOutcomeSuccess)
	})

	t.Run("first rule with fallback", func(t *testing.T) {
		session := newRuleRuntimeSession(t, []RuleDefinition{
			{
				ID:         "fails-without-fallback",
				Trigger:    testCardPlayedTrigger(),
				Conditions: []RuleCondition{FlagEqualsCondition("missing", true)},
				Effects:    []RuleEffect{SetMessageEffect("wrong")},
			},
			{
				ID:          "first-fallback",
				Trigger:     testCardPlayedTrigger(),
				Conditions:  []RuleCondition{FlagEqualsCondition("missing", true)},
				Effects:     []RuleEffect{SetMessageEffect("wrong")},
				ElseEffects: []RuleEffect{SetMessageEffect("fallback")},
			},
			{
				ID:          "later-fallback",
				Trigger:     testCardPlayedTrigger(),
				Conditions:  []RuleCondition{FlagEqualsCondition("missing", true)},
				Effects:     []RuleEffect{SetMessageEffect("wrong")},
				ElseEffects: []RuleEffect{SetMessageEffect("later")},
			},
		})
		result := collectAndPlayRuleSource(t, session)
		if result.Snapshot.Message != "fallback" {
			t.Fatalf("message = %q", result.Snapshot.Message)
		}
		assertResolvedRule(t, result.Events, "first-fallback", RuleOutcomeConditionsFailed)
	})

	t.Run("failure without fallback uses first trigger", func(t *testing.T) {
		session := newRuleRuntimeSession(t, []RuleDefinition{
			{
				ID:         "first-failure",
				Trigger:    testCardPlayedTrigger(),
				Conditions: []RuleCondition{FlagEqualsCondition("missing", true)},
				Effects:    []RuleEffect{SetMessageEffect("wrong")},
			},
			{
				ID:         "second-failure",
				Trigger:    testCardPlayedTrigger(),
				Conditions: []RuleCondition{FlagEqualsCondition("missing", true)},
				Effects:    []RuleEffect{SetMessageEffect("wrong")},
			},
		})
		result := collectAndPlayRuleSource(t, session)
		assertResolvedRule(t, result.Events, "first-failure", RuleOutcomeConditionsFailed)
		if !hasEventType(result.Events, EventCardConsumed) {
			t.Fatal("relevant failed play did not consume its source")
		}
	})
}

func TestLoadedRulesAppendInStableDeclarationOrder(t *testing.T) {
	registry := catalog.MustNew()
	definition, err := LoadEmbeddedDeck(registry, GeneratorDeckDefinition)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	before := ruleIDs(session.rules)
	if loaded, err := session.loadDeck(ArchiveTerminalDefinition); err != nil || !loaded {
		t.Fatalf("loadDeck() = %v, %v", loaded, err)
	}
	after := ruleIDs(session.rules)
	want := append(append([]string(nil), before...), "controller-mounts-archive-login", "nightjar-unlocks-archive")
	if !reflect.DeepEqual(after, want) {
		t.Fatalf("rule order = %v, want %v", after, want)
	}
}

func TestGeneratorMountAndEditPathsUseSameActivationRule(t *testing.T) {
	for _, test := range []struct {
		name            string
		valueBeforePlay int
		editAfterPlay   bool
	}{
		{name: "mounted at target value", valueBeforePlay: 73},
		{name: "edited after mount", valueBeforePlay: 50, editAfterPlay: true},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			if test.valueBeforePlay != 50 {
				mustExecuteResult(t, session, ChangeEditComponentCommand{
					ComponentID: sliderID, Control: "value",
					Value: json.RawMessage(`73`),
				})
			}
			mustExecuteResult(t, session, SaveEditCommand{})
			played := mustExecuteResult(t, session, PlayCardCommand{
				SourceCardID: "slider-component", TargetCardID: "generator-panel",
			})

			var activated Result
			if test.editAfterPlay {
				assertResolvedRule(t, played.Events, "regulator-activates-generator", RuleOutcomeConditionsFailed)
				activated = mustExecuteResult(t, session, ChangeWorldComponentCommand{
					CardID: "generator-panel", ComponentID: sliderID, ComponentKind: card.KindSlider,
					Control: "value", Value: json.RawMessage(`73`),
				})
			} else {
				activated = played
			}
			assertResolvedRule(t, activated.Events, "regulator-activates-generator", RuleOutcomeSuccess)
			if countResolvedRule(activated.Events, "regulator-activates-generator") != 1 {
				t.Fatalf("activation resolved more than once: %#v", activated.Events)
			}
			generator, ok := cardByID(activated.Snapshot.WorldDeck, "generator-panel")
			if !ok {
				t.Fatal("generator is not in world deck")
			}
			slider := findNodeByID(generator.Document.Root, sliderID)
			if slider == nil || slider.ComponentKind != card.KindSlider {
				t.Fatalf("triggering slider %q was not preserved", sliderID)
			}
			sliderDefinition, _ := registry.Lookup(card.KindSlider)
			value, present, issues := sliderDefinition.ReadProperty(slider.Config, "value")
			if len(issues) > 0 || !present || value.Number != 73 {
				t.Fatalf("preserved slider value = %#v, present=%v, issues=%v", value, present, issues)
			}
			if !activated.Snapshot.SolvedFlags["generatorPowered"] ||
				!containsCard(activated.Snapshot.WorldDeck, "archive-terminal") {
				t.Fatalf("generator did not activate: %#v", activated.Snapshot)
			}
		})
	}
}

func TestRuleSignalLimitRollsBackCommand(t *testing.T) {
	registry := catalog.MustNew()
	borderDefinition, ok := registry.Lookup(card.KindBorder)
	if !ok {
		t.Fatal("border definition is not registered")
	}
	document := simpleDocument("target")
	document.Root.Children = []card.Node{{
		ID: "target-border", ComponentKind: card.KindBorder,
		Config: append(json.RawMessage(nil), borderDefinition.Schema().Default...),
	}}
	emptyDocument := simpleDocument("target")
	definition := DeckDefinition{
		ID: "loop-test", Name: "Loop Test", InitialActiveCardID: "target",
		Cards: []CardDefinition{{
			ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "with-border",
			Documents: map[string]card.Document{"with-border": document, "empty": emptyDocument},
		}},
		Rules: []RuleDefinition{{
			ID:      "copy-border-forever",
			Trigger: ComponentUpdatedTrigger(CardMatcherDefinition{ID: "target"}, card.KindBorder, ""),
			Effects: []RuleEffect{
				SetDocumentVariantEffect("target", "empty"),
				CopyComponentEffect(RuleComponentTrigger, "target", card.KindBorder, "", ""),
			},
		}},
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	before := mustViewResult(t, session)
	result, err := session.Execute(ChangeWorldComponentCommand{
		CardID: "target", ComponentID: "target-border", ComponentKind: card.KindBorder,
		Control: "border_width_px", Value: json.RawMessage(`2`),
	})
	if err == nil || IsInvalidCommand(err) || !strings.Contains(err.Error(), "rule signal limit") {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Revision != 0 || len(result.Events) != 0 {
		t.Fatalf("failed result = %#v", result)
	}
	after := mustViewResult(t, session)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("loop failure did not roll back:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func ruleTestDeck(rules []RuleDefinition) DeckDefinition {
	return DeckDefinition{
		ID: "rule-test", Name: "Rule Test", InitialActiveCardID: "target",
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("target")}},
			{ID: "source", Name: "Source", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": sliderDocument("source", 3)}},
		},
		Rules: rules,
	}
}

func newRuleRuntimeSession(t *testing.T, rules []RuleDefinition) *Session {
	t.Helper()
	session, err := NewSessionFromDeck(catalog.MustNew(), ruleTestDeck(rules))
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func testCardPlayedTrigger() RuleTrigger {
	return CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"})
}

func collectAndPlayRuleSource(t *testing.T, session *Session) Result {
	t.Helper()
	mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	return mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "source", TargetCardID: "target"})
}

func assertResolvedRule(t *testing.T, events []Event, ruleID string, outcome RuleOutcome) {
	t.Helper()
	for _, event := range events {
		if event.Type != EventRuleResolved {
			continue
		}
		payload, ok := event.Payload.(RuleResolvedPayload)
		if !ok {
			t.Fatalf("ruleResolved payload = %#v", event.Payload)
		}
		if payload.RuleID != ruleID {
			continue
		}
		if payload.Outcome != string(outcome) {
			t.Fatalf("ruleResolved = %#v, want rule %q outcome %q", payload, ruleID, outcome)
		}
		return
	}
	t.Fatalf("no ruleResolved event in %#v", events)
}

func countResolvedRule(events []Event, ruleID string) int {
	count := 0
	for _, event := range events {
		if event.Type != EventRuleResolved {
			continue
		}
		payload, ok := event.Payload.(RuleResolvedPayload)
		if ok && payload.RuleID == ruleID {
			count++
		}
	}
	return count
}

func cardByID(cards []Card, cardID string) (Card, bool) {
	for _, candidate := range cards {
		if candidate.ID == cardID {
			return candidate, true
		}
	}
	return Card{}, false
}

func ruleIDs(rules []RuleDefinition) []string {
	ids := make([]string, len(rules))
	for index, rule := range rules {
		ids[index] = rule.ID
	}
	return ids
}
