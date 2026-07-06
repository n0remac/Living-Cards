package game

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
)

//go:embed decks/*.json
var deckFiles embed.FS

const (
	EffectSetFlag             = "setFlag"
	EffectSetCardState        = "setCardState"
	EffectRemoveCardTags      = "removeCardTags"
	EffectSetDocumentVariant  = "setDocumentVariant"
	EffectSetMessage          = "setMessage"
	EffectLoadDeck            = "loadDeck"
	SeededWorldDeckDefinition = "seeded_world"
	FuseRoomDeckDefinition    = "fuse_room"
	GeneratorDeckDefinition   = "generator_room"
)

type DeckDefinition struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	InitialActiveCardID string              `json:"initialActiveCardId"`
	InitialMessage      string              `json:"initialMessage"`
	InitialSolvedFlags  map[string]bool     `json:"initialSolvedFlags,omitempty"`
	Cards               []CardDefinition    `json:"cards"`
	UseRules            []UseRuleDefinition `json:"useRules,omitempty"`
}

type CardDefinition struct {
	ID              string                            `json:"id"`
	Name            string                            `json:"name"`
	Kind            string                            `json:"kind"`
	Tags            []string                          `json:"tags,omitempty"`
	Collectible     bool                              `json:"collectible"`
	State           map[string]any                    `json:"state,omitempty"`
	InitialDocument string                            `json:"initialDocument"`
	Documents       map[string]cardcomponent.Document `json:"documents"`
}

type UseRuleDefinition struct {
	ID                        string                         `json:"id,omitempty"`
	Source                    CardMatcherDefinition          `json:"source"`
	Target                    CardMatcherDefinition          `json:"target"`
	FlagConditions            map[string]bool                `json:"flagConditions,omitempty"`
	SourceComponentConditions []ComponentConditionDefinition `json:"sourceComponentConditions,omitempty"`
	FailureMessage            string                         `json:"failureMessage,omitempty"`
	Effects                   []RuleEffectDefinition         `json:"effects"`
}

