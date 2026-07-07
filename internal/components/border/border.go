package border

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
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
	return []card.LibraryItem{
		preset("seed-border-cyan-glow", "Cyan Glow", "Glowing cyan sci-fi border", Config{
			BorderWidthPX:  1,
			BorderRadiusPX: 24,
			BorderColor:    "rgba(103, 232, 249, 0.7)",
			CSS:            "border: 1px solid rgba(103, 232, 249, 0.7); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
		}),
		preset("seed-border-brass-frame", "Brass Frame", "Old brass picture-frame border", Config{
			BorderWidthPX:  3,
			BorderRadiusPX: 18,
			BorderColor:    "#b08d57",
			CSS:            "border: 3px double #b08d57; box-shadow: inset 0 0 0 1px rgba(255,255,255,0.25);",
		}),
		preset("seed-border-ink-line", "Ink Line", "Fine black editorial border", Config{
			BorderWidthPX:  1,
			BorderRadiusPX: 8,
			BorderColor:    "#111827",
			CSS:            "border: 1px solid #111827;",
		}),
	}
}

func RandomGenerated(seed int64, level int) design.GeneratedConfig[Config] {
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
	return design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   pick.description,
		Config:        pick.part,
	}
}

func Spec() design.Spec[Config] {
	return design.Spec[Config]{
		ComponentKind: Kind,
		SystemPrompt:  systemPrompt,
		Example:       exampleJSON,
		Normalize:     NormalizeGenerated,
		Validate:      ValidateGenerated,
	}
}

func Definition() card.Definition {
	return card.Definition{
		ComponentKind: Kind,
		Contribute: func(node card.Node, _ card.RenderContext) (card.Contribution, error) {
			part, err := card.DecodeConfig[Config](node)
			if err != nil {
				return card.Contribution{}, err
			}
			generated := design.GeneratedConfig[Config]{
				ComponentKind: Kind,
				Description:   "Rendered border",
				Config:        part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid border config at %s: %s", issues[0].Path, issues[0].Message)
			}
			part = generated.Config
			styles := map[string]string{
				"border":        fmt.Sprintf("%dpx %s %s", part.BorderWidthPX, part.BorderStyle, part.BorderColor),
				"border-color":  part.BorderColor,
				"border-radius": fmt.Sprintf("%dpx", part.BorderRadiusPX),
				"border-style":  part.BorderStyle,
				"border-width":  fmt.Sprintf("%dpx", part.BorderWidthPX),
			}
			for property, value := range design.CSSDeclarations(part.CSS, AllowedCSS()) {
				styles[property] = value
			}
			return card.Contribution{ShellStyle: styles}, nil
		},
	}
}

func NormalizeGenerated(generated *design.GeneratedConfig[Config]) {
	if generated == nil {
		return
	}
	generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Config.BorderColor = strings.TrimSpace(generated.Config.BorderColor)
	generated.Config.BorderStyle = normalizeBorderStyle(generated.Config.BorderStyle)
	generated.Config.CSS = strings.TrimSpace(generated.Config.CSS)
	generated.Config.BorderWidthPX = clamp(generated.Config.BorderWidthPX, 0, 16)
	generated.Config.BorderRadiusPX = clamp(generated.Config.BorderRadiusPX, 0, 64)
}

func ValidateGenerated(generated design.GeneratedConfig[Config]) []design.Issue {
	var issues []design.Issue
	generated.Config.BorderStyle = normalizeBorderStyle(generated.Config.BorderStyle)
	if generated.Config.BorderWidthPX < 0 || generated.Config.BorderWidthPX > 16 {
		issues = append(issues, design.Issue{
			Path:    "config.border_width_px",
			Code:    "out_of_range",
			Message: "border_width_px must be between 0 and 16",
			Actual:  generated.Config.BorderWidthPX,
		})
	}
	if generated.Config.BorderRadiusPX < 0 || generated.Config.BorderRadiusPX > 64 {
		issues = append(issues, design.Issue{
			Path:    "config.border_radius_px",
			Code:    "out_of_range",
			Message: "border_radius_px must be between 0 and 64",
			Actual:  generated.Config.BorderRadiusPX,
		})
	}
	color := strings.TrimSpace(generated.Config.BorderColor)
	if color == "" {
		issues = append(issues, design.Issue{
			Path:    "config.border_color",
			Code:    "required",
			Message: "border_color is required",
		})
	} else if !design.IsAllowedColor(color) {
		issues = append(issues, design.Issue{
			Path:    "config.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if !borderStyleAllowed(generated.Config.BorderStyle) {
		issues = append(issues, design.Issue{
			Path:    "config.border_style",
			Code:    "invalid_option",
			Message: "border_style must be solid, dashed, dotted, or double",
			Actual:  generated.Config.BorderStyle,
			Allowed: []string{"solid", "dashed", "dotted", "double"},
		})
	}
	issues = append(issues, design.ValidateInlineCSS("config.css", generated.Config.CSS, AllowedCSS())...)
	return issues
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

func normalizeBorderStyle(style string) string {
	style = strings.ToLower(strings.TrimSpace(style))
	if style == "" {
		return "solid"
	}
	return style
}

func borderStyleAllowed(style string) bool {
	for _, candidate := range AllowedStyles() {
		if style == candidate {
			return true
		}
	}
	return false
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func preset(id, name, description string, part Config) card.LibraryItem {
	raw, err := json.Marshal(part)
	if err != nil {
		panic(err)
	}
	return card.LibraryItem{
		ID:            id,
		Name:          name,
		ComponentKind: Kind,
		Description:   description,
		Config:        raw,
	}
}

const exampleJSON = `{
  "componentKind": "border",
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
  "componentKind": "border",
  "description": "short human-readable summary",
  "config": {
    "border_width_px": 1,
    "border_radius_px": 24,
    "border_color": "rgba(255,255,255,0.16)",
    "css": "optional inline CSS declarations"
  }
}
Rules:
- componentKind must be "border".
- description is required.
- border_width_px is clamped to 0..16.
- border_radius_px is clamped to 0..64.
- border_color must be a safe color: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- css is optional inline declarations only.
- Allowed css properties: border, border-color, border-width, border-radius, border-image, box-shadow.
- Do not output url(...), javascript:, expression(...), @import, position, content, raw HTML, selectors, braces, or JavaScript.`
