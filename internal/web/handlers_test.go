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

func TestPageRendersInteractiveStageWorkflow(t *testing.T) {
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
		`id="living-card-stage"`,
		`id="card-workspace"`,
		`data-game-stage`,
		`id="game-world-card"`,
		`id="game-prev-card"`,
		`id="game-next-card"`,
		`id="game-collect-card"`,
		`id="game-status"`,
		`id="game-library-list"`,
		`id="game-library-count"`,
		`id="stage-overlay-root"`,
		`id="stage-edge-controls"`,
		`id="stage-edge-controls-top"`,
		`id="stage-edge-controls-left"`,
		`id="stage-edge-controls-right"`,
		`id="stage-edge-controls-bottom"`,
		`id="stage-edge-controls-status"`,
		`id="reset-draft-btn"`,
		`id="designer-overlay"`,
		`id="fragment-target"`,
		`value="background"`,
		`value="border"`,
		`value="textarea"`,
		`value="image"`,
		`id="generate-fragment-btn"`,
		`id="update-fragment-btn"`,
		`id="design-library-list"`,
		`id="add-textarea-component-btn"`,
		`id="add-shape-component-btn"`,
		`id="add-image-component-input"`,
		`id="fragment-preview"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("page missing %s:\n%s", marker, body)
		}
	}
	for _, marker := range []string{
		`id="app-header"`,
		`id="designer-toggle-btn"`,
		`<aside`,
		`lg:grid-cols`,
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

func TestInteractiveDraftCardResourceReturnsGameStateAndPreviewHTML(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/draft-card/interactive", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Document         cardcomponent.Document      `json:"document"`
		GameState        GameState                   `json:"gameState"`
		PreviewHTML      string                      `json:"preview_html"`
		AvailableTargets []string                    `json:"availableTargets"`
		Library          []cardcomponent.LibraryItem `json:"library"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.Document.CardID != cardcomponent.DefaultCardID {
		t.Fatalf("document = %#v", payload.Document)
	}
	if payload.GameState.Level != 1 || payload.GameState.XP != 0 || payload.GameState.TapCount != 0 {
		t.Fatalf("gameState = %#v", payload.GameState)
	}
	if !hasString(payload.AvailableTargets, "background") || !hasString(payload.AvailableTargets, "border") {
		t.Fatalf("availableTargets = %#v", payload.AvailableTargets)
	}
	if len(payload.Library) == 0 {
		t.Fatal("library should include seeded presets")
	}
	if !strings.Contains(payload.PreviewHTML, `id="draft-card-preview"`) {
		t.Fatalf("preview_html did not include rendered card: %s", payload.PreviewHTML)
	}
}

func TestGameSessionStartsWithEmptyLibraryAndScriptedDeck(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/game/session", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload GameSessionSnapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if len(payload.Library) != 0 {
		t.Fatalf("library = %#v, want empty", payload.Library)
	}
	if len(payload.WorldDeck) < 5 {
		t.Fatalf("worldDeck length = %d, want scripted deck", len(payload.WorldDeck))
	}
	if payload.ActiveWorldCard.ID != "rusted-cell-door" || !strings.Contains(payload.ActiveWorldCard.PreviewHTML, `game-world-rusted-cell-door`) {
		t.Fatalf("activeWorldCard = %#v", payload.ActiveWorldCard)
	}
	if payload.SolvedFlags["doorUnlocked"] {
		t.Fatalf("solvedFlags = %#v, want locked door", payload.SolvedFlags)
	}
}

