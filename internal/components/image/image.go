package imagecomponent

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const (
	Type            = "image"
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

type Fragment struct {
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

func DefaultFragment() Fragment {
	return Fragment{
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

func RandomGenerated(seed int64, level int) fragment.Generated[Fragment] {
	part := DefaultFragment()
	part.X = pickInt(seed, []int{34, 50, 66})
	part.Y = pickInt(seed+17, []int{32, 50, 68})
	if level > 2 {
		part.Rotation = pickInt(seed+29, []int{-8, 0, 8})
	}
	return fragment.Generated[Fragment]{
		Target:      Type,
		Description: "A safe uploaded image layer.",
		Fragment:    part,
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
				Description: "Rendered image",
				Fragment:    part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid image fragment at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, generated.Fragment, renderContext)},
			}, nil
		},
	}
}

const exampleJSON = `{
  "target": "image",
  "description": "A safe uploaded image layer.",
  "fragment": {
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

const systemPrompt = `You generate safe declarative JSON fragments for one image component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match the image component schema.
Rules:
- target must be "image".
- src must be a data URL for PNG, JPEG, WebP, or GIF.
- SVG, external URLs, HTML, and JavaScript are forbidden.
- x, y, width, and height are percentage values within the allowed ranges.`

func NormalizeGenerated(generated *fragment.Generated[Fragment]) {
	if generated == nil {
		return
	}
	defaults := DefaultFragment()
	generated.Target = strings.TrimSpace(generated.Target)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Fragment.Src = strings.TrimSpace(generated.Fragment.Src)
	if generated.Fragment.Src == "" {
		generated.Fragment.Src = defaults.Src
	}
	generated.Fragment.Alt = strings.TrimSpace(generated.Fragment.Alt)
	if generated.Fragment.Alt == "" {
		generated.Fragment.Alt = defaults.Alt
	}
	generated.Fragment.BorderColor = strings.TrimSpace(generated.Fragment.BorderColor)
	if generated.Fragment.BorderColor == "" {
		generated.Fragment.BorderColor = defaults.BorderColor
	}
	if generated.Fragment.Width == 0 {
		generated.Fragment.Width = defaults.Width
	}
	if generated.Fragment.Height == 0 {
		generated.Fragment.Height = defaults.Height
	}
	if generated.Fragment.BorderRadiusPX == 0 {
		generated.Fragment.BorderRadiusPX = defaults.BorderRadiusPX
	}
	generated.Fragment.X = clamp(generated.Fragment.X, 0, 100)
	generated.Fragment.Y = clamp(generated.Fragment.Y, 0, 100)
	generated.Fragment.Width = clamp(generated.Fragment.Width, 8, 100)
	generated.Fragment.Height = clamp(generated.Fragment.Height, 8, 100)
	generated.Fragment.BorderWidthPX = clamp(generated.Fragment.BorderWidthPX, 0, 12)
	generated.Fragment.BorderRadiusPX = clamp(generated.Fragment.BorderRadiusPX, 0, 48)
	generated.Fragment.Rotation = normalizeRotation(generated.Fragment.Rotation)
}

func ValidateGenerated(generated fragment.Generated[Fragment]) []fragment.Issue {
	var issues []fragment.Issue
	if issue := validateDataURL("fragment.src", generated.Fragment.Src); issue != nil {
		issues = append(issues, *issue)
	}
	if generated.Fragment.X < 0 || generated.Fragment.X > 100 {
		issues = append(issues, rangeIssue("fragment.x", "x", generated.Fragment.X, 0, 100))
	}
	if generated.Fragment.Y < 0 || generated.Fragment.Y > 100 {
		issues = append(issues, rangeIssue("fragment.y", "y", generated.Fragment.Y, 0, 100))
	}
	if generated.Fragment.Width < 8 || generated.Fragment.Width > 100 {
		issues = append(issues, rangeIssue("fragment.width", "width", generated.Fragment.Width, 8, 100))
	}
	if generated.Fragment.Height < 8 || generated.Fragment.Height > 100 {
		issues = append(issues, rangeIssue("fragment.height", "height", generated.Fragment.Height, 8, 100))
	}
	if generated.Fragment.BorderWidthPX < 0 || generated.Fragment.BorderWidthPX > 12 {
		issues = append(issues, rangeIssue("fragment.border_width_px", "border_width_px", generated.Fragment.BorderWidthPX, 0, 12))
	}
	if generated.Fragment.BorderRadiusPX < 0 || generated.Fragment.BorderRadiusPX > 48 {
		issues = append(issues, rangeIssue("fragment.border_radius_px", "border_radius_px", generated.Fragment.BorderRadiusPX, 0, 48))
	}
	if color := strings.TrimSpace(generated.Fragment.BorderColor); color == "" {
		issues = append(issues, fragment.Issue{Path: "fragment.border_color", Code: "required", Message: "border_color is required"})
	} else if !fragment.IsAllowedColor(color) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	return issues
}

func RenderLayer(componentID string, part Fragment) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Fragment, renderContext card.RenderContext) *godom.Node {
	part = normalizedFragment(part)
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
		godom.Attr("data-component-type", Type),
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

func validateDataURL(path, value string) *fragment.Issue {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return &fragment.Issue{Path: path, Code: "required", Message: "src is required"}
	}
	if len(trimmed) > maxDataURLBytes {
		return &fragment.Issue{Path: path, Code: "too_large", Message: "src data URL is too large", Actual: len(trimmed)}
	}
	if strings.ContainsAny(trimmed, "<>") || strings.Contains(strings.ToLower(trimmed), "javascript:") {
		return &fragment.Issue{Path: path, Code: "unsafe_value", Message: "src contains unsafe content"}
	}
	if !strings.HasPrefix(trimmed, "data:") {
		return &fragment.Issue{Path: path, Code: "invalid_data_url", Message: "src must be an embedded image data URL", Actual: trimmed}
	}
	meta, payload, ok := strings.Cut(trimmed[len("data:"):], ",")
	if !ok || payload == "" {
		return &fragment.Issue{Path: path, Code: "invalid_data_url", Message: "src must include data URL metadata and payload"}
	}
	metaParts := strings.Split(meta, ";")
	mimeType := strings.ToLower(strings.TrimSpace(metaParts[0]))
	if !contains(AllowedMIMETypes(), mimeType) {
		return &fragment.Issue{
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
		return &fragment.Issue{Path: path, Code: "invalid_data_url", Message: "src image data URL must be base64 encoded"}
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		return &fragment.Issue{Path: path, Code: "invalid_base64", Message: "src payload must be valid base64"}
	}
	return nil
}

func normalizedFragment(part Fragment) Fragment {
	generated := fragment.Generated[Fragment]{Target: Type, Description: "Current image", Fragment: part}
	NormalizeGenerated(&generated)
	return generated.Fragment
}

func rangeIssue(path, field string, actual, min, max int) fragment.Issue {
	return fragment.Issue{
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
