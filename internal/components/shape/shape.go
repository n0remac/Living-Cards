package shape

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = "shape"

type Config struct {
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

func DefaultConfig() Config {
	return Config{
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

func RandomGenerated(seed int64, level int) schema.GeneratedConfig[Config] {
	options := []struct {
		description string
		part        Config
	}{
		{
			description: "A warm circular shape layer.",
			part: Config{
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
			part: Config{
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
			part: Config{
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
			part        Config
		}{
			description: "A playful star shape layer.",
			part: Config{
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
	return schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   pick.description,
		Config:        pick.part,
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Shape", Structure: card.StructureLeaf, Default: DefaultConfig, Normalize: NormalizeConfig, Validate: ValidateConfig,
		Render: func(node card.Node, part Config, renderContext card.RenderContext) (card.Contribution, error) {
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)},
			}, nil
		},
		Controls: []card.Control[Config]{
			card.StringControl("shape", "shape", "select", "Shape", "shape", func(c Config) string { return c.Shape }, func(c *Config, v string) { c.Shape = v }, shapeOptions()...),
			positionControl(),
			card.IntControl("x", "layout", "range", "X position", "left", 0, 100, 1, func(c Config) int { return c.X }, func(c *Config, v int) { c.X = v }),
			card.IntControl("y", "layout", "range", "Y position", "top", 0, 100, 1, func(c Config) int { return c.Y }, func(c *Config, v int) { c.Y = v }),
			card.IntControl("width", "layout", "range", "Width", "width", 8, 100, 1, func(c Config) int { return c.Width }, func(c *Config, v int) { c.Width = v }),
			card.IntControl("height", "layout", "range", "Height", "height", 8, 100, 1, func(c Config) int { return c.Height }, func(c *Config, v int) { c.Height = v }),
			card.IntControl("rotation", "layout", "range", "Rotation", "transform", 0, 359, 1, func(c Config) int { return c.Rotation }, func(c *Config, v int) { c.Rotation = v }),
			card.StringControl("background_color", "style", "color", "Fill color", "fill", func(c Config) string { return c.BackgroundColor }, func(c *Config, v string) { c.BackgroundColor = v }),
			card.StringControl("border_color", "style", "color", "Border color", "stroke", func(c Config) string { return c.BorderColor }, func(c *Config, v string) { c.BorderColor = v }),
			card.IntControl("border_width_px", "style", "range", "Border width", "stroke-width", 0, 10, 1, func(c Config) int { return c.BorderWidthPX }, func(c *Config, v int) { c.BorderWidthPX = v }),
			card.StringControl("shadow", "style", "select", "Shadow", "filter", func(c Config) string { return c.Shadow }, func(c *Config, v string) { c.Shadow = v }, shadowOptions()...),
		}, Install: &card.InstallSpec{Policy: card.InstallAppend}, Generation: &card.TypedGenerationDefinition[Config]{Random: RandomGenerated},
	})
}

const exampleJSON = `{
  "component_kind": "shape",
  "description": "A warm circle shape layer.",
  "config": {
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

const systemPrompt = `You generate safe declarative JSON configs for one shape component of a card.
Return exactly one JSON object and no markdown, prose, HTML, selectors, braces, or JavaScript.
The JSON object must match the shape component schema.
Rules:
- component_kind must be "shape".
- shape must be one of circle, oval, rectangle, roundedRectangle, triangle, diamond, star, blob.
- colors must be safe colors: hex, rgb(...), rgba(...), hsl(...), or hsla(...).
- x, y, width, and height are percentage values within the allowed ranges.`

func NormalizeGenerated(generated *schema.GeneratedConfig[Config]) {
	if generated == nil {
		return
	}
	generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Config = NormalizeConfig(generated.Config)
}

func NormalizeConfig(config Config) Config {
	config.Shape = strings.TrimSpace(config.Shape)
	config.BackgroundColor = strings.TrimSpace(config.BackgroundColor)
	config.BorderColor = strings.TrimSpace(config.BorderColor)
	config.Shadow = strings.TrimSpace(config.Shadow)
	return config
}
func ValidateConfig(config Config) []schema.Issue {
	return ValidateGenerated(schema.GeneratedConfig[Config]{ComponentKind: Kind, Description: "Shape config", Config: config})
}

func ValidateGenerated(generated schema.GeneratedConfig[Config]) []schema.Issue {
	var issues []schema.Issue
	if !contains(AllowedShapes(), generated.Config.Shape) {
		issues = append(issues, schema.Issue{
			Path:    "config.shape",
			Code:    "invalid_value",
			Message: "shape is not allowed",
			Actual:  generated.Config.Shape,
			Allowed: AllowedShapes(),
		})
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
	if generated.Config.BorderWidthPX < 0 || generated.Config.BorderWidthPX > 10 {
		issues = append(issues, rangeIssue("config.border_width_px", "border_width_px", generated.Config.BorderWidthPX, 0, 10))
	}
	if color := strings.TrimSpace(generated.Config.BackgroundColor); color == "" {
		issues = append(issues, schema.Issue{
			Path:    "config.background_color",
			Code:    "required",
			Message: "background_color is required",
		})
	} else if !schema.IsAllowedColor(color) {
		issues = append(issues, schema.Issue{
			Path:    "config.background_color",
			Code:    "invalid_color",
			Message: "background_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if color := strings.TrimSpace(generated.Config.BorderColor); color == "" {
		issues = append(issues, schema.Issue{
			Path:    "config.border_color",
			Code:    "required",
			Message: "border_color is required",
		})
	} else if !schema.IsAllowedColor(color) {
		issues = append(issues, schema.Issue{
			Path:    "config.border_color",
			Code:    "invalid_color",
			Message: "border_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  color,
		})
	}
	if shadow := strings.TrimSpace(generated.Config.Shadow); shadow != "" && !contains(AllowedShadows(), shadow) {
		issues = append(issues, schema.Issue{
			Path:    "config.shadow",
			Code:    "invalid_value",
			Message: "shadow is not an allowed preset value",
			Actual:  shadow,
			Allowed: AllowedShadows(),
		})
	}
	return issues
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
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
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("style", styleString(style)),
		renderSVG(part),
	)
}

func renderSVG(part Config) *godom.Node {
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

func MarshalGenerated(generated schema.GeneratedConfig[Config]) (json.RawMessage, error) {
	NormalizeGenerated(&generated)
	if issues := ValidateGenerated(generated); len(issues) > 0 {
		return nil, fmt.Errorf("invalid shape config at %s: %s", issues[0].Path, issues[0].Message)
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

func rangeIssue(path, name string, value, min, max int) schema.Issue {
	return schema.Issue{
		Path:    path,
		Code:    "out_of_range",
		Message: fmt.Sprintf("%s must be between %d and %d", name, min, max),
		Actual:  value,
	}
}

func positionControl() card.Control[Config] {
	type position struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	return card.Control[Config]{ID: "position", Descriptor: card.ControlDescriptor{Trait: "layout", Kind: "position", Label: "Position", Property: "position"}, Read: func(c Config) json.RawMessage { raw, _ := json.Marshal(position{c.X, c.Y}); return raw }, Apply: func(c *Config, raw json.RawMessage) error {
		var value position
		if err := card.DecodeControlObject(raw, &value); err != nil {
			return fmt.Errorf("value must include integer x and y: %w", err)
		}
		c.X, c.Y = value.X, value.Y
		return nil
	}}
}
func shapeOptions() []card.ControlOption {
	labels := map[string]string{"circle": "Circle", "oval": "Oval", "rectangle": "Rectangle", "roundedRectangle": "Rounded rectangle", "triangle": "Triangle", "diamond": "Diamond", "star": "Star", "blob": "Blob"}
	out := make([]card.ControlOption, 0, len(AllowedShapes()))
	for _, value := range AllowedShapes() {
		out = append(out, card.Option(labels[value], value))
	}
	return out
}
func shadowOptions() []card.ControlOption {
	labels := []string{"None", "Soft slate", "Deep slate", "Rose glow", "Sky glow"}
	values := AllowedShadows()
	out := make([]card.ControlOption, 0, len(values))
	for index, value := range values {
		out = append(out, card.Option(labels[index], value))
	}
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if target == value {
			return true
		}
	}
	return false
}
