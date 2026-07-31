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
		resolution := RuleResolution{RuleID: selected.ID, TriggerKind: signal.triggerKind(), Outcome: RuleOutcomeSuccess}
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
	resolution := RuleResolution{RuleID: resolved.ID, TriggerKind: signal.triggerKind(), Outcome: RuleOutcomeConditionsFailed}
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
		source, sourceOK := s.instance(typed.SourceInstanceID)
		target, targetOK := s.instance(typed.TargetInstanceID)
		if !sourceOK || !targetOK {
			return false, nil
		}
		return cardMatches(source, trigger.cardPlayed.Source) &&
			cardMatches(target, trigger.cardPlayed.Target), nil
	case FormSubmittedSignal:
		if trigger.formSubmitted == nil || trigger.formSubmitted.FormID != typed.FormID {
			return false, nil
		}
		target, ok := s.instance(typed.InstanceID)
		return ok && cardMatches(target, trigger.formSubmitted.Target), nil
	case ComponentUpdatedSignal:
		if trigger.componentUpdated == nil ||
			trigger.componentUpdated.ComponentKind != typed.ComponentKind ||
			(trigger.componentUpdated.ComponentID != "" && trigger.componentUpdated.ComponentID != typed.ComponentID) {
			return false, nil
		}
		target, ok := s.instance(typed.InstanceID)
		return ok && cardMatches(target, trigger.componentUpdated.Target), nil
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

