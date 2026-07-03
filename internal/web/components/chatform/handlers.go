package chatform

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/chat"
	"github.com/n0remac/Living-Card/internal/web/component"
)

const ComponentType = cards.ChatFormComponentType

func Definition() component.Definition {
	return component.Definition{
		Type: ComponentType,
		Render: func(cards.ComponentInstance) *godom.Node {
			return View()
		},
		HandleAction:      HandleAction,
		ClientInitializer: "initChatForm",
		ContextFiles: []string{
			"internal/web/components/chatform/view.go",
			"internal/web/components/chatform/handlers.go",
			"internal/web/components/chatform/client.ts",
			"internal/web/components/chatform/types.ts",
		},
	}
}

func RegisterRoutes(_ *http.ServeMux, _ component.Dependencies) {
}

func HandleAction(w http.ResponseWriter, r *http.Request, deps component.Dependencies, card cards.Card, _ cards.ComponentInstance, action string) bool {
	if action != "send" {
		return false
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}
	if deps.Chat == nil {
		http.Error(w, "chat service is not initialized", http.StatusInternalServerError)
		return true
	}
	var request struct {
		UserID  string `json:"user_id"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return true
	}
	result, err := deps.Chat.Chat(r.Context(), chat.Request{
		UserID:  request.UserID,
		CardID:  card.CardID,
		Message: request.Message,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, chat.ErrCardNotFound):
			status = http.StatusNotFound
		case isBadRequestError(err):
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return true
	}
	writeJSONResponse(w, result)
	return true
}

func isBadRequestError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"cannot be empty",
		"invalid",
		"required",
		"must be",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func writeJSONResponse(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "failed to write json response", http.StatusInternalServerError)
	}
}
