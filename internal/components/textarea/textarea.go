package textarea

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const Type = "textarea"

type Fragment struct {
	Content    string `json:"content"`
	FontFamily string `json:"font_family"`
	FontSizePX int    `json:"font_size_px"`
	FontWeight int    `json:"font_weight"`
	FontStyle  string `json:"font_style"`
	Color      string `json:"color"`
	Align      string `json:"align"`
	Position   string `json:"position"`
	CSS        string `json:"css"`
}

func DefaultFragment() Fragment {
	return Fragment{
		Content:    "Start designing this card.",
		FontFamily: "system",
		FontSizePX: 16,
		FontWeight: 400,
		FontStyle:  "normal",
		Color:      "#cbd5e1",
		Align:      "left",
		Position:   "center",
		CSS:        "",
	}
}

func Presets() []card.LibraryItem {
	return []card.LibraryItem{
		preset("seed-textarea-bold-statement", "Bold Statement", "Large centered display text", Fragment{
			Content:    "Signal Bloom",
			FontFamily: "display",
			FontSizePX: 42,
			FontWeight: 800,
			FontStyle:  "normal",
			Color:      "#f8fafc",
			Align:      "center",
			Position:   "center",
			CSS:        "font-size: 42px; font-weight: 800; text-align: center; letter-spacing: 0.04em; text-transform: uppercase;",
		}),
		preset("seed-textarea-elegant-serif", "Elegant Serif", "Refined serif text treatment", Fragment{
			Content:    "A quiet note from the edge of the map.",
			FontFamily: "serif",
			FontSizePX: 18,
			FontWeight: 400,
			FontStyle:  "italic",
			Color:      "#e2e8f0",
			Align:      "center",
			Position:   "center",
			CSS:        "font-family: Georgia, serif; font-style: italic; line-height: 1.5; text-align: center;",
		}),
		preset("seed-textarea-bottom-caption", "Bottom Caption", "Small readable note near the bottom", Fragment{
			Content:    "Generated fragments become reviewable card design data.",
			FontFamily: "system",
			FontSizePX: 15,
			FontWeight: 400,
			FontStyle:  "normal",
			Color:      "#cbd5e1",
			Align:      "center",
			Position:   "bottom-center",
			CSS:        "font-size: 15px; line-height: 1.45; text-align: center;",
		}),
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
		Contribute: func(node card.Node) (card.Contribution, error) {
			part, err := card.DecodeFragment[Fragment](node)
			if err != nil {
				return card.Contribution{}, err
			}
			generated := fragment.Generated[Fragment]{
				Target:      Type,
				Description: "Rendered textarea",
				Fragment:    part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid textarea fragment at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayer(node.ID, part)},
			}, nil
		},
	}
}

func RenderLayer(componentID string, part Fragment) *godom.Node {
	style := map[string]string{
		"color":         part.Color,
		"font-family":   fontFamilyCSS(part.FontFamily),
		"font-size":     fmt.Sprintf("%dpx", part.FontSizePX),
		"font-style":    part.FontStyle,
		"font-weight":   fmt.Sprintf("%d", part.FontWeight),
		"line-height":   "1.35",
		"max-width":     "82%",
		"overflow-wrap": "anywhere",
		"text-align":    part.Align,
		"white-space":   "pre-wrap",
		"z-index":       "1",
	}
	for property, value := range positionCSS(part.Position) {
		style[property] = value
	}
	for property, value := range fragment.CSSDeclarations(part.CSS, AllowedCSS()) {
		style[property] = value
	}
	return godom.Div(
		godom.Id(componentID+"-layer"),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-type", Type),
		godom.Attr("style", styleString(style)),
		godom.T(part.Content),
	)
}

func NormalizeGenerated(generated *fragment.Generated[Fragment]) {
	if generated == nil {
		return
	}
	generated.Target = strings.TrimSpace(generated.Target)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Fragment.Content = strings.TrimSpace(generated.Fragment.Content)
	generated.Fragment.FontFamily = strings.TrimSpace(generated.Fragment.FontFamily)
	generated.Fragment.FontStyle = strings.TrimSpace(generated.Fragment.FontStyle)
	generated.Fragment.Color = strings.TrimSpace(generated.Fragment.Color)
	generated.Fragment.Align = strings.TrimSpace(generated.Fragment.Align)
	generated.Fragment.Position = strings.TrimSpace(generated.Fragment.Position)
	generated.Fragment.CSS = strings.TrimSpace(generated.Fragment.CSS)
	generated.Fragment.FontSizePX = clamp(generated.Fragment.FontSizePX, 10, 72)
}

