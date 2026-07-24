package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	. "github.com/n0remac/GoDom/html"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
	"github.com/n0remac/Living-Card/internal/components/schema"
	"github.com/n0remac/Living-Card/internal/design"
	"github.com/n0remac/Living-Card/internal/game"
)

type Dependencies struct {
	Patch      design.ChatClient
	PatchModel string
	Registry   *cardcomponent.Registry
}

func Register(mux *http.ServeMux, deps Dependencies) {
	if deps.Registry == nil {
		deps.Registry = catalog.MustNew()
	}
	state := newDesignerState(deps.Registry)
	gameSession := game.NewSession(deps.Registry)
	mux.HandleFunc("/", pageHandler())
	mux.HandleFunc("/api/", http.NotFound)
	mux.HandleFunc("/assets/", frontendAssetHandler())
	mux.HandleFunc("/api/game/", gameResourceHandler(deps.Registry, gameSession))
	mux.HandleFunc("/api/draft-card", draftCardResourceHandler(deps, state))
	mux.HandleFunc("/api/draft-card/", draftCardResourceHandler(deps, state))
}

func pageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		ServeNode(Page())(w, r)
	}
}

func frontendAssetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var path string
		switch r.URL.Path {
		case "/assets/app.js":
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			path = frontendAssetPath("app.js")
		case "/assets/app.js.map":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			path = frontendAssetPath("app.js.map")
		default:
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, path)
	}
}

func frontendAssetPath(name string) string {
	return filepath.Join(projectRoot(), "web", "dist", name)
}

func projectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

func draftCardResourceHandler(deps Dependencies, state *designerState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/draft-card")
		if path == "" || path == "/" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			document, _ := state.snapshot()
			writeJSONResponse(w, document)
			return
		}

		path = strings.Trim(path, "/")
		if path == "rendered" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			document, library := state.snapshot()
			writeRenderedDraftCard(state.registry, w, document, library)
			return
		}
		if path == "interactive" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			document, gameState, library := state.interactiveSnapshot()
			writeInteractiveDraftCard(state.registry, w, document, gameState, library)
			return
		}
		if path == "reset" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			document, library := state.reset()
			writeRenderedDraftCard(state.registry, w, document, library)
			return
		}
		if path == "apply-config" {
			applyDraftConfigHandler(w, r, state)
			return
		}
		if path == "tap" {
			tapDraftCardHandler(w, r, state)
			return
		}
		if path == "interact" {
			interactDraftCardHandler(w, r, state)
			return
		}
		if path == "control-change" {
			controlChangeDraftCardHandler(w, r, state)
			return
		}
		if path == "randomize-component" {
			randomizeDraftCardHandler(w, r, state)
			return
		}
		if path == "components" {
			addDraftComponentHandler(w, r, state)
			return
		}
		if path == "library" {
			designLibraryHandler(w, r, state)
			return
		}
		if path == "library/save-applied" {
			saveAppliedDesignHandler(w, r, state)
			return
		}
		if path == "library/apply" {
			applyLibraryDesignHandler(w, r, state)
			return
		}
		parts := strings.Split(path, "/")
		if len(parts) != 2 || parts[0] != "configs" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request struct {
			Instruction string `json:"instruction"`
			OldCode     string `json:"old_code"`
			ComponentID string `json:"component_id"`
			Update      bool   `json:"update"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if request.Update && request.OldCode == "" {
			if current, componentID := state.currentConfig(parts[1]); current != "" {
				request.OldCode = current
				if request.ComponentID == "" {
					request.ComponentID = componentID
				}
			}
		}
		definition, ok := deps.Registry.Lookup(parts[1])
		if !ok {
			http.NotFound(w, r)
			return
		}
		generation, supported := definition.Generation()
		if !supported || !generation.SupportsAI() {
			http.NotFound(w, r)
			return
		}
		generateDraftConfig(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, definition)
	}
}

type applyDraftConfigRequest struct {
	GeneratedConfig json.RawMessage `json:"generated_config"`
	ComponentID     string          `json:"component_id,omitempty"`
}

type applyDraftConfigResponse struct {
	Document         cardcomponent.Document      `json:"document"`
	NormalizedConfig json.RawMessage             `json:"normalized_config"`
	PreviewHTML      string                      `json:"preview_html"`
	Library          []cardcomponent.LibraryItem `json:"library"`
}

type renderedDraftCardResponse struct {
	Document    cardcomponent.Document      `json:"document"`
	PreviewHTML string                      `json:"preview_html"`
	Library     []cardcomponent.LibraryItem `json:"library"`
}

type interactiveDraftCardResponse struct {
	Document             cardcomponent.Document      `json:"document"`
	GameState            GameState                   `json:"gameState"`
	PreviewHTML          string                      `json:"preview_html"`
	AvailableConfigKinds []string                    `json:"availableConfigKinds"`
	AvailableComponents  []ComponentDescriptor       `json:"availableComponents"`
	Overlay              *ComponentOverlay           `json:"overlay,omitempty"`
	Library              []cardcomponent.LibraryItem `json:"library"`
}

type tapDraftCardRequest struct {
	ComponentKind string  `json:"componentKind"`
	Zone          string  `json:"zone"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
}

