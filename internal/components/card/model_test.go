package card_test

import (
	"encoding/json"
	"strings"
	"testing"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

type syntheticConfig struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

func syntheticDefinition(kind string, policy card.InstallPolicy) card.Definition {
	return card.MustDefine(card.TypedDefinition[syntheticConfig]{
		Kind: kind, Label: "Synthetic", Structure: card.StructureLeaf,
		Default: func() syntheticConfig {
			return syntheticConfig{Name: "default", Count: 5, Enabled: true, Mode: "quiet"}
		},
		Normalize: func(config syntheticConfig) syntheticConfig {
			config.Name = strings.TrimSpace(config.Name)
			return config
		},
		ConfigRules: []schema.FieldRule{
			schema.IntegerRange("count", 0, 10),
			schema.Enum("mode", "quiet", "loud"),
		},
		Render: func(node card.Node, config syntheticConfig, _ card.RenderContext) (card.Contribution, error) {
			return card.Contribution{Layers: []*godom.Node{godom.Div(godom.Attr("data-synthetic", node.ID), godom.T(config.Name))}}, nil
		},
		Controls: []card.Control[syntheticConfig]{
			card.StringControl("name", "content", "text", "Name", "name", func(c syntheticConfig) string { return c.Name }, func(c *syntheticConfig, value string) { c.Name = value }),
		},
		Properties: []card.Property[syntheticConfig]{
			{ID: "count", Kind: schema.PropertyNumber, Read: func(c syntheticConfig) (schema.PropertyValue, bool) {
				return schema.NumberValue(float64(c.Count)), true
			}},
			{ID: "name", Kind: schema.PropertyString, Read: func(c syntheticConfig) (schema.PropertyValue, bool) {
				return schema.StringValue(c.Name), true
			}},
			{ID: "enabled", Kind: schema.PropertyBool, Read: func(c syntheticConfig) (schema.PropertyValue, bool) {
				return schema.BoolValue(c.Enabled), true
			}},
		},
		Roles:   []card.ComponentRole{card.RoleFormField, card.RoleFormSubmitter},
		Install: &card.InstallSpec{Policy: policy},
		Presets: []card.TypedPreset[syntheticConfig]{{ID: "synthetic-default", Name: "Default", Config: syntheticConfig{Name: "preset", Count: 1, Enabled: true, Mode: "quiet"}}},
		Generation: &card.TypedGenerationDefinition[syntheticConfig]{
			SystemPrompt: "Generate synthetic config.", Example: `{"component_kind":"` + kind + `","description":"example","config":{"name":"generated","count":2,"enabled":true,"mode":"quiet"}}`,
			Random: func(_ int64, _ int) schema.GeneratedConfig[syntheticConfig] {
				return schema.GeneratedConfig[syntheticConfig]{ComponentKind: kind, Description: "random", Config: syntheticConfig{Name: "random", Count: 3, Enabled: true, Mode: "quiet"}}
			},
		},
	})
}

func syntheticRegistry(definitions ...card.Definition) *card.Registry {
	return card.MustNewRegistry(append([]card.Definition{card.RootDefinition()}, definitions...)...)
}

func TestTypedDefinitionExposesEveryCapabilityThroughErasedAPI(t *testing.T) {
	definition := syntheticDefinition("synthetic", card.InstallAppend)
	registry := syntheticRegistry(definition)
	componentSchema := registry.Schema().Components[1]
	if componentSchema.Kind != "synthetic" ||
		!componentSchema.Capabilities.Editable ||
		!componentSchema.Capabilities.Installable ||
		!componentSchema.Capabilities.HasProperties ||
		!componentSchema.Capabilities.HasPresets ||
		!componentSchema.Capabilities.RandomGeneration ||
		!componentSchema.Capabilities.AIGeneration {
		t.Fatalf("component schema capabilities = %#v", componentSchema)
	}
	if len(componentSchema.Controls) != 1 || componentSchema.Controls[0].ConfigPath != "name" {
		t.Fatalf("component schema controls = %#v", componentSchema.Controls)
	}
	componentSchema.Controls[0].ID = "mutated"
	if registry.Schema().Components[1].Controls[0].ID != "name" {
		t.Fatal("Registry.Schema() did not return a defensive copy")
	}

	canonical, issues := definition.CanonicalizeConfig(card.RawConfig{})
	if len(issues) != 0 || string(canonical) != `{"name":"default","count":5,"enabled":true,"mode":"quiet"}` {
		t.Fatalf("canonical = %s, issues = %#v", canonical, issues)
	}
	controls, issues := definition.Controls(canonical)
	if len(issues) != 0 || len(controls) != 1 || string(controls[0].Value) != `"default"` {
		t.Fatalf("controls = %#v, issues = %#v", controls, issues)
	}
	edited, issues := definition.ApplyControl(canonical, "name", json.RawMessage(`" changed "`))
	if len(issues) != 0 || !strings.Contains(string(edited), `"name":"changed"`) {
		t.Fatalf("edited = %s, issues = %#v", edited, issues)
	}
	property, ok, issues := definition.ReadProperty(edited, "count")
	if len(issues) != 0 || !ok || property.Kind != schema.PropertyNumber || property.Number != 5 {
		t.Fatalf("property = %#v, ok = %v, issues = %#v", property, ok, issues)
	}
	property, ok, issues = definition.ReadProperty(edited, "name")
	if len(issues) != 0 || !ok || property.Kind != schema.PropertyString || property.String != "changed" {
		t.Fatalf("property = %#v, ok = %v, issues = %#v", property, ok, issues)
	}
	property, ok, issues = definition.ReadProperty(edited, "enabled")
	if len(issues) != 0 || !ok || property.Kind != schema.PropertyBool || !property.Bool {
		t.Fatalf("property = %#v, ok = %v, issues = %#v", property, ok, issues)
	}
	if !definition.HasRole(card.RoleFormField) || len(definition.Presets()) != 1 {
		t.Fatalf("roles = %#v, presets = %#v", definition.Roles(), definition.Presets())
	}
	generation, ok := definition.Generation()
	if !ok || !generation.SupportsAI() || !generation.SupportsRandom() {
		t.Fatalf("generation = %#v, ok = %v", generation, ok)
	}
	random, issues := generation.Random(1, 1)
	if len(issues) != 0 || !strings.Contains(string(random), `"component_kind":"synthetic"`) {
		t.Fatalf("random = %s, issues = %#v", random, issues)
	}
	if _, issues := generation.CanonicalizeEnvelope(json.RawMessage(`{"component_kind":"synthetic","description":"missing"}`)); !hasIssue(issues, "required") {
		t.Fatalf("missing generated config issues = %#v", issues)
	}
	if _, issues := generation.CanonicalizeEnvelope(json.RawMessage(`{"component_kind":"synthetic","description":"null","config":null}`)); !hasIssue(issues, "null_config") {
		t.Fatalf("null generated config issues = %#v", issues)
	}
	document := documentWithChildren()
	document, _, err := registry.InstallTemplate(document, card.ComponentTemplate{ComponentKind: "synthetic"})
	if err != nil || document.Root.Children[len(document.Root.Children)-1].ID != "target-synthetic" {
		t.Fatalf("install document = %#v, err = %v", document, err)
	}
	view, err := card.RenderDocument(document, registry)
	if err != nil || !strings.Contains(view.Render(), `data-synthetic="target-synthetic"`) {
		t.Fatalf("render = %v, err = %v", view, err)
	}
}

func TestStrictConfigPresenceAndCanonicalValues(t *testing.T) {
	definition := syntheticDefinition("synthetic", card.InstallAppend)
	tests := []struct {
		name      string
		raw       card.RawConfig
		want      string
		wantIssue string
	}{
		{name: "omitted", raw: card.RawConfig{}, want: `{"name":"default","count":5,"enabled":true,"mode":"quiet"}`},
		{name: "empty object", raw: card.RawConfig{Present: true, Value: json.RawMessage(`{}`)}, want: `{"name":"default","count":5,"enabled":true,"mode":"quiet"}`},
		{name: "explicit zero and empty", raw: card.RawConfig{Present: true, Value: json.RawMessage(`{"name":"","count":0,"enabled":false}`)}, want: `{"name":"","count":0,"enabled":false,"mode":"quiet"}`},
		{name: "null", raw: card.RawConfig{Present: true, Value: json.RawMessage(`null`)}, wantIssue: "null_config"},
		{name: "unknown", raw: card.RawConfig{Present: true, Value: json.RawMessage(`{"unknown":1}`)}, wantIssue: "unknown_field"},
		{name: "range", raw: card.RawConfig{Present: true, Value: json.RawMessage(`{"count":99}`)}, wantIssue: "out_of_range"},
		{name: "enum casing", raw: card.RawConfig{Present: true, Value: json.RawMessage(`{"mode":"LOUD"}`)}, wantIssue: "invalid_option"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, issues := definition.CanonicalizeConfig(test.raw)
			if test.wantIssue != "" {
				if len(issues) == 0 || issues[0].Code != test.wantIssue {
					t.Fatalf("got %s with issues %#v, want %s", got, issues, test.wantIssue)
				}
				return
			}
			if len(issues) != 0 || string(got) != test.want {
				t.Fatalf("got %s with issues %#v, want %s", got, issues, test.want)
			}
		})
	}
}

