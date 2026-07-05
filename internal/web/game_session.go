package web

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"
	"unicode"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

const (
	gameKindWorld = "world"
	gameKindItem  = "item"
	gameKindClue  = "clue"
)

type GameCard struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Tags        []string               `json:"tags,omitempty"`
	Collectible bool                   `json:"collectible"`
	Collected   bool                   `json:"collected,omitempty"`
	State       map[string]any         `json:"state,omitempty"`
	Document    cardcomponent.Document `json:"document"`
}

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
	WorldDeck         []RenderedGameCard `json:"worldDeck"`
	ActiveWorldCard   RenderedGameCard   `json:"activeWorldCard"`
	ActiveWorldCardID string             `json:"activeWorldCardId"`
	ActiveIndex       int                `json:"activeIndex"`
	Library           []RenderedGameCard `json:"library"`
	SolvedFlags       map[string]bool    `json:"solvedFlags"`
	Message           string             `json:"message,omitempty"`
}

type gameSession struct {
	mu          sync.Mutex
	worldDeck   []GameCard
	activeIndex int
	library     []GameCard
	solvedFlags map[string]bool
	lastMessage string
}

func newGameSession() *gameSession {
	return &gameSession{
		worldDeck:   seededWorldDeck(),
		activeIndex: 0,
		library:     nil,
		solvedFlags: map[string]bool{
			"doorUnlocked": false,
		},
		lastMessage: "Cycle through the cards. Some are useful; some only remember the room.",
	}
}

func (s *gameSession) snapshot() (GameSessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.snapshotLocked()
}

func (s *gameSession) reset() (GameSessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := newGameSession()
	s.worldDeck = next.worldDeck
	s.activeIndex = next.activeIndex
	s.library = next.library
	s.solvedFlags = next.solvedFlags
	s.lastMessage = next.lastMessage
	return s.snapshotLocked()
}

func (s *gameSession) cycle(direction string) (GameSessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.worldDeck) == 0 {
		return GameSessionSnapshot{}, fmt.Errorf("world deck is empty")
	}
	switch strings.TrimSpace(direction) {
	case "previous", "prev", "back":
		s.activeIndex--
	case "", "next":
		s.activeIndex++
	default:
		return GameSessionSnapshot{}, fmt.Errorf("direction must be next or previous")
	}
	if s.activeIndex < 0 {
		s.activeIndex = len(s.worldDeck) - 1
	}
	if s.activeIndex >= len(s.worldDeck) {
		s.activeIndex = 0
	}
	s.lastMessage = "The next card slides into view."
	return s.snapshotLocked()
}

func (s *gameSession) collect(cardID string) (GameSessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cardID = strings.TrimSpace(cardID)
	if cardID == "" && len(s.worldDeck) > 0 {
		cardID = s.worldDeck[s.activeIndex].ID
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return GameSessionSnapshot{}, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	card := s.worldDeck[index]
	if !card.Collectible {
		return GameSessionSnapshot{}, fmt.Errorf("%s cannot be collected", card.Name)
	}
	if card.Collected {
		s.lastMessage = card.Name + " is already in your library."
		return s.snapshotLocked()
	}
	card.Collected = true
	card.Collectible = false
	s.worldDeck[index] = card
	libraryCard := cloneValue(card)
	libraryCard.Collectible = false
	libraryCard.Collected = true
	s.library = append(s.library, libraryCard)
	s.lastMessage = card.Name + " moved into your library."
	return s.snapshotLocked()
}

func (s *gameSession) playCard(sourceCardID, targetCardID string) (GameSessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceCardID = strings.TrimSpace(sourceCardID)
	targetCardID = strings.TrimSpace(targetCardID)
	source := s.libraryCard(sourceCardID)
	if source == nil {
		return GameSessionSnapshot{}, fmt.Errorf("card %q is not in your library", sourceCardID)
	}
	if targetCardID == "" && len(s.worldDeck) > 0 {
		targetCardID = s.worldDeck[s.activeIndex].ID
	}
	targetIndex := s.worldCardIndex(targetCardID)
	if targetIndex < 0 {
		return GameSessionSnapshot{}, fmt.Errorf("target card %q is not in the world deck", targetCardID)
	}
	target := s.worldDeck[targetIndex]
	if hasTag(*source, "iron-key") && target.ID == "rusted-cell-door" && !s.solvedFlags["doorUnlocked"] {
		s.solvedFlags["doorUnlocked"] = true
		target.State["locked"] = false
		target.Tags = removeString(target.Tags, "locked")
		target.Document = doorDocument(false)
		s.worldDeck[targetIndex] = target
		s.lastMessage = "The iron key catches. The door card unlocks."
		return s.snapshotLocked()
	}
	s.lastMessage = "Nothing on this card responds to " + source.Name + "."
	return s.snapshotLocked()
}

func (s *gameSession) snapshotLocked() (GameSessionSnapshot, error) {
	if len(s.worldDeck) == 0 {
		return GameSessionSnapshot{}, fmt.Errorf("world deck is empty")
	}
	if s.activeIndex < 0 || s.activeIndex >= len(s.worldDeck) {
		s.activeIndex = 0
	}
	worldDeck, err := renderGameCards(s.worldDeck)
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	library, err := renderGameCards(s.library)
	if err != nil {
		return GameSessionSnapshot{}, err
	}
	return GameSessionSnapshot{
		WorldDeck:         worldDeck,
		ActiveWorldCard:   worldDeck[s.activeIndex],
		ActiveWorldCardID: worldDeck[s.activeIndex].ID,
		ActiveIndex:       s.activeIndex,
		Library:           library,
		SolvedFlags:       cloneValue(s.solvedFlags),
		Message:           s.lastMessage,
	}, nil
}

func (s *gameSession) worldCardIndex(cardID string) int {
	for index, card := range s.worldDeck {
		if card.ID == cardID {
			return index
		}
	}
	return -1
}

func (s *gameSession) libraryCard(cardID string) *GameCard {
	for index := range s.library {
		if s.library[index].ID == cardID {
			return &s.library[index]
		}
	}
	return nil
}

func gameResourceHandler(state *gameSession) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/game"), "/")
		switch path {
		case "session":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := state.snapshot()
			writeGameSnapshot(w, snapshot, err)
		case "reset":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			snapshot, err := state.reset()
			writeGameSnapshot(w, snapshot, err)
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
			snapshot, err := state.cycle(request.Direction)
			writeGameSnapshot(w, snapshot, err)
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
			snapshot, err := state.collect(request.CardID)
			writeGameSnapshot(w, snapshot, err)
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
			snapshot, err := state.playCard(request.SourceCardID, request.TargetCardID)
			writeGameSnapshot(w, snapshot, err)
		default:
			http.NotFound(w, r)
		}
	}
}

