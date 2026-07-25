package border

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = "border"

type Config struct {
	BorderWidthPX  int    `json:"border_width_px"`
	BorderRadiusPX int    `json:"border_radius_px"`
	BorderColor    string `json:"border_color"`
	BorderStyle    string `json:"border_style"`
	CSS            string `json:"css"`
}

func DefaultConfig() Config {
	return Config{
		BorderWidthPX:  1,
		BorderRadiusPX: 24,
		BorderColor:    "rgba(255,255,255,0.16)",
		BorderStyle:    "solid",
		CSS:            "",
	}
}

func Presets() []card.LibraryItem {
	return Definition().Presets()
}

func typedPresets() []card.TypedPreset[Config] {
	return []card.TypedPreset[Config]{
		{ID: "seed-border-cyan-glow", Name: "Cyan Glow", Description: "Glowing cyan sci-fi border", Config: Config{
			BorderWidthPX:  1,
			BorderRadiusPX: 24,
			BorderColor:    "rgba(103, 232, 249, 0.7)",
			BorderStyle:    "solid",
			CSS:            "border: 1px solid rgba(103, 232, 249, 0.7); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
		}},
		{ID: "seed-border-brass-frame", Name: "Brass Frame", Description: "Old brass picture-frame border", Config: Config{
			BorderWidthPX:  3,
			BorderRadiusPX: 18,
			BorderColor:    "#b08d57",
			BorderStyle:    "double",
			CSS:            "border: 3px double #b08d57; box-shadow: inset 0 0 0 1px rgba(255,255,255,0.25);",
		}},
		{ID: "seed-border-ink-line", Name: "Ink Line", Description: "Fine black editorial border", Config: Config{
			BorderWidthPX:  1,
			BorderRadiusPX: 8,
			BorderColor:    "#111827",
			BorderStyle:    "solid",
			CSS:            "border: 1px solid #111827;",
		}},
	}
}

func RandomGenerated(seed int64, level int) schema.GeneratedConfig[Config] {
	options := []struct {
		description string
		part        Config
	}{
		{
			description: "A fine luminous cyan border.",
			part: Config{
				BorderWidthPX:  1,
				BorderRadiusPX: 24,
				BorderColor:    "rgba(103, 232, 249, 0.72)",
				CSS:            "border: 1px solid rgba(103, 232, 249, 0.72); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
			},
		},
		{
			description: "A brass double-line frame.",
			part: Config{
				BorderWidthPX:  3,
				BorderRadiusPX: 18,
				BorderColor:    "#b08d57",
				CSS:            "border: 3px double #b08d57; box-shadow: inset 0 0 0 1px rgba(255,255,255,0.25);",
			},
		},
		{
			description: "A crisp editorial ink border.",
			part: Config{
				BorderWidthPX:  1,
				BorderRadiusPX: 8,
				BorderColor:    "#111827",
				CSS:            "border: 1px solid #111827;",
			},
		},
		{
			description: "A soft pearl border with deep shadow.",
			part: Config{
				BorderWidthPX:  2,
				BorderRadiusPX: 32,
				BorderColor:    "rgba(255,255,255,0.64)",
				CSS:            "border: 2px solid rgba(255,255,255,0.64); box-shadow: 0 24px 70px rgba(15,23,42,0.42);",
			},
		},
		{
			description: "A compact slate border.",
			part: Config{
				BorderWidthPX:  2,
				BorderRadiusPX: 14,
				BorderColor:    "#64748b",
				CSS:            "border: 2px solid #64748b;",
			},
		},
	}
	if level > 2 {
		options = append(options, struct {
			description string
			part        Config
		}{
			description: "A strong arcade magenta border.",
			part: Config{
				BorderWidthPX:  4,
				BorderRadiusPX: 28,
				BorderColor:    "rgba(244,114,182,0.84)",
				CSS:            "border: 4px solid rgba(244,114,182,0.84); box-shadow: 0 0 28px rgba(244,114,182,0.26);",
			},
		})
	}
	pick := options[rand.New(rand.NewSource(seed)).Intn(len(options))]
	if pick.part.BorderStyle == "" {
		pick.part.BorderStyle = "solid"
	}
	return schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   pick.description,
		Config:        pick.part,
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Border", Structure: card.StructureLeaf,
		Default: DefaultConfig, Normalize: NormalizeConfig, Validate: ValidateConfig,
		ConfigRules: []schema.FieldRule{
			schema.IntegerRange("border_width_px", 0, 16),
			schema.IntegerRange("border_radius_px", 0, 64),
			schema.StringFormat("border_color", schema.FormatCSSColor),
			schema.Enum("border_style", AllowedStyles()...),
		},
		Render: func(_ card.Node, part Config, _ card.RenderContext) (card.Contribution, error) {
			styles := map[string]string{
				"border":        fmt.Sprintf("%dpx %s %s", part.BorderWidthPX, part.BorderStyle, part.BorderColor),
				"border-color":  part.BorderColor,
				"border-radius": fmt.Sprintf("%dpx", part.BorderRadiusPX),
				"border-style":  part.BorderStyle,
				"border-width":  fmt.Sprintf("%dpx", part.BorderWidthPX),
			}
			for property, value := range schema.CSSDeclarations(part.CSS, AllowedCSS()) {
				styles[property] = value
			}
			return card.Contribution{ShellStyle: styles}, nil
		},
		Controls: []card.Control[Config]{
			borderStringControl("border_color", "Color", "color", "border-color", func(c Config) string { return c.BorderColor }, func(c *Config, v string) { c.BorderColor = v }),
			borderIntControl("border_width_px", "Width", "border-width", func(c Config) int { return c.BorderWidthPX }, func(c *Config, v int) { c.BorderWidthPX = v }),
			borderIntControl("border_radius_px", "Radius", "border-radius", func(c Config) int { return c.BorderRadiusPX }, func(c *Config, v int) { c.BorderRadiusPX = v }),
			borderStringControl("border_style", "Line type", "select", "border-style", func(c Config) string { return c.BorderStyle }, func(c *Config, v string) { c.BorderStyle = v }),
		},
		Install: &card.InstallSpec{Policy: card.InstallReplaceKind}, Presets: typedPresets(),
		Generation: &card.TypedGenerationDefinition[Config]{SystemPrompt: systemPrompt, Example: exampleJSON, Random: RandomGenerated},
	})
}