func TestRootConfigRejectsUnsafeShadow(t *testing.T) {
	definition := card.RootDefinition()
	if _, issues := definition.CanonicalizeConfig(card.RawConfig{Present: true, Value: json.RawMessage(`{"shadow":"url(javascript:alert(1))"}`)}); len(issues) == 0 {
		t.Fatal("unsafe root shadow was accepted")
	}
}

func TestStrictDocumentDecodingAndStructure(t *testing.T) {
	registry := syntheticRegistry(syntheticDefinition("synthetic", card.InstallAppend))
	valid := []byte(`{"card_id":"strict","name":"Strict","root":{"id":"root","component_kind":"card","children":[{"id":"child","component_kind":"synthetic","config":{}}]}}`)
	document, issues := registry.DecodeDocument(valid)
	if len(issues) != 0 || !strings.Contains(string(document.Root.Children[0].Config), `"count":5`) {
		t.Fatalf("document = %#v, issues = %#v", document, issues)
	}
	assertDocumentIssue(t, registry, []byte(`{"card_id":"strict","name":"Strict","extra":true,"root":{"id":"root","component_kind":"card"}}`), "invalid_document")
	assertDocumentIssue(t, registry, []byte(`{"card_id":"strict","name":"Strict","root":null}`), "invalid_node")
	assertDocumentIssue(t, registry, []byte(`{"card_id":"strict","name":"Strict","root":{"id":"root","component_kind":"card","children":[{"id":"leaf","component_kind":"synthetic","children":[{"id":"hidden","component_kind":"synthetic"}]}]}}`), "children_not_allowed")
	_, recursiveIssues := registry.DecodeDocument([]byte(`{"card_id":"strict","name":"Strict","root":{"id":"root","component_kind":"card","children":[{"id":"leaf","component_kind":"synthetic","children":[{"id":"hidden","component_kind":"unknown"}]}]}}`))
	if !hasIssue(recursiveIssues, "children_not_allowed") || !hasIssue(recursiveIssues, "unknown_component_kind") {
		t.Fatalf("recursive issues = %#v", recursiveIssues)
	}
	assertDocumentIssue(t, registry, []byte(`{"card_id":"strict","name":"Strict","root":{"id":"root","component_kind":"synthetic"}}`), "invalid_root")
	assertDocumentIssue(t, registry, []byte(`{"card_id":"strict","name":"Strict","root":{"id":"root","component_kind":"card","children":[{"id":"root-2","component_kind":"card"}]}}`), "root_only")
	assertDocumentIssue(t, registry, []byte(`{"card_id":"strict","name":"Strict","root":{"id":"root","component_kind":"card","children":[{"id":"root","component_kind":"synthetic"}]}}`), "duplicate_id")
}

