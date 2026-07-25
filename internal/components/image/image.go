package imagecomponent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
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

func RandomGenerated(seed int64, level int) schema.GeneratedConfig[Config] {
	part := DefaultConfig()
	part.X = pickInt(seed, []int{34, 50, 66})
	part.Y = pickInt(seed+17, []int{32, 50, 68})
	if level > 2 {
		part.Rotation = pickInt(seed+29, []int{0, 8, 352})
	}
	return schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "A safe uploaded image layer.",
		Config:        part,
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Image", Structure: card.StructureLeaf, Default: DefaultConfig, Normalize: NormalizeConfig, Validate: ValidateConfig,
		ConfigRules: []schema.FieldRule{
			schema.IntegerRange("x", 0, 100),
			schema.IntegerRange("y", 0, 100),
			schema.IntegerRange("width", 8, 100),
			schema.IntegerRange("height", 8, 100),
			schema.IntegerRange("rotation", 0, 359),
			schema.StringFormat("border_color", schema.FormatCSSColor),
			schema.IntegerRange("border_width_px", 0, 12),
			schema.IntegerRange("border_radius_px", 0, 48),
		},
		Render: func(node card.Node, part Config, renderContext card.RenderContext) (card.Contribution, error) {
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)},
			}, nil
		},
		Controls: []card.Control[Config]{
			card.StringControl("alt", "content", "text", "Alt text", "alt", func(c Config) string { return c.Alt }, func(c *Config, v string) { c.Alt = v }), positionControl(),
			card.IntControl("x", "layout", "range", "X position", "left", 1, func(c Config) int { return c.X }, func(c *Config, v int) { c.X = v }),
			card.IntControl("y", "layout", "range", "Y position", "top", 1, func(c Config) int { return c.Y }, func(c *Config, v int) { c.Y = v }),
			card.IntControl("width", "layout", "range", "Width", "width", 1, func(c Config) int { return c.Width }, func(c *Config, v int) { c.Width = v }),
			card.IntControl("height", "layout", "range", "Height", "height", 1, func(c Config) int { return c.Height }, func(c *Config, v int) { c.Height = v }),
			card.IntControl("rotation", "layout", "range", "Rotation", "transform", 1, func(c Config) int { return c.Rotation }, func(c *Config, v int) { c.Rotation = v }),
			card.StringControl("border_color", "style", "color", "Border color", "border-color", func(c Config) string { return c.BorderColor }, func(c *Config, v string) { c.BorderColor = v }),
			card.IntControl("border_width_px", "style", "range", "Border width", "border-width", 1, func(c Config) int { return c.BorderWidthPX }, func(c *Config, v int) { c.BorderWidthPX = v }),
			card.IntControl("border_radius_px", "style", "range", "Border radius", "border-radius", 1, func(c Config) int { return c.BorderRadiusPX }, func(c *Config, v int) { c.BorderRadiusPX = v }),
		}, Install: &card.InstallSpec{Policy: card.InstallAppend}, Generation: &card.TypedGenerationDefinition[Config]{SystemPrompt: systemPrompt, Example: exampleJSON, Random: RandomGenerated},
	})
}

const exampleJSON = `{
  "component_kind": "image",
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
- component_kind must be "image".
- src must be a data URL for PNG, JPEG, WebP, or GIF.
- SVG, external URLs, HTML, and JavaScript are forbidden.
- x, y, width, and height are percentage values within the allowed ranges.`

func NormalizeConfig(config Config) Config {
	config.Src = strings.TrimSpace(config.Src)
	config.Alt = strings.TrimSpace(config.Alt)
	config.BorderColor = strings.TrimSpace(config.BorderColor)
	return config
}
func ValidateConfig(config Config) []schema.Issue {
	if issue := validateDataURL("config.src", config.Src); issue != nil {
		return []schema.Issue{*issue}
	}
	return nil
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

func validateDataURL(path, value string) *schema.Issue {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return &schema.Issue{Path: path, Code: "required", Message: "src is required"}
	}
	if len(trimmed) > maxDataURLBytes {
		return &schema.Issue{Path: path, Code: "too_large", Message: "src data URL is too large", Actual: len(trimmed)}
	}
	if strings.ContainsAny(trimmed, "<>") || strings.Contains(strings.ToLower(trimmed), "javascript:") {
		return &schema.Issue{Path: path, Code: "unsafe_value", Message: "src contains unsafe content"}
	}
	if !strings.HasPrefix(trimmed, "data:") {
		return &schema.Issue{Path: path, Code: "invalid_data_url", Message: "src must be an embedded image data URL", Actual: trimmed}
	}
	meta, payload, ok := strings.Cut(trimmed[len("data:"):], ",")
	if !ok || payload == "" {
		return &schema.Issue{Path: path, Code: "invalid_data_url", Message: "src must include data URL metadata and payload"}
	}
	metaParts := strings.Split(meta, ";")
	mimeType := strings.ToLower(strings.TrimSpace(metaParts[0]))
	if !contains(AllowedMIMETypes(), mimeType) {
		return &schema.Issue{
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
		return &schema.Issue{Path: path, Code: "invalid_data_url", Message: "src image data URL must be base64 encoded"}
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
		return &schema.Issue{Path: path, Code: "invalid_base64", Message: "src payload must be valid base64"}
	}
	return nil
}

func normalizedConfig(part Config) Config {
	return NormalizeConfig(part)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func positionControl() card.Control[Config] {
	type position struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	return card.Control[Config]{ID: "position", ValueSchema: positionControlSchema(), Descriptor: card.ControlDescriptor{Trait: "layout", Kind: "position", Label: "Position", Property: "position"}, Read: func(c Config) json.RawMessage { raw, _ := json.Marshal(position{c.X, c.Y}); return raw }, Apply: func(c *Config, raw json.RawMessage) error {
		var value position
		if err := card.DecodeControlObject(raw, &value); err != nil {
			return fmt.Errorf("value must include integer x and y: %w", err)
		}
		c.X, c.Y = value.X, value.Y
		return nil
	}}
}

func positionControlSchema() schema.ValueSchema {
	minimum, maximum := float64(0), float64(100)
	coordinate := schema.ValueSchema{Kind: schema.ValueInteger, Minimum: &minimum, Maximum: &maximum}
	return schema.ValueSchema{Kind: schema.ValueObject, Fields: []schema.FieldSchema{
		{JSONName: "x", Schema: coordinate, Required: true},
		{JSONName: "y", Schema: coordinate, Required: true},
	}}
}
