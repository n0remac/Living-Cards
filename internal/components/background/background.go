package background

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = "background"

type Config struct {
	BackgroundColor string `json:"background_color"`
	CSS             string `json:"css"`
}

func DefaultConfig() Config {
	return Config{
		BackgroundColor: "#111827",
		CSS:             "",
	}
}

func Presets() []card.LibraryItem {
	return Definition().Presets()
}

func typedPresets() []card.TypedPreset[Config] {
	return []card.TypedPreset[Config]{
		{ID: "seed-background-night-sky", Name: "Night Sky", Description: "Deep blue night-sky background", Config: Config{
			BackgroundColor: "#0f172a",
			CSS:             "background: radial-gradient(circle at top, #1e3a8a 0%, #0f172a 60%, #020617 100%);",
		}},
		{ID: "seed-background-parchment", Name: "Parchment", Description: "Warm parchment background", Config: Config{
			BackgroundColor: "#f5e6c8",
			CSS:             "background: linear-gradient(135deg, #f8edd5 0%, #e7cfa6 100%);",
		}},
		{ID: "seed-background-mint", Name: "Soft Mint", Description: "Soft mint studio background", Config: Config{
			BackgroundColor: "#d9f99d",
			CSS:             "background: linear-gradient(145deg, #ecfccb 0%, #bbf7d0 100%); box-shadow: inset 0 0 40px rgba(22, 101, 52, 0.12);",
		}},
	}
}

func RandomGenerated(seed int64, level int) schema.GeneratedConfig[Config] {
	options := []struct {
		description string
		part        Config
	}{
		{
			description: "A midnight card surface with a cool highlight.",
			part: Config{
				BackgroundColor: "#0f172a",
				CSS:             "background: radial-gradient(circle at top, rgba(56,189,248,0.24), transparent 44%), linear-gradient(160deg, #0f172a 0%, #111827 100%);",
			},
		},
		{
			description: "A warm parchment card surface.",
			part: Config{
				BackgroundColor: "#f5e6c8",
				CSS:             "background: linear-gradient(135deg, #f8edd5 0%, #e7cfa6 100%); box-shadow: inset 0 0 36px rgba(120, 53, 15, 0.12);",
			},
		},
		{
			description: "A soft mint card surface with gentle depth.",
			part: Config{
				BackgroundColor: "#d9f99d",
				CSS:             "background: linear-gradient(145deg, #ecfccb 0%, #bbf7d0 100%); box-shadow: inset 0 0 40px rgba(22, 101, 52, 0.12);",
			},
		},
		{
			description: "A rose dusk card surface.",
			part: Config{
				BackgroundColor: "#581c87",
				CSS:             "background: radial-gradient(circle at top right, rgba(244,114,182,0.28), transparent 42%), linear-gradient(155deg, #581c87 0%, #1f2937 100%);",
			},
		},
		{
			description: "A quiet stone card surface.",
			part: Config{
				BackgroundColor: "#374151",
				CSS:             "background: linear-gradient(150deg, #4b5563 0%, #1f2937 100%); box-shadow: inset 0 0 34px rgba(255,255,255,0.08);",
			},
		},
	}
	if level > 2 {
		options = append(options, struct {
			description string
			part        Config
		}{
			description: "A bright ember card surface.",
			part: Config{
				BackgroundColor: "#7c2d12",
				CSS:             "background: radial-gradient(circle at bottom, rgba(251,146,60,0.34), transparent 42%), linear-gradient(145deg, #7c2d12 0%, #111827 100%);",
			},
		})
	}
	pick := options[rand.New(rand.NewSource(seed)).Intn(len(options))]
	return schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   pick.description,
		Config:        pick.part,
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Background", Structure: card.StructureLeaf,
		Default: DefaultConfig, Normalize: NormalizeConfig, Validate: ValidateConfig,
		Render: func(_ card.Node, part Config, _ card.RenderContext) (card.Contribution, error) {
			styles := map[string]string{
				"background-color": part.BackgroundColor,
			}
			for property, value := range schema.CSSDeclarations(part.CSS, AllowedCSS()) {
				styles[property] = value
			}
			return card.Contribution{ShellStyle: styles}, nil
		},
		Controls:   []card.Control[Config]{backgroundColorControl()},
		Install:    &card.InstallSpec{Policy: card.InstallReplaceKind},
		Presets:    typedPresets(),
		Generation: &card.TypedGenerationDefinition[Config]{SystemPrompt: systemPrompt, Example: exampleJSON, Random: RandomGenerated},
	})
}

func backgroundColorControl() card.Control[Config] {
	control := card.StringControl("background_color", "style", "color", "Color", "background-color", func(c Config) string { return c.BackgroundColor }, func(c *Config, value string) { c.BackgroundColor = value })
	control.Apply = func(config *Config, raw json.RawMessage) error {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("value must be a string")
		}
		config.BackgroundColor = value
		declarations := schema.CSSDeclarations(config.CSS, AllowedCSS())
		declarations["background"] = value
		declarations["background-color"] = value
		config.CSS = cssString(declarations)
		return nil
	}
	return control
}

func NormalizeConfig(config Config) Config {
	config.BackgroundColor = strings.TrimSpace(config.BackgroundColor)
	config.CSS = strings.TrimSpace(config.CSS)
	return config
}

func ValidateConfig(config Config) []schema.Issue {
	return ValidateGenerated(schema.GeneratedConfig[Config]{ComponentKind: Kind, Description: "Background config", Config: config})
}

func NormalizeGenerated(generated *schema.GeneratedConfig[Config]) {
	if generated == nil {
		return
	}
	generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Config = NormalizeConfig(generated.Config)
}

func ValidateGenerated(generated schema.GeneratedConfig[Config]) []schema.Issue {
	var issues []schema.Issue
	color := strings.TrimSpace(generated.Config.BackgroundColor)
	if color == "" {
		issues = append(issues, schema.Issue{
			Path:    "config.background_color",
			Code:    "required",
			Message: "background_color is required",
		})
	} else if !schema.IsAllowedColor(color) {
		issues = append(issues, schema.Issue{
			Path:    "config.background_color",
			Code:    "invalid_color",
			Message: "background_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	issues = append(issues, schema.ValidateInlineCSS("config.css", generated.Config.CSS, AllowedCSS())...)
	return issues
}

func AllowedCSS() map[string]struct{} {
	return map[string]struct{}{
		"background":       {},
		"background-color": {},
		"box-shadow":       {},
	}
}

func cssString(declarations map[string]string) string {
	keys := make([]string, 0, len(declarations))
	for key := range declarations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		if value := strings.TrimSpace(declarations[key]); value != "" {
			fmt.Fprintf(&out, "%s: %s; ", key, value)
		}
	}
	return strings.TrimSpace(out.String())
}

const exampleJSON = `{
  "component_kind": "background",
  "description": "A deep slate card background with a subtle radial highlight.",
  "config": {
    "background_color": "#111827",
    "css": "background: radial-gradient(circle at top, rgba(56,189,248,0.22), transparent 45%), #111827;"
  }
}`

const systemPrompt = `You generate safe declarative JSON configs for the background component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match this shape:
{
  "component_kind": "background",
  "description": "short human-readable summary",
  "config": {
    "background_color": "#111827",
    "css": "optional inline CSS declarations"
  }
}
Rules:
- component_kind must be "background".
- description is required.
- background_color must be a safe color: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- css is optional inline declarations only.
- Allowed css properties: background, background-color, box-shadow.
- Do not output url(...), javascript:, expression(...), @import, position, content, raw HTML, selectors, braces, or JavaScript.`
