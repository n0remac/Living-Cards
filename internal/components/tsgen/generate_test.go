package tsgen

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestGenerateIsDeterministicAndEmitsDeepInputTypes(t *testing.T) {
	stringValue := schema.ValueSchema{Kind: schema.ValueString}
	child := schema.ValueSchema{Kind: schema.ValueObject, Fields: []schema.FieldSchema{
		{JSONName: "label", Schema: stringValue, Required: true},
	}}
	items := schema.CloneValue(child)
	matrixItems := schema.ValueSchema{Kind: schema.ValueArray, Items: &items}
	catalog := schema.CatalogSchema{Components: []schema.ComponentSchema{
		{
			Kind: "card", Label: "Card", Structure: schema.StructureRoot,
			Config:  object(field("title", stringValue)),
			Default: json.RawMessage(`{"title":"Root"}`),
		},
		{
			Kind: "synthetic_widget", Label: "Synthetic", Structure: schema.StructureLeaf,
			Config: object(
				field("child", child),
				field("items", schema.ValueSchema{Kind: schema.ValueArray, Items: &items}),
				field("matrix", schema.ValueSchema{Kind: schema.ValueArray, Items: &matrixItems}),
				field("mode", schema.ValueSchema{Kind: schema.ValueString, Enum: []json.RawMessage{json.RawMessage(`"quiet"`), json.RawMessage(`"loud"`)}}),
			),
			Default: json.RawMessage(`{"child":{"label":"one"},"items":[{"label":"two"}],"matrix":[[{"label":"three"}]],"mode":"quiet"}`),
			Install: &schema.InstallSchema{Policy: schema.InstallAppend},
			Capabilities: schema.CapabilitySchema{
				Installable: true, RandomGeneration: true, AIGeneration: true,
			},
		},
	}}
	first, err := Generate(catalog)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(catalog)
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range first {
		if !bytes.Equal(content, second[path]) {
			t.Fatalf("%s generation is not deterministic", path)
		}
	}
	types := string(first[TypesFile])
	for _, marker := range []string{
		`export interface SyntheticWidgetConfigChildInput`,
		`"label"?: string;`,
		`"child"?: SyntheticWidgetConfigChildInput;`,
		`"items"?: Array<SyntheticWidgetConfigItemsItemInput>;`,
		`"matrix"?: Array<Array<SyntheticWidgetConfigMatrixItemItemInput>>;`,
		`"mode": "quiet" | "loud";`,
		`export type ComponentTemplateInput<K extends InstallableComponentKind`,
		`export type GeneratedConfigEnvelopeInput<K extends AIGeneratableComponentKind`,
	} {
		if !strings.Contains(types, marker) {
			t.Fatalf("generated types missing %q:\n%s", marker, types)
		}
	}
}

func TestGenerateRejectsTypeNameCollisions(t *testing.T) {
	config := object(field("name", schema.ValueSchema{Kind: schema.ValueString}))
	_, err := Generate(schema.CatalogSchema{Components: []schema.ComponentSchema{
		{Kind: "card", Structure: schema.StructureRoot, Config: config, Default: json.RawMessage(`{"name":""}`)},
		{Kind: "widget_1", Structure: schema.StructureLeaf, Config: config, Default: json.RawMessage(`{"name":""}`)},
		{Kind: "widget1", Structure: schema.StructureLeaf, Config: config, Default: json.RawMessage(`{"name":""}`)},
	}})
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("collision error = %v", err)
	}

	child := object(field("name", schema.ValueSchema{Kind: schema.ValueString}))
	_, err = Generate(schema.CatalogSchema{Components: []schema.ComponentSchema{
		{
			Kind: "card", Structure: schema.StructureRoot,
			Config:  object(field("child", child), field("child_input", child)),
			Default: json.RawMessage(`{"child":{"name":""},"child_input":{"name":""}}`),
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("nested input collision error = %v", err)
	}
}

func object(fields ...schema.FieldSchema) schema.ValueSchema {
	return schema.ValueSchema{Kind: schema.ValueObject, Fields: fields}
}

func field(name string, value schema.ValueSchema) schema.FieldSchema {
	return schema.FieldSchema{JSONName: name, Schema: value, Required: true}
}
