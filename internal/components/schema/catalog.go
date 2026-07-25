package schema

import "encoding/json"

type ValueKind string

const (
	ValueString  ValueKind = "string"
	ValueBoolean ValueKind = "boolean"
	ValueInteger ValueKind = "integer"
	ValueNumber  ValueKind = "number"
	ValueObject  ValueKind = "object"
	ValueArray   ValueKind = "array"
)

type FieldFormat string

const (
	FormatNone             FieldFormat = ""
	FormatCSSColor         FieldFormat = "css_color"
	FormatOptionalCSSColor FieldFormat = "optional_css_color"
	FormatSafeToken        FieldFormat = "safe_token"
)

type ValueSchema struct {
	Kind      ValueKind         `json:"kind"`
	Fields    []FieldSchema     `json:"fields,omitempty"`
	Items     *ValueSchema      `json:"items,omitempty"`
	Enum      []json.RawMessage `json:"enum,omitempty"`
	Minimum   *float64          `json:"minimum,omitempty"`
	Maximum   *float64          `json:"maximum,omitempty"`
	MinLength *int              `json:"min_length,omitempty"`
	MaxLength *int              `json:"max_length,omitempty"`
	Format    FieldFormat       `json:"format,omitempty"`
	Nullable  bool              `json:"nullable"`
}

type FieldSchema struct {
	JSONName string      `json:"json_name"`
	Schema   ValueSchema `json:"schema"`
	Required bool        `json:"required"`
}

type ComponentStructure string

const (
	StructureRoot ComponentStructure = "root"
	StructureLeaf ComponentStructure = "leaf"
)

type ComponentRole string

const (
	RoleFormField     ComponentRole = "form_field"
	RoleFormSubmitter ComponentRole = "form_submitter"
)

type InstallPolicy string

const (
	InstallAppend      InstallPolicy = "append"
	InstallReplaceKind InstallPolicy = "replace_kind"
)

type ControlOptionSchema struct {
	Label string          `json:"label"`
	Value json.RawMessage `json:"value"`
}

type ControlSchema struct {
	ID          string                `json:"id"`
	ConfigPath  string                `json:"config_path,omitempty"`
	Trait       string                `json:"trait"`
	Kind        string                `json:"kind"`
	Label       string                `json:"label"`
	Property    string                `json:"property,omitempty"`
	ValueSchema ValueSchema           `json:"value_schema"`
	Default     json.RawMessage       `json:"default"`
	Options     []ControlOptionSchema `json:"options,omitempty"`
	Suggestions []ControlOptionSchema `json:"suggestions,omitempty"`
	Step        int                   `json:"step,omitempty"`
}

type PropertySchema struct {
	ID   string       `json:"id"`
	Kind PropertyKind `json:"kind"`
}

type InstallSchema struct {
	Policy InstallPolicy `json:"policy"`
}

type PresetSchema struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Config      json.RawMessage `json:"config"`
}

type CapabilitySchema struct {
	Editable         bool `json:"editable"`
	Installable      bool `json:"installable"`
	HasProperties    bool `json:"has_properties"`
	HasPresets       bool `json:"has_presets"`
	RandomGeneration bool `json:"random_generation"`
	AIGeneration     bool `json:"ai_generation"`
}

type ComponentSchema struct {
	Kind         string             `json:"kind"`
	Label        string             `json:"label"`
	Structure    ComponentStructure `json:"structure"`
	Config       ValueSchema        `json:"config"`
	Default      json.RawMessage    `json:"default"`
	Controls     []ControlSchema    `json:"controls,omitempty"`
	Properties   []PropertySchema   `json:"properties,omitempty"`
	Roles        []ComponentRole    `json:"roles,omitempty"`
	Install      *InstallSchema     `json:"install,omitempty"`
	Presets      []PresetSchema     `json:"presets,omitempty"`
	Capabilities CapabilitySchema   `json:"capabilities"`
}

type CatalogSchema struct {
	Components []ComponentSchema `json:"components"`
}

type RuleKind string

const (
	RuleEnum         RuleKind = "enum"
	RuleRange        RuleKind = "range"
	RuleStringLength RuleKind = "string_length"
	RuleFormat       RuleKind = "format"
)