func TestGameCycleCollectAndUnlockDoor(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)

	for index := 0; index < 2; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/game/cycle", strings.NewReader(`{"direction":"next"}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("cycle %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	collectRecorder := httptest.NewRecorder()
	collectRequest := httptest.NewRequest(http.MethodPost, "/api/game/collect", strings.NewReader(`{"cardId":"bent-iron-key"}`))
	collectRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(collectRecorder, collectRequest)
	if collectRecorder.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200 body=%s", collectRecorder.Code, collectRecorder.Body.String())
	}
	var collected GameSessionSnapshot
	if err := json.Unmarshal(collectRecorder.Body.Bytes(), &collected); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, collectRecorder.Body.String())
	}
	if len(collected.Library) != 1 || collected.Library[0].ID != "bent-iron-key" {
		t.Fatalf("library = %#v, want collected key", collected.Library)
	}

	wrongRecorder := httptest.NewRecorder()
	wrongRequest := httptest.NewRequest(http.MethodPost, "/api/game/play-card", strings.NewReader(`{"sourceCardId":"bent-iron-key","targetCardId":"faded-photograph"}`))
	wrongRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(wrongRecorder, wrongRequest)
	if wrongRecorder.Code != http.StatusOK {
		t.Fatalf("wrong play status = %d, want 200 body=%s", wrongRecorder.Code, wrongRecorder.Body.String())
	}
	var wrong GameSessionSnapshot
	if err := json.Unmarshal(wrongRecorder.Body.Bytes(), &wrong); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, wrongRecorder.Body.String())
	}
	if wrong.SolvedFlags["doorUnlocked"] {
		t.Fatalf("wrong play unlocked door: %#v", wrong.SolvedFlags)
	}

	playRecorder := httptest.NewRecorder()
	playRequest := httptest.NewRequest(http.MethodPost, "/api/game/play-card", strings.NewReader(`{"sourceCardId":"bent-iron-key","targetCardId":"rusted-cell-door"}`))
	playRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(playRecorder, playRequest)
	if playRecorder.Code != http.StatusOK {
		t.Fatalf("play status = %d, want 200 body=%s", playRecorder.Code, playRecorder.Body.String())
	}
	var unlocked GameSessionSnapshot
	if err := json.Unmarshal(playRecorder.Body.Bytes(), &unlocked); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, playRecorder.Body.String())
	}
	if !unlocked.SolvedFlags["doorUnlocked"] {
		t.Fatalf("solvedFlags = %#v, want doorUnlocked", unlocked.SolvedFlags)
	}
	if len(unlocked.Library) != 1 || unlocked.Library[0].ID != "bent-iron-key" {
		t.Fatalf("library = %#v, key should remain visible as history", unlocked.Library)
	}
	if !strings.Contains(gameCardHTML(unlocked.WorldDeck, "rusted-cell-door"), "OPEN") {
		t.Fatalf("door preview did not update: %s", gameCardHTML(unlocked.WorldDeck, "rusted-cell-door"))
	}
}

func TestGameRendersActiveAndLibraryCardsWithUniqueDOMIDs(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for index := 0; index < 2; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/game/cycle", strings.NewReader(`{"direction":"next"}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("cycle %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/game/collect", strings.NewReader(`{"cardId":"bent-iron-key"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("collect status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload GameSessionSnapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.ActiveWorldCard.ID != "bent-iron-key" || len(payload.Library) != 1 || payload.Library[0].ID != "bent-iron-key" {
		t.Fatalf("payload active/library = %#v / %#v", payload.ActiveWorldCard, payload.Library)
	}
	combined := payload.ActiveWorldCard.PreviewHTML + payload.Library[0].PreviewHTML
	for _, marker := range []string{
		`id="game-world-bent-iron-key"`,
		`id="game-world-bent-iron-key-key-title-layer"`,
		`id="game-library-0-bent-iron-key"`,
		`id="game-library-0-bent-iron-key-key-title-layer"`,
		`data-component-id="key-title"`,
	} {
		if !strings.Contains(combined, marker) {
			t.Fatalf("combined render missing %q:\n%s", marker, combined)
		}
	}
	for _, duplicated := range []string{
		`id="game-world-bent-iron-key"`,
		`id="game-library-0-bent-iron-key"`,
		`id="game-world-bent-iron-key-key-title-layer"`,
		`id="game-library-0-bent-iron-key-key-title-layer"`,
	} {
		if strings.Count(combined, duplicated) != 1 {
			t.Fatalf("render should include %q exactly once:\n%s", duplicated, combined)
		}
	}
	if strings.Contains(combined, `id="key-title-layer"`) {
		t.Fatalf("render should not include unscoped key title layer id:\n%s", combined)
	}
}

