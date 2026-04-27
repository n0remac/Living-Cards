package cards

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadsCards(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeCardFixture(t, dir, "ember.json", `{"card_id":"ember","name":"Ember","personality":{"tone":"calm","style_rules":["speak softly"]},"constraints":{"knowledge_scope":"mythic","tool_access":[]}}`)
	store, err := NewStore(dir)
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
}

func TestStoreRejectsInvalidCard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeCardFixture(t, dir, "invalid.json", `{"card_id":"","name":"Ember","personality":{},"constraints":{}}`)
	if _, err := NewStore(dir); err == nil {
		t.Fatal("NewStore() error = nil, want validation error")
	}
}

func writeCardFixture(t *testing.T, dir, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}