type controlChangeDraftCardRequest struct {
	ComponentID string          `json:"componentId"`
	Trait       string          `json:"trait,omitempty"`
	Control     string          `json:"control"`
	Value       json.RawMessage `json:"value"`
}

type interactDraftCardRequest struct {
	ComponentID string  `json:"componentId"`
	Trait       string  `json:"trait,omitempty"`
	Interaction string  `json:"interaction"`
	X           float64 `json:"x,omitempty"`
	Y           float64 `json:"y,omitempty"`
}

type randomizeDraftCardRequest struct {
	ComponentID string `json:"componentId"`
	Trait       string `json:"trait,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

type addDraftComponentRequest struct {
	ComponentKind string          `json:"componentKind"`
	Config        json.RawMessage `json:"config,omitempty"`
}

type tapDraftCardResponse struct {
	Document      cardcomponent.Document      `json:"document"`
	GameState     GameState                   `json:"gameState"`
	AppliedConfig json.RawMessage             `json:"appliedConfig,omitempty"`
	PreviewHTML   string                      `json:"preview_html"`
	Events        []CardEvent                 `json:"events"`
	Overlay       *ComponentOverlay           `json:"overlay,omitempty"`
	Library       []cardcomponent.LibraryItem `json:"library"`
}

type libraryResponse struct {
	Item    cardcomponent.LibraryItem   `json:"item,omitempty"`
	Library []cardcomponent.LibraryItem `json:"library"`
}

func applyDraftConfigHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request applyDraftConfigRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(request.GeneratedConfig) == 0 {
		writeApplyConfigError(w, design.NewInvalidModelOutputError("", []schema.Issue{{
			Path:    "generated_config",
			Code:    "required",
			Message: "generated_config is required",
		}}, design.ErrInvalidModelOutput), http.StatusBadRequest)
		return
	}
	if err := validateGeneratedConfigEnvelope(state.registry, request.GeneratedConfig); err != nil {
		writeApplyConfigError(w, err, http.StatusBadRequest)
		return
	}
	document, normalized, err := state.apply(request.GeneratedConfig, request.ComponentID)
	if err != nil {
		writeApplyConfigError(w, err, http.StatusBadRequest)
		return
	}
	_, library := state.snapshot()
	writeAppliedDraftConfig(state.registry, w, document, normalized, library)
}

func tapDraftCardHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request tapDraftCardRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := state.tap(request.ComponentKind, request.Zone, request.X, request.Y)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTappedDraftCard(state.registry, w, result)
}

func interactDraftCardHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request interactDraftCardRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := state.interact(request.ComponentID, request.Trait, request.Interaction, request.X, request.Y)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTappedDraftCard(state.registry, w, result)
}

func controlChangeDraftCardHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request controlChangeDraftCardRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := state.applyControlChange(request.ComponentID, request.Trait, request.Control, request.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTappedDraftCard(state.registry, w, result)
}

func randomizeDraftCardHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request randomizeDraftCardRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := state.randomizeComponent(request.ComponentID, request.Trait, request.Scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTappedDraftCard(state.registry, w, result)
}

func addDraftComponentHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request addDraftComponentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := state.addComponent(request.ComponentKind, request.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTappedDraftCard(state.registry, w, result)
}

func validateGeneratedConfigEnvelope(registry *cardcomponent.Registry, raw json.RawMessage) error {
	var envelope schema.GeneratedConfigEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return design.NewInvalidModelOutputError(string(raw), []schema.Issue{{
			Path:    "generated_config",
			Code:    "invalid_json",
			Message: "generated_config must be one JSON object: " + err.Error(),
		}}, design.ErrInvalidModelOutput)
	}
	definition, ok := registry.Lookup(strings.TrimSpace(envelope.ComponentKind))
	if !ok {
		return design.NewInvalidModelOutputError(string(raw), []schema.Issue{{
			Path:    "component_kind",
			Code:    "invalid_component_kind",
			Message: "component_kind is not registered",
			Actual:  envelope.ComponentKind,
		}}, design.ErrInvalidModelOutput)
	}
	generation, ok := definition.Generation()
	if !ok {
		return fmt.Errorf("component %q does not accept generated configs", definition.Kind())
	}
	_, issues := generation.CanonicalizeEnvelope(raw)
	if len(issues) > 0 {
		return design.NewInvalidModelOutputError(string(raw), issues, design.ErrInvalidModelOutput)
	}
	return nil
}

func designLibraryHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSONResponse(w, libraryResponse{
		Library: state.libraryForComponentKind(r.URL.Query().Get("componentKind")),
	})
}

func saveAppliedDesignHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	item, library, err := state.saveLastApplied()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, libraryResponse{Item: item, Library: library})
}

func applyLibraryDesignHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		ItemID string `json:"item_id"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.ItemID) == "" {
		http.Error(w, "item_id is required", http.StatusBadRequest)
		return
	}
	document, normalized, err := state.applyLibraryItem(strings.TrimSpace(request.ItemID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	_, library := state.snapshot()
	writeAppliedDraftConfig(state.registry, w, document, normalized, library)
}

func writeAppliedDraftConfig(registry *cardcomponent.Registry, w http.ResponseWriter, document cardcomponent.Document, normalized json.RawMessage, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(registry, w, document, http.StatusBadRequest)
	if !ok {
		return
	}
	writeJSONResponse(w, applyDraftConfigResponse{
		Document:         document,
		NormalizedConfig: normalized,
		PreviewHTML:      previewHTML,
		Library:          library,
	})
}

func writeRenderedDraftCard(registry *cardcomponent.Registry, w http.ResponseWriter, document cardcomponent.Document, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(registry, w, document, http.StatusInternalServerError)
	if !ok {
		return
	}
	writeJSONResponse(w, renderedDraftCardResponse{
		Document:    document,
		PreviewHTML: previewHTML,
		Library:     library,
	})
}

func writeInteractiveDraftCard(registry *cardcomponent.Registry, w http.ResponseWriter, document cardcomponent.Document, gameState GameState, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(registry, w, document, http.StatusInternalServerError)
	if !ok {
		return
	}
	writeJSONResponse(w, interactiveDraftCardResponse{
		Document:             document,
		GameState:            gameState,
		PreviewHTML:          previewHTML,
		AvailableConfigKinds: append([]string(nil), gameState.UnlockedConfigKinds...),
		AvailableComponents:  availableComponents(registry, gameState, document),
		Overlay:              buildOverlay(registry, document, gameState, gameState.SelectedComponentID),
		Library:              library,
	})
}

func writeTappedDraftCard(registry *cardcomponent.Registry, w http.ResponseWriter, result tapResult) {
	previewHTML, ok := renderDraftPreview(registry, w, result.document, http.StatusBadRequest)
	if !ok {
		return
	}
	writeJSONResponse(w, tapDraftCardResponse{
		Document:      result.document,
		GameState:     result.gameState,
		AppliedConfig: result.appliedConfig,
		PreviewHTML:   previewHTML,
		Events:        nonNilCardEvents(result.events),
		Overlay:       result.overlay,
		Library:       result.library,
	})
}

func nonNilCardEvents(events []CardEvent) []CardEvent {
	if len(events) == 0 {
		return []CardEvent{}
	}
	return events
}

func renderDraftPreview(registry *cardcomponent.Registry, w http.ResponseWriter, document cardcomponent.Document, status int) (string, bool) {
	preview, err := cardcomponent.RenderDocument(document, registry)
	if err != nil {
		http.Error(w, err.Error(), status)
		return "", false
	}
	return preview.Render(), true
}

func applyGeneratedConfigToDocument(registry *cardcomponent.Registry, raw json.RawMessage, document *cardcomponent.Document) (json.RawMessage, cardcomponent.LibraryItem, string, error) {
	return applyGeneratedConfigToDocumentForComponent(registry, raw, document, "")
}

func applyGeneratedConfigToDocumentForComponent(registry *cardcomponent.Registry, raw json.RawMessage, document *cardcomponent.Document, componentID string) (json.RawMessage, cardcomponent.LibraryItem, string, error) {
	var envelope schema.GeneratedConfigEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, cardcomponent.LibraryItem{}, "", design.NewInvalidModelOutputError(string(raw), []schema.Issue{{
			Path:    "generated_config",
			Code:    "invalid_json",
			Message: "generated_config must be one JSON object: " + err.Error(),
		}}, design.ErrInvalidModelOutput)
	}
	definition, ok := registry.Lookup(strings.TrimSpace(envelope.ComponentKind))
	if !ok {
		return nil, cardcomponent.LibraryItem{}, "", design.NewInvalidModelOutputError(string(raw), []schema.Issue{{
			Path:    "component_kind",
			Code:    "invalid_component_kind",
			Message: "component_kind is not registered",
			Actual:  envelope.ComponentKind,
		}}, design.ErrInvalidModelOutput)
	}
	generation, ok := definition.Generation()
	if !ok {
		return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("component %q does not accept generated configs", definition.Kind())
	}
	canonical, issues := generation.CanonicalizeEnvelope(raw)
	if len(issues) > 0 {
		return nil, cardcomponent.LibraryItem{}, "", design.NewInvalidModelOutputError(string(raw), issues, design.ErrInvalidModelOutput)
	}
	if err := json.Unmarshal(canonical, &envelope); err != nil {
		return nil, cardcomponent.LibraryItem{}, "", err
	}
	configRaw := envelope.Config
	replaced := false
	if strings.TrimSpace(componentID) != "" {
		replaced = replaceComponentConfigByID(&document.Root, componentID, definition.Kind(), configRaw)
	} else {
		replaced = replaceComponentConfig(&document.Root, definition.Kind(), configRaw)
	}
	if !replaced {
		return nil, cardcomponent.LibraryItem{}, "", design.NewInvalidModelOutputError(string(raw), []schema.Issue{{
			Path:    "component_kind",
			Code:    "missing_component",
			Message: "selected component is not present in this card",
			Actual:  definition.Kind(),
		}}, design.ErrInvalidModelOutput)
	}
	return canonical, cardcomponent.LibraryItem{
		ID:            "applied-" + definition.Kind(),
		Name:          fallbackLibraryName(definition.Kind(), envelope.Description),
		ComponentKind: definition.Kind(),
		Description:   envelope.Description,
		Config:        configRaw,
	}, definition.Kind(), nil
}

