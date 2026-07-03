package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
	"github.com/n0remac/Living-Card/internal/ollama"
)

func TestPageRendersDesignerOnlyWorkflow(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	for _, marker := range []string{
		`src="/assets/app.js"`,
		`id="app-header"`,
		`id="card-workspace"`,
		`id="draft-card-preview"`,
		`id="fragment-target"`,
		`value="background"`,
		`value="border"`,
		`value="textarea"`,
		`id="generate-fragment-btn"`,
		`id="update-fragment-btn"`,
		`id="design-library-list"`,
		`id="fragment-preview"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("page missing %s:\n%s", marker, body)
		}
	}
	for _, marker := range []string{
		`value="title"`,
		`value="body"`,
		`id="card-list"`,
		`id="chat-form"`,
	} {
		if strings.Contains(body, marker) {
			t.Fatalf("page should not include removed workflow marker %s:\n%s", marker, body)
		}
	}
}

func TestUnknownPagePathReturnsNotFound(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/cards/ember", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestDraftCardResourceReturnsDefaultDocument(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/draft-card", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var document cardcomponent.Document
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if document.CardID != cardcomponent.DefaultCardID || document.Root.Type != cardcomponent.Type {
		t.Fatalf("document = %#v", document)
	}
	if len(document.Root.Children) != 3 {
		t.Fatalf("children = %#v", document.Root.Children)
	}
}

func TestRenderedDraftCardResourceReturnsDocumentAndPreviewHTML(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/draft-card/rendered", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Document    cardcomponent.Document `json:"document"`
		PreviewHTML string                 `json:"preview_html"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.Document.CardID != cardcomponent.DefaultCardID {
		t.Fatalf("document = %#v", payload.Document)
	}
	if !strings.Contains(payload.PreviewHTML, `id="draft-card-preview"`) || !strings.Contains(payload.PreviewHTML, `Start designing this card.`) {
		t.Fatalf("preview_html did not include default preview: %s", payload.PreviewHTML)
	}
}

func TestDraftFragmentRoutesGenerateFragments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target string
		raw    string
		field  string
	}{
		{
			target: "background",
			raw:    `{"target":"background","description":"Moody teal background","fragment":{"background_color":"#0f766e","css":"background: linear-gradient(135deg, #0f766e, #111827);"}}`,
			field:  `"background_color":"#0f766e"`,
		},
		{
			target: "border",
			raw:    `{"target":"border","description":"Fine white border","fragment":{"border_width_px":2,"border_radius_px":18,"border_color":"#ffffff","css":"box-shadow: 0 0 20px rgba(255,255,255,0.2);"}}`,
			field:  `"border_width_px":2`,
		},
		{
			target: "textarea",
			raw:    `{"target":"textarea","description":"Centered text","fragment":{"content":"Hello card","font_family":"system","font_size_px":28,"font_weight":700,"font_style":"normal","color":"#f8fafc","align":"center","position":"center","css":"text-align: center;"}}`,
			field:  `"content":"Hello card"`,
		},
	}
	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, &fakePatchClient{responses: []string{test.raw}})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/fragments/"+test.target, strings.NewReader(`{"instruction":"make it polished"}`))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), test.field) {
				t.Fatalf("response missing %s: %s", test.field, recorder.Body.String())
			}
		})
	}
}

