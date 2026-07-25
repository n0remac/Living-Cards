package card

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

type ComponentStructure = schema.ComponentStructure

const (
	StructureRoot = schema.StructureRoot
	StructureLeaf = schema.StructureLeaf
)

type ComponentRole = schema.ComponentRole

const (
	RoleFormField     = schema.RoleFormField
	RoleFormSubmitter = schema.RoleFormSubmitter
)

type InstallPolicy = schema.InstallPolicy

const (
	InstallAppend      = schema.InstallAppend
	InstallReplaceKind = schema.InstallReplaceKind
)

var ErrUnsupportedOperation = errors.New("component operation is not supported")

type UnsupportedOperationError struct {
	ComponentKind string
	Operation     string
}

func (e UnsupportedOperationError) Error() string {
	return fmt.Sprintf("component %q does not support %s", e.ComponentKind, e.Operation)
}

func (e UnsupportedOperationError) Unwrap() error { return ErrUnsupportedOperation }

func NewUnsupportedOperationError(componentKind, operation string) error {
	return UnsupportedOperationError{ComponentKind: componentKind, Operation: operation}
}

type InstallSpec struct {
	Policy InstallPolicy `json:"policy"`
}

type RawConfig struct {
	Present bool
	Value   json.RawMessage
}

type ControlOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func Option(label, value string) ControlOption { return ControlOption{Label: label, Value: value} }

type ControlDescriptor struct {
	Trait    string          `json:"trait"`
	Control  string          `json:"control"`
	Kind     string          `json:"kind"`
	Label    string          `json:"label"`
	Property string          `json:"property,omitempty"`
	Value    json.RawMessage `json:"value,omitempty"`
	Options  []ControlOption `json:"options,omitempty"`
	Min      int             `json:"min,omitempty"`
	Max      int             `json:"max,omitempty"`
	Step     int             `json:"step,omitempty"`
}

type Control[T any] struct {
	ID           string
	ConfigPath   string
	ValueSchema  schema.ValueSchema
	OptionLabels map[string]string
	Suggestions  []ControlOption
	Descriptor   ControlDescriptor
	Read         func(T) json.RawMessage
	Apply        func(*T, json.RawMessage) error
}

type Property[T any] struct {
	ID   string
	Kind schema.PropertyKind
	Read func(T) (schema.PropertyValue, bool)
}

type TypedPreset[T any] struct {
	ID, Name, Description string
	Config                T
}
type TypedGenerationDefinition[T any] struct {
	SystemPrompt string
	Example      string
	Random       func(seed int64, level int) schema.GeneratedConfig[T]
}

type TypedDefinition[T any] struct {
	Kind        string
	Label       string
	Structure   ComponentStructure
	Default     func() T
	Normalize   func(T) T
	Validate    func(T) []schema.Issue
	Render      func(Node, T, RenderContext) (Contribution, error)
	ConfigRules []schema.FieldRule
	Controls    []Control[T]
	Properties  []Property[T]
	Roles       []ComponentRole
	Install     *InstallSpec
	Presets     []TypedPreset[T]
	Generation  *TypedGenerationDefinition[T]
}

type GenerationDefinition struct {
	systemPrompt         string
	example              string
	random               func(int64, int) (json.RawMessage, []schema.Issue)
	canonicalizeEnvelope func(json.RawMessage) (json.RawMessage, []schema.Issue)
}

func (g GenerationDefinition) SystemPrompt() string { return g.systemPrompt }
func (g GenerationDefinition) Example() string      { return g.example }
func (g GenerationDefinition) SupportsAI() bool     { return strings.TrimSpace(g.systemPrompt) != "" }
func (g GenerationDefinition) SupportsRandom() bool { return g.random != nil }
func (g GenerationDefinition) Random(seed int64, level int) (json.RawMessage, []schema.Issue) {
	if g.random == nil {
		return nil, []schema.Issue{{Path: "$", Code: "unsupported", Message: "random generation is not supported"}}
	}
	return g.random(seed, level)
}
func (g GenerationDefinition) CanonicalizeEnvelope(raw json.RawMessage) (json.RawMessage, []schema.Issue) {
	return g.canonicalizeEnvelope(raw)
}

