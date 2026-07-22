package game

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	"github.com/n0remac/Living-Card/internal/components/button"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/components/textinput"
	"github.com/n0remac/Living-Card/internal/design"
)

type componentMergePolicy string

const (
	componentMergeAppend  componentMergePolicy = "append"
	componentMergeReplace componentMergePolicy = "replace"
)

type installableComponentDefinition struct {
	kind             string
	mergePolicy      componentMergePolicy
	defaultID        func(targetCardID string) string
	normalizedConfig func(value any) (json.RawMessage, error)
}

var installableComponentDefinitions = map[string]installableComponentDefinition{
	background.Kind: {
		kind:             background.Kind,
		mergePolicy:      componentMergeReplace,
		defaultID:        cardComponentID(background.Kind),
		normalizedConfig: normalizeGeneratedConfig(background.Kind, background.DefaultConfig(), background.NormalizeGenerated, background.ValidateGenerated),
	},
	border.Kind: {
		kind:             border.Kind,
		mergePolicy:      componentMergeReplace,
		defaultID:        cardComponentID(border.Kind),
		normalizedConfig: normalizeGeneratedConfig(border.Kind, border.DefaultConfig(), border.NormalizeGenerated, border.ValidateGenerated),
	},
	textarea.Kind: {
		kind:             textarea.Kind,
		mergePolicy:      componentMergeAppend,
		defaultID:        cardComponentID(textarea.Kind),
		normalizedConfig: normalizeGeneratedConfig(textarea.Kind, textarea.DefaultConfig(), textarea.NormalizeGenerated, textarea.ValidateGenerated),
	},
	shape.Kind: {
		kind:             shape.Kind,
		mergePolicy:      componentMergeAppend,
		defaultID:        cardComponentID(shape.Kind),
		normalizedConfig: normalizeGeneratedConfig(shape.Kind, shape.DefaultConfig(), shape.NormalizeGenerated, shape.ValidateGenerated),
	},
	imagecomponent.Kind: {
		kind:             imagecomponent.Kind,
		mergePolicy:      componentMergeAppend,
		defaultID:        cardComponentID(imagecomponent.Kind),
		normalizedConfig: normalizeGeneratedConfig(imagecomponent.Kind, imagecomponent.DefaultConfig(), imagecomponent.NormalizeGenerated, imagecomponent.ValidateGenerated),
	},
	slider.Kind: {
		kind:        slider.Kind,
		mergePolicy: componentMergeAppend,
		defaultID:   preferredSliderNodeID,
		normalizedConfig: normalizeSimpleConfig(slider.DefaultConfig(), slider.NormalizeConfig, func(part slider.Config) []design.Issue {
			return slider.ValidateConfig(part)
		}),
	},
	textinput.Kind: {
		kind:        textinput.Kind,
		mergePolicy: componentMergeAppend,
		defaultID:   cardComponentID(textinput.Kind),
		normalizedConfig: normalizeSimpleConfig(textinput.DefaultConfig(), textinput.NormalizeConfig, func(part textinput.Config) []design.Issue {
			return textinput.ValidateConfig(part)
		}),
	},
	button.Kind: {
		kind:        button.Kind,
		mergePolicy: componentMergeAppend,
		defaultID:   cardComponentID(button.Kind),
		normalizedConfig: normalizeSimpleConfig(button.DefaultConfig(), button.NormalizeConfig, func(part button.Config) []design.Issue {
			return button.ValidateConfig(part)
		}),
	},
}

func installableComponentDefinitionForKind(kind string) (installableComponentDefinition, bool) {
	kind = strings.TrimSpace(kind)
	definition, ok := installableComponentDefinitions[kind]
	if !ok || definition.kind != kind || definition.defaultID == nil || definition.normalizedConfig == nil {
		return installableComponentDefinition{}, false
	}
	if definition.mergePolicy != componentMergeAppend && definition.mergePolicy != componentMergeReplace {
		return installableComponentDefinition{}, false
	}
	return definition, true
}

