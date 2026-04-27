package cards

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/n0remac/Living-Card/internal/fileutil"
)

type Personality struct {
	Tone       string   `json:"tone"`
	StyleRules []string `json:"style_rules"`
}

type Constraints struct {
	KnowledgeScope string   `json:"knowledge_scope"`
	ToolAccess     []string `json:"tool_access"`
}

type Card struct {
	CardID      string      `json:"card_id"`
	Name        string      `json:"name"`
	Domain      []string    `json:"domain"`
	Archetype   string      `json:"archetype"`
	Personality Personality `json:"personality"`
	Constraints Constraints `json:"constraints"`
}

type Store struct {
	dir   string
	cards map[string]Card
}

func NewStore(dir string) (*Store, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || dir == "." {
		return nil, fmt.Errorf("cards dir cannot be empty")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read cards dir: %w", err)
	}

	store := &Store{
		dir:   dir,
		cards: make(map[string]Card),
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var card Card
		if err := fileutil.ReadJSONFile(filepath.Join(dir, entry.Name()), &card); err != nil {
			return nil, err
		}
		card = sanitizeCard(card)
		if err := validateCard(card); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if _, exists := store.cards[card.CardID]; exists {
			return nil, fmt.Errorf("duplicate card_id %q", card.CardID)
		}
		store.cards[card.CardID] = card
	}
	if len(store.cards) == 0 {
		return nil, fmt.Errorf("no cards found in %q", dir)
	}
	return store, nil
}

func (s *Store) List() []Card {
	if s == nil {
		return nil
	}
	out := make([]Card, 0, len(s.cards))
	for _, card := range s.cards {
		out = append(out, cloneCard(card))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].CardID < out[j].CardID
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (s *Store) Get(cardID string) (Card, bool) {
	if s == nil {
		return Card{}, false
	}
	card, ok := s.cards[strings.TrimSpace(cardID)]
	return cloneCard(card), ok
}

func sanitizeCard(card Card) Card {
	card.CardID = strings.TrimSpace(card.CardID)
	card.Name = strings.TrimSpace(card.Name)
	card.Archetype = strings.TrimSpace(card.Archetype)
	card.Personality.Tone = strings.TrimSpace(card.Personality.Tone)
	card.Constraints.KnowledgeScope = strings.TrimSpace(card.Constraints.KnowledgeScope)
	card.Domain = sanitizeStrings(card.Domain)
	card.Personality.StyleRules = sanitizeStrings(card.Personality.StyleRules)
	card.Constraints.ToolAccess = sanitizeStrings(card.Constraints.ToolAccess)
	return card
}

func validateCard(card Card) error {
	if card.CardID == "" {
		return fmt.Errorf("card_id is required")
	}
	if card.Name == "" {
		return fmt.Errorf("name is required")
	}
	if card.Personality.Tone == "" && len(card.Personality.StyleRules) == 0 {
		return fmt.Errorf("personality is required")
	}
	if card.Constraints.KnowledgeScope == "" && len(card.Constraints.ToolAccess) == 0 {
		return fmt.Errorf("constraints are required")
	}
	return nil
}

func sanitizeStrings(input []string) []string {
	out := make([]string, 0, len(input))
	for _, item := range input {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func cloneCard(card Card) Card {
	card.Domain = append([]string(nil), card.Domain...)
	card.Personality.StyleRules = append([]string(nil), card.Personality.StyleRules...)
	card.Constraints.ToolAccess = append([]string(nil), card.Constraints.ToolAccess...)
	return card
}
