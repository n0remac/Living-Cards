package chat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/embedding"
	"github.com/n0remac/Living-Card/internal/memory"
	"github.com/n0remac/Living-Card/internal/ollama"
)

func TestBuildPromptIncludesIdentityAndMemories(t *testing.T) {
	t.Parallel()

	card := cards.Card{
		CardID:    "ember",
		Name:      "Ember",
		Archetype: "ancient guardian",
		Domain:    []string{"fire"},
		Personality: cards.Personality{
			Tone:       "calm, proud, poetic",
			StyleRules: []string{"use metaphor", "avoid slang"},
		},
		Constraints: cards.Constraints{KnowledgeScope: "abstract and philosophical"},
	}
	systemPrompt, userPrompt := BuildPrompt(card, "What is fear?", []memory.SearchResult{
		{Memory: memory.Memory{Summary: "The user once asked about courage."}},
	})
	if !strings.Contains(systemPrompt, "You are Ember.") {
		t.Fatalf("systemPrompt missing identity: %q", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "use metaphor") {
		t.Fatalf("systemPrompt missing style rule: %q", systemPrompt)
	}
	if !strings.Contains(userPrompt, "The user once asked about courage.") {
		t.Fatalf("userPrompt missing memory: %q", userPrompt)
	}
	if !strings.Contains(userPrompt, "What is fear?") {
		t.Fatalf("userPrompt missing input: %q", userPrompt)
	}
}

func TestBuildPromptHandlesEmptyMemories(t *testing.T) {
	t.Parallel()

	systemPrompt, userPrompt := BuildPrompt(cards.Card{
		CardID: "ember",
		Name:   "Ember",
		Personality: cards.Personality{
			Tone: "calm",
		},
		Constraints: cards.Constraints{
			KnowledgeScope: "abstract",
		},
	}, "Hello", nil)
	if !strings.Contains(systemPrompt, "Stay in character") {
		t.Fatalf("systemPrompt = %q", systemPrompt)
	}
	if !strings.Contains(userPrompt, "- none") {
		t.Fatalf("userPrompt missing empty memory marker: %q", userPrompt)
	}
}

func TestServiceChatPersistsMemory(t *testing.T) {
	t.Parallel()

	service, store := newTestChatService(t, chatFixture{chatResponses: []string{
		"Fear is the shadow cast by change.",
		"User asked about fear; Ember answered poetically.",
	}})

	result, err := service.Chat(context.Background(), Request{
		CardID:  "ember_stag_001",
		Message: "What is fear?",
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if result.AssistantResponse == "" {
		t.Fatalf("AssistantResponse is empty")
	}
	memories, err := store.ListByCard(context.Background(), "ember_stag_001", 10)
	if err != nil {
		t.Fatalf("ListByCard() error = %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("len(memories) = %d, want 1", len(memories))
	}
}

func TestServiceChatSurfacesQdrantFailure(t *testing.T) {
	t.Parallel()

	service, store := newTestChatService(t, chatFixture{
		chatResponses: []string{
			"Fear is the shadow cast by change.",
			"User asked about fear; Ember answered poetically.",
		},
		failQdrantUpsert: true,
	})

	if _, err := service.Chat(context.Background(), Request{
		CardID:  "ember_stag_001",
		Message: "What is fear?",
	}); err == nil {
		t.Fatal("Chat() error = nil, want qdrant failure")
	}
	memories, err := store.ListByCard(context.Background(), "ember_stag_001", 10)
	if err != nil {
		t.Fatalf("ListByCard() error = %v", err)
	}
	if len(memories) != 0 {
		t.Fatalf("len(memories) = %d, want 0 after qdrant failure", len(memories))
	}
}

func TestServiceChatDoesNotWritePartialMemoryOnOllamaFailure(t *testing.T) {
	t.Parallel()

	service, store := newTestChatService(t, chatFixture{chatError: errors.New("chat failure")})
	if _, err := service.Chat(context.Background(), Request{
		CardID:  "ember_stag_001",
		Message: "What is fear?",
	}); err == nil {
		t.Fatal("Chat() error = nil, want ollama failure")
	}
	memories, err := store.ListByCard(context.Background(), "ember_stag_001", 10)
	if err != nil {
		t.Fatalf("ListByCard() error = %v", err)
	}
	if len(memories) != 0 {
		t.Fatalf("len(memories) = %d, want 0 after ollama failure", len(memories))
	}
}

func TestChatSmokeRetrievesPriorMemory(t *testing.T) {
	t.Parallel()

	service, _ := newTestChatService(t, chatFixture{chatResponses: []string{
		"Fear is the shadow cast by change.",
		"User asked about fear; Ember answered poetically.",
		"Courage is the ember that stays lit.",
		"User asked about courage after fear; Ember answered poetically.",
	}})

	if _, err := service.Chat(context.Background(), Request{
		CardID:  "ember_stag_001",
		Message: "What is fear?",
	}); err != nil {
		t.Fatalf("Chat(first) error = %v", err)
	}
	second, err := service.Chat(context.Background(), Request{
		CardID:  "ember_stag_001",
		Message: "And what is courage?",
	})
	if err != nil {
		t.Fatalf("Chat(second) error = %v", err)
	}
	if len(second.RetrievedMemories) == 0 {
		t.Fatalf("len(RetrievedMemories) = 0, want prior memory")
	}
}

type chatFixture struct {
	chatResponses    []string
	chatError        error
	failQdrantUpsert bool
}

func newTestChatService(t *testing.T, fixture chatFixture) (*Service, *memory.Store) {
	t.Helper()

	cardStore, err := cards.NewStore("../../data/cards")
	if err != nil {
		t.Fatalf("cards.NewStore() error = %v", err)
	}
	store, err := memory.NewStore(t.TempDir()+"/memory.db", newChatVectorIndex(fixture.failQdrantUpsert), "nomic-embed-text")
	if err != nil {
		t.Fatalf("memory.NewStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	service := NewService(Config{
		Cards:          cardStore,
		Memory:         store,
		Ollama:         &fakeChatClient{responses: fixture.chatResponses, err: fixture.chatError},
		ChatModel:      "qwen2.5:3b-instruct",
		RequestTimeout: 0,
		TopK:           3,
	})
	return service, store
}

func TestErrCardNotFoundIsReturned(t *testing.T) {
	t.Parallel()
	service := NewService(Config{
		Cards:  fakeCards{},
		Memory: fakeMemory{},
		Ollama: &fakeChatClient{},
	})
	_, err := service.Chat(context.Background(), Request{CardID: "missing", Message: "hello"})
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Chat() error = %v, want ErrCardNotFound", err)
	}
}

type fakeCards struct{}

func (fakeCards) Get(string) (cards.Card, bool) { return cards.Card{}, false }

type fakeMemory struct{}

func (fakeMemory) SaveMemory(context.Context, memory.SaveInput) (memory.Memory, error) {
	return memory.Memory{}, nil
}

func (fakeMemory) Search(context.Context, string, string, int) ([]memory.SearchResult, error) {
	return nil, nil
}

type fakeChatClient struct {
	responses []string
	err       error
	index     int
}

func (f *fakeChatClient) Chat(_ context.Context, _ string, _ []ollama.ChatMessage) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if f.index >= len(f.responses) {
		return "", errors.New("missing fixture response")
	}
	response := f.responses[f.index]
	f.index++
	return response, nil
}

type chatVectorIndex struct {
	failUpsert bool
	docs       map[string]embedding.Document
}

func newChatVectorIndex(failUpsert bool) *chatVectorIndex {
	return &chatVectorIndex{
		failUpsert: failUpsert,
		docs:       make(map[string]embedding.Document),
	}
}

func (f *chatVectorIndex) CollectionName(model string) string {
	return "chat-v1-" + model
}

func (f *chatVectorIndex) UpsertDocuments(_ context.Context, _ string, docs []embedding.Document) (string, error) {
	if f.failUpsert {
		return "", errors.New("upsert failed")
	}
	for _, doc := range docs {
		f.docs[doc.DocumentID] = doc
	}
	return "chat-v1-nomic-embed-text", nil
}

func (f *chatVectorIndex) Search(_ context.Context, _ string, query string, topK int, filters map[string]string) (embedding.SearchResponse, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	cardID := filters["card_id"]
	results := make([]embedding.SearchResult, 0)
	for _, doc := range f.docs {
		payloadCardID, _ := doc.Payload["card_id"].(string)
		if cardID != "" && payloadCardID != cardID {
			continue
		}
		if !strings.Contains(strings.ToLower(doc.Text), "fear") && strings.Contains(query, "courage") {
			continue
		}
		results = append(results, embedding.SearchResult{
			Rank:       len(results) + 1,
			Score:      1,
			DocumentID: doc.DocumentID,
			Text:       doc.Text,
			Payload:    doc.Payload,
		})
		if topK > 0 && len(results) >= topK {
			break
		}
	}
	return embedding.SearchResponse{
		CollectionName: "chat-v1-nomic-embed-text",
		Results:        results,
	}, nil
}
