package cards

import "testing"

func TestStoreLoadsStaticCards(t *testing.T) {
	t.Parallel()

	store, err := NewStore([]Card{
		testCard("ember", "Ember"),
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if len(store.List()) != 1 {
		t.Fatalf("len(List()) = %d, want 1", len(store.List()))
	}
	card, ok := store.Get("ember")
	if !ok {
		t.Fatalf("Get() card not found")
	}
	if card.Name != "Ember" {
		t.Fatalf("card.Name = %q", card.Name)
	}
	if len(card.Components) != 1 || card.Components[0].ID != ChatFormComponentType {
		t.Fatalf("card.Components = %#v", card.Components)
	}
}

func TestStoreSortsCardsByName(t *testing.T) {
	t.Parallel()

	store, err := NewStore([]Card{
		testCard("zeta", "Zeta"),
		testCard("alpha", "Alpha"),
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	list := store.List()
	if list[0].CardID != "alpha" || list[1].CardID != "zeta" {
		t.Fatalf("List() order = %#v", list)
	}
}

func TestStoreRejectsInvalidCard(t *testing.T) {
	t.Parallel()

	if _, err := NewStore([]Card{{CardID: "", Name: "Ember"}}); err == nil {
		t.Fatal("NewStore() error = nil, want validation error")
	}
}

func TestStoreRejectsDuplicateComponentIDs(t *testing.T) {
	t.Parallel()

	card := testCard("ember", "Ember")
	card.Components = append(card.Components, ComponentInstance{
		ID:    ChatFormComponentType,
		Type:  ChatFormComponentType,
		Props: map[string]any{},
	})
	if _, err := NewStore([]Card{card}); err == nil {
		t.Fatal("NewStore() error = nil, want duplicate component error")
	}
}

func TestStoreClonesComponentInstances(t *testing.T) {
	t.Parallel()

	store, err := NewStore([]Card{testCard("ember", "Ember")})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	card, ok := store.Get("ember")
	if !ok {
		t.Fatal("Get() card not found")
	}
	card.Components[0].Props["label"] = "changed"

	card, ok = store.Get("ember")
	if !ok {
		t.Fatal("Get() card not found")
	}
	if _, exists := card.Components[0].Props["label"]; exists {
		t.Fatalf("component props were not cloned: %#v", card.Components[0].Props)
	}
}

func TestDefaultCatalogDefinesCards(t *testing.T) {
	t.Parallel()

	store, err := NewStaticStore()
	if err != nil {
		t.Fatalf("NewStaticStore() error = %v", err)
	}
	list := store.List()
	if len(list) != 2 {
		t.Fatalf("len(DefaultCatalog()) = %d, want 2", len(list))
	}
	for _, card := range list {
		if len(card.Components) == 0 {
			t.Fatalf("card %q has no components", card.CardID)
		}
	}
}

func testCard(cardID, name string) Card {
	return Card{
		CardID:    cardID,
		Name:      name,
		Archetype: "guardian",
		Personality: Personality{
			Tone:       "calm",
			StyleRules: []string{"speak softly"},
		},
		Constraints: Constraints{
			KnowledgeScope: "mythic",
			ToolAccess:     []string{},
		},
		Components: DefaultComponents(),
	}
}
