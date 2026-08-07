package card

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

const (
	Kind                = "card"
	KindBackground      = "background"
	KindBorder          = "border"
	KindText            = "text"
	KindShape           = "shape"
	KindImage           = "image"
	KindSlider          = "slider"
	KindTextInput       = "text_input"
	KindButton          = "button"
	KindCreature        = "creature"
	KindAttack          = "attack"
	DefaultCardID       = "draft-card"
	DefaultRootID       = "card-root"
	DefaultBackgroundID = "background-primary"
	DefaultBorderID     = "border-primary"
	DefaultTextID       = "text-main"
	DefaultShapeID      = "shape-1"
	DefaultImageID      = "image-1"
)

type Document struct {
	CardID string `json:"card_id"`
	Name   string `json:"name"`
	Root   Node   `json:"root"`
}

type Node struct {
	ID            string          `json:"id"`
	ComponentKind string          `json:"component_kind"`
	Config        json.RawMessage `json:"config,omitempty"`
	Children      []Node          `json:"children,omitempty"`
}

type ComponentTemplate struct {
	ComponentKind string          `json:"component_kind"`
	ComponentID   string          `json:"component_id,omitempty"`
	Config        json.RawMessage `json:"config,omitempty"`
}

type LibraryItem struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	ComponentKind string          `json:"component_kind"`
	Description   string          `json:"description"`
	Config        json.RawMessage `json:"config"`
	Saved         bool            `json:"saved,omitempty"`
}

type RootConfig struct {
	PaddingPX int    `json:"padding_px"`
	Shadow    string `json:"shadow"`
}

func DefaultRootConfig() RootConfig { return RootConfig{PaddingPX: 24} }

func normalizeRootConfig(config RootConfig) RootConfig {
	config.Shadow = strings.TrimSpace(config.Shadow)
	return config
}

func validateRootConfig(config RootConfig) []schema.Issue {
	if config.Shadow != "" {
		return schema.ValidateInlineCSS("config.shadow", "box-shadow: "+config.Shadow+";", map[string]struct{}{"box-shadow": {}})
	}
	return nil
}

func RootDefinition() Definition {
	return MustDefine(TypedDefinition[RootConfig]{
		Kind: Kind, Label: "Card", Structure: StructureRoot,
		Default: DefaultRootConfig, Normalize: normalizeRootConfig, Validate: validateRootConfig,
		ConfigRules: []schema.FieldRule{
			schema.IntegerRange("padding_px", 0, 48),
		},
		Render: func(_ Node, config RootConfig, _ RenderContext) (Contribution, error) {
			styles := map[string]string{"padding": fmt.Sprintf("%dpx", config.PaddingPX)}
			if config.Shadow != "" {
				styles["box-shadow"] = config.Shadow
			}
			return Contribution{ShellStyle: styles}, nil
		},
		Controls: []Control[RootConfig]{
			IntControl("padding_px", "padding", "range", "Padding", "padding", 1, func(c RootConfig) int { return c.PaddingPX }, func(c *RootConfig, v int) { c.PaddingPX = v }),
			WithSuggestions(StringControl("shadow", "shadow", "select", "Shadow", "box-shadow", func(c RootConfig) string { return c.Shadow }, func(c *RootConfig, v string) { c.Shadow = v }),
				Option("None", ""), Option("Soft", "0 16px 40px rgba(15,23,42,0.25)"), Option("Deep", "0 28px 70px rgba(15,23,42,0.45)")),
		},
		Generation: &TypedGenerationDefinition[RootConfig]{Random: func(seed int64, _ int) schema.GeneratedConfig[RootConfig] {
			random := rand.New(rand.NewSource(seed))
			paddings := []int{8, 16, 20, 24, 32, 40}
			shadows := []string{"", "0 18px 48px rgba(15,23,42,0.28)", "0 28px 70px rgba(15,23,42,0.42)", "0 0 34px rgba(52,211,153,0.28)"}
			return schema.GeneratedConfig[RootConfig]{ComponentKind: Kind, Description: "Randomized card shell", Config: RootConfig{PaddingPX: paddings[random.Intn(len(paddings))], Shadow: shadows[random.Intn(len(shadows))]}}
		}},
	})
}

