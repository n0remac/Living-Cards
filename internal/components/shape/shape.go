package shape

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

const Type = "shape"

type Fragment struct {
	Shape           string `json:"shape"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Rotation        int    `json:"rotation"`
	BackgroundColor string `json:"background_color"`
	BorderColor     string `json:"border_color"`
	BorderWidthPX   int    `json:"border_width_px"`
	Shadow          string `json:"shadow"`
}

func DefaultFragment() Fragment {
	return Fragment{
		Shape:           "circle",
		X:               34,
		Y:               26,
		Width:           32,
		Height:          24,
		Rotation:        0,
		BackgroundColor: "#f6d365",
		BorderColor:     "#111827",
		BorderWidthPX:   2,
		Shadow:          "",
	}
}

func RandomGenerated(seed int64, level int) fragment.Generated[Fragment] {
	options := []struct {
		description string
		part        Fragment
	}{
		{
			description: "A warm circular shape layer.",
			part: Fragment{
				Shape:           "circle",
				X:               38,
				Y:               24,
				Width:           26,
				Height:          26,
				Rotation:        0,
				BackgroundColor: "#f6d365",
				BorderColor:     "#111827",
				BorderWidthPX:   2,
				Shadow:          "0 12px 28px rgba(15,23,42,0.28)",
			},
		},
		{
			description: "A crisp diamond shape layer.",
			part: Fragment{
				Shape:           "diamond",
				X:               35,
				Y:               34,
				Width:           30,
				Height:          24,
				Rotation:        0,
				BackgroundColor: "#38bdf8",
				BorderColor:     "rgba(15,23,42,0.8)",
				BorderWidthPX:   2,
				Shadow:          "0 10px 24px rgba(14,165,233,0.22)",
			},
		},
		{
			description: "A quiet rounded rectangle shape layer.",
			part: Fragment{
				Shape:           "roundedRectangle",
				X:               22,
				Y:               58,
				Width:           56,
				Height:          14,
				Rotation:        0,
				BackgroundColor: "rgba(248,250,252,0.72)",
				BorderColor:     "rgba(17,24,39,0.28)",
				BorderWidthPX:   1,
				Shadow:          "",
			},
		},
	}
	if level > 2 {
		options = append(options, struct {
			description string
			part        Fragment
		}{
			description: "A playful star shape layer.",
			part: Fragment{
				Shape:           "star",
				X:               54,
				Y:               16,
				Width:           22,
				Height:          22,
				Rotation:        12,
				BackgroundColor: "#f43f5e",
				BorderColor:     "#f8fafc",
				BorderWidthPX:   2,
				Shadow:          "0 12px 26px rgba(244,63,94,0.24)",
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
		Contribute: func(node card.Node) (card.Contribution, error) {
			part, err := card.DecodeFragment[Fragment](node)
			if err != nil {
				return card.Contribution{}, err
			}
			generated := fragment.Generated[Fragment]{
				Target:      Type,
				Description: "Rendered shape",
				Fragment:    part,
			}
			NormalizeGenerated(&generated)
			if issues := ValidateGenerated(generated); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid shape fragment at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayer(node.ID, generated.Fragment)},
			}, nil
		},
	}
}

const exampleJSON = `{
  "target": "shape",
  "description": "A warm circle shape layer.",
  "fragment": {
    "shape": "circle",
    "x": 34,
    "y": 26,
    "width": 32,
    "height": 24,
    "rotation": 0,
    "background_color": "#f6d365",
    "border_color": "#111827",
    "border_width_px": 2,
    "shadow": ""
  }
}`

const systemPrompt = `You generate safe declarative JSON fragments for one shape component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match the shape component schema.
Rules:
- target must be "shape".
- shape must be one of circle, oval, rectangle, roundedRectangle, triangle, diamond, star, blob.
- colors must be safe colors: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- x, y, width, and height are percentage values within the allowed ranges.`

func NormalizeGenerated(generated *fragment.Generated[Fragment]) {
	if generated == nil {
		return
	}
	defaults := DefaultFragment()
	generated.Target = strings.TrimSpace(generated.Target)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Fragment.Shape = strings.TrimSpace(generated.Fragment.Shape)
	if generated.Fragment.Shape == "" {
		generated.Fragment.Shape = defaults.Shape
	}
	generated.Fragment.BackgroundColor = strings.TrimSpace(generated.Fragment.BackgroundColor)
	if generated.Fragment.BackgroundColor == "" {
		generated.Fragment.BackgroundColor = defaults.BackgroundColor
	}
	generated.Fragment.BorderColor = strings.TrimSpace(generated.Fragment.BorderColor)
	if generated.Fragment.BorderColor == "" {
		generated.Fragment.BorderColor = defaults.BorderColor
	}
	generated.Fragment.Shadow = strings.TrimSpace(generated.Fragment.Shadow)
	if generated.Fragment.Width == 0 {
		generated.Fragment.Width = defaults.Width
	}
	if generated.Fragment.Height == 0 {
		generated.Fragment.Height = defaults.Height
	}
	generated.Fragment.X = clamp(generated.Fragment.X, 0, 100)
	generated.Fragment.Y = clamp(generated.Fragment.Y, 0, 100)
	generated.Fragment.Width = clamp(generated.Fragment.Width, 8, 100)
	generated.Fragment.Height = clamp(generated.Fragment.Height, 8, 100)
	generated.Fragment.BorderWidthPX = clamp(generated.Fragment.BorderWidthPX, 0, 10)
	generated.Fragment.Rotation = normalizeRotation(generated.Fragment.Rotation)
}

func ValidateGenerated(generated fragment.Generated[Fragment]) []fragment.Issue {
	var issues []fragment.Issue
	if !contains(AllowedShapes(), generated.Fragment.Shape) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.shape",
			Code:    "invalid_value",
			Message: "shape is not allowed",
			Actual:  generated.Fragment.Shape,
			Allowed: AllowedShapes(),
		})
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
	if generated.Fragment.BorderWidthPX < 0 || generated.Fragment.BorderWidthPX > 10 {
		issues = append(issues, rangeIssue("fragment.border_width_px", "border_width_px", generated.Fragment.BorderWidthPX, 0, 10))
	}
	if color := strings.TrimSpace(generated.Fragment.BackgroundColor); color == "" {
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
	if color := strings.TrimSpace(generated.Fragment.BorderColor); color == "" {
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
	if shadow := strings.TrimSpace(generated.Fragment.Shadow); shadow != "" && !contains(AllowedShadows(), shadow) {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.shadow",
			Code:    "invalid_value",
			Message: "shadow is not an allowed preset value",
			Actual:  shadow,
			Allowed: AllowedShadows(),
		})
	}
	return issues
}

func RenderLayer(componentID string, part Fragment) *godom.Node {
	style := map[string]string{
		"height":           fmt.Sprintf("%d%%", part.Height),
		"left":             fmt.Sprintf("%d%%", part.X),
		"pointer-events":   "auto",
		"top":              fmt.Sprintf("%d%%", part.Y),
		"transform":        fmt.Sprintf("rotate(%ddeg)", part.Rotation),
		"transform-origin": "center",
		"width":            fmt.Sprintf("%d%%", part.Width),
		"z-index":          "0",
	}
	if strings.TrimSpace(part.Shadow) != "" {
		style["filter"] = "drop-shadow(" + part.Shadow + ")"
	}
	return godom.Div(
		godom.Id(componentID+"-layer"),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-type", Type),
		godom.Attr("style", styleString(style)),
		renderSVG(part),
	)
}

func renderSVG(part Fragment) *godom.Node {
	shapeAttrs := []*godom.Node{
		godom.Attr("fill", part.BackgroundColor),
		godom.Attr("stroke", part.BorderColor),
		godom.Attr("stroke-width", fmt.Sprintf("%d", part.BorderWidthPX)),
		godom.Attr("vector-effect", "non-scaling-stroke"),
	}
	var shapeNode *godom.Node
	switch part.Shape {
	case "oval":
		shapeNode = godom.NewNode("ellipse", append([]*godom.Node{
			godom.Attr("cx", "50"),
			godom.Attr("cy", "50"),
			godom.Attr("rx", "44"),
			godom.Attr("ry", "34"),
		}, shapeAttrs...))
	case "rectangle":
		shapeNode = godom.Rect(append([]*godom.Node{
			godom.X("8"),
			godom.Y("12"),
			godom.Width("84"),
			godom.Height("76"),
		}, shapeAttrs...)...)
	case "roundedRectangle":
		shapeNode = godom.Rect(append([]*godom.Node{
			godom.X("8"),
			godom.Y("16"),
			godom.Width("84"),
			godom.Height("68"),
			godom.Rx("16"),
			godom.Ry("16"),
		}, shapeAttrs...)...)
	case "triangle":
		shapeNode = godom.Polygon(append([]*godom.Node{godom.Points("50,8 92,88 8,88")}, shapeAttrs...)...)
	case "diamond":
		shapeNode = godom.Polygon(append([]*godom.Node{godom.Points("50,6 94,50 50,94 6,50")}, shapeAttrs...)...)
	case "star":
		shapeNode = godom.Polygon(append([]*godom.Node{godom.Points("50,6 61,36 94,36 67,56 78,90 50,70 22,90 33,56 6,36 39,36")}, shapeAttrs...)...)
	case "blob":
		shapeNode = godom.Path(append([]*godom.Node{godom.Attr("d", "M55 8 C76 8 92 23 90 45 C88 68 70 91 46 88 C22 85 7 66 10 43 C13 21 32 8 55 8 Z")}, shapeAttrs...)...)
	default:
		shapeNode = godom.Circle(append([]*godom.Node{
			godom.Cx("50"),
			godom.Cy("50"),
			godom.R("40"),
		}, shapeAttrs...)...)
	}
	return godom.Svg(
		godom.Attr("viewBox", "0 0 100 100"),
		godom.Attr("aria-hidden", "true"),
		godom.Attr("focusable", "false"),
		godom.Attr("width", "100%"),
		godom.Attr("height", "100%"),
		shapeNode,
	)
}

func AllowedShapes() []string {
	return []string{"circle", "oval", "rectangle", "roundedRectangle", "triangle", "diamond", "star", "blob"}
}

func AllowedShadows() []string {
	return []string{
		"",
		"0 10px 24px rgba(15,23,42,0.22)",
		"0 12px 28px rgba(15,23,42,0.28)",
		"0 12px 26px rgba(244,63,94,0.24)",
		"0 10px 24px rgba(14,165,233,0.22)",
	}
}

func MarshalGenerated(generated fragment.Generated[Fragment]) (json.RawMessage, error) {
	NormalizeGenerated(&generated)
	if issues := ValidateGenerated(generated); len(issues) > 0 {
		return nil, fmt.Errorf("invalid shape fragment at %s: %s", issues[0].Path, issues[0].Message)
	}
	return json.Marshal(generated)
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

func rangeIssue(path, name string, value, min, max int) fragment.Issue {
	return fragment.Issue{
		Path:    path,
		Code:    "out_of_range",
		Message: fmt.Sprintf("%s must be between %d and %d", name, min, max),
		Actual:  value,
	}
}

func normalizeRotation(value int) int {
	value = value % 360
	if value < 0 {
		value += 360
	}
	return value
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if target == value {
			return true
		}
	}
	return false
}
