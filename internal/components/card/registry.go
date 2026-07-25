package card

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

type Registry struct {
	definitions map[string]Definition
	ordered     []Definition
}

func NewRegistry(definitions ...Definition) (*Registry, error) {
	r := &Registry{definitions: make(map[string]Definition, len(definitions)), ordered: make([]Definition, 0, len(definitions))}
	for _, definition := range definitions {
		kind := definition.Kind()
		if kind == "" || definition.canonicalizeConfig == nil {
			return nil, fmt.Errorf("invalid component definition")
		}
		if _, exists := r.definitions[kind]; exists {
			return nil, fmt.Errorf("duplicate component kind %q", kind)
		}
		r.definitions[kind] = definition
		r.ordered = append(r.ordered, definition)
	}
	rootCount := 0
	for _, definition := range r.ordered {
		if definition.Structure() == StructureRoot {
			rootCount++
			if definition.Kind() != Kind {
				return nil, fmt.Errorf("component %q cannot be a root", definition.Kind())
			}
		}
	}
	if rootCount != 1 {
		return nil, fmt.Errorf("catalog must contain exactly one card root definition")
	}
	return r, nil
}
func MustNewRegistry(definitions ...Definition) *Registry {
	r, err := NewRegistry(definitions...)
	if err != nil {
		panic(err)
	}
	return r
}
func (r *Registry) Lookup(kind string) (Definition, bool) {
	if r == nil {
		return Definition{}, false
	}
	d, ok := r.definitions[kind]
	return d, ok
}
func (r *Registry) Definitions() []Definition {
	if r == nil {
		return nil
	}
	return append([]Definition(nil), r.ordered...)
}
func (r *Registry) Presets() []LibraryItem {
	var out []LibraryItem
	for _, definition := range r.ordered {
		out = append(out, definition.Presets()...)
	}
	return out
}

func (r *Registry) Schema() schema.CatalogSchema {
	if r == nil {
		return schema.CatalogSchema{}
	}
	out := schema.CatalogSchema{Components: make([]schema.ComponentSchema, 0, len(r.ordered))}
	for _, definition := range r.ordered {
		out.Components = append(out.Components, definition.Schema())
	}
	return schema.CloneCatalog(out)
}

type documentWire struct {
	CardID string          `json:"card_id"`
	Name   string          `json:"name"`
	Root   json.RawMessage `json:"root"`
}
type nodeWire struct {
	ID            string            `json:"id"`
	ComponentKind string            `json:"component_kind"`
	Config        json.RawMessage   `json:"config"`
	Children      []json.RawMessage `json:"children,omitempty"`
}

func (r *Registry) DecodeDocument(raw []byte) (Document, []schema.Issue) {
	var wire documentWire
	if err := decodeStrictObject(raw, &wire); err != nil {
		return Document{}, []schema.Issue{{Path: "$", Code: "invalid_document", Message: err.Error()}}
	}
	document := Document{CardID: wire.CardID, Name: strings.TrimSpace(wire.Name)}
	var issues []schema.Issue
	if !ValidComponentID(document.CardID) {
		issues = append(issues, schema.Issue{Path: "card_id", Code: "invalid_id", Message: "card_id must contain only letters, numbers, hyphens, or underscores"})
	}
	if len(wire.Root) == 0 {
		issues = append(issues, schema.Issue{Path: "root", Code: "required", Message: "root is required"})
		return document, issues
	}
	seen := map[string]bool{}
	root, nodeIssues := r.decodeNode(wire.Root, "root", true, seen)
	issues = append(issues, nodeIssues...)
	document.Root = root
	return document, issues
}

func (r *Registry) CanonicalizeDocument(document Document) (Document, []schema.Issue) {
	raw, err := json.Marshal(document)
	if err != nil {
		return Document{}, []schema.Issue{{Path: "$", Code: "encode_failed", Message: err.Error()}}
	}
	return r.DecodeDocument(raw)
}