func (s *Session) ruleSignalCard(signal ruleSignal, reference string) (CardInstance, bool) {
	var instanceID CardInstanceID
	switch typed := signal.(type) {
	case CardPlayedSignal:
		switch reference {
		case RuleCardSource:
			return s.instance(typed.SourceInstanceID)
		case RuleCardTarget:
			instanceID = typed.TargetInstanceID
		}
	case FormSubmittedSignal:
		if reference == RuleCardTarget {
			instanceID = typed.InstanceID
		}
	case ComponentUpdatedSignal:
		if reference == RuleCardTarget {
			instanceID = typed.InstanceID
		}
	}
	if instanceID == "" {
		return CardInstance{}, false
	}
	return s.instance(instanceID)
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
			instanceID, err := s.resolveRuleInstanceReference(value.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			if err := s.updateInstance(instanceID, func(instance *CardInstance) error {
				if instance.State == nil {
					instance.State = map[string]any{}
				}
				instance.State[value.Key] = cloneValue(value.Value)
				return nil
			}); err != nil {
				return nil, err
			}
			events.emit(EventCardStateChanged, CardStateChangedPayload{CardID: string(instanceID), Key: value.Key, Value: cloneValue(value.Value)})
		case EffectRemoveCardTags:
			value := effect.removeCardTags
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			instanceID, err := s.resolveRuleInstanceReference(value.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			if err := s.updateInstance(instanceID, func(instance *CardInstance) error {
				for _, tag := range value.Tags {
					instance.Tags = removeString(instance.Tags, tag)
				}
				return nil
			}); err != nil {
				return nil, err
			}
			events.emit(EventCardTagsRemoved, CardTagsRemovedPayload{CardID: string(instanceID), Tags: append([]string(nil), value.Tags...)})
		case EffectSetDocumentVariant:
			value := effect.setDocumentVariant
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			instanceID, err := s.resolveRuleInstanceReference(value.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			if err := s.updateInstance(instanceID, func(instance *CardInstance) error {
				definition, ok := s.cardDefinitions[instance.DefinitionID]
				if !ok {
					return fmt.Errorf("card instance %q references missing definition %q", instanceID, instance.DefinitionID)
				}
				document, ok := definition.Documents[value.Variant]
				if !ok {
					return fmt.Errorf("card definition %q document variant %q does not exist", definition.ID, value.Variant)
				}
				instance.Document = cloneValue(document)
				instance.Document.CardID = string(instanceID)
				return nil
			}); err != nil {
				return nil, err
			}
			events.emit(EventCardVariantChanged, CardVariantChangedPayload{CardID: string(instanceID), Variant: value.Variant})
		case EffectSetMessage:
			if effect.setMessage == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			s.setMessageLocked(effect.setMessage.Message, events)
		case EffectLoadDeck:
			if effect.loadDeck == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			loaded, err := s.loadDeck(effect.loadDeck.DeckID, events)
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
		case EffectMoveCard:
			if effect.moveCard == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			instanceID, err := s.resolveRuleInstanceReference(effect.moveCard.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			from, unique := s.instanceZone(instanceID)
			if !unique {
				return nil, fmt.Errorf("card instance %q does not belong to exactly one zone", instanceID)
			}
			move, err := s.moveCard(instanceID, from, effect.moveCard.To)
			if err != nil {
				return nil, err
			}
			events.emit(EventCardMoved, CardMovedPayloadFromMove(move))
			if effect.moveCard.To == ZoneDiscard {
				events.emit(EventCardConsumed, CardConsumedPayload{CardID: string(instanceID)})
			}
		case EffectStartEncounter:
			value := effect.startEncounter
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			if err := s.startEncounter(EncounterState{
				ID: value.ID, Phase: EncounterPhaseActive, Participants: cloneValue(value.Participants),
				Pressure: value.Pressure, MaxPressure: value.MaxPressure, ReactionPressure: value.ReactionPressure,
			}, events); err != nil {
				return nil, err
			}
		case EffectChangePressure:
			if effect.changePressure == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			if err := s.changeEncounterPressure(effect.changePressure.Delta, events); err != nil {
				return nil, err
			}
		case EffectChangeActorTrack:
			value := effect.changeActorTrack
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			instanceID, err := s.resolveRuleInstanceReference(value.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			instance, _ := s.instance(instanceID)
			if instance.Actor == nil {
				return nil, fmt.Errorf("card instance %q has no actor state", instanceID)
			}
			current, ok := actorResourceCurrent(*instance.Actor, value.Track)
			if !ok {
				return nil, fmt.Errorf("actor %q has no track %q", instanceID, value.Track)
			}
			if err := s.changeActorResource(instanceID, value.Track, current+value.Delta, events); err != nil {
				return nil, err
			}
		case EffectSetActorDisposition:
			value := effect.setActorDisposition
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			instanceID, err := s.resolveRuleInstanceReference(value.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			if err := s.setActorDisposition(instanceID, value.Disposition, events); err != nil {
				return nil, err
			}
		case EffectAddActorStatus, EffectRemoveActorStatus:
			value := effect.actorStatus
			if value == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			instanceID, err := s.resolveRuleInstanceReference(value.RuleInstanceReference, signal)
			if err != nil {
				return nil, err
			}
			if err := s.changeActorStatus(instanceID, value.Status, effect.kind == EffectAddActorStatus, events); err != nil {
				return nil, err
			}
		case EffectResolveEncounter:
			if effect.resolveEncounter == nil {
				return nil, fmt.Errorf("%s effect payload is missing", effect.kind)
			}
			if err := s.resolveEncounter(effect.resolveEncounter.Outcome, events); err != nil {
				return nil, err
			}
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
		source, ok := s.instance(played.SourceInstanceID)
		if !ok {
			return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("source card instance %q does not exist", played.SourceInstanceID)
		}
		sourceCardID = string(played.SourceInstanceID)
		sourceNode = findRuleComponent(source.Document, effect.ComponentKind, effect.SourceComponentID)
	case RuleComponentTrigger:
		updated, ok := signal.(ComponentUpdatedSignal)
		if !ok {
			return ComponentMountedPayload{}, ComponentUpdatedSignal{}, fmt.Errorf("%s source requires %s", effect.Kind, TriggerComponentUpdated)
		}
		sourceCardID = string(updated.InstanceID)
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
	targetInstanceID, err := s.resolveRuleInstanceReference(effect.Target, signal)
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
	if err := s.updateInstance(targetInstanceID, func(instance *CardInstance) error {
		document, node, err := s.registry.InstallTemplate(instance.Document, template)
		if err != nil {
			return err
		}
		instance.Document = document
		installed = cloneValue(node)
		return nil
	}); err != nil {
		return ComponentMountedPayload{}, ComponentUpdatedSignal{}, err
	}
	mounted := ComponentMountedPayload{
		SourceCardID: sourceCardID, TargetCardID: string(targetInstanceID),
		ComponentID: installed.ID, ComponentKind: installed.ComponentKind,
	}
	followUp := ComponentUpdatedSignal{
		InstanceID: targetInstanceID, ComponentID: installed.ID,
		ComponentKind: installed.ComponentKind, Component: installed,
	}
	return mounted, followUp, nil
}

func (s *Session) resolveRuleInstanceReference(reference RuleInstanceReference, signal ruleSignal) (CardInstanceID, error) {
	if reference.InstanceID != "" {
		if _, ok := s.instance(reference.InstanceID); !ok {
			return "", fmt.Errorf("rule effect references unknown card instance %q", reference.InstanceID)
		}
		return reference.InstanceID, nil
	}
	card, ok := s.ruleSignalCard(signal, string(reference.Card))
	if !ok {
		return "", fmt.Errorf("rule signal %T has no %q card instance", signal, reference.Card)
	}
	return card.InstanceID, nil
}
