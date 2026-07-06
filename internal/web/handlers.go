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

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/design"
	"github.com/n0remac/Living-Card/internal/game"
)

type Dependencies struct {
	Patch      design.ChatClient
	PatchModel string
}

func Register(mux *http.ServeMux, deps Dependencies) {
	state := newDesignerState()
	gameSession := game.NewSession()
	mux.HandleFunc("/", pageHandler())
	mux.HandleFunc("/api/", http.NotFound)
	mux.HandleFunc("/assets/", frontendAssetHandler())
	mux.HandleFunc("/api/game/", gameResourceHandler(gameSession))
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
			writeRenderedDraftCard(w, document, library)
			return
		}
		if path == "interactive" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			document, gameState, library := state.interactiveSnapshot()
			writeInteractiveDraftCard(w, document, gameState, library)
			return
		}
		if path == "reset" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			document, library := state.reset()
			writeRenderedDraftCard(w, document, library)
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
		switch parts[1] {
		case background.Kind:
			generateDraftConfig(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, background.Spec())
		case border.Kind:
			generateDraftConfig(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, border.Spec())
		case textarea.Kind:
			generateDraftConfig(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, textarea.Spec())
		case imagecomponent.Kind:
			generateDraftConfig(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, imagecomponent.Spec())
		default:
			http.NotFound(w, r)
		}
	}
}

type applyDraftConfigRequest struct {
	GeneratedConfig json.RawMessage `json:"generated_config"`
	ComponentID     string          `json:"component_id,omitempty"`
}

