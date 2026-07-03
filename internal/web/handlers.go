package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	. "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/profile"
	"github.com/n0remac/Living-Card/internal/web/component"
	"github.com/n0remac/Living-Card/internal/web/components/appheader"
	"github.com/n0remac/Living-Card/internal/web/components/chatform"
)

type Dependencies = component.Dependencies

func Register(mux *http.ServeMux, deps Dependencies) {
	registry := component.MustNewRegistry(
		appheader.Definition(),
		chatform.Definition(),
	)
	deps.ProjectRoot = firstNonEmpty(deps.ProjectRoot, projectRoot())
	mux.HandleFunc("/", ServeNode(Page()))
	mux.HandleFunc("/assets/", frontendAssetHandler())
	mux.HandleFunc("/api/components/", componentResourceHandler(deps, registry))
	mux.HandleFunc("/api/cards", cardsListHandler(deps))
	mux.HandleFunc("/api/cards/", cardResourceHandler(deps, registry))
	mux.HandleFunc("/api/users/", userResourceHandler(deps))
	appheader.RegisterRoutes(mux, deps)
	chatform.RegisterRoutes(mux, deps)
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

func cardsListHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSONResponse(w, deps.Cards.List())
	}
}

func cardResourceHandler(deps Dependencies, registry *component.Registry) http.HandlerFunc {
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
		case "canvas":
			if len(parts) != 2 {
				http.NotFound(w, r)
				return
			}
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			node, err := registry.RenderCard(card)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write([]byte(node.Render()))
		case "memories":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			userID := profile.NormalizeUserID(r.URL.Query().Get("user_id"))
			memories, err := deps.Memory.ListByCard(r.Context(), userID, card.CardID, 10)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSONResponse(w, memories)
		default:
			if registry.HandleCardResource(w, r, deps, card, parts) {
				return
			}
			http.NotFound(w, r)
		}
	}
}

func componentResourceHandler(deps Dependencies, registry *component.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/components/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] != "patch-proposals" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request component.PatchProposalRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		service := component.NewPatchProposalService(registry, deps.Patch, deps.PatchModel, deps.ProjectRoot)
		response, err := service.Propose(r.Context(), parts[0], request.Instruction)
		if err != nil {
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, component.ErrEmptyInstruction):
				status = http.StatusBadRequest
			case errors.Is(err, component.ErrComponentTypeNotFound):
				status = http.StatusNotFound
			}
			http.Error(w, err.Error(), status)
			return
		}
		writeJSONResponse(w, response)
	}
}

func userResourceHandler(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/users/")
		path = strings.Trim(path, "/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] != "profile" {
			http.NotFound(w, r)
			return
		}
		if deps.Profile == nil {
			http.Error(w, "profile store is not initialized", http.StatusInternalServerError)
			return
		}
		userID := profile.NormalizeUserID(parts[0])
		switch r.Method {
		case http.MethodGet:
			profile, err := deps.Profile.Get(r.Context(), userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSONResponse(w, profile)
		case http.MethodDelete:
			if err := deps.Profile.Reset(r.Context(), userID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			profile, err := deps.Profile.Get(r.Context(), userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSONResponse(w, profile)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func writeJSONResponse(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "failed to write json response", http.StatusInternalServerError)
	}
}
