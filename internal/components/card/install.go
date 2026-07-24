package card

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (r *Registry) CanonicalizeTemplate(template ComponentTemplate) (ComponentTemplate, error) {
	raw, err := json.Marshal(template)
	if err != nil {
		return ComponentTemplate{}, err
	}
	canonical, issues := r.DecodeTemplate(raw)
	if len(issues) > 0 {
		if issues[0].Code == "unsupported" {
			return ComponentTemplate{}, NewUnsupportedOperationError(template.ComponentKind, "installation")
		}
		return ComponentTemplate{}, fmt.Errorf("invalid component template at %s: %s", issues[0].Path, issues[0].Message)
	}
	return canonical, nil
}

func (r *Registry) InstallTemplate(document Document, template ComponentTemplate) (Document, Node, error) {
	canonicalDocument, issues := r.CanonicalizeDocument(document)
	if len(issues) > 0 {
		return Document{}, Node{}, fmt.Errorf("invalid target document at %s: %s", issues[0].Path, issues[0].Message)
	}
	template, err := r.CanonicalizeTemplate(template)
	if err != nil {
		return Document{}, Node{}, err
	}
	definition, _ := r.Lookup(template.ComponentKind)
	install, _ := definition.Install()

	requestedID := template.ComponentID
	if requestedID != "" && findNodeByID(&canonicalDocument.Root, requestedID) != nil {
		return Document{}, Node{}, fmt.Errorf("component id %q already exists", requestedID)
	}
	matches := findNodesByKind(&canonicalDocument.Root, template.ComponentKind)
	if install.Policy == InstallReplaceKind && len(matches) > 1 {
		return Document{}, Node{}, fmt.Errorf("replace_kind is ambiguous: document contains %d %s components", len(matches), template.ComponentKind)
	}
	if install.Policy == InstallReplaceKind && len(matches) == 1 {
		node := matches[0]
		if requestedID != "" {
			node.ID = requestedID
		}
		node.Config = append(json.RawMessage(nil), template.Config...)
		return canonicalDocument, *node, nil
	}
	if requestedID == "" {
		requestedID = nextID(canonicalDocument, canonicalDocument.CardID+"-"+strings.ReplaceAll(template.ComponentKind, "_", "-"))
	}
	node := Node{ID: requestedID, ComponentKind: template.ComponentKind, Config: append(json.RawMessage(nil), template.Config...)}
	canonicalDocument.Root.Children = append(canonicalDocument.Root.Children, node)
	return canonicalDocument, node, nil
}

func nextID(document Document, base string) string {
	if findNodeByID(&document.Root, base) == nil {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if findNodeByID(&document.Root, candidate) == nil {
			return candidate
		}
	}
}
func findNodeByID(node *Node, id string) *Node {
	if node == nil {
		return nil
	}
	if node.ID == id {
		return node
	}
	for index := range node.Children {
		if found := findNodeByID(&node.Children[index], id); found != nil {
			return found
		}
	}
	return nil
}
func findNodesByKind(node *Node, kind string) []*Node {
	var out []*Node
	var walk func(*Node)
	walk = func(current *Node) {
		if current.ComponentKind == kind {
			out = append(out, current)
		}
		for index := range current.Children {
			walk(&current.Children[index])
		}
	}
	walk(node)
	return out
}
