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
	"github.com/n0remac/Living-Card/internal/profile"
)

func TestCardsListHandler(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards: fakeCardStore{
			cards: []cards.Card{{CardID: "ember", Name: "Ember"}},
		},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
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

func TestChatFormHandlerReturnsValidResponse(t *testing.T) {
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
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/ember/components/chat-form/actions/send", strings.NewReader(`{"user_id":"tester","message":"What is fear?"}`))
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

func TestChatFormHandlerReturnsNotFoundForInvalidCard(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:   fakeCardStore{},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/missing/components/chat-form/actions/send", strings.NewReader(`{"message":"hi"}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestChatFormHandlerRejectsMalformedRequest(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards: fakeCardStore{
			cards: []cards.Card{{CardID: "ember", Name: "Ember"}},
		},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/ember/components/chat-form/actions/send", strings.NewReader(`{`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestOldChatHandlerRouteIsNotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards: fakeCardStore{
			cards: []cards.Card{{CardID: "ember", Name: "Ember"}},
		},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/cards/ember/chat", strings.NewReader(`{"message":"hi"}`))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestProfileHandlerFetchesAndResetsProfile(t *testing.T) {
	t.Parallel()

	store := &fakeProfileStore{
		profile: profile.Profile{
			UserID:         "tester",
			ProfileSummary: "- preferences: likes concise replies",
			Facts: []profile.Fact{{
				ID:         "fact_1",
				UserID:     "tester",
				Key:        "preferences",
				Value:      "likes concise replies",
				Confidence: 0.9,
				Evidence:   "I like concise replies.",
				Status:     profile.StatusAccepted,
			}},
		},
	}
	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:   fakeCardStore{},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: store,
	})

	getRequest := httptest.NewRequest(http.MethodGet, "/api/users/tester/profile", nil)
	getRecorder := httptest.NewRecorder()
	mux.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200 body=%s", getRecorder.Code, getRecorder.Body.String())
	}
	var fetched profile.Profile
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("json.Unmarshal(GET) error = %v", err)
	}
	if fetched.UserID != "tester" || fetched.ProfileSummary == "" {
		t.Fatalf("GET payload = %#v", fetched)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/users/tester/profile", nil)
	deleteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want 200 body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	if !store.reset {
		t.Fatal("Reset was not called")
	}
}

func TestPageReferencesFrontendBundle(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:   fakeCardStore{},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `type="module"`) || !strings.Contains(body, `src="/assets/app.js"`) {
		t.Fatalf("page did not reference module frontend bundle: %s", body)
	}
}

func TestPageIncludesHeaderAndChatFormComponentMounts(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:   fakeCardStore{},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, marker := range []string{
		`id="app-header"`,
		`id="reload-cards-btn"`,
		`id="chat-form-component"`,
		`id="conversation-status"`,
		`id="card-meta"`,
		`id="transcript"`,
		`id="chat-form"`,
		`id="chat-input"`,
		`id="send-btn"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("page missing %s: %s", marker, body)
		}
	}
}

func TestFrontendAssetHandlerServesBundle(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:   fakeCardStore{},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/javascript") {
		t.Fatalf("Content-Type = %q, want javascript", contentType)
	}
	if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
	if !strings.Contains(recorder.Body.String(), "livingCardState") {
		t.Fatalf("bundle body did not contain expected app code")
	}
}

func TestFrontendAssetHandlerServesSourceMap(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Cards:   fakeCardStore{},
		Memory:  fakeMemoryStore{},
		Chat:    fakeChatService{},
		Profile: &fakeProfileStore{},
	})
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js.map", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want json", contentType)
	}
	if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
	if !strings.Contains(recorder.Body.String(), `"sources"`) {
		t.Fatalf("source map body did not contain sources")
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

func (fakeMemoryStore) ListByCard(context.Context, string, string, int) ([]memory.Memory, error) {
	return nil, nil
}

type fakeChatService struct {
	result chat.Result
	err    error
}

func (f fakeChatService) Chat(context.Context, chat.Request) (chat.Result, error) {
	return f.result, f.err
}

type fakeProfileStore struct {
	profile profile.Profile
	reset   bool
}

func (f *fakeProfileStore) Get(_ context.Context, userID string) (profile.Profile, error) {
	if f.profile.UserID == "" {
		return profile.Profile{UserID: userID}, nil
	}
	return f.profile, nil
}

func (f *fakeProfileStore) Reset(context.Context, string) error {
	f.reset = true
	f.profile.ProfileSummary = ""
	f.profile.Facts = nil
	return nil
}
