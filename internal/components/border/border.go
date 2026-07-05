package border

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const Type = "border"

type Fragment struct {
	BorderWidthPX  int    `json:"border_width_px"`
	BorderRadiusPX int    `json:"border_radius_px"`
	BorderColor    string `json:"border_color"`
	CSS            string `json:"css"`
}

func DefaultFragment() Fragment {
	return Fragment{
		BorderWidthPX:  1,
		BorderRadiusPX: 24,
		BorderColor:    "rgba(255,255,255,0.16)",
		CSS:            "",
	}
}

func Presets() []card.LibraryItem {
	return []card.LibraryItem{
		preset("seed-border-cyan-glow", "Cyan Glow", "Glowing cyan sci-fi border", Fragment{
			BorderWidthPX:  1,
			BorderRadiusPX: 24,
			BorderColor:    "rgba(103, 232, 249, 0.7)",
			CSS:            "border: 1px solid rgba(103, 232, 249, 0.7); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
		}),
		preset("seed-border-brass-frame", "Brass Frame", "Old brass picture-frame border", Fragment{
			BorderWidthPX:  3,
			BorderRadiusPX: 18,
			BorderColor:    "#b08d57",
			CSS:            "border: 3px double #b08d57; box-shadow: inset 0 0 0 1px rgba(255,255,255,0.25);",
		}),
		preset("seed-border-ink-line", "Ink Line", "Fine black editorial border", Fragment{
			BorderWidthPX:  1,
			BorderRadiusPX: 8,
			BorderColor:    "#111827",
			CSS:            "border: 1px solid #111827;",
		}),
	}
}

func RandomGenerated(seed int64, level int) fragment.Generated[Fragment] {
	options := []struct {
		description string
		part        Fragment
	}{
		{
			description: "A fine luminous cyan border.",
			part: Fragment{
				BorderWidthPX:  1,
				BorderRadiusPX: 24,
				BorderColor:    "rgba(103, 232, 249, 0.72)",
				CSS:            "border: 1px solid rgba(103, 232, 249, 0.72); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
			},
		},
		{
			description: "A brass double-line frame.",
			part: Fragment{
				BorderWidthPX:  3,
				BorderRadiusPX: 18,
				BorderColor:    "#b08d57",
				CSS:            "border: 3px double #b08d57; box-shadow: inset 0 0 0 1px rgba(255,255,255,0.25);",
			},
		},
		{
			description: "A crisp editorial ink border.",
			part: Fragment{
				BorderWidthPX:  1,
				BorderRadiusPX: 8,
				BorderColor:    "#111827",
				CSS:            "border: 1px solid #111827;",
			},
		},
		{
			description: "A soft pearl border with deep shadow.",
			part: Fragment{
				BorderWidthPX:  2,
				BorderRadiusPX: 32,
				BorderColor:    "rgba(255,255,255,0.64)",
				CSS:            "border: 2px solid rgba(255,255,255,0.64); box-shadow: 0 24px 70px rgba(15,23,42,0.42);",
			},
		},
		{
			description: "A compact slate border.",
			part: Fragment{
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
			part        Fragment
		}{
			description: "A strong arcade magenta border.",
			part: Fragment{
				BorderWidthPX:  4,
				BorderRadiusPX: 28,
				BorderColor:    "rgba(244,114,182,0.84)",
				CSS:            "border: 4px solid rgba(244,114,182,0.84); box-shadow: 0 0 28px rgba(244,114,182,0.26);",
			},
		})
	}
	pick := options[rand.New(rand.NewSource(seed)).Intn(len(options))]
	return fragment.Generated[Fragment]{
		Target:      Type,
		Description: pick.description,
		Fragment:    pick.part,
	}
}

func Spec() fragment.Spec[Fragment] {
	return fragment.Spec[Fragment]{
		Target:       Type,
		SystemPrompt: systemPrompt,
		Example:      exampleJSON,
		Normalize:    NormalizeGenerated,
		Validate:     ValidateGenerated,
	}
}

