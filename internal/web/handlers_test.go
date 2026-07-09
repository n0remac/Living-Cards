package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/design"
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
		`id="game-edit-mode"`,
		`id="game-edit-title"`,
		`id="game-edit-status"`,
		`id="game-edit-save"`,
		`id="game-edit-cancel"`,
		`id="game-edit-canvas"`,
		`id="game-edit-card"`,
		`id="game-edit-component-tray"`,
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
		`id="config-target"`,
		`value="background"`,
		`value="border"`,
		`value="textarea"`,
		`value="image"`,
		`id="generate-config-btn"`,
		`id="update-config-btn"`,
		`id="design-library-list"`,
		`id="add-textarea-component-btn"`,
		`id="add-shape-component-btn"`,
		`id="add-image-component-input"`,
		`id="config-preview"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("page missing %s:\n%s", marker, body)
		}
	}
	for _, marker := range []string{
		`id="app-header"`,
		`id="designer-toggle-btn"`,
		`id="controller-builder-overlay"`,
		`id="controller-slider-input"`,
		`id="controller-builder-save"`,
		`id="game-field-panel"`,
		`id="game-field-card"`,
		`id="game-field-edit"`,
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
	if document.CardID != cardcomponent.DefaultCardID || document.Root.ComponentKind != cardcomponent.Kind {
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
		Document             cardcomponent.Document      `json:"document"`
		GameState            GameState                   `json:"gameState"`
		PreviewHTML          string                      `json:"preview_html"`
		AvailableConfigKinds []string                    `json:"availableConfigKinds"`
		Library              []cardcomponent.LibraryItem `json:"library"`
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
	if !hasString(payload.AvailableConfigKinds, "background") || !hasString(payload.AvailableConfigKinds, "border") {
		t.Fatalf("availableConfigKinds = %#v", payload.AvailableConfigKinds)
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

func TestGameSaveControllerRequiresCollectedParts(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := postJSON(t, mux, "/api/game/save-controller", map[string]any{
		"templateCardId": "blank-controller",
		"document":       webControllerDocument(t, 73),
	})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "blank controller") {
		t.Fatalf("body = %q, want blank controller error", recorder.Body.String())
	}
}

func TestGameSaveControllerRejectsMalformedDocument(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	advanceWebToGeneratorRoom(t, mux)
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "slider-component"})
	recorder := postJSON(t, mux, "/api/game/save-controller", map[string]any{
		"templateCardId": "blank-controller",
		"document": cardcomponent.Document{
			CardID: "bad-controller",
			Name:   "Bad Controller",
			Root:   cardcomponent.Node{ID: "bad-controller-root", ComponentKind: cardcomponent.Kind},
		},
	})
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "slider component") {
		t.Fatalf("body = %q, want slider document error", recorder.Body.String())
	}
}

func TestGameSaveControllerAddsRenderedLibraryCard(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	advanceWebToGeneratorRoom(t, mux)
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "slider-component"})
	recorder := postJSON(t, mux, "/api/game/save-controller", map[string]any{
		"templateCardId": "blank-controller",
		"document":       webControllerDocument(t, 73),
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload GameSessionSnapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	controller := gameCard(payload.Library, "generator-regulator-controller")
	if controller == nil {
		t.Fatalf("library = %#v, want regulator controller", payload.Library)
	}
	if !strings.Contains(controller.PreviewHTML, `data-component-kind="slider"`) || !strings.Contains(controller.PreviewHTML, `value="73"`) {
		t.Fatalf("controller preview missing slider: %s", controller.PreviewHTML)
	}
}

func TestGameEditWorkflowInstallsTunesAndSavesSlider(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	advanceWebToGeneratorRoom(t, mux)
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "slider-component"})

	startRecorder := postJSON(t, mux, "/api/game/edit/start", map[string]any{"cardId": "blank-controller"})
	if startRecorder.Code != http.StatusOK {
		t.Fatalf("start status = %d, want 200 body=%s", startRecorder.Code, startRecorder.Body.String())
	}
	var startPayload GameSessionSnapshot
	if err := json.Unmarshal(startRecorder.Body.Bytes(), &startPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, startRecorder.Body.String())
	}
	if startPayload.EditSession == nil || startPayload.EditSession.DraftCard.ID != "blank-controller" || startPayload.EditSession.EditingOverlay != nil {
		t.Fatalf("start payload = %#v, want draft with no selected component overlay", startPayload)
	}

	installRecorder := postJSON(t, mux, "/api/game/edit/install-component", map[string]any{"componentCardId": "slider-component"})
	if installRecorder.Code != http.StatusOK {
		t.Fatalf("install status = %d, want 200 body=%s", installRecorder.Code, installRecorder.Body.String())
	}
	var installPayload GameSessionSnapshot
	if err := json.Unmarshal(installRecorder.Body.Bytes(), &installPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, installRecorder.Body.String())
	}
	if gameCard(installPayload.Library, "slider-component") == nil {
		t.Fatalf("library = %#v, slider component should not be consumed before save", installPayload.Library)
	}
	if installPayload.EditSession == nil || !strings.Contains(installPayload.EditSession.DraftCard.PreviewHTML, `data-component-kind="slider"`) {
		t.Fatalf("draft card missing slider: %#v", installPayload.EditSession)
	}
	if installPayload.EditSession.EditingOverlay == nil ||
		installPayload.EditSession.EditingOverlay.ComponentKind != "slider" ||
		len(installPayload.EditSession.EditingOverlay.Controls) != 0 {
		t.Fatalf("editing overlay = %#v, want selectable slider overlay without controls", installPayload.EditSession.EditingOverlay)
	}

	tuneRecorder := postJSON(t, mux, "/api/game/edit/control-change", map[string]any{
		"componentId": "regulator-output-slider",
		"control":     "value",
		"value":       73,
	})
	if tuneRecorder.Code != http.StatusOK {
		t.Fatalf("tune status = %d, want 200 body=%s", tuneRecorder.Code, tuneRecorder.Body.String())
	}
	var tuned GameSessionSnapshot
	if err := json.Unmarshal(tuneRecorder.Body.Bytes(), &tuned); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, tuneRecorder.Body.String())
	}
	if tuned.EditSession == nil || !strings.Contains(tuned.EditSession.DraftCard.PreviewHTML, `value="73"`) {
		t.Fatalf("tuned draft preview missing value 73: %#v", tuned.EditSession)
	}

	saveRecorder := postJSON(t, mux, "/api/game/edit/save", map[string]any{})
	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, want 200 body=%s", saveRecorder.Code, saveRecorder.Body.String())
	}
	var saved GameSessionSnapshot
	if err := json.Unmarshal(saveRecorder.Body.Bytes(), &saved); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, saveRecorder.Body.String())
	}
	if saved.EditSession != nil || gameCard(saved.Library, "slider-component") != nil {
		t.Fatalf("saved payload = %#v, want closed edit and consumed slider", saved)
	}

	playRecorder := postJSON(t, mux, "/api/game/play-card", map[string]any{
		"sourceCardId": "blank-controller",
		"targetCardId": "generator-panel",
	})
	if playRecorder.Code != http.StatusOK {
		t.Fatalf("play status = %d, want 200 body=%s", playRecorder.Code, playRecorder.Body.String())
	}
	var played GameSessionSnapshot
	if err := json.Unmarshal(playRecorder.Body.Bytes(), &played); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, playRecorder.Body.String())
	}
	if !played.SolvedFlags["generatorPowered"] || !strings.Contains(gameCardHTML(played.WorldDeck, "generator-panel"), "GENERATOR ONLINE") {
		t.Fatalf("played payload = %#v, want powered generator", played)
	}
}

func TestGameEditComponentSelectEndpointReopensSliderAndBorderOverlays(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	advanceWebToGeneratorRoom(t, mux)
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "slider-component"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "border-component"})
	postJSON(t, mux, "/api/game/edit/start", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/edit/install-component", map[string]any{"componentCardId": "slider-component"})
	postJSON(t, mux, "/api/game/edit/install-component", map[string]any{"componentCardId": "border-component"})

	sliderRecorder := postJSON(t, mux, "/api/game/edit/component/select", map[string]any{"componentKind": "slider"})
	if sliderRecorder.Code != http.StatusOK {
		t.Fatalf("slider select status = %d, want 200 body=%s", sliderRecorder.Code, sliderRecorder.Body.String())
	}
	var sliderPayload GameSessionSnapshot
	if err := json.Unmarshal(sliderRecorder.Body.Bytes(), &sliderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, sliderRecorder.Body.String())
	}
	if sliderPayload.EditSession == nil ||
		sliderPayload.EditSession.SelectedComponentID != "regulator-output-slider" ||
		sliderPayload.EditSession.EditingOverlay == nil ||
		sliderPayload.EditSession.EditingOverlay.ComponentKind != "slider" ||
		len(sliderPayload.EditSession.EditingOverlay.Controls) != 0 {
		t.Fatalf("slider select payload = %#v, want slider overlay without controls", sliderPayload.EditSession)
	}

	borderRecorder := postJSON(t, mux, "/api/game/edit/component/select", map[string]any{"componentKind": "border"})
	if borderRecorder.Code != http.StatusOK {
		t.Fatalf("border select status = %d, want 200 body=%s", borderRecorder.Code, borderRecorder.Body.String())
	}
	var borderPayload GameSessionSnapshot
	if err := json.Unmarshal(borderRecorder.Body.Bytes(), &borderPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, borderRecorder.Body.String())
	}
	if borderPayload.EditSession == nil ||
		borderPayload.EditSession.SelectedComponentID != "blank-controller-border" ||
		borderPayload.EditSession.EditingOverlay == nil ||
		borderPayload.EditSession.EditingOverlay.ComponentKind != "border" ||
		!hasControl(borderPayload.EditSession.EditingOverlay.Controls, "border_color") {
		t.Fatalf("border select payload = %#v, want border overlay controls", borderPayload.EditSession)
	}
}

func TestGameLibrarySliderControlChangeEndpointRetunesSavedController(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	advanceWebToGeneratorRoom(t, mux)
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "slider-component"})
	recorder := postJSON(t, mux, "/api/game/save-controller", map[string]any{
		"templateCardId": "blank-controller",
		"document":       webControllerDocument(t, 73),
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("save-controller status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}

	updateRecorder := postJSON(t, mux, "/api/game/library/component/control-change", map[string]any{
		"cardId":        "generator-regulator-controller",
		"componentId":   "regulator-output-slider",
		"componentKind": "slider",
		"control":       "value",
		"value":         72,
	})
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("library update status = %d, want 200 body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var payload GameSessionSnapshot
	if err := json.Unmarshal(updateRecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, updateRecorder.Body.String())
	}
	controller := gameCard(payload.Library, "generator-regulator-controller")
	if controller == nil || !strings.Contains(controller.PreviewHTML, `value="72"`) {
		t.Fatalf("controller = %#v, want retuned library slider", controller)
	}
}

func TestGameEditWorkflowExposesBorderControls(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	advanceWebToGeneratorRoom(t, mux)
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "blank-controller"})
	postJSON(t, mux, "/api/game/collect", map[string]any{"cardId": "border-component"})
	postJSON(t, mux, "/api/game/edit/start", map[string]any{"cardId": "blank-controller"})

	installRecorder := postJSON(t, mux, "/api/game/edit/install-component", map[string]any{"componentCardId": "border-component"})
	if installRecorder.Code != http.StatusOK {
		t.Fatalf("install status = %d, want 200 body=%s", installRecorder.Code, installRecorder.Body.String())
	}
	var installPayload GameSessionSnapshot
	if err := json.Unmarshal(installRecorder.Body.Bytes(), &installPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, installRecorder.Body.String())
	}
	if installPayload.EditSession == nil || installPayload.EditSession.EditingOverlay == nil || installPayload.EditSession.EditingOverlay.ComponentKind != "border" {
		t.Fatalf("editing overlay = %#v, want border controls", installPayload.EditSession)
	}
	if !hasControlWithProperty(installPayload.EditSession.EditingOverlay.Controls, "border_color", "border-color") ||
		!hasControlWithProperty(installPayload.EditSession.EditingOverlay.Controls, "border_style", "border-style") {
		t.Fatalf("border overlay controls = %#v, want property hints", installPayload.EditSession.EditingOverlay.Controls)
	}

	cancelRecorder := postJSON(t, mux, "/api/game/edit/cancel", map[string]any{})
	if cancelRecorder.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200 body=%s", cancelRecorder.Code, cancelRecorder.Body.String())
	}
	var canceled GameSessionSnapshot
	if err := json.Unmarshal(cancelRecorder.Body.Bytes(), &canceled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, cancelRecorder.Body.String())
	}
	if canceled.EditSession != nil || gameCard(canceled.Library, "border-component") == nil {
		t.Fatalf("canceled payload = %#v, want closed edit and retained border component", canceled)
	}
}

func TestGameActiveComponentSelectionAndControlChange(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	selectRecorder := postJSON(t, mux, "/api/game/component/select", map[string]any{
		"cardId":        "rusted-cell-door",
		"componentId":   "door-title",
		"componentKind": "textarea",
	})
	if selectRecorder.Code != http.StatusOK {
		t.Fatalf("select status = %d, want 200 body=%s", selectRecorder.Code, selectRecorder.Body.String())
	}
	var selected GameSessionSnapshot
	if err := json.Unmarshal(selectRecorder.Body.Bytes(), &selected); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, selectRecorder.Body.String())
	}
	if selected.ActiveEditingComponentID != "door-title" || selected.ActiveEditingOverlay == nil || selected.ActiveEditingOverlay.ComponentKind != "textarea" {
		t.Fatalf("selected payload = %#v, want text active editing overlay", selected)
	}
	if !hasControl(selected.ActiveEditingOverlay.Controls, "content") || !hasControl(selected.ActiveEditingOverlay.Controls, "x") {
		t.Fatalf("active text controls = %#v, want content and x controls", selected.ActiveEditingOverlay.Controls)
	}

	moveRecorder := postJSON(t, mux, "/api/game/component/control-change", map[string]any{
		"cardId":        "rusted-cell-door",
		"componentId":   "door-title",
		"componentKind": "textarea",
		"control":       "position",
		"value":         map[string]any{"x": 27, "y": 33},
	})
	if moveRecorder.Code != http.StatusOK {
		t.Fatalf("move status = %d, want 200 body=%s", moveRecorder.Code, moveRecorder.Body.String())
	}
	var moved GameSessionSnapshot
	if err := json.Unmarshal(moveRecorder.Body.Bytes(), &moved); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, moveRecorder.Body.String())
	}
	rawDocument, err := json.Marshal(moved.ActiveWorldCard.Document)
	if err != nil {
		t.Fatalf("json.Marshal(document) error = %v", err)
	}
	if !strings.Contains(string(rawDocument), `"x":27`) || !strings.Contains(string(rawDocument), `"y":33`) {
		t.Fatalf("moved document = %s, want x/y position", string(rawDocument))
	}
	if !strings.Contains(moved.ActiveWorldCard.PreviewHTML, `left: 27%`) || !strings.Contains(moved.ActiveWorldCard.PreviewHTML, `top: 33%`) {
		t.Fatalf("moved preview missing position: %s", moved.ActiveWorldCard.PreviewHTML)
	}

	borderRecorder := postJSON(t, mux, "/api/game/component/select", map[string]any{
		"cardId":        "rusted-cell-door",
		"componentKind": "border",
	})
	if borderRecorder.Code != http.StatusBadRequest {
		t.Fatalf("border status = %d, want 400 body=%s", borderRecorder.Code, borderRecorder.Body.String())
	}
	if !strings.Contains(borderRecorder.Body.String(), "require finding") {
		t.Fatalf("border body = %q, want missing component card error", borderRecorder.Body.String())
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

func TestTapDraftCardAppliesRandomConfigsAndProgresses(t *testing.T) {
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
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"`+test.target+`","zone":"`+test.target+`","x":0.2,"y":0.4}`))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				Document      cardcomponent.Document `json:"document"`
				GameState     GameState              `json:"gameState"`
				AppliedConfig json.RawMessage        `json:"appliedConfig"`
				PreviewHTML   string                 `json:"preview_html"`
				Events        []CardEvent            `json:"events"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
			}
			if payload.GameState.TapCount != 1 || payload.GameState.XP != 1 || payload.GameState.Level != 1 {
				t.Fatalf("gameState = %#v", payload.GameState)
			}
			if !hasEvent(payload.Events, "configApplied", test.target) || !hasEvent(payload.Events, "xpGained", "") {
				t.Fatalf("events = %#v", payload.Events)
			}
			if !strings.Contains(string(payload.AppliedConfig), test.field) {
				t.Fatalf("appliedConfig = %s, want field %s", string(payload.AppliedConfig), test.field)
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"border","zone":"border","x":0.05,"y":0.5}`))
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"`+target+`","zone":"`+target+`","x":`+x+`,"y":0.5}`))
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
	if !hasString(payload.GameState.ComponentKindProgress["background"].UnlockedModes, "simpleControls") {
		t.Fatalf("background progress = %#v, want simpleControls", payload.GameState.ComponentKindProgress["background"])
	}
	if !hasString(payload.GameState.ComponentKindProgress["border"].UnlockedModes, "simpleControls") {
		t.Fatalf("border progress = %#v, want simpleControls", payload.GameState.ComponentKindProgress["border"])
	}
}

func TestTapDraftCardHandlesLockedAndInvalidTargets(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	lockedRecorder := httptest.NewRecorder()
	lockedRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"textarea","zone":"textarea","x":0.5,"y":0.5}`))
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
	invalidRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"shadow","zone":"shadow","x":0.5,"y":0.5}`))
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
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentKind":"`+test.target+`","color":"`+test.color+`"}`))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
			}
			var payload struct {
				GameState     GameState       `json:"gameState"`
				AppliedConfig json.RawMessage `json:"appliedConfig"`
				PreviewHTML   string          `json:"preview_html"`
				Events        []CardEvent     `json:"events"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
			}
			if payload.GameState.Level < 5 || payload.GameState.TapCount != 21 {
				t.Fatalf("gameState = %#v", payload.GameState)
			}
			if !strings.Contains(string(payload.AppliedConfig), test.field) {
				t.Fatalf("appliedConfig = %s, want %s", string(payload.AppliedConfig), test.field)
			}
			if !strings.Contains(payload.PreviewHTML, test.color) {
				t.Fatalf("preview_html did not include %s: %s", test.color, payload.PreviewHTML)
			}
			if !hasEvent(payload.Events, "configApplied", test.target) {
				t.Fatalf("events = %#v, want configApplied", payload.Events)
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
			body:   `{"componentKind":"background","color":"#22c55e","secondaryColor":"#38bdf8","gradient":true,"angle":45}`,
			field:  `"background_color":"#22c55e"`,
			css:    `linear-gradient(45deg, #22c55e 0%, #38bdf8 100%)`,
		},
		{
			name:   "border",
			target: "border",
			body:   `{"componentKind":"border","color":"#f59e0b","secondaryColor":"#a78bfa","gradient":true,"angle":210}`,
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
				AppliedConfig json.RawMessage `json:"appliedConfig"`
				PreviewHTML   string          `json:"preview_html"`
				Events        []CardEvent     `json:"events"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
			}
			if !strings.Contains(string(payload.AppliedConfig), test.field) || !strings.Contains(string(payload.AppliedConfig), test.css) {
				t.Fatalf("appliedConfig = %s, want field %s and css %s", string(payload.AppliedConfig), test.field, test.css)
			}
			if !strings.Contains(payload.PreviewHTML, test.css) {
				t.Fatalf("preview_html did not include %s: %s", test.css, payload.PreviewHTML)
			}
			if !hasEvent(payload.Events, "configApplied", test.target) {
				t.Fatalf("events = %#v, want configApplied", payload.Events)
			}
		})
	}
}

func TestControlChangeDraftCardRequiresUnlockedMode(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentKind":"background","color":"#22c55e"}`))
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
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentKind":"background","color":"url(https://example.test)"}`))
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
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/control-change", strings.NewReader(`{"componentKind":"background","color":"#22c55e","gradient":true}`))
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"border","zone":"border","x":0.05,"y":0.5}`))
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"border","zone":"border","x":0.05,"y":0.5}`))
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
		GameState     GameState       `json:"gameState"`
		AppliedConfig json.RawMessage `json:"appliedConfig"`
		Events        []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.GameState.TotalXP != 13 || !payload.GameState.ComponentProgress["card-root"].RandomTapEnabled {
		t.Fatalf("gameState = %#v", payload.GameState)
	}
	if len(payload.AppliedConfig) == 0 {
		t.Fatal("appliedConfig should be present while randomizing is enabled")
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"border","zone":"border","x":0.05,"y":0.5}`))
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
		GameState     GameState       `json:"gameState"`
		AppliedConfig json.RawMessage `json:"appliedConfig"`
		Events        []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(tapRecorder.Body.Bytes(), &tapPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, tapRecorder.Body.String())
	}
	if tapPayload.GameState.TotalXP != 7 || len(tapPayload.AppliedConfig) != 0 || len(tapPayload.Events) != 0 {
		t.Fatalf("tap payload = %#v applied=%s events=%#v", tapPayload.GameState, string(tapPayload.AppliedConfig), tapPayload.Events)
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
		GameState     GameState       `json:"gameState"`
		AppliedConfig json.RawMessage `json:"appliedConfig"`
		PreviewHTML   string          `json:"preview_html"`
		Events        []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.GameState.TotalXP != 21 || !strings.Contains(payload.PreviewHTML, `padding: 32px`) {
		t.Fatalf("payload = %#v preview=%s", payload.GameState, payload.PreviewHTML)
	}
	if !strings.Contains(string(payload.AppliedConfig), `"componentKind":"card"`) {
		t.Fatalf("appliedConfig = %s", string(payload.AppliedConfig))
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/components", strings.NewReader(`{"componentKind":"image"}`))
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
	if strings.Contains(string(imageOne.Config), `"x":72`) {
		t.Fatalf("image-1 was mutated instead of image-2: %s", string(imageOne.Config))
	}
	if !strings.Contains(string(imageTwo.Config), `"x":72`) || !strings.Contains(string(imageTwo.Config), `"y":22`) {
		t.Fatalf("image-2 config was not moved: %s", string(imageTwo.Config))
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
	body := `{"componentKind":"image","config":{"src":"data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=","alt":"svg","x":50,"y":50,"width":20,"height":20,"border_color":"#ffffff"}}`
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
		GameState     GameState       `json:"gameState"`
		AppliedConfig json.RawMessage `json:"appliedConfig"`
		Events        []CardEvent     `json:"events"`
	}
	if err := json.Unmarshal(randomizeRecorder.Body.Bytes(), &randomizePayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, randomizeRecorder.Body.String())
	}
	if !strings.Contains(string(randomizePayload.AppliedConfig), `"componentKind":"shape"`) {
		t.Fatalf("appliedConfig = %s", string(randomizePayload.AppliedConfig))
	}
	if randomizePayload.GameState.ComponentProgress["shape-1"].XP != 1 || !hasEvent(randomizePayload.Events, "xpGained", "") {
		t.Fatalf("randomize payload = %#v events=%#v", randomizePayload.GameState.ComponentProgress["shape-1"], randomizePayload.Events)
	}
}

func TestResetDraftCardRestoresGameProgress(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	tapRecorder := httptest.NewRecorder()
	tapRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"background","zone":"background","x":0.5,"y":0.5}`))
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

