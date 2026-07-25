package game

import (
	"fmt"
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

func validateRuleDefinitions(
	registry *cardcomponent.Registry,
	rules []RuleDefinition,
	cardsByID map[string]CardDefinition,
	existingRuleIDs map[string]bool,
) error {
	seen := map[string]bool{}
	for id := range existingRuleIDs {
		seen[id] = true
	}
	for index, rule := range rules {
		id := strings.TrimSpace(rule.ID)
		if id == "" {
			return fmt.Errorf("rule at index %d must have an id", index)
		}
		if !cardcomponent.ValidComponentID(id) {
			return fmt.Errorf("rule id %q is invalid", rule.ID)
		}
		if seen[id] {
			return fmt.Errorf("duplicate rule id %q", id)
		}
		seen[id] = true
		if err := validateRuleDefinition(registry, rule, cardsByID); err != nil {
			return fmt.Errorf("rule %q: %w", id, err)
		}
	}
	return nil
}

func validateRuleDefinition(registry *cardcomponent.Registry, rule RuleDefinition, cardsByID map[string]CardDefinition) error {
	target, err := validateRuleTrigger(registry, rule.Trigger, cardsByID)
	if err != nil {
		return err
	}
	if len(rule.Effects) == 0 {
		return fmt.Errorf("effects are required")
	}
	for _, condition := range rule.Conditions {
		if err := validateRuleCondition(registry, rule.Trigger, condition); err != nil {
			return err
		}
	}
	for _, effect := range rule.Effects {
		if err := validateRuleEffect(registry, rule.Trigger, target, effect, cardsByID); err != nil {
			return err
		}
	}
	for _, effect := range rule.ElseEffects {
		if err := validateRuleEffect(registry, rule.Trigger, target, effect, cardsByID); err != nil {
			return fmt.Errorf("else effect: %w", err)
		}
	}
	return nil
}

func validateRuleTrigger(
	registry *cardcomponent.Registry,
	trigger RuleTrigger,
	cardsByID map[string]CardDefinition,
) (CardMatcherDefinition, error) {
	switch trigger.kind {
	case TriggerCardPlayed:
		if trigger.cardPlayed == nil {
			return CardMatcherDefinition{}, fmt.Errorf("%s trigger payload is missing", trigger.kind)
		}
		if err := validateMatcher("source", trigger.cardPlayed.Source, cardsByID); err != nil {
			return CardMatcherDefinition{}, err
		}
		if err := validateMatcher("target", trigger.cardPlayed.Target, cardsByID); err != nil {
			return CardMatcherDefinition{}, err
		}
		return trigger.cardPlayed.Target, nil
	case TriggerFormSubmitted:
		if trigger.formSubmitted == nil {
			return CardMatcherDefinition{}, fmt.Errorf("%s trigger payload is missing", trigger.kind)
		}
		if err := validateMatcher("target", trigger.formSubmitted.Target, cardsByID); err != nil {
			return CardMatcherDefinition{}, err
		}
		if !cardcomponent.ValidComponentID(trigger.formSubmitted.FormID) {
			return CardMatcherDefinition{}, fmt.Errorf("formId must contain only letters, numbers, hyphens, and underscores")
		}
		return trigger.formSubmitted.Target, nil
	case TriggerComponentUpdated:
		if trigger.componentUpdated == nil {
			return CardMatcherDefinition{}, fmt.Errorf("%s trigger payload is missing", trigger.kind)
		}
		if err := validateMatcher("target", trigger.componentUpdated.Target, cardsByID); err != nil {
			return CardMatcherDefinition{}, err
		}
		if strings.TrimSpace(trigger.componentUpdated.ComponentKind) == "" {
			return CardMatcherDefinition{}, fmt.Errorf("%s trigger requires componentKind", trigger.kind)
		}
		if _, ok := registry.Lookup(trigger.componentUpdated.ComponentKind); !ok {
			return CardMatcherDefinition{}, fmt.Errorf(
				"%s trigger componentKind %q is not registered",
				trigger.kind, trigger.componentUpdated.ComponentKind,
			)
		}
		if trigger.componentUpdated.ComponentID != "" && !cardcomponent.ValidComponentID(trigger.componentUpdated.ComponentID) {
			return CardMatcherDefinition{}, fmt.Errorf("%s trigger componentId %q is invalid", trigger.kind, trigger.componentUpdated.ComponentID)
		}
		return trigger.componentUpdated.Target, nil
	default:
		return CardMatcherDefinition{}, fmt.Errorf("unsupported trigger kind %q", trigger.kind)
	}
}

func validateRuleCondition(registry *cardcomponent.Registry, trigger RuleTrigger, condition RuleCondition) error {
	switch condition.kind {
	case ConditionFlagEquals:
		if condition.flagEquals == nil || strings.TrimSpace(condition.flagEquals.Flag) == "" {
			return fmt.Errorf("%s condition requires flag", condition.kind)
		}
	case ConditionComponentPresent:
		value := condition.componentPresent
		if value == nil {
			return fmt.Errorf("%s condition payload is missing", condition.kind)
		}
		if err := validateRuleCardReference(trigger.kind, value.Card); err != nil {
			return fmt.Errorf("%s condition: %w", condition.kind, err)
		}
		if err := validateComponentReference(registry, value.ComponentKind, value.ComponentID); err != nil {
			return fmt.Errorf("%s condition: %w", condition.kind, err)
		}
	case ConditionComponentPropertyEquals:
		value := condition.componentPropertyEquals
		if value == nil {
			return fmt.Errorf("%s condition payload is missing", condition.kind)
		}
		switch {
		case value.Component == RuleComponentTrigger:
			if value.Card != "" {
				return fmt.Errorf("%s condition cannot define both card and trigger component references", condition.kind)
			}
			if trigger.kind != TriggerComponentUpdated {
				return fmt.Errorf("%s condition trigger component is only valid for %s", condition.kind, TriggerComponentUpdated)
			}
			if trigger.componentUpdated == nil || value.ComponentKind != trigger.componentUpdated.ComponentKind {
				return fmt.Errorf("%s condition componentKind must match its trigger", condition.kind)
			}
			if value.ComponentID != "" && trigger.componentUpdated.ComponentID != "" && value.ComponentID != trigger.componentUpdated.ComponentID {
				return fmt.Errorf("%s condition componentId must match its trigger", condition.kind)
			}
		case value.Component == "":
			if err := validateRuleCardReference(trigger.kind, value.Card); err != nil {
				return fmt.Errorf("%s condition: %w", condition.kind, err)
			}
		default:
			return fmt.Errorf("%s condition has unsupported component reference %q", condition.kind, value.Component)
		}
		if err := validateComponentReference(registry, value.ComponentKind, value.ComponentID); err != nil {
			return fmt.Errorf("%s condition: %w", condition.kind, err)
		}
		definition, _ := registry.Lookup(value.ComponentKind)
		propertyKind, ok := definition.PropertyKind(value.Property)
		if !ok {
			return fmt.Errorf("%s condition references unknown property %q on %q", condition.kind, value.Property, value.ComponentKind)
		}
		if propertyKind != value.Value.Kind {
			return fmt.Errorf(
				"%s condition property %q on %q is %s, not %s",
				condition.kind, value.Property, value.ComponentKind, propertyKind, value.Value.Kind,
			)
		}
	case ConditionFormFieldEquals:
		value := condition.formFieldEquals
		if value == nil {
			return fmt.Errorf("%s condition payload is missing", condition.kind)
		}
		if trigger.kind != TriggerFormSubmitted {
			return fmt.Errorf("%s condition is only valid for %s", condition.kind, TriggerFormSubmitted)
		}
		if !cardcomponent.ValidComponentID(value.Name) {
			return fmt.Errorf("%s condition name %q is invalid", condition.kind, value.Name)
		}
		if len([]rune(value.Value)) > 128 {
			return fmt.Errorf("%s condition value must be at most 128 characters", condition.kind)
		}
	default:
		return fmt.Errorf("unsupported condition kind %q", condition.kind)
	}
	return nil
}

func validateRuleCardReference(triggerKind RuleTriggerKind, reference string) error {
	switch reference {
	case RuleCardTarget:
		return nil
	case RuleCardSource:
		if triggerKind != TriggerCardPlayed {
			return fmt.Errorf("source reference is only valid for %s", TriggerCardPlayed)
		}
		return nil
	default:
		return fmt.Errorf("unsupported card reference %q", reference)
	}
}

func validateComponentReference(registry *cardcomponent.Registry, componentKind, componentID string) error {
	if strings.TrimSpace(componentKind) == "" {
		return fmt.Errorf("componentKind is required")
	}
	if _, ok := registry.Lookup(componentKind); !ok {
		return fmt.Errorf("componentKind %q is not registered", componentKind)
	}
	if componentID != "" && !cardcomponent.ValidComponentID(componentID) {
		return fmt.Errorf("componentId %q is invalid", componentID)
	}
	return nil
}

func validateRuleEffect(
	registry *cardcomponent.Registry,
	trigger RuleTrigger,
	target CardMatcherDefinition,
	effect RuleEffect,
	cardsByID map[string]CardDefinition,
) error {
	switch effect.kind {
	case EffectSetFlag:
		if effect.setFlag == nil || strings.TrimSpace(effect.setFlag.Flag) == "" {
			return fmt.Errorf("%s effect requires flag", effect.kind)
		}
	case EffectSetCardState:
		value := effect.setCardState
		if value == nil || strings.TrimSpace(value.Key) == "" {
			return fmt.Errorf("%s effect requires key", effect.kind)
		}
		if value.Value == nil {
			return fmt.Errorf("%s effect requires value", effect.kind)
		}
		if _, _, err := ruleEffectCard(value.CardID, target, cardsByID, effect.kind); err != nil {
			return err
		}
	case EffectRemoveCardTags:
		value := effect.removeCardTags
		if value == nil || len(value.Tags) == 0 {
			return fmt.Errorf("%s effect requires tags", effect.kind)
		}
		for _, tag := range value.Tags {
			if strings.TrimSpace(tag) == "" {
				return fmt.Errorf("%s effect contains an empty tag", effect.kind)
			}
		}
		if _, _, err := ruleEffectCard(value.CardID, target, cardsByID, effect.kind); err != nil {
			return err
		}
	case EffectSetDocumentVariant:
		value := effect.setDocumentVariant
		if value == nil || strings.TrimSpace(value.Variant) == "" {
			return fmt.Errorf("%s effect requires variant", effect.kind)
		}
		cardID, card, err := ruleEffectCard(value.CardID, target, cardsByID, effect.kind)
		if err != nil {
			return err
		}
		if _, exists := card.Documents[value.Variant]; !exists {
			return fmt.Errorf("%s effect references missing variant %q for card %q", effect.kind, value.Variant, cardID)
		}
	case EffectSetMessage:
		if effect.setMessage == nil || strings.TrimSpace(effect.setMessage.Message) == "" {
			return fmt.Errorf("%s effect requires message", effect.kind)
		}
	case EffectLoadDeck:
		if effect.loadDeck == nil {
			return fmt.Errorf("%s effect payload is missing", effect.kind)
		}
		if err := validateDeckID(effect.loadDeck.DeckID); err != nil {
			return fmt.Errorf("%s effect requires valid deckId: %w", effect.kind, err)
		}
	case EffectCopyComponent:
		value := effect.copyComponent
		if value == nil {
			return fmt.Errorf("%s effect payload is missing", effect.kind)
		}
		switch value.Source {
		case RuleCardSource:
			if trigger.kind != TriggerCardPlayed {
				return fmt.Errorf("%s effect source %q is only valid for %s", effect.kind, value.Source, TriggerCardPlayed)
			}
		case RuleComponentTrigger:
			if trigger.kind != TriggerComponentUpdated {
				return fmt.Errorf("%s effect source %q is only valid for %s", effect.kind, value.Source, TriggerComponentUpdated)
			}
			if trigger.componentUpdated == nil || value.ComponentKind != trigger.componentUpdated.ComponentKind {
				return fmt.Errorf("%s effect componentKind must match its trigger", effect.kind)
			}
		default:
			return fmt.Errorf("%s effect has unsupported source %q", effect.kind, value.Source)
		}
		if err := validateComponentReference(registry, value.ComponentKind, value.SourceComponentID); err != nil {
			return fmt.Errorf("%s effect: %w", effect.kind, err)
		}
		definition, _ := registry.Lookup(value.ComponentKind)
		if _, ok := definition.Install(); !ok {
			return fmt.Errorf("%s effect requires an installable componentKind", effect.kind)
		}
		if value.ComponentID != "" && !cardcomponent.ValidComponentID(value.ComponentID) {
			return fmt.Errorf("%s effect componentId %q is invalid", effect.kind, value.ComponentID)
		}
		if _, _, err := ruleEffectCard(value.CardID, target, cardsByID, effect.kind); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported effect kind %q", effect.kind)
	}
	return nil
}

func ruleEffectCard(
	explicitCardID string,
	target CardMatcherDefinition,
	cardsByID map[string]CardDefinition,
	effectKind RuleEffectKind,
) (string, CardDefinition, error) {
	cardID := strings.TrimSpace(explicitCardID)
	if cardID == "" {
		cardID = strings.TrimSpace(target.ID)
	}
	if cardID == "" {
		return "", CardDefinition{}, fmt.Errorf("%s effect requires cardId when target matcher has no id", effectKind)
	}
	card, exists := cardsByID[cardID]
	if !exists {
		return "", CardDefinition{}, fmt.Errorf("%s effect references unknown card %q", effectKind, cardID)
	}
	return cardID, card, nil
}
