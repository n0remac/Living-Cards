package web

import (
	"encoding/json"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
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

func gameActiveEditingOverlay(card game.Card, selectedComponentID string, library []game.Card) *ComponentOverlay {
	selectedComponentID = strings.TrimSpace(selectedComponentID)
	if selectedComponentID == "" {
		return nil
	}
	node := findNodeByID(card.Document.Root, selectedComponentID)
	if node == nil {
		return nil
	}
	switch node.ComponentKind {
	case background.Kind, border.Kind:
		if !gameLibraryHasComponentKind(library, node.ComponentKind) {
			return nil
		}
	}
	return gameEditingOverlayForNode(*node)
}

func gameEditingOverlayForNode(node cardcomponent.Node) *ComponentOverlay {
	switch node.ComponentKind {
	case background.Kind:
		return backgroundEditingOverlay(node)
	case slider.Kind:
		return sliderEditingOverlay(node)
	case border.Kind:
		return borderEditingOverlay(node)
	case textarea.Kind:
		return textareaEditingOverlay(node)
	case shape.Kind:
		return shapeEditingOverlay(node)
	case imagecomponent.Kind:
		return imageEditingOverlay(node)
	default:
		return nil
	}
}

func backgroundEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	part := background.DefaultConfig()
	if len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[background.Config]{
		ComponentKind: background.Kind,
		Description:   "Background config",
		Config:        part,
	}
	background.NormalizeGenerated(&generated)
	part = generated.Config
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    background.Kind,
		Title:            "Background",
		RandomizeEnabled: false,
		Controls: []ControlDescriptor{
			colorEditControl("style", "background_color", "Color", "background-color", part.BackgroundColor),
		},
	}
}

func sliderEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    slider.Kind,
		Title:            "Slider",
		RandomizeEnabled: false,
		Controls:         nil,
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

func textareaEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	part := textarea.DefaultConfig()
	if len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[textarea.Config]{
		ComponentKind: textarea.Kind,
		Description:   "Text config",
		Config:        part,
	}
	textarea.NormalizeGenerated(&generated)
	part = generated.Config
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    textarea.Kind,
		Title:            "Text",
		RandomizeEnabled: false,
		Controls: []ControlDescriptor{
			textEditControl("content", "content", "Text", "content", part.Content),
			rangeEditControl("layout", "x", "X position", "left", part.X, 0, 100, 1),
			rangeEditControl("layout", "y", "Y position", "top", part.Y, 0, 100, 1),
			rangeEditControl("type", "font_size_px", "Font size", "font-size", part.FontSizePX, 10, 72, 1),
			rangeEditControl("layout", "padding_px", "Padding", "padding", part.PaddingPX, 0, 32, 1),
			colorEditControl("style", "color", "Text color", "color", part.Color),
			colorEditControl("style", "background_color", "Fill color", "background-color", part.BackgroundColor),
			colorEditControl("style", "border_color", "Border color", "border-color", part.BorderColor),
			rangeEditControl("style", "border_width_px", "Border width", "border-width", part.BorderWidthPX, 0, 12, 1),
			rangeEditControl("style", "border_radius_px", "Border radius", "border-radius", part.BorderRadiusPX, 0, 40, 1),
		},
	}
}

func shapeEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	part := shape.DefaultConfig()
	if len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[shape.Config]{
		ComponentKind: shape.Kind,
		Description:   "Shape config",
		Config:        part,
	}
	shape.NormalizeGenerated(&generated)
	part = generated.Config
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    shape.Kind,
		Title:            "Shape",
		RandomizeEnabled: false,
		Controls: []ControlDescriptor{
			{
				Trait:    "shape",
				Control:  "shape",
				Kind:     "select",
				Label:    "Shape",
				Property: "shape",
				Value:    part.Shape,
				Options: []ControlOption{
					{Label: "Circle", Value: "circle"},
					{Label: "Oval", Value: "oval"},
					{Label: "Rectangle", Value: "rectangle"},
					{Label: "Rounded rectangle", Value: "roundedRectangle"},
					{Label: "Triangle", Value: "triangle"},
					{Label: "Diamond", Value: "diamond"},
					{Label: "Star", Value: "star"},
					{Label: "Blob", Value: "blob"},
				},
			},
			rangeEditControl("layout", "x", "X position", "left", part.X, 0, 100, 1),
			rangeEditControl("layout", "y", "Y position", "top", part.Y, 0, 100, 1),
			rangeEditControl("layout", "width", "Width", "width", part.Width, 8, 100, 1),
			rangeEditControl("layout", "height", "Height", "height", part.Height, 8, 100, 1),
			rangeEditControl("layout", "rotation", "Rotation", "transform", part.Rotation, 0, 359, 1),
			colorEditControl("style", "background_color", "Fill color", "fill", part.BackgroundColor),
			colorEditControl("style", "border_color", "Border color", "stroke", part.BorderColor),
			rangeEditControl("style", "border_width_px", "Border width", "stroke-width", part.BorderWidthPX, 0, 10, 1),
			{
				Trait:    "style",
				Control:  "shadow",
				Kind:     "select",
				Label:    "Shadow",
				Property: "filter",
				Value:    part.Shadow,
				Options: []ControlOption{
					{Label: "None", Value: ""},
					{Label: "Soft slate", Value: "0 10px 24px rgba(15,23,42,0.22)"},
					{Label: "Deep slate", Value: "0 12px 28px rgba(15,23,42,0.28)"},
					{Label: "Rose glow", Value: "0 12px 26px rgba(244,63,94,0.24)"},
					{Label: "Sky glow", Value: "0 10px 24px rgba(14,165,233,0.22)"},
				},
			},
		},
	}
}

func imageEditingOverlay(node cardcomponent.Node) *ComponentOverlay {
	part := imagecomponent.DefaultConfig()
	if len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[imagecomponent.Config]{
		ComponentKind: imagecomponent.Kind,
		Description:   "Image config",
		Config:        part,
	}
	imagecomponent.NormalizeGenerated(&generated)
	part = generated.Config
	return &ComponentOverlay{
		ComponentID:      node.ID,
		ComponentKind:    imagecomponent.Kind,
		Title:            "Image",
		RandomizeEnabled: false,
		Controls: []ControlDescriptor{
			textEditControl("content", "alt", "Alt text", "alt", part.Alt),
			rangeEditControl("layout", "x", "X position", "left", part.X, 0, 100, 1),
			rangeEditControl("layout", "y", "Y position", "top", part.Y, 0, 100, 1),
			rangeEditControl("layout", "width", "Width", "width", part.Width, 8, 100, 1),
			rangeEditControl("layout", "height", "Height", "height", part.Height, 8, 100, 1),
			rangeEditControl("layout", "rotation", "Rotation", "transform", part.Rotation, 0, 359, 1),
			colorEditControl("style", "border_color", "Border color", "border-color", part.BorderColor),
			rangeEditControl("style", "border_width_px", "Border width", "border-width", part.BorderWidthPX, 0, 12, 1),
			rangeEditControl("style", "border_radius_px", "Border radius", "border-radius", part.BorderRadiusPX, 0, 48, 1),
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

func gameLibraryHasComponentKind(cards []game.Card, componentKind string) bool {
	componentKind = strings.TrimSpace(componentKind)
	for _, card := range cards {
		value, ok := card.State["componentKind"].(string)
		if ok && strings.TrimSpace(value) == componentKind {
			return true
		}
	}
	return false
}