func TestInstallationPoliciesAndIDs(t *testing.T) {
	appendRegistry := syntheticRegistry(syntheticDefinition("synthetic", card.InstallAppend))
	document := documentWithChildren()
	var err error
	document, first, err := appendRegistry.InstallTemplate(document, card.ComponentTemplate{ComponentKind: "synthetic"})
	if err != nil || first.ID != "target-synthetic" {
		t.Fatalf("first = %#v, err = %v", first, err)
	}
	document, second, err := appendRegistry.InstallTemplate(document, card.ComponentTemplate{ComponentKind: "synthetic"})
	if err != nil || second.ID != "target-synthetic-2" {
		t.Fatalf("second = %#v, err = %v", second, err)
	}
	if _, _, err := appendRegistry.InstallTemplate(document, card.ComponentTemplate{ComponentKind: "synthetic", ComponentID: first.ID}); err == nil {
		t.Fatal("explicit ID collision succeeded")
	}
	if _, _, err := appendRegistry.InstallTemplate(document, card.ComponentTemplate{ComponentKind: "synthetic", ComponentID: " padded "}); err == nil {
		t.Fatal("noncanonical explicit ID succeeded")
	}
	document, explicit, err := appendRegistry.InstallTemplate(document, card.ComponentTemplate{ComponentKind: "synthetic", ComponentID: "kept_exactly"})
	if err != nil || explicit.ID != "kept_exactly" {
		t.Fatalf("explicit = %#v, err = %v", explicit, err)
	}

	replaceRegistry := syntheticRegistry(syntheticDefinition("synthetic", card.InstallReplaceKind))
	inserted, insertedNode, err := replaceRegistry.InstallTemplate(documentWithChildren(), card.ComponentTemplate{ComponentKind: "synthetic"})
	if err != nil || insertedNode.ID != "target-synthetic" || len(inserted.Root.Children) != 1 {
		t.Fatalf("replace insert = %#v, node = %#v, err = %v", inserted, insertedNode, err)
	}
	replaced, node, err := replaceRegistry.InstallTemplate(documentWithSynthetic("old-id"), card.ComponentTemplate{ComponentKind: "synthetic", Config: json.RawMessage(`{"name":"new"}`)})
	if err != nil || node.ID != "old-id" || len(replaced.Root.Children) != 1 || !strings.Contains(string(node.Config), `"name":"new"`) {
		t.Fatalf("replaced = %#v, node = %#v, err = %v", replaced, node, err)
	}
	replaced, node, err = replaceRegistry.InstallTemplate(documentWithSynthetic("old-id"), card.ComponentTemplate{ComponentKind: "synthetic", ComponentID: "new-id"})
	if err != nil || node.ID != "new-id" || replaced.Root.Children[0].ID != "new-id" {
		t.Fatalf("explicit replacement = %#v, node = %#v, err = %v", replaced, node, err)
	}
	ambiguous := documentWithSynthetic("one")
	ambiguous.Root.Children = append(ambiguous.Root.Children, ambiguous.Root.Children[0])
	ambiguous.Root.Children[1].ID = "two"
	if _, _, err := replaceRegistry.InstallTemplate(ambiguous, card.ComponentTemplate{ComponentKind: "synthetic"}); err == nil {
		t.Fatal("ambiguous replace_kind succeeded")
	}
}

