package game

import (
	"fmt"
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

type CardDefinitionID string
type CardInstanceID string
type EncounterID string
type Zone string

const (
	ZoneScene   Zone = "scene"
	ZoneLibrary Zone = "library"
	ZoneDiscard Zone = "discard"
)

type ZoneState map[Zone][]CardInstanceID

type CardInstance struct {
	InstanceID   CardInstanceID         `json:"id"`
	DefinitionID CardDefinitionID       `json:"definitionId"`
	Tags         []string               `json:"tags,omitempty"`
	State        map[string]any         `json:"state,omitempty"`
	Document     cardcomponent.Document `json:"document"`
	Actor        *ActorState            `json:"actor,omitempty"`
}

type ResourceState struct {
	Current int `json:"current"`
	Max     int `json:"max"`
}

type ActorDisposition string

const (
	DispositionNeutral  ActorDisposition = "neutral"
	DispositionFriendly ActorDisposition = "friendly"
	DispositionHostile  ActorDisposition = "hostile"
)

type ActorState struct {
	Integrity   ResourceState    `json:"integrity"`
	Charge      ResourceState    `json:"charge"`
	Disposition ActorDisposition `json:"disposition"`
	Statuses    []string         `json:"statuses,omitempty"`
}

type EncounterRole string

const (
	EncounterRolePlayer        EncounterRole = "player"
	EncounterRoleAlly          EncounterRole = "ally"
	EncounterRoleHostile       EncounterRole = "hostile"
	EncounterRoleProtected     EncounterRole = "protected"
	EncounterRoleEnvironmental EncounterRole = "environmental"
)

type EncounterPhase string

const (
	EncounterPhaseSetup      EncounterPhase = "setup"
	EncounterPhaseActive     EncounterPhase = "active"
	EncounterPhaseResolution EncounterPhase = "resolution"
	EncounterPhaseResolved   EncounterPhase = "resolved"
)

type EncounterParticipant struct {
	InstanceID CardInstanceID `json:"instanceId"`
	Role       EncounterRole  `json:"role"`
}

type EncounterState struct {
	ID           EncounterID            `json:"id"`
	Phase        EncounterPhase         `json:"phase"`
	Participants []EncounterParticipant `json:"participants"`
	Pressure     int                    `json:"pressure"`
	Outcome      string                 `json:"outcome,omitempty"`
}

type CardSnapshot struct {
	ID          CardInstanceID         `json:"id"`
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Tags        []string               `json:"tags,omitempty"`
	Collectible bool                   `json:"collectible"`
	Collected   bool                   `json:"collected,omitempty"`
	State       map[string]any         `json:"state,omitempty"`
	Document    cardcomponent.Document `json:"document"`
}

type CardMove struct {
	InstanceID CardInstanceID
	From       Zone
	To         Zone
	FromIndex  int
	ToIndex    int
}

type materializedDeck struct {
	definitions map[CardDefinitionID]CardDefinition
	instances   map[CardInstanceID]CardInstance
	zones       ZoneState
	activeID    CardInstanceID
}

func materializeDeck(definition DeckDefinition) (materializedDeck, error) {
	out := materializedDeck{
		definitions: make(map[CardDefinitionID]CardDefinition, len(definition.Cards)),
		instances:   make(map[CardInstanceID]CardInstance),
		zones: ZoneState{
			ZoneScene:   nil,
			ZoneLibrary: nil,
			ZoneDiscard: nil,
		},
	}
	for _, card := range definition.Cards {
		out.definitions[card.ID] = cloneValue(card)
	}
	for _, spec := range initialInstanceSpecs(definition) {
		card, ok := out.definitions[spec.DefinitionID]
		if !ok {
			return materializedDeck{}, fmt.Errorf("initial instance %q references unknown definition %q", spec.InstanceID, spec.DefinitionID)
		}
		document, ok := card.Documents[card.InitialDocument]
		if !ok {
			return materializedDeck{}, fmt.Errorf("card %q initial document variant %q does not exist", card.ID, card.InitialDocument)
		}
		document = cloneValue(document)
		document.CardID = string(spec.InstanceID)
		out.instances[spec.InstanceID] = CardInstance{
			InstanceID:   spec.InstanceID,
			DefinitionID: spec.DefinitionID,
			Tags:         append([]string(nil), card.Tags...),
			State:        cloneValue(card.State),
			Document:     document,
			Actor:        cloneValue(spec.Actor),
		}
		zone := spec.Zone
		if zone == "" {
			zone = ZoneScene
		}
		out.zones[zone] = append(out.zones[zone], spec.InstanceID)
	}
	activeID, err := initialActiveInstanceID(definition)
	if err != nil {
		return materializedDeck{}, err
	}
	out.activeID = activeID
	return out, nil
}

func validZone(zone Zone) bool {
	switch zone {
	case ZoneScene, ZoneLibrary, ZoneDiscard:
		return true
	default:
		return false
	}
}

func validDisposition(disposition ActorDisposition) bool {
	switch disposition {
	case DispositionNeutral, DispositionFriendly, DispositionHostile:
		return true
	default:
		return false
	}
}

func validEncounterRole(role EncounterRole) bool {
	switch role {
	case EncounterRolePlayer, EncounterRoleAlly, EncounterRoleHostile, EncounterRoleProtected, EncounterRoleEnvironmental:
		return true
	default:
		return false
	}
}

func validEncounterPhase(phase EncounterPhase) bool {
	switch phase {
	case EncounterPhaseSetup, EncounterPhaseActive, EncounterPhaseResolution, EncounterPhaseResolved:
		return true
	default:
		return false
	}
}

func validateResource(name string, resource ResourceState) error {
	if resource.Max < 0 {
		return fmt.Errorf("%s max cannot be negative", name)
	}
	if resource.Current < 0 || resource.Current > resource.Max {
		return fmt.Errorf("%s current must be between 0 and max", name)
	}
	return nil
}

func validateActor(actor ActorState) error {
	if err := validateResource("integrity", actor.Integrity); err != nil {
		return err
	}
	if err := validateResource("charge", actor.Charge); err != nil {
		return err
	}
	if !validDisposition(actor.Disposition) {
		return fmt.Errorf("unsupported actor disposition %q", actor.Disposition)
	}
	for _, status := range actor.Statuses {
		if strings.TrimSpace(status) == "" {
			return fmt.Errorf("actor status cannot be empty")
		}
	}
	return nil
}
