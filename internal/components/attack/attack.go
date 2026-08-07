package attack

import (
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = card.KindAttack

type Config struct {
	Label           string `json:"label"`
	Power           int    `json:"power"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Width           int    `json:"width"`
	BackgroundColor string `json:"background_color"`
	AccentColor     string `json:"accent_color"`
}

func DefaultConfig() Config {
	return Config{Label: "Attack", Power: 1, X: 50, Y: 64, Width: 64, BackgroundColor: "rgba(69,10,10,0.86)", AccentColor: "#fca5a5"}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Attack", Structure: card.StructureLeaf,
		Default: DefaultConfig, Normalize: NormalizeConfig,
		ConfigRules: []schema.FieldRule{
			schema.StringMinLength("label", 1), schema.IntegerRange("power", 1, 999),
			schema.IntegerRange("x", 0, 100), schema.IntegerRange("y", 0, 100), schema.IntegerRange("width", 12, 100),
			schema.StringFormat("background_color", schema.FormatCSSColor), schema.StringFormat("accent_color", schema.FormatCSSColor),
		},
		Render: func(node card.Node, config Config, context card.RenderContext) (card.Contribution, error) {
			return card.Contribution{Layers: []*godom.Node{RenderLayerWithContext(node.ID, config, context)}}, nil
		},
		Controls: []card.Control[Config]{
			card.StringControl("label", "content", "text", "Label", "label", func(c Config) string { return c.Label }, func(c *Config, value string) { c.Label = value }),
			card.IntControl("x", "layout", "range", "X position", "left", 1, func(c Config) int { return c.X }, func(c *Config, value int) { c.X = value }),
			card.IntControl("y", "layout", "range", "Y position", "top", 1, func(c Config) int { return c.Y }, func(c *Config, value int) { c.Y = value }),
			card.IntControl("width", "layout", "range", "Width", "width", 1, func(c Config) int { return c.Width }, func(c *Config, value int) { c.Width = value }),
			card.StringControl("background_color", "style", "color", "Background color", "background", func(c Config) string { return c.BackgroundColor }, func(c *Config, value string) { c.BackgroundColor = value }),
			card.StringControl("accent_color", "style", "color", "Accent color", "border-color", func(c Config) string { return c.AccentColor }, func(c *Config, value string) { c.AccentColor = value }),
		},
		Properties: []card.Property[Config]{{ID: "power", Kind: schema.PropertyNumber, Read: func(c Config) (schema.PropertyValue, bool) { return schema.NumberValue(float64(c.Power)), true }}},
		Install:    &card.InstallSpec{Policy: card.InstallAppend},
	})
}

func NormalizeConfig(config Config) Config {
	config.Label = strings.TrimSpace(config.Label)
	config.BackgroundColor = strings.TrimSpace(config.BackgroundColor)
	config.AccentColor = strings.TrimSpace(config.AccentColor)
	return config
}

func RenderLayer(componentID string, config Config) *godom.Node {
	return RenderLayerWithContext(componentID, config, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, config Config, context card.RenderContext) *godom.Node {
	config = NormalizeConfig(config)
	return godom.Div(
		godom.Id(context.LayerID(componentID)), godom.Class("absolute"),
		godom.Attr("data-component-id", componentID), godom.Attr("data-component-kind", Kind), godom.Attr("data-attack-power", fmt.Sprintf("%d", config.Power)),
		godom.Attr("style", styleString(map[string]string{
			"background": config.BackgroundColor, "border": "1px solid " + config.AccentColor, "border-radius": "14px", "color": config.AccentColor,
			"display": "flex", "justify-content": "space-between", "left": fmt.Sprintf("%d%%", config.X), "padding": "10px 12px", "pointer-events": "none",
			"top": fmt.Sprintf("%d%%", config.Y), "transform": "translate(-50%, -50%)", "width": fmt.Sprintf("%d%%", config.Width), "z-index": "3",
		})),
		godom.Span(godom.Class("text-xs font-bold uppercase"), godom.T(config.Label)),
		godom.Span(godom.Attr("data-attack-power-label", ""), godom.Class("text-sm font-black"), godom.T(fmt.Sprintf("+%d", config.Power))),
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
		if value := strings.TrimSpace(styles[key]); value != "" {
			out.WriteString(key)
			out.WriteString(": ")
			out.WriteString(value)
			out.WriteString("; ")
		}
	}
	return strings.TrimSpace(out.String())
}
