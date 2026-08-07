package game

import (
	"bytes"
	"encoding/json"
	"fmt"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

type RuleTriggerKind string

const (
	TriggerCardPlayed       RuleTriggerKind = "cardPlayed"
	TriggerFormSubmitted    RuleTriggerKind = "formSubmitted"
	TriggerComponentUpdated RuleTriggerKind = "componentUpdated"
)

type RuleConditionKind string

const (
	ConditionFlagEquals              RuleConditionKind = "flagEquals"
	ConditionComponentPresent        RuleConditionKind = "componentPresent"
	ConditionComponentPropertyEquals RuleConditionKind = "componentPropertyEquals"
	ConditionFormFieldEquals         RuleConditionKind = "formFieldEquals"
)

type RuleEffectKind string

const (
	EffectSetFlag            RuleEffectKind = "setFlag"
	EffectSetCardState       RuleEffectKind = "setCardState"
	EffectRemoveCardTags     RuleEffectKind = "removeCardTags"
	EffectSetDocumentVariant RuleEffectKind = "setDocumentVariant"
	EffectSetMessage         RuleEffectKind = "setMessage"
	EffectLoadDeck           RuleEffectKind = "loadDeck"
	EffectCopyComponent      RuleEffectKind = "copyComponent"
)

const (
	RuleCardSource       = "source"
	RuleCardTarget       = "target"
	RuleComponentTrigger = "trigger"
)

type RuleDefinition struct {
	ID          string          `json:"id"`
	Trigger     RuleTrigger     `json:"trigger"`
	Conditions  []RuleCondition `json:"conditions,omitempty"`
	Effects     []RuleEffect    `json:"effects"`
	ElseEffects []RuleEffect    `json:"elseEffects,omitempty"`
}

type CardMatcherDefinition struct {
	ID   string   `json:"id,omitempty"`
	Tags []string `json:"tags,omitempty"`
}

type CardPlayedTriggerDefinition struct {
	Kind   RuleTriggerKind       `json:"kind"`
	Source CardMatcherDefinition `json:"source"`
	Target CardMatcherDefinition `json:"target"`
}

type FormSubmittedTriggerDefinition struct {
	Kind   RuleTriggerKind       `json:"kind"`
	Target CardMatcherDefinition `json:"target"`
	FormID string                `json:"formId"`
}

type ComponentUpdatedTriggerDefinition struct {
	Kind          RuleTriggerKind       `json:"kind"`
	Target        CardMatcherDefinition `json:"target"`
	ComponentKind string                `json:"componentKind"`
	ComponentID   string                `json:"componentId,omitempty"`
}

type RuleTrigger struct {
	kind             RuleTriggerKind
	cardPlayed       *CardPlayedTriggerDefinition
	formSubmitted    *FormSubmittedTriggerDefinition
	componentUpdated *ComponentUpdatedTriggerDefinition
}

func (t RuleTrigger) Kind() RuleTriggerKind { return t.kind }

func CardPlayedTrigger(source, target CardMatcherDefinition) RuleTrigger {
	value := CardPlayedTriggerDefinition{Kind: TriggerCardPlayed, Source: source, Target: target}
	return RuleTrigger{kind: value.Kind, cardPlayed: &value}
}

func FormSubmittedTrigger(target CardMatcherDefinition, formID string) RuleTrigger {
	value := FormSubmittedTriggerDefinition{Kind: TriggerFormSubmitted, Target: target, FormID: formID}
	return RuleTrigger{kind: value.Kind, formSubmitted: &value}
}

func ComponentUpdatedTrigger(target CardMatcherDefinition, componentKind, componentID string) RuleTrigger {
	value := ComponentUpdatedTriggerDefinition{Kind: TriggerComponentUpdated, Target: target, ComponentKind: componentKind, ComponentID: componentID}
	return RuleTrigger{kind: value.Kind, componentUpdated: &value}
}

func (t RuleTrigger) MarshalJSON() ([]byte, error) {
	switch t.kind {
	case TriggerCardPlayed:
		return json.Marshal(t.cardPlayed)
	case TriggerFormSubmitted:
		return json.Marshal(t.formSubmitted)
	case TriggerComponentUpdated:
		return json.Marshal(t.componentUpdated)
	default:
		return nil, fmt.Errorf("unsupported trigger kind %q", t.kind)
	}
}

func (t *RuleTrigger) UnmarshalJSON(raw []byte) error {
	var tag struct {
		Kind RuleTriggerKind `json:"kind"`
	}
	if err := json.Unmarshal(raw, &tag); err != nil {
		return err
	}
	switch tag.Kind {
	case TriggerCardPlayed:
		var value CardPlayedTriggerDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*t = CardPlayedTrigger(value.Source, value.Target)
	case TriggerFormSubmitted:
		var value FormSubmittedTriggerDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*t = FormSubmittedTrigger(value.Target, value.FormID)
	case TriggerComponentUpdated:
		var value ComponentUpdatedTriggerDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*t = ComponentUpdatedTrigger(value.Target, value.ComponentKind, value.ComponentID)
	default:
		return fmt.Errorf("unsupported trigger kind %q", tag.Kind)
	}
	return nil
}

type RuleValue struct {
	Kind   schema.PropertyKind
	String string
	Number float64
	Bool   bool
}

func StringRuleValue(value string) RuleValue {
	return RuleValue{Kind: schema.PropertyString, String: value}
}

func NumberRuleValue(value float64) RuleValue {
	return RuleValue{Kind: schema.PropertyNumber, Number: value}
}

func BoolRuleValue(value bool) RuleValue {
	return RuleValue{Kind: schema.PropertyBool, Bool: value}
}

func (v RuleValue) MarshalJSON() ([]byte, error) {
	switch v.Kind {
	case schema.PropertyString:
		return json.Marshal(v.String)
	case schema.PropertyNumber:
		return json.Marshal(v.Number)
	case schema.PropertyBool:
		return json.Marshal(v.Bool)
	default:
		return nil, fmt.Errorf("unsupported rule value kind %q", v.Kind)
	}
}

func (v *RuleValue) UnmarshalJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	switch typed := value.(type) {
	case string:
		*v = StringRuleValue(typed)
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return fmt.Errorf("rule number is invalid: %w", err)
		}
		*v = NumberRuleValue(number)
	case bool:
		*v = BoolRuleValue(typed)
	default:
		return fmt.Errorf("rule value must be a string, number, or boolean")
	}
	return nil
}