func TestGameCollectRejectsDecoyCards(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/game/collect", strings.NewReader(`{"cardId":"inventory-label"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTapDraftCardAppliesRandomFragmentsAndProgresses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		field  string
	}{
		{name: "background", target: "background", field: "background_color"},
		{name: "border", target: "border", field: "border_color"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, nil)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"`+test.target+`","zone":"`+test.target+`","x":0.2,"y":0.4}`))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				Document        cardcomponent.Document `json:"document"`
				GameState       GameState              `json:"gameState"`
				AppliedFragment json.RawMessage        `json:"appliedFragment"`
				PreviewHTML     string                 `json:"preview_html"`
				Events          []CardEvent            `json:"events"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
			}
			if payload.GameState.TapCount != 1 || payload.GameState.XP != 1 || payload.GameState.Level != 1 {
				t.Fatalf("gameState = %#v", payload.GameState)
			}
			if !hasEvent(payload.Events, "fragmentApplied", test.target) || !hasEvent(payload.Events, "xpGained", "") {
				t.Fatalf("events = %#v", payload.Events)
			}
			if !strings.Contains(string(payload.AppliedFragment), test.field) {
				t.Fatalf("appliedFragment = %s, want field %s", string(payload.AppliedFragment), test.field)
			}
			if !strings.Contains(payload.PreviewHTML, `id="draft-card-preview"`) {
				t.Fatalf("preview_html did not include rendered card: %s", payload.PreviewHTML)
			}
		})
	}
}

func TestTapDraftCardReturnsLevelUpEvent(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	var payload struct {
		GameState GameState   `json:"gameState"`
		Events    []CardEvent `json:"events"`
	}
	for index := 0; index < 5; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"border","zone":"border","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
		payload = struct {
			GameState GameState   `json:"gameState"`
			Events    []CardEvent `json:"events"`
		}{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
		}
	}
	if payload.GameState.Level != 2 || payload.GameState.XP != 5 || payload.GameState.TapCount != 5 {
		t.Fatalf("gameState = %#v", payload.GameState)
	}
	if !hasEvent(payload.Events, "levelUp", "") {
		t.Fatalf("events = %#v, want levelUp", payload.Events)
	}
}

func TestTapDraftCardUnlocksSimpleControlsAtLevelFive(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	var payload struct {
		GameState GameState   `json:"gameState"`
		Events    []CardEvent `json:"events"`
	}
	for index := 0; index < 20; index++ {
		target := "border"
		x := "0.05"
		if index >= 12 {
			target = "textarea"
			x = "0.5"
		}
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"`+target+`","zone":"`+target+`","x":`+x+`,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
		payload = struct {
			GameState GameState   `json:"gameState"`
			Events    []CardEvent `json:"events"`
		}{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
		}
	}
	if payload.GameState.Level != 5 || payload.GameState.XP != 20 || payload.GameState.TapCount != 20 {
		t.Fatalf("gameState = %#v", payload.GameState)
	}
	if !hasEvent(payload.Events, "modeUnlocked", "background") || !hasEvent(payload.Events, "modeUnlocked", "border") {
		t.Fatalf("events = %#v, want background and border mode unlocks", payload.Events)
	}
	if !hasString(payload.GameState.UnlockedModes, "simpleControls") {
		t.Fatalf("unlocked modes = %#v, want simpleControls", payload.GameState.UnlockedModes)
	}
	if !hasString(payload.GameState.TargetProgress["background"].UnlockedModes, "simpleControls") {
		t.Fatalf("background progress = %#v, want simpleControls", payload.GameState.TargetProgress["background"])
	}
	if !hasString(payload.GameState.TargetProgress["border"].UnlockedModes, "simpleControls") {
		t.Fatalf("border progress = %#v, want simpleControls", payload.GameState.TargetProgress["border"])
	}
}