func ValidateGenerated(generated fragment.Generated[Fragment]) []fragment.Issue {
	var issues []fragment.Issue
	if strings.TrimSpace(generated.Fragment.Content) == "" {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.content",
			Code:    "required",
			Message: "content is required",
		})
	}
	if !contains(AllowedFontFamilies(), generated.Fragment.FontFamily) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.font_family",
			Code:    "invalid_value",
			Message: "font_family is not allowed",
			Actual:  generated.Fragment.FontFamily,
			Allowed: AllowedFontFamilies(),
		})
	}
	if generated.Fragment.FontSizePX < 10 || generated.Fragment.FontSizePX > 72 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.font_size_px",
			Code:    "out_of_range",
			Message: "font_size_px must be between 10 and 72",
			Actual:  generated.Fragment.FontSizePX,
		})
	}
	if !containsInt(AllowedWeights(), generated.Fragment.FontWeight) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.font_weight",
			Code:    "invalid_value",
			Message: "font_weight is not allowed",
			Actual:  generated.Fragment.FontWeight,
			Allowed: intsToStrings(AllowedWeights()),
		})
	}
	if !contains(AllowedFontStyles(), generated.Fragment.FontStyle) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.font_style",
			Code:    "invalid_value",
			Message: "font_style is not allowed",
			Actual:  generated.Fragment.FontStyle,
			Allowed: AllowedFontStyles(),
		})
	}
	if color := strings.TrimSpace(generated.Fragment.Color); color == "" {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.color",
			Code:    "required",
			Message: "color is required",
		})
	} else if !fragment.IsAllowedColor(color) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.color",
			Code:    "invalid_color",
			Message: "color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if !contains(AllowedAlignments(), generated.Fragment.Align) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.align",
			Code:    "invalid_value",
			Message: "align is not allowed",
			Actual:  generated.Fragment.Align,
			Allowed: AllowedAlignments(),
		})
	}
	if !contains(AllowedPositions(), generated.Fragment.Position) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.position",
			Code:    "invalid_value",
			Message: "position is not allowed",
			Actual:  generated.Fragment.Position,
			Allowed: AllowedPositions(),
		})
	}
	issues = append(issues, fragment.ValidateInlineCSS("fragment.css", generated.Fragment.CSS, AllowedCSS())...)
	return issues
}

func AllowedCSS() map[string]struct{} {
	return map[string]struct{}{
		"color":          {},
		"font-family":    {},
		"font-size":      {},
		"font-style":     {},
		"font-weight":    {},
		"letter-spacing": {},
		"line-height":    {},
		"text-align":     {},
		"text-shadow":    {},
		"text-transform": {},
	}
}

func AllowedFontFamilies() []string {
	return []string{"system", "serif", "mono", "display"}
}

func AllowedFontStyles() []string {
	return []string{"normal", "italic"}
}

func AllowedWeights() []int {
	return []int{400, 500, 600, 700, 800}
}

func AllowedAlignments() []string {
	return []string{"left", "center", "right"}
}

func AllowedPositions() []string {
	return []string{"top-left", "top-center", "center", "bottom-left", "bottom-center"}
}

func fontFamilyCSS(value string) string {
	switch value {
	case "serif":
		return "Georgia, ui-serif, serif"
	case "mono":
		return "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace"
	case "display":
		return "Trebuchet MS, ui-sans-serif, system-ui, sans-serif"
	default:
		return "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
	}
}

func positionCSS(value string) map[string]string {
	switch value {
	case "top-left":
		return map[string]string{"left": "1.5rem", "top": "1.5rem"}
	case "top-center":
		return map[string]string{"left": "50%", "top": "1.5rem", "transform": "translateX(-50%)"}
	case "bottom-left":
		return map[string]string{"bottom": "1.5rem", "left": "1.5rem"}
	case "bottom-center":
		return map[string]string{"bottom": "1.5rem", "left": "50%", "transform": "translateX(-50%)"}
	default:
		return map[string]string{"left": "50%", "top": "50%", "transform": "translate(-50%, -50%)", "width": "calc(100% - 3rem)"}
	}
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if target == value {
			return true
		}
	}
	return false
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if target == value {
			return true
		}
	}
	return false
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

func intsToStrings(values []int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("%d", value))
	}
	return out
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

const exampleJSON = `{
  "target": "textarea",
  "description": "A centered calm text treatment for the main card message.",
  "fragment": {
    "content": "Start designing this card.",
    "font_family": "system",
    "font_size_px": 16,
    "font_weight": 400,
    "font_style": "normal",
    "color": "#cbd5e1",
    "align": "left",
    "position": "center",
    "css": ""
  }
}`

const systemPrompt = `You generate safe declarative JSON fragments for the textarea component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match this shape:
{
  "target": "textarea",
  "description": "short human-readable summary",
  "fragment": {
    "content": "text shown on the card",
    "font_family": "system",
    "font_size_px": 16,
    "font_weight": 400,
    "font_style": "normal",
    "color": "#cbd5e1",
    "align": "left",
    "position": "center",
    "css": "optional inline CSS declarations"
  }
}
Rules:
- target must be "textarea".
- description and content are required.
- font_family must be one of: system, serif, mono, display.
- font_size_px is clamped to 10..72.
- font_weight must be one of: 400, 500, 600, 700, 800.
- font_style must be one of: normal, italic.
- color must be a safe color: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- align must be one of: left, center, right.
- position must be one of: top-left, top-center, center, bottom-left, bottom-center.
- css is optional inline text declarations only.
- Allowed css properties: font-family, font-size, font-weight, font-style, color, text-align, letter-spacing, line-height, text-transform, text-shadow.
- Do not output url(...), javascript:, expression(...), @import, position, content, raw HTML, selectors, braces, or JavaScript.`
