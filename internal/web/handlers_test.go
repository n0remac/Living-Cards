package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/chat"
	"github.com/n0remac/Living-Card/internal/memory"
)

func TestCardsListHandler(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards: fakeCardStore{
			cards: []cards.Card{{CardID: "ember", Name: "Ember"}},
		},
		Memory: fakeMemoryStore{},
		Chat:   fakeChatService{},
	})
	request := httptest.NewRequest(http.MethodGet, "/api/cards", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var payload []cards.Card
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload) != 1 || payload[0].CardID != "ember" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestChatHandlerReturnsValidResponse(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards: fakeCardStore{
			cards: []cards.Card{{CardID: "ember", Name: "Ember"}},
		},
		Memory: fakeMemoryStore{},
		Chat: fakeChatService{
			result: chat.Result{
				Card:              cards.Card{CardID: "ember", Name: "Ember"},
				AssistantResponse: "Fear is a shadow.",
			},
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/ember/chat", strings.NewReader(`{"message":"What is fear?"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload chat.Result
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.AssistantResponse != "Fear is a shadow." {
		t.Fatalf("AssistantResponse = %q", payload.AssistantResponse)
	}
}

func TestChatHandlerReturnsNotFoundForInvalidCard(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:  fakeCardStore{},
		Memory: fakeMemoryStore{},
		Chat:   fakeChatService{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/missing/chat", strings.NewReader(`{"message":"hi"}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestChatHandlerRejectsMalformedRequest(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards: fakeCardStore{
			cards: []cards.Card{{CardID: "ember", Name: "Ember"}},
		},
		Memory: fakeMemoryStore{},
		Chat:   fakeChatService{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/ember/chat", strings.NewReader(`{`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

type fakeCardStore struct {
	cards []cards.Card
}

func (f fakeCardStore) List() []cards.Card {
	return append([]cards.Card(nil), f.cards...)
}

func (f fakeCardStore) Get(cardID string) (cards.Card, bool) {
	for _, card := range f.cards {
		if card.CardID == cardID {
			return card, true
		}
	}
	return cards.Card{}, false
}

type fakeMemoryStore struct{}

func (fakeMemoryStore) ListByCard(context.Context, string, int) ([]memory.Memory, error) {
	return nil, nil
}

type fakeChatService struct {
	result chat.Result
	err    error
}

func (f fakeChatService) Chat(context.Context, chat.Request) (chat.Result, error) {
	return f.result, f.err
}