func NormalizeConfig(config Config) Config {
	config.BorderColor = strings.TrimSpace(config.BorderColor)
	config.BorderStyle = strings.TrimSpace(config.BorderStyle)
	config.CSS = strings.TrimSpace(config.CSS)
	return config
}
func ValidateConfig(config Config) []schema.Issue {
	return schema.ValidateInlineCSS("config.css", config.CSS, AllowedCSS())
}

func AllowedCSS() map[string]struct{} {
	return map[string]struct{}{
		"border":        {},
		"border-color":  {},
		"border-image":  {},
		"border-radius": {},
		"border-style":  {},
		"border-width":  {},
		"box-shadow":    {},
	}
}

func AllowedStyles() []string {
	return []string{"solid", "dashed", "dotted", "double"}
}

func borderStringControl(id, label, kind, property string, get func(Config) string, set func(*Config, string)) card.Control[Config] {
	control := card.StringControl(id, "style", kind, label, property, get, set)
	base := control.Apply
	control.Apply = func(c *Config, raw json.RawMessage) error {
		if err := base(c, raw); err != nil {
			return err
		}
		syncBorderCSS(c)
		return nil
	}
	return control
}
func borderIntControl(id, label, property string, get func(Config) int, set func(*Config, int)) card.Control[Config] {
	control := card.IntControl(id, "style", "range", label, property, 1, get, set)
	base := control.Apply
	control.Apply = func(c *Config, raw json.RawMessage) error {
		if err := base(c, raw); err != nil {
			return err
		}
		syncBorderCSS(c)
		return nil
	}
	return control
}
func syncBorderCSS(config *Config) {
	declarations := schema.CSSDeclarations(config.CSS, AllowedCSS())
	declarations["border"] = fmt.Sprintf("%dpx %s %s", config.BorderWidthPX, config.BorderStyle, config.BorderColor)
	declarations["border-color"] = config.BorderColor
	declarations["border-radius"] = fmt.Sprintf("%dpx", config.BorderRadiusPX)
	declarations["border-style"] = config.BorderStyle
	declarations["border-width"] = fmt.Sprintf("%dpx", config.BorderWidthPX)
	config.CSS = cssString(declarations)
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
  "component_kind": "border",
  "description": "A soft translucent border with a large rounded radius.",
  "config": {
    "border_width_px": 1,
    "border_radius_px": 24,
    "border_color": "rgba(255,255,255,0.16)",
    "css": "box-shadow: 0 24px 70px rgba(15,23,42,0.42);"
  }
}`

const systemPrompt = `You generate safe declarative JSON configs for the border component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match this shape:
{
  "component_kind": "border",
  "description": "short human-readable summary",
  "config": {
    "border_width_px": 1,
    "border_radius_px": 24,
    "border_color": "rgba(255,255,255,0.16)",
    "css": "optional inline CSS declarations"
  }
}
Rules:
- component_kind must be "border".
- description is required.
- border_width_px must be within 0..16.
- border_radius_px must be within 0..64.
- border_color must be a safe color: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- css is optional inline declarations only.
- Allowed css properties: border, border-color, border-width, border-radius, border-image, box-shadow.
- Do not output url(...), javascript:, expression(...), @import, position, content, raw HTML, selectors, braces, or JavaScript.`