type Definition struct {
	kind               string
	label              string
	structure          ComponentStructure
	roles              []ComponentRole
	configType         reflect.Type
	canonicalizeConfig func(RawConfig) (json.RawMessage, []schema.Issue)
	render             func(Node, RenderContext) (Contribution, error)
	describeControls   func(json.RawMessage) ([]ControlDescriptor, []schema.Issue)
	applyControl       func(json.RawMessage, string, json.RawMessage) (json.RawMessage, []schema.Issue)
	readProperty       func(json.RawMessage, string) (schema.PropertyValue, bool, []schema.Issue)
	controlIDs         []string
	propertyIDs        []string
	install            *InstallSpec
	presets            []LibraryItem
	generation         *GenerationDefinition
	componentSchema    schema.ComponentSchema
}

func (d Definition) Kind() string                  { return d.kind }
func (d Definition) Label() string                 { return d.label }
func (d Definition) Structure() ComponentStructure { return d.structure }
func (d Definition) ConfigType() reflect.Type      { return d.configType }
func (d Definition) Roles() []ComponentRole        { return append([]ComponentRole(nil), d.roles...) }
func (d Definition) ControlIDs() []string          { return append([]string(nil), d.controlIDs...) }
func (d Definition) PropertyIDs() []string         { return append([]string(nil), d.propertyIDs...) }
func (d Definition) PropertyKind(id string) (schema.PropertyKind, bool) {
	for _, property := range d.componentSchema.Properties {
		if property.ID == id {
			return property.Kind, true
		}
	}
	return "", false
}
func (d Definition) Schema() schema.ComponentSchema {
	return schema.CloneComponent(d.componentSchema)
}
func (d Definition) CanonicalizeConfig(raw RawConfig) (json.RawMessage, []schema.Issue) {
	return d.canonicalizeConfig(raw)
}
func (d Definition) Render(node Node, context RenderContext) (Contribution, error) {
	return d.render(node, context)
}
func (d Definition) Controls(raw json.RawMessage) ([]ControlDescriptor, []schema.Issue) {
	return d.describeControls(raw)
}
func (d Definition) ApplyControl(raw json.RawMessage, control string, value json.RawMessage) (json.RawMessage, []schema.Issue) {
	return d.applyControl(raw, control, value)
}
func (d Definition) ReadProperty(raw json.RawMessage, property string) (schema.PropertyValue, bool, []schema.Issue) {
	return d.readProperty(raw, property)
}
func (d Definition) Install() (InstallSpec, bool) {
	if d.install == nil {
		return InstallSpec{}, false
	}
	return *d.install, true
}
func (d Definition) Presets() []LibraryItem { return cloneLibraryItems(d.presets) }
func (d Definition) Generation() (GenerationDefinition, bool) {
	if d.generation == nil {
		return GenerationDefinition{}, false
	}
	return *d.generation, true
}
func (d Definition) HasRole(role ComponentRole) bool {
	for _, candidate := range d.roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func Define[T any](typed TypedDefinition[T]) (Definition, error) {
	declaredKind := typed.Kind
	typed.Label = strings.TrimSpace(typed.Label)
	if typed.Kind == "" {
		return Definition{}, fmt.Errorf("component kind is required")
	}
	if typed.Kind != strings.TrimSpace(declaredKind) {
		return Definition{}, fmt.Errorf("component kind %q must not contain surrounding whitespace", declaredKind)
	}
	if typed.Label == "" {
		return Definition{}, fmt.Errorf("component %q label is required", typed.Kind)
	}
	if !validSchemaIdentifier(typed.Kind) {
		return Definition{}, fmt.Errorf("component kind %q must use snake_case", typed.Kind)
	}
	if typed.Structure != StructureRoot && typed.Structure != StructureLeaf {
		return Definition{}, fmt.Errorf("component %q has invalid structure %q", typed.Kind, typed.Structure)
	}
	if typed.Default == nil || typed.Render == nil {
		return Definition{}, fmt.Errorf("component %q requires defaults and rendering", typed.Kind)
	}
	configType := reflect.TypeOf((*T)(nil)).Elem()
	if configType.Kind() != reflect.Struct {
		return Definition{}, fmt.Errorf("component %q config type must be a struct", typed.Kind)
	}
	if typed.Normalize == nil {
		typed.Normalize = func(value T) T { return value }
	}
	if typed.Validate == nil {
		typed.Validate = func(T) []schema.Issue { return nil }
	}
	configSchema, err := deriveConfigSchema(configType, typed.ConfigRules)
	if err != nil {
		return Definition{}, fmt.Errorf("component %q config schema: %w", typed.Kind, err)
	}

	controls := make(map[string]Control[T], len(typed.Controls))
	controlIDs := make([]string, 0, len(typed.Controls))
	for _, control := range typed.Controls {
		control.ID = strings.TrimSpace(control.ID)
		if control.ID == "" || !validSchemaIdentifier(control.ID) || control.Read == nil || control.Apply == nil || strings.TrimSpace(control.Descriptor.Kind) == "" || strings.TrimSpace(control.Descriptor.Label) == "" {
			return Definition{}, fmt.Errorf("component %q has invalid control", typed.Kind)
		}
		if _, exists := controls[control.ID]; exists {
			return Definition{}, fmt.Errorf("component %q has duplicate control %q", typed.Kind, control.ID)
		}
		if err := prepareControl(&control, configSchema); err != nil {
			return Definition{}, fmt.Errorf("component %q control %q: %w", typed.Kind, control.ID, err)
		}
		control.Descriptor.Control = control.ID
		control.Descriptor = cloneControlDescriptor(control.Descriptor)
		controls[control.ID], controlIDs = control, append(controlIDs, control.ID)
	}
	properties := make(map[string]Property[T], len(typed.Properties))
	propertyIDs := make([]string, 0, len(typed.Properties))
	for _, property := range typed.Properties {
		property.ID = strings.TrimSpace(property.ID)
		if property.ID == "" || !validSchemaIdentifier(property.ID) || property.Read == nil {
			return Definition{}, fmt.Errorf("component %q has invalid property", typed.Kind)
		}
		if property.Kind != schema.PropertyString && property.Kind != schema.PropertyNumber && property.Kind != schema.PropertyBool {
			return Definition{}, fmt.Errorf("component %q property %q has invalid kind", typed.Kind, property.ID)
		}
		if _, exists := properties[property.ID]; exists {
			return Definition{}, fmt.Errorf("component %q has duplicate property %q", typed.Kind, property.ID)
		}
		properties[property.ID], propertyIDs = property, append(propertyIDs, property.ID)
	}
	seenRoles := map[ComponentRole]bool{}
	for _, role := range typed.Roles {
		if (role != RoleFormField && role != RoleFormSubmitter) || seenRoles[role] {
			return Definition{}, fmt.Errorf("component %q has invalid or duplicate role %q", typed.Kind, role)
		}
		seenRoles[role] = true
	}
	if typed.Install != nil && typed.Install.Policy != InstallAppend && typed.Install.Policy != InstallReplaceKind {
		return Definition{}, fmt.Errorf("component %q has invalid install policy %q", typed.Kind, typed.Install.Policy)
	}
	if typed.Generation != nil {
		prompt, example := strings.TrimSpace(typed.Generation.SystemPrompt), strings.TrimSpace(typed.Generation.Example)
		if prompt == "" && typed.Generation.Random == nil {
			return Definition{}, fmt.Errorf("component %q has an empty generation capability", typed.Kind)
		}
		if prompt == "" && example != "" {
			return Definition{}, fmt.Errorf("component %q has an AI example without a prompt", typed.Kind)
		}
		if prompt != "" && example == "" {
			return Definition{}, fmt.Errorf("component %q AI generation requires an example", typed.Kind)
		}
	}

	validateAndEncode := func(config T) (json.RawMessage, []schema.Issue) {
		config = typed.Normalize(config)
		raw, err := json.Marshal(config)
		if err != nil {
			return nil, []schema.Issue{{Path: "config", Code: "encode_failed", Message: err.Error()}}
		}
		issues := schema.ValidateCanonicalJSON(raw, configSchema, "config")
		issues = append(issues, typed.Validate(config)...)
		if len(issues) > 0 {
			return nil, cloneIssues(issues)
		}
		return raw, nil
	}
	canonicalize := func(input RawConfig) (json.RawMessage, []schema.Issue) {
		config := typed.Default()
		if input.Present {
			trimmed := bytes.TrimSpace(input.Value)
			if bytes.Equal(trimmed, []byte("null")) {
				return nil, []schema.Issue{{Path: "config", Code: "null_config", Message: "config must be an object, not null"}}
			}
			if len(trimmed) == 0 {
				return nil, []schema.Issue{{Path: "config", Code: "invalid_json", Message: "config must be a JSON object"}}
			}
			if issues := schema.ValidateJSONStructure(trimmed, configSchema, "config"); len(issues) > 0 {
				return nil, cloneIssues(issues)
			}
			if err := decodeStrict(trimmed, &config); err != nil {
				return nil, []schema.Issue{{Path: "config", Code: "invalid_config", Message: err.Error()}}
			}
		}
		return validateAndEncode(config)
	}
	decodeCanonical := func(raw json.RawMessage) (T, []schema.Issue) {
		canonical, issues := canonicalize(RawConfig{Present: true, Value: raw})
		if len(issues) > 0 {
			var zero T
			return zero, issues
		}
		var value T
		if err := json.Unmarshal(canonical, &value); err != nil {
			var zero T
			return zero, []schema.Issue{{Path: "config", Code: "invalid_config", Message: err.Error()}}
		}
		return value, nil
	}

	d := Definition{kind: typed.Kind, label: typed.Label, structure: typed.Structure, roles: append([]ComponentRole(nil), typed.Roles...), configType: configType, canonicalizeConfig: canonicalize, controlIDs: controlIDs, propertyIDs: propertyIDs}
	d.render = func(node Node, context RenderContext) (Contribution, error) {
		value, issues := decodeCanonical(node.Config)
		if len(issues) > 0 {
			return Contribution{}, issuesError(typed.Kind, issues)
		}
		return typed.Render(node, value, context)
	}
	d.describeControls = func(raw json.RawMessage) ([]ControlDescriptor, []schema.Issue) {
		value, issues := decodeCanonical(raw)
		if len(issues) > 0 {
			return nil, issues
		}
		out := make([]ControlDescriptor, 0, len(controlIDs))
		for _, controlID := range controlIDs {
			control := controls[controlID]
			descriptor := cloneControlDescriptor(control.Descriptor)
			descriptor.Value = append(json.RawMessage(nil), control.Read(value)...)
			out = append(out, descriptor)
		}
		return out, nil
	}
	d.applyControl = func(raw json.RawMessage, id string, input json.RawMessage) (json.RawMessage, []schema.Issue) {
		value, issues := decodeCanonical(raw)
		if len(issues) > 0 {
			return nil, issues
		}
		control, ok := controls[strings.TrimSpace(id)]
		if !ok {
			return nil, []schema.Issue{{Path: "control", Code: "unsupported", Message: fmt.Sprintf("control %q is not supported for %s", id, typed.Kind)}}
		}
		if err := control.Apply(&value, input); err != nil {
			return nil, []schema.Issue{{Path: "control." + control.ID, Code: "invalid_value", Message: err.Error()}}
		}
		return validateAndEncode(value)
	}
	d.readProperty = func(raw json.RawMessage, id string) (schema.PropertyValue, bool, []schema.Issue) {
		value, issues := decodeCanonical(raw)
		if len(issues) > 0 {
			return schema.PropertyValue{}, false, issues
		}
		property, ok := properties[strings.TrimSpace(id)]
		if !ok {
			return schema.PropertyValue{}, false, nil
		}
		result, present := property.Read(value)
		if present && result.Kind != property.Kind {
			return schema.PropertyValue{}, false, []schema.Issue{{Path: "property." + id, Code: "invalid_kind", Message: "property returned the wrong value kind"}}
		}
		return result, present, nil
	}
	if typed.Install != nil {
		copy := *typed.Install
		d.install = &copy
	}
	seenPresets := map[string]bool{}
	for _, preset := range typed.Presets {
		preset.ID = strings.TrimSpace(preset.ID)
		if preset.ID == "" || seenPresets[preset.ID] {
			return Definition{}, fmt.Errorf("component %q has invalid or duplicate preset %q", typed.Kind, preset.ID)
		}
		seenPresets[preset.ID] = true
		raw, err := json.Marshal(preset.Config)
		if err != nil {
			return Definition{}, err
		}
		canonical, issues := canonicalize(RawConfig{Present: true, Value: raw})
		if len(issues) > 0 {
			return Definition{}, issuesError(typed.Kind, issues)
		}
		d.presets = append(d.presets, LibraryItem{ID: preset.ID, Name: preset.Name, ComponentKind: typed.Kind, Description: preset.Description, Config: canonical})
	}
	if typed.Generation != nil {
		generation := &GenerationDefinition{systemPrompt: strings.TrimSpace(typed.Generation.SystemPrompt), example: strings.TrimSpace(typed.Generation.Example)}
		generation.canonicalizeEnvelope = func(raw json.RawMessage) (json.RawMessage, []schema.Issue) {
			return canonicalizeGeneratedEnvelope(raw, typed.Kind, canonicalize)
		}
		if typed.Generation.Random != nil {
			randomGenerator := typed.Generation.Random
			generation.random = func(seed int64, level int) (json.RawMessage, []schema.Issue) {
				generated := randomGenerator(seed, level)
				configRaw, err := json.Marshal(generated.Config)
				if err != nil {
					return nil, []schema.Issue{{Path: "config", Code: "encode_failed", Message: err.Error()}}
				}
				envelopeRaw, err := json.Marshal(schema.GeneratedConfigEnvelope{ComponentKind: generated.ComponentKind, Description: generated.Description, Config: configRaw})
				if err != nil {
					return nil, []schema.Issue{{Path: "$", Code: "encode_failed", Message: err.Error()}}
				}
				return canonicalizeGeneratedEnvelope(envelopeRaw, typed.Kind, canonicalize)
			}
		}
		d.generation = generation
	}
	defaultRaw, issues := canonicalize(RawConfig{})
	if len(issues) > 0 {
		return Definition{}, fmt.Errorf("component %q defaults are invalid: %w", typed.Kind, issuesError(typed.Kind, issues))
	}
	descriptors, issues := d.Controls(defaultRaw)
	if len(issues) > 0 {
		return Definition{}, fmt.Errorf("component %q default controls are invalid: %w", typed.Kind, issuesError(typed.Kind, issues))
	}
	for _, descriptor := range descriptors {
		if len(descriptor.Value) == 0 || !json.Valid(descriptor.Value) {
			return Definition{}, fmt.Errorf("component %q control %q returned invalid JSON", typed.Kind, descriptor.Control)
		}
		control := controls[descriptor.Control]
		if controlIssues := schema.ValidateCanonicalJSON(descriptor.Value, control.ValueSchema, "control."+descriptor.Control+".value"); len(controlIssues) > 0 {
			return Definition{}, fmt.Errorf("component %q control %q default is invalid: %s", typed.Kind, descriptor.Control, controlIssues[0].Message)
		}
	}
	for _, propertyID := range propertyIDs {
		if _, _, propertyIssues := d.ReadProperty(defaultRaw, propertyID); len(propertyIssues) > 0 {
			return Definition{}, fmt.Errorf("component %q property %q is invalid: %w", typed.Kind, propertyID, issuesError(typed.Kind, propertyIssues))
		}
	}
	descriptorByID := make(map[string]ControlDescriptor, len(descriptors))
	for _, descriptor := range descriptors {
		descriptorByID[descriptor.Control] = descriptor
	}
	componentControls := make([]schema.ControlSchema, 0, len(controlIDs))
	for _, controlID := range controlIDs {
		componentControls = append(componentControls, exportControlSchema(controls[controlID], descriptorByID[controlID]))
	}
	componentProperties := make([]schema.PropertySchema, 0, len(propertyIDs))
	for _, propertyID := range propertyIDs {
		componentProperties = append(componentProperties, schema.PropertySchema{ID: propertyID, Kind: properties[propertyID].Kind})
	}
	componentPresets := make([]schema.PresetSchema, 0, len(d.presets))
	for _, preset := range d.presets {
		componentPresets = append(componentPresets, schema.PresetSchema{
			ID: preset.ID, Name: preset.Name, Description: preset.Description,
			Config: append(json.RawMessage(nil), preset.Config...),
		})
	}
	var installSchema *schema.InstallSchema
	if d.install != nil {
		installSchema = &schema.InstallSchema{Policy: d.install.Policy}
	}
	d.componentSchema = schema.ComponentSchema{
		Kind: typed.Kind, Label: typed.Label, Structure: typed.Structure,
		Config: schema.CloneValue(configSchema), Default: append(json.RawMessage(nil), defaultRaw...),
		Controls: componentControls, Properties: componentProperties,
		Roles:   append([]schema.ComponentRole(nil), typed.Roles...),
		Install: installSchema, Presets: componentPresets,
		Capabilities: schema.CapabilitySchema{
			Editable: len(componentControls) > 0, Installable: installSchema != nil,
			HasProperties: len(componentProperties) > 0, HasPresets: len(componentPresets) > 0,
			RandomGeneration: d.generation != nil && d.generation.SupportsRandom(),
			AIGeneration:     d.generation != nil && d.generation.SupportsAI(),
		},
	}
	return d, nil
}

func MustDefine[T any](typed TypedDefinition[T]) Definition {
	definition, err := Define(typed)
	if err != nil {
		panic(err)
	}
	return definition
}

func StringControl[T any](id, trait, kind, label, property string, get func(T) string, set func(*T, string)) Control[T] {
	return Control[T]{ID: id, ConfigPath: id, Descriptor: ControlDescriptor{Trait: trait, Kind: kind, Label: label, Property: property}, Read: func(v T) json.RawMessage { raw, _ := json.Marshal(get(v)); return raw }, Apply: func(v *T, raw json.RawMessage) error {
		var next string
		if err := decodeStrict(raw, &next); err != nil {
			return fmt.Errorf("value must be a string")
		}
		set(v, next)
		return nil
	}}
}
func IntControl[T any](id, trait, kind, label, property string, step int, get func(T) int, set func(*T, int)) Control[T] {
	return Control[T]{ID: id, ConfigPath: id, Descriptor: ControlDescriptor{Trait: trait, Kind: kind, Label: label, Property: property, Step: step}, Read: func(v T) json.RawMessage { raw, _ := json.Marshal(get(v)); return raw }, Apply: func(v *T, raw json.RawMessage) error {
		var next int
		if err := decodeStrict(raw, &next); err != nil {
			return fmt.Errorf("value must be an integer")
		}
		set(v, next)
		return nil
	}}
}

func WithOptionLabels[T any](control Control[T], options ...ControlOption) Control[T] {
	control.OptionLabels = make(map[string]string, len(options))
	for _, option := range options {
		control.OptionLabels[option.Value] = option.Label
	}
	return control
}

func WithSuggestions[T any](control Control[T], options ...ControlOption) Control[T] {
	control.Suggestions = append([]ControlOption(nil), options...)
	return control
}

// DecodeControlObject applies the same strict object rules used by component
// configs to structured control payloads.
func DecodeControlObject(raw json.RawMessage, target any) error {
	return decodeStrictObject(raw, target)
}

func decodeStrictObject(raw []byte, target any) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("value must be a JSON object")
	}
	return decodeStrict(trimmed, target)
}