func TestDefinitionAndRegistryInvariants(t *testing.T) {
	base := func(kind string) card.TypedDefinition[syntheticConfig] {
		return card.TypedDefinition[syntheticConfig]{Kind: kind, Label: "Test", Structure: card.StructureLeaf, Default: func() syntheticConfig { return syntheticConfig{Mode: "quiet"} }, Validate: func(syntheticConfig) []schema.Issue { return nil }, Render: func(card.Node, syntheticConfig, card.RenderContext) (card.Contribution, error) {
			return card.Contribution{}, nil
		}}
	}
	t.Run("duplicate definitions", func(t *testing.T) {
		definition := card.MustDefine(base("same"))
		if _, err := card.NewRegistry(card.RootDefinition(), definition, definition); err == nil {
			t.Fatal("duplicate definitions succeeded")
		}
	})
	t.Run("duplicate controls", func(t *testing.T) {
		definition := base("controls")
		control := card.StringControl("name", "content", "text", "Name", "name", func(c syntheticConfig) string { return c.Name }, func(c *syntheticConfig, value string) { c.Name = value })
		definition.Controls = []card.Control[syntheticConfig]{control, control}
		if _, err := card.Define(definition); err == nil {
			t.Fatal("duplicate controls succeeded")
		}
	})
	t.Run("duplicate properties", func(t *testing.T) {
		definition := base("properties")
		property := card.Property[syntheticConfig]{ID: "count", Kind: schema.PropertyNumber, Read: func(c syntheticConfig) (schema.PropertyValue, bool) {
			return schema.NumberValue(float64(c.Count)), true
		}}
		definition.Properties = []card.Property[syntheticConfig]{property, property}
		if _, err := card.Define(definition); err == nil {
			t.Fatal("duplicate properties succeeded")
		}
	})
	t.Run("duplicate roles", func(t *testing.T) {
		definition := base("roles")
		definition.Roles = []card.ComponentRole{card.RoleFormField, card.RoleFormField}
		if _, err := card.Define(definition); err == nil {
			t.Fatal("duplicate roles succeeded")
		}
	})
	t.Run("duplicate presets", func(t *testing.T) {
		definition := base("presets")
		preset := card.TypedPreset[syntheticConfig]{ID: "one", Name: "One", Config: syntheticConfig{Mode: "quiet"}}
		definition.Presets = []card.TypedPreset[syntheticConfig]{preset, preset}
		if _, err := card.Define(definition); err == nil {
			t.Fatal("duplicate presets succeeded")
		}
	})
	t.Run("empty generation capability", func(t *testing.T) {
		definition := base("generation")
		definition.Generation = &card.TypedGenerationDefinition[syntheticConfig]{}
		if _, err := card.Define(definition); err == nil {
			t.Fatal("empty generation capability succeeded")
		}
	})
	t.Run("noncanonical kind", func(t *testing.T) {
		definition := base("badKind")
		if _, err := card.Define(definition); err == nil {
			t.Fatal("noncanonical component kind succeeded")
		}
	})
}

