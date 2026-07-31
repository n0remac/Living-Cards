package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
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
	Actor       *game.ActorState       `json:"actor,omitempty"`
	PreviewHTML string                 `json:"preview_html"`
}

type RenderedEncounterParticipant struct {
	Role string           `json:"role"`
	Card RenderedGameCard `json:"card"`
}

type RenderedEncounterSnapshot struct {
	ID           string                         `json:"id"`
	Phase        string                         `json:"phase"`
	Participants []RenderedEncounterParticipant `json:"participants"`
	Pressure     int                            `json:"pressure"`
	MaxPressure  int                            `json:"maxPressure,omitempty"`
	Outcome      string                         `json:"outcome,omitempty"`
}

type GameSessionSnapshot struct {
	WorldDeck                []RenderedGameCard         `json:"worldDeck"`
	ActiveWorldCard          RenderedGameCard           `json:"activeWorldCard"`
	ActiveWorldCardID        string                     `json:"activeWorldCardId"`
	ActiveIndex              int                        `json:"activeIndex"`
	ActiveEditingComponentID string                     `json:"activeEditingComponentId,omitempty"`
	ActiveEditingOverlay     *ComponentOverlay          `json:"activeEditingOverlay,omitempty"`
	Library                  []RenderedGameCard         `json:"library"`
	EditSession              *RenderedGameEditSession   `json:"editSession,omitempty"`
	Encounter                *RenderedEncounterSnapshot `json:"encounter,omitempty"`
	SolvedFlags              map[string]bool            `json:"solvedFlags"`
	Message                  string                     `json:"message,omitempty"`
}

type RenderedGameEditSession struct {
	TargetCardID                string            `json:"targetCardId"`
	DraftCard                   RenderedGameCard  `json:"draftCard"`
	PendingConsumedComponentIDs []string          `json:"pendingConsumedComponentIds,omitempty"`
	SelectedComponentID         string            `json:"selectedComponentId,omitempty"`
	EditingOverlay              *ComponentOverlay `json:"editingOverlay,omitempty"`
}

type GameResultResponse struct {
	Revision uint64              `json:"revision"`
	Snapshot GameSessionSnapshot `json:"snapshot"`
	Events   []game.Event        `json:"events"`
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
			result, err := state.View()
			writeGameResult(registry, w, result, err)
		case "reset":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := decodeEmptyGameCommand(w, r); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.ResetCommand{})
			writeGameResult(registry, w, result, err)
		case "cycle":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				Direction string `json:"direction"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.CycleCardCommand{Direction: request.Direction})
			writeGameResult(registry, w, result, err)
		case "collect":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID string `json:"cardId"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.CollectCardCommand{CardID: game.CardInstanceID(request.CardID)})
			writeGameResult(registry, w, result, err)
		case "play-card":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				SourceCardID string `json:"sourceCardId"`
				TargetCardID string `json:"targetCardId"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.PlayCardCommand{SourceCardID: game.CardInstanceID(request.SourceCardID), TargetCardID: game.CardInstanceID(request.TargetCardID)})
			writeGameResult(registry, w, result, err)
		case "submit-form":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID string            `json:"cardId"`
				FormID string            `json:"formId"`
				Fields map[string]string `json:"fields"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.SubmitFormCommand{CardID: game.CardInstanceID(request.CardID), FormID: request.FormID, Fields: request.Fields})
			writeGameResult(registry, w, result, err)
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
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.SelectWorldComponentCommand{CardID: game.CardInstanceID(request.CardID), ComponentID: request.ComponentID, ComponentKind: request.ComponentKind})
			writeGameResult(registry, w, result, err)
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
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.ChangeWorldComponentCommand{CardID: game.CardInstanceID(request.CardID), ComponentID: request.ComponentID, ComponentKind: request.ComponentKind, Control: request.Control, Value: request.Value})
			writeGameResult(registry, w, result, err)
		case "edit/start":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				CardID string `json:"cardId"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.StartEditingCommand{CardID: game.CardInstanceID(request.CardID)})
			writeGameResult(registry, w, result, err)
		case "edit/install-component":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				ComponentCardID string `json:"componentCardId"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.InstallEditComponentCommand{ComponentCardID: game.CardInstanceID(request.ComponentCardID)})
			writeGameResult(registry, w, result, err)
		case "edit/component/select":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				ComponentID   string `json:"componentId"`
				ComponentKind string `json:"componentKind"`
			}
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.SelectEditComponentCommand{ComponentID: request.ComponentID, ComponentKind: request.ComponentKind})
			writeGameResult(registry, w, result, err)
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
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.ChangeEditComponentCommand{ComponentID: request.ComponentID, Control: request.Control, Value: request.Value})
			writeGameResult(registry, w, result, err)
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
			if err := decodeGameCommand(w, r, &request); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.ChangeLibraryComponentCommand{CardID: game.CardInstanceID(request.CardID), ComponentID: request.ComponentID, ComponentKind: request.ComponentKind, Control: request.Control, Value: request.Value})
			writeGameResult(registry, w, result, err)
		case "edit/save":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := decodeEmptyGameCommand(w, r); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.SaveEditCommand{})
			writeGameResult(registry, w, result, err)
		case "edit/cancel":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if err := decodeEmptyGameCommand(w, r); err != nil {
				writeInvalidGameRequest(w)
				return
			}
			result, err := state.Execute(game.CancelEditCommand{})
			writeGameResult(registry, w, result, err)
		default:
			http.NotFound(w, r)
		}
	}
}