func replaceComponentConfigByID(node *cardcomponent.Node, componentID, target string, raw json.RawMessage) bool {
	if node == nil {
		return false
	}
	if node.ID == componentID {
		if node.ComponentKind != target {
			return false
		}
		node.Config = append(json.RawMessage(nil), raw...)
		return true
	}
	for index := range node.Children {
		if replaceComponentConfigByID(&node.Children[index], componentID, target, raw) {
			return true
		}
	}
	return false
}

func fallbackLibraryName(target, description string) string {
	if strings.TrimSpace(description) != "" {
		return strings.TrimSpace(description)
	}
	return fmt.Sprintf("Saved %s", target)
}

func replaceComponentConfig(node *cardcomponent.Node, target string, raw json.RawMessage) bool {
	if node == nil {
		return false
	}
	if node.ComponentKind == target {
		node.Config = append(json.RawMessage(nil), raw...)
		return true
	}
	for index := range node.Children {
		if replaceComponentConfig(&node.Children[index], target, raw) {
			return true
		}
	}
	return false
}

func generateDraftConfig(w http.ResponseWriter, r *http.Request, deps Dependencies, instruction, oldCode, componentID string, definition cardcomponent.Definition) {
	service, err := design.NewService(deps.Patch, deps.PatchModel, definition)
	if err != nil {
		writeConfigError(w, err)
		return
	}
	response, err := service.Generate(r.Context(), design.GenerateRequest{
		Instruction: instruction,
		OldCode:     oldCode,
		ComponentID: componentID,
	})
	if err != nil {
		writeConfigError(w, err)
		return
	}
	writeJSONResponse(w, response)
}

func writeApplyConfigError(w http.ResponseWriter, err error, status int) {
	raw, _ := design.RawModelOutput(err)
	issues := design.Issues(err)
	if len(issues) == 0 && raw == "" {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Message     string         `json:"message"`
		RawResponse string         `json:"raw_response,omitempty"`
		Issues      []schema.Issue `json:"issues,omitempty"`
	}{
		Message:     err.Error(),
		RawResponse: raw,
		Issues:      issues,
	})
}

func writeConfigError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, design.ErrEmptyInstruction):
		status = http.StatusBadRequest
	case errors.Is(err, design.ErrInvalidModelOutput):
		status = http.StatusBadGateway
	}
	raw, hasRaw := design.RawModelOutput(err)
	issues := design.Issues(err)
	if hasRaw || len(issues) > 0 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(struct {
			Message     string         `json:"message"`
			RawResponse string         `json:"raw_response,omitempty"`
			Issues      []schema.Issue `json:"issues,omitempty"`
		}{
			Message:     err.Error(),
			RawResponse: raw,
			Issues:      issues,
		})
		return
	}
	http.Error(w, err.Error(), status)
}

func writeJSONResponse(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		http.Error(w, "failed to write json response", http.StatusInternalServerError)
	}
}
