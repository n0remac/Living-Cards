package game

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/design"
)

type componentPositionValue struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func worldComponentKindEditable(componentKind string) bool {
	switch componentKind {
	case background.Kind, border.Kind, textarea.Kind, shape.Kind, imagecomponent.Kind, slider.Kind:
		return true
	default:
		return false
	}
}

func componentEditLabel(componentKind string) string {
	switch componentKind {
	case background.Kind:
		return "Background"
	case border.Kind:
		return "Border"
	case textarea.Kind:
		return "Text"
	case shape.Kind:
		return "Shape"
	case imagecomponent.Kind:
		return "Image"
	case slider.Kind:
		return "Slider"
	default:
		return "Component"
	}
}

func applyGameComponentControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	if node == nil {
		return fmt.Errorf("component is required")
	}
	switch node.ComponentKind {
	case background.Kind:
		return applyBackgroundControl(node, control, value)
	case border.Kind:
		return applyBorderControl(node, control, value)
	case textarea.Kind:
		return applyTextareaControl(node, control, value)
	case shape.Kind:
		return applyShapeControl(node, control, value)
	case imagecomponent.Kind:
		return applyImageControl(node, control, value)
	case slider.Kind:
		return applySliderControl(node, control, value)
	default:
		return fmt.Errorf("component kind %q does not support active card controls", node.ComponentKind)
	}
}

func applyBackgroundControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	part, err := decodeNodeConfig(*node, background.DefaultConfig(), "background")
	if err != nil {
		return err
	}
	switch control {
	case "background_color", "backgroundColor":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BackgroundColor = next
	default:
		return fmt.Errorf("control %q is not supported for background", control)
	}
	part = normalizeBackgroundConfig(part)
	part.CSS = cssWithDeclarations(part.CSS, background.AllowedCSS(), map[string]string{
		"background":       part.BackgroundColor,
		"background-color": part.BackgroundColor,
	})
	part = normalizeBackgroundConfig(part)
	if issues := validateBackgroundConfig(part); len(issues) > 0 {
		return fmt.Errorf("invalid background config at %s: %s", issues[0].Path, issues[0].Message)
	}
	node.Config = mustRaw(part)
	return nil
}

func applyBorderControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	part, err := decodeBorderNode(*node)
	if err != nil {
		return err
	}
	switch control {
	case "border_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BorderColor = next
	case "border_width_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderWidthPX = next
	case "border_radius_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderRadiusPX = next
	case "border_style":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BorderStyle = next
	default:
		return fmt.Errorf("control %q is not supported for border", control)
	}
	part = normalizeBorderConfig(part)
	part.CSS = cssWithDeclarations(part.CSS, border.AllowedCSS(), map[string]string{
		"border":        fmt.Sprintf("%dpx %s %s", part.BorderWidthPX, part.BorderStyle, part.BorderColor),
		"border-color":  part.BorderColor,
		"border-radius": fmt.Sprintf("%dpx", part.BorderRadiusPX),
		"border-style":  part.BorderStyle,
		"border-width":  fmt.Sprintf("%dpx", part.BorderWidthPX),
	})
	part = normalizeBorderConfig(part)
	if issues := validateBorderConfig(part); len(issues) > 0 {
		return fmt.Errorf("invalid border config at %s: %s", issues[0].Path, issues[0].Message)
	}
	node.Config = mustRaw(part)
	return nil
}

func applyTextareaControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	part, err := decodeNodeConfig(*node, textarea.DefaultConfig(), "textarea")
	if err != nil {
		return err
	}
	switch control {
	case "content":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Content = next
	case "font_family":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.FontFamily = next
	case "font_size_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.FontSizePX = next
	case "font_weight":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.FontWeight = next
	case "font_style":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.FontStyle = next
	case "color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Color = next
	case "align":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Align = next
	case "position":
		next, err := readJSONPosition(value)
		if err != nil {
			return err
		}
		part.Position = "center"
		part.X = next.X
		part.Y = next.Y
	case "x":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.X = next
	case "y":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Y = next
	case "background_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BackgroundColor = next
	case "border_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BorderColor = next
	case "border_width_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderWidthPX = next
	case "border_radius_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderRadiusPX = next
	case "padding_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.PaddingPX = next
	default:
		return fmt.Errorf("control %q is not supported for text", control)
	}
	part = normalizeTextareaConfig(part)
	part.CSS = updateTextareaCSS(part, control)
	part = normalizeTextareaConfig(part)
	if issues := validateTextareaConfig(part); len(issues) > 0 {
		return fmt.Errorf("invalid text config at %s: %s", issues[0].Path, issues[0].Message)
	}
	node.Config = mustRaw(part)
	return nil
}

func applyShapeControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	part, err := decodeNodeConfig(*node, shape.DefaultConfig(), "shape")
	if err != nil {
		return err
	}
	switch control {
	case "shape":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Shape = next
	case "position":
		next, err := readJSONPosition(value)
		if err != nil {
			return err
		}
		part.X = next.X
		part.Y = next.Y
	case "x":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.X = next
	case "y":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Y = next
	case "width":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Width = next
	case "height":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Height = next
	case "rotation":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Rotation = next
	case "background_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BackgroundColor = next
	case "border_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BorderColor = next
	case "border_width_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderWidthPX = next
	case "shadow":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Shadow = next
	default:
		return fmt.Errorf("control %q is not supported for shape", control)
	}
	part = normalizeShapeConfig(part)
	if issues := validateShapeConfig(part); len(issues) > 0 {
		return fmt.Errorf("invalid shape config at %s: %s", issues[0].Path, issues[0].Message)
	}
	node.Config = mustRaw(part)
	return nil
}

func applyImageControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	part, err := decodeNodeConfig(*node, imagecomponent.DefaultConfig(), "image")
	if err != nil {
		return err
	}
	switch control {
	case "alt":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Alt = next
	case "position":
		next, err := readJSONPosition(value)
		if err != nil {
			return err
		}
		part.X = next.X
		part.Y = next.Y
	case "x":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.X = next
	case "y":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Y = next
	case "width":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Width = next
	case "height":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Height = next
	case "rotation":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Rotation = next
	case "border_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.BorderColor = next
	case "border_width_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderWidthPX = next
	case "border_radius_px":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.BorderRadiusPX = next
	default:
		return fmt.Errorf("control %q is not supported for image", control)
	}
	part = normalizeImageConfig(part)
	if issues := validateImageConfig(part); len(issues) > 0 {
		return fmt.Errorf("invalid image config at %s: %s", issues[0].Path, issues[0].Message)
	}
	node.Config = mustRaw(part)
	return nil
}

func applySliderControl(node *cardcomponent.Node, control string, value json.RawMessage) error {
	part, err := decodeSliderNode(*node)
	if err != nil {
		return err
	}
	switch control {
	case "label":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.Label = next
	case "value":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Value = next
	case "min":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Min = next
	case "max":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Max = next
	case "step":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Step = next
	case "position":
		next, err := readJSONPosition(value)
		if err != nil {
			return err
		}
		part.X = next.X
		part.Y = next.Y
	case "x":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.X = next
	case "y":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Y = next
	case "width":
		next, err := readJSONInt(value)
		if err != nil {
			return err
		}
		part.Width = next
	case "track_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.TrackColor = next
	case "accent_color":
		next, err := readJSONString(value)
		if err != nil {
			return err
		}
		part.AccentColor = next
	default:
		return fmt.Errorf("control %q is not supported for slider", control)
	}
	part = slider.NormalizeConfig(part)
	if issues := slider.ValidateConfig(part); len(issues) > 0 {
		return fmt.Errorf("invalid slider config at %s: %s", issues[0].Path, issues[0].Message)
	}
	node.Config = mustRaw(part)
	return nil
}

func decodeNodeConfig[T any](node cardcomponent.Node, defaults T, label string) (T, error) {
	part := defaults
	if len(node.Config) == 0 {
		return part, nil
	}
	if err := json.Unmarshal(node.Config, &part); err != nil {
		return part, fmt.Errorf("decode %s config: %w", label, err)
	}
	return part, nil
}

func readJSONPosition(raw json.RawMessage) (componentPositionValue, error) {
	var value componentPositionValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return componentPositionValue{}, fmt.Errorf("value must include x and y")
	}
	value.X = clampPercent(value.X)
	value.Y = clampPercent(value.Y)
	return value, nil
}

