package textarea

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const Type = "textarea"

type Fragment struct {
	Content         string `json:"content"`
	FontFamily      string `json:"font_family"`
	FontSizePX      int    `json:"font_size_px"`
	FontWeight      int    `json:"font_weight"`
	FontStyle       string `json:"font_style"`
	Color           string `json:"color"`
	Align           string `json:"align"`
	Position        string `json:"position"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	BackgroundColor string `json:"background_color"`
	BorderColor     string `json:"border_color"`
	BorderWidthPX   int    `json:"border_width_px"`
	BorderRadiusPX  int    `json:"border_radius_px"`
	PaddingPX       int    `json:"padding_px"`
	CSS             string `json:"css"`
}

func DefaultFragment() Fragment {
	return Fragment{
		Content:         "Start designing this card.",
		FontFamily:      "system",
		FontSizePX:      16,
		FontWeight:      400,
		FontStyle:       "normal",
		Color:           "#cbd5e1",
		Align:           "left",
		Position:        "center",
		X:               50,
		Y:               50,
		BackgroundColor: "rgba(255,255,255,0)",
		BorderColor:     "rgba(255,255,255,0)",
		BorderWidthPX:   0,
		BorderRadiusPX:  12,
		PaddingPX:       0,
		CSS:             "",
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
			X:          50,
			Y:          50,
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
			X:          50,
			Y:          50,
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
			X:          50,
			Y:          86,
			CSS:        "font-size: 15px; line-height: 1.45; text-align: center;",
		}),
	}
}

func RandomGenerated(seed int64, level int) fragment.Generated[Fragment] {
	options := []struct {
		description string
		part        Fragment
	}{
		{
			description: "A compact centered title treatment.",
			part: Fragment{
				Content:         "Living Card",
				FontFamily:      "display",
				FontSizePX:      34,
				FontWeight:      800,
				FontStyle:       "normal",
				Color:           "#f8fafc",
				Align:           "center",
				Position:        "center",
				X:               50,
				Y:               50,
				BackgroundColor: "rgba(15,23,42,0.24)",
				BorderColor:     "rgba(255,255,255,0.16)",
				BorderWidthPX:   1,
				BorderRadiusPX:  16,
				PaddingPX:       14,
				CSS:             "text-align: center;",
			},
		},
		{
			description: "A small bottom caption with quiet contrast.",
			part: Fragment{
				Content:         "Tap, tune, and lock in the design.",
				FontFamily:      "system",
				FontSizePX:      15,
				FontWeight:      500,
				FontStyle:       "normal",
				Color:           "#e2e8f0",
				Align:           "center",
				Position:        "bottom-center",
				X:               50,
				Y:               86,
				BackgroundColor: "rgba(17,24,39,0.42)",
				BorderColor:     "rgba(148,163,184,0.22)",
				BorderWidthPX:   1,
				BorderRadiusPX:  14,
				PaddingPX:       12,
				CSS:             "line-height: 1.4; text-align: center;",
			},
		},
		{
			description: "A refined serif note in the center.",
			part: Fragment{
				Content:         "A small surface for deliberate edits.",
				FontFamily:      "serif",
				FontSizePX:      19,
				FontWeight:      400,
				FontStyle:       "italic",
				Color:           "#1f2937",
				Align:           "center",
				Position:        "center",
				X:               50,
				Y:               50,
				BackgroundColor: "rgba(248,250,252,0.72)",
				BorderColor:     "rgba(31,41,55,0.18)",
				BorderWidthPX:   1,
				BorderRadiusPX:  18,
				PaddingPX:       16,
				CSS:             "font-family: Georgia, serif; line-height: 1.5; text-align: center;",
			},
		},
	}
	if level > 2 {
		options = append(options, struct {
			description string
			part        Fragment
		}{
			description: "A bold editorial text panel.",
			part: Fragment{
				Content:         "DETERMINISTIC",
				FontFamily:      "mono",
				FontSizePX:      24,
				FontWeight:      700,
				FontStyle:       "normal",
				Color:           "#111827",
				Align:           "center",
				Position:        "top-center",
				X:               50,
				Y:               14,
				BackgroundColor: "#f8fafc",
				BorderColor:     "#111827",
				BorderWidthPX:   2,
				BorderRadiusPX:  10,
				PaddingPX:       12,
				CSS:             "letter-spacing: 0.04em; text-align: center; text-transform: uppercase;",
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
		Contribute: func(node card.Node, renderContext card.RenderContext) (card.Contribution, error) {
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
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, generated.Fragment, renderContext)},
			}, nil
		},
	}
}

func RenderLayer(componentID string, part Fragment) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Fragment, renderContext card.RenderContext) *godom.Node {
	part = normalizedFragment(part)
	style := map[string]string{
		"color":         part.Color,
		"font-family":   fontFamilyCSS(part.FontFamily),
		"font-size":     fmt.Sprintf("%dpx", part.FontSizePX),
		"font-style":    part.FontStyle,
		"font-weight":   fmt.Sprintf("%d", part.FontWeight),
		"line-height":   "1.35",
		"max-width":     "82%",
		"overflow-wrap": "anywhere",
		"padding":       fmt.Sprintf("%dpx", part.PaddingPX),
		"text-align":    part.Align,
		"left":          fmt.Sprintf("%d%%", part.X),
		"top":           fmt.Sprintf("%d%%", part.Y),
		"transform":     "translate(-50%, -50%)",
		"white-space":   "pre-wrap",
		"width":         "calc(100% - 3rem)",
		"z-index":       "1",
	}
	if strings.TrimSpace(part.BackgroundColor) != "" {
		style["background-color"] = part.BackgroundColor
	}
	if part.BorderWidthPX > 0 {
		style["border"] = fmt.Sprintf("%dpx solid %s", part.BorderWidthPX, part.BorderColor)
		style["border-color"] = part.BorderColor
		style["border-width"] = fmt.Sprintf("%dpx", part.BorderWidthPX)
	}
	style["border-radius"] = fmt.Sprintf("%dpx", part.BorderRadiusPX)
	for property, value := range fragment.CSSDeclarations(part.CSS, AllowedCSS()) {
		style[property] = value
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
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
	generated.Fragment.BackgroundColor = strings.TrimSpace(generated.Fragment.BackgroundColor)
	generated.Fragment.BorderColor = strings.TrimSpace(generated.Fragment.BorderColor)
	generated.Fragment.CSS = strings.TrimSpace(generated.Fragment.CSS)
	defaults := DefaultFragment()
	if generated.Fragment.FontFamily == "" {
		generated.Fragment.FontFamily = defaults.FontFamily
	}
	if generated.Fragment.FontSizePX == 0 {
		generated.Fragment.FontSizePX = defaults.FontSizePX
	}
	if generated.Fragment.FontWeight == 0 {
		generated.Fragment.FontWeight = defaults.FontWeight
	}
	if generated.Fragment.FontStyle == "" {
		generated.Fragment.FontStyle = defaults.FontStyle
	}
	if generated.Fragment.Color == "" {
		generated.Fragment.Color = defaults.Color
	}
	if generated.Fragment.Align == "" {
		generated.Fragment.Align = defaults.Align
	}
	if generated.Fragment.Position == "" {
		generated.Fragment.Position = defaults.Position
	}
	if generated.Fragment.X == 0 && generated.Fragment.Y == 0 {
		generated.Fragment.X, generated.Fragment.Y = positionDefaults(generated.Fragment.Position)
	}
	if generated.Fragment.BackgroundColor == "" {
		generated.Fragment.BackgroundColor = defaults.BackgroundColor
	}
	if generated.Fragment.BorderColor == "" {
		generated.Fragment.BorderColor = defaults.BorderColor
	}
	if generated.Fragment.BorderRadiusPX == 0 {
		generated.Fragment.BorderRadiusPX = defaults.BorderRadiusPX
	}
	generated.Fragment.FontSizePX = clamp(generated.Fragment.FontSizePX, 10, 72)
	generated.Fragment.X = clamp(generated.Fragment.X, 0, 100)
	generated.Fragment.Y = clamp(generated.Fragment.Y, 0, 100)
	generated.Fragment.BorderWidthPX = clamp(generated.Fragment.BorderWidthPX, 0, 12)
	generated.Fragment.BorderRadiusPX = clamp(generated.Fragment.BorderRadiusPX, 0, 40)
	generated.Fragment.PaddingPX = clamp(generated.Fragment.PaddingPX, 0, 32)
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
	if generated.Fragment.X < 0 || generated.Fragment.X > 100 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.x",
			Code:    "out_of_range",
			Message: "x must be between 0 and 100",
			Actual:  generated.Fragment.X,
		})
	}
	if generated.Fragment.Y < 0 || generated.Fragment.Y > 100 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.y",
			Code:    "out_of_range",
			Message: "y must be between 0 and 100",
			Actual:  generated.Fragment.Y,
		})
	}
	if color := strings.TrimSpace(generated.Fragment.BackgroundColor); color != "" && !fragment.IsAllowedColor(color) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.background_color",
			Code:    "invalid_color",
			Message: "background_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if color := strings.TrimSpace(generated.Fragment.BorderColor); color != "" && !fragment.IsAllowedColor(color) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if generated.Fragment.BorderWidthPX < 0 || generated.Fragment.BorderWidthPX > 12 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_width_px",
			Code:    "out_of_range",
			Message: "border_width_px must be between 0 and 12",
			Actual:  generated.Fragment.BorderWidthPX,
		})
	}
	if generated.Fragment.BorderRadiusPX < 0 || generated.Fragment.BorderRadiusPX > 40 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_radius_px",
			Code:    "out_of_range",
			Message: "border_radius_px must be between 0 and 40",
			Actual:  generated.Fragment.BorderRadiusPX,
		})
	}
	if generated.Fragment.PaddingPX < 0 || generated.Fragment.PaddingPX > 32 {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.padding_px",
			Code:    "out_of_range",
			Message: "padding_px must be between 0 and 32",
			Actual:  generated.Fragment.PaddingPX,
		})
	}
	issues = append(issues, fragment.ValidateInlineCSS("fragment.css", generated.Fragment.CSS, AllowedCSS())...)
	return issues
}

func AllowedCSS() map[string]struct{} {
	return map[string]struct{}{
		"background-color": {},
		"border":           {},
		"border-color":     {},
		"border-radius":    {},
		"border-width":     {},
		"box-shadow":       {},
		"color":            {},
		"font-family":      {},
		"font-size":        {},
		"font-style":       {},
		"font-weight":      {},
		"letter-spacing":   {},
		"line-height":      {},
		"padding":          {},
		"text-align":       {},
		"text-shadow":      {},
		"text-transform":   {},
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

func normalizedFragment(part Fragment) Fragment {
	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Rendered textarea",
		Fragment:    part,
	}
	NormalizeGenerated(&generated)
	return generated.Fragment
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

func positionDefaults(value string) (int, int) {
	switch value {
	case "top-left":
		return 12, 14
	case "top-center":
		return 50, 14
	case "bottom-left":
		return 12, 86
	case "bottom-center":
		return 50, 86
	default:
		return 50, 50
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
