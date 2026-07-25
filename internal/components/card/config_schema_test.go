package card

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

type schemaChild struct {
	Label string `json:"label"`
}

type schemaNestedConfig struct {
	Child schemaChild   `json:"child"`
	Items []schemaChild `json:"items"`
}

func TestDeriveConfigSchemaSupportsTheDeliberateGoSubset(t *testing.T) {
	value, err := deriveConfigSchema(reflect.TypeOf(schemaNestedConfig{}), []schema.FieldRule{
		schema.StringLength("child.label", 1, 20),
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.Kind != schema.ValueObject || len(value.Fields) != 2 {
		t.Fatalf("schema = %#v", value)
	}
	child := value.Fields[0].Schema
	if child.Kind != schema.ValueObject || child.Fields[0].Schema.MinLength == nil || *child.Fields[0].Schema.MinLength != 1 {
		t.Fatalf("child schema = %#v", child)
	}
	items := value.Fields[1].Schema
	if items.Kind != schema.ValueArray || items.Items == nil || items.Items.Kind != schema.ValueObject {
		t.Fatalf("items schema = %#v", items)
	}
}

type customJSONConfig struct {
	Value customJSONValue `json:"value"`
}

type customJSONValue string

func (customJSONValue) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

type recursiveConfig struct {
	Children []recursiveConfig `json:"children"`
}

type embeddedFields struct {
	Name string `json:"name"`
}

func TestDeriveConfigSchemaRejectsUnsupportedGoShapes(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{name: "map", typ: reflect.TypeOf(struct {
			Values map[string]string `json:"values"`
		}{}), want: "maps are not supported"},
		{name: "pointer", typ: reflect.TypeOf(struct {
			Value *string `json:"value"`
		}{}), want: "pointers are not supported"},
		{name: "interface", typ: reflect.TypeOf(struct {
			Value any `json:"value"`
		}{}), want: "interfaces are not supported"},
		{name: "raw message", typ: reflect.TypeOf(struct {
			Value json.RawMessage `json:"value"`
		}{}), want: "custom JSON or text encoding"},
		{name: "custom marshaler", typ: reflect.TypeOf(customJSONConfig{}), want: "custom JSON or text encoding"},
		{name: "recursive", typ: reflect.TypeOf(recursiveConfig{}), want: "is recursive"},
		{name: "embedded", typ: reflect.TypeOf(struct {
			embeddedFields
		}{}), want: "must not be embedded"},
		{name: "omitempty", typ: reflect.TypeOf(struct {
			Name string `json:"name,omitempty"`
		}{}), want: "omitempty"},
		{name: "int64", typ: reflect.TypeOf(struct {
			Count int64 `json:"count"`
		}{}), want: "is not supported"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := deriveConfigSchema(test.typ, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestConfigSchemaRejectsNestedNullAndUnknownFieldsBeforeDecode(t *testing.T) {
	value, err := deriveConfigSchema(reflect.TypeOf(schemaNestedConfig{}), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		raw  string
		code string
	}{
		{name: "nested null", raw: `{"child":null}`, code: "null_value"},
		{name: "array item null", raw: `{"items":[null]}`, code: "null_value"},
		{name: "nested unknown", raw: `{"child":{"unknown":true}}`, code: "unknown_field"},
	} {
		t.Run(test.name, func(t *testing.T) {
			issues := schema.ValidateJSONStructure(json.RawMessage(test.raw), value, "config")
			if len(issues) == 0 || issues[0].Code != test.code {
				t.Fatalf("issues = %#v, want %q", issues, test.code)
			}
		})
	}
}

func TestConfigSchemaRejectsContradictoryDeclarativeRules(t *testing.T) {
	type config struct {
		Count int `json:"count"`
	}
	_, err := deriveConfigSchema(reflect.TypeOf(config{}), []schema.FieldRule{
		schema.IntegerRange("count", 0, 1),
		schema.Enum("count", 0, 2),
	})
	if err == nil || !strings.Contains(err.Error(), "violates its field rules") {
		t.Fatalf("error = %v", err)
	}
}
