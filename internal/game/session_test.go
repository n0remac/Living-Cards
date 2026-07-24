package game

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
)

func TestEmbeddedDecksDecodeAgainstCatalog(t *testing.T) {
	registry := catalog.MustNew()
	ids := []string{SeededWorldDeckDefinition, FuseRoomDeckDefinition, GeneratorDeckDefinition, ArchiveTerminalDefinition}
	definitions := make([]DeckDefinition, 0, len(ids))
	known := map[string]CardDefinition{}
	for _, id := range ids {
		definition, err := LoadEmbeddedDeck(registry, id)
		if err != nil {
			t.Fatalf("LoadEmbeddedDeck(%q): %v", id, err)
		}
		definitions = append(definitions, definition)
		if id == SeededWorldDeckDefinition {
			if err := ValidateDeckDefinition(registry, definition); err != nil {
				t.Fatalf("ValidateDeckDefinition(%q): %v", id, err)
			}
		} else if err := ValidateDeckPackDefinition(registry, definition, known); err != nil {
			t.Fatalf("ValidateDeckPackDefinition(%q): %v", id, err)
		}
		for _, card := range definition.Cards {
			known[card.ID] = card
		}
	}
	if len(definitions) != len(ids) {
		t.Fatalf("definitions = %d", len(definitions))
	}
}

func TestSessionUsesInjectedRegistry(t *testing.T) {
	registry := catalog.MustNew()
	session, err := NewSessionFromEmbeddedDeck(registry)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.WorldDeck) == 0 || snapshot.ActiveWorldCard.ID != snapshot.ActiveWorldCardID {
		t.Fatalf("snapshot = %#v", snapshot)
	}
	if snapshot.ActiveWorldCard.Document.Root.ComponentKind != card.Kind {
		t.Fatalf("root = %#v", snapshot.ActiveWorldCard.Document.Root)
	}
	if _, err := NewSessionFromDeck(nil, DeckDefinition{}); err == nil {
		t.Fatal("NewSessionFromDeck(nil) succeeded")
	}
}

func TestSliderConditionReadsRegistryProperty(t *testing.T) {
	registry := catalog.MustNew()
	value := 7
	definition := DeckDefinition{
		ID: "property-test", Name: "Property Test", InitialActiveCardID: "target",
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("target")}},
			{ID: "source", Name: "Source", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": sliderDocument("source", value)}},
		},
		UseRules: []UseRuleDefinition{{
			ID: "property", Source: CardMatcherDefinition{ID: "source"}, Target: CardMatcherDefinition{ID: "target"},
			SourceComponentConditions: []ComponentConditionDefinition{{ComponentKind: card.KindSlider, ComponentID: "source-slider", ValueEquals: &value}},
			Effects:                   []RuleEffectDefinition{{EffectKind: EffectSetMessage, Message: "matched through property"}},
		}},
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Collect("source"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.UseCard("source", "target")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Message != "matched through property" {
		t.Fatalf("message = %q", snapshot.Message)
	}
}

func TestGenericWorldControlEditing(t *testing.T) {
	registry := catalog.MustNew()
	definition := DeckDefinition{ID: "edit-test", Name: "Edit Test", InitialActiveCardID: "target", Cards: []CardDefinition{{ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": sliderDocument("target", 4)}}}}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.ApplyWorldComponentControl("target", "target-slider", card.KindSlider, "value", json.RawMessage(`8`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snapshot.ActiveWorldCard.Document.Root.Children[0].Config), `"value":8`) {
		t.Fatalf("config = %s", snapshot.ActiveWorldCard.Document.Root.Children[0].Config)
	}
	if _, err := session.ApplyWorldComponentControl("target", "target-slider", card.KindSlider, "value", json.RawMessage(`999`)); err == nil {
		t.Fatal("invalid range edit succeeded")
	}
	if _, err := session.ApplyWorldComponentControl("target", "target-slider", card.KindSlider, "legacyValue", json.RawMessage(`5`)); err == nil {
		t.Fatal("legacy control alias succeeded")
	}
}

func TestFormDiscoveryUsesRegisteredRolesAndProperties(t *testing.T) {
	registry := catalog.MustNew()
	document := simpleDocument("form-card")
	document.Root.Children = []card.Node{
		{ID: "password", ComponentKind: card.KindTextInput, Config: json.RawMessage(`{"form_id":"login","name":"password"}`)},
		{ID: "submit", ComponentKind: card.KindButton, Config: json.RawMessage(`{"form_id":"login"}`)},
	}
	conditions := []FormFieldConditionDefinition{{Name: "password", ValueEquals: "secret"}}
	if !documentHasFormFields(registry, document, "login", conditions) || !documentHasSubmitButton(registry, document, "login") {
		t.Fatal("registered form roles were not discovered")
	}
	if documentHasFormFields(registry, document, "other", conditions) || documentHasSubmitButton(registry, document, "other") {
		t.Fatal("form discovery ignored form_id properties")
	}
}

func TestDeckDecoderRejectsUnknownDocumentFields(t *testing.T) {
	registry := catalog.MustNew()
	raw := []byte(`{"id":"strict","name":"Strict","initialActiveCardId":"one","cards":[{"id":"one","name":"One","kind":"world","collectible":false,"initialDocument":"default","documents":{"default":{"card_id":"one","name":"One","legacy":true,"root":{"id":"root","component_kind":"card"}}}}]}`)
	if _, err := DecodeDeckDefinition(registry, raw); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("DecodeDeckDefinition() error = %v", err)
	}
}

func simpleDocument(id string) card.Document {
	root, _ := json.Marshal(card.DefaultRootConfig())
	return card.Document{CardID: id, Name: id, Root: card.Node{ID: id + "-root", ComponentKind: card.Kind, Config: root}}
}

func sliderDocument(id string, value int) card.Document {
	document := simpleDocument(id)
	document.Root.Children = []card.Node{{ID: id + "-slider", ComponentKind: card.KindSlider, Config: json.RawMessage(`{"label":"Output","min":0,"max":10,"step":1,"value":` + jsonNumber(value) + `,"x":50,"y":50,"width":72,"track_color":"#111827","accent_color":"#38bdf8"}`)}}
	return document
}

func jsonNumber(value int) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