type FlagEqualsConditionDefinition struct {
	Kind  RuleConditionKind `json:"kind"`
	Flag  string            `json:"flag"`
	Value bool              `json:"value"`
}

type ComponentPresentConditionDefinition struct {
	Kind          RuleConditionKind `json:"kind"`
	Card          string            `json:"card"`
	ComponentKind string            `json:"componentKind"`
	ComponentID   string            `json:"componentId,omitempty"`
}

type ComponentPropertyEqualsConditionDefinition struct {
	Kind          RuleConditionKind `json:"kind"`
	Card          string            `json:"card,omitempty"`
	Component     string            `json:"component,omitempty"`
	ComponentKind string            `json:"componentKind"`
	ComponentID   string            `json:"componentId,omitempty"`
	Property      string            `json:"property"`
	Value         RuleValue         `json:"value"`
}

type FormFieldEqualsConditionDefinition struct {
	Kind          RuleConditionKind `json:"kind"`
	Name          string            `json:"name"`
	Value         string            `json:"value"`
	TrimSpace     bool              `json:"trimSpace,omitempty"`
	CaseSensitive bool              `json:"caseSensitive,omitempty"`
}

type RuleCondition struct {
	kind                    RuleConditionKind
	flagEquals              *FlagEqualsConditionDefinition
	componentPresent        *ComponentPresentConditionDefinition
	componentPropertyEquals *ComponentPropertyEqualsConditionDefinition
	formFieldEquals         *FormFieldEqualsConditionDefinition
}

func (c RuleCondition) Kind() RuleConditionKind { return c.kind }

func FlagEqualsCondition(flag string, value bool) RuleCondition {
	definition := FlagEqualsConditionDefinition{Kind: ConditionFlagEquals, Flag: flag, Value: value}
	return RuleCondition{kind: definition.Kind, flagEquals: &definition}
}

func ComponentPresentCondition(cardReference, componentKind, componentID string) RuleCondition {
	definition := ComponentPresentConditionDefinition{Kind: ConditionComponentPresent, Card: cardReference, ComponentKind: componentKind, ComponentID: componentID}
	return RuleCondition{kind: definition.Kind, componentPresent: &definition}
}

func ComponentPropertyEqualsCondition(cardReference, componentReference, componentKind, componentID, property string, value RuleValue) RuleCondition {
	definition := ComponentPropertyEqualsConditionDefinition{
		Kind: ConditionComponentPropertyEquals, Card: cardReference, Component: componentReference,
		ComponentKind: componentKind, ComponentID: componentID, Property: property, Value: value,
	}
	return RuleCondition{kind: definition.Kind, componentPropertyEquals: &definition}
}

func FormFieldEqualsCondition(name, value string, trimSpace, caseSensitive bool) RuleCondition {
	definition := FormFieldEqualsConditionDefinition{Kind: ConditionFormFieldEquals, Name: name, Value: value, TrimSpace: trimSpace, CaseSensitive: caseSensitive}
	return RuleCondition{kind: definition.Kind, formFieldEquals: &definition}
}

