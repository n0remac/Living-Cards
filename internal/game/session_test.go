package game

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSessionStartsWithEmptyLibraryAndScriptedDeck(t *testing.T) {
	t.Parallel()

	snapshot := mustResult(t, NewSession().Snapshot)
	if len(snapshot.Library) != 0 {
		t.Fatalf("library = %#v, want empty", snapshot.Library)
	}
	if len(snapshot.WorldDeck) < 5 {
		t.Fatalf("world deck length = %d, want scripted deck", len(snapshot.WorldDeck))
	}
	if snapshot.ActiveWorldCard.ID != "rusted-cell-door" {
		t.Fatalf("active card = %#v, want rusted-cell-door", snapshot.ActiveWorldCard)
	}
	if snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want locked door", snapshot.SolvedFlags)
	}
	if !documentContains(snapshot.ActiveWorldCard.Document, "LOCKED") {
		t.Fatalf("active door document should be locked: %#v", snapshot.ActiveWorldCard.Document)
	}
}

func TestEmbeddedSeededWorldDeckLoadsAndValidates(t *testing.T) {
	t.Parallel()

	definition, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		t.Fatalf("LoadEmbeddedSeededWorldDeck() error = %v", err)
	}
	if definition.ID != SeededWorldDeckDefinition {
		t.Fatalf("deck id = %q, want %q", definition.ID, SeededWorldDeckDefinition)
	}
	session, err := NewSessionFromDeck(definition)
	if err != nil {
		t.Fatalf("NewSessionFromDeck() error = %v", err)
	}
	snapshot := mustResult(t, session.Snapshot)
	key := findCard(snapshot.WorldDeck, "bent-iron-key")
	if key == nil {
		t.Fatal("world deck missing key")
	}
	if !documentContains(key.Document, "BENT IRON KEY") {
		t.Fatalf("key document should come from deck data: %#v", key.Document)
	}
}

func TestSessionCycleWrapsPreviousAndNext(t *testing.T) {
	t.Parallel()

	session := NewSession()
	previous := mustResult(t, func() (Snapshot, error) {
		return session.Cycle("previous")
	})
	if previous.ActiveIndex != len(previous.WorldDeck)-1 || previous.ActiveWorldCard.ID != "sleeping-switch" {
		t.Fatalf("previous snapshot = %#v", previous)
	}
	next := mustResult(t, func() (Snapshot, error) {
		return session.Cycle("next")
	})
	if next.ActiveIndex != 0 || next.ActiveWorldCard.ID != "rusted-cell-door" {
		t.Fatalf("next snapshot = %#v", next)
	}
}

func TestSessionCollectsKeyIntoLibrary(t *testing.T) {
	t.Parallel()

	snapshot := mustResult(t, func() (Snapshot, error) {
		return NewSession().Collect("bent-iron-key")
	})
	if len(snapshot.Library) != 1 || snapshot.Library[0].ID != "bent-iron-key" {
		t.Fatalf("library = %#v, want collected key", snapshot.Library)
	}
	key := findCard(snapshot.WorldDeck, "bent-iron-key")
	if key == nil {
		t.Fatal("world deck missing key")
	}
	if key.Collectible || !key.Collected {
		t.Fatalf("world key = %#v, want collected and non-collectible", *key)
	}
}

func TestSessionRejectsDecoyCollection(t *testing.T) {
	t.Parallel()

	if _, err := NewSession().Collect("inventory-label"); err == nil {
		t.Fatal("Collect() decoy error = nil, want error")
	}
}

func TestSessionWrongCardUseDoesNotUnlockDoor(t *testing.T) {
	t.Parallel()

	session := NewSession()
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "faded-photograph")
	})
	if snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want locked door", snapshot.SolvedFlags)
	}
	if !strings.Contains(snapshot.Message, "Nothing on this card responds") {
		t.Fatalf("message = %q, want wrong-card message", snapshot.Message)
	}
}

func TestSessionKeyUnlocksDoorWithEffect(t *testing.T) {
	t.Parallel()

	session := NewSession()
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	if !snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want door unlocked", snapshot.SolvedFlags)
	}
	if len(snapshot.Library) != 1 || snapshot.Library[0].ID != "bent-iron-key" {
		t.Fatalf("library = %#v, key should remain visible", snapshot.Library)
	}
	door := findCard(snapshot.WorldDeck, "rusted-cell-door")
	if door == nil {
		t.Fatal("world deck missing door")
	}
	if locked, _ := door.State["locked"].(bool); locked {
		t.Fatalf("door state = %#v, want unlocked", door.State)
	}
	if hasTag(*door, "locked") {
		t.Fatalf("door tags = %#v, want locked tag removed", door.Tags)
	}
	if !documentContains(door.Document, "OPEN") {
		t.Fatalf("door document did not switch to open variant: %#v", door.Document)
	}
}

func TestSessionUseRulesComeFromDeckData(t *testing.T) {
	t.Parallel()

	definition, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		t.Fatalf("LoadEmbeddedSeededWorldDeck() error = %v", err)
	}
	definition.UseRules = nil
	session, err := NewSessionFromDeck(definition)
	if err != nil {
		t.Fatalf("NewSessionFromDeck() error = %v", err)
	}
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	if snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want locked door without data rule", snapshot.SolvedFlags)
	}
	door := findCard(snapshot.WorldDeck, "rusted-cell-door")
	if door == nil {
		t.Fatal("world deck missing door")
	}
	if !documentContains(door.Document, "LOCKED") {
		t.Fatalf("door document should remain locked without data rule: %#v", door.Document)
	}
}

func TestValidateDeckDefinitionRejectsInvalidFixtures(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		t.Fatalf("LoadEmbeddedSeededWorldDeck() error = %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*DeckDefinition)
	}{
		{
			name: "duplicate card id",
			mutate: func(definition *DeckDefinition) {
				definition.Cards = append(definition.Cards, definition.Cards[0])
			},
		},
		{
			name: "missing initial document variant",
			mutate: func(definition *DeckDefinition) {
				definition.Cards[0].InitialDocument = "missing"
			},
		},
		{
			name: "bad rule target reference",
			mutate: func(definition *DeckDefinition) {
				definition.UseRules[0].Target.ID = "missing-card"
			},
		},
		{
			name: "bad rule document variant reference",
			mutate: func(definition *DeckDefinition) {
				for index := range definition.UseRules[0].Effects {
					if definition.UseRules[0].Effects[index].Type == EffectSetDocumentVariant {
						definition.UseRules[0].Effects[index].Variant = "missing"
					}
				}
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			definition := cloneValue(base)
			test.mutate(&definition)
			if err := ValidateDeckDefinition(definition); err == nil {
				t.Fatal("ValidateDeckDefinition() error = nil, want error")
			}
		})
	}
}

func mustResult(t *testing.T, result func() (Snapshot, error)) Snapshot {
	t.Helper()
	snapshot, err := result()
	if err != nil {
		t.Fatalf("snapshot error = %v", err)
	}
	return snapshot
}

func findCard(cards []Card, id string) *Card {
	for index := range cards {
		if cards[index].ID == id {
			return &cards[index]
		}
	}
	return nil
}

func documentContains(document any, marker string) bool {
	raw, err := json.Marshal(document)
	return err == nil && strings.Contains(string(raw), marker)
}
