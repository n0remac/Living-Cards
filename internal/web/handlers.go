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
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/fragment"
)

type Dependencies struct {
	Patch      fragment.ChatClient
	PatchModel string
}

func Register(mux *http.ServeMux, deps Dependencies) {
	state := newDesignerState()
	mux.HandleFunc("/", pageHandler())
	mux.HandleFunc("/api/", http.NotFound)
	mux.HandleFunc("/assets/", frontendAssetHandler())
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
		if path == "apply-fragment" {
			applyDraftFragmentHandler(w, r, state)
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
		if len(parts) != 2 || parts[0] != "fragments" {
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
			if current, componentID := state.currentFragment(parts[1]); current != "" {
				request.OldCode = current
				if request.ComponentID == "" {
					request.ComponentID = componentID
				}
			}
		}
		switch parts[1] {
		case background.Type:
			generateDraftFragment(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, background.Spec())
		case border.Type:
			generateDraftFragment(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, border.Spec())
		case textarea.Type:
			generateDraftFragment(w, r, deps, request.Instruction, request.OldCode, request.ComponentID, textarea.Spec())
		default:
			http.NotFound(w, r)
		}
	}
}

type applyDraftFragmentRequest struct {
	GeneratedFragment json.RawMessage `json:"generated_fragment"`
}

type applyDraftFragmentResponse struct {
	Document           cardcomponent.Document      `json:"document"`
	NormalizedFragment any                         `json:"normalized_fragment"`
	PreviewHTML        string                      `json:"preview_html"`
	Library            []cardcomponent.LibraryItem `json:"library"`
}

type renderedDraftCardResponse struct {
	Document    cardcomponent.Document      `json:"document"`
	PreviewHTML string                      `json:"preview_html"`
	Library     []cardcomponent.LibraryItem `json:"library"`
}

type interactiveDraftCardResponse struct {
	Document            cardcomponent.Document      `json:"document"`
	GameState           GameState                   `json:"gameState"`
	PreviewHTML         string                      `json:"preview_html"`
	AvailableTargets    []string                    `json:"availableTargets"`
	AvailableComponents []ComponentDescriptor       `json:"availableComponents"`
	Overlay             *ComponentOverlay           `json:"overlay,omitempty"`
	Library             []cardcomponent.LibraryItem `json:"library"`
}