func writeGameResult(registry *cardcomponent.Registry, w http.ResponseWriter, result game.Result, err error) {
	if err != nil {
		status := http.StatusInternalServerError
		if game.IsInvalidCommand(err) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	snapshot, err := renderGameSessionSnapshot(registry, result.Snapshot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, GameResultResponse{
		Revision: result.Revision,
		Snapshot: snapshot,
		Events:   result.Events,
	})
}

func decodeGameCommand(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func decodeEmptyGameCommand(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body struct{}
	err := decoder.Decode(&body)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeInvalidGameRequest(w http.ResponseWriter) {
	http.Error(w, "invalid request body", http.StatusBadRequest)
}

func renderGameSessionSnapshot(registry *cardcomponent.Registry, snapshot game.Snapshot) (GameSessionSnapshot, error) {
	worldDeck, err := renderWorldGameCards(registry, snapshot.WorldDeck)
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	activeWorldCard, err := renderGameCard(registry, snapshot.ActiveWorldCard, "game-world-"+safeDOMID(string(snapshot.ActiveWorldCard.ID)))
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	library, err := renderLibraryGameCards(registry, snapshot.Library)
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	var editSession *RenderedGameEditSession
	if snapshot.EditSession != nil {
		rendered, err := renderGameCard(registry, snapshot.EditSession.DraftCard, "game-edit-"+safeDOMID(string(snapshot.EditSession.DraftCard.ID)))
		if err != nil {
			return GameSessionSnapshot{}, err
		}
		editSession = &RenderedGameEditSession{
			TargetCardID:                string(snapshot.EditSession.TargetCardID),
			DraftCard:                   rendered,
			PendingConsumedComponentIDs: cardInstanceIDsToStrings(snapshot.EditSession.PendingConsumedComponentIDs),
			SelectedComponentID:         snapshot.EditSession.SelectedComponentID,
			EditingOverlay:              gameEditingOverlay(registry, snapshot.EditSession.DraftCard, snapshot.EditSession.SelectedComponentID),
		}
	}
	var encounter *RenderedEncounterSnapshot
	if snapshot.Encounter != nil {
		participants := make([]RenderedEncounterParticipant, 0, len(snapshot.Encounter.Participants))
		for index, participant := range snapshot.Encounter.Participants {
			prefix := fmt.Sprintf("game-encounter-%d-%s", index, safeDOMID(string(participant.Card.ID)))
			rendered, err := renderGameCard(registry, participant.Card, prefix)
			if err != nil {
				return GameSessionSnapshot{}, err
			}
			participants = append(participants, RenderedEncounterParticipant{
				Role: string(participant.Role), Card: rendered,
			})
		}
		encounter = &RenderedEncounterSnapshot{
			ID: string(snapshot.Encounter.ID), Phase: string(snapshot.Encounter.Phase),
			Participants: participants, Pressure: snapshot.Encounter.Pressure,
			MaxPressure: snapshot.Encounter.MaxPressure, Outcome: snapshot.Encounter.Outcome,
		}
	}
	return GameSessionSnapshot{
		WorldDeck:                worldDeck,
		ActiveWorldCard:          activeWorldCard,
		ActiveWorldCardID:        string(snapshot.ActiveWorldCardID),
		ActiveIndex:              snapshot.ActiveIndex,
		ActiveEditingComponentID: snapshot.ActiveEditingComponentID,
		ActiveEditingOverlay:     gameActiveEditingOverlay(registry, snapshot.ActiveWorldCard, snapshot.ActiveEditingComponentID, snapshot.Library),
		Library:                  library,
		EditSession:              editSession,
		Encounter:                encounter,
		SolvedFlags:              cloneValue(snapshot.SolvedFlags),
		Message:                  snapshot.Message,
	}, nil
}

func renderWorldGameCards(registry *cardcomponent.Registry, cards []game.CardSnapshot) ([]RenderedGameCard, error) {
	out := make([]RenderedGameCard, 0, len(cards))
	for _, card := range cards {
		rendered, err := renderGameCard(registry, card, "game-world-"+safeDOMID(string(card.ID)))
		if err != nil {
			return nil, err
		}
		out = append(out, rendered)
	}
	return out, nil
}

func renderLibraryGameCards(registry *cardcomponent.Registry, cards []game.CardSnapshot) ([]RenderedGameCard, error) {
	out := make([]RenderedGameCard, 0, len(cards))
	for index, card := range cards {
		prefix := fmt.Sprintf("game-library-%d-%s", index, safeDOMID(string(card.ID)))
		rendered, err := renderGameCard(registry, card, prefix)
		if err != nil {
			return nil, err
		}
		out = append(out, rendered)
	}
	return out, nil
}

func renderGameCard(registry *cardcomponent.Registry, card game.CardSnapshot, domIDPrefix string) (RenderedGameCard, error) {
	preview, err := cardcomponent.RenderDocumentWithOptions(card.Document, registry, cardcomponent.RenderOptions{
		ElementID:   domIDPrefix,
		DOMIDPrefix: domIDPrefix,
	})
	if err != nil {
		return RenderedGameCard{}, err
	}
	return RenderedGameCard{
		ID:          string(card.ID),
		Name:        card.Name,
		Kind:        card.Kind,
		Tags:        append([]string(nil), card.Tags...),
		Collectible: card.Collectible,
		Collected:   card.Collected,
		State:       cloneValue(card.State),
		Document:    cloneValue(card.Document),
		Actor:       cloneValue(card.Actor),
		PreviewHTML: preview.Render(),
	}, nil
}

func cardInstanceIDsToStrings(ids []game.CardInstanceID) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for index, id := range ids {
		out[index] = string(id)
	}
	return out
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