func (r *Registry) decodeNode(raw json.RawMessage, path string, isRoot bool, seen map[string]bool) (Node, []schema.Issue) {
	var wire nodeWire
	if err := decodeStrictObject(raw, &wire); err != nil {
		return Node{}, []schema.Issue{{Path: path, Code: "invalid_node", Message: err.Error()}}
	}
	node := Node{ID: wire.ID, ComponentKind: wire.ComponentKind}
	var issues []schema.Issue
	if !ValidComponentID(wire.ID) {
		issues = append(issues, schema.Issue{Path: path + ".id", Code: "invalid_id", Message: "id must contain only letters, numbers, hyphens, or underscores", Actual: wire.ID})
	} else if seen[wire.ID] {
		issues = append(issues, schema.Issue{Path: path + ".id", Code: "duplicate_id", Message: "component id must be unique", Actual: wire.ID})
	} else {
		seen[wire.ID] = true
	}
	definition, ok := r.Lookup(wire.ComponentKind)
	if !ok {
		issues = append(issues, schema.Issue{Path: path + ".component_kind", Code: "unknown_component_kind", Message: "component kind is not registered", Actual: wire.ComponentKind})
		node.Config = append(json.RawMessage(nil), wire.Config...)
	} else {
		if isRoot {
			if definition.Structure() != StructureRoot || wire.ComponentKind != Kind {
				issues = append(issues, schema.Issue{Path: path + ".component_kind", Code: "invalid_root", Message: "document root must be card"})
			}
		} else if definition.Structure() != StructureLeaf {
			issues = append(issues, schema.Issue{Path: path + ".component_kind", Code: "root_only", Message: "root components cannot be nested"})
		}
		canonical, configIssues := definition.CanonicalizeConfig(RawConfig{Present: wire.Config != nil, Value: wire.Config})
		for _, issue := range configIssues {
			issue.Path = path + "." + issue.Path
			issues = append(issues, issue)
		}
		node.Config = canonical
		if definition.Structure() == StructureLeaf && len(wire.Children) > 0 {
			issues = append(issues, schema.Issue{Path: path + ".children", Code: "children_not_allowed", Message: "leaf components cannot have children"})
		}
	}
	for index, childRaw := range wire.Children {
		child, childIssues := r.decodeNode(childRaw, fmt.Sprintf("%s.children[%d]", path, index), false, seen)
		node.Children = append(node.Children, child)
		issues = append(issues, childIssues...)
	}
	return node, issues
}

func (r *Registry) DecodeTemplate(raw []byte) (ComponentTemplate, []schema.Issue) {
	type templateWire struct {
		ComponentKind string          `json:"component_kind"`
		ComponentID   string          `json:"component_id,omitempty"`
		Config        json.RawMessage `json:"config"`
	}
	var wire templateWire
	if err := decodeStrictObject(raw, &wire); err != nil {
		return ComponentTemplate{}, []schema.Issue{{Path: "$", Code: "invalid_template", Message: err.Error()}}
	}
	definition, ok := r.Lookup(wire.ComponentKind)
	if !ok {
		return ComponentTemplate{}, []schema.Issue{{Path: "component_kind", Code: "unknown_component_kind", Message: "component kind is not registered", Actual: wire.ComponentKind}}
	}
	if _, ok := definition.Install(); !ok {
		return ComponentTemplate{}, []schema.Issue{{Path: "component_kind", Code: "unsupported", Message: "component is not installable", Actual: wire.ComponentKind}}
	}
	if wire.ComponentID != "" && !ValidComponentID(wire.ComponentID) {
		return ComponentTemplate{}, []schema.Issue{{Path: "component_id", Code: "invalid_id", Message: "component_id must contain only letters, numbers, hyphens, or underscores"}}
	}
	config, issues := definition.CanonicalizeConfig(RawConfig{Present: wire.Config != nil, Value: wire.Config})
	if len(issues) > 0 {
		return ComponentTemplate{}, issues
	}
	return ComponentTemplate{ComponentKind: wire.ComponentKind, ComponentID: wire.ComponentID, Config: config}, nil
}

func ValidComponentID(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9', char == '-', char == '_':
		default:
			return false
		}
	}
	return true
}