func writeGameSnapshot(w http.ResponseWriter, snapshot GameSessionSnapshot, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSONResponse(w, snapshot)
}

func renderGameCards(cards []GameCard) ([]RenderedGameCard, error) {
	out := make([]RenderedGameCard, 0, len(cards))
	for _, card := range cards {
		preview, err := cardcomponent.RenderDocumentWithID(card.Document, cardComponentRegistry(), "game-card-"+safeDOMID(card.ID))
		if err != nil {
			return nil, err
		}
		out = append(out, RenderedGameCard{
			ID:          card.ID,
			Name:        card.Name,
			Kind:        card.Kind,
			Tags:        append([]string(nil), card.Tags...),
			Collectible: card.Collectible,
			Collected:   card.Collected,
			State:       cloneValue(card.State),
			Document:    cloneValue(card.Document),
			PreviewHTML: preview.Render(),
		})
	}
	return out, nil
}

func seededWorldDeck() []GameCard {
	return []GameCard{
		{
			ID:       "rusted-cell-door",
			Name:     "Rusted Cell Door",
			Kind:     gameKindWorld,
			Tags:     []string{"door", "locked", "iron-key"},
			State:    map[string]any{"locked": true},
			Document: doorDocument(true),
		},
		{
			ID:       "inventory-label",
			Name:     "Inventory Label",
			Kind:     gameKindClue,
			Tags:     []string{"decoy", "worldbuilding"},
			State:    map[string]any{"useful": false},
			Document: textSceneDocument("inventory-label", "Inventory Label", "A little brass plate reads: CARRY ONLY WHAT OPENS A WAY.", "#18212f", "#f59e0b"),
		},
		{
			ID:          "bent-iron-key",
			Name:        "Bent Iron Key",
			Kind:        gameKindItem,
			Tags:        []string{"item", "key", "iron-key"},
			Collectible: true,
			State:       map[string]any{"opens": "rusted-cell-door"},
			Document:    keyDocument(),
		},
		{
			ID:       "faded-photograph",
			Name:     "Faded Photograph",
			Kind:     gameKindWorld,
			Tags:     []string{"decoy", "worldbuilding", "image"},
			State:    map[string]any{"useful": false},
			Document: photographDocument(),
		},
		{
			ID:       "sleeping-switch",
			Name:     "Sleeping Switch",
			Kind:     gameKindWorld,
			Tags:     []string{"decoy", "machine"},
			State:    map[string]any{"useful": false},
			Document: textSceneDocument("sleeping-switch", "Sleeping Switch", "A switch card dreams of being important. It is not connected to anything yet.", "#211827", "#a78bfa"),
		},
	}
}

func doorDocument(locked bool) cardcomponent.Document {
	title := "LOCKED"
	status := "Requires a bent iron key."
	bg := "#201815"
	accent := "#f59e0b"
	if !locked {
		title = "OPEN"
		status = "A dark route has appeared behind the door."
		bg = "#10231f"
		accent = "#34d399"
	}
	return cardDocument("rusted-cell-door", "Rusted Cell Door", bg, "#5b4636",
		textNode("door-title", title, 50, 15, 34, accent, 800),
		shapeNode("door-panel", "roundedRectangle", 50, 51, 64, 64, "#35251d", "#a16207", 3, 0),
		shapeNode("door-window", "roundedRectangle", 50, 32, 34, 14, "#111827", "#fbbf24", 2, 0),
		shapeNode("door-knob", "circle", 68, 54, 10, 10, accent, "#111827", 2, 0),
		textNode("door-status", status, 50, 88, 15, "#f8fafc", 600),
	)
}

