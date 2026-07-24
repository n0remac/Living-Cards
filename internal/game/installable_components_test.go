package game

import (
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
)

func TestComponentCardUsesCanonicalTemplate(t *testing.T) {
	registry := catalog.MustNew()
	component := Card{Name: "Input", State: map[string]any{
		componentTemplateStateKey: map[string]any{
			"component_kind": card.KindTextInput,
			"component_id":   "password-input",
			"config":         map[string]any{},
		},
	}}
	template, err := componentTemplateFromCard(registry, component)
	if err != nil {
		t.Fatal(err)
	}
	if template.ComponentKind != card.KindTextInput || template.ComponentID != "password-input" || len(template.Config) == 0 {
		t.Fatalf("template = %#v", template)
	}
}

func TestComponentCardTemplateIsStrictAndCapabilityAware(t *testing.T) {
	registry := catalog.MustNew()
	for name, state := range map[string]map[string]any{
		"legacy fields": {componentTemplateStateKey: map[string]any{"componentKind": card.KindSlider}},
		"unknown kind":  {componentTemplateStateKey: map[string]any{"component_kind": "future_widget"}},
		"root":          {componentTemplateStateKey: map[string]any{"component_kind": card.Kind}},
		"null config":   {componentTemplateStateKey: map[string]any{"component_kind": card.KindSlider, "config": nil}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateComponentCardState(registry, name, state); err == nil {
				t.Fatal("validateComponentCardState() error = nil")
			}
		})
	}
	if err := validateComponentCardState(registry, "ordinary", map[string]any{"editable": true}); err != nil {
		t.Fatalf("ordinary card state: %v", err)
	}
}