func Definition() card.Definition {
	return card.Definition{
		Type: Type,
		Contribute: func(node card.Node, _ card.RenderContext) (card.Contribution, error) {
			part, err := card.DecodeFragment[Fragment](node)
			if err != nil {
				return card.Contribution{}, err
			}
			generated := fragment.Generated[Fragment]{
				Target:      Type,
				Description: "Rendered border",
				Fragment:    part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid border fragment at %s: %s", issues[0].Path, issues[0].Message)
			}
			styles := map[string]string{
				"border":        fmt.Sprintf("%dpx solid %s", part.BorderWidthPX, part.BorderColor),
				"border-color":  part.BorderColor,
				"border-radius": fmt.Sprintf("%dpx", part.BorderRadiusPX),
				"border-width":  fmt.Sprintf("%dpx", part.BorderWidthPX),
			}
			for property, value := range fragment.CSSDeclarations(part.CSS, AllowedCSS()) {
				styles[property] = value
			}
			return card.Contribution{ShellStyle: styles}, nil
		},
	}
}

func NormalizeGenerated(generated *fragment.Generated[Fragment]) {
	if generated == nil {
		return
	}
	generated.Target = strings.TrimSpace(generated.Target)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Fragment.BorderColor = strings.TrimSpace(generated.Fragment.BorderColor)
	generated.Fragment.CSS = strings.TrimSpace(generated.Fragment.CSS)
	generated.Fragment.BorderWidthPX = clamp(generated.Fragment.BorderWidthPX, 0, 16)
	generated.Fragment.BorderRadiusPX = clamp(generated.Fragment.BorderRadiusPX, 0, 64)
}

func ValidateGenerated(generated fragment.Generated[Fragment]) []fragment.Issue {
	var issues []fragment.Issue
	if generated.Fragment.BorderWidthPX < 0 || generated.Fragment.BorderWidthPX > 16 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_width_px",
			Code:    "out_of_range",
			Message: "border_width_px must be between 0 and 16",
			Actual:  generated.Fragment.BorderWidthPX,
		})
	}
	if generated.Fragment.BorderRadiusPX < 0 || generated.Fragment.BorderRadiusPX > 64 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_radius_px",
			Code:    "out_of_range",
			Message: "border_radius_px must be between 0 and 64",
			Actual:  generated.Fragment.BorderRadiusPX,
		})
	}
	color := strings.TrimSpace(generated.Fragment.BorderColor)
	if color == "" {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_color",
			Code:    "required",
			Message: "border_color is required",
		})
	} else if !fragment.IsAllowedColor(color) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	issues = append(issues, fragment.ValidateInlineCSS("fragment.css", generated.Fragment.CSS, AllowedCSS())...)
	return issues
}

func AllowedCSS() map[string]struct{} {
	return map[string]struct{}{
		"border":        {},
		"border-color":  {},
		"border-image":  {},
		"border-radius": {},
		"border-width":  {},
		"box-shadow":    {},
	}
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

func preset(id, name, description string, part Fragment) card.LibraryItem {
	raw, err := json.Marshal(part)
	if err != nil {
		panic(err)
	}
	return card.LibraryItem{
		ID:          id,
		Name:        name,
		Target:      Type,
		Description: description,
		Fragment:    raw,
	}
}

const exampleJSON = `{
  "target": "border",
  "description": "A soft translucent border with a large rounded radius.",
  "fragment": {
    "border_width_px": 1,
    "border_radius_px": 24,
    "border_color": "rgba(255,255,255,0.16)",
    "css": "box-shadow: 0 24px 70px rgba(15,23,42,0.42);"
  }
}`

const systemPrompt = `You generate safe declarative JSON fragments for the border component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match this shape:
{
  "target": "border",
  "description": "short human-readable summary",
  "fragment": {
    "border_width_px": 1,
    "border_radius_px": 24,
    "border_color": "rgba(255,255,255,0.16)",
    "css": "optional inline CSS declarations"
  }
}
Rules:
- target must be "border".
- description is required.
- border_width_px is clamped to 0..16.
- border_radius_px is clamped to 0..64.
- border_color must be a safe color: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- css is optional inline declarations only.
- Allowed css properties: border, border-color, border-width, border-radius, border-image, box-shadow.
- Do not output url(...), javascript:, expression(...), @import, position, content, raw HTML, selectors, braces, or JavaScript.`
