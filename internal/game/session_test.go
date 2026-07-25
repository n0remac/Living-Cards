package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	backgroundcomponent "github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
	textcomponent "github.com/n0remac/Living-Card/internal/components/text"
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

func TestEmbeddedDecksUseGeneratedArtworkAndOpaqueTextPanels(t *testing.T) {
	registry := catalog.MustNew()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate test file")
	}
	assetDirectory := filepath.Join(filepath.Dir(testFile), "..", "..", "web", "assets", "card-backgrounds")

	for _, deckID := range []string{SeededWorldDeckDefinition, FuseRoomDeckDefinition, GeneratorDeckDefinition, ArchiveTerminalDefinition} {
		definition, err := LoadEmbeddedDeck(registry, deckID)
		if err != nil {
			t.Fatalf("LoadEmbeddedDeck(%q): %v", deckID, err)
		}
		for _, cardDefinition := range definition.Cards {
			assetPath := filepath.Join(assetDirectory, cardDefinition.ID+".webp")
			assetInfo, err := os.Stat(assetPath)
			if err != nil {
				t.Fatalf("card %q background asset: %v", cardDefinition.ID, err)
			}
			if !assetInfo.Mode().IsRegular() || assetInfo.Size() == 0 {
				t.Fatalf("card %q background asset is not a non-empty regular file", cardDefinition.ID)
			}

			for variant, document := range cardDefinition.Documents {
				backgroundCount, textCount := 0, 0
				for _, child := range document.Root.Children {
					switch child.ComponentKind {
					case card.KindBackground:
						backgroundCount++
						var config backgroundcomponent.Config
						if err := json.Unmarshal(child.Config, &config); err != nil {
							t.Fatalf("card %q variant %q background config: %v", cardDefinition.ID, variant, err)
						}
						if config.AssetID != cardDefinition.ID {
							t.Fatalf("card %q variant %q background asset = %q", cardDefinition.ID, variant, config.AssetID)
						}
					case card.KindText:
						textCount++
						var config textcomponent.Config
						if err := json.Unmarshal(child.Config, &config); err != nil {
							t.Fatalf("card %q variant %q text %q config: %v", cardDefinition.ID, variant, child.ID, err)
						}
						if config.BackgroundColor != "#101713" || config.PaddingPX < 10 || config.BorderWidthPX < 1 {
							t.Fatalf("card %q variant %q text %q is not an opaque readable panel: %#v", cardDefinition.ID, variant, child.ID, config)
						}
					case card.KindShape, card.KindImage:
						t.Fatalf("card %q variant %q still uses illustration component %q", cardDefinition.ID, variant, child.ComponentKind)
					}
				}
				if backgroundCount != 1 {
					t.Fatalf("card %q variant %q backgrounds = %d, want 1", cardDefinition.ID, variant, backgroundCount)
				}
				if textCount == 0 {
					t.Fatalf("card %q variant %q has no readable text panel", cardDefinition.ID, variant)
				}
			}
		}
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
	if containsCard(snapshot.Library, "source") {
		t.Fatal("successfully played source remained in library")
	}
}

func TestCollectMovesCardsOutOfWorldDeckAndPreservesActiveCard(t *testing.T) {
	registry := catalog.MustNew()
	definition := DeckDefinition{
		ID: "collection-test", Name: "Collection Test", InitialActiveCardID: "first",
		Cards: []CardDefinition{
			{ID: "first", Name: "First", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("first")}},
			{ID: "anchor", Name: "Anchor", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("anchor")}},
			{ID: "later", Name: "Later", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("later")}},
		},
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := session.Collect("later")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ActiveWorldCardID != "first" || containsCard(snapshot.WorldDeck, "later") || !containsCard(snapshot.Library, "later") {
		t.Fatalf("snapshot after non-active collect = %#v", snapshot)
	}

	snapshot, err = session.Collect("first")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ActiveWorldCardID != "anchor" || snapshot.ActiveIndex != 0 {
		t.Fatalf("active card after collection = %q at %d", snapshot.ActiveWorldCardID, snapshot.ActiveIndex)
	}
	if containsCard(snapshot.WorldDeck, "first") || !containsCard(snapshot.Library, "first") || len(snapshot.Library) != 2 {
		t.Fatalf("ownership after active collect: world=%v library=%v", cardIDs(snapshot.WorldDeck), cardIDs(snapshot.Library))
	}
	if _, err := session.Collect("first"); err == nil {
		t.Fatal("collecting an already owned card succeeded")
	}
}