type tapDraftCardRequest struct {
	Target string  `json:"target"`
	Zone   string  `json:"zone"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
}

type controlChangeDraftCardRequest struct {
	ComponentID    string          `json:"componentId,omitempty"`
	Trait          string          `json:"trait,omitempty"`
	Control        string          `json:"control,omitempty"`
	Value          json.RawMessage `json:"value,omitempty"`
	Target         string          `json:"target"`
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

type tapDraftCardResponse struct {
	Document        cardcomponent.Document      `json:"document"`
	GameState       GameState                   `json:"gameState"`
	AppliedFragment any                         `json:"appliedFragment,omitempty"`
	PreviewHTML     string                      `json:"preview_html"`
	Events          []CardEvent                 `json:"events"`
	Overlay         *ComponentOverlay           `json:"overlay,omitempty"`
	Library         []cardcomponent.LibraryItem `json:"library"`
}

type libraryResponse struct {
	Item    cardcomponent.LibraryItem   `json:"item,omitempty"`
	Library []cardcomponent.LibraryItem `json:"library"`
}

func applyDraftFragmentHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request applyDraftFragmentRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(request.GeneratedFragment) == 0 {
		writeApplyFragmentError(w, fragment.NewInvalidModelOutputError("", []fragment.Issue{{
			Path:    "generated_fragment",
			Code:    "required",
			Message: "generated_fragment is required",
		}}, fragment.ErrInvalidModelOutput), http.StatusBadRequest)
		return
	}
	if err := validateGeneratedFragmentEnvelope(request.GeneratedFragment); err != nil {
		writeApplyFragmentError(w, err, http.StatusBadRequest)
		return
	}
	document, normalized, err := state.apply(request.GeneratedFragment)
	if err != nil {
		writeApplyFragmentError(w, err, http.StatusBadRequest)
		return
	}
	_, library := state.snapshot()
	writeAppliedDraftFragment(w, document, normalized, library)
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
	result, err := state.tap(request.Target, request.Zone, request.X, request.Y)
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
	result, err := state.applyColorControl(request.Target, colorControlRequest{
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

func validateGeneratedFragmentEnvelope(raw json.RawMessage) error {
	var envelope struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fragment.NewInvalidModelOutputError(string(raw), []fragment.Issue{{
			Path:    "generated_fragment",
			Code:    "invalid_json",
			Message: "generated_fragment must be one JSON object: " + err.Error(),
		}}, fragment.ErrInvalidModelOutput)
	}
	switch strings.TrimSpace(envelope.Target) {
	case background.Type, border.Type, textarea.Type, shape.Type:
		return nil
	default:
		return fragment.NewInvalidModelOutputError(string(raw), []fragment.Issue{{
			Path:    "target",
			Code:    "invalid_target",
			Message: "target must be background, border, textarea, or shape",
			Actual:  envelope.Target,
			Allowed: []string{background.Type, border.Type, textarea.Type, shape.Type},
		}}, fragment.ErrInvalidModelOutput)
	}
}

func designLibraryHandler(w http.ResponseWriter, r *http.Request, state *designerState) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSONResponse(w, libraryResponse{
		Library: state.libraryForTarget(r.URL.Query().Get("target")),
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
	writeAppliedDraftFragment(w, document, normalized, library)
}

func writeAppliedDraftFragment(w http.ResponseWriter, document cardcomponent.Document, normalized any, library []cardcomponent.LibraryItem) {
	previewHTML, ok := renderDraftPreview(w, document, http.StatusBadRequest)
	if !ok {
		return
	}
	writeJSONResponse(w, applyDraftFragmentResponse{
		Document:           document,
		NormalizedFragment: normalized,
		PreviewHTML:        previewHTML,
		Library:            library,
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
		Document:            document,
		GameState:           gameState,
		PreviewHTML:         previewHTML,
		AvailableTargets:    append([]string(nil), gameState.UnlockedTargets...),
		AvailableComponents: availableComponents(gameState, document),
		Overlay:             buildOverlay(document, gameState, gameState.SelectedComponentID),
		Library:             library,
	})
}

func writeTappedDraftCard(w http.ResponseWriter, result tapResult) {
	previewHTML, ok := renderDraftPreview(w, result.document, http.StatusBadRequest)
	if !ok {
		return
	}
	writeJSONResponse(w, tapDraftCardResponse{
		Document:        result.document,
		GameState:       result.gameState,
		AppliedFragment: result.appliedFragment,
		PreviewHTML:     previewHTML,
		Events:          result.events,
		Overlay:         result.overlay,
		Library:         result.library,
	})
}

func renderDraftPreview(w http.ResponseWriter, document cardcomponent.Document, status int) (string, bool) {
	preview, err := cardcomponent.RenderDocument(document, cardComponentRegistry())
	if err != nil {
		http.Error(w, err.Error(), status)
		return "", false
	}
	return preview.Render(), true
}

func applyGeneratedFragmentToDocument(raw json.RawMessage, document *cardcomponent.Document) (any, cardcomponent.LibraryItem, error) {
	var envelope struct {
		Target string `json:"target"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, cardcomponent.LibraryItem{}, fragment.NewInvalidModelOutputError(string(raw), []fragment.Issue{{
			Path:    "generated_fragment",
			Code:    "invalid_json",
			Message: "generated_fragment must be one JSON object: " + err.Error(),
		}}, fragment.ErrInvalidModelOutput)
	}
	switch strings.TrimSpace(envelope.Target) {
	case background.Type:
		return applyDraftFragment(raw, background.Spec(), document)
	case border.Type:
		return applyDraftFragment(raw, border.Spec(), document)
	case textarea.Type:
		return applyDraftFragment(raw, textarea.Spec(), document)
	case shape.Type:
		return applyDraftFragment(raw, shape.Spec(), document)
	default:
		return nil, cardcomponent.LibraryItem{}, fragment.NewInvalidModelOutputError(string(raw), []fragment.Issue{{
			Path:    "target",
			Code:    "invalid_target",
			Message: "target must be background, border, textarea, or shape",
			Actual:  envelope.Target,
			Allowed: []string{background.Type, border.Type, textarea.Type, shape.Type},
		}}, fragment.ErrInvalidModelOutput)
	}
}

