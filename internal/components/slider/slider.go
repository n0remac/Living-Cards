package slider

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = "slider"

type Config struct {
	Label       string `json:"label"`
	Min         int    `json:"min"`
	Max         int    `json:"max"`
	Step        int    `json:"step"`
	Value       int    `json:"value"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Width       int    `json:"width"`
	TrackColor  string `json:"track_color"`
	AccentColor string `json:"accent_color"`
}

func DefaultConfig() Config {
	return Config{
		Label:       "Output",
		Min:         0,
		Max:         100,
		Step:        1,
		Value:       50,
		X:           50,
		Y:           70,
		Width:       72,
		TrackColor:  "rgba(15,23,42,0.72)",
		AccentColor: "#7dd3fc",
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Slider", Structure: card.StructureLeaf,
		Default: DefaultConfig, Normalize: NormalizeConfig, Validate: ValidateConfig,
		ConfigRules: []schema.FieldRule{
			schema.StringMinLength("label", 1),
			schema.IntegerRange("min", 0, 100),
			schema.IntegerRange("max", 0, 100),
			schema.IntegerRange("step", 1, 100),
			schema.IntegerRange("value", 0, 100),
			schema.IntegerRange("x", 0, 100),
			schema.IntegerRange("y", 0, 100),
			schema.IntegerRange("width", 12, 100),
			schema.StringFormat("track_color", schema.FormatCSSColor),
			schema.StringFormat("accent_color", schema.FormatCSSColor),
		},
		Render: func(node card.Node, part Config, renderContext card.RenderContext) (card.Contribution, error) {
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)},
			}, nil
		},
		Controls: []card.Control[Config]{
			card.StringControl("label", "content", "text", "Label", "label", func(c Config) string { return c.Label }, func(c *Config, v string) { c.Label = v }),
			card.IntControl("value", "value", "range", "Value", "value", 1, func(c Config) int { return c.Value }, func(c *Config, v int) { c.Value = v }),
			card.IntControl("min", "value", "range", "Minimum", "min", 1, func(c Config) int { return c.Min }, func(c *Config, v int) { c.Min = v }),
			card.IntControl("max", "value", "range", "Maximum", "max", 1, func(c Config) int { return c.Max }, func(c *Config, v int) { c.Max = v }),
			card.IntControl("step", "value", "range", "Step", "step", 1, func(c Config) int { return c.Step }, func(c *Config, v int) { c.Step = v }),
			positionControl(),
			card.IntControl("x", "layout", "range", "X position", "left", 1, func(c Config) int { return c.X }, func(c *Config, v int) { c.X = v }),
			card.IntControl("y", "layout", "range", "Y position", "top", 1, func(c Config) int { return c.Y }, func(c *Config, v int) { c.Y = v }),
			card.IntControl("width", "layout", "range", "Width", "width", 1, func(c Config) int { return c.Width }, func(c *Config, v int) { c.Width = v }),
			card.StringControl("track_color", "style", "color", "Track color", "background", func(c Config) string { return c.TrackColor }, func(c *Config, v string) { c.TrackColor = v }),
			card.StringControl("accent_color", "style", "color", "Accent color", "accent-color", func(c Config) string { return c.AccentColor }, func(c *Config, v string) { c.AccentColor = v }),
		},
		Properties: []card.Property[Config]{{ID: "value", Kind: schema.PropertyNumber, Read: func(c Config) (schema.PropertyValue, bool) { return schema.NumberValue(float64(c.Value)), true }}},
		Install:    &card.InstallSpec{Policy: card.InstallAppend},
	})
}

func NormalizeConfig(part Config) Config {
	part.Label = strings.TrimSpace(part.Label)
	part.TrackColor = strings.TrimSpace(part.TrackColor)
	part.AccentColor = strings.TrimSpace(part.AccentColor)
	return part
}

func ValidateConfig(part Config) []schema.Issue {
	var issues []schema.Issue
	if part.Max < part.Min {
		issues = append(issues, schema.Issue{
			Path:    "config.max",
			Code:    "out_of_range",
			Message: "max must be greater than or equal to min",
			Actual:  part.Max,
		})
	}
	if part.Value < part.Min || part.Value > part.Max {
		issues = append(issues, schema.Issue{
			Path:    "config.value",
			Code:    "out_of_range",
			Message: "value must be between min and max",
			Actual:  part.Value,
		})
	}
	return issues
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = NormalizeConfig(part)
	style := map[string]string{
		"background":     part.TrackColor,
		"border":         "1px solid " + part.AccentColor,
		"border-radius":  "14px",
		"box-shadow":     "0 14px 34px rgba(8,47,73,0.22)",
		"color":          "#e0f2fe",
		"display":        "grid",
		"gap":            "8px",
		"left":           fmt.Sprintf("%d%%", part.X),
		"padding":        "12px",
		"pointer-events": "auto",
		"top":            fmt.Sprintf("%d%%", part.Y),
		"transform":      "translate(-50%, -50%)",
		"width":          fmt.Sprintf("%d%%", part.Width),
		"z-index":        "2",
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("style", styleString(style)),
		godom.Div(
			godom.Class("flex items-center justify-between gap-2 text-xs font-bold uppercase"),
			godom.Span(godom.T(part.Label)),
			godom.Span(
				godom.Attr("data-slider-value", ""),
				godom.T(fmt.Sprintf("%d", part.Value)),
			),
		),
		godom.Input(
			godom.Type("range"),
			godom.Attr("data-slider-input", ""),
			godom.Attr("min", fmt.Sprintf("%d", part.Min)),
			godom.Attr("max", fmt.Sprintf("%d", part.Max)),
			godom.Attr("step", fmt.Sprintf("%d", part.Step)),
			godom.Value(fmt.Sprintf("%d", part.Value)),
			godom.Attr("aria-label", part.Label),
			godom.Attr("style", "accent-color: "+part.AccentColor+";"),
			godom.Class("w-full"),
		),
	)
}

func styleString(styles map[string]string) string {
	keys := make([]string, 0, len(styles))
	for key := range styles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		value := strings.TrimSpace(styles[key])
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

func positionControl() card.Control[Config] {
	type position struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	return card.Control[Config]{ID: "position", ValueSchema: positionControlSchema(), Descriptor: card.ControlDescriptor{Trait: "layout", Kind: "position", Label: "Position", Property: "position"}, Read: func(c Config) json.RawMessage { raw, _ := json.Marshal(position{c.X, c.Y}); return raw }, Apply: func(c *Config, raw json.RawMessage) error {
		var value position
		if err := card.DecodeControlObject(raw, &value); err != nil {
			return fmt.Errorf("value must include integer x and y: %w", err)
		}
		c.X, c.Y = value.X, value.Y
		return nil
	}}
}

func positionControlSchema() schema.ValueSchema {
	minimum, maximum := float64(0), float64(100)
	coordinate := schema.ValueSchema{Kind: schema.ValueInteger, Minimum: &minimum, Maximum: &maximum}
	return schema.ValueSchema{Kind: schema.ValueObject, Fields: []schema.FieldSchema{
		{JSONName: "x", Schema: coordinate, Required: true},
		{JSONName: "y", Schema: coordinate, Required: true},
	}}
}