func TestCollectingLastActiveCardWrapsToFirstRemainingCard(t *testing.T) {
	registry := catalog.MustNew()
	definition := DeckDefinition{
		ID: "collection-wrap", Name: "Collection Wrap", InitialActiveCardID: "last",
		Cards: []CardDefinition{
			{ID: "anchor", Name: "Anchor", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("anchor")}},
			{ID: "last", Name: "Last", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("last")}},
		},
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.Collect("last")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ActiveWorldCardID != "anchor" || snapshot.ActiveIndex != 0 {
		t.Fatalf("active card = %q at %d", snapshot.ActiveWorldCardID, snapshot.ActiveIndex)
	}
}

func TestUseCardConsumesMatchedAttemptButRetainsUnmatchedTarget(t *testing.T) {
	registry := catalog.MustNew()
	requiredValue := 7
	definition := DeckDefinition{
		ID: "play-test", Name: "Play Test", InitialActiveCardID: "target",
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("target")}},
			{ID: "other", Name: "Other", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("other")}},
			{ID: "source", Name: "Source", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": sliderDocument("source", 3)}},
		},
		UseRules: []UseRuleDefinition{{
			ID: "conditional", Source: CardMatcherDefinition{ID: "source"}, Target: CardMatcherDefinition{ID: "target"},
			SourceComponentConditions: []ComponentConditionDefinition{{ComponentKind: card.KindSlider, ValueEquals: &requiredValue}},
			FailureMessage:            "The value is wrong.",
			Effects:                   []RuleEffectDefinition{{EffectKind: EffectSetMessage, Message: "matched"}},
		}},
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Collect("source"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.UseCard("source", "other")
	if err != nil {
		t.Fatal(err)
	}
	if !containsCard(snapshot.Library, "source") {
		t.Fatal("unmatched target consumed source")
	}
	snapshot, err = session.UseCard("source", "target")
	if err != nil {
		t.Fatal(err)
	}
	if containsCard(snapshot.Library, "source") || snapshot.Message != "The value is wrong." {
		t.Fatalf("matched failure snapshot = %#v", snapshot)
	}
}

func TestDeckRequiresPersistentWorldCard(t *testing.T) {
	registry := catalog.MustNew()
	definition := DeckDefinition{
		ID: "collectible-only", Name: "Collectible Only", InitialActiveCardID: "only",
		Cards: []CardDefinition{{
			ID: "only", Name: "Only", Kind: KindItem, Collectible: true,
			InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("only")},
		}},
	}
	if err := ValidateDeckDefinition(registry, definition); err == nil || !strings.Contains(err.Error(), "non-collectible") {
		t.Fatalf("ValidateDeckDefinition() error = %v", err)
	}
}

