package textinput

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

const Kind = "text_input"

type Config struct {
	FormID      string `json:"form_id"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	InputType   string `json:"input_type"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Width       int    `json:"width"`
}

func DefaultConfig() Config {
	return Config{
		FormID:      "controller-form",
		Name:        "text",
		Label:       "Text",
		Placeholder: "Enter text",
		InputType:   "text",
		X:           50,
		Y:           54,
		Width:       72,
	}
}

func Definition() card.Definition {
	return card.MustDefine(card.TypedDefinition[Config]{
		Kind: Kind, Label: "Text Input", Structure: card.StructureLeaf, Default: DefaultConfig, Normalize: NormalizeConfig,
		ConfigRules: []schema.FieldRule{
			schema.StringFormat("form_id", schema.FormatSafeToken),
			schema.StringFormat("name", schema.FormatSafeToken),
			schema.StringLength("label", 1, 80),
			schema.StringMaxLength("placeholder", 120),
			schema.Enum("input_type", "text", "password"),
			schema.IntegerRange("x", 0, 100),
			schema.IntegerRange("y", 0, 100),
			schema.IntegerRange("width", 12, 100),
		},
		Render: func(node card.Node, part Config, renderContext card.RenderContext) (card.Contribution, error) {
			return card.Contribution{Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)}}, nil
		},
		Controls: []card.Control[Config]{
			card.StringControl("form_id", "form", "text", "Form ID", "form", func(c Config) string { return c.FormID }, func(c *Config, v string) { c.FormID = v }),
			card.StringControl("name", "form", "text", "Field name", "name", func(c Config) string { return c.Name }, func(c *Config, v string) { c.Name = v }),
			card.StringControl("label", "content", "text", "Label", "label", func(c Config) string { return c.Label }, func(c *Config, v string) { c.Label = v }),
			card.StringControl("placeholder", "content", "text", "Placeholder", "placeholder", func(c Config) string { return c.Placeholder }, func(c *Config, v string) { c.Placeholder = v }),
			card.StringControl("input_type", "content", "select", "Input type", "type", func(c Config) string { return c.InputType }, func(c *Config, v string) { c.InputType = v }),
			positionControl(),
			card.IntControl("x", "layout", "range", "X position", "left", 1, func(c Config) int { return c.X }, func(c *Config, v int) { c.X = v }),
			card.IntControl("y", "layout", "range", "Y position", "top", 1, func(c Config) int { return c.Y }, func(c *Config, v int) { c.Y = v }),
			card.IntControl("width", "layout", "range", "Width", "width", 1, func(c Config) int { return c.Width }, func(c *Config, v int) { c.Width = v }),
		},
		Properties: []card.Property[Config]{
			{ID: "form_id", Kind: schema.PropertyString, Read: func(c Config) (schema.PropertyValue, bool) { return schema.StringValue(c.FormID), true }},
			{ID: "name", Kind: schema.PropertyString, Read: func(c Config) (schema.PropertyValue, bool) { return schema.StringValue(c.Name), true }},
		}, Roles: []card.ComponentRole{card.RoleFormField}, Install: &card.InstallSpec{Policy: card.InstallAppend},
	})
}

func NormalizeConfig(part Config) Config {
	part.FormID = strings.TrimSpace(part.FormID)
	part.Name = strings.TrimSpace(part.Name)
	part.Label = strings.TrimSpace(part.Label)
	part.Placeholder = strings.TrimSpace(part.Placeholder)
	part.InputType = strings.TrimSpace(part.InputType)
	return part
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = NormalizeConfig(part)
	formID := renderContext.DOMID(part.FormID)
	inputID := renderContext.DOMID(componentID + "-input")
	style := map[string]string{
		"left":      fmt.Sprintf("%d%%", part.X),
		"top":       fmt.Sprintf("%d%%", part.Y),
		"transform": "translate(-50%, -50%)",
		"width":     fmt.Sprintf("%d%%", part.Width),
		"z-index":   "3",
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("data-card-form-control", ""),
		godom.Attr("style", styleString(style)),
		godom.Form(
			godom.Id(formID),
			godom.Attr("data-card-form", ""),
			godom.Attr("data-form-id", part.FormID),
			godom.Class("grid gap-2"),
			godom.Label(
				godom.For(inputID),
				godom.Class("text-xs font-bold uppercase tracking-[0.12em] text-slate-200"),
				godom.T(part.Label),
			),
			godom.Input(
				godom.Id(inputID),
				godom.Type(part.InputType),
				godom.Name(part.Name),
				godom.Placeholder(part.Placeholder),
				godom.Attr("autocomplete", "off"),
				godom.Attr("data-card-form-input", ""),
				godom.Class("w-full rounded-xl border border-sky-300/60 bg-slate-950/80 px-3 py-2 text-sm text-slate-50 outline-none"),
			),
		),
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