func TestDraftConfigRoutesGenerateConfigs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		target string
		raw    string
		field  string
	}{
		{
			target: "background",
			raw:    `{"componentKind":"background","description":"Moody teal background","config":{"background_color":"#0f766e","css":"background: linear-gradient(135deg, #0f766e, #111827);"}}`,
			field:  `"background_color":"#0f766e"`,
		},
		{
			target: "border",
			raw:    `{"componentKind":"border","description":"Fine white border","config":{"border_width_px":2,"border_radius_px":18,"border_color":"#ffffff","css":"box-shadow: 0 0 20px rgba(255,255,255,0.2);"}}`,
			field:  `"border_width_px":2`,
		},
		{
			target: "textarea",
			raw:    `{"componentKind":"textarea","description":"Centered text","config":{"content":"Hello card","font_family":"system","font_size_px":28,"font_weight":700,"font_style":"normal","color":"#f8fafc","align":"center","position":"center","css":"text-align: center;"}}`,
			field:  `"content":"Hello card"`,
		},
	}
	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, &fakePatchClient{responses: []string{test.raw}})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/draft-card/configs/"+test.target, strings.NewReader(`{"instruction":"make it polished"}`))
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

func TestDraftConfigRoutesRepairInvalidOutput(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"componentKind":"textarea","config":{"content":"","font_family":"system","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":""}}`,
		`{"componentKind":"textarea","description":"Repaired centered text","config":{"content":"Repaired text","font_family":"system","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"center","position":"center","css":"text-align: center;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/configs/textarea", strings.NewReader(`{"instruction":"write a centered note"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
	if !strings.Contains(recorder.Body.String(), `"content":"Repaired text"`) {
		t.Fatalf("response did not include repaired config: %s", recorder.Body.String())
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

func TestDraftConfigRouteReturnsRawRepairFailure(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"componentKind":"textarea","description":"","config":{"content":"","font_family":"bad","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":""}}`,
		`{"componentKind":"textarea","description":"","config":{"content":"","font_family":"bad","font_size_px":18,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":"position: absolute;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/configs/textarea", strings.NewReader(`{"instruction":"make broken text"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 body=%s", recorder.Code, recorder.Body.String())
	}
	if client.calls != 2 {
		t.Fatalf("calls = %d, want 2", client.calls)
	}
	var payload struct {
		Message     string         `json:"message"`
		RawResponse string         `json:"raw_response"`
		Issues      []design.Issue `json:"issues"`
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

func TestDraftConfigRouteIncludesUpdateContext(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"componentKind":"background","description":"Updated background","config":{"background_color":"#111827","css":"background: #111827;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/configs/background", strings.NewReader(`{"instruction":"make it darker","old_code":"{\"background_color\":\"#ffffff\",\"css\":\"\"}","component_id":"background-primary"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	prompt := joinedMessages(client.messages[0])
	for _, marker := range []string{
		"make it darker",
		"Existing config JSON to update",
		`"background_color":"#ffffff"`,
		"background-primary",
	} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("prompt missing %q:\n%s", marker, prompt)
		}
	}
}

func TestDraftConfigRouteUsesServerCurrentConfigForUpdate(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"componentKind":"background","description":"Updated background","config":{"background_color":"#111827","css":"background: #111827;"}}`,
	}}
	mux := testMux(t, client)
	applyRecorder := httptest.NewRecorder()
	applyBody := applyRequestBody(t, json.RawMessage(`{
		"componentKind":"background",
		"description":"Applied white background",
		"config":{"background_color":"#ffffff","css":"background: #ffffff;"}
	}`))
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-config", strings.NewReader(applyBody))
	applyRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(applyRecorder, applyRequest)
	if applyRecorder.Code != http.StatusOK {
		t.Fatalf("apply status = %d, want 200 body=%s", applyRecorder.Code, applyRecorder.Body.String())
	}

	generateRecorder := httptest.NewRecorder()
	generateRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/configs/background", strings.NewReader(`{"instruction":"make it darker","update":true}`))
	generateRequest.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(generateRecorder, generateRequest)
	if generateRecorder.Code != http.StatusOK {
		t.Fatalf("generate status = %d, want 200 body=%s", generateRecorder.Code, generateRecorder.Body.String())
	}
	prompt := joinedMessages(client.messages[0])
	for _, marker := range []string{
		"Existing config JSON to update",
		`"background_color": "#ffffff"`,
		"background-primary",
	} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("prompt missing %q:\n%s", marker, prompt)
		}
	}
}

func TestDraftConfigRouteDoesNotUseServerCurrentConfigForGenerate(t *testing.T) {
	t.Parallel()

	client := &fakePatchClient{responses: []string{
		`{"componentKind":"background","description":"Updated background","config":{"background_color":"#111827","css":"background: #111827;"}}`,
	}}
	mux := testMux(t, client)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/configs/background", strings.NewReader(`{"instruction":"make it darker"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	prompt := joinedMessages(client.messages[0])
	if strings.Contains(prompt, "Existing config JSON to update") {
		t.Fatalf("generate prompt should not include update context:\n%s", prompt)
	}
}

func TestApplyDraftConfigValidatesAndRendersPreview(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := applyRequestBody(t, json.RawMessage(`{
		"componentKind":"textarea",
		"description":"Large centered note",
		"config":{
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
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-config", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Document         cardcomponent.Document `json:"document"`
		NormalizedConfig json.RawMessage        `json:"normalized_config"`
		PreviewHTML      string                 `json:"preview_html"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if !strings.Contains(payload.PreviewHTML, `id="draft-card-preview"`) || !strings.Contains(payload.PreviewHTML, `Server rendered text`) {
		t.Fatalf("preview_html did not include rendered card: %s", payload.PreviewHTML)
	}
	if !strings.Contains(string(payload.NormalizedConfig), `"font_size_px":72`) {
		t.Fatalf("normalized config did not clamp font size: %s", string(payload.NormalizedConfig))
	}
	if !strings.Contains(string(payload.Document.Root.Children[2].Config), `"Server rendered text"`) {
		t.Fatalf("document textarea config was not replaced: %s", string(payload.Document.Root.Children[2].Config))
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
	if !strings.Contains(string(document.Root.Children[2].Config), `"Server rendered text"`) {
		t.Fatalf("server state did not retain applied config: %s", string(document.Root.Children[2].Config))
	}
}

func TestDesignLibraryRoutesUseServerState(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/draft-card/library?componentKind=background", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("library status = %d, want 200 body=%s", recorder.Code, recorder.Body.String())
	}
	var libraryPayload struct {
		Library []cardcomponent.LibraryItem `json:"library"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &libraryPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if len(libraryPayload.Library) == 0 || libraryPayload.Library[0].ComponentKind != "background" {
		t.Fatalf("library = %#v, want background presets", libraryPayload.Library)
	}

	recorder = httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/draft-card/library/save-applied", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("save before apply status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}

	applyRecorder := httptest.NewRecorder()
	applyBody := applyRequestBody(t, json.RawMessage(`{
		"componentKind":"border",
		"description":"Saved border",
		"config":{"border_width_px":2,"border_radius_px":16,"border_color":"#ffffff","css":"border: 2px solid #ffffff;"}
	}`))
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-config", strings.NewReader(applyBody))
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
	if !savePayload.Item.Saved || savePayload.Item.ComponentKind != "border" {
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
		t.Fatalf("apply library response did not include preset config: %s", applyLibraryRecorder.Body.String())
	}
}

func TestApplyDraftConfigReturnsStructuredValidationIssues(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := applyRequestBody(t, json.RawMessage(`{
		"componentKind":"background",
		"description":"Unsafe background",
		"config":{
			"background_color":"#111827",
			"css":"background-image: url(https://example.test/image.png);"
		}
	}`))
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-config", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Message     string         `json:"message"`
		RawResponse string         `json:"raw_response"`
		Issues      []design.Issue `json:"issues"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v body=%s", err, recorder.Body.String())
	}
	if payload.RawResponse == "" || len(payload.Issues) == 0 {
		t.Fatalf("payload = %#v, want raw response and issues", payload)
	}
	if payload.Issues[0].Path != "config.css" {
		t.Fatalf("issue path = %q, want config.css", payload.Issues[0].Path)
	}
}

func TestResetDraftCardRestoresServerState(t *testing.T) {
	t.Parallel()

	mux := testMux(t, nil)
	recorder := httptest.NewRecorder()
	body := applyRequestBody(t, json.RawMessage(`{
		"componentKind":"textarea",
		"description":"Text",
		"config":{
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
	request := httptest.NewRequest(http.MethodPost, "/api/draft-card/apply-config", strings.NewReader(body))
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
	if !strings.Contains(string(payload.Document.Root.Children[2].Config), `Start designing this card.`) {
		t.Fatalf("reset did not restore default textarea: %s", string(payload.Document.Root.Children[2].Config))
	}
	if !strings.Contains(payload.PreviewHTML, `Start designing this card.`) {
		t.Fatalf("reset preview did not render default text: %s", payload.PreviewHTML)
	}
}

func TestApplyDraftConfigRejectsBadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		body   string
		status int
	}{
		{name: "wrong method", method: http.MethodGet, body: ``, status: http.StatusMethodNotAllowed},
		{name: "malformed body", method: http.MethodPost, body: `{`, status: http.StatusBadRequest},
		{name: "unknown request field", method: http.MethodPost, body: `{"generated_config":{},"extra":true}`, status: http.StatusBadRequest},
		{name: "unknown target", method: http.MethodPost, body: applyRequestBody(t, json.RawMessage(`{"componentKind":"shadow","description":"Shadow","config":{}}`)), status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mux := testMux(t, nil)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(test.method, "/api/draft-card/apply-config", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(recorder, request)

			if recorder.Code != test.status {
				t.Fatalf("status = %d, want %d body=%s", recorder.Code, test.status, recorder.Body.String())
			}
		})
	}
}

func TestDraftConfigRouteRejectsBadRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "empty instruction", method: http.MethodPost, path: "/api/draft-card/configs/background", body: `{"instruction":" "}`, status: http.StatusBadRequest},
		{name: "malformed body", method: http.MethodPost, path: "/api/draft-card/configs/background", body: `{`, status: http.StatusBadRequest},
		{name: "unknown target", method: http.MethodPost, path: "/api/draft-card/configs/shadow", body: `{"instruction":"make it"}`, status: http.StatusNotFound},
		{name: "removed title target", method: http.MethodPost, path: "/api/draft-card/configs/title", body: `{"instruction":"make it"}`, status: http.StatusNotFound},
		{name: "removed body target", method: http.MethodPost, path: "/api/draft-card/configs/body", body: `{"instruction":"make it"}`, status: http.StatusNotFound},
		{name: "wrong method", method: http.MethodGet, path: "/api/draft-card/configs/background", body: ``, status: http.StatusMethodNotAllowed},
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
		GeneratedConfig json.RawMessage `json:"generated_config"`
	}{
		GeneratedConfig: generated,
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
		if target == "" || event.ComponentKind == target {
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

func hasControlWithProperty(controls []ControlDescriptor, control, property string) bool {
	for _, candidate := range controls {
		if candidate.Control == control && candidate.Property == property {
			return true
		}
	}
	return false
}

func postJSON(t *testing.T, mux *http.ServeMux, path string, value any) *httptest.ResponseRecorder {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	return recorder
}

func advanceWebToGeneratorRoom(t *testing.T, mux *http.ServeMux) {
	t.Helper()

	for _, request := range []struct {
		path string
		body map[string]any
	}{
		{path: "/api/game/collect", body: map[string]any{"cardId": "bent-iron-key"}},
		{path: "/api/game/play-card", body: map[string]any{"sourceCardId": "bent-iron-key", "targetCardId": "rusted-cell-door"}},
		{path: "/api/game/collect", body: map[string]any{"cardId": "glass-fuse"}},
		{path: "/api/game/play-card", body: map[string]any{"sourceCardId": "glass-fuse", "targetCardId": "sleeping-switch"}},
	} {
		recorder := postJSON(t, mux, request.path, request.body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200 body=%s", request.path, recorder.Code, recorder.Body.String())
		}
	}
}

func webControllerDocument(t *testing.T, value int) cardcomponent.Document {
	t.Helper()

	raw, err := json.Marshal(slider.Config{
		Label: "Output",
		Min:   0,
		Max:   100,
		Step:  1,
		Value: value,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return cardcomponent.Document{
		CardID: "generator-regulator-controller",
		Name:   "Regulator Controller",
		Root: cardcomponent.Node{
			ID:            "generator-regulator-controller-root",
			ComponentKind: cardcomponent.Kind,
			Children: []cardcomponent.Node{{
				ID:            "regulator-output-slider",
				ComponentKind: slider.Kind,
				Config:        raw,
			}},
		},
	}
}

func gameCard(cards []RenderedGameCard, cardID string) *RenderedGameCard {
	for index := range cards {
		if cards[index].ID == cardID {
			return &cards[index]
		}
	}
	return nil
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
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"border","zone":"border","x":0.05,"y":0.5}`))
		request.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("tap %d status = %d, want 200 body=%s", index+1, recorder.Code, recorder.Body.String())
		}
	}
	for index := 0; index < 8; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/draft-card/tap", strings.NewReader(`{"componentKind":"textarea","zone":"textarea","x":0.5,"y":0.5}`))
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
