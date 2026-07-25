// Package tsgen emits TypeScript contracts from the language-neutral component
// catalog schema. It deliberately does not import the component registry or any
// concrete component package.
package tsgen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

const (
	TypesFile    = "web/src/generated/card-types.generated.ts"
	MetadataFile = "web/src/generated/component-catalog.generated.ts"
)

type Files map[string][]byte

type typeDeclaration struct {
	Name   string
	Fields []typeField
}

type typeField struct {
	JSONName string
	Type     string
}

type generator struct {
	catalog      schema.CatalogSchema
	declarations []typeDeclaration
	names        map[string]string
}

var kindPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

func Generate(catalog schema.CatalogSchema) (Files, error) {
	g := &generator{catalog: schema.CloneCatalog(catalog), names: reservedTypeScriptNames()}
	if err := g.validateAndCollect(); err != nil {
		return nil, err
	}
	types, err := g.generateTypes()
	if err != nil {
		return nil, err
	}
	metadata, err := g.generateMetadata()
	if err != nil {
		return nil, err
	}
	return Files{TypesFile: types, MetadataFile: metadata}, nil
}

func (g *generator) validateAndCollect() error {
	if len(g.catalog.Components) == 0 {
		return fmt.Errorf("catalog has no components")
	}
	seenKinds := map[string]bool{}
	rootCount := 0
	for _, component := range g.catalog.Components {
		if !kindPattern.MatchString(component.Kind) {
			return fmt.Errorf("component kind %q must use lowercase snake_case", component.Kind)
		}
		if seenKinds[component.Kind] {
			return fmt.Errorf("duplicate component kind %q", component.Kind)
		}
		seenKinds[component.Kind] = true
		if component.Structure == schema.StructureRoot {
			rootCount++
		} else if component.Structure != schema.StructureLeaf {
			return fmt.Errorf("component %q has unsupported structure %q", component.Kind, component.Structure)
		}
		if component.Config.Kind != schema.ValueObject {
			return fmt.Errorf("component %q config must be an object", component.Kind)
		}
		if issues := schema.ValidateCanonicalJSON(component.Default, component.Config, "components."+component.Kind+".default"); len(issues) > 0 {
			return fmt.Errorf("component %q has invalid default: %s", component.Kind, issues[0].Message)
		}
		name := pascalCase(component.Kind) + "Config"
		if err := g.collectObject(name, component.Config); err != nil {
			return fmt.Errorf("component %q: %w", component.Kind, err)
		}
	}
	if rootCount != 1 {
		return fmt.Errorf("catalog must have exactly one root component")
	}
	return nil
}

func (g *generator) collectObject(name string, value schema.ValueSchema) error {
	if value.Kind != schema.ValueObject {
		return fmt.Errorf("%s must describe an object", name)
	}
	for _, generatedName := range []string{name, name + "Input"} {
		if owner, exists := g.names[generatedName]; exists {
			return fmt.Errorf("generated TypeScript name %q for %s collides with %s", generatedName, name, owner)
		}
		g.names[generatedName] = name
	}
	declaration := typeDeclaration{Name: name, Fields: make([]typeField, 0, len(value.Fields))}
	seenFields := map[string]bool{}
	for _, field := range value.Fields {
		if !kindPattern.MatchString(field.JSONName) {
			return fmt.Errorf("field %q must use lowercase snake_case", field.JSONName)
		}
		if seenFields[field.JSONName] {
			return fmt.Errorf("%s has duplicate field %q", name, field.JSONName)
		}
		if !field.Required {
			return fmt.Errorf("%s field %q must be required in canonical config", name, field.JSONName)
		}
		seenFields[field.JSONName] = true
		fieldType, err := g.typeExpression(name+pascalCase(field.JSONName), field.Schema)
		if err != nil {
			return err
		}
		declaration.Fields = append(declaration.Fields, typeField{JSONName: field.JSONName, Type: fieldType})
	}
	g.declarations = append(g.declarations, declaration)
	return nil
}