func (c RuleCondition) MarshalJSON() ([]byte, error) {
	switch c.kind {
	case ConditionFlagEquals:
		return json.Marshal(c.flagEquals)
	case ConditionComponentPresent:
		return json.Marshal(c.componentPresent)
	case ConditionComponentPropertyEquals:
		return json.Marshal(c.componentPropertyEquals)
	case ConditionFormFieldEquals:
		return json.Marshal(c.formFieldEquals)
	default:
		return nil, fmt.Errorf("unsupported condition kind %q", c.kind)
	}
}

func (c *RuleCondition) UnmarshalJSON(raw []byte) error {
	var tag struct {
		Kind RuleConditionKind `json:"kind"`
	}
	if err := json.Unmarshal(raw, &tag); err != nil {
		return err
	}
	switch tag.Kind {
	case ConditionFlagEquals:
		var value FlagEqualsConditionDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		if err := requireJSONField(raw, "value"); err != nil {
			return err
		}
		*c = FlagEqualsCondition(value.Flag, value.Value)
	case ConditionComponentPresent:
		var value ComponentPresentConditionDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*c = ComponentPresentCondition(value.Card, value.ComponentKind, value.ComponentID)
	case ConditionComponentPropertyEquals:
		var value ComponentPropertyEqualsConditionDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		if err := requireJSONField(raw, "value"); err != nil {
			return err
		}
		*c = ComponentPropertyEqualsCondition(value.Card, value.Component, value.ComponentKind, value.ComponentID, value.Property, value.Value)
	case ConditionFormFieldEquals:
		var value FormFieldEqualsConditionDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		if err := requireJSONField(raw, "value"); err != nil {
			return err
		}
		*c = FormFieldEqualsCondition(value.Name, value.Value, value.TrimSpace, value.CaseSensitive)
	default:
		return fmt.Errorf("unsupported condition kind %q", tag.Kind)
	}
	return nil
}

type SetFlagEffectDefinition struct {
	Kind  RuleEffectKind `json:"kind"`
	Flag  string         `json:"flag"`
	Value bool           `json:"value"`
}

type SetCardStateEffectDefinition struct {
	Kind   RuleEffectKind `json:"kind"`
	CardID string         `json:"cardId,omitempty"`
	Key    string         `json:"key"`
	Value  any            `json:"value"`
}

type RemoveCardTagsEffectDefinition struct {
	Kind   RuleEffectKind `json:"kind"`
	CardID string         `json:"cardId,omitempty"`
	Tags   []string       `json:"tags"`
}

type SetDocumentVariantEffectDefinition struct {
	Kind    RuleEffectKind `json:"kind"`
	CardID  string         `json:"cardId,omitempty"`
	Variant string         `json:"variant"`
}

type SetMessageEffectDefinition struct {
	Kind    RuleEffectKind `json:"kind"`
	Message string         `json:"message"`
}

type LoadDeckEffectDefinition struct {
	Kind   RuleEffectKind `json:"kind"`
	DeckID string         `json:"deckId"`
}

type CopyComponentEffectDefinition struct {
	Kind              RuleEffectKind `json:"kind"`
	Source            string         `json:"source"`
	CardID            string         `json:"cardId,omitempty"`
	ComponentKind     string         `json:"componentKind"`
	SourceComponentID string         `json:"sourceComponentId,omitempty"`
	ComponentID       string         `json:"componentId,omitempty"`
}

type RuleEffect struct {
	kind               RuleEffectKind
	setFlag            *SetFlagEffectDefinition
	setCardState       *SetCardStateEffectDefinition
	removeCardTags     *RemoveCardTagsEffectDefinition
	setDocumentVariant *SetDocumentVariantEffectDefinition
	setMessage         *SetMessageEffectDefinition
	loadDeck           *LoadDeckEffectDefinition
	copyComponent      *CopyComponentEffectDefinition
}

func (e RuleEffect) Kind() RuleEffectKind { return e.kind }

func SetFlagEffect(flag string, value bool) RuleEffect {
	definition := SetFlagEffectDefinition{Kind: EffectSetFlag, Flag: flag, Value: value}
	return RuleEffect{kind: definition.Kind, setFlag: &definition}
}

func SetCardStateEffect(cardID, key string, value any) RuleEffect {
	definition := SetCardStateEffectDefinition{Kind: EffectSetCardState, CardID: cardID, Key: key, Value: value}
	return RuleEffect{kind: definition.Kind, setCardState: &definition}
}

func RemoveCardTagsEffect(cardID string, tags []string) RuleEffect {
	definition := RemoveCardTagsEffectDefinition{Kind: EffectRemoveCardTags, CardID: cardID, Tags: append([]string(nil), tags...)}
	return RuleEffect{kind: definition.Kind, removeCardTags: &definition}
}