func applyDraftFragment[T any](raw json.RawMessage, spec fragment.Spec[T], document *cardcomponent.Document) (fragment.Generated[T], cardcomponent.LibraryItem, error) {
	generated, err := fragment.DecodeNormalizeValidate[T](string(raw), spec)
	if err != nil {
		return fragment.Generated[T]{}, cardcomponent.LibraryItem{}, err
	}
	fragmentRaw, err := json.Marshal(generated.Fragment)
	if err != nil {
		return fragment.Generated[T]{}, cardcomponent.LibraryItem{}, err
	}
	if !replaceComponentFragment(&document.Root, spec.Target, fragmentRaw) {
		return fragment.Generated[T]{}, cardcomponent.LibraryItem{}, fragment.NewInvalidModelOutputError(string(raw), []fragment.Issue{{
			Path:    "target",
			Code:    "missing_component",
			Message: "selected component is not present in this card",
			Actual:  spec.Target,
		}}, fragment.ErrInvalidModelOutput)
	}
	return generated, cardcomponent.LibraryItem{
		ID:          "applied-" + spec.Target,
		Name:        fallbackLibraryName(generated.Target, generated.Description),
		Target:      spec.Target,
		Description: generated.Description,
		Fragment:    fragmentRaw,
	}, nil
}

func fallbackLibraryName(target, description string) string {
	if strings.TrimSpace(description) != "" {
		return strings.TrimSpace(description)
	}
	return fmt.Sprintf("Saved %s", target)
}

func replaceComponentFragment(node *cardcomponent.Node, target string, raw json.RawMessage) bool {
	if node == nil {
		return false
	}
	if node.Type == target {
		node.Fragment = append(json.RawMessage(nil), raw...)
		return true
	}
	for index := range node.Children {
		if replaceComponentFragment(&node.Children[index], target, raw) {
			return true
		}
	}
	return false
}

func generateDraftFragment[T any](w http.ResponseWriter, r *http.Request, deps Dependencies, instruction, oldCode, componentID string, spec fragment.Spec[T]) {
	service := fragment.NewService(deps.Patch, deps.PatchModel, spec)
	response, err := service.Generate(r.Context(), fragment.GenerateRequest{
		Instruction: instruction,
		OldCode:     oldCode,
		ComponentID: componentID,
	})
	if err != nil {
		writeFragmentError(w, err)
		return
	}
	writeJSONResponse(w, response)
}

func writeApplyFragmentError(w http.ResponseWriter, err error, status int) {
	raw, _ := fragment.RawModelOutput(err)
	issues := fragment.Issues(err)
	if len(issues) == 0 && raw == "" {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Message     string           `json:"message"`
		RawResponse string           `json:"raw_response,omitempty"`
		Issues      []fragment.Issue `json:"issues,omitempty"`
	}{
		Message:     err.Error(),
		RawResponse: raw,
		Issues:      issues,
	})
}

func writeFragmentError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, fragment.ErrEmptyInstruction):
		status = http.StatusBadRequest
	case errors.Is(err, fragment.ErrInvalidModelOutput):
		status = http.StatusBadGateway
	}
	raw, hasRaw := fragment.RawModelOutput(err)
	issues := fragment.Issues(err)
	if hasRaw || len(issues) > 0 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(struct {
			Message     string           `json:"message"`
			RawResponse string           `json:"raw_response,omitempty"`
			Issues      []fragment.Issue `json:"issues,omitempty"`
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