func TestComponentCardEditSaveConvertsBaseAndCancelPreservesLibrary(t *testing.T) {
	registry := catalog.MustNew()
	definition, err := LoadEmbeddedDeck(registry, GeneratorDeckDefinition)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	if containsCard(mustSnapshot(t, session).WorldDeck, "blank-controller") {
		t.Fatal("generator deck still contains blank controller")
	}
	if _, err := session.Collect("slider-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Collect("border-component"); err != nil {
		t.Fatal(err)
	}

	started, err := session.StartEdit("slider-component")
	if err != nil {
		t.Fatal(err)
	}
	if started.EditSession == nil || started.EditSession.SelectedComponentID == "" || findNodeByKind(started.EditSession.DraftCard.Document.Root, card.KindSlider) == nil {
		t.Fatalf("component base was not installed into draft: %#v", started.EditSession)
	}
	if _, err := session.InstallEditComponent("slider-component"); err == nil {
		t.Fatal("component card installed itself")
	}
	if _, err := session.InstallEditComponent("border-component"); err != nil {
		t.Fatal(err)
	}
	canceled, err := session.CancelEdit()
	if err != nil {
		t.Fatal(err)
	}
	if len(canceled.Library) != 2 || !containsCard(canceled.Library, "slider-component") || !containsCard(canceled.Library, "border-component") {
		t.Fatalf("cancel consumed cards: %v", cardIDs(canceled.Library))
	}
	if _, declared := canceled.Library[0].State[componentTemplateStateKey]; !declared {
		t.Fatal("cancel converted the component base")
	}

	started, err = session.StartEdit("slider-component")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ApplyEditControl(started.EditSession.SelectedComponentID, "value", json.RawMessage(`73`)); err != nil {
		t.Fatal(err)
	}
	if _, err := session.InstallEditComponent("border-component"); err != nil {
		t.Fatal(err)
	}
	saved, err := session.SaveEdit()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Library) != 1 || saved.Library[0].ID != "slider-component" || saved.Library[0].Name != "Slider Component" {
		t.Fatalf("saved library = %#v", saved.Library)
	}
	controller := saved.Library[0]
	if _, declared := controller.State[componentTemplateStateKey]; declared {
		t.Fatal("saved controller retained component template")
	}
	if !stateBool(controller.State, "editable") || !stateBool(controller.State, "built") || !hasTag(controller, "controller") {
		t.Fatalf("controller metadata = %#v tags=%v", controller.State, controller.Tags)
	}
	for _, unwanted := range []string{"component", "slider-component"} {
		if hasTag(controller, unwanted) {
			t.Fatalf("controller retained component tag %q: %v", unwanted, controller.Tags)
		}
	}
	installed := appendStateStringOnce(controller.State["installedComponents"], "")
	if !stringInSlice(installed, card.KindSlider) || !stringInSlice(installed, card.KindBorder) {
		t.Fatalf("installed components = %v", installed)
	}
	if findNodeByKind(controller.Document.Root, card.KindSlider) == nil || findNodeByKind(controller.Document.Root, card.KindBorder) == nil {
		t.Fatalf("controller document = %#v", controller.Document)
	}
	if _, err := session.StartEdit("slider-component"); err != nil {
		t.Fatalf("saved controller could not be edited again: %v", err)
	}
}

func TestEveryEmbeddedComponentCardCanBeAnEditBase(t *testing.T) {
	registry := catalog.MustNew()
	tests := []struct {
		deckID        string
		cardID        string
		componentKind string
	}{
		{GeneratorDeckDefinition, "slider-component", card.KindSlider},
		{GeneratorDeckDefinition, "border-component", card.KindBorder},
		{ArchiveTerminalDefinition, "archive-password-input-component", card.KindTextInput},
		{ArchiveTerminalDefinition, "archive-submit-button-component", card.KindButton},
	}
	for _, test := range tests {
		t.Run(test.cardID, func(t *testing.T) {
			definition, err := LoadEmbeddedDeck(registry, test.deckID)
			if err != nil {
				t.Fatal(err)
			}
			session, err := NewSessionFromDeck(registry, definition)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := session.Collect(test.cardID); err != nil {
				t.Fatal(err)
			}
			snapshot, err := session.StartEdit(test.cardID)
			if err != nil {
				t.Fatal(err)
			}
			if snapshot.EditSession == nil || snapshot.EditSession.SelectedComponentID == "" || findNodeByKind(snapshot.EditSession.DraftCard.Document.Root, test.componentKind) == nil {
				t.Fatalf("%s was not installed as the edit base", test.componentKind)
			}
		})
	}
}