func componentNodeFromCard(component Card, target cardcomponent.Document) (cardcomponent.Node, componentMergePolicy, error) {
	kind := stateString(component.State, "componentKind")
	definition, ok := installableComponentDefinitionForKind(kind)
	if !ok {
		if kind == "" {
			return cardcomponent.Node{}, "", fmt.Errorf("%s is not a component card", component.Name)
		}
		return cardcomponent.Node{}, "", fmt.Errorf("component kind %q is not installable", kind)
	}
	config, err := definition.normalizedConfig(component.State["componentDefaults"])
	if err != nil {
		return cardcomponent.Node{}, "", fmt.Errorf("invalid %s defaults: %w", kind, err)
	}
	preferredID := stateString(component.State, "componentId")
	if preferredID == "" {
		preferredID = definition.defaultID(target.CardID)
	}
	return cardcomponent.Node{
		ID:            nextComponentNodeID(target, preferredID),
		ComponentKind: kind,
		Config:        config,
	}, definition.mergePolicy, nil
}

func installComponentNode(document *cardcomponent.Document, node cardcomponent.Node, mergePolicy componentMergePolicy) {
	if document == nil {
		return
	}
	if mergePolicy == componentMergeReplace {
		if existing := findNodeByKindPtr(&document.Root, node.ComponentKind); existing != nil {
			existing.Config = node.Config
			return
		}
	}
	document.Root.Children = append(document.Root.Children, node)
}

func validateComponentCardState(name string, state map[string]any) error {
	rawKind, declared := state["componentKind"]
	if !declared {
		return nil
	}
	kindValue, ok := rawKind.(string)
	if !ok {
		return fmt.Errorf("card %q componentKind must be a string", name)
	}
	kind := strings.TrimSpace(kindValue)
	if kind == "" {
		return fmt.Errorf("card %q componentKind is required when declared", name)
	}
	definition, ok := installableComponentDefinitionForKind(kind)
	if !ok {
		return fmt.Errorf("card %q has unsupported componentKind %q", name, kind)
	}
	if rawComponentID, declared := state["componentId"]; declared {
		componentID, ok := rawComponentID.(string)
		if !ok {
			return fmt.Errorf("card %q componentId must be a string", name)
		}
		componentID = strings.TrimSpace(componentID)
		if !safeComponentID(componentID) {
			return fmt.Errorf("card %q componentId %q may only contain letters, numbers, hyphens, and underscores", name, componentID)
		}
	}
	if _, err := definition.normalizedConfig(state["componentDefaults"]); err != nil {
		return fmt.Errorf("card %q componentDefaults: %w", name, err)
	}
	return nil
}

func normalizeGeneratedConfig[T any](kind string, defaults T, normalize func(*design.GeneratedConfig[T]), validate func(design.GeneratedConfig[T]) []design.Issue) func(any) (json.RawMessage, error) {
	return func(value any) (json.RawMessage, error) {
		part := cloneValue(defaults)
		if value != nil {
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(raw, &part); err != nil {
				return nil, err
			}
		}
		generated := design.GeneratedConfig[T]{ComponentKind: kind, Description: "Installed component", Config: part}
		normalize(&generated)
		if issues := validate(generated); len(issues) > 0 {
			return nil, fmt.Errorf("%s: %s", issues[0].Path, issues[0].Message)
		}
		return json.Marshal(generated.Config)
	}
}

func normalizeSimpleConfig[T any](defaults T, normalize func(T) T, validate func(T) []design.Issue) func(any) (json.RawMessage, error) {
	return func(value any) (json.RawMessage, error) {
		part := cloneValue(defaults)
		if value != nil {
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(raw, &part); err != nil {
				return nil, err
			}
		}
		part = normalize(part)
		if issues := validate(part); len(issues) > 0 {
			return nil, fmt.Errorf("%s: %s", issues[0].Path, issues[0].Message)
		}
		return json.Marshal(part)
	}
}

func cardComponentID(kind string) func(string) string {
	return func(cardID string) string {
		return strings.TrimSpace(cardID) + "-" + kind
	}
}

func safeComponentID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
		case char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9':
		case char == '-', char == '_':
		default:
			return false
		}
	}
	return true
}
