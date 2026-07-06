package imagecomponent

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
)

const (
	Kind            = "image"
	maxDataURLBytes = 512 * 1024
	defaultImageSrc = "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw=="
	defaultImageAlt = "Card image"
	defaultBorder   = "rgba(255,255,255,0.2)"
	defaultRadiusPX = 14
	defaultWidth    = 42
	defaultHeight   = 30
	defaultX        = 50
	defaultY        = 48
)

type Config struct {
	Src            string `json:"src"`
	Alt            string `json:"alt"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Rotation       int    `json:"rotation"`
	BorderColor    string `json:"border_color"`
	BorderWidthPX  int    `json:"border_width_px"`
	BorderRadiusPX int    `json:"border_radius_px"`
}

func DefaultConfig() Config {
	return Config{
		Src:            defaultImageSrc,
		Alt:            defaultImageAlt,
		X:              defaultX,
		Y:              defaultY,
		Width:          defaultWidth,
		Height:         defaultHeight,
		Rotation:       0,
		BorderColor:    defaultBorder,
		BorderWidthPX:  1,
		BorderRadiusPX: defaultRadiusPX,
	}
}

func RandomGenerated(seed int64, level int) design.GeneratedConfig[Config] {
	part := DefaultConfig()
	part.X = pickInt(seed, []int{34, 50, 66})
	part.Y = pickInt(seed+17, []int{32, 50, 68})
	if level > 2 {
		part.Rotation = pickInt(seed+29, []int{-8, 0, 8})
	}
	return design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "A safe uploaded image layer.",
		Config:        part,
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
				Description:   "Rendered image",
				Config:        part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid image config at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, generated.Config, renderContext)},
			}, nil
		},
	}
}

const exampleJSON = `{
  "componentKind": "image",
  "description": "A safe uploaded image layer.",
  "config": {
    "src": "data:image/png;base64,iVBORw0KGgo=",
    "alt": "Uploaded key image",
    "x": 50,
    "y": 48,
    "width": 42,
    "height": 30,
    "rotation": 0,
    "border_color": "rgba(255,255,255,0.2)",
    "border_width_px": 1,
    "border_radius_px": 14
  }
}`

const systemPrompt = `You generate safe declarative JSON configs for one image component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match the image component schema.
Rules:
- componentKind must be "image".
- src must be a data URL for PNG, JPEG, WebP, or GIF.
- SVG, external URLs, HTML, and JavaScript are forbidden.
- x, y, width, and height are percentage values within the allowed ranges.`

func NormalizeGenerated(generated *design.GeneratedConfig[Config]) {
	if generated == nil {
		return
	}
	defaults := DefaultConfig()
	generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Config.Src = strings.TrimSpace(generated.Config.Src)
	if generated.Config.Src == "" {
		generated.Config.Src = defaults.Src
	}
	generated.Config.Alt = strings.TrimSpace(generated.Config.Alt)
	if generated.Config.Alt == "" {
		generated.Config.Alt = defaults.Alt
	}
	generated.Config.BorderColor = strings.TrimSpace(generated.Config.BorderColor)
	if generated.Config.BorderColor == "" {
		generated.Config.BorderColor = defaults.BorderColor
	}
	if generated.Config.Width == 0 {
		generated.Config.Width = defaults.Width
	}
	if generated.Config.Height == 0 {
		generated.Config.Height = defaults.Height
	}
	if generated.Config.BorderRadiusPX == 0 {
		generated.Config.BorderRadiusPX = defaults.BorderRadiusPX
	}
	generated.Config.X = clamp(generated.Config.X, 0, 100)
	generated.Config.Y = clamp(generated.Config.Y, 0, 100)
	generated.Config.Width = clamp(generated.Config.Width, 8, 100)
	generated.Config.Height = clamp(generated.Config.Height, 8, 100)
	generated.Config.BorderWidthPX = clamp(generated.Config.BorderWidthPX, 0, 12)
	generated.Config.BorderRadiusPX = clamp(generated.Config.BorderRadiusPX, 0, 48)
	generated.Config.Rotation = normalizeRotation(generated.Config.Rotation)
}

func ValidateGenerated(generated design.GeneratedConfig[Config]) []design.Issue {
	var issues []design.Issue
	if issue := validateDataURL("config.src", generated.Config.Src); issue != nil {
		issues = append(issues, *issue)
	}
	if generated.Config.X < 0 || generated.Config.X > 100 {
		issues = append(issues, rangeIssue("config.x", "x", generated.Config.X, 0, 100))
	}
	if generated.Config.Y < 0 || generated.Config.Y > 100 {
		issues = append(issues, rangeIssue("config.y", "y", generated.Config.Y, 0, 100))
	}
	if generated.Config.Width < 8 || generated.Config.Width > 100 {
		issues = append(issues, rangeIssue("config.width", "width", generated.Config.Width, 8, 100))
	}
	if generated.Config.Height < 8 || generated.Config.Height > 100 {
		issues = append(issues, rangeIssue("config.height", "height", generated.Config.Height, 8, 100))
	}
	if generated.Config.BorderWidthPX < 0 || generated.Config.BorderWidthPX > 12 {
		issues = append(issues, rangeIssue("config.border_width_px", "border_width_px", generated.Config.BorderWidthPX, 0, 12))
	}
	if generated.Config.BorderRadiusPX < 0 || generated.Config.BorderRadiusPX > 48 {
		issues = append(issues, rangeIssue("config.border_radius_px", "border_radius_px", generated.Config.BorderRadiusPX, 0, 48))
	}
	if color := strings.TrimSpace(generated.Config.BorderColor); color == "" {
		issues = append(issues, design.Issue{Path: "config.border_color", Code: "required", Message: "border_color is required"})
	} else if !design.IsAllowedColor(color) {
		issues = append(issues, design.Issue{
			Path:    "config.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	return issues
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = normalizedConfig(part)
	style := map[string]string{
		"border":           fmt.Sprintf("%dpx solid %s", part.BorderWidthPX, part.BorderColor),
		"border-radius":    fmt.Sprintf("%dpx", part.BorderRadiusPX),
		"height":           fmt.Sprintf("%d%%", part.Height),
		"left":             fmt.Sprintf("%d%%", part.X),
		"overflow":         "hidden",
		"pointer-events":   "auto",
		"top":              fmt.Sprintf("%d%%", part.Y),
		"transform":        fmt.Sprintf("translate(-50%%, -50%%) rotate(%ddeg)", part.Rotation),
		"transform-origin": "center",
		"width":            fmt.Sprintf("%d%%", part.Width),
		"z-index":          "1",
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute bg-black/10"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("style", styleString(style)),
		godom.Img(
			godom.Src(part.Src),
			godom.Alt(part.Alt),
			godom.Class("block h-full w-full object-cover"),
			godom.Attr("draggable", "false"),
		),
	)
}

func AllowedMIMETypes() []string {
	return []string{"image/png", "image/jpeg", "image/webp", "image/gif"}
}

func MaxDataURLBytes() int {
	return maxDataURLBytes
}

func validateDataURL(path, value string) *design.Issue {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return &design.Issue{Path: path, Code: "required", Message: "src is required"}
	}
	if len(trimmed) > maxDataURLBytes {
		return &design.Issue{Path: path, Code: "too_large", Message: "src data URL is too large", Actual: len(trimmed)}
	}
	if strings.ContainsAny(trimmed, "<>") || strings.Contains(strings.ToLower(trimmed), "javascript:") {
		return &design.Issue{Path: path, Code: "unsafe_value", Message: "src contains unsafe content"}
	}
	if !strings.HasPrefix(trimmed, "data:") {
		return &design.Issue{Path: path, Code: "invalid_data_url", Message: "src must be an embedded image data URL", Actual: trimmed}
	}
	meta, payload, ok := strings.Cut(trimmed[len("data:"):], ",")
	if !ok || payload == "" {
		return &design.Issue{Path: path, Code: "invalid_data_url", Message: "src must include data URL metadata and payload"}
	}
	metaParts := strings.Split(meta, ";")
	mimeType := strings.ToLower(strings.TrimSpace(metaParts[0]))
	if !contains(AllowedMIMETypes(), mimeType) {
		return &design.Issue{
			Path:    path,
			Code:    "invalid_mime_type",
			Message: "src must be a PNG, JPEG, WebP, or GIF data URL",
			Actual:  mimeType,
			Allowed: AllowedMIMETypes(),
		}
	}
	base64Encoded := false
	for _, part := range metaParts[1:] {
		if strings.EqualFold(strings.TrimSpace(part), "base64") {
			base64Encoded = true
			break
		}
	}
	if !base64Encoded {
		return &design.Issue{Path: path, Code: "invalid_data_url", Message: "src image data URL must be base64 encoded"}
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		return &design.Issue{Path: path, Code: "invalid_base64", Message: "src payload must be valid base64"}
	}
	return nil
}

func normalizedConfig(part Config) Config {
	generated := design.GeneratedConfig[Config]{ComponentKind: Kind, Description: "Current image", Config: part}
	NormalizeGenerated(&generated)
	return generated.Config
}

func rangeIssue(path, field string, actual, min, max int) design.Issue {
	return design.Issue{
		Path:    path,
		Code:    "out_of_range",
		Message: fmt.Sprintf("%s must be between %d and %d", field, min, max),
		Actual:  actual,
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func normalizeRotation(rotation int) int {
	rotation = rotation % 360
	if rotation < 0 {
		rotation += 360
	}
	return rotation
}

func pickInt(seed int64, values []int) int {
	if len(values) == 0 {
		return 0
	}
	return values[rand.New(rand.NewSource(seed)).Intn(len(values))]
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

func DataURLForTesting() string {
	return defaultImageSrc
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