func TestTapDraftCardHandlesLockedAndInvalidTargets(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	lockedRecorder := httptest.NewRecorder()
	lockedRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"textarea","zone":"textarea","x":0.5,"y":0.5}`))
	lockedRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(lockedRecorder, lockedRequest)
	if lockedRecorder.Code != http.StatusOK {
		t.Fatalf("locked status = %d, want 200 body=%s", lockedRecorder.Code, lockedRecorder.Body.String())
	}
	var lockedPayload struct {
		GameState GameState   `json:"gameState"`
		Events    []CardEvent `json:"events"`
	}
	if err := json.Unmarshal(lockedRecorder.Body.Bytes(), &lockedPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, lockedRecorder.Body.String())
	}
	if lockedPayload.GameState.TapCount != 0 || !hasEvent(lockedPayload.Events, "invalidAction", "textarea") {
		t.Fatalf("locked payload = %#v", lockedPayload)
	}

	invalidRecorder := httptest.NewRecorder()
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"shadow","zone":"shadow","x":0.5,"y":0.5}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400 body=%s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
}

func TestControlChangeDraftCardAppliesUnlockedColors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		color  string
		field  string
	}{
		{name: "background", target: "background", color: "#22c55e", field: `"background_color":"#22c55e"`},
		{name: "border", target: "border", color: "#f59e0b", field: `"border_color":"#f59e0b"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, nil)
			levelDraftCardToFive(t, mux)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"target":"`+test.target+`","color":"`+test.color+`"}`))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				GameState       GameState       `json:"gameState"`
				AppliedFragment json.RawMessage `json:"appliedFragment"`
				PreviewHTML     string          `json:"preview_html"`
				Events          []CardEvent     `json:"events"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
			}
			if payload.GameState.Level < 5 || payload.GameState.TapCount != 21 {
				t.Fatalf("gameState = %#v", payload.GameState)
			}
			if !strings.Contains(string(payload.AppliedFragment), test.field) {
				t.Fatalf("appliedFragment = %s, want %s", string(payload.AppliedFragment), test.field)
			}
			if !strings.Contains(payload.PreviewHTML, test.color) {
				t.Fatalf("preview_html did not include %s: %s", test.color, payload.PreviewHTML)
			}
			if !hasEvent(payload.Events, "fragmentApplied", test.target) {
				t.Fatalf("events = %#v, want fragmentApplied", payload.Events)
			}
		})
	}
}