type FieldRule struct {
	Path      string
	Kind      RuleKind
	Enum      []json.RawMessage
	Minimum   *float64
	Maximum   *float64
	MinLength *int
	MaxLength *int
	Format    FieldFormat
}

type RuleScalar interface {
	~string | ~bool | ~int | ~float64
}

func Enum[T RuleScalar](path string, values ...T) FieldRule {
	encoded := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		encoded = append(encoded, raw)
	}
	return FieldRule{Path: path, Kind: RuleEnum, Enum: encoded}
}

func IntegerRange(path string, minimum, maximum int) FieldRule {
	minimumValue, maximumValue := float64(minimum), float64(maximum)
	return FieldRule{Path: path, Kind: RuleRange, Minimum: &minimumValue, Maximum: &maximumValue}
}

func NumberRange(path string, minimum, maximum float64) FieldRule {
	return FieldRule{Path: path, Kind: RuleRange, Minimum: &minimum, Maximum: &maximum}
}

func StringLength(path string, minimum, maximum int) FieldRule {
	return FieldRule{Path: path, Kind: RuleStringLength, MinLength: &minimum, MaxLength: &maximum}
}

func StringMinLength(path string, minimum int) FieldRule {
	return FieldRule{Path: path, Kind: RuleStringLength, MinLength: &minimum}
}

func StringMaxLength(path string, maximum int) FieldRule {
	return FieldRule{Path: path, Kind: RuleStringLength, MaxLength: &maximum}
}

func StringFormat(path string, format FieldFormat) FieldRule {
	return FieldRule{Path: path, Kind: RuleFormat, Format: format}
}

func CloneCatalog(value CatalogSchema) CatalogSchema {
	out := CatalogSchema{Components: make([]ComponentSchema, len(value.Components))}
	for index, component := range value.Components {
		out.Components[index] = CloneComponent(component)
	}
	return out
}

func CloneComponent(value ComponentSchema) ComponentSchema {
	out := value
	out.Config = CloneValue(value.Config)
	out.Default = cloneRaw(value.Default)
	out.Controls = make([]ControlSchema, len(value.Controls))
	for index, control := range value.Controls {
		out.Controls[index] = cloneControl(control)
	}
	out.Properties = append([]PropertySchema(nil), value.Properties...)
	out.Roles = append([]ComponentRole(nil), value.Roles...)
	if value.Install != nil {
		install := *value.Install
		out.Install = &install
	}
	out.Presets = make([]PresetSchema, len(value.Presets))
	for index, preset := range value.Presets {
		out.Presets[index] = preset
		out.Presets[index].Config = cloneRaw(preset.Config)
	}
	return out
}

func CloneValue(value ValueSchema) ValueSchema {
	out := value
	out.Fields = make([]FieldSchema, len(value.Fields))
	for index, field := range value.Fields {
		out.Fields[index] = field
		out.Fields[index].Schema = CloneValue(field.Schema)
	}
	if value.Items != nil {
		items := CloneValue(*value.Items)
		out.Items = &items
	}
	out.Enum = make([]json.RawMessage, len(value.Enum))
	for index, candidate := range value.Enum {
		out.Enum[index] = cloneRaw(candidate)
	}
	if value.Minimum != nil {
		minimum := *value.Minimum
		out.Minimum = &minimum
	}
	if value.Maximum != nil {
		maximum := *value.Maximum
		out.Maximum = &maximum
	}
	if value.MinLength != nil {
		minimum := *value.MinLength
		out.MinLength = &minimum
	}
	if value.MaxLength != nil {
		maximum := *value.MaxLength
		out.MaxLength = &maximum
	}
	return out
}

func cloneControl(value ControlSchema) ControlSchema {
	out := value
	out.ValueSchema = CloneValue(value.ValueSchema)
	out.Default = cloneRaw(value.Default)
	out.Options = cloneControlOptions(value.Options)
	out.Suggestions = cloneControlOptions(value.Suggestions)
	return out
}

func cloneControlOptions(values []ControlOptionSchema) []ControlOptionSchema {
	out := make([]ControlOptionSchema, len(values))
	for index, value := range values {
		out[index] = value
		out[index].Value = cloneRaw(value.Value)
	}
	return out
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), value...)
}
