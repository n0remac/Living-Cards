package game

import (
	"fmt"
	"math"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func (s *Session) attackCreature(signal ruleSignal) (CreatureAttackedPayload, ComponentUpdatedSignal, error) {
	played, ok := signal.(CardPlayedSignal)
	if !ok {
		return CreatureAttackedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("%s effect requires %s", EffectAttackCreature, TriggerCardPlayed)
	}
	sourceIndex := s.libraryCardIndex(played.SourceCardID)
	if sourceIndex < 0 {
		return CreatureAttackedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("source card %q is not in the library", played.SourceCardID)
	}
	source := s.library[sourceIndex]
	if findRuleComponent(source.Document, cardcomponent.KindCreature, "") == nil {
		return CreatureAttackedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("source card %q has no creature component", source.ID)
	}
	attack, err := s.cardAttackPower(source)
	if err != nil {
		return CreatureAttackedPayload{}, ComponentUpdatedSignal{}, err
	}

	var previousHealth, health int
	var updated cardcomponent.Node
	if err := s.updateRuleWorldCard(played.TargetCardID, EffectAttackCreature, func(target *Card) error {
		creature := findNodeByKindPtr(&target.Document.Root, cardcomponent.KindCreature)
		if creature == nil {
			return fmt.Errorf("target card %q has no creature component", target.ID)
		}
		definition, ok := s.registry.Lookup(cardcomponent.KindCreature)
		if !ok {
			return fmt.Errorf("component kind %q is not registered", cardcomponent.KindCreature)
		}
		value, present, issues := definition.ReadProperty(creature.Config, "health")
		if len(issues) > 0 {
			return fmt.Errorf("read creature health: %s", issues[0].Message)
		}
		if !present || value.Kind != schema.PropertyNumber || math.Trunc(value.Number) != value.Number {
			return fmt.Errorf("creature health is not an integer property")
		}
		previousHealth = int(value.Number)
		health = calculateCreatureHealth(previousHealth, attack)
		next, writable, issues := definition.WriteProperty(creature.Config, "health", schema.NumberValue(float64(health)))
		if len(issues) > 0 {
			return fmt.Errorf("write creature health: %s", issues[0].Message)
		}
		if !writable {
			return fmt.Errorf("creature health is not writable")
		}
		creature.Config = next
		updated = cloneValue(*creature)
		return nil
	}); err != nil {
		return CreatureAttackedPayload{}, ComponentUpdatedSignal{}, err
	}
	payload := CreatureAttackedPayload{
		SourceCardID: source.ID, TargetCardID: played.TargetCardID, Attack: attack,
		PreviousHealth: previousHealth, Health: health,
	}
	followUp := ComponentUpdatedSignal{
		CardID: played.TargetCardID, ComponentID: updated.ID,
		ComponentKind: updated.ComponentKind, Component: updated,
	}
	return payload, followUp, nil
}

func (s *Session) cardAttackPower(source Card) (int, error) {
	definition, ok := s.registry.Lookup(cardcomponent.KindAttack)
	if !ok {
		return 0, fmt.Errorf("component kind %q is not registered", cardcomponent.KindAttack)
	}
	power, count := 0, 0
	var invalid bool
	visitNodes(source.Document.Root, func(node cardcomponent.Node) {
		if node.ComponentKind != cardcomponent.KindAttack {
			return
		}
		count++
		value, present, issues := definition.ReadProperty(node.Config, "power")
		if len(issues) > 0 || !present || value.Kind != schema.PropertyNumber || math.Trunc(value.Number) != value.Number {
			invalid = true
			return
		}
		power += int(value.Number)
	})
	if count == 0 {
		return 0, fmt.Errorf("source card %q has no attack component", source.ID)
	}
	if invalid {
		return 0, fmt.Errorf("source card %q has an invalid attack power", source.ID)
	}
	return power, nil
}

func calculateCreatureHealth(health, attack int) int {
	if attack >= health {
		return 0
	}
	return health - attack
}
