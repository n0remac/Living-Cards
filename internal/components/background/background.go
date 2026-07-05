package background

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const Type = "background"

type Fragment struct {
	BackgroundColor string `json:"background_color"`
	CSS             string `json:"css"`
}

func DefaultFragment() Fragment {
	return Fragment{
		BackgroundColor: "#111827",
		CSS:             "",
	}
}

func Presets() []card.LibraryItem {
	return []card.LibraryItem{
		preset("seed-background-night-sky", "Night Sky", "Deep blue night-sky background", Fragment{
			BackgroundColor: "#0f172a",
			CSS:             "background: radial-gradient(circle at top, #1e3a8a 0%, #0f172a 60%, #020617 100%);",
		}),
		preset("seed-background-parchment", "Parchment", "Warm parchment background", Fragment{
			BackgroundColor: "#f5e6c8",
			CSS:             "background: linear-gradient(135deg, #f8edd5 0%, #e7cfa6 100%);",
		}),
		preset("seed-background-mint", "Soft Mint", "Soft mint studio background", Fragment{
			BackgroundColor: "#d9f99d",
			CSS:             "background: linear-gradient(145deg, #ecfccb 0%, #bbf7d0 100%); box-shadow: inset 0 0 40px rgba(22, 101, 52, 0.12);",
		}),
	}
}

func RandomGenerated(seed int64, level int) fragment.Generated[Fragment] {
	options := []struct {
		description string
		part        Fragment
	}{
		{
			description: "A midnight card surface with a cool highlight.",
			part: Fragment{
				BackgroundColor: "#0f172a",
				CSS:             "background: radial-gradient(circle at top, rgba(56,189,248,0.24), transparent 44%), linear-gradient(160deg, #0f172a 0%, #111827 100%);",
			},
		},
		{
			description: "A warm parchment card surface.",
			part: Fragment{
				BackgroundColor: "#f5e6c8",
				CSS:             "background: linear-gradient(135deg, #f8edd5 0%, #e7cfa6 100%); box-shadow: inset 0 0 36px rgba(120, 53, 15, 0.12);",
			},
		},
		{
			description: "A soft mint card surface with gentle depth.",
			part: Fragment{
				BackgroundColor: "#d9f99d",
				CSS:             "background: linear-gradient(145deg, #ecfccb 0%, #bbf7d0 100%); box-shadow: inset 0 0 40px rgba(22, 101, 52, 0.12);",
			},
		},
		{
			description: "A rose dusk card surface.",
			part: Fragment{
				BackgroundColor: "#581c87",
				CSS:             "background: radial-gradient(circle at top right, rgba(244,114,182,0.28), transparent 42%), linear-gradient(155deg, #581c87 0%, #1f2937 100%);",
			},
		},
		{
			description: "A quiet stone card surface.",
			part: Fragment{
				BackgroundColor: "#374151",
				CSS:             "background: linear-gradient(150deg, #4b5563 0%, #1f2937 100%); box-shadow: inset 0 0 34px rgba(255,255,255,0.08);",
			},
		},
	}
	if level > 2 {
		options = append(options, struct {
			description string
			part        Fragment
		}{
			description: "A bright ember card surface.",
			part: Fragment{
				BackgroundColor: "#7c2d12",
				CSS:             "background: radial-gradient(circle at bottom, rgba(251,146,60,0.34), transparent 42%), linear-gradient(145deg, #7c2d12 0%, #111827 100%);",
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
				Description: "Rendered background",
				Fragment:    part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid background fragment at %s: %s", issues[0].Path, issues[0].Message)
			}
			styles := map[string]string{
				"background-color": part.BackgroundColor,
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
	generated.Fragment.BackgroundColor = strings.TrimSpace(generated.Fragment.BackgroundColor)
	generated.Fragment.CSS = strings.TrimSpace(generated.Fragment.CSS)
}

func ValidateGenerated(generated fragment.Generated[Fragment]) []fragment.Issue {
	var issues []fragment.Issue
	color := strings.TrimSpace(generated.Fragment.BackgroundColor)
	if color == "" {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.background_color",
			Code:    "required",
			Message: "background_color is required",
		})
	} else if !fragment.IsAllowedColor(color) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.background_color",
			Code:    "invalid_color",
			Message: "background_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	issues = append(issues, fragment.ValidateInlineCSS("fragment.css", generated.Fragment.CSS, AllowedCSS())...)
	return issues
}

func AllowedCSS() map[string]struct{} {
	return map[string]struct{}{
		"background":       {},
		"background-color": {},
		"box-shadow":       {},
	}
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
  "target": "background",
  "description": "A deep slate card background with a subtle radial highlight.",
  "fragment": {
    "background_color": "#111827",
    "css": "background: radial-gradient(circle at top, rgba(56,189,248,0.22), transparent 45%), #111827;"
  }
}`

const systemPrompt = `You generate safe declarative JSON fragments for the background component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match this shape:
{
  "target": "background",
  "description": "short human-readable summary",
  "fragment": {
    "background_color": "#111827",
    "css": "optional inline CSS declarations"
  }
}
Rules:
- target must be "background".
- description is required.
- background_color must be a safe color: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- css is optional inline declarations only.
- Allowed css properties: background, background-color, box-shadow.
- Do not output url(...), javascript:, expression(...), @import, position, content, raw HTML, selectors, braces, or JavaScript.`