func (g *generator) typeExpression(nestedName string, value schema.ValueSchema) (string, error) {
	if value.Nullable {
		return "", fmt.Errorf("%s is nullable; generated card values must be non-null", nestedName)
	}
	switch value.Kind {
	case schema.ValueString:
		return scalarType("string", value.Enum)
	case schema.ValueBoolean:
		return scalarType("boolean", value.Enum)
	case schema.ValueInteger, schema.ValueNumber:
		return scalarType("number", value.Enum)
	case schema.ValueObject:
		if err := g.collectObject(nestedName, value); err != nil {
			return "", err
		}
		return nestedName, nil
	case schema.ValueArray:
		if value.Items == nil {
			return "", fmt.Errorf("%s array has no item schema", nestedName)
		}
		itemName := nestedName + "Item"
		itemType, err := g.typeExpression(itemName, *value.Items)
		if err != nil {
			return "", err
		}
		return "Array<" + itemType + ">", nil
	default:
		return "", fmt.Errorf("%s has unsupported value kind %q", nestedName, value.Kind)
	}
}

func scalarType(fallback string, values []json.RawMessage) (string, error) {
	if len(values) == 0 {
		return fallback, nil
	}
	literals := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		var decoded any
		decoder := json.NewDecoder(bytes.NewReader(value))
		decoder.UseNumber()
		if err := decoder.Decode(&decoded); err != nil {
			return "", fmt.Errorf("invalid enum JSON %q: %w", value, err)
		}
		var literal string
		switch candidate := decoded.(type) {
		case string:
			literal = strconv.Quote(candidate)
		case bool:
			literal = strconv.FormatBool(candidate)
		case json.Number:
			literal = candidate.String()
		default:
			return "", fmt.Errorf("enum value %q is not scalar", value)
		}
		if !seen[literal] {
			seen[literal] = true
			literals = append(literals, literal)
		}
	}
	return strings.Join(literals, " | "), nil
}

func (g *generator) generateTypes() ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(generatedHeader())
	out.WriteString("// Canonical types represent validated backend output. Input types mirror\n")
	out.WriteString("// default-overlay decoding: fields may be omitted recursively, but never null.\n\n")

	for _, declaration := range g.declarations {
		writeInterface(&out, declaration, false)
		writeInterface(&out, declaration, true)
	}

	allKinds := componentKinds(g.catalog.Components, func(schema.ComponentSchema) bool { return true })
	rootKinds := componentKinds(g.catalog.Components, func(c schema.ComponentSchema) bool { return c.Structure == schema.StructureRoot })
	leafKinds := componentKinds(g.catalog.Components, func(c schema.ComponentSchema) bool { return c.Structure == schema.StructureLeaf })
	installableKinds := componentKinds(g.catalog.Components, func(c schema.ComponentSchema) bool { return c.Capabilities.Installable })
	presetKinds := componentKinds(g.catalog.Components, func(c schema.ComponentSchema) bool { return c.Capabilities.HasPresets })
	generatedKinds := componentKinds(g.catalog.Components, func(c schema.ComponentSchema) bool {
		return c.Capabilities.RandomGeneration || c.Capabilities.AIGeneration
	})
	aiKinds := componentKinds(g.catalog.Components, func(c schema.ComponentSchema) bool { return c.Capabilities.AIGeneration })

	writeKindType(&out, "ComponentKind", allKinds)
	writeKindType(&out, "RootComponentKind", rootKinds)
	writeKindType(&out, "LeafComponentKind", leafKinds)
	writeKindType(&out, "InstallableComponentKind", installableKinds)
	writeKindType(&out, "PresetComponentKind", presetKinds)
	writeKindType(&out, "GeneratedComponentKind", generatedKinds)
	writeKindType(&out, "AIGeneratableComponentKind", aiKinds)

	out.WriteString("export interface ComponentConfigMap {\n")
	for _, component := range g.catalog.Components {
		fmt.Fprintf(&out, "  %s: %sConfig;\n", quoteTS(component.Kind), pascalCase(component.Kind))
	}
	out.WriteString("}\n\n")
	out.WriteString("export interface ComponentConfigInputMap {\n")
	for _, component := range g.catalog.Components {
		fmt.Fprintf(&out, "  %s: %sConfigInput;\n", quoteTS(component.Kind), pascalCase(component.Kind))
	}
	out.WriteString("}\n\n")
	out.WriteString("export type ComponentConfig<K extends ComponentKind = ComponentKind> = ComponentConfigMap[K];\n")
	out.WriteString("export type ComponentConfigInput<K extends ComponentKind = ComponentKind> = ComponentConfigInputMap[K];\n\n")

	out.WriteString("export type LeafComponentNode<K extends LeafComponentKind = LeafComponentKind> = {\n")
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"id\": string;\n")
	out.WriteString("    \"component_kind\": P;\n")
	out.WriteString("    \"config\": ComponentConfigMap[P];\n")
	out.WriteString("    \"children\"?: never;\n")
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
	out.WriteString("export type RootComponentNode<K extends RootComponentKind = RootComponentKind> = {\n")
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"id\": string;\n")
	out.WriteString("    \"component_kind\": P;\n")
	out.WriteString("    \"config\": ComponentConfigMap[P];\n")
	out.WriteString("    \"children\"?: Array<LeafComponentNode>;\n")
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
	out.WriteString("export type ComponentNode<K extends ComponentKind = ComponentKind> =\n")
	out.WriteString("  K extends RootComponentKind ? RootComponentNode<K> :\n")
	out.WriteString("  K extends LeafComponentKind ? LeafComponentNode<K> : never;\n\n")

	out.WriteString("export type LeafComponentNodeInput<K extends LeafComponentKind = LeafComponentKind> = {\n")
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"id\": string;\n")
	out.WriteString("    \"component_kind\": P;\n")
	out.WriteString("    \"config\"?: ComponentConfigInputMap[P];\n")
	out.WriteString("    \"children\"?: never;\n")
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
	out.WriteString("export type RootComponentNodeInput<K extends RootComponentKind = RootComponentKind> = {\n")
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"id\": string;\n")
	out.WriteString("    \"component_kind\": P;\n")
	out.WriteString("    \"config\"?: ComponentConfigInputMap[P];\n")
	out.WriteString("    \"children\"?: Array<LeafComponentNodeInput>;\n")
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
	out.WriteString("export type ComponentNodeInput<K extends ComponentKind = ComponentKind> =\n")
	out.WriteString("  K extends RootComponentKind ? RootComponentNodeInput<K> :\n")
	out.WriteString("  K extends LeafComponentKind ? LeafComponentNodeInput<K> : never;\n\n")

	out.WriteString("export interface CardDocument {\n")
	out.WriteString("  \"card_id\": string;\n")
	out.WriteString("  \"name\": string;\n")
	out.WriteString("  \"root\": RootComponentNode;\n")
	out.WriteString("}\n\n")
	out.WriteString("export interface CardDocumentInput {\n")
	out.WriteString("  \"card_id\": string;\n")
	out.WriteString("  \"name\": string;\n")
	out.WriteString("  \"root\": RootComponentNodeInput;\n")
	out.WriteString("}\n\n")

	writeCorrelatedType(&out, "ComponentTemplate", "InstallableComponentKind", false, true)
	writeCorrelatedType(&out, "ComponentTemplateInput", "InstallableComponentKind", true, true)
	writeLibraryType(&out)
	writeGeneratedType(&out, "GeneratedConfigEnvelope", "GeneratedComponentKind", false)
	writeGeneratedType(&out, "GeneratedConfigEnvelopeInput", "AIGeneratableComponentKind", true)

	return out.Bytes(), nil
}

