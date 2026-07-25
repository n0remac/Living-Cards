package button

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = "button"

type Config struct {
	FormID string `json:"form_id"`
	Label  string `json:"label"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
}

func DefaultConfig() Config {
	return Config{FormID: "controller-form", Label: "Submit", X: 50, Y: 72, Width: 44}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Button", Structure: card.StructureLeaf, Default: DefaultConfig, Normalize: NormalizeConfig,
		ConfigRules: []schema.FieldRule{
			schema.StringFormat("form_id", schema.FormatSafeToken),
			schema.StringLength("label", 1, 80),
			schema.IntegerRange("x", 0, 100),
			schema.IntegerRange("y", 0, 100),
			schema.IntegerRange("width", 12, 100),
		},
		Render: func(node card.Node, part Config, renderContext card.RenderContext) (card.Contribution, error) {
			return card.Contribution{Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)}}, nil
		},
		Controls: []card.Control[Config]{
			card.StringControl("form_id", "form", "text", "Form ID", "form", func(c Config) string { return c.FormID }, func(c *Config, v string) { c.FormID = v }),
			card.StringControl("label", "content", "text", "Label", "text-content", func(c Config) string { return c.Label }, func(c *Config, v string) { c.Label = v }),
			positionControl(),
			card.IntControl("x", "layout", "range", "X position", "left", 1, func(c Config) int { return c.X }, func(c *Config, v int) { c.X = v }),
			card.IntControl("y", "layout", "range", "Y position", "top", 1, func(c Config) int { return c.Y }, func(c *Config, v int) { c.Y = v }),
			card.IntControl("width", "layout", "range", "Width", "width", 1, func(c Config) int { return c.Width }, func(c *Config, v int) { c.Width = v }),
		}, Properties: []card.Property[Config]{{ID: "form_id", Kind: schema.PropertyString, Read: func(c Config) (schema.PropertyValue, bool) { return schema.StringValue(c.FormID), true }}}, Roles: []card.ComponentRole{card.RoleFormSubmitter}, Install: &card.InstallSpec{Policy: card.InstallAppend},
	})
}

func NormalizeConfig(part Config) Config {
	part.FormID = strings.TrimSpace(part.FormID)
	part.Label = strings.TrimSpace(part.Label)
	return part
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = NormalizeConfig(part)
	style := map[string]string{
		"left":      fmt.Sprintf("%d%%", part.X),
		"top":       fmt.Sprintf("%d%%", part.Y),
		"transform": "translate(-50%, -50%)",
		"width":     fmt.Sprintf("%d%%", part.Width),
		"z-index":   "4",
	}
	return godom.Button(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Type("submit"),
		godom.Attr("form", renderContext.DOMID(part.FormID)),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("data-card-form-control", ""),
		godom.Attr("data-form-id", part.FormID),
		godom.Attr("style", styleString(style)),
		godom.Class("absolute rounded-xl border border-emerald-200/70 bg-emerald-500/90 px-4 py-2 text-sm font-black uppercase tracking-[0.12em] text-emerald-950 shadow-lg"),
		godom.T(part.Label),
	)
}

func styleString(styles map[string]string) string {
	keys := make([]string, 0, len(styles))
	for key := range styles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(styles[key])
		out.WriteString("; ")
	}
	return strings.TrimSpace(out.String())
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
