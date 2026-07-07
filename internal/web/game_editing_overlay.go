package web

import (
	"encoding/json"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/design"
	"github.com/n0remac/Living-Card/internal/game"
)

func gameEditingOverlay(card game.Card, selectedComponentID string) *ComponentOverlay {
	selectedComponentID = strings.TrimSpace(selectedComponentID)
	if selectedComponentID != "" {
		if node := findNodeByID(card.Document.Root, selectedComponentID); node != nil {
			return gameEditingOverlayForNode(*node)
		}
	}
	return nil
}

func gameEditingOverlayForNode(node cardcomponent.Node) *ComponentOverlay {
	switch node.ComponentKind {
	case slider.Kind:
		return sliderEditingOverlay(node)
	case border.Kind:
		return borderEditingOverlay(node)
	default:
		return nil
	}
}

func sliderEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	part := slider.DefaultConfig()
	if len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	part = slider.NormalizeConfig(part)
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    slider.Kind,
		Title:            "Slider",
		RandomizeEnabled: false,
		Controls: []ControlDescriptor{
			textEditControl("html", "label", "Label", `label`, part.Label),
			rangeEditControl("html", "value", "Current value", `input[type=range].value`, part.Value, part.Min, part.Max, part.Step),
			rangeEditControl("html", "min", "Minimum", `input[type=range].min`, part.Min, 0, 100, 1),
			rangeEditControl("html", "max", "Maximum", `input[type=range].max`, part.Max, 0, 100, 1),
			rangeEditControl("html", "step", "Step", `input[type=range].step`, part.Step, 1, 25, 1),
			rangeEditControl("layout", "x", "X position", "left", part.X, 0, 100, 1),
			rangeEditControl("layout", "y", "Y position", "top", part.Y, 0, 100, 1),
			rangeEditControl("layout", "width", "Width", "width", part.Width, 12, 100, 1),
			colorEditControl("style", "track_color", "Track color", "background-color", part.TrackColor),
			colorEditControl("style", "accent_color", "Accent color", "accent-color", part.AccentColor),
		},
	}
}

func borderEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	part := border.DefaultConfig()
	if len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := normalizeBorderOverlayConfig(part)
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    border.Kind,
		Title:            "Border",
		RandomizeEnabled: false,
		Controls: []ControlDescriptor{
			colorEditControl("style", "border_color", "Color", "border-color", generated.BorderColor),
			rangeEditControl("style", "border_width_px", "Width", "border-width", generated.BorderWidthPX, 0, 16, 1),
			rangeEditControl("style", "border_radius_px", "Radius", "border-radius", generated.BorderRadiusPX, 0, 64, 1),
			{
				Trait:    "style",
				Control:  "border_style",
				Kind:     "select",
				Label:    "Line type",
				Property: "border-style",
				Value:    generated.BorderStyle,
				Options: []ControlOption{
					{Label: "Solid", Value: "solid"},
					{Label: "Dashed", Value: "dashed"},
					{Label: "Dotted", Value: "dotted"},
					{Label: "Double", Value: "double"},
				},
			},
		},
	}
}

func textEditControl(trait, control, label, property string, value any) ControlDescriptor {
	return ControlDescriptor{Trait: trait, Control: control, Kind: "text", Label: label, Property: property, Value: value}
}

func colorEditControl(trait, control, label, property string, value any) ControlDescriptor {
	return ControlDescriptor{Trait: trait, Control: control, Kind: "color", Label: label, Property: property, Value: value}
}

func rangeEditControl(trait, control, label, property string, value, min, max, step int) ControlDescriptor {
	return ControlDescriptor{
		Trait:    trait,
		Control:  control,
		Kind:     "range",
		Label:    label,
		Property: property,
		Value:    value,
		Min:      min,
		Max:      max,
		Step:     step,
	}
}

func normalizeBorderOverlayConfig(part border.Config) border.Config {
	generated := design.GeneratedConfig[border.Config]{
		ComponentKind: border.Kind,
		Description:   "Border config",
		Config:        part,
	}
	border.NormalizeGenerated(&generated)
	return generated.Config
}
