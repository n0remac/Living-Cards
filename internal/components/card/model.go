package card

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"
)

const (
	Type                 = "card"
	DefaultCardID        = "draft-card"
	DefaultRootID        = "card-root"
	DefaultBackgroundID  = "background-primary"
	DefaultBorderID      = "border-primary"
	DefaultTextareaID    = "textarea-main"
	DefaultShapeID       = "shape-1"
	DefaultImageID       = "image-1"
	TypeBackground       = "background"
	TypeBorder           = "border"
	TypeTextarea         = "textarea"
	TypeShape            = "shape"
	TypeImage            = "image"
	TypeSlider           = "slider"
	defaultRootRaw       = `{"padding_px":24,"shadow":""}`
	defaultBackgroundRaw = `{"background_color":"#111827","css":""}`
	defaultBorderRaw     = `{"border_width_px":1,"border_radius_px":24,"border_color":"rgba(255,255,255,0.16)","css":""}`
	defaultTextareaRaw   = `{"content":"Start designing this card.","font_family":"system","font_size_px":16,"font_weight":400,"font_style":"normal","color":"#cbd5e1","align":"left","position":"center","css":""}`
)

type Document struct {
	CardID string `json:"card_id"`
	Name   string `json:"name"`
	Root   Node   `json:"root"`
}

type LibraryItem struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Target      string          `json:"target"`
	Description string          `json:"description"`
	Fragment    json.RawMessage `json:"fragment"`
	Saved       bool            `json:"saved,omitempty"`
}

type Node struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Fragment json.RawMessage `json:"fragment,omitempty"`
	Children []Node          `json:"children,omitempty"`
}

type RootFragment struct {
	PaddingPX int    `json:"padding_px"`
	Shadow    string `json:"shadow"`
}

type Contribution struct {
	ShellStyle map[string]string
	Layers     []*godom.Node
}

type RenderOptions struct {
	ElementID   string
	DOMIDPrefix string
}

type RenderContext struct {
	DOMIDPrefix string
}

func (c RenderContext) LayerID(componentID string) string {
	componentID = strings.TrimSpace(componentID)
	layerID := componentID + "-layer"
	prefix := strings.TrimSpace(c.DOMIDPrefix)
	if prefix == "" {
		return layerID
	}
	return prefix + "-" + layerID
}

type Definition struct {
	Type       string
	Contribute func(Node, RenderContext) (Contribution, error)
}

type Registry struct {
	definitions map[string]Definition
}

func NewRegistry(definitions ...Definition) (*Registry, error) {
	registry := &Registry{definitions: make(map[string]Definition, len(definitions))}
	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func MustNewRegistry(definitions ...Definition) *Registry {
	registry, err := NewRegistry(definitions...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *Registry) Register(definition Definition) error {
	definition.Type = strings.TrimSpace(definition.Type)
	if definition.Type == "" {
		return fmt.Errorf("component type is required")
	}
	if definition.Contribute == nil {
		return fmt.Errorf("component %q contribution is required", definition.Type)
	}
	if _, exists := r.definitions[definition.Type]; exists {
		return fmt.Errorf("duplicate component type %q", definition.Type)
	}
	r.definitions[definition.Type] = definition
	return nil
}

func DefaultDocument() Document {
	return Document{
		CardID: DefaultCardID,
		Name:   "Empty Card",
		Root: Node{
			ID:       DefaultRootID,
			Type:     Type,
			Fragment: json.RawMessage(defaultRootRaw),
			Children: []Node{
				{ID: DefaultBackgroundID, Type: TypeBackground, Fragment: json.RawMessage(defaultBackgroundRaw)},
				{ID: DefaultBorderID, Type: TypeBorder, Fragment: json.RawMessage(defaultBorderRaw)},
				{ID: DefaultTextareaID, Type: TypeTextarea, Fragment: json.RawMessage(defaultTextareaRaw)},
			},
		},
	}
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
	if document.Root.Type != Type {
		return nil, fmt.Errorf("root component type must be %q", Type)
	}
	rootStyle := DecodeRootFragment(document.Root.Fragment)
	shellStyle := map[string]string{
		"background-color": "#111827",
		"border":           "1px solid rgba(255,255,255,0.16)",
		"border-radius":    "24px",
		"padding":          fmt.Sprintf("%dpx", rootStyle.PaddingPX),
	}
	var layers []*godom.Node
	for _, child := range document.Root.Children {
		definition, ok := registry.definitions[child.Type]
		if !ok {
			return nil, fmt.Errorf("component type %q is not registered", child.Type)
		}
		contribution, err := definition.Contribute(child, RenderContext{DOMIDPrefix: options.DOMIDPrefix})
		if err != nil {
			return nil, err
		}
		for property, value := range contribution.ShellStyle {
			if strings.TrimSpace(value) == "" {
				continue
			}
			shellStyle[property] = value
		}
		layers = append(layers, contribution.Layers...)
	}
	if strings.TrimSpace(rootStyle.Shadow) != "" {
		shellStyle["box-shadow"] = strings.TrimSpace(rootStyle.Shadow)
	}
	attributes := []*godom.Node{
		godom.Class("relative aspect-[5/7] w-full max-w-md overflow-hidden p-6 shadow-2xl transition-[background,border,border-radius,box-shadow] duration-200"),
		godom.Attr("data-card-id", document.CardID),
		godom.Attr("data-component-id", document.Root.ID),
		godom.Attr("data-component-type", Type),
		godom.Attr("style", styleString(shellStyle)),
		godom.Ch(layers),
	}
	if strings.TrimSpace(options.ElementID) != "" {
		attributes = append([]*godom.Node{godom.Id(strings.TrimSpace(options.ElementID))}, attributes...)
	}
	return godom.Div(attributes...), nil
}

func DefaultRootFragment() RootFragment {
	return RootFragment{
		PaddingPX: 24,
		Shadow:    "",
	}
}

func DecodeRootFragment(raw json.RawMessage) RootFragment {
	part := DefaultRootFragment()
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &part)
	}
	part.PaddingPX = clamp(part.PaddingPX, 0, 48)
	part.Shadow = strings.TrimSpace(part.Shadow)
	return part
}

func EncodeRootFragment(part RootFragment) json.RawMessage {
	part.PaddingPX = clamp(part.PaddingPX, 0, 48)
	part.Shadow = strings.TrimSpace(part.Shadow)
	raw, err := json.Marshal(part)
	if err != nil {
		panic(err)
	}
	return raw
}

func DecodeFragment[T any](node Node) (T, error) {
	var fragment T
	if len(node.Fragment) == 0 {
		return fragment, fmt.Errorf("component %q has no fragment", node.ID)
	}
	if err := json.Unmarshal(node.Fragment, &fragment); err != nil {
		return fragment, fmt.Errorf("decode %s fragment: %w", node.Type, err)
	}
	return fragment, nil
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

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