func TestControlChangeDraftCardAppliesUnlockedGradients(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
		body   string
		field  string
		css    string
	}{
		{
			name:   "background",
			target: "background",
			body:   `{"target":"background","color":"#22c55e","secondaryColor":"#38bdf8","gradient":true,"angle":45}`,
			field:  `"background_color":"#22c55e"`,
			css:    `linear-gradient(45deg, #22c55e 0%, #38bdf8 100%)`,
		},
		{
			name:   "border",
			target: "border",
			body:   `{"target":"border","color":"#f59e0b","secondaryColor":"#a78bfa","gradient":true,"angle":210}`,
			field:  `"border_color":"#f59e0b"`,
			css:    `border-image: linear-gradient(210deg, #f59e0b 0%, #a78bfa 100%) 1`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, nil)
			levelDraftCardToFive(t, mux)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				AppliedFragment json.RawMessage `json:"appliedFragment"`
				PreviewHTML     string          `json:"preview_html"`
				Events          []CardEvent     `json:"events"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
			}
			if !strings.Contains(string(payload.AppliedFragment), test.field) || !strings.Contains(string(payload.AppliedFragment), test.css) {
				t.Fatalf("appliedFragment = %s, want field %s and css %s", string(payload.AppliedFragment), test.field, test.css)
			}
			if !strings.Contains(payload.PreviewHTML, test.css) {
				t.Fatalf("preview_html did not include %s: %s", test.css, payload.PreviewHTML)
			}
			if !hasEvent(payload.Events, "fragmentApplied", test.target) {
				t.Fatalf("events = %#v, want fragmentApplied", payload.Events)
			}
		})
	}
}

func TestControlChangeDraftCardRequiresUnlockedMode(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"target":"background","color":"#22c55e"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		GameState GameState   `json:"gameState"`
		Events    []CardEvent `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.GameState.TapCount != 0 || !hasEvent(payload.Events, "invalidAction", "background") {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestControlChangeDraftCardRejectsInvalidColor(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	levelDraftCardToFive(t, mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"target":"background","color":"url(https://example.test)"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestControlChangeDraftCardRejectsInvalidGradient(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	levelDraftCardToFive(t, mux)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"target":"background","color":"#22c55e","gradient":true}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestInteractLongPressAwardsXPAndOpensOverlayOnce(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for index := 0; index < 6; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"border","zone":"border","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/interact", strings.NewReader(`{"componentId":"card-root","trait":"border","interaction":"longPress","x":0.05,"y":0.5}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		GameState GameState         `json:"gameState"`
		Events    []CardEvent       `json:"events"`
		Overlay   *ComponentOverlay `json:"overlay"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.GameState.TotalXP != 8 || !payload.GameState.ComponentProgress["card-root"].OverlayOpened {
		t.Fatalf("gameState = %#v", payload.GameState)
	}
	if payload.Overlay == nil || payload.Overlay.ComponentID != "card-root" {
		t.Fatalf("overlay = %#v", payload.Overlay)
	}
	if !hasEvent(payload.Events, "overlayOpened", "") {
		t.Fatalf("events = %#v, want overlayOpened", payload.Events)
	}
}

func TestComponentKeepsRandomizingAfterLevelFive(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for index := 0; index < 12; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"border","zone":"border","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/interact", strings.NewReader(`{"componentId":"card-root","trait":"background","interaction":"shortTap","x":0.5,"y":0.5}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		GameState       GameState       `json:"gameState"`
		AppliedFragment json.RawMessage `json:"appliedFragment"`
		Events          []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.GameState.TotalXP != 13 || !payload.GameState.ComponentProgress["card-root"].RandomTapEnabled {
		t.Fatalf("gameState = %#v", payload.GameState)
	}
	if len(payload.AppliedFragment) == 0 {
		t.Fatal("appliedFragment should be present while randomizing is enabled")
	}
	if !hasEvent(payload.Events, "xpGained", "") {
		t.Fatalf("events = %#v, want xpGained", payload.Events)
	}
}

func TestPreventRandomizingCheckboxDisablesTapRandomization(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for index := 0; index < 6; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"border","zone":"border","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	controlRecorder := httptest.NewRecorder()
	controlRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentId":"card-root","control":"preventRandomizing","value":true}`))
	controlRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(controlRecorder, controlRequest)
	if controlRecorder.Code != http.StatusOK {
		t.Fatalf("control status = %d, want 200 body=%s", controlRecorder.Code, controlRecorder.Body.String())
	}
	var controlPayload struct {
		GameState GameState         `json:"gameState"`
		Overlay   *ComponentOverlay `json:"overlay"`
		Events    []CardEvent       `json:"events"`
	}
	if err := json.Unmarshal(controlRecorder.Body.Bytes(), &controlPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, controlRecorder.Body.String())
	}
	if !controlPayload.GameState.ComponentProgress["card-root"].PreventRandomizing || controlPayload.GameState.ComponentProgress["card-root"].RandomTapEnabled {
		t.Fatalf("gameState = %#v", controlPayload.GameState)
	}
	if controlPayload.Overlay == nil || !hasControl(controlPayload.Overlay.Controls, "preventRandomizing") {
		t.Fatalf("overlay = %#v, want preventRandomizing control", controlPayload.Overlay)
	}
	if !hasEvent(controlPayload.Events, "xpGained", "") {
		t.Fatalf("events = %#v, want xpGained", controlPayload.Events)
	}

	tapRecorder := httptest.NewRecorder()
	tapRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/interact", strings.NewReader(`{"componentId":"card-root","trait":"background","interaction":"shortTap","x":0.5,"y":0.5}`))
	tapRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(tapRecorder, tapRequest)
	if tapRecorder.Code != http.StatusOK {
		t.Fatalf("tap status = %d, want 200 body=%s", tapRecorder.Code, tapRecorder.Body.String())
	}
	if !strings.Contains(tapRecorder.Body.String(), `"events":[]`) {
		t.Fatalf("events should encode as [] instead of null: %s", tapRecorder.Body.String())
	}
	var tapPayload struct {
		GameState       GameState       `json:"gameState"`
		AppliedFragment json.RawMessage `json:"appliedFragment"`
		Events          []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(tapRecorder.Body.Bytes(), &tapPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, tapRecorder.Body.String())
	}
	if tapPayload.GameState.TotalXP != 7 || len(tapPayload.AppliedFragment) != 0 || len(tapPayload.Events) != 0 {
		t.Fatalf("tap payload = %#v applied=%s events=%#v", tapPayload.GameState, string(tapPayload.AppliedFragment), tapPayload.Events)
	}
}

func TestGenericControlChangeAppliesRootPaddingAndGrantsXP(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	levelDraftCardToFive(t, mux)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentId":"card-root","trait":"padding","control":"paddingPx","value":32}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		GameState       GameState       `json:"gameState"`
		AppliedFragment json.RawMessage `json:"appliedFragment"`
		PreviewHTML     string          `json:"preview_html"`
		Events          []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.GameState.TotalXP != 21 || !strings.Contains(payload.PreviewHTML, `padding: 32px`) {
		t.Fatalf("payload = %#v preview=%s", payload.GameState, payload.PreviewHTML)
	}
	if !strings.Contains(string(payload.AppliedFragment), `"target":"card"`) {
		t.Fatalf("appliedFragment = %s", string(payload.AppliedFragment))
	}
	if !hasEvent(payload.Events, "xpGained", "") {
		t.Fatalf("events = %#v, want xpGained", payload.Events)
	}
}

func TestGenericControlChangeMovesTextareaAndShape(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	levelDraftCardToFive(t, mux)

	textRecorder := httptest.NewRecorder()
	textRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentId":"textarea-main","trait":"position","control":"position","value":{"x":42,"y":57}}`))
	textRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(textRecorder, textRequest)
	if textRecorder.Code != http.StatusOK {
		t.Fatalf("textarea status = %d, want 200 body=%s", textRecorder.Code, textRecorder.Body.String())
	}
	var textPayload struct {
		PreviewHTML string            `json:"preview_html"`
		Events      []CardEvent       `json:"events"`
		Overlay     *ComponentOverlay `json:"overlay"`
	}
	if err := json.Unmarshal(textRecorder.Body.Bytes(), &textPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, textRecorder.Body.String())
	}
	for _, marker := range []string{`data-component-id="textarea-main"`, `left: 42%`, `top: 57%`} {
		if !strings.Contains(textPayload.PreviewHTML, marker) {
			t.Fatalf("textarea preview missing %q: %s", marker, textPayload.PreviewHTML)
		}
	}
	if textPayload.Overlay == nil || !hasControl(textPayload.Overlay.Controls, "x") || !hasControl(textPayload.Overlay.Controls, "y") {
		t.Fatalf("textarea overlay = %#v, want x/y controls", textPayload.Overlay)
	}
	if !hasEvent(textPayload.Events, "xpGained", "") {
		t.Fatalf("textarea events = %#v, want xpGained", textPayload.Events)
	}

	levelDraftCardToSeven(t, mux)
	levelShapeComponentToThree(t, mux)

	shapeRecorder := httptest.NewRecorder()
	shapeRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentId":"shape-1","trait":"position","control":"position","value":{"x":64,"y":22}}`))
	shapeRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(shapeRecorder, shapeRequest)
	if shapeRecorder.Code != http.StatusOK {
		t.Fatalf("shape status = %d, want 200 body=%s", shapeRecorder.Code, shapeRecorder.Body.String())
	}
	var shapePayload struct {
		PreviewHTML string            `json:"preview_html"`
		Events      []CardEvent       `json:"events"`
		Overlay     *ComponentOverlay `json:"overlay"`
	}
	if err := json.Unmarshal(shapeRecorder.Body.Bytes(), &shapePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, shapeRecorder.Body.String())
	}
	for _, marker := range []string{`data-component-id="shape-1"`, `left: 64%`, `top: 22%`} {
		if !strings.Contains(shapePayload.PreviewHTML, marker) {
			t.Fatalf("shape preview missing %q: %s", marker, shapePayload.PreviewHTML)
		}
	}
	if shapePayload.Overlay == nil || !hasControl(shapePayload.Overlay.Controls, "x") || !hasControl(shapePayload.Overlay.Controls, "y") {
		t.Fatalf("shape overlay = %#v, want x/y controls", shapePayload.Overlay)
	}
	if !hasEvent(shapePayload.Events, "xpGained", "") {
		t.Fatalf("shape events = %#v, want xpGained", shapePayload.Events)
	}
}

func TestAddDraftComponentsSupportsMultipleImagesAndControlByID(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	for index := 0; index < 2; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/components", strings.NewReader(`{"componentType":"image"}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("add image %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	moveRecorder := httptest.NewRecorder()
	moveRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentId":"image-2","trait":"position","control":"position","value":{"x":72,"y":22}}`))
	moveRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(moveRecorder, moveRequest)
	if moveRecorder.Code != http.StatusOK {
		t.Fatalf("move status = %d, want 200 body=%s", moveRecorder.Code, moveRecorder.Body.String())
	}
	var payload struct {
		Document    cardcomponent.Document `json:"document"`
		PreviewHTML string                 `json:"preview_html"`
	}
	if err := json.Unmarshal(moveRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, moveRecorder.Body.String())
	}
	imageOne := findTestNode(payload.Document.Root, "image-1")
	imageTwo := findTestNode(payload.Document.Root, "image-2")
	if imageOne == nil || imageTwo == nil {
		t.Fatalf("document should contain image-1 and image-2: %#v", payload.Document.Root.Children)
	}
	if strings.Contains(string(imageOne.Fragment), `"x":72`) {
		t.Fatalf("image-1 was mutated instead of image-2: %s", string(imageOne.Fragment))
	}
	if !strings.Contains(string(imageTwo.Fragment), `"x":72`) || !strings.Contains(string(imageTwo.Fragment), `"y":22`) {
		t.Fatalf("image-2 fragment was not moved: %s", string(imageTwo.Fragment))
	}
	for _, marker := range []string{`data-component-id="image-1"`, `data-component-id="image-2"`, `left: 72%`, `top: 22%`} {
		if !strings.Contains(payload.PreviewHTML, marker) {
			t.Fatalf("preview missing %q: %s", marker, payload.PreviewHTML)
		}
	}
}