type applyDraftConfigResponse struct {
	Document         cardcomponent.Document      `json:"document"`
	NormalizedConfig any                         `json:"normalized_config"`
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
	ComponentID    string          `json:"componentId,omitempty"`
	Trait          string          `json:"trait,omitempty"`
	Control        string          `json:"control,omitempty"`
	Value          json.RawMessage `json:"value,omitempty"`
	ComponentKind  string          `json:"componentKind"`
	Color          string          `json:"color"`
	SecondaryColor string          `json:"secondaryColor,omitempty"`
	Gradient       bool            `json:"gradient,omitempty"`
	Angle          int             `json:"angle,omitempty"`
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
	AppliedConfig any                         `json:"appliedConfig,omitempty"`
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
		writeApplyConfigError(w, design.NewInvalidModelOutputError("", []design.Issue{{
			Path:    "generated_config",
			Code:    "required",
			Message: "generated_config is required",
		}}, design.ErrInvalidModelOutput), http.StatusBadRequest)
		return
	}
	if err := validateGeneratedConfigEnvelope(request.GeneratedConfig); err != nil {
		writeApplyConfigError(w, err, http.StatusBadRequest)
		return
	}
	document, normalized, err := state.apply(request.GeneratedConfig, request.ComponentID)
	if err != nil {
		writeApplyConfigError(w, err, http.StatusBadRequest)
		return
	}
	_, library := state.snapshot()
	writeAppliedDraftConfig(w, document, normalized, library)
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
	writeTappedDraftCard(w, result)
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
	writeTappedDraftCard(w, result)
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
	if strings.TrimSpace(request.ComponentID) != "" || strings.TrimSpace(request.Control) != "" || len(request.Value) > 0 {
		result, err := state.applyControlChange(request.ComponentID, request.Trait, request.Control, request.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeTappedDraftCard(w, result)
		return
	}
	result, err := state.applyColorControl(request.ComponentKind, colorControlRequest{
		Color:          request.Color,
		SecondaryColor: request.SecondaryColor,
		Gradient:       request.Gradient,
		Angle:          request.Angle,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTappedDraftCard(w, result)
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
	writeTappedDraftCard(w, result)
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
	writeTappedDraftCard(w, result)
}

func validateGeneratedConfigEnvelope(raw json.RawMessage) error {
	var envelope struct {
		ComponentKind string `json:"componentKind"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return design.NewInvalidModelOutputError(string(raw), []design.Issue{{
			Path:    "generated_config",
			Code:    "invalid_json",
			Message: "generated_config must be one JSON object: " + err.Error(),
		}}, design.ErrInvalidModelOutput)
	}
	switch strings.TrimSpace(envelope.ComponentKind) {
	case background.Kind, border.Kind, textarea.Kind, shape.Kind, imagecomponent.Kind:
		return nil
	default:
		return design.NewInvalidModelOutputError(string(raw), []design.Issue{{
			Path:    "componentKind",
			Code:    "invalid_component_kind",
			Message: "componentKind must be background, border, textarea, shape, or image",
			Actual:  envelope.ComponentKind,
			Allowed: []string{background.Kind, border.Kind, textarea.Kind, shape.Kind, imagecomponent.Kind},
		}}, design.ErrInvalidModelOutput)
	}
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
	writeAppliedDraftConfig(w, document, normalized, library)
}

func writeAppliedDraftConfig(w http.ResponseWriter, document cardcomponent.Document, normalized any, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(w, document, http.StatusBadRequest)
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

func writeRenderedDraftCard(w http.ResponseWriter, document cardcomponent.Document, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(w, document, http.StatusInternalServerError)
	if !ok {
		return
	}
	writeJSONResponse(w, renderedDraftCardResponse{
		Document:    document,
		PreviewHTML: previewHTML,
		Library:     library,
	})
}

func writeInteractiveDraftCard(w http.ResponseWriter, document cardcomponent.Document, gameState GameState, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(w, document, http.StatusInternalServerError)
	if !ok {
		return
	}
	writeJSONResponse(w, interactiveDraftCardResponse{
		Document:             document,
		GameState:            gameState,
		PreviewHTML:          previewHTML,
		AvailableConfigKinds: append([]string(nil), gameState.UnlockedConfigKinds...),
		AvailableComponents:  availableComponents(gameState, document),
		Overlay:              buildOverlay(document, gameState, gameState.SelectedComponentID),
		Library:              library,
	})
}

func writeTappedDraftCard(w http.ResponseWriter, result tapResult) {
	previewHTML, ok := renderDraftPreview(w, result.document, http.StatusBadRequest)
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

func renderDraftPreview(w http.ResponseWriter, document cardcomponent.Document, status int) (string, bool) {
	preview, err := cardcomponent.RenderDocument(document, cardComponentRegistry())
	if err != nil {
		http.Error(w, err.Error(), status)
		return "", false
	}
	return preview.Render(), true
}

func applyGeneratedConfigToDocument(raw json.RawMessage, document *cardcomponent.Document) (any, cardcomponent.LibraryItem, error) {
	return applyGeneratedConfigToDocumentForComponent(raw, document, "")
}

func applyGeneratedConfigToDocumentForComponent(raw json.RawMessage, document *cardcomponent.Document, componentID string) (any, cardcomponent.LibraryItem, error) {
	var envelope struct {
		ComponentKind  string `json:"componentKind"`
		ComponentID    string `json:"component_id"`
		ComponentIDAlt string `json:"componentId"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, cardcomponent.LibraryItem{}, design.NewInvalidModelOutputError(string(raw), []design.Issue{{
			Path:    "generated_config",
			Code:    "invalid_json",
			Message: "generated_config must be one JSON object: " + err.Error(),
		}}, design.ErrInvalidModelOutput)
	}
	if strings.TrimSpace(componentID) == "" {
		componentID = strings.TrimSpace(firstNonEmpty(envelope.ComponentID, envelope.ComponentIDAlt))
	}
	switch strings.TrimSpace(envelope.ComponentKind) {
	case background.Kind:
		return applyDraftConfig(raw, background.Spec(), document, componentID)
	case border.Kind:
		return applyDraftConfig(raw, border.Spec(), document, componentID)
	case textarea.Kind:
		return applyDraftConfig(raw, textarea.Spec(), document, componentID)
	case shape.Kind:
		return applyDraftConfig(raw, shape.Spec(), document, componentID)
	case imagecomponent.Kind:
		return applyDraftConfig(raw, imagecomponent.Spec(), document, componentID)
	default:
		return nil, cardcomponent.LibraryItem{}, design.NewInvalidModelOutputError(string(raw), []design.Issue{{
			Path:    "componentKind",
			Code:    "invalid_component_kind",
			Message: "componentKind must be background, border, textarea, shape, or image",
			Actual:  envelope.ComponentKind,
			Allowed: []string{background.Kind, border.Kind, textarea.Kind, shape.Kind, imagecomponent.Kind},
		}}, design.ErrInvalidModelOutput)
	}
}

func applyDraftConfig[T any](raw json.RawMessage, spec design.Spec[T], document *cardcomponent.Document, componentID string) (design.GeneratedConfig[T], cardcomponent.LibraryItem, error) {
	decodeRaw, err := generatedConfigDecodeRaw(raw)
	if err != nil {
		return design.GeneratedConfig[T]{}, cardcomponent.LibraryItem{}, err
	}
	generated, err := design.DecodeNormalizeValidateConfig[T](string(decodeRaw), spec)
	if err != nil {
		return design.GeneratedConfig[T]{}, cardcomponent.LibraryItem{}, err
	}
	configRaw, err := json.Marshal(generated.Config)
	if err != nil {
		return design.GeneratedConfig[T]{}, cardcomponent.LibraryItem{}, err
	}
	replaced := false
	if strings.TrimSpace(componentID) != "" {
		replaced = replaceComponentConfigByID(&document.Root, componentID, spec.ComponentKind, configRaw)
	} else {
		replaced = replaceComponentConfig(&document.Root, spec.ComponentKind, configRaw)
	}
	if !replaced {
		return design.GeneratedConfig[T]{}, cardcomponent.LibraryItem{}, design.NewInvalidModelOutputError(string(raw), []design.Issue{{
			Path:    "componentKind",
			Code:    "missing_component",
			Message: "selected component is not present in this card",
			Actual:  spec.ComponentKind,
		}}, design.ErrInvalidModelOutput)
	}
	return generated, cardcomponent.LibraryItem{
		ID:            "applied-" + spec.ComponentKind,
		Name:          fallbackLibraryName(generated.ComponentKind, generated.Description),
		ComponentKind: spec.ComponentKind,
		Description:   generated.Description,
		Config:        configRaw,
	}, nil
}

func generatedConfigDecodeRaw(raw json.RawMessage) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	delete(envelope, "component_id")
	delete(envelope, "componentId")
	return json.Marshal(envelope)
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

func generateDraftConfig[T any](w http.ResponseWriter, r *http.Request, deps Dependencies, instruction, oldCode, componentID string, spec design.Spec[T]) {
	service := design.NewService(deps.Patch, deps.PatchModel, spec)
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
		Issues      []design.Issue `json:"issues,omitempty"`
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
			Issues      []design.Issue `json:"issues,omitempty"`
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