func (g *generator) generateMetadata() ([]byte, error) {
	raw, err := json.MarshalIndent(g.catalog.Components, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal catalog metadata: %w", err)
	}
	var out bytes.Buffer
	out.WriteString(generatedHeader())
	out.WriteString("import type { ComponentKind, GeneratedComponentKind } from \"./card-types.generated\";\n\n")
	out.WriteString("export const componentCatalog = ")
	out.Write(raw)
	out.WriteString(" as const;\n\n")
	fmt.Fprintf(&out, "export const componentKinds = %s as const satisfies readonly ComponentKind[];\n", stringSlice(componentKinds(g.catalog.Components, func(schema.ComponentSchema) bool {
		return true
	})))
	fmt.Fprintf(&out, "export const generatedComponentKinds = %s as const satisfies readonly GeneratedComponentKind[];\n", stringSlice(componentKinds(g.catalog.Components, func(component schema.ComponentSchema) bool {
		return component.Capabilities.RandomGeneration || component.Capabilities.AIGeneration
	})))
	out.WriteString("export type ComponentCatalogMetadata = typeof componentCatalog;\n")
	return out.Bytes(), nil
}

func writeInterface(out *bytes.Buffer, declaration typeDeclaration, input bool) {
	name := declaration.Name
	if input {
		name += "Input"
	}
	fmt.Fprintf(out, "export interface %s {\n", name)
	for _, field := range declaration.Fields {
		fieldType := field.Type
		if input {
			fieldType = inputType(fieldType)
		}
		optional := ""
		if input {
			optional = "?"
		}
		fmt.Fprintf(out, "  %s%s: %s;\n", quoteTS(field.JSONName), optional, fieldType)
	}
	out.WriteString("}\n\n")
}