func NewDefaultDocument(registry *Registry) (Document, error) {
	if registry == nil {
		return Document{}, fmt.Errorf("component registry is not initialized")
	}
	defaultConfig := func(kind string) (json.RawMessage, error) {
		definition, ok := registry.Lookup(kind)
		if !ok {
			return nil, fmt.Errorf("default catalog is missing component %q", kind)
		}
		config, issues := definition.CanonicalizeConfig(RawConfig{})
		if len(issues) > 0 {
			return nil, issuesError(kind, issues)
		}
		return config, nil
	}
	root, err := defaultConfig(Kind)
	if err != nil {
		return Document{}, err
	}
	background, err := defaultConfig(KindBackground)
	if err != nil {
		return Document{}, err
	}
	border, err := defaultConfig(KindBorder)
	if err != nil {
		return Document{}, err
	}
	text, err := defaultConfig(KindText)
	if err != nil {
		return Document{}, err
	}
	return Document{CardID: DefaultCardID, Name: "Empty Card", Root: Node{
		ID: DefaultRootID, ComponentKind: Kind, Config: root,
		Children: []Node{
			{ID: DefaultBackgroundID, ComponentKind: KindBackground, Config: background},
			{ID: DefaultBorderID, ComponentKind: KindBorder, Config: border},
			{ID: DefaultTextID, ComponentKind: KindText, Config: text},
		},
	}}, nil
}

func MustDefaultDocument(registry *Registry) Document {
	document, err := NewDefaultDocument(registry)
	if err != nil {
		panic(err)
	}
	return document
}

type Contribution struct {
	ShellStyle map[string]string
	Layers     []*godom.Node
}

type RenderOptions struct {
	ElementID   string
	DOMIDPrefix string
}

type RenderContext struct{ DOMIDPrefix string }

func (c RenderContext) LayerID(componentID string) string {
	return c.DOMID(strings.TrimSpace(componentID) + "-layer")
}
func (c RenderContext) DOMID(value string) string {
	value, prefix := strings.TrimSpace(value), strings.TrimSpace(c.DOMIDPrefix)
	if prefix == "" {
		return value
	}
	return prefix + "-" + value
}

func RenderDocument(document Document, registry *Registry) (*godom.Node, error) {
	return RenderDocumentWithID(document, registry, "draft-card-preview")
}

func RenderDocumentWithID(document Document, registry *Registry, elementID string) (*godom.Node, error) {
	return RenderDocumentWithOptions(document, registry, RenderOptions{ElementID: elementID})
}

func RenderDocumentWithOptions(document Document, registry *Registry, options RenderOptions) (*godom.Node, error) {
	if registry == nil {
		return nil, fmt.Errorf("card component registry is not initialized")
	}
	canonical, issues := registry.CanonicalizeDocument(document)
	if len(issues) > 0 {
		return nil, fmt.Errorf("invalid card document at %s: %s", issues[0].Path, issues[0].Message)
	}
	context := RenderContext{DOMIDPrefix: options.DOMIDPrefix}
	rootDefinition, _ := registry.Lookup(Kind)
	rootContribution, err := rootDefinition.Render(canonical.Root, context)
	if err != nil {
		return nil, err
	}
	shellStyle := map[string]string{
		"background-color": "#111827", "border": "1px solid rgba(255,255,255,0.16)", "border-radius": "24px", "padding": "24px",
	}
	for key, value := range rootContribution.ShellStyle {
		if strings.TrimSpace(value) != "" {
			shellStyle[key] = value
		}
	}
	layers := append([]*godom.Node(nil), rootContribution.Layers...)
	for _, child := range canonical.Root.Children {
		definition, _ := registry.Lookup(child.ComponentKind)
		contribution, err := definition.Render(child, context)
		if err != nil {
			return nil, err
		}
		for key, value := range contribution.ShellStyle {
			if strings.TrimSpace(value) != "" {
				shellStyle[key] = value
			}
		}
		layers = append(layers, contribution.Layers...)
	}
	attributes := []*godom.Node{
		godom.Class("relative aspect-[5/7] w-full max-w-md overflow-hidden p-6 shadow-2xl transition-[background,border,border-radius,box-shadow] duration-200"),
		godom.Attr("data-card-id", canonical.CardID), godom.Attr("data-component-id", canonical.Root.ID),
		godom.Attr("data-component-kind", Kind), godom.Attr("style", styleString(shellStyle)), godom.Ch(layers),
	}
	if strings.TrimSpace(options.ElementID) != "" {
		attributes = append([]*godom.Node{godom.Id(strings.TrimSpace(options.ElementID))}, attributes...)
	}
	return godom.Div(attributes...), nil
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
