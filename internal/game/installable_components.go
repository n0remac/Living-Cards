package game

import (
	"encoding/json"
	"fmt"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

const componentTemplateStateKey = "component_template"

func componentTemplateFromCard(registry *cardcomponent.Registry, component CardInstance) (cardcomponent.ComponentTemplate, error) {
	if registry == nil {
		return cardcomponent.ComponentTemplate{}, fmt.Errorf("component registry is not initialized")
	}
	value, declared := component.State[componentTemplateStateKey]
	if !declared {
		return cardcomponent.ComponentTemplate{}, fmt.Errorf("card instance %q is not a component card", component.InstanceID)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return cardcomponent.ComponentTemplate{}, fmt.Errorf("encode card instance %q component template: %w", component.InstanceID, err)
	}
	template, issues := registry.DecodeTemplate(raw)
	if len(issues) > 0 {
		return cardcomponent.ComponentTemplate{}, fmt.Errorf("invalid card instance %q component template at %s: %s", component.InstanceID, issues[0].Path, issues[0].Message)
	}
	return template, nil
}

func validateComponentCardState(registry *cardcomponent.Registry, name string, state map[string]any) error {
	if _, declared := state[componentTemplateStateKey]; !declared {
		return nil
	}
	_, err := componentTemplateFromCard(registry, CardInstance{InstanceID: CardInstanceID(name), State: state})
	return err
}
