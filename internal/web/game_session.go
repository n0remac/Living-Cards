package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"unicode"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/game"
)

type RenderedGameCard struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Tags        []string               `json:"tags,omitempty"`
	Collectible bool                   `json:"collectible"`
	Collected   bool                   `json:"collected,omitempty"`
	State       map[string]any         `json:"state,omitempty"`
	Document    cardcomponent.Document `json:"document"`
	PreviewHTML string                 `json:"preview_html"`
}

type GameSessionSnapshot struct {
	WorldDeck                []RenderedGameCard       `json:"worldDeck"`
	ActiveWorldCard          RenderedGameCard         `json:"activeWorldCard"`
	ActiveWorldCardID        string                   `json:"activeWorldCardId"`
	ActiveIndex              int                      `json:"activeIndex"`
	ActiveEditingComponentID string                   `json:"activeEditingComponentId,omitempty"`
	ActiveEditingOverlay     *ComponentOverlay        `json:"activeEditingOverlay,omitempty"`
	Library                  []RenderedGameCard       `json:"library"`
	EditSession              *RenderedGameEditSession `json:"editSession,omitempty"`
	SolvedFlags              map[string]bool          `json:"solvedFlags"`
	Message                  string                   `json:"message,omitempty"`
}

type RenderedGameEditSession struct {
	TargetCardID                string            `json:"targetCardId"`
	DraftCard                   RenderedGameCard  `json:"draftCard"`
	PendingConsumedComponentIDs []string          `json:"pendingConsumedComponentIds,omitempty"`
	SelectedComponentID         string            `json:"selectedComponentId,omitempty"`
	EditingOverlay              *ComponentOverlay `json:"editingOverlay,omitempty"`
}

