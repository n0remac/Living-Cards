package game

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

//go:embed decks/*.json
var deckFiles embed.FS

const (
	SeededWorldDeckDefinition = "seeded_world"
	FuseRoomDeckDefinition    = "fuse_room"
	GeneratorDeckDefinition   = "generator_room"
	ArchiveTerminalDefinition = "archive_terminal"
)

type DeckDefinition struct {
	ID                  string           `json:"id"`
	Name                string           `json:"name"`
	InitialActiveCardID string           `json:"initialActiveCardId"`
	InitialMessage      string           `json:"initialMessage"`
	InitialSolvedFlags  map[string]bool  `json:"initialSolvedFlags,omitempty"`
	Cards               []CardDefinition `json:"cards"`
	Rules               []RuleDefinition `json:"rules,omitempty"`
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

func LoadEmbeddedSeededWorldDeck(registry *cardcomponent.Registry) (DeckDefinition, error) {
	definition, err := LoadEmbeddedDeck(registry, SeededWorldDeckDefinition)
	if err != nil {
		return DeckDefinition{}, err
	}
	if err := ValidateDeckDefinition(registry, definition); err != nil {
		return DeckDefinition{}, err
	}
	return definition, nil
}

func LoadEmbeddedDeck(registry *cardcomponent.Registry, deckID string) (DeckDefinition, error) {
	deckID = strings.TrimSpace(deckID)
	if err := validateDeckID(deckID); err != nil {
		return DeckDefinition{}, err
	}
	raw, err := deckFiles.ReadFile("decks/" + deckID + ".json")
	if err != nil {
		return DeckDefinition{}, fmt.Errorf("read embedded deck %q: %w", deckID, err)
	}
	definition, err := decodeDeckDefinition(registry, raw)
	if err != nil {
		return DeckDefinition{}, err
	}
	if definition.ID != deckID {
		return DeckDefinition{}, fmt.Errorf("embedded deck %q has id %q", deckID, definition.ID)
	}
	if _, err := validateDeckCards(registry, definition); err != nil {
		return DeckDefinition{}, err
	}
	return definition, nil
}

func DecodeDeckDefinition(registry *cardcomponent.Registry, raw []byte) (DeckDefinition, error) {
	definition, err := decodeDeckDefinition(registry, raw)
	if err != nil {
		return DeckDefinition{}, err
	}
	if err := ValidateDeckDefinition(registry, definition); err != nil {
		return DeckDefinition{}, err
	}
	return definition, nil
}

func decodeDeckDefinition(registry *cardcomponent.Registry, raw []byte) (DeckDefinition, error) {
	type cardWire struct {
		ID              string                     `json:"id"`
		Name            string                     `json:"name"`
		Kind            string                     `json:"kind"`
		Tags            []string                   `json:"tags,omitempty"`
		Collectible     bool                       `json:"collectible"`
		State           map[string]any             `json:"state,omitempty"`
		InitialDocument string                     `json:"initialDocument"`
		Documents       map[string]json.RawMessage `json:"documents"`
	}
	type deckWire struct {
		ID                  string           `json:"id"`
		Name                string           `json:"name"`
		InitialActiveCardID string           `json:"initialActiveCardId"`
		InitialMessage      string           `json:"initialMessage"`
		InitialSolvedFlags  map[string]bool  `json:"initialSolvedFlags,omitempty"`
		Cards               []cardWire       `json:"cards"`
		Rules               []RuleDefinition `json:"rules,omitempty"`
	}
	var wire deckWire
	if err := decodeStrictJSON(raw, &wire); err != nil {
		return DeckDefinition{}, fmt.Errorf("decode deck definition: %w", err)
	}
	definition := DeckDefinition{
		ID: wire.ID, Name: wire.Name, InitialActiveCardID: wire.InitialActiveCardID,
		InitialMessage: wire.InitialMessage, InitialSolvedFlags: wire.InitialSolvedFlags, Rules: wire.Rules,
	}
	for _, cardWire := range wire.Cards {
		card := CardDefinition{ID: cardWire.ID, Name: cardWire.Name, Kind: cardWire.Kind, Tags: cardWire.Tags, Collectible: cardWire.Collectible, State: cardWire.State, InitialDocument: cardWire.InitialDocument, Documents: map[string]cardcomponent.Document{}}
		for variant, documentRaw := range cardWire.Documents {
			document, issues := registry.DecodeDocument(documentRaw)
			if len(issues) > 0 {
				return DeckDefinition{}, fmt.Errorf("card %q document %q at %s: %s", card.ID, variant, issues[0].Path, issues[0].Message)
			}
			card.Documents[variant] = document
		}
		definition.Cards = append(definition.Cards, card)
	}
	return definition, nil
}

func ValidateDeckDefinition(registry *cardcomponent.Registry, definition DeckDefinition) error {
	cardsByID, err := validateDeckCards(registry, definition)
	if err != nil {
		return err
	}
	return validateRuleDefinitions(registry, definition.Rules, cardsByID, nil)
}

func ValidateDeckPackDefinition(
	registry *cardcomponent.Registry,
	definition DeckDefinition,
	existingCards map[string]CardDefinition,
	existingRuleIDs map[string]bool,
) error {
	cardsByID, err := validateDeckCards(registry, definition)
	if err != nil {
		return err
	}
	for cardID, card := range existingCards {
		if _, exists := cardsByID[cardID]; exists {
			return fmt.Errorf("deck %q card %q already exists", definition.ID, cardID)
		}
		cardsByID[cardID] = card
	}
	return validateRuleDefinitions(registry, definition.Rules, cardsByID, existingRuleIDs)
}

func validateDeckCards(registry *cardcomponent.Registry, definition DeckDefinition) (map[string]CardDefinition, error) {
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
	hasPersistentCard := false
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
			if document.Root.ComponentKind != cardcomponent.Kind {
				return nil, fmt.Errorf("card %q document variant %q root type must be %q", card.ID, variant, cardcomponent.Kind)
			}
			if _, issues := registry.CanonicalizeDocument(document); len(issues) > 0 {
				return nil, fmt.Errorf("card %q document variant %q at %s: %s", card.ID, variant, issues[0].Path, issues[0].Message)
			}
		}
		if err := validateComponentCardState(registry, card.ID, card.State); err != nil {
			return nil, err
		}
		if !card.Collectible {
			hasPersistentCard = true
		}
		cardsByID[card.ID] = card
	}
	if !hasPersistentCard {
		return nil, fmt.Errorf("deck %q must contain at least one non-collectible card", definition.ID)
	}
	if _, exists := cardsByID[definition.InitialActiveCardID]; !exists {
		return nil, fmt.Errorf("initial active card %q does not exist", definition.InitialActiveCardID)
	}
	return cardsByID, nil
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

func decodeStrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
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