func normalizeBackgroundConfig(part background.Config) background.Config {
	generated := design.GeneratedConfig[background.Config]{
		ComponentKind: background.Kind,
		Description:   "Background config",
		Config:        part,
	}
	background.NormalizeGenerated(&generated)
	return generated.Config
}

func validateBackgroundConfig(part background.Config) []design.Issue {
	return background.ValidateGenerated(design.GeneratedConfig[background.Config]{
		ComponentKind: background.Kind,
		Description:   "Background config",
		Config:        part,
	})
}

func normalizeTextareaConfig(part textarea.Config) textarea.Config {
	generated := design.GeneratedConfig[textarea.Config]{
		ComponentKind: textarea.Kind,
		Description:   "Text config",
		Config:        part,
	}
	textarea.NormalizeGenerated(&generated)
	return generated.Config
}

func validateTextareaConfig(part textarea.Config) []design.Issue {
	return textarea.ValidateGenerated(design.GeneratedConfig[textarea.Config]{
		ComponentKind: textarea.Kind,
		Description:   "Text config",
		Config:        part,
	})
}

func normalizeShapeConfig(part shape.Config) shape.Config {
	generated := design.GeneratedConfig[shape.Config]{
		ComponentKind: shape.Kind,
		Description:   "Shape config",
		Config:        part,
	}
	shape.NormalizeGenerated(&generated)
	return generated.Config
}

func validateShapeConfig(part shape.Config) []design.Issue {
	return shape.ValidateGenerated(design.GeneratedConfig[shape.Config]{
		ComponentKind: shape.Kind,
		Description:   "Shape config",
		Config:        part,
	})
}

func normalizeImageConfig(part imagecomponent.Config) imagecomponent.Config {
	generated := design.GeneratedConfig[imagecomponent.Config]{
		ComponentKind: imagecomponent.Kind,
		Description:   "Image config",
		Config:        part,
	}
	imagecomponent.NormalizeGenerated(&generated)
	return generated.Config
}

func validateImageConfig(part imagecomponent.Config) []design.Issue {
	return imagecomponent.ValidateGenerated(design.GeneratedConfig[imagecomponent.Config]{
		ComponentKind: imagecomponent.Kind,
		Description:   "Image config",
		Config:        part,
	})
}

func updateTextareaCSS(part textarea.Config, control string) string {
	updates := map[string]string{}
	switch control {
	case "font_size_px":
		updates["font-size"] = fmt.Sprintf("%dpx", part.FontSizePX)
	case "font_weight":
		updates["font-weight"] = fmt.Sprintf("%d", part.FontWeight)
	case "font_style":
		updates["font-style"] = part.FontStyle
	case "color":
		updates["color"] = part.Color
	case "align":
		updates["text-align"] = part.Align
	case "background_color":
		updates["background-color"] = part.BackgroundColor
	case "border_color", "border_width_px":
		updates["border"] = fmt.Sprintf("%dpx solid %s", part.BorderWidthPX, part.BorderColor)
		updates["border-color"] = part.BorderColor
		updates["border-width"] = fmt.Sprintf("%dpx", part.BorderWidthPX)
	case "border_radius_px":
		updates["border-radius"] = fmt.Sprintf("%dpx", part.BorderRadiusPX)
	case "padding_px":
		updates["padding"] = fmt.Sprintf("%dpx", part.PaddingPX)
	default:
		return part.CSS
	}
	return cssWithDeclarations(part.CSS, textarea.AllowedCSS(), updates)
}

func cssWithDeclarations(css string, allowed map[string]struct{}, updates map[string]string) string {
	declarations := design.CSSDeclarations(css, allowed)
	for property, value := range updates {
		property = strings.ToLower(strings.TrimSpace(property))
		value = strings.TrimSpace(value)
		if property == "" {
			continue
		}
		if value == "" {
			delete(declarations, property)
			continue
		}
		if _, ok := allowed[property]; ok {
			declarations[property] = value
		}
	}
	if len(declarations) == 0 {
		return ""
	}
	keys := make([]string, 0, len(declarations))
	for key := range declarations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		value := strings.TrimSpace(declarations[key])
		if value == "" {
			continue
		}
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value)
		out.WriteString("; ")
	}
	return strings.TrimSpace(out.String())
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