func TestFailedRegulatorPlayMovesSliderToFieldAndCanRevealArchive(t *testing.T) {
	registry := catalog.MustNew()
	definition, err := LoadEmbeddedDeck(registry, GeneratorDeckDefinition)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSessionFromDeck(registry, definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.Collect("slider-component"); err != nil {
		t.Fatal(err)
	}
	started, err := session.StartEdit("slider-component")
	if err != nil {
		t.Fatal(err)
	}
	sliderID := started.EditSession.SelectedComponentID
	if _, err := session.SaveEdit(); err != nil {
		t.Fatal(err)
	}
	failed, err := session.UseCard("slider-component", "generator-panel")
	if err != nil {
		t.Fatal(err)
	}
	if containsCard(failed.Library, "slider-component") || findNodeByID(failed.ActiveWorldCard.Document.Root, sliderID) == nil {
		t.Fatalf("failed play did not move controller to field: %#v", failed)
	}
	revealed, err := session.ApplyWorldComponentControl("generator-panel", sliderID, card.KindSlider, "value", json.RawMessage(`73`))
	if err != nil {
		t.Fatal(err)
	}
	if !revealed.SolvedFlags[GeneratorPoweredFlag] || !containsCard(revealed.WorldDeck, "archive-terminal") || revealed.ActiveWorldCardID != "archive-terminal" {
		t.Fatalf("archive was not revealed: active=%q flags=%v world=%v", revealed.ActiveWorldCardID, revealed.SolvedFlags, cardIDs(revealed.WorldDeck))
	}
}

func TestEmbeddedPuzzleUsesComponentCardsAsControllerBases(t *testing.T) {
	registry := catalog.MustNew()
	session, err := NewSessionFromEmbeddedDeck(registry)
	if err != nil {
		t.Fatal(err)
	}

	collectAndUse(t, session, "bent-iron-key", "rusted-cell-door")
	collectAndUse(t, session, "glass-fuse", "sleeping-switch")
	if _, err := session.Collect("slider-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Collect("border-component"); err != nil {
		t.Fatal(err)
	}
	started, err := session.StartEdit("slider-component")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := session.ApplyEditControl(started.EditSession.SelectedComponentID, "value", json.RawMessage(`73`)); err != nil {
		t.Fatal(err)
	}
	if _, err := session.InstallEditComponent("border-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.SaveEdit(); err != nil {
		t.Fatal(err)
	}
	generator, err := session.UseCard("slider-component", "generator-panel")
	if err != nil {
		t.Fatal(err)
	}
	if containsCard(generator.Library, "slider-component") || generator.ActiveWorldCardID != "archive-terminal" {
		t.Fatalf("generator play = active %q library %v", generator.ActiveWorldCardID, cardIDs(generator.Library))
	}

	if _, err := session.Collect("archive-password-input-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Collect("archive-submit-button-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.StartEdit("archive-password-input-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.InstallEditComponent("archive-submit-button-component"); err != nil {
		t.Fatal(err)
	}
	if _, err := session.SaveEdit(); err != nil {
		t.Fatal(err)
	}
	mounted, err := session.UseCard("archive-password-input-component", "archive-terminal")
	if err != nil {
		t.Fatal(err)
	}
	if containsCard(mounted.Library, "archive-password-input-component") || !mounted.SolvedFlags["archiveControllerMounted"] {
		t.Fatalf("archive controller was not mounted: %#v", mounted)
	}
	unlocked, err := session.SubmitForm("archive-terminal", "archive-login", map[string]string{"password": " nightjar "})
	if err != nil {
		t.Fatal(err)
	}
	if !unlocked.SolvedFlags["archiveUnlocked"] || unlocked.Message != "NIGHTJAR accepted. The archive opens." {
		t.Fatalf("archive unlock = flags %v message %q", unlocked.SolvedFlags, unlocked.Message)
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

func containsCard(cards []Card, cardID string) bool {
	for _, candidate := range cards {
		if candidate.ID == cardID {
			return true
		}
	}
	return false
}

func cardIDs(cards []Card) []string {
	ids := make([]string, 0, len(cards))
	for _, candidate := range cards {
		ids = append(ids, candidate.ID)
	}
	return ids
}

func mustSnapshot(t *testing.T, session *Session) Snapshot {
	t.Helper()
	snapshot, err := session.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func collectAndUse(t *testing.T, session *Session, sourceCardID, targetCardID string) {
	t.Helper()
	if _, err := session.Collect(sourceCardID); err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.UseCard(sourceCardID, targetCardID)
	if err != nil {
		t.Fatal(err)
	}
	if containsCard(snapshot.Library, sourceCardID) {
		t.Fatalf("%s remained in library after play", sourceCardID)
	}
}
