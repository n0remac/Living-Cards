package creature

import (
	"fmt"
	"math"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = card.KindCreature

type Config struct {
	Health          int    `json:"health"`
	MaxHealth       int    `json:"max_health"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Width           int    `json:"width"`
	HealthColor     string `json:"health_color"`
	BackgroundColor string `json:"background_color"`
	AccentColor     string `json:"accent_color"`
}

func DefaultConfig() Config {
	return Config{
		Health: 4, MaxHealth: 4, X: 50, Y: 82, Width: 72,
		HealthColor: "#4ade80", BackgroundColor: "rgba(15,23,42,0.86)", AccentColor: "#bbf7d0",
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Creature", Structure: card.StructureLeaf,
		Default: DefaultConfig, Normalize: NormalizeConfig, Validate: ValidateConfig,
		ConfigRules: []schema.FieldRule{
			schema.IntegerRange("health", 0, 999),
			schema.IntegerRange("max_health", 1, 999),
			schema.IntegerRange("x", 0, 100),
			schema.IntegerRange("y", 0, 100),
			schema.IntegerRange("width", 12, 100),
			schema.StringFormat("health_color", schema.FormatCSSColor),
			schema.StringFormat("background_color", schema.FormatCSSColor),
			schema.StringFormat("accent_color", schema.FormatCSSColor),
		},
		Render: func(node card.Node, config Config, context card.RenderContext) (card.Contribution, error) {
			return card.Contribution{Layers: []*godom.Node{RenderLayerWithContext(node.ID, config, context)}}, nil
		},
		Controls: []card.Control[Config]{
			card.IntControl("x", "layout", "range", "X position", "left", 1, func(c Config) int { return c.X }, func(c *Config, value int) { c.X = value }),
			card.IntControl("y", "layout", "range", "Y position", "top", 1, func(c Config) int { return c.Y }, func(c *Config, value int) { c.Y = value }),
			card.IntControl("width", "layout", "range", "Width", "width", 1, func(c Config) int { return c.Width }, func(c *Config, value int) { c.Width = value }),
			card.StringControl("health_color", "style", "color", "Health color", "color", func(c Config) string { return c.HealthColor }, func(c *Config, value string) { c.HealthColor = value }),
			card.StringControl("background_color", "style", "color", "Background color", "background", func(c Config) string { return c.BackgroundColor }, func(c *Config, value string) { c.BackgroundColor = value }),
			card.StringControl("accent_color", "style", "color", "Accent color", "border-color", func(c Config) string { return c.AccentColor }, func(c *Config, value string) { c.AccentColor = value }),
		},
		Properties: []card.Property[Config]{
			{
				ID: "health", Kind: schema.PropertyNumber,
				Read: func(c Config) (schema.PropertyValue, bool) { return schema.NumberValue(float64(c.Health)), true },
				Write: func(c *Config, value schema.PropertyValue) error {
					if math.IsNaN(value.Number) || math.IsInf(value.Number, 0) || math.Trunc(value.Number) != value.Number {
						return fmt.Errorf("health must be an integer")
					}
					c.Health = int(value.Number)
					return nil
				},
			},
			{ID: "max_health", Kind: schema.PropertyNumber, Read: func(c Config) (schema.PropertyValue, bool) { return schema.NumberValue(float64(c.MaxHealth)), true }},
		},
	})
}

func NormalizeConfig(config Config) Config {
	config.HealthColor = strings.TrimSpace(config.HealthColor)
	config.BackgroundColor = strings.TrimSpace(config.BackgroundColor)
	config.AccentColor = strings.TrimSpace(config.AccentColor)
	return config
}

func ValidateConfig(config Config) []schema.Issue {
	if config.Health > config.MaxHealth {
		return []schema.Issue{{Path: "config.health", Code: "out_of_range", Message: "health must not exceed max_health", Actual: config.Health}}
	}
	return nil
}

func RenderLayer(componentID string, config Config) *godom.Node {
	return RenderLayerWithContext(componentID, config, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, config Config, context card.RenderContext) *godom.Node {
	config = NormalizeConfig(config)
	percent := 0
	if config.MaxHealth > 0 {
		percent = config.Health * 100 / config.MaxHealth
	}
	return godom.Div(
		godom.Id(context.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("data-health", fmt.Sprintf("%d", config.Health)),
		godom.Attr("data-max-health", fmt.Sprintf("%d", config.MaxHealth)),
		godom.Attr("style", styleString(map[string]string{
			"background": config.BackgroundColor, "border": "1px solid " + config.AccentColor,
			"border-radius": "14px", "color": config.AccentColor, "left": fmt.Sprintf("%d%%", config.X),
			"padding": "10px 12px", "pointer-events": "none", "top": fmt.Sprintf("%d%%", config.Y),
			"transform": "translate(-50%, -50%)", "width": fmt.Sprintf("%d%%", config.Width), "z-index": "3",
		})),
		godom.Div(
			godom.Class("flex items-center justify-between text-xs font-bold uppercase"),
			godom.Span(godom.T("Health")),
			godom.Span(godom.Attr("data-creature-health", ""), godom.T(fmt.Sprintf("%d / %d", config.Health, config.MaxHealth))),
		),
		godom.Div(
			godom.Attr("aria-label", fmt.Sprintf("Health %d of %d", config.Health, config.MaxHealth)),
			godom.Attr("role", "meter"), godom.Attr("aria-valuemin", "0"),
			godom.Attr("aria-valuemax", fmt.Sprintf("%d", config.MaxHealth)), godom.Attr("aria-valuenow", fmt.Sprintf("%d", config.Health)),
			godom.Attr("style", "height: 8px; margin-top: 7px; overflow: hidden; border-radius: 999px; background: rgba(255,255,255,0.16);"),
			godom.Div(godom.Attr("data-creature-health-bar", ""), godom.Attr("style", fmt.Sprintf("height: 100%%; width: %d%%; background: %s;", percent, config.HealthColor))),
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
		if value := strings.TrimSpace(styles[key]); value != "" {
			out.WriteString(key)
			out.WriteString(": ")
			out.WriteString(value)
			out.WriteString("; ")
		}
	}
	return strings.TrimSpace(out.String())
}
