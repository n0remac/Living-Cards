// Package schema contains component-language values shared by the registry,
// component definitions, and generation services. It deliberately has no
// dependency on component implementations or application packages.
package schema

import "encoding/json"

type Issue struct {
	Path    string   `json:"path"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Actual  any      `json:"actual,omitempty"`
	Allowed []string `json:"allowed,omitempty"`
}

type PropertyKind string

const (
	PropertyString PropertyKind = "string"
	PropertyNumber PropertyKind = "number"
	PropertyBool   PropertyKind = "bool"
)

type PropertyValue struct {
	Kind   PropertyKind `json:"kind"`
	String string       `json:"string,omitempty"`
	Number float64      `json:"number,omitempty"`
	Bool   bool         `json:"bool,omitempty"`
}

func StringValue(value string) PropertyValue {
	return PropertyValue{Kind: PropertyString, String: value}
}

func NumberValue(value float64) PropertyValue {
	return PropertyValue{Kind: PropertyNumber, Number: value}
}

func BoolValue(value bool) PropertyValue {
	return PropertyValue{Kind: PropertyBool, Bool: value}
}

type GeneratedConfig[T any] struct {
	ComponentKind string `json:"component_kind"`
	Description   string `json:"description"`
	Config        T      `json:"config"`
}

type GeneratedConfigEnvelope struct {
	ComponentKind string          `json:"component_kind"`
	Description   string          `json:"description"`
	Config        json.RawMessage `json:"config"`
}
