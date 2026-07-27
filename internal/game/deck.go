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
	ID                      string             `json:"id"`
	Name                    string             `json:"name"`
	InitialActiveCardID     string             `json:"initialActiveCardId,omitempty"`
	InitialActiveInstanceID CardInstanceID     `json:"initialActiveInstanceId,omitempty"`
	InitialInstances        []CardInstanceSpec `json:"initialInstances,omitempty"`
	InitialMessage          string             `json:"initialMessage"`
	InitialSolvedFlags      map[string]bool    `json:"initialSolvedFlags,omitempty"`
	Cards                   []CardDefinition   `json:"cards"`
	Rules                   []RuleDefinition   `json:"rules,omitempty"`
}

type CardDefinition struct {
	ID              CardDefinitionID                  `json:"id"`
	Name            string                            `json:"name"`
	Kind            string                            `json:"kind"`
	Tags            []string                          `json:"tags,omitempty"`
	Collectible     bool                              `json:"collectible"`
	State           map[string]any                    `json:"state,omitempty"`
	InitialDocument string                            `json:"initialDocument"`
	Documents       map[string]cardcomponent.Document `json:"documents"`
}

type CardInstanceSpec struct {
	InstanceID   CardInstanceID   `json:"id"`
	DefinitionID CardDefinitionID `json:"definitionId"`
	Zone         Zone             `json:"zone,omitempty"`
	Actor        *ActorState      `json:"actor,omitempty"`
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
		ID                      string             `json:"id"`
		Name                    string             `json:"name"`
		InitialActiveCardID     string             `json:"initialActiveCardId,omitempty"`
		InitialActiveInstanceID CardInstanceID     `json:"initialActiveInstanceId,omitempty"`
		InitialInstances        []CardInstanceSpec `json:"initialInstances,omitempty"`
		InitialMessage          string             `json:"initialMessage"`
		InitialSolvedFlags      map[string]bool    `json:"initialSolvedFlags,omitempty"`
		Cards                   []cardWire         `json:"cards"`
		Rules                   []RuleDefinition   `json:"rules,omitempty"`
	}
	var wire deckWire
	if err := decodeStrictJSON(raw, &wire); err != nil {
		return DeckDefinition{}, fmt.Errorf("decode deck definition: %w", err)
	}
	definition := DeckDefinition{
		ID: wire.ID, Name: wire.Name, InitialActiveCardID: wire.InitialActiveCardID,
		InitialActiveInstanceID: wire.InitialActiveInstanceID, InitialInstances: wire.InitialInstances,
		InitialMessage: wire.InitialMessage, InitialSolvedFlags: wire.InitialSolvedFlags, Rules: wire.Rules,
	}
	for _, cardWire := range wire.Cards {
		card := CardDefinition{ID: CardDefinitionID(cardWire.ID), Name: cardWire.Name, Kind: cardWire.Kind, Tags: cardWire.Tags, Collectible: cardWire.Collectible, State: cardWire.State, InitialDocument: cardWire.InitialDocument, Documents: map[string]cardcomponent.Document{}}
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
	instancesByID, err := validateInitialInstances(definition, cardsByID, nil)
	if err != nil {
		return err
	}
	return validateRuleDefinitions(registry, definition.Rules, cardsByID, instancesByID, nil)
}

func ValidateDeckPackDefinition(
	registry *cardcomponent.Registry,
	definition DeckDefinition,
	existingCards map[CardDefinitionID]CardDefinition,
	existingInstances map[CardInstanceID]CardDefinitionID,
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
	instancesByID, err := validateInitialInstances(definition, cardsByID, existingInstances)
	if err != nil {
		return err
	}
	return validateRuleDefinitions(registry, definition.Rules, cardsByID, instancesByID, existingRuleIDs)
}

