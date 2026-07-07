package game

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/design"
)

func stateBool(state map[string]any, key string) bool {
	value, ok := state[key].(bool)
	return ok && value
}

func stateString(state map[string]any, key string) string {
	value, ok := state[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func appendStringOnce(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, candidate := range values {
		if candidate == value {
			return values
		}
	}
	return append(values, value)
}

func appendStateStringOnce(value any, next string) []string {
	next = strings.TrimSpace(next)
	var values []string
	switch typed := value.(type) {
	case []string:
		values = append(values, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = appendStringOnce(values, text)
			}
		}
	}
	return appendStringOnce(values, next)
}

func removeLibraryCardAt(cards []Card, index int) []Card {
	if index < 0 || index >= len(cards) {
		return cards
	}
	out := make([]Card, 0, len(cards)-1)
	out = append(out, cards[:index]...)
	out = append(out, cards[index+1:]...)
	return out
}

func preferredSliderNodeID(cardID string) string {
	if cardID == BlankControllerCardID {
		return "regulator-output-slider"
	}
	return strings.TrimSpace(cardID) + "-slider"
}

func nextComponentNodeID(document cardcomponent.Document, preferred string) string {
	preferred = strings.TrimSpace(preferred)
	if preferred == "" {
		preferred = "component"
	}
	if findNodeByID(document.Root, preferred) == nil {
		return preferred
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d", preferred, index)
		if findNodeByID(document.Root, candidate) == nil {
			return candidate
		}
	}
}

func findNodeByKind(node cardcomponent.Node, componentKind string) *cardcomponent.Node {
	if node.ComponentKind == componentKind {
		return &node
	}
	for _, child := range node.Children {
		if match := findNodeByKind(child, componentKind); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByKindPtr(node *cardcomponent.Node, componentKind string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ComponentKind == componentKind {
		return node
	}
	for index := range node.Children {
		if match := findNodeByKindPtr(&node.Children[index], componentKind); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByID(node cardcomponent.Node, componentID string) *cardcomponent.Node {
	if node.ID == componentID {
		return &node
	}
	for _, child := range node.Children {
		if match := findNodeByID(child, componentID); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByIDPtr(node *cardcomponent.Node, componentID string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ID == componentID {
		return node
	}
	for index := range node.Children {
		if match := findNodeByIDPtr(&node.Children[index], componentID); match != nil {
			return match
		}
	}
	return nil
}

func decodeSliderNode(node cardcomponent.Node) (slider.Config, error) {
	var part slider.Config
	if len(node.Config) == 0 {
		return slider.Config{}, fmt.Errorf("slider component %q has no config", node.ID)
	}
	if err := json.Unmarshal(node.Config, &part); err != nil {
		return slider.Config{}, fmt.Errorf("decode slider config: %w", err)
	}
	part = slider.NormalizeConfig(part)
	if issues := slider.ValidateConfig(part); len(issues) > 0 {
		return slider.Config{}, fmt.Errorf("invalid slider config at %s: %s", issues[0].Path, issues[0].Message)
	}
	return part, nil
}

func decodeBorderNode(node cardcomponent.Node) (border.Config, error) {
	part := border.DefaultConfig()
	if len(node.Config) > 0 {
		if err := json.Unmarshal(node.Config, &part); err != nil {
			return border.Config{}, fmt.Errorf("decode border config: %w", err)
		}
	}
	part = normalizeBorderConfig(part)
	if issues := validateBorderConfig(part); len(issues) > 0 {
		return border.Config{}, fmt.Errorf("invalid border config at %s: %s", issues[0].Path, issues[0].Message)
	}
	return part, nil
}

func sliderConfigFromComponentCard(component Card) (slider.Config, error) {
	part := slider.DefaultConfig()
	if defaults, ok := component.State["componentDefaults"]; ok {
		raw, err := json.Marshal(defaults)
		if err != nil {
			return slider.Config{}, fmt.Errorf("encode slider component defaults: %w", err)
		}
		if err := json.Unmarshal(raw, &part); err != nil {
			return slider.Config{}, fmt.Errorf("decode slider component defaults: %w", err)
		}
	}
	part = slider.NormalizeConfig(part)
	if issues := slider.ValidateConfig(part); len(issues) > 0 {
		return slider.Config{}, fmt.Errorf("invalid slider component defaults at %s: %s", issues[0].Path, issues[0].Message)
	}
	return part, nil
}

func borderConfigFromComponentCard(component Card) (border.Config, error) {
	part := border.DefaultConfig()
	if defaults, ok := component.State["componentDefaults"]; ok {
		raw, err := json.Marshal(defaults)
		if err != nil {
			return border.Config{}, fmt.Errorf("encode border component defaults: %w", err)
		}
		if err := json.Unmarshal(raw, &part); err != nil {
			return border.Config{}, fmt.Errorf("decode border component defaults: %w", err)
		}
	}
	part = normalizeBorderConfig(part)
	if issues := validateBorderConfig(part); len(issues) > 0 {
		return border.Config{}, fmt.Errorf("invalid border component defaults at %s: %s", issues[0].Path, issues[0].Message)
	}
	return part, nil
}

func normalizeBorderConfig(part border.Config) border.Config {
	generated := design.GeneratedConfig[border.Config]{
		ComponentKind: border.Kind,
		Description:   "Border config",
		Config:        part,
	}
	border.NormalizeGenerated(&generated)
	return generated.Config
}

func validateBorderConfig(part border.Config) []design.Issue {
	return border.ValidateGenerated(design.GeneratedConfig[border.Config]{
		ComponentKind: border.Kind,
		Description:   "Border config",
		Config:        part,
	})
}

func readJSONInt(raw json.RawMessage) (int, error) {
	var intValue int
	if err := json.Unmarshal(raw, &intValue); err == nil {
		return intValue, nil
	}
	var floatValue float64
	if err := json.Unmarshal(raw, &floatValue); err != nil {
		return 0, fmt.Errorf("value must be a number")
	}
	return int(floatValue), nil
}

func readJSONString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("value must be a string")
	}
	return strings.TrimSpace(value), nil
}

func stringInSlice(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
