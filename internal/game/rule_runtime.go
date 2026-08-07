package game

import (
	"encoding/json"
	"fmt"
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const maxProcessedRuleSignals = 32

func (s *Session) runRules(initial ruleSignal, events *eventCollector) (RuleResolution, error) {
	queue := []ruleSignal{initial}
	initialResolution := RuleResolution{TriggerKind: initial.triggerKind(), Outcome: RuleOutcomeNoMatch}
	for processed := 0; len(queue) > 0; processed++ {
		if processed >= maxProcessedRuleSignals {
			return RuleResolution{}, fmt.Errorf("rule signal limit of %d exceeded", maxProcessedRuleSignals)
		}
		signal := queue[0]
		queue = queue[1:]
		resolution, followUps, err := s.resolveRuleSignal(signal, events)
		if err != nil {
			return RuleResolution{}, err
		}
		if processed == 0 {
			initialResolution = resolution
		}
		queue = append(queue, followUps...)
	}
	return initialResolution, nil
}

func (s *Session) resolveRuleSignal(signal ruleSignal, events *eventCollector) (RuleResolution, []ruleSignal, error) {
	var firstTriggerMatch *RuleDefinition
	var firstFallback *RuleDefinition
	var selected *RuleDefinition
	for index := range s.rules {
		rule := &s.rules[index]
		matches, err := s.ruleTriggerMatches(rule.Trigger, signal)
		if err != nil {
			return RuleResolution{}, nil, err
		}
		if !matches {
			continue
		}
		if firstTriggerMatch == nil {
			firstTriggerMatch = rule
		}
		if firstFallback == nil && len(rule.ElseEffects) > 0 {
			firstFallback = rule
		}
		conditionsMatch, err := s.ruleConditionsMatch(rule.Conditions, signal)
		if err != nil {
			return RuleResolution{}, nil, err
		}
		if conditionsMatch {
			selected = rule
			break
		}
	}
	if selected != nil {
		followUps, err := s.applyRuleEffects(selected.Effects, signal, events)
		if err != nil {
			return RuleResolution{}, nil, err
		}
		resolution := RuleResolution{RuleID: selected.ID, TriggerKind: signal.triggerKind(), Outcome: RuleOutcomeSuccess, RetainSource: selected.RetainSource}
		events.emit(EventRuleResolved, RuleResolvedPayload{
			RuleID: selected.ID, TriggerKind: string(signal.triggerKind()), Outcome: string(resolution.Outcome),
		})
		return resolution, followUps, nil
	}
	if firstTriggerMatch == nil {
		return RuleResolution{TriggerKind: signal.triggerKind(), Outcome: RuleOutcomeNoMatch}, nil, nil
	}
	resolved := firstTriggerMatch
	var followUps []ruleSignal
	if firstFallback != nil {
		resolved = firstFallback
		var err error
		followUps, err = s.applyRuleEffects(resolved.ElseEffects, signal, events)
		if err != nil {
			return RuleResolution{}, nil, err
		}
	}
	resolution := RuleResolution{RuleID: resolved.ID, TriggerKind: signal.triggerKind(), Outcome: RuleOutcomeConditionsFailed, RetainSource: resolved.RetainSource}
	events.emit(EventRuleResolved, RuleResolvedPayload{
		RuleID: resolved.ID, TriggerKind: string(signal.triggerKind()), Outcome: string(resolution.Outcome),
	})
	return resolution, followUps, nil
}

func (s *Session) ruleTriggerMatches(trigger RuleTrigger, signal ruleSignal) (bool, error) {
	if trigger.kind != signal.triggerKind() {
		return false, nil
	}
	switch typed := signal.(type) {
	case CardPlayedSignal:
		if trigger.cardPlayed == nil {
			return false, fmt.Errorf("%s trigger payload is missing", trigger.kind)
		}
		sourceIndex := s.libraryCardIndex(typed.SourceCardID)
		targetIndex := s.worldCardIndex(typed.TargetCardID)
		if sourceIndex < 0 || targetIndex < 0 {
			return false, nil
		}
		return cardMatches(s.library[sourceIndex], trigger.cardPlayed.Source) &&
			cardMatches(s.worldDeck[targetIndex], trigger.cardPlayed.Target), nil
	case FormSubmittedSignal:
		if trigger.formSubmitted == nil || trigger.formSubmitted.FormID != typed.FormID {
			return false, nil
		}
		targetIndex := s.worldCardIndex(typed.CardID)
		return targetIndex >= 0 && cardMatches(s.worldDeck[targetIndex], trigger.formSubmitted.Target), nil
	case ComponentUpdatedSignal:
		if trigger.componentUpdated == nil ||
			trigger.componentUpdated.ComponentKind != typed.ComponentKind ||
			(trigger.componentUpdated.ComponentID != "" && trigger.componentUpdated.ComponentID != typed.ComponentID) {
			return false, nil
		}
		targetIndex := s.worldCardIndex(typed.CardID)
		return targetIndex >= 0 && cardMatches(s.worldDeck[targetIndex], trigger.componentUpdated.Target), nil
	default:
		return false, fmt.Errorf("unsupported rule signal %T", signal)
	}
}

func (s *Session) ruleConditionsMatch(conditions []RuleCondition, signal ruleSignal) (bool, error) {
	for _, condition := range conditions {
		matches, err := s.ruleConditionMatches(condition, signal)
		if err != nil {
			return false, err
		}
		if !matches {
			return false, nil
		}
	}
	return true, nil
}

func (s *Session) ruleConditionMatches(condition RuleCondition, signal ruleSignal) (bool, error) {
	switch condition.kind {
	case ConditionFlagEquals:
		return condition.flagEquals != nil && s.solvedFlags[condition.flagEquals.Flag] == condition.flagEquals.Value, nil
	case ConditionComponentPresent:
		if condition.componentPresent == nil {
			return false, fmt.Errorf("%s condition payload is missing", condition.kind)
		}
		card, ok := s.ruleSignalCard(signal, condition.componentPresent.Card)
		if !ok {
			return false, nil
		}
		return findRuleComponent(card.Document, condition.componentPresent.ComponentKind, condition.componentPresent.ComponentID) != nil, nil
	case ConditionComponentPropertyEquals:
		if condition.componentPropertyEquals == nil {
			return false, fmt.Errorf("%s condition payload is missing", condition.kind)
		}
		value := condition.componentPropertyEquals
		var node *cardcomponent.Node
		if value.Component == RuleComponentTrigger {
			updated, ok := signal.(ComponentUpdatedSignal)
			if !ok {
				return false, nil
			}
			candidate := cloneValue(updated.Component)
			if candidate.ComponentKind == value.ComponentKind &&
				(value.ComponentID == "" || candidate.ID == value.ComponentID) {
				node = &candidate
			}
		} else {
			card, ok := s.ruleSignalCard(signal, value.Card)
			if !ok {
				return false, nil
			}
			node = findRuleComponent(card.Document, value.ComponentKind, value.ComponentID)
		}
		if node == nil {
			return false, nil
		}
		definition, ok := s.registry.Lookup(value.ComponentKind)
		if !ok {
			return false, fmt.Errorf("component kind %q is not registered", value.ComponentKind)
		}
		actual, present, issues := definition.ReadProperty(node.Config, value.Property)
		if len(issues) > 0 {
			return false, fmt.Errorf("read %s property %q: %s", value.ComponentKind, value.Property, issues[0].Message)
		}
		return present && rulePropertyEquals(actual, value.Value), nil
	case ConditionFormFieldEquals:
		if condition.formFieldEquals == nil {
			return false, fmt.Errorf("%s condition payload is missing", condition.kind)
		}
		submitted, ok := signal.(FormSubmittedSignal)
		if !ok {
			return false, nil
		}
		actual, exists := submitted.Fields[condition.formFieldEquals.Name]
		if !exists {
			return false, nil
		}
		expected := condition.formFieldEquals.Value
		if condition.formFieldEquals.TrimSpace {
			actual = strings.TrimSpace(actual)
			expected = strings.TrimSpace(expected)
		}
		if condition.formFieldEquals.CaseSensitive {
			return actual == expected, nil
		}
		return strings.EqualFold(actual, expected), nil
	default:
		return false, fmt.Errorf("unsupported condition kind %q", condition.kind)
	}
}

func (s *Session) ruleSignalCard(signal ruleSignal, reference string) (Card, bool) {
	var cardID string
	switch typed := signal.(type) {
	case CardPlayedSignal:
		switch reference {
		case RuleCardSource:
			index := s.libraryCardIndex(typed.SourceCardID)
			if index < 0 {
				return Card{}, false
			}
			return s.library[index], true
		case RuleCardTarget:
			cardID = typed.TargetCardID
		}
	case FormSubmittedSignal:
		if reference == RuleCardTarget {
			cardID = typed.CardID
		}
	case ComponentUpdatedSignal:
		if reference == RuleCardTarget {
			cardID = typed.CardID
		}
	}
	if cardID == "" {
		return Card{}, false
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return Card{}, false
	}
	return s.worldDeck[index], true
}

func findRuleComponent(document cardcomponent.Document, componentKind, componentID string) *cardcomponent.Node {
	if componentID != "" {
		node := findNodeByID(document.Root, componentID)
		if node == nil || node.ComponentKind != componentKind {
			return nil
		}
		return node
	}
	return findNodeByKind(document.Root, componentKind)
}

func rulePropertyEquals(actual schema.PropertyValue, expected RuleValue) bool {
	if actual.Kind != expected.Kind {
		return false
	}
	switch actual.Kind {
	case schema.PropertyString:
		return actual.String == expected.String
	case schema.PropertyNumber:
		return actual.Number == expected.Number
	case schema.PropertyBool:
		return actual.Bool == expected.Bool
	default:
		return false
	}
}

func (s *Session) applyRuleEffects(effects []RuleEffect, signal ruleSignal, events *eventCollector) ([]ruleSignal, error) {
	var followUps []ruleSignal
	for _, effect := range effects {
		switch effect.kind {
		case EffectSetFlag:
			value := effect.setFlag
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			if s.solvedFlags == nil {
				s.solvedFlags = map[string]bool{}
			}
			s.solvedFlags[value.Flag] = value.Value
			events.emit(EventFlagChanged, FlagChangedPayload{Flag: value.Flag, Value: value.Value})
		case EffectSetCardState:
			value := effect.setCardState
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			cardID, err := s.ruleEffectTargetCardID(value.CardID, signal)
			if err != nil {
				return nil, err
			}
			if err := s.updateRuleWorldCard(cardID, effect.kind, func(card *Card) error {
				if card.State == nil {
					card.State = map[string]any{}
				}
				card.State[value.Key] = cloneValue(value.Value)
				return nil
			}); err != nil {
				return nil, err
			}
			events.emit(EventCardStateChanged, CardStateChangedPayload{CardID: cardID, Key: value.Key, Value: cloneValue(value.Value)})
		case EffectRemoveCardTags:
			value := effect.removeCardTags
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			cardID, err := s.ruleEffectTargetCardID(value.CardID, signal)
			if err != nil {
				return nil, err
			}
			if err := s.updateRuleWorldCard(cardID, effect.kind, func(card *Card) error {
				for _, tag := range value.Tags {
					card.Tags = removeString(card.Tags, tag)
				}
				return nil
			}); err != nil {
				return nil, err
			}
			events.emit(EventCardTagsRemoved, CardTagsRemovedPayload{CardID: cardID, Tags: append([]string(nil), value.Tags...)})
		case EffectSetDocumentVariant:
			value := effect.setDocumentVariant
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			cardID, err := s.ruleEffectTargetCardID(value.CardID, signal)
			if err != nil {
				return nil, err
			}
			if err := s.updateRuleWorldCard(cardID, effect.kind, func(card *Card) error {
				document, ok := s.documentVariants[card.ID][value.Variant]
				if !ok {
					return fmt.Errorf("card %q document variant %q does not exist", card.ID, value.Variant)
				}
				card.Document = cloneValue(document)
				return nil
			}); err != nil {
				return nil, err
			}
			events.emit(EventCardVariantChanged, CardVariantChangedPayload{CardID: cardID, Variant: value.Variant})
		case EffectSetMessage:
			if effect.setMessage == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			s.setMessageLocked(effect.setMessage.Message, events)
		case EffectLoadDeck:
			if effect.loadDeck == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			loaded, err := s.loadDeck(effect.loadDeck.DeckID)
			if err != nil {
				return nil, err
			}
			if loaded {
				events.emit(EventDeckLoaded, DeckLoadedPayload{DeckID: effect.loadDeck.DeckID})
			}
		case EffectCopyComponent:
			if effect.copyComponent == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			mounted, updated, err := s.copyRuleComponent(*effect.copyComponent, signal)
			if err != nil {
				return nil, err
			}
			events.emit(EventComponentMounted, mounted)
			followUps = append(followUps, updated)
		case EffectAttackCreature:
			if effect.attackCreature == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			attacked, updated, err := s.attackCreature(signal)
			if err != nil {
				return nil, err
			}
			events.emit(EventCreatureAttacked, attacked)
			followUps = append(followUps, updated)
		default:
			return nil, fmt.Errorf("unsupported effect kind %q", effect.kind)
		}
	}
	return followUps, nil
}

func (s *Session) copyRuleComponent(
	effect CopyComponentEffectDefinition,
	signal ruleSignal,
) (ComponentMountedPayload, ComponentUpdatedSignal, error) {
	var sourceCardID string
	var sourceNode *cardcomponent.Node
	switch effect.Source {
	case RuleCardSource:
		played, ok := signal.(CardPlayedSignal)
		if !ok {
			return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("%s source requires %s", effect.Kind, TriggerCardPlayed)
		}
		sourceIndex := s.libraryCardIndex(played.SourceCardID)
		if sourceIndex < 0 {
			return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("source card %q is not in the library", played.SourceCardID)
		}
		sourceCardID = played.SourceCardID
		sourceNode = findRuleComponent(s.library[sourceIndex].Document, effect.ComponentKind, effect.SourceComponentID)
	case RuleComponentTrigger:
		updated, ok := signal.(ComponentUpdatedSignal)
		if !ok {
			return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("%s source requires %s", effect.Kind, TriggerComponentUpdated)
		}
		sourceCardID = updated.CardID
		candidate := cloneValue(updated.Component)
		if candidate.ComponentKind == effect.ComponentKind &&
			(effect.SourceComponentID == "" || candidate.ID == effect.SourceComponentID) {
			sourceNode = &candidate
		}
	default:
		return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("%s effect has unsupported source %q", effect.Kind, effect.Source)
	}
	if sourceNode == nil {
		return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf(
			"%s source has no %s component", effect.Kind, effect.ComponentKind,
		)
	}
	targetCardID, err := s.ruleEffectTargetCardID(effect.CardID, signal)
	if err != nil {
		return ComponentMountedPayload{}, ComponentUpdatedSignal{}, err
	}
	template := cardcomponent.ComponentTemplate{
		ComponentKind: sourceNode.ComponentKind,
		ComponentID:   sourceNode.ID,
		Config:        append(json.RawMessage(nil), sourceNode.Config...),
	}
	if effect.ComponentID != "" {
		template.ComponentID = effect.ComponentID
	}
	var installed cardcomponent.Node
	if err := s.updateRuleWorldCard(targetCardID, effect.Kind, func(card *Card) error {
		document, node, err := s.registry.InstallTemplate(card.Document, template)
		if err != nil {
			return err
		}
		card.Document = document
		installed = cloneValue(node)
		return nil
	}); err != nil {
		return ComponentMountedPayload{}, ComponentUpdatedSignal{}, err
	}
	mounted := ComponentMountedPayload{
		SourceCardID: sourceCardID, TargetCardID: targetCardID,
		ComponentID: installed.ID, ComponentKind: installed.ComponentKind,
	}
	followUp := ComponentUpdatedSignal{
		CardID: targetCardID, ComponentID: installed.ID,
		ComponentKind: installed.ComponentKind, Component: installed,
	}
	return mounted, followUp, nil
}

func (s *Session) ruleEffectTargetCardID(explicitCardID string, signal ruleSignal) (string, error) {
	if cardID := strings.TrimSpace(explicitCardID); cardID != "" {
		return cardID, nil
	}
	switch typed := signal.(type) {
	case CardPlayedSignal:
		return typed.TargetCardID, nil
	case FormSubmittedSignal:
		return typed.CardID, nil
	case ComponentUpdatedSignal:
		return typed.CardID, nil
	default:
		return "", fmt.Errorf("rule signal %T has no target card", signal)
	}
}

func (s *Session) updateRuleWorldCard(cardID string, effectKind RuleEffectKind, update func(*Card) error) error {
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return fmt.Errorf("%s effect references card %q outside world deck", effectKind, cardID)
	}
	card := s.worldDeck[index]
	if err := update(&card); err != nil {
		return err
	}
	s.worldDeck[index] = card
	return nil
}