func keyDocument() cardcomponent.Document {
	return cardDocument("bent-iron-key", "Bent Iron Key", "#17211d", "#d6a84f",
		textNode("key-title", "BENT IRON KEY", 50, 15, 24, "#fde68a", 800),
		shapeNode("key-ring", "circle", 35, 47, 24, 24, "rgba(0,0,0,0)", "#facc15", 5, 0),
		shapeNode("key-stem", "rectangle", 56, 47, 38, 8, "#facc15", "#713f12", 2, 0),
		shapeNode("key-tooth-a", "rectangle", 74, 53, 8, 13, "#facc15", "#713f12", 2, 0),
		shapeNode("key-tooth-b", "rectangle", 82, 51, 8, 9, "#facc15", "#713f12", 2, 0),
		textNode("key-note", "Its teeth match a rusted lock.", 50, 84, 16, "#f8fafc", 600),
	)
}

func photographDocument() cardcomponent.Document {
	return cardDocument("faded-photograph", "Faded Photograph", "#171717", "#94a3b8",
		textNode("photo-title", "FADED PHOTOGRAPH", 50, 14, 20, "#e2e8f0", 800),
		cardcomponent.Node{
			ID:       "photo-image",
			Type:     imagecomponent.Type,
			Fragment: mustGameRaw(imagecomponent.Fragment{Src: imagecomponent.DataURLForTesting(), Alt: "A nearly blank old photograph", X: 50, Y: 48, Width: 58, Height: 40, BorderColor: "#e2e8f0", BorderWidthPX: 2, BorderRadiusPX: 6}),
		},
		textNode("photo-note", "The picture has forgotten almost everything.", 50, 84, 15, "#cbd5e1", 600),
	)
}

func textSceneDocument(id, name, text, bg, accent string) cardcomponent.Document {
	return cardDocument(id, name, bg, accent,
		shapeNode(id+"-mark", "diamond", 50, 28, 22, 22, accent, "#111827", 2, 0),
		textNode(id+"-title", strings.ToUpper(name), 50, 51, 22, "#f8fafc", 800),
		textNode(id+"-body", text, 50, 73, 16, "#cbd5e1", 600),
	)
}

func cardDocument(id, name, bgColor, borderColor string, children ...cardcomponent.Node) cardcomponent.Document {
	allChildren := []cardcomponent.Node{
		{ID: id + "-background", Type: background.Type, Fragment: mustGameRaw(background.Fragment{BackgroundColor: bgColor, CSS: "background: " + bgColor + ";"})},
		{ID: id + "-border", Type: border.Type, Fragment: mustGameRaw(border.Fragment{BorderWidthPX: 2, BorderRadiusPX: 22, BorderColor: borderColor, CSS: "border: 2px solid " + borderColor + "; border-radius: 22px;"})},
	}
	allChildren = append(allChildren, children...)
	return cardcomponent.Document{
		CardID: id,
		Name:   name,
		Root: cardcomponent.Node{
			ID:       id + "-root",
			Type:     cardcomponent.Type,
			Fragment: cardcomponent.EncodeRootFragment(cardcomponent.RootFragment{PaddingPX: 18, Shadow: "0 24px 60px rgba(0,0,0,0.34)"}),
			Children: allChildren,
		},
	}
}

func textNode(id, text string, x, y, size int, color string, weight int) cardcomponent.Node {
	return cardcomponent.Node{
		ID:   id,
		Type: textarea.Type,
		Fragment: mustGameRaw(textarea.Fragment{
			Content:    text,
			FontFamily: "system",
			FontSizePX: size,
			FontWeight: weight,
			FontStyle:  "normal",
			Color:      color,
			Align:      "center",
			Position:   "center",
			X:          x,
			Y:          y,
			PaddingPX:  4,
			CSS:        "text-align: center;",
		}),
	}
}

func shapeNode(id, shapeName string, x, y, width, height int, fill, stroke string, strokeWidth, rotation int) cardcomponent.Node {
	return cardcomponent.Node{
		ID:   id,
		Type: shape.Type,
		Fragment: mustGameRaw(shape.Fragment{
			Shape:           shapeName,
			X:               x,
			Y:               y,
			Width:           width,
			Height:          height,
			Rotation:        rotation,
			BackgroundColor: fill,
			BorderColor:     stroke,
			BorderWidthPX:   strokeWidth,
		}),
	}
}

func mustGameRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func hasTag(card GameCard, tag string) bool {
	for _, candidate := range card.Tags {
		if candidate == tag {
			return true
		}
	}
	return false
}

func removeString(values []string, value string) []string {
	out := values[:0]
	for _, candidate := range values {
		if candidate != value {
			out = append(out, candidate)
		}
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
