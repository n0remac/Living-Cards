package web

import (
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/game"
)

func gameEditingOverlay(registry *cardcomponent.Registry, card game.Card, selectedComponentID string) *ComponentOverlay {
	selectedComponentID = strings.TrimSpace(selectedComponentID)
	if selectedComponentID == "" {
		return nil
	}
	if node := findNodeByID(card.Document.Root, selectedComponentID); node != nil {
		return gameEditingOverlayForNode(registry, *node)
	}
	return nil
}

func gameActiveEditingOverlay(registry *cardcomponent.Registry, card game.Card, selectedComponentID string, _ []game.Card) *ComponentOverlay {
	return gameEditingOverlay(registry, card, selectedComponentID)
}

func gameEditingOverlayForNode(registry *cardcomponent.Registry, node cardcomponent.Node) *ComponentOverlay {
	definition, ok := registry.Lookup(node.ComponentKind)
	if !ok {
		return nil
	}
	controls, issues := definition.Controls(node.Config)
	if len(issues) > 0 || len(controls) == 0 {
		return nil
	}
	return &ComponentOverlay{ComponentID: node.ID, ComponentKind: node.ComponentKind, Title: definition.Label(), Controls: controls}
}