func TestAddDraftImageRejectsUnsafeSource(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := `{"componentType":"image","fragment":{"src":"data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=","alt":"svg","x":50,"y":50,"width":20,"height":20,"border_color":"#ffffff"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/components", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestShapeUnlockCreatesInitialShapeAndCanRandomize(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	levelDraftCardToSeven(t, mux)

	interactiveRecorder := httptest.NewRecorder()
	mux.ServeHTTP(interactiveRecorder, httptest.NewRequest(http.MethodGet, "/api/draft-card/interactive", nil))
	if interactiveRecorder.Code != http.StatusOK {
		t.Fatalf("interactive status = %d, want 200 body=%s", interactiveRecorder.Code, interactiveRecorder.Body.String())
	}
	var interactive struct {
		GameState           GameState             `json:"gameState"`
		PreviewHTML         string                `json:"preview_html"`
		AvailableComponents []ComponentDescriptor `json:"availableComponents"`
	}
	if err := json.Unmarshal(interactiveRecorder.Body.Bytes(), &interactive); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, interactiveRecorder.Body.String())
	}
	if interactive.GameState.GlobalLevel < 7 || !strings.Contains(interactive.PreviewHTML, `data-component-id="shape-1"`) {
		t.Fatalf("interactive = %#v preview=%s", interactive.GameState, interactive.PreviewHTML)
	}
	if !hasComponent(interactive.AvailableComponents, "shape-1") {
		t.Fatalf("availableComponents = %#v, want shape-1", interactive.AvailableComponents)
	}

	randomizeRecorder := httptest.NewRecorder()
	randomizeRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/randomize-component", strings.NewReader(`{"componentId":"shape-1","scope":"unlockedTraits"}`))
	randomizeRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(randomizeRecorder, randomizeRequest)
	if randomizeRecorder.Code != http.StatusOK {
		t.Fatalf("randomize status = %d, want 200 body=%s", randomizeRecorder.Code, randomizeRecorder.Body.String())
	}
	var randomizePayload struct {
		GameState       GameState       `json:"gameState"`
		AppliedFragment json.RawMessage `json:"appliedFragment"`
		Events          []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(randomizeRecorder.Body.Bytes(), &randomizePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, randomizeRecorder.Body.String())
	}
	if !strings.Contains(string(randomizePayload.AppliedFragment), `"target":"shape"`) {
		t.Fatalf("appliedFragment = %s", string(randomizePayload.AppliedFragment))
	}
	if randomizePayload.GameState.ComponentProgress["shape-1"].XP != 1 || !hasEvent(randomizePayload.Events, "xpGained", "") {
		t.Fatalf("randomize payload = %#v events=%#v", randomizePayload.GameState.ComponentProgress["shape-1"], randomizePayload.Events)
	}
}

func TestResetDraftCardRestoresGameProgress(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	tapRecorder := httptest.NewRecorder()
	tapRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"background","zone":"background","x":0.5,"y":0.5}`))
	tapRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(tapRecorder, tapRequest)
	if tapRecorder.Code != http.StatusOK {
		t.Fatalf("tap status = %d, want 200 body=%s", tapRecorder.Code, tapRecorder.Body.String())
	}

	resetRecorder := httptest.NewRecorder()
	mux.ServeHTTP(resetRecorder, httptest.NewRequest(http.MethodPost, "/api/draft-card/reset", nil))
	if resetRecorder.Code != http.StatusOK {
		t.Fatalf("reset status = %d, want 200 body=%s", resetRecorder.Code, resetRecorder.Body.String())
	}

	interactiveRecorder := httptest.NewRecorder()
	mux.ServeHTTP(interactiveRecorder, httptest.NewRequest(http.MethodGet, "/api/draft-card/interactive", nil))
	if interactiveRecorder.Code != http.StatusOK {
		t.Fatalf("interactive status = %d, want 200 body=%s", interactiveRecorder.Code, interactiveRecorder.Body.String())
	}
	var payload struct {
		GameState GameState `json:"gameState"`
	}
	if err := json.Unmarshal(interactiveRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, interactiveRecorder.Body.String())
	}
	if payload.GameState.TapCount != 0 || payload.GameState.XP != 0 || payload.GameState.Level != 1 {
		t.Fatalf("gameState = %#v", payload.GameState)
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

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func hasEvent(events []CardEvent, eventType, target string) bool {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		if target == "" || event.Target == target {
			return true
		}
	}
	return false
}