func validateDeckCards(registry *cardcomponent.Registry, definition DeckDefinition) (map[CardDefinitionID]CardDefinition, error) {
	if strings.TrimSpace(definition.ID) == "" {
		return nil, fmt.Errorf("deck id is required")
	}
	if err := validateDeckID(definition.ID); err != nil {
		return nil, err
	}
	if len(definition.Cards) == 0 {
		return nil, fmt.Errorf("deck %q must contain at least one card", definition.ID)
	}
	cardsByID := make(map[CardDefinitionID]CardDefinition, len(definition.Cards))
	hasPersistentCard := false
	for index, card := range definition.Cards {
		if strings.TrimSpace(string(card.ID)) == "" {
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
			if document.CardID != string(card.ID) {
				return nil, fmt.Errorf("card %q document variant %q has card_id %q", card.ID, variant, document.CardID)
			}
			if document.Root.ComponentKind != cardcomponent.Kind {
				return nil, fmt.Errorf("card %q document variant %q root type must be %q", card.ID, variant, cardcomponent.Kind)
			}
			if _, issues := registry.CanonicalizeDocument(document); len(issues) > 0 {
				return nil, fmt.Errorf("card %q document variant %q at %s: %s", card.ID, variant, issues[0].Path, issues[0].Message)
			}
		}
		if err := validateComponentCardState(registry, string(card.ID), card.State); err != nil {
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
	if _, err := validateInitialInstances(definition, cardsByID, nil); err != nil {
		return nil, err
	}
	return cardsByID, nil
}

func initialInstanceSpecs(definition DeckDefinition) []CardInstanceSpec {
	if definition.InitialInstances != nil {
		return cloneValue(definition.InitialInstances)
	}
	specs := make([]CardInstanceSpec, 0, len(definition.Cards))
	for _, card := range definition.Cards {
		specs = append(specs, CardInstanceSpec{
			InstanceID:   CardInstanceID(card.ID),
			DefinitionID: card.ID,
			Zone:         ZoneScene,
		})
	}
	return specs
}

func initialActiveInstanceID(definition DeckDefinition) (CardInstanceID, error) {
	if definition.InitialInstances == nil {
		if strings.TrimSpace(definition.InitialActiveCardID) == "" {
			return "", fmt.Errorf("initialActiveCardId is required when initialInstances is absent")
		}
		if definition.InitialActiveInstanceID != "" {
			return "", fmt.Errorf("initialActiveInstanceId is only valid when initialInstances is present")
		}
		return CardInstanceID(definition.InitialActiveCardID), nil
	}
	if strings.TrimSpace(definition.InitialActiveCardID) != "" {
		return "", fmt.Errorf("initialActiveCardId is only valid when initialInstances is absent")
	}
	if strings.TrimSpace(string(definition.InitialActiveInstanceID)) == "" {
		return "", fmt.Errorf("initialActiveInstanceId is required when initialInstances is present")
	}
	return definition.InitialActiveInstanceID, nil
}

func validateInitialInstances(
	definition DeckDefinition,
	cardsByID map[CardDefinitionID]CardDefinition,
	existing map[CardInstanceID]CardDefinitionID,
) (map[CardInstanceID]CardDefinitionID, error) {
	instances := make(map[CardInstanceID]CardDefinitionID, len(existing)+len(definition.Cards))
	for instanceID, definitionID := range existing {
		instances[instanceID] = definitionID
	}
	sceneCount := 0
	currentSpecs := make(map[CardInstanceID]Zone)
	for index, spec := range initialInstanceSpecs(definition) {
		if strings.TrimSpace(string(spec.InstanceID)) == "" {
			return nil, fmt.Errorf("initial instance at index %d must have an id", index)
		}
		if _, exists := instances[spec.InstanceID]; exists {
			return nil, fmt.Errorf("duplicate card instance id %q", spec.InstanceID)
		}
		if _, exists := cardsByID[spec.DefinitionID]; !exists {
			return nil, fmt.Errorf("initial instance %q references unknown definition %q", spec.InstanceID, spec.DefinitionID)
		}
		zone := spec.Zone
		if zone == "" {
			zone = ZoneScene
		}
		if !validZone(zone) {
			return nil, fmt.Errorf("initial instance %q uses unsupported zone %q", spec.InstanceID, zone)
		}
		if spec.Actor != nil {
			if err := validateActor(*spec.Actor); err != nil {
				return nil, fmt.Errorf("initial instance %q actor: %w", spec.InstanceID, err)
			}
		}
		if zone == ZoneScene {
			sceneCount++
		}
		instances[spec.InstanceID] = spec.DefinitionID
		currentSpecs[spec.InstanceID] = zone
	}
	if sceneCount == 0 {
		return nil, fmt.Errorf("deck %q must contain at least one scene instance", definition.ID)
	}
	activeID, err := initialActiveInstanceID(definition)
	if err != nil {
		return nil, err
	}
	_, exists := currentSpecs[activeID]
	if !exists {
		return nil, fmt.Errorf("initial active instance %q does not exist in deck %q", activeID, definition.ID)
	}
	if currentSpecs[activeID] != ZoneScene {
		return nil, fmt.Errorf("initial active instance %q is not in the scene zone", activeID)
	}
	return instances, nil
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

func validateMatcher(name string, matcher CardMatcherDefinition, cardsByID map[CardDefinitionID]CardDefinition) error {
	if strings.TrimSpace(matcher.ID) == "" && len(matcher.Tags) == 0 {
		return fmt.Errorf("%s matcher must define an id or tags", name)
	}
	if strings.TrimSpace(matcher.ID) != "" {
		if _, exists := cardsByID[CardDefinitionID(matcher.ID)]; !exists {
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
