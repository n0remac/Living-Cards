package cards

import (
	"fmt"
	"sort"
	"strings"
)

const ChatFormComponentType = "chat-form"

type Personality struct {
	Tone       string   `json:"tone"`
	StyleRules []string `json:"style_rules"`
}

type Constraints struct {
	KnowledgeScope string   `json:"knowledge_scope"`
	ToolAccess     []string `json:"tool_access"`
}

type ComponentInstance struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"`
	Props map[string]any `json:"props"`
}

type Card struct {
	CardID      string              `json:"card_id"`
	Name        string              `json:"name"`
	Domain      []string            `json:"domain"`
	Archetype   string              `json:"archetype"`
	Personality Personality         `json:"personality"`
	Constraints Constraints         `json:"constraints"`
	Components  []ComponentInstance `json:"components"`
}

type Store struct {
	cards map[string]Card
}

func DefaultCatalog() []Card {
	return []Card{
		{
			CardID:    "ember_stag_001",
			Name:      "Ember Stag",
			Domain:    []string{"fire", "transformation"},
			Archetype: "ancient guardian",
			Personality: Personality{
				Tone: "calm, proud, poetic",
				StyleRules: []string{
					"use short sentences",
					"prefer metaphor over direct explanation",
					"avoid modern slang",
				},
			},
			Constraints: Constraints{
				KnowledgeScope: "abstract and philosophical",
				ToolAccess:     []string{},
			},
			Components: DefaultComponents(),
		},
		{
			CardID:    "aldren_scribe_001",
			Name:      "Aldren the Scribe",
			Domain:    []string{"knowledge", "study", "patterns"},
			Archetype: "scholar wizard",
			Personality: Personality{
				Tone: "calm, thoughtful, mildly curious",
				StyleRules: []string{
					"speak clearly and concisely",
					"favor explanation over metaphor",
					"use occasional light analogies when helpful",
					"maintain a neutral, slightly formal tone",
					"avoid dramatic or mystical language, except when making a point",
				},
			},
			Constraints: Constraints{
				KnowledgeScope: "analytical and explanatory",
				ToolAccess:     []string{},
			},
			Components: DefaultComponents(),
		},
	}
}

func DefaultComponents() []ComponentInstance {
	return []ComponentInstance{{
		ID:    ChatFormComponentType,
		Type:  ChatFormComponentType,
		Props: map[string]any{},
	}}
}

func NewStore(catalog []Card) (*Store, error) {
	if len(catalog) == 0 {
		return nil, fmt.Errorf("card catalog cannot be empty")
	}
	store := &Store{
		cards: make(map[string]Card),
	}
	for idx, card := range catalog {
		card = sanitizeCard(card)
		if err := validateCard(card); err != nil {
			return nil, fmt.Errorf("card %d: %w", idx, err)
		}
		if _, exists := store.cards[card.CardID]; exists {
			return nil, fmt.Errorf("duplicate card_id %q", card.CardID)
		}
		store.cards[card.CardID] = card
	}
	return store, nil
}

func NewStaticStore() (*Store, error) {
	return NewStore(DefaultCatalog())
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
	card.Components = sanitizeComponents(card.Components)
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
	if len(card.Components) == 0 {
		return fmt.Errorf("components are required")
	}
	seenComponents := make(map[string]struct{}, len(card.Components))
	for _, component := range card.Components {
		if component.ID == "" {
			return fmt.Errorf("component id is required")
		}
		if component.Type == "" {
			return fmt.Errorf("component type is required")
		}
		if _, exists := seenComponents[component.ID]; exists {
			return fmt.Errorf("duplicate component id %q", component.ID)
		}
		seenComponents[component.ID] = struct{}{}
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

func sanitizeComponents(input []ComponentInstance) []ComponentInstance {
	out := make([]ComponentInstance, 0, len(input))
	for _, item := range input {
		item.ID = strings.TrimSpace(item.ID)
		item.Type = strings.TrimSpace(item.Type)
		if item.Props == nil {
			item.Props = map[string]any{}
		}
		out = append(out, item)
	}
	return out
}

func cloneCard(card Card) Card {
	card.Domain = append([]string(nil), card.Domain...)
	card.Personality.StyleRules = append([]string(nil), card.Personality.StyleRules...)
	card.Constraints.ToolAccess = append([]string(nil), card.Constraints.ToolAccess...)
	card.Components = cloneComponents(card.Components)
	return card
}

func cloneComponents(components []ComponentInstance) []ComponentInstance {
	out := make([]ComponentInstance, 0, len(components))
	for _, component := range components {
		cloned := component
		cloned.Props = cloneProps(component.Props)
		out = append(out, cloned)
	}
	return out
}

func cloneProps(props map[string]any) map[string]any {
	out := make(map[string]any, len(props))
	for key, value := range props {
		out[key] = value
	}
	return out
}
