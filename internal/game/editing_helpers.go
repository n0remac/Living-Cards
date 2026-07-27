package game

import (
	"strings"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

func stateBool(state map[string]any, key string) bool {
	value, ok := state[key].(bool)
	return ok && value
}

func appendStringOnce(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, candidate := range values {
		if candidate == value {
			return values
		}
	}
	return append(values, value)
}

func appendStateStringOnce(value any, next string) []string {
	next = strings.TrimSpace(next)
	var values []string
	switch typed := value.(type) {
	case []string:
		values = append(values, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = appendStringOnce(values, text)
			}
		}
	}
	return appendStringOnce(values, next)
}

func findNodeByKind(node cardcomponent.Node, componentKind string) *cardcomponent.Node {
	if node.ComponentKind == componentKind {
		return &node
	}
	for _, child := range node.Children {
		if match := findNodeByKind(child, componentKind); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByKindPtr(node *cardcomponent.Node, componentKind string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ComponentKind == componentKind {
		return node
	}
	for index := range node.Children {
		if match := findNodeByKindPtr(&node.Children[index], componentKind); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByID(node cardcomponent.Node, componentID string) *cardcomponent.Node {
	if node.ID == componentID {
		return &node
	}
	for _, child := range node.Children {
		if match := findNodeByID(child, componentID); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByIDPtr(node *cardcomponent.Node, componentID string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ID == componentID {
		return node
	}
	for index := range node.Children {
		if match := findNodeByIDPtr(&node.Children[index], componentID); match != nil {
			return match
		}
	}
	return nil
}

func stringInSlice(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func instanceIDInSlice(values []CardInstanceID, value CardInstanceID) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