type CardMatcherDefinition struct {
	ID   string   `json:"id,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type ComponentConditionDefinition struct {
	Type        string `json:"type"`
	ValueEquals *int   `json:"valueEquals,omitempty"`
}

type RuleEffectDefinition struct {
	Type    string   `json:"type"`
	CardID  string   `json:"cardId,omitempty"`
	Key     string   `json:"key,omitempty"`
	Flag    string   `json:"flag,omitempty"`
	Value   any      `json:"value,omitempty"`
	Tags    []string `json:"tags,omitempty"`
	Variant string   `json:"variant,omitempty"`
	Message string   `json:"message,omitempty"`
	DeckID  string   `json:"deckId,omitempty"`
}

func LoadEmbeddedSeededWorldDeck() (DeckDefinition, error) {
	definition, err := LoadEmbeddedDeck(SeededWorldDeckDefinition)
	if err != nil {
		return DeckDefinition{}, err
	}
	if err := ValidateDeckDefinition(definition); err != nil {
		return DeckDefinition{}, err
	}
	return definition, nil
}

func LoadEmbeddedDeck(deckID string) (DeckDefinition, error) {
	deckID = strings.TrimSpace(deckID)
	if err := validateDeckID(deckID); err != nil {
		return DeckDefinition{}, err
	}
	raw, err := deckFiles.ReadFile("decks/" + deckID + ".json")
	if err != nil {
		return DeckDefinition{}, fmt.Errorf("read embedded deck %q: %w", deckID, err)
	}
	definition, err := decodeDeckDefinition(raw)
	if err != nil {
		return DeckDefinition{}, err
	}
	if definition.ID != deckID {
		return DeckDefinition{}, fmt.Errorf("embedded deck %q has id %q", deckID, definition.ID)
	}
	if _, err := validateDeckCards(definition); err != nil {
		return DeckDefinition{}, err
	}
	return definition, nil
}

func DecodeDeckDefinition(raw []byte) (DeckDefinition, error) {
	definition, err := decodeDeckDefinition(raw)
	if err != nil {
		return DeckDefinition{}, err
	}
	if err := ValidateDeckDefinition(definition); err != nil {
		return DeckDefinition{}, err
	}
	return definition, nil
}

func decodeDeckDefinition(raw []byte) (DeckDefinition, error) {
	var definition DeckDefinition
	if err := json.Unmarshal(raw, &definition); err != nil {
		return DeckDefinition{}, fmt.Errorf("decode deck definition: %w", err)
	}
	return definition, nil
}

func ValidateDeckDefinition(definition DeckDefinition) error {
	cardsByID, err := validateDeckCards(definition)
	if err != nil {
		return err
	}
	return validateRules(definition.UseRules, cardsByID)
}

func ValidateDeckPackDefinition(definition DeckDefinition, existingCards map[string]CardDefinition) error {
	cardsByID, err := validateDeckCards(definition)
	if err != nil {
		return err
	}
	for cardID, card := range existingCards {
		if _, exists := cardsByID[cardID]; exists {
			return fmt.Errorf("deck %q card %q already exists", definition.ID, cardID)
		}
		cardsByID[cardID] = card
	}
	return validateRules(definition.UseRules, cardsByID)
}

func validateDeckCards(definition DeckDefinition) (map[string]CardDefinition, error) {
	if strings.TrimSpace(definition.ID) == "" {
		return nil, fmt.Errorf("deck id is required")
	}
	if err := validateDeckID(definition.ID); err != nil {
		return nil, err
	}
	if len(definition.Cards) == 0 {
		return nil, fmt.Errorf("deck %q must contain at least one card", definition.ID)
	}
	cardsByID := make(map[string]CardDefinition, len(definition.Cards))
	for index, card := range definition.Cards {
		if strings.TrimSpace(card.ID) == "" {
			return nil, fmt.Errorf("card at index %d must have an id", index)
		}
		if _, exists := cardsByID[card.ID]; exists {
			return nil, fmt.Errorf("duplicate card id %q", card.ID)
		}
		if strings.TrimSpace(card.InitialDocument) == "" {
			return nil, fmt.Errorf("card %q initialDocument is required", card.ID)
		}
		if len(card.Documents) == 0 {
			return nil, fmt.Errorf("card %q must define at least one document variant", card.ID)
		}
		if _, exists := card.Documents[card.InitialDocument]; !exists {
			return nil, fmt.Errorf("card %q initial document variant %q does not exist", card.ID, card.InitialDocument)
		}
		for variant, document := range card.Documents {
			if strings.TrimSpace(variant) == "" {
				return nil, fmt.Errorf("card %q has an empty document variant key", card.ID)
			}
			if document.CardID != card.ID {
				return nil, fmt.Errorf("card %q document variant %q has card_id %q", card.ID, variant, document.CardID)
			}
			if document.Root.Type != cardcomponent.Type {
				return nil, fmt.Errorf("card %q document variant %q root type must be %q", card.ID, variant, cardcomponent.Type)
			}
		}
		cardsByID[card.ID] = card
	}
	if _, exists := cardsByID[definition.InitialActiveCardID]; !exists {
		return nil, fmt.Errorf("initial active card %q does not exist", definition.InitialActiveCardID)
	}
	return cardsByID, nil
}

func validateRules(rules []UseRuleDefinition, cardsByID map[string]CardDefinition) error {
	for _, rule := range rules {
		if err := validateRuleDefinition(rule, cardsByID); err != nil {
			if strings.TrimSpace(rule.ID) == "" {
				return err
			}
			return fmt.Errorf("use rule %q: %w", rule.ID, err)
		}
	}
	return nil
}

func validateDeckID(deckID string) error {
	if strings.TrimSpace(deckID) == "" {
		return fmt.Errorf("deck id is required")
	}
	for _, char := range deckID {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '-', char == '_':
		default:
			return fmt.Errorf("deck id %q may only contain letters, numbers, hyphens, and underscores", deckID)
		}
	}
	return nil
}

func validateRuleDefinition(rule UseRuleDefinition, cardsByID map[string]CardDefinition) error {
	if err := validateMatcher("source", rule.Source, cardsByID); err != nil {
		return err
	}
	if err := validateMatcher("target", rule.Target, cardsByID); err != nil {
		return err
	}
	if len(rule.Effects) == 0 {
		return fmt.Errorf("effects are required")
	}
	if err := validateComponentConditions(rule.SourceComponentConditions); err != nil {
		return err
	}
	for _, effect := range rule.Effects {
		if err := validateRuleEffect(effect, rule.Target, cardsByID); err != nil {
			return err
		}
	}
	return nil
}

func validateComponentConditions(conditions []ComponentConditionDefinition) error {
	for _, condition := range conditions {
		switch strings.TrimSpace(condition.Type) {
		case slider.Type:
			if condition.ValueEquals == nil {
				return fmt.Errorf("slider source component condition requires valueEquals")
			}
			value := *condition.ValueEquals
			if value < 0 || value > 100 {
				return fmt.Errorf("slider source component condition valueEquals must be between 0 and 100")
			}
		case "":
			return fmt.Errorf("source component condition requires type")
		default:
			return fmt.Errorf("unsupported source component condition type %q", condition.Type)
		}
	}
	return nil
}

func validateMatcher(name string, matcher CardMatcherDefinition, cardsByID map[string]CardDefinition) error {
	if strings.TrimSpace(matcher.ID) == "" && len(matcher.Tags) == 0 {
		return fmt.Errorf("%s matcher must define an id or tags", name)
	}
	if strings.TrimSpace(matcher.ID) != "" {
		if _, exists := cardsByID[matcher.ID]; !exists {
			return fmt.Errorf("%s matcher references unknown card %q", name, matcher.ID)
		}
	}
	for _, tag := range matcher.Tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("%s matcher contains an empty tag", name)
		}
	}
	return nil
}

func validateRuleEffect(effect RuleEffectDefinition, target CardMatcherDefinition, cardsByID map[string]CardDefinition) error {
	switch effect.Type {
	case EffectSetFlag:
		if strings.TrimSpace(effect.Flag) == "" {
			return fmt.Errorf("%s effect requires flag", EffectSetFlag)
		}
		if _, ok := effect.Value.(bool); !ok {
			return fmt.Errorf("%s effect value must be a boolean", EffectSetFlag)
		}
	case EffectSetCardState:
		if strings.TrimSpace(effect.Key) == "" {
			return fmt.Errorf("%s effect requires key", EffectSetCardState)
		}
		if _, err := effectCardDefinition(effect, target, cardsByID); err != nil {
			return err
		}
	case EffectRemoveCardTags:
		if len(effect.Tags) == 0 {
			return fmt.Errorf("%s effect requires tags", EffectRemoveCardTags)
		}
		for _, tag := range effect.Tags {
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("%s effect contains an empty tag", EffectRemoveCardTags)
			}
		}
		if _, err := effectCardDefinition(effect, target, cardsByID); err != nil {
			return err
		}
	case EffectSetDocumentVariant:
		card, err := effectCardDefinition(effect, target, cardsByID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(effect.Variant) == "" {
			return fmt.Errorf("%s effect requires variant", EffectSetDocumentVariant)
		}
		if _, exists := card.Documents[effect.Variant]; !exists {
			return fmt.Errorf("%s effect references missing variant %q for card %q", EffectSetDocumentVariant, effect.Variant, resolvedEffectCardID(effect, target))
		}
	case EffectSetMessage:
		if strings.TrimSpace(effect.Message) == "" {
			return fmt.Errorf("%s effect requires message", EffectSetMessage)
		}
	case EffectLoadDeck:
		if err := validateDeckID(effect.DeckID); err != nil {
			return fmt.Errorf("%s effect requires valid deckId: %w", EffectLoadDeck, err)
		}
	default:
		return fmt.Errorf("unsupported effect type %q", effect.Type)
	}
	return nil
}

func effectCardDefinition(effect RuleEffectDefinition, target CardMatcherDefinition, cardsByID map[string]CardDefinition) (CardDefinition, error) {
	cardID := resolvedEffectCardID(effect, target)
	if strings.TrimSpace(cardID) == "" {
		return CardDefinition{}, fmt.Errorf("effect %q requires cardId when target matcher has no id", effect.Type)
	}
	card, exists := cardsByID[cardID]
	if !exists {
		return CardDefinition{}, fmt.Errorf("effect %q references unknown card %q", effect.Type, cardID)
	}
	return card, nil
}

func resolvedEffectCardID(effect RuleEffectDefinition, target CardMatcherDefinition) string {
	if strings.TrimSpace(effect.CardID) != "" {
		return effect.CardID
	}
	return target.ID
}