func decodeStrict(raw []byte, target any) error {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return fmt.Errorf("value must not be null")
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}
func cloneControlDescriptor(value ControlDescriptor) ControlDescriptor {
	value.Value = append(json.RawMessage(nil), value.Value...)
	value.Options = append([]ControlOption(nil), value.Options...)
	return value
}
func cloneLibraryItems(items []LibraryItem) []LibraryItem {
	out := make([]LibraryItem, len(items))
	for i, item := range items {
		out[i] = item
		out[i].Config = append(json.RawMessage(nil), item.Config...)
	}
	return out
}
func cloneIssues(issues []schema.Issue) []schema.Issue {
	out := append([]schema.Issue(nil), issues...)
	for i := range out {
		out[i].Allowed = append([]string(nil), out[i].Allowed...)
	}
	return out
}
func issuesError(kind string, issues []schema.Issue) error {
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("invalid %s config at %s: %s", kind, issues[0].Path, issues[0].Message)
}

func canonicalizeGeneratedEnvelope(raw json.RawMessage, kind string, canonicalize func(RawConfig) (json.RawMessage, []schema.Issue)) (json.RawMessage, []schema.Issue) {
	var envelope schema.GeneratedConfigEnvelope
	if err := decodeStrictObject(raw, &envelope); err != nil {
		return nil, []schema.Issue{{Path: "$", Code: "invalid_json", Message: "response must be one strict JSON object: " + err.Error()}}
	}
	var issues []schema.Issue
	if envelope.ComponentKind != kind {
		issues = append(issues, schema.Issue{Path: "component_kind", Code: "invalid_component_kind", Message: "component_kind must be " + kind, Actual: envelope.ComponentKind, Allowed: []string{kind}})
	}
	envelope.Description = strings.TrimSpace(envelope.Description)
	if envelope.Description == "" {
		issues = append(issues, schema.Issue{Path: "description", Code: "required", Message: "description is required"})
	}
	if envelope.Config == nil {
		issues = append(issues, schema.Issue{Path: "config", Code: "required", Message: "config is required"})
	} else {
		config, configIssues := canonicalize(RawConfig{Present: true, Value: envelope.Config})
		issues = append(issues, configIssues...)
		envelope.Config = config
	}
	if len(issues) > 0 {
		return nil, issues
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, []schema.Issue{{Path: "$", Code: "encode_failed", Message: err.Error()}}
	}
	return encoded, nil
}

func validSchemaIdentifier(value string) bool {
	if value == "" {
		return false
	}
	previousUnderscore := false
	for index, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			previousUnderscore = false
		case index > 0 && char >= '0' && char <= '9':
			previousUnderscore = false
		case index > 0 && char == '_' && !previousUnderscore:
			previousUnderscore = true
		default:
			return false
		}
	}
	return !previousUnderscore
}