func inputType(value string) string {
	if strings.HasPrefix(value, "Array<") && strings.HasSuffix(value, ">") {
		item := strings.TrimSuffix(strings.TrimPrefix(value, "Array<"), ">")
		return "Array<" + inputType(item) + ">"
	}
	if isNamedType(value) {
		return value + "Input"
	}
	return value
}

func isNamedType(value string) bool {
	if value == "string" || value == "number" || value == "boolean" || strings.Contains(value, " | ") {
		return false
	}
	return value != "" && unicode.IsUpper(rune(value[0]))
}

func writeKindType(out *bytes.Buffer, name string, kinds []string) {
	fmt.Fprintf(out, "export type %s = %s;\n", name, union(kinds))
}

func writeCorrelatedType(out *bytes.Buffer, name, kindType string, input, componentID bool) {
	fmt.Fprintf(out, "export type %s<K extends %s = %s> = {\n", name, kindType, kindType)
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"component_kind\": P;\n")
	if componentID {
		out.WriteString("    \"component_id\"?: string;\n")
	}
	optional := ""
	configMap := "ComponentConfigMap"
	if input {
		optional = "?"
		configMap = "ComponentConfigInputMap"
	}
	fmt.Fprintf(out, "    \"config\"%s: %s[P];\n", optional, configMap)
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
}

func writeLibraryType(out *bytes.Buffer) {
	out.WriteString("export type ComponentLibraryItem<K extends ComponentKind = ComponentKind> = {\n")
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"id\": string;\n")
	out.WriteString("    \"name\": string;\n")
	out.WriteString("    \"component_kind\": P;\n")
	out.WriteString("    \"description\": string;\n")
	out.WriteString("    \"config\": ComponentConfigMap[P];\n")
	out.WriteString("    \"saved\"?: boolean;\n")
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
	out.WriteString("export type PresetLibraryItem<K extends PresetComponentKind = PresetComponentKind> = ComponentLibraryItem<K>;\n\n")
}

func writeGeneratedType(out *bytes.Buffer, name, kindType string, input bool) {
	fmt.Fprintf(out, "export type %s<K extends %s = %s> = {\n", name, kindType, kindType)
	out.WriteString("  [P in K]: {\n")
	out.WriteString("    \"component_kind\": P;\n")
	out.WriteString("    \"description\": string;\n")
	configMap := "ComponentConfigMap"
	if input {
		configMap = "ComponentConfigInputMap"
	}
	fmt.Fprintf(out, "    \"config\": %s[P];\n", configMap)
	out.WriteString("  }\n")
	out.WriteString("}[K];\n\n")
}

func componentKinds(components []schema.ComponentSchema, include func(schema.ComponentSchema) bool) []string {
	var out []string
	for _, component := range components {
		if include(component) {
			out = append(out, component.Kind)
		}
	}
	return out
}

func union(values []string) string {
	if len(values) == 0 {
		return "never"
	}
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = quoteTS(value)
	}
	return strings.Join(quoted, " | ")
}

func quoteTS(value string) string {
	return strconv.Quote(value)
}

func stringSlice(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = quoteTS(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func pascalCase(value string) string {
	var out strings.Builder
	for _, part := range strings.Split(value, "_") {
		if part == "" {
			continue
		}
		runes := []rune(part)
		out.WriteRune(unicode.ToUpper(runes[0]))
		out.WriteString(string(runes[1:]))
	}
	return out.String()
}

func generatedHeader() string {
	return "/* Code generated by cmd/cardtypes; DO NOT EDIT. */\n\n"
}

func reservedTypeScriptNames() map[string]string {
	names := map[string]string{}
	for _, name := range []string{
		"ComponentKind", "RootComponentKind", "LeafComponentKind", "InstallableComponentKind",
		"PresetComponentKind", "GeneratedComponentKind", "AIGeneratableComponentKind",
		"ComponentConfigMap", "ComponentConfigInputMap", "ComponentConfig", "ComponentConfigInput",
		"LeafComponentNode", "RootComponentNode", "ComponentNode",
		"LeafComponentNodeInput", "RootComponentNodeInput", "ComponentNodeInput",
		"CardDocument", "CardDocumentInput", "ComponentTemplate", "ComponentTemplateInput",
		"ComponentLibraryItem", "PresetLibraryItem", "GeneratedConfigEnvelope", "GeneratedConfigEnvelopeInput",
	} {
		names[name] = "generated catalog contract " + name
	}
	return names
}

func SortedPaths(files Files) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
