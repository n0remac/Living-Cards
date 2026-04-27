package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/embedding"
)

func TestStoreSavesAndListsMemories(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	ctx := context.Background()

	first, err := store.SaveMemory(ctx, SaveInput{
		CardID:       "ember",
		UserInput:    "What is fear?",
		CardResponse: "Fear is the shadow cast by change.",
		Summary:      "User asked about fear; Ember answered poetically.",
	})
	if err != nil {
		t.Fatalf("SaveMemory(first) error = %v", err)
	}
	second, err := store.SaveMemory(ctx, SaveInput{
		CardID:       "ember",
		UserInput:    "What is courage?",
		CardResponse: "Courage is the ember that stays lit.",
		Summary:      "User asked about courage; Ember answered poetically.",
	})
	if err != nil {
		t.Fatalf("SaveMemory(second) error = %v", err)
	}

	memories, err := store.ListByCard(ctx, "ember", 10)
	if err != nil {
		t.Fatalf("ListByCard() error = %v", err)
	}
	if len(memories) != 2 {
		t.Fatalf("len(memories) = %d, want 2", len(memories))
	}
	if memories[0].ID != second.ID || memories[1].ID != first.ID {
		t.Fatalf("ListByCard() order = %#v", memories)
	}
}

func TestStoreSearchFiltersByCardID(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	ctx := context.Background()
	_, _ = store.SaveMemory(ctx, SaveInput{
		CardID:       "ember",
		UserInput:    "What is fear?",
		CardResponse: "Fear is the shadow cast by change.",
		Summary:      "Fear and change were discussed.",
	})
	_, _ = store.SaveMemory(ctx, SaveInput{
		CardID:       "river",
		UserInput:    "What is fear?",
		CardResponse: "Fear is a stone under water.",
		Summary:      "Fear was discussed near the river.",
	})

	results, err := store.Search(ctx, "ember", "fear", 5)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Memory.CardID != "ember" {
		t.Fatalf("results[0].Memory.CardID = %q", results[0].Memory.CardID)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir()+"/memory.db", newFakeVectorIndex(), "nomic-embed-text")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

type fakeVectorIndex struct {
	docs map[string]embedding.Document
}

func newFakeVectorIndex() *fakeVectorIndex {
	return &fakeVectorIndex{docs: make(map[string]embedding.Document)}
}

func (f *fakeVectorIndex) CollectionName(model string) string {
	return "memory-v1-" + model
}

func (f *fakeVectorIndex) UpsertDocuments(_ context.Context, _ string, docs []embedding.Document) (string, error) {
	for _, doc := range docs {
		f.docs[doc.DocumentID] = doc
	}
	return "memory-v1-nomic-embed-text", nil
}

func (f *fakeVectorIndex) Search(_ context.Context, _ string, query string, _ int, filters map[string]string) (embedding.SearchResponse, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	cardID := filters["card_id"]
	results := make([]embedding.SearchResult, 0)
	for _, doc := range f.docs {
		payloadCardID, _ := doc.Payload["card_id"].(string)
		if cardID != "" && payloadCardID != cardID {
			continue
		}
		if !strings.Contains(strings.ToLower(doc.Text), query) {
			continue
		}
		results = append(results, embedding.SearchResult{
			Rank:       len(results) + 1,
			Score:      1,
			DocumentID: doc.DocumentID,
			Text:       doc.Text,
			Payload:    doc.Payload,
		})
	}
	return embedding.SearchResponse{
		CollectionName: "memory-v1-nomic-embed-text",
		Results:        results,
	}, nil
}