func TestErasedMetadataReturnsDefensiveCopies(t *testing.T) {
	definition := syntheticDefinition("synthetic", card.InstallAppend)
	roles := definition.Roles()
	roles[0] = "changed"
	if definition.Roles()[0] != card.RoleFormField {
		t.Fatalf("roles mutated through returned slice: %#v", definition.Roles())
	}
	presets := definition.Presets()
	presets[0].Config[0] = 'x'
	if definition.Presets()[0].Config[0] == 'x' {
		t.Fatal("preset config mutated through returned slice")
	}
	registry := syntheticRegistry(definition)
	definitions := registry.Definitions()
	definitions[0] = definition
	if registry.Definitions()[0].Kind() != card.Kind {
		t.Fatal("catalog order mutated through returned slice")
	}
}

func TestDefaultCatalogRenderingAndCapabilities(t *testing.T) {
	registry := catalog.MustNew()
	document, issues := registry.CanonicalizeDocument(card.MustDefaultDocument(registry))
	if len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
	view, err := card.RenderDocumentWithOptions(document, registry, card.RenderOptions{ElementID: "preview", DOMIDPrefix: "scope"})
	if err != nil {
		t.Fatal(err)
	}
	html := view.Render()
	for _, marker := range []string{`data-component-kind="card"`, `data-component-kind="text"`, `id="scope-text-main-layer"`} {
		if !strings.Contains(html, marker) {
			t.Fatalf("render missing %q:\n%s", marker, html)
		}
	}
	shape, _ := registry.Lookup(card.KindShape)
	generation, ok := shape.Generation()
	if !ok || !generation.SupportsRandom() || generation.SupportsAI() {
		t.Fatalf("shape generation = %#v, ok = %v", generation, ok)
	}
	if _, issues := generation.CanonicalizeEnvelope(json.RawMessage(`{"component_kind":"shape","description":"shape","config":{"shape":"nope"}}`)); len(issues) == 0 {
		t.Fatal("invalid shape config was accepted")
	}
	if install, ok := background.Definition().Install(); !ok || install.Policy != card.InstallReplaceKind {
		t.Fatalf("background install = %#v, ok = %v", install, ok)
	}
}

func assertDocumentIssue(t *testing.T, registry *card.Registry, raw []byte, code string) {
	t.Helper()
	_, issues := registry.DecodeDocument(raw)
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issues = %#v, want code %q", issues, code)
}

func hasIssue(issues []schema.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func documentWithChildren() card.Document {
	root, _ := json.Marshal(card.DefaultRootConfig())
	return card.Document{CardID: "target", Name: "Target", Root: card.Node{ID: "root", ComponentKind: card.Kind, Config: root}}
}

func documentWithSynthetic(id string) card.Document {
	document := documentWithChildren()
	document.Root.Children = []card.Node{{ID: id, ComponentKind: "synthetic", Config: json.RawMessage(`{"name":"old","count":1,"enabled":true,"mode":"quiet"}`)}}
	return document
}
