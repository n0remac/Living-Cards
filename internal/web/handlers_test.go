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

func TestGameEndpointsReturnRevisionedResults(t *testing.T) {
	mux := testMux(nil)
	read := request(t, mux, http.MethodGet, "/api/game/session", "")
	if read.Code != http.StatusOK {
		t.Fatalf("read status = %d body=%s", read.Code, read.Body.String())
	}
	var initial GameResultResponse
	if err := json.Unmarshal(read.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.Revision != 0 || len(initial.Events) != 0 || initial.Snapshot.ActiveWorldCardID == "" {
		t.Fatalf("initial result = %#v", initial)
	}

	cycled := request(t, mux, http.MethodPost, "/api/game/cycle", `{"direction":"next"}`)
	if cycled.Code != http.StatusOK {
		t.Fatalf("cycle status = %d body=%s", cycled.Code, cycled.Body.String())
	}
	var result GameResultResponse
	if err := json.Unmarshal(cycled.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Revision != 1 || len(result.Events) == 0 || result.Events[0].Type != "cardCycled" {
		t.Fatalf("cycle result = %#v", result)
	}
}

func TestInvalidGameRequestsAreStrictAndDoNotAdvanceRevision(t *testing.T) {
	mux := testMux(nil)
	for _, body := range []string{
		`{"direction":"next","legacy":true}`,
		`{"direction":"next"} {"direction":"next"}`,
		`{"direction":`,
	} {
		recorder := request(t, mux, http.MethodPost, "/api/game/cycle", body)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %q status = %d response=%s", body, recorder.Code, recorder.Body.String())
		}
	}
	invalidCommand := request(t, mux, http.MethodPost, "/api/game/collect", `{"cardId":"missing"}`)
	if invalidCommand.Code != http.StatusBadRequest {
		t.Fatalf("invalid command status = %d body=%s", invalidCommand.Code, invalidCommand.Body.String())
	}
	invalidEmpty := request(t, mux, http.MethodPost, "/api/game/reset", `{"legacy":true}`)
	if invalidEmpty.Code != http.StatusBadRequest {
		t.Fatalf("reset body status = %d body=%s", invalidEmpty.Code, invalidEmpty.Body.String())
	}

	read := request(t, mux, http.MethodGet, "/api/game/session", "")
	var result GameResultResponse
	if err := json.Unmarshal(read.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Revision != 0 {
		t.Fatalf("revision after invalid requests = %d", result.Revision)
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
