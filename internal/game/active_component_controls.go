package game

import (
	"encoding/json"
	"fmt"
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

func worldComponentKindEditable(registry *cardcomponent.Registry, componentKind string) bool {
	definition, ok := registry.Lookup(componentKind)
	return ok && len(definition.ControlIDs()) > 0
}

func componentEditLabel(registry *cardcomponent.Registry, componentKind string) string {
	if definition, ok := registry.Lookup(componentKind); ok {
		return definition.Label()
	}
	return "Component"
}

func applyGameComponentControl(registry *cardcomponent.Registry, node *cardcomponent.Node, control string, value json.RawMessage) error {
	if node == nil {
		return fmt.Errorf("component is required")
	}
	definition, ok := registry.Lookup(node.ComponentKind)
	if !ok {
		return fmt.Errorf("component kind %q is not registered", node.ComponentKind)
	}
	canonical, issues := definition.ApplyControl(node.Config, strings.TrimSpace(control), value)
	if len(issues) > 0 {
		return fmt.Errorf("invalid %s control at %s: %s", node.ComponentKind, issues[0].Path, issues[0].Message)
	}
	node.Config = canonical
	return nil
}