func TestDraftFragmentRoutesRepairInvalidOutput(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"target":"textarea","fragment":{"content":"","font_family":"system","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":""}}`,
		`{"target":"textarea","description":"Repaired centered text","fragment":{"content":"Repaired text","font_family":"system","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"center","position":"center","css":"text-align: center;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/fragments/textarea", strings.NewReader(`{"instruction":"write a centered note"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
	if !strings.Contains(recorder.Body.String(), `"content":"Repaired text"`) {
		t.Fatalf("response did not include repaired fragment: %s", recorder.Body.String())
	}
	repairPrompt := joinedMessages(client.messages[1])
	for _, marker := range []string{
		"write a centered note",
		"description is required",
		"Invalid raw model response",
		"Working example",
	} {
		if !strings.Contains(repairPrompt, marker) {
			t.Fatalf("repair prompt missing %q:\n%s", marker, repairPrompt)
		}
	}
}

func TestDraftFragmentRouteReturnsRawRepairFailure(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"target":"textarea","description":"","fragment":{"content":"","font_family":"bad","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":""}}`,
		`{"target":"textarea","description":"","fragment":{"content":"","font_family":"bad","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":"position: absolute;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/fragments/textarea", strings.NewReader(`{"instruction":"make broken text"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 body=%s", recorder.Code, recorder.Body.String())
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
	var payload struct {
		Message     string           `json:"message"`
		RawResponse string           `json:"raw_response"`
		Issues      []fragment.Issue `json:"issues"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.RawResponse != client.responses[1] {
		t.Fatalf("raw_response = %q, want repair output %q", payload.RawResponse, client.responses[1])
	}
	if len(payload.Issues) == 0 {
		t.Fatalf("issues = %#v, want structured issues", payload.Issues)
	}
}

func TestDraftFragmentRouteIncludesUpdateContext(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"target":"background","description":"Updated background","fragment":{"background_color":"#111827","css":"background: #111827;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/fragments/background", strings.NewReader(`{"instruction":"make it darker","old_code":"{\"background_color\":\"#ffffff\",\"css\":\"\"}","component_id":"background-primary"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	prompt := joinedMessages(client.messages[0])
	for _, marker := range []string{
		"make it darker",
		"Existing fragment JSON to update",
		`"background_color":"#ffffff"`,
		"background-primary",
	} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("prompt missing %q:\n%s", marker, prompt)
		}
	}
}

func TestDraftFragmentRouteUsesServerCurrentFragmentForUpdate(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"target":"background","description":"Updated background","fragment":{"background_color":"#111827","css":"background: #111827;"}}`,
	}}
	mux := testMux(t, client)
	applyRecorder := httptest.NewRecorder()
	applyBody := applyRequestBody(t, json.RawMessage(`{
		"target":"background",
		"description":"Applied white background",
		"fragment":{"background_color":"#ffffff","css":"background: #ffffff;"}
	}`))
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-fragment", strings.NewReader(applyBody))
	applyRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("apply status = %d, want 200 body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}

	generateRecorder := httptest.NewRecorder()
	generateRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/fragments/background", strings.NewReader(`{"instruction":"make it darker","update":true}`))
	generateRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(generateRecorder, generateRequest)
	if generateRecorder.Code != http.StatusOK {
		t.Fatalf("generate status = %d, want 200 body=%s", generateRecorder.Code, generateRecorder.Body.String())
	}
	prompt := joinedMessages(client.messages[0])
	for _, marker := range []string{
		"Existing fragment JSON to update",
		`"background_color": "#ffffff"`,
		"background-primary",
	} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("prompt missing %q:\n%s", marker, prompt)
		}
	}
}

func TestDraftFragmentRouteDoesNotUseServerCurrentFragmentForGenerate(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"target":"background","description":"Updated background","fragment":{"background_color":"#111827","css":"background: #111827;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/fragments/background", strings.NewReader(`{"instruction":"make it darker"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	prompt := joinedMessages(client.messages[0])
	if strings.Contains(prompt, "Existing fragment JSON to update") {
		t.Fatalf("generate prompt should not include update context:\n%s", prompt)
	}
}

func TestApplyDraftFragmentValidatesAndRendersPreview(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := applyRequestBody(t, json.RawMessage(`{
		"target":"textarea",
		"description":"Large centered note",
		"fragment":{
			"content":"Server rendered text",
			"font_family":"system",
			"font_size_px":90,
			"font_weight":700,
			"font_style":"normal",
			"color":"#f8fafc",
			"align":"center",
			"position":"center",
			"css":"text-align: center;"
		}
	}`))
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-fragment", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Document           cardcomponent.Document `json:"document"`
		NormalizedFragment json.RawMessage        `json:"normalized_fragment"`
		PreviewHTML        string                 `json:"preview_html"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if !strings.Contains(payload.PreviewHTML, `id="draft-card-preview"`) || !strings.Contains(payload.PreviewHTML, `Server rendered text`) {
		t.Fatalf("preview_html did not include rendered card: %s", payload.PreviewHTML)
	}
	if !strings.Contains(string(payload.NormalizedFragment), `"font_size_px":72`) {
		t.Fatalf("normalized fragment did not clamp font size: %s", string(payload.NormalizedFragment))
	}
	if !strings.Contains(string(payload.Document.Root.Children[2].Fragment), `"Server rendered text"`) {
		t.Fatalf("document textarea fragment was not replaced: %s", string(payload.Document.Root.Children[2].Fragment))
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/draft-card", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var document cardcomponent.Document
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !strings.Contains(string(document.Root.Children[2].Fragment), `"Server rendered text"`) {
		t.Fatalf("server state did not retain applied fragment: %s", string(document.Root.Children[2].Fragment))
	}
}

func TestDesignLibraryRoutesUseServerState(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/draft-card/library?target=background", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("library status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var libraryPayload struct {
		Library []cardcomponent.LibraryItem `json:"library"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &libraryPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if len(libraryPayload.Library) == 0 || libraryPayload.Library[0].Target != "background" {
		t.Fatalf("library = %#v, want background presets", libraryPayload.Library)
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/draft-card/library/save-applied", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("save before apply status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}

	applyRecorder := httptest.NewRecorder()
	applyBody := applyRequestBody(t, json.RawMessage(`{
		"target":"border",
		"description":"Saved border",
		"fragment":{"border_width_px":2,"border_radius_px":16,"border_color":"#ffffff","css":"border: 2px solid #ffffff;"}
	}`))
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-fragment", strings.NewReader(applyBody))
	applyRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("apply status = %d, want 200 body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}

	saveRecorder := httptest.NewRecorder()
	mux.ServeHTTP(saveRecorder, httptest.NewRequest(http.MethodPost, "/api/draft-card/library/save-applied", nil))
	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, want 200 body=%s", saveRecorder.Code, saveRecorder.Body.String())
	}
	var savePayload struct {
		Item    cardcomponent.LibraryItem   `json:"item"`
		Library []cardcomponent.LibraryItem `json:"library"`
	}
	if err := json.Unmarshal(saveRecorder.Body.Bytes(), &savePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, saveRecorder.Body.String())
	}
	if !savePayload.Item.Saved || savePayload.Item.Target != "border" {
		t.Fatalf("saved item = %#v", savePayload.Item)
	}

	applyLibraryRecorder := httptest.NewRecorder()
	applyLibraryRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/library/apply", strings.NewReader(`{"item_id":"seed-background-night-sky"}`))
	applyLibraryRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(applyLibraryRecorder, applyLibraryRequest)
	if applyLibraryRecorder.Code != http.StatusOK {
		t.Fatalf("apply library status = %d, want 200 body=%s", applyLibraryRecorder.Code, applyLibraryRecorder.Body.String())
	}
	if !strings.Contains(applyLibraryRecorder.Body.String(), `"background_color":"#0f172a"`) {
		t.Fatalf("apply library response did not include preset fragment: %s", applyLibraryRecorder.Body.String())
	}
}

func TestApplyDraftFragmentReturnsStructuredValidationIssues(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := applyRequestBody(t, json.RawMessage(`{
		"target":"background",
		"description":"Unsafe background",
		"fragment":{
			"background_color":"#111827",
			"css":"background-image: url(https://example.test/image.png);"
		}
	}`))
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-fragment", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Message     string           `json:"message"`
		RawResponse string           `json:"raw_response"`
		Issues      []fragment.Issue `json:"issues"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.RawResponse == "" || len(payload.Issues) == 0 {
		t.Fatalf("payload = %#v, want raw response and issues", payload)
	}
	if payload.Issues[0].Path != "fragment.css" {
		t.Fatalf("issue path = %q, want fragment.css", payload.Issues[0].Path)
	}
}

func TestResetDraftCardRestoresServerState(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := applyRequestBody(t, json.RawMessage(`{
		"target":"textarea",
		"description":"Text",
		"fragment":{
			"content":"Missing node",
			"font_family":"system",
			"font_size_px":16,
			"font_weight":400,
			"font_style":"normal",
			"color":"#cbd5e1",
			"align":"left",
			"position":"center",
			"css":""
		}
	}`))
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-fragment", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("apply status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/draft-card/reset", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Document    cardcomponent.Document `json:"document"`
		PreviewHTML string                 `json:"preview_html"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if !strings.Contains(string(payload.Document.Root.Children[2].Fragment), `Start designing this card.`) {
		t.Fatalf("reset did not restore default textarea: %s", string(payload.Document.Root.Children[2].Fragment))
	}
	if !strings.Contains(payload.PreviewHTML, `Start designing this card.`) {
		t.Fatalf("reset preview did not render default text: %s", payload.PreviewHTML)
	}
}

func TestApplyDraftFragmentRejectsBadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		body   string
		status int
	}{
		{name: "wrong method", method: http.MethodGet, body: ``, status: http.StatusMethodNotAllowed},
		{name: "malformed body", method: http.MethodPost, body: `{`, status: http.StatusBadRequest},
		{name: "unknown request field", method: http.MethodPost, body: `{"generated_fragment":{},"extra":true}`, status: http.StatusBadRequest},
		{name: "unknown target", method: http.MethodPost, body: applyRequestBody(t, json.RawMessage(`{"target":"shadow","description":"Shadow","fragment":{}}`)), status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, nil)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, "/api/draft-card/apply-fragment", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d body=%s", recorder.Code, test.status, recorder.Body.String())
			}
		})
	}
}

func TestDraftFragmentRouteRejectsBadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "empty instruction", method: http.MethodPost, path: "/api/draft-card/fragments/background", body: `{"instruction":" "}`, status: http.StatusBadRequest},
		{name: "malformed body", method: http.MethodPost, path: "/api/draft-card/fragments/background", body: `{`, status: http.StatusBadRequest},
		{name: "unknown target", method: http.MethodPost, path: "/api/draft-card/fragments/shadow", body: `{"instruction":"make it"}`, status: http.StatusNotFound},
		{name: "removed title target", method: http.MethodPost, path: "/api/draft-card/fragments/title", body: `{"instruction":"make it"}`, status: http.StatusNotFound},
		{name: "removed body target", method: http.MethodPost, path: "/api/draft-card/fragments/body", body: `{"instruction":"make it"}`, status: http.StatusNotFound},
		{name: "wrong method", method: http.MethodGet, path: "/api/draft-card/fragments/background", body: ``, status: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, &fakePatchClient{responses: []string{`{}`}})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d body=%s", recorder.Code, test.status, recorder.Body.String())
			}
		})
	}
}

func TestRemovedLegacyRoutesReturnNotFound(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for _, path := range []string{
		"/api/cards",
		"/api/cards/ember",
		"/api/cards/ember/canvas",
		"/api/cards/ember/memories",
		"/api/cards/ember/components/chat-form/actions/send",
		"/api/users/tester/profile",
		"/api/components/chat-form/patch-proposals",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestFrontendAssetsServed(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for _, asset := range []string{"/assets/app.js", "/assets/app.js.map"} {
		t.Run(asset, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, asset, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			if strings.TrimSpace(recorder.Body.String()) == "" {
				t.Fatal("asset body is empty")
			}
		})
	}
}

func testMux(t *testing.T, client *fakePatchClient) *http.ServeMux {
	t.Helper()

	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Patch:      client,
		PatchModel: "test-model",
	})
	return mux
}

type fakePatchClient struct {
	responses []string
	messages  [][]ollama.ChatMessage
	calls     int
}

func (c *fakePatchClient) Chat(_ context.Context, model string, messages []ollama.ChatMessage) (string, error) {
	if model != "test-model" {
		return "", nil
	}
	c.messages = append(c.messages, append([]ollama.ChatMessage(nil), messages...))
	index := c.calls
	c.calls++
	if index >= len(c.responses) {
		return `{}`, nil
	}
	return c.responses[index], nil
}

func joinedMessages(messages []ollama.ChatMessage) string {
	var parts []string
	for _, message := range messages {
		parts = append(parts, message.Role+": "+message.Content)
	}
	return strings.Join(parts, "\n\n")
}

func applyRequestBody(t *testing.T, generated json.RawMessage) string {
	t.Helper()

	raw, err := json.Marshal(struct {
		GeneratedFragment json.RawMessage `json:"generated_fragment"`
	}{
		GeneratedFragment: generated,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(raw)
}