func hasComponent(components []ComponentDescriptor, componentID string) bool {
	for _, component := range components {
		if component.ComponentID == componentID {
			return true
		}
	}
	return false
}

func hasControl(controls []ControlDescriptor, control string) bool {
	for _, candidate := range controls {
		if candidate.Control == control {
			return true
		}
	}
	return false
}

func gameCardHTML(cards []RenderedGameCard, cardID string) string {
	for _, card := range cards {
		if card.ID == cardID {
			return card.PreviewHTML
		}
	}
	return ""
}

func findTestNode(node cardcomponent.Node, id string) *cardcomponent.Node {
	if node.ID == id {
		return &node
	}
	for _, child := range node.Children {
		if match := findTestNode(child, id); match != nil {
			return match
		}
	}
	return nil
}

func levelDraftCardToFive(t *testing.T, mux *http.ServeMux) {
	t.Helper()

	for index := 0; index < 12; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"border","zone":"border","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}
	for index := 0; index < 8; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"target":"textarea","zone":"textarea","x":0.5,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("textarea tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}
}

func levelDraftCardToSeven(t *testing.T, mux *http.ServeMux) {
	t.Helper()

	levelDraftCardToFive(t, mux)
	for index := 0; index < 9; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/interact", strings.NewReader(`{"componentId":"card-root","trait":"border","interaction":"longPress","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("long press %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}
}

func levelShapeComponentToThree(t *testing.T, mux *http.ServeMux) {
	t.Helper()

	for index := 0; index < 6; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/interact", strings.NewReader(`{"componentId":"shape-1","trait":"geometry","interaction":"shortTap","x":0.5,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("shape tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}
}