func SetDocumentVariantEffect(cardID, variant string) RuleEffect {
	definition := SetDocumentVariantEffectDefinition{Kind: EffectSetDocumentVariant, CardID: cardID, Variant: variant}
	return RuleEffect{kind: definition.Kind, setDocumentVariant: &definition}
}

func SetMessageEffect(message string) RuleEffect {
	definition := SetMessageEffectDefinition{Kind: EffectSetMessage, Message: message}
	return RuleEffect{kind: definition.Kind, setMessage: &definition}
}

func LoadDeckEffect(deckID string) RuleEffect {
	definition := LoadDeckEffectDefinition{Kind: EffectLoadDeck, DeckID: deckID}
	return RuleEffect{kind: definition.Kind, loadDeck: &definition}
}

func CopyComponentEffect(source, cardID, componentKind, sourceComponentID, componentID string) RuleEffect {
	definition := CopyComponentEffectDefinition{
		Kind: EffectCopyComponent, Source: source, CardID: cardID, ComponentKind: componentKind,
		SourceComponentID: sourceComponentID, ComponentID: componentID,
	}
	return RuleEffect{kind: definition.Kind, copyComponent: &definition}
}

func (e RuleEffect) MarshalJSON() ([]byte, error) {
	switch e.kind {
	case EffectSetFlag:
		return json.Marshal(e.setFlag)
	case EffectSetCardState:
		return json.Marshal(e.setCardState)
	case EffectRemoveCardTags:
		return json.Marshal(e.removeCardTags)
	case EffectSetDocumentVariant:
		return json.Marshal(e.setDocumentVariant)
	case EffectSetMessage:
		return json.Marshal(e.setMessage)
	case EffectLoadDeck:
		return json.Marshal(e.loadDeck)
	case EffectCopyComponent:
		return json.Marshal(e.copyComponent)
	default:
		return nil, fmt.Errorf("unsupported effect kind %q", e.kind)
	}
}

func (e *RuleEffect) UnmarshalJSON(raw []byte) error {
	var tag struct {
		Kind RuleEffectKind `json:"kind"`
	}
	if err := json.Unmarshal(raw, &tag); err != nil {
		return err
	}
	switch tag.Kind {
	case EffectSetFlag:
		var value SetFlagEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		if err := requireJSONField(raw, "value"); err != nil {
			return err
		}
		*e = SetFlagEffect(value.Flag, value.Value)
	case EffectSetCardState:
		var value SetCardStateEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		if err := requireJSONField(raw, "value"); err != nil {
			return err
		}
		*e = SetCardStateEffect(value.CardID, value.Key, value.Value)
	case EffectRemoveCardTags:
		var value RemoveCardTagsEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*e = RemoveCardTagsEffect(value.CardID, value.Tags)
	case EffectSetDocumentVariant:
		var value SetDocumentVariantEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*e = SetDocumentVariantEffect(value.CardID, value.Variant)
	case EffectSetMessage:
		var value SetMessageEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*e = SetMessageEffect(value.Message)
	case EffectLoadDeck:
		var value LoadDeckEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*e = LoadDeckEffect(value.DeckID)
	case EffectCopyComponent:
		var value CopyComponentEffectDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return err
		}
		*e = CopyComponentEffect(value.Source, value.CardID, value.ComponentKind, value.SourceComponentID, value.ComponentID)
	default:
		return fmt.Errorf("unsupported effect kind %q", tag.Kind)
	}
	return nil
}

func requireJSONField(raw []byte, name string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if _, ok := fields[name]; !ok {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

type CardPlayedSignal struct {
	SourceCardID string
	TargetCardID string
}

type FormSubmittedSignal struct {
	CardID string
	FormID string
	Fields map[string]string
}

type ComponentUpdatedSignal struct {
	CardID        string
	ComponentID   string
	ComponentKind string
	Component     cardcomponent.Node
}

type ruleSignal interface {
	triggerKind() RuleTriggerKind
}

func (CardPlayedSignal) triggerKind() RuleTriggerKind       { return TriggerCardPlayed }
func (FormSubmittedSignal) triggerKind() RuleTriggerKind    { return TriggerFormSubmitted }
func (ComponentUpdatedSignal) triggerKind() RuleTriggerKind { return TriggerComponentUpdated }

type RuleOutcome string

const (
	RuleOutcomeSuccess          RuleOutcome = "success"
	RuleOutcomeConditionsFailed RuleOutcome = "conditionsFailed"
	RuleOutcomeNoMatch          RuleOutcome = "noMatch"
)

type RuleResolution struct {
	RuleID      string
	TriggerKind RuleTriggerKind
	Outcome     RuleOutcome
}
