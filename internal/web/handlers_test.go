package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
	"github.com/n0remac/Living-Card/internal/ollama"
)

func TestPageUsesCanonicalComponentVocabulary(t *testing.T) {
	recorder := request(t, testMux(nil), http.MethodGet, "/", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, marker := range []string{`src="/assets/app.js"`, `id="living-card-stage"`, `value="text"`, `id="add-text-component-btn"`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("page missing %q", marker)
		}
	}
	for _, legacy := range []string{`value="textarea"`, `value="textinput"`, `id="add-textarea-component-btn"`} {
		if strings.Contains(body, legacy) {
			t.Fatalf("page contains legacy vocabulary %q", legacy)
		}
	}
}

func TestCardBackgroundAssetsAreServedSafely(t *testing.T) {
	mux := testMux(nil)

	recorder := request(t, mux, http.MethodGet, "/assets/card-backgrounds/rusted-cell-door.webp", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("asset status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/webp" {
		t.Fatalf("content type = %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control = %q", got)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("asset response is empty")
	}

	head := request(t, mux, http.MethodHead, "/assets/card-backgrounds/rusted-cell-door.webp", "")
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD status = %d body bytes = %d", head.Code, head.Body.Len())
	}

	for _, path := range []string{
		"/assets/card-backgrounds/rusted-cell-door.json",
		"/assets/card-backgrounds/Rusted-Cell-Door.webp",
		"/assets/card-backgrounds/nested/rusted-cell-door.webp",
		"/assets/card-backgrounds/rusted-cell-door.webp/extra",
	} {
		recorder := request(t, mux, http.MethodGet, path, "")
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d", path, recorder.Code)
		}
	}
	for _, path := range []string{
		"/assets/card-backgrounds/../app.webp",
		"/assets/card-backgrounds/%2e%2e/app.webp",
		"/assets/card-backgrounds//rusted-cell-door.webp",
	} {
		if assetID, ok := cardBackgroundAssetID(path); ok {
			t.Fatalf("unsafe path %q resolved to %q", path, assetID)
		}
	}

	post := request(t, mux, http.MethodPost, "/assets/card-backgrounds/rusted-cell-door.webp", "")
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d", post.Code)
	}
}

func TestDraftResourcesExposeCanonicalDocuments(t *testing.T) {
	mux := testMux(nil)
	recorder := request(t, mux, http.MethodGet, "/api/draft-card", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"component_kind":"text"`) || strings.Contains(recorder.Body.String(), `"componentKind"`) {
		t.Fatalf("document is not canonical: %s", recorder.Body.String())
	}
	var document card.Document
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.CardID != card.DefaultCardID || len(document.Root.Children) != 3 {
		t.Fatalf("document = %#v", document)
	}

	recorder = request(t, mux, http.MethodGet, "/api/draft-card/rendered", "")
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `data-component-kind=\"text\"`) {
		t.Fatalf("rendered status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestEditorControlsRouteThroughDefinition(t *testing.T) {
	mux := testMux(nil)
	body := `{"componentId":"text-main","trait":"content","control":"content","value":"Changed through registry"}`
	recorder := request(t, mux, http.MethodPost, "/api/draft-card/control-change", body)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Changed through registry") {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	legacy := `{"componentId":"text-main","trait":"style","control":"fontSizePX","value":20}`
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/control-change", legacy)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "not supported") {
		t.Fatalf("legacy status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	invalid := `{"componentId":"text-main","trait":"style","control":"font_size_px","value":999}`
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/control-change", invalid)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "at most 72") {
		t.Fatalf("invalid status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	unknown := `{"componentId":"text-main","trait":"layout","control":"position","value":{"x":20,"y":30,"extra":true}}`
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/control-change", unknown)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "unknown field") {
		t.Fatalf("unknown value status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	nullValue := `{"componentId":"text-main","trait":"layout","control":"x","value":null}`
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/control-change", nullValue)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "integer") {
		t.Fatalf("null value status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAddComponentUsesRegisteredDefaultsAndStrictConfig(t *testing.T) {
	mux := testMux(nil)
	recorder := request(t, mux, http.MethodPost, "/api/draft-card/components", `{"componentKind":"text_input","config":{}}`)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"component_kind":"text_input"`) {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/components", `{"componentKind":"text_input","config":{"inputType":"text"}}`)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "field is not allowed") {
		t.Fatalf("alias status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/components", `{"componentKind":"text_input","config":null}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("null status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAIGenerationUsesErasedEnvelopeAndCapabilities(t *testing.T) {
	client := &fakeChatClient{responses: []string{`{"component_kind":"text","description":" Generated text ","config":{"content":"Hello"}}`}}
	mux := testMux(client)
	recorder := request(t, mux, http.MethodPost, "/api/draft-card/configs/text", `{"instruction":"write a greeting"}`)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"component_kind":"text"`) || !strings.Contains(recorder.Body.String(), `"content":"Hello"`) {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if len(client.calls) != 1 || !strings.Contains(joinedMessages(client.calls[0]), "component_kind") {
		t.Fatalf("calls = %#v", client.calls)
	}
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/configs/shape", `{"instruction":"make a circle"}`)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("shape AI status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyGeneratedConfigIsStrict(t *testing.T) {
	mux := testMux(nil)
	valid := `{"generated_config":{"component_kind":"text","description":"New copy","config":{"content":"Canonical"}}}`
	recorder := request(t, mux, http.MethodPost, "/api/draft-card/apply-config", valid)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Canonical") {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	invalid := `{"generated_config":{"component_kind":"text","description":"Bad","config":{"align":"CENTER"}}}`
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/apply-config", invalid)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_option") {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	legacy := `{"generated_config":{"componentKind":"text","description":"Bad","config":{}}}`
	recorder = request(t, mux, http.MethodPost, "/api/draft-card/apply-config", legacy)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("legacy status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGameSessionRendersWithInjectedRegistry(t *testing.T) {
	recorder := request(t, testMux(nil), http.MethodGet, "/api/game/session", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, marker := range []string{
		`"component_kind":"card"`,
		`"preview_html"`,
		`src=\"/assets/card-backgrounds/rusted-cell-door.webp\"`,
		`background-color: #101713`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("body missing %q: %s", marker, body)
		}
	}
	if strings.Contains(body, `data-component-kind=\"shape\"`) || strings.Contains(body, `data-component-kind=\"image\"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestUnknownRoutesAndMethods(t *testing.T) {
	mux := testMux(nil)
	if recorder := request(t, mux, http.MethodGet, "/missing", ""); recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d", recorder.Code)
	}
	if recorder := request(t, mux, http.MethodPost, "/api/draft-card", `{}`); recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d", recorder.Code)
	}
}

func testMux(client *fakeChatClient) *http.ServeMux {
	mux := http.NewServeMux()
	Register(mux, Dependencies{Patch: client, PatchModel: "test", Registry: catalog.MustNew()})
	return mux
}

func request(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

type fakeChatClient struct {
	responses []string
	calls     [][]ollama.ChatMessage
}

func (f *fakeChatClient) Chat(_ context.Context, _ string, messages []ollama.ChatMessage) (string, error) {
	f.calls = append(f.calls, append([]ollama.ChatMessage(nil), messages...))
	if len(f.responses) == 0 {
		return "", nil
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func joinedMessages(messages []ollama.ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		parts = append(parts, message.Content)
	}
	return strings.Join(parts, "\n")
}