func gameResourceHandler(registry *cardcomponent.Registry, state *game.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/game"), "/")
		switch path {
		case "session":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := state.Snapshot()
			writeGameSnapshot(registry, w, snapshot, err)
		case "reset":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := state.Reset()
			writeGameSnapshot(registry, w, snapshot, err)
		case "cycle":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				Direction string `json:"direction"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.Cycle(request.Direction)
			writeGameSnapshot(registry, w, snapshot, err)
		case "collect":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID string `json:"cardId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.Collect(request.CardID)
			writeGameSnapshot(registry, w, snapshot, err)
		case "play-card":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				SourceCardID string `json:"sourceCardId"`
				TargetCardID string `json:"targetCardId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.UseCard(request.SourceCardID, request.TargetCardID)
			writeGameSnapshot(registry, w, snapshot, err)
		case "submit-form":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
			var request struct {
				CardID string            `json:"cardId"`
				FormID string            `json:"formId"`
				Fields map[string]string `json:"fields"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.SubmitForm(request.CardID, request.FormID, request.Fields)
			writeGameSnapshot(registry, w, snapshot, err)
		case "component/select":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID        string `json:"cardId"`
				ComponentID   string `json:"componentId"`
				ComponentKind string `json:"componentKind"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.SelectWorldComponent(request.CardID, request.ComponentID, request.ComponentKind)
			writeGameSnapshot(registry, w, snapshot, err)
		case "component/control-change":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID        string          `json:"cardId"`
				ComponentID   string          `json:"componentId"`
				ComponentKind string          `json:"componentKind"`
				Control       string          `json:"control"`
				Value         json.RawMessage `json:"value"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.ApplyWorldComponentControl(request.CardID, request.ComponentID, request.ComponentKind, request.Control, request.Value)
			writeGameSnapshot(registry, w, snapshot, err)
		case "edit/start":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID string `json:"cardId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.StartEdit(request.CardID)
			writeGameSnapshot(registry, w, snapshot, err)
		case "edit/install-component":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				ComponentCardID string `json:"componentCardId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.InstallEditComponent(request.ComponentCardID)
			writeGameSnapshot(registry, w, snapshot, err)
		case "edit/component/select":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				ComponentID   string `json:"componentId"`
				ComponentKind string `json:"componentKind"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.SelectEditComponent(request.ComponentID, request.ComponentKind)
			writeGameSnapshot(registry, w, snapshot, err)
		case "edit/control-change":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				ComponentID string          `json:"componentId"`
				Control     string          `json:"control"`
				Value       json.RawMessage `json:"value"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.ApplyEditControl(request.ComponentID, request.Control, request.Value)
			writeGameSnapshot(registry, w, snapshot, err)
		case "library/component/control-change":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID        string          `json:"cardId"`
				ComponentID   string          `json:"componentId"`
				ComponentKind string          `json:"componentKind"`
				Control       string          `json:"control"`
				Value         json.RawMessage `json:"value"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			snapshot, err := state.ApplyLibraryComponentControl(request.CardID, request.ComponentID, request.ComponentKind, request.Control, request.Value)
			writeGameSnapshot(registry, w, snapshot, err)
		case "edit/save":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := state.SaveEdit()
			writeGameSnapshot(registry, w, snapshot, err)
		case "edit/cancel":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := state.CancelEdit()
			writeGameSnapshot(registry, w, snapshot, err)
		default:
			http.NotFound(w, r)
		}
	}
}

func writeGameSnapshot(registry *cardcomponent.Registry, w http.ResponseWriter, snapshot game.Snapshot, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	response, err := renderGameSessionSnapshot(registry, snapshot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, response)
}

func renderGameSessionSnapshot(registry *cardcomponent.Registry, snapshot game.Snapshot) (GameSessionSnapshot, error) {
	worldDeck, err := renderWorldGameCards(registry, snapshot.WorldDeck)
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	activeWorldCard, err := renderGameCard(registry, snapshot.ActiveWorldCard, "game-world-"+safeDOMID(snapshot.ActiveWorldCard.ID))
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	library, err := renderLibraryGameCards(registry, snapshot.Library)
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	var editSession *RenderedGameEditSession
	if snapshot.EditSession != nil {
		rendered, err := renderGameCard(registry, snapshot.EditSession.DraftCard, "game-edit-"+safeDOMID(snapshot.EditSession.DraftCard.ID))
		if err != nil {
			return GameSessionSnapshot{}, err
		}
		editSession = &RenderedGameEditSession{
			TargetCardID:                snapshot.EditSession.TargetCardID,
			DraftCard:                   rendered,
			PendingConsumedComponentIDs: append([]string(nil), snapshot.EditSession.PendingConsumedComponentIDs...),
			SelectedComponentID:         snapshot.EditSession.SelectedComponentID,
			EditingOverlay:              gameEditingOverlay(registry, snapshot.EditSession.DraftCard, snapshot.EditSession.SelectedComponentID),
		}
	}
	return GameSessionSnapshot{
		WorldDeck:                worldDeck,
		ActiveWorldCard:          activeWorldCard,
		ActiveWorldCardID:        snapshot.ActiveWorldCardID,
		ActiveIndex:              snapshot.ActiveIndex,
		ActiveEditingComponentID: snapshot.ActiveEditingComponentID,
		ActiveEditingOverlay:     gameActiveEditingOverlay(registry, snapshot.ActiveWorldCard, snapshot.ActiveEditingComponentID, snapshot.Library),
		Library:                  library,
		EditSession:              editSession,
		SolvedFlags:              cloneValue(snapshot.SolvedFlags),
		Message:                  snapshot.Message,
	}, nil
}

func renderWorldGameCards(registry *cardcomponent.Registry, cards []game.Card) ([]RenderedGameCard, error) {
	out := make([]RenderedGameCard, 0, len(cards))
	for _, card := range cards {
		rendered, err := renderGameCard(registry, card, "game-world-"+safeDOMID(card.ID))
		if err != nil {
			return nil, err
		}
		out = append(out, rendered)
	}
	return out, nil
}

func renderLibraryGameCards(registry *cardcomponent.Registry, cards []game.Card) ([]RenderedGameCard, error) {
	out := make([]RenderedGameCard, 0, len(cards))
	for index, card := range cards {
		prefix := fmt.Sprintf("game-library-%d-%s", index, safeDOMID(card.ID))
		rendered, err := renderGameCard(registry, card, prefix)
		if err != nil {
			return nil, err
		}
		out = append(out, rendered)
	}
	return out, nil
}

func renderGameCard(registry *cardcomponent.Registry, card game.Card, domIDPrefix string) (RenderedGameCard, error) {
	preview, err := cardcomponent.RenderDocumentWithOptions(card.Document, registry, cardcomponent.RenderOptions{
		ElementID:   domIDPrefix,
		DOMIDPrefix: domIDPrefix,
	})
	if err != nil {
		return RenderedGameCard{}, err
	}
	return RenderedGameCard{
		ID:          card.ID,
		Name:        card.Name,
		Kind:        card.Kind,
		Tags:        append([]string(nil), card.Tags...),
		Collectible: card.Collectible,
		Collected:   card.Collected,
		State:       cloneValue(card.State),
		Document:    cloneValue(card.Document),
		PreviewHTML: preview.Render(),
	}, nil
}

func safeDOMID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	for _, char := range value {
		switch {
		case unicode.IsLetter(char), unicode.IsDigit(char):
			out.WriteRune(char)
		case char == '-', char == '_':
			out.WriteRune(char)
		case unicode.IsSpace(char):
			out.WriteRune('-')
		}
	}
	if out.Len() == 0 {
		return "card"
	}
	return html.EscapeString(out.String())
}
