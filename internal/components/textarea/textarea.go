package textarea

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
)

const Kind = "textarea"

type Config struct {
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

func DefaultConfig() Config {
	return Config{
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
		preset("seed-textarea-bold-statement", "Bold Statement", "Large centered display text", Config{
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
		preset("seed-textarea-elegant-serif", "Elegant Serif", "Refined serif text treatment", Config{
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
		preset("seed-textarea-bottom-caption", "Bottom Caption", "Small readable note near the bottom", Config{
			Content:    "Generated configs become reviewable card design data.",
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

func RandomGenerated(seed int64, level int) design.GeneratedConfig[Config] {
	options := []struct {
		description string
		part        Config
	}{
		{
			description: "A compact centered title treatment.",
			part: Config{
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
			part: Config{
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
			part: Config{
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
			part        Config
		}{
			description: "A bold editorial text panel.",
			part: Config{
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
		Contribute: func(node card.Node, renderContext card.RenderContext) (card.Contribution, error) {
			part, err := card.DecodeConfig[Config](node)
			if err != nil {
				return card.Contribution{}, err
			}
			generated := design.GeneratedConfig[Config]{
				ComponentKind: Kind,
				Description:   "Rendered textarea",
				Config:        part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid textarea config at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, generated.Config, renderContext)},
			}, nil
		},
	}
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = normalizedConfig(part)
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
	for property, value := range design.CSSDeclarations(part.CSS, AllowedCSS()) {
		style[property] = value
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("style", styleString(style)),
		godom.T(part.Content),
	)
}

func NormalizeGenerated(generated *design.GeneratedConfig[Config]) {
	if generated == nil {
		return
	}
	generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Config.Content = strings.TrimSpace(generated.Config.Content)
	generated.Config.FontFamily = strings.TrimSpace(generated.Config.FontFamily)
	generated.Config.FontStyle = strings.TrimSpace(generated.Config.FontStyle)
	generated.Config.Color = strings.TrimSpace(generated.Config.Color)
	generated.Config.Align = strings.TrimSpace(generated.Config.Align)
	generated.Config.Position = strings.TrimSpace(generated.Config.Position)
	generated.Config.BackgroundColor = strings.TrimSpace(generated.Config.BackgroundColor)
	generated.Config.BorderColor = strings.TrimSpace(generated.Config.BorderColor)
	generated.Config.CSS = strings.TrimSpace(generated.Config.CSS)
	defaults := DefaultConfig()
	if generated.Config.FontFamily == "" {
		generated.Config.FontFamily = defaults.FontFamily
	}
	if generated.Config.FontSizePX == 0 {
		generated.Config.FontSizePX = defaults.FontSizePX
	}
	if generated.Config.FontWeight == 0 {
		generated.Config.FontWeight = defaults.FontWeight
	}
	if generated.Config.FontStyle == "" {
		generated.Config.FontStyle = defaults.FontStyle
	}
	if generated.Config.Color == "" {
		generated.Config.Color = defaults.Color
	}
	if generated.Config.Align == "" {
		generated.Config.Align = defaults.Align
	}
	if generated.Config.Position == "" {
		generated.Config.Position = defaults.Position
	}
	if generated.Config.X == 0 && generated.Config.Y == 0 {
		generated.Config.X, generated.Config.Y = positionDefaults(generated.Config.Position)
	}
	if generated.Config.BackgroundColor == "" {
		generated.Config.BackgroundColor = defaults.BackgroundColor
	}
	if generated.Config.BorderColor == "" {
		generated.Config.BorderColor = defaults.BorderColor
	}
	if generated.Config.BorderRadiusPX == 0 {
		generated.Config.BorderRadiusPX = defaults.BorderRadiusPX
	}
	generated.Config.FontSizePX = clamp(generated.Config.FontSizePX, 10, 72)
	generated.Config.X = clamp(generated.Config.X, 0, 100)
	generated.Config.Y = clamp(generated.Config.Y, 0, 100)
	generated.Config.BorderWidthPX = clamp(generated.Config.BorderWidthPX, 0, 12)
	generated.Config.BorderRadiusPX = clamp(generated.Config.BorderRadiusPX, 0, 40)
	generated.Config.PaddingPX = clamp(generated.Config.PaddingPX, 0, 32)
}

func ValidateGenerated(generated design.GeneratedConfig[Config]) []design.Issue {
	var issues []design.Issue
	if strings.TrimSpace(generated.Config.Content) == "" {
		issues = append(issues, design.Issue{
			Path:    "config.content",
			Code:    "required",
			Message: "content is required",
		})
	}
	if !contains(AllowedFontFamilies(), generated.Config.FontFamily) {
		issues = append(issues, design.Issue{
			Path:    "config.font_family",
			Code:    "invalid_value",
			Message: "font_family is not allowed",
			Actual:  generated.Config.FontFamily,
			Allowed: AllowedFontFamilies(),
		})
	}
	if generated.Config.FontSizePX < 10 || generated.Config.FontSizePX > 72 {
		issues = append(issues, design.Issue{
			Path:    "config.font_size_px",
			Code:    "out_of_range",
			Message: "font_size_px must be between 10 and 72",
			Actual:  generated.Config.FontSizePX,
		})
	}
	if !containsInt(AllowedWeights(), generated.Config.FontWeight) {
		issues = append(issues, design.Issue{
			Path:    "config.font_weight",
			Code:    "invalid_value",
			Message: "font_weight is not allowed",
			Actual:  generated.Config.FontWeight,
			Allowed: intsToStrings(AllowedWeights()),
		})
	}
	if !contains(AllowedFontStyles(), generated.Config.FontStyle) {
		issues = append(issues, design.Issue{
			Path:    "config.font_style",
			Code:    "invalid_value",
			Message: "font_style is not allowed",
			Actual:  generated.Config.FontStyle,
			Allowed: AllowedFontStyles(),
		})
	}
	if color := strings.TrimSpace(generated.Config.Color); color == "" {
		issues = append(issues, design.Issue{
			Path:    "config.color",
			Code:    "required",
			Message: "color is required",
		})
	} else if !design.IsAllowedColor(color) {
		issues = append(issues, design.Issue{
			Path:    "config.color",
			Code:    "invalid_color",
			Message: "color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if !contains(AllowedAlignments(), generated.Config.Align) {
		issues = append(issues, design.Issue{
			Path:    "config.align",
			Code:    "invalid_value",
			Message: "align is not allowed",
			Actual:  generated.Config.Align,
			Allowed: AllowedAlignments(),
		})
	}
	if !contains(AllowedPositions(), generated.Config.Position) {
		issues = append(issues, design.Issue{
			Path:    "config.position",
			Code:    "invalid_value",
			Message: "position is not allowed",
			Actual:  generated.Config.Position,
			Allowed: AllowedPositions(),
		})
	}
	if generated.Config.X < 0 || generated.Config.X > 100 {
		issues = append(issues, design.Issue{
			Path:    "config.x",
			Code:    "out_of_range",
			Message: "x must be between 0 and 100",
			Actual:  generated.Config.X,
		})
	}
	if generated.Config.Y < 0 || generated.Config.Y > 100 {
		issues = append(issues, design.Issue{
			Path:    "config.y",
			Code:    "out_of_range",
			Message: "y must be between 0 and 100",
			Actual:  generated.Config.Y,
		})
	}
	if color := strings.TrimSpace(generated.Config.BackgroundColor); color != "" && !design.IsAllowedColor(color) {
		issues = append(issues, design.Issue{
			Path:    "config.background_color",
			Code:    "invalid_color",
			Message: "background_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if color := strings.TrimSpace(generated.Config.BorderColor); color != "" && !design.IsAllowedColor(color) {
		issues = append(issues, design.Issue{
			Path:    "config.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if generated.Config.BorderWidthPX < 0 || generated.Config.BorderWidthPX > 12 {
		issues = append(issues, design.Issue{
			Path:    "config.border_width_px",
			Code:    "out_of_range",
			Message: "border_width_px must be between 0 and 12",
			Actual:  generated.Config.BorderWidthPX,
		})
	}
	if generated.Config.BorderRadiusPX < 0 || generated.Config.BorderRadiusPX > 40 {
		issues = append(issues, design.Issue{
			Path:    "config.border_radius_px",
			Code:    "out_of_range",
			Message: "border_radius_px must be between 0 and 40",
			Actual:  generated.Config.BorderRadiusPX,
		})
	}
	if generated.Config.PaddingPX < 0 || generated.Config.PaddingPX > 32 {
		issues = append(issues, design.Issue{
			Path:    "config.padding_px",
			Code:    "out_of_range",
			Message: "padding_px must be between 0 and 32",
			Actual:  generated.Config.PaddingPX,
		})
	}
	issues = append(issues, design.ValidateInlineCSS("config.css", generated.Config.CSS, AllowedCSS())...)
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

func normalizedConfig(part Config) Config {
	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Rendered textarea",
		Config:        part,
	}
	NormalizeGenerated(&generated)
	return generated.Config
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
  "componentKind": "textarea",
  "description": "A centered calm text treatment for the main card message.",
  "config": {
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

const systemPrompt = `You generate safe declarative JSON configs for the textarea component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match this shape:
{
  "componentKind": "textarea",
  "description": "short human-readable summary",
  "config": {
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
- componentKind must be "textarea".
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
