package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/chat"
	"github.com/n0remac/Living-Card/internal/memory"
)

type CardStore interface {
	List() []cards.Card
	Get(cardID string) (cards.Card, bool)
}

type MemoryStore interface {
	ListByCard(ctx context.Context, cardID string, limit int) ([]memory.Memory, error)
}

type ChatService interface {
	Chat(ctx context.Context, request chat.Request) (chat.Result, error)
}

type Dependencies struct {
	Cards  CardStore
	Memory MemoryStore
	Chat   ChatService
}

func Register(mux *http.ServeMux, deps Dependencies) {
	mux.HandleFunc("/", ServeNode(Page()))
	mux.HandleFunc("/api/cards", cardsListHandler(deps))
	mux.HandleFunc("/api/cards/", cardResourceHandler(deps))
}

func cardsListHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSONResponse(w, deps.Cards.List())
	}
}

func cardResourceHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/cards/")
		path = strings.Trim(path, "/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		parts := strings.Split(path, "/")
		cardID := parts[0]
		card, ok := deps.Cards.Get(cardID)
		if !ok {
			http.Error(w, "card not found", http.StatusNotFound)
			return
		}

		if len(parts) == 1 {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			writeJSONResponse(w, card)
			return
		}

		switch parts[1] {
		case "memories":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			memories, err := deps.Memory.ListByCard(r.Context(), card.CardID, 10)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSONResponse(w, memories)
		case "chat":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var request struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "invalid request body", http.StatusBadRequest)
				return
			}
			result, err := deps.Chat.Chat(r.Context(), chat.Request{
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
				return
			}
			writeJSONResponse(w, result)
		default:
			http.NotFound(w, r)
		}
	}
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
