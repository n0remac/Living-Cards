package card

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

var (
	jsonMarshalerType   = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textMarshalerType   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func deriveConfigSchema(configType reflect.Type, rules []schema.FieldRule) (schema.ValueSchema, error) {
	value, err := deriveValueSchema(configType, nil, "config")
	if err != nil {
		return schema.ValueSchema{}, err
	}
	if value.Kind != schema.ValueObject {
		return schema.ValueSchema{}, fmt.Errorf("config type must be an object")
	}
	if err := applyConfigRules(&value, rules); err != nil {
		return schema.ValueSchema{}, err
	}
	return value, nil
}

func deriveValueSchema(valueType reflect.Type, stack []reflect.Type, path string) (schema.ValueSchema, error) {
	if hasCustomEncoding(valueType) {
		return schema.ValueSchema{}, fmt.Errorf("%s type %s uses custom JSON or text encoding", path, valueType)
	}
	switch valueType.Kind() {
	case reflect.String:
		return schema.ValueSchema{Kind: schema.ValueString}, nil
	case reflect.Bool:
		return schema.ValueSchema{Kind: schema.ValueBoolean}, nil
	case reflect.Int:
		return schema.ValueSchema{Kind: schema.ValueInteger}, nil
	case reflect.Float64:
		return schema.ValueSchema{Kind: schema.ValueNumber}, nil
	case reflect.Struct:
		for _, active := range stack {
			if active == valueType {
				return schema.ValueSchema{}, fmt.Errorf("%s type %s is recursive", path, valueType)
			}
		}
		nextStack := append(append([]reflect.Type(nil), stack...), valueType)
		out := schema.ValueSchema{Kind: schema.ValueObject}
		seen := map[string]bool{}
		for index := 0; index < valueType.NumField(); index++ {
			field := valueType.Field(index)
			if field.Anonymous {
				return schema.ValueSchema{}, fmt.Errorf("%s field %s must not be embedded", path, field.Name)
			}
			if !field.IsExported() {
				return schema.ValueSchema{}, fmt.Errorf("%s field %s must be exported", path, field.Name)
			}
			jsonName, err := configJSONName(field)
			if err != nil {
				return schema.ValueSchema{}, fmt.Errorf("%s field %s: %w", path, field.Name, err)
			}
			if seen[jsonName] {
				return schema.ValueSchema{}, fmt.Errorf("%s has duplicate JSON field %q", path, jsonName)
			}
			seen[jsonName] = true
			child, err := deriveValueSchema(field.Type, nextStack, path+"."+jsonName)
			if err != nil {
				return schema.ValueSchema{}, err
			}
			out.Fields = append(out.Fields, schema.FieldSchema{JSONName: jsonName, Schema: child, Required: true})
		}
		return out, nil
	case reflect.Slice:
		items, err := deriveValueSchema(valueType.Elem(), stack, path+"[]")
		if err != nil {
			return schema.ValueSchema{}, err
		}
		return schema.ValueSchema{Kind: schema.ValueArray, Items: &items}, nil
	case reflect.Map:
		return schema.ValueSchema{}, fmt.Errorf("%s maps are not supported", path)
	case reflect.Pointer:
		return schema.ValueSchema{}, fmt.Errorf("%s pointers are not supported", path)
	case reflect.Interface:
		return schema.ValueSchema{}, fmt.Errorf("%s interfaces are not supported", path)
	default:
		return schema.ValueSchema{}, fmt.Errorf("%s type %s is not supported", path, valueType)
	}
}

func hasCustomEncoding(valueType reflect.Type) bool {
	candidates := []reflect.Type{valueType}
	if valueType.Kind() != reflect.Pointer {
		candidates = append(candidates, reflect.PointerTo(valueType))
	}
	for _, candidate := range candidates {
		if candidate.Implements(jsonMarshalerType) ||
			candidate.Implements(jsonUnmarshalerType) ||
			candidate.Implements(textMarshalerType) ||
			candidate.Implements(textUnmarshalerType) {
			return true
		}
	}
	return false
}

func configJSONName(field reflect.StructField) (string, error) {
	tag, ok := field.Tag.Lookup("json")
	if !ok || strings.TrimSpace(tag) == "" {
		return "", fmt.Errorf("an explicit json tag is required")
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "" || parts[0] == "-" {
		return "", fmt.Errorf("json field must have a canonical name")
	}
	if len(parts) != 1 {
		return "", fmt.Errorf("json tag options such as omitempty are not supported")
	}
	if !validSchemaIdentifier(parts[0]) {
		return "", fmt.Errorf("json field %q must use lowercase snake_case", parts[0])
	}
	return parts[0], nil
}

func applyConfigRules(root *schema.ValueSchema, rules []schema.FieldRule) error {
	seen := map[string]bool{}
	for _, rule := range rules {
		rule.Path = strings.TrimSpace(rule.Path)
		if rule.Path == "" {
			return fmt.Errorf("config rule path is required")
		}
		key := rule.Path + "\x00" + string(rule.Kind)
		if seen[key] {
			return fmt.Errorf("duplicate %s rule for config.%s", rule.Kind, rule.Path)
		}
		seen[key] = true
		target, err := findConfigField(root, rule.Path)
		if err != nil {
			return err
		}
		switch rule.Kind {
		case schema.RuleEnum:
			if target.Kind != schema.ValueString && target.Kind != schema.ValueBoolean && target.Kind != schema.ValueInteger && target.Kind != schema.ValueNumber {
				return fmt.Errorf("enum rule config.%s targets non-scalar %s", rule.Path, target.Kind)
			}
			if len(rule.Enum) == 0 {
				return fmt.Errorf("enum rule config.%s requires values", rule.Path)
			}
			unique := map[string]bool{}
			target.Enum = make([]json.RawMessage, 0, len(rule.Enum))
			for _, candidate := range rule.Enum {
				if issues := schema.ValidateJSONStructure(candidate, schema.ValueSchema{Kind: target.Kind}, "config."+rule.Path); len(issues) > 0 {
					return fmt.Errorf("enum rule config.%s has invalid value: %s", rule.Path, issues[0].Message)
				}
				var compact bytes.Buffer
				if err := json.Compact(&compact, candidate); err != nil {
					return fmt.Errorf("enum rule config.%s has invalid JSON: %w", rule.Path, err)
				}
				value := compact.String()
				if unique[value] {
					return fmt.Errorf("enum rule config.%s has duplicate value %s", rule.Path, value)
				}
				unique[value] = true
				target.Enum = append(target.Enum, append(json.RawMessage(nil), candidate...))
			}
		case schema.RuleRange:
			if target.Kind != schema.ValueInteger && target.Kind != schema.ValueNumber {
				return fmt.Errorf("range rule config.%s targets non-numeric %s", rule.Path, target.Kind)
			}
			if rule.Minimum == nil || rule.Maximum == nil || math.IsNaN(*rule.Minimum) || math.IsNaN(*rule.Maximum) || math.IsInf(*rule.Minimum, 0) || math.IsInf(*rule.Maximum, 0) || *rule.Minimum > *rule.Maximum {
				return fmt.Errorf("range rule config.%s has invalid bounds", rule.Path)
			}
			if target.Kind == schema.ValueInteger && (math.Trunc(*rule.Minimum) != *rule.Minimum || math.Trunc(*rule.Maximum) != *rule.Maximum) {
				return fmt.Errorf("range rule config.%s requires integer bounds", rule.Path)
			}
			minimum, maximum := *rule.Minimum, *rule.Maximum
			target.Minimum, target.Maximum = &minimum, &maximum
		case schema.RuleStringLength:
			if target.Kind != schema.ValueString {
				return fmt.Errorf("string length rule config.%s targets %s", rule.Path, target.Kind)
			}
			if rule.MinLength == nil && rule.MaxLength == nil {
				return fmt.Errorf("string length rule config.%s requires a bound", rule.Path)
			}
			if rule.MinLength != nil && *rule.MinLength < 0 || rule.MaxLength != nil && *rule.MaxLength < 0 || rule.MinLength != nil && rule.MaxLength != nil && *rule.MinLength > *rule.MaxLength {
				return fmt.Errorf("string length rule config.%s has invalid bounds", rule.Path)
			}
			if rule.MinLength != nil {
				minimum := *rule.MinLength
				target.MinLength = &minimum
			}
			if rule.MaxLength != nil {
				maximum := *rule.MaxLength
				target.MaxLength = &maximum
			}
		case schema.RuleFormat:
			if target.Kind != schema.ValueString {
				return fmt.Errorf("format rule config.%s targets %s", rule.Path, target.Kind)
			}
			if rule.Format != schema.FormatCSSColor && rule.Format != schema.FormatOptionalCSSColor && rule.Format != schema.FormatSafeToken {
				return fmt.Errorf("format rule config.%s has unsupported format %q", rule.Path, rule.Format)
			}
			target.Format = rule.Format
		default:
			return fmt.Errorf("config.%s has unsupported rule kind %q", rule.Path, rule.Kind)
		}
	}
	return validateRuleConsistency(root, "config")
}

func validateRuleConsistency(value *schema.ValueSchema, path string) error {
	for _, candidate := range value.Enum {
		if issues := schema.ValidateCanonicalJSON(candidate, *value, path); len(issues) > 0 {
			return fmt.Errorf("%s enum value %s violates its field rules: %s", path, candidate, issues[0].Message)
		}
	}
	for index := range value.Fields {
		field := &value.Fields[index]
		if err := validateRuleConsistency(&field.Schema, path+"."+field.JSONName); err != nil {
			return err
		}
	}
	if value.Items != nil {
		if err := validateRuleConsistency(value.Items, path+"[]"); err != nil {
			return err
		}
	}
	return nil
}

func findConfigField(root *schema.ValueSchema, path string) (*schema.ValueSchema, error) {
	current := root
	for _, part := range strings.Split(path, ".") {
		if !validSchemaIdentifier(part) {
			return nil, fmt.Errorf("config rule path %q must use lowercase snake_case fields", path)
		}
		if current.Kind != schema.ValueObject {
			return nil, fmt.Errorf("config rule path %q traverses non-object field", path)
		}
		var next *schema.ValueSchema
		for index := range current.Fields {
			if current.Fields[index].JSONName == part {
				next = &current.Fields[index].Schema
				break
			}
		}
		if next == nil {
			return nil, fmt.Errorf("config rule path %q does not exist", path)
		}
		current = next
	}
	return current, nil
}

func prepareControl[T any](control *Control[T], configSchema schema.ValueSchema) error {
	control.ConfigPath = strings.TrimSpace(control.ConfigPath)
	control.Descriptor.Kind = strings.TrimSpace(control.Descriptor.Kind)
	control.Descriptor.Label = strings.TrimSpace(control.Descriptor.Label)
	if control.ConfigPath != "" {
		if control.ValueSchema.Kind != "" {
			return fmt.Errorf("field-bound controls must derive their value schema")
		}
		fieldSchema, err := findConfigField(&configSchema, control.ConfigPath)
		if err != nil {
			return err
		}
		if fieldSchema.Kind == schema.ValueObject || fieldSchema.Kind == schema.ValueArray {
			return fmt.Errorf("field-bound controls require a scalar config field")
		}
		control.ValueSchema = schema.CloneValue(*fieldSchema)
		if err := deriveControlDescriptor(control); err != nil {
			return err
		}
	} else if control.ValueSchema.Kind == "" {
		return fmt.Errorf("structured controls require an explicit value schema")
	}
	if len(control.Suggestions) > 0 {
		if len(control.ValueSchema.Enum) > 0 {
			return fmt.Errorf("enum controls cannot also declare suggestions")
		}
		control.Descriptor.Options = append([]ControlOption(nil), control.Suggestions...)
	}
	return nil
}

func deriveControlDescriptor[T any](control *Control[T]) error {
	value := control.ValueSchema
	if control.Descriptor.Kind == "range" {
		if value.Kind != schema.ValueInteger {
			return fmt.Errorf("range controls require an integer field")
		}
		minimum, maximum, ok := integerBounds(value)
		if !ok {
			return fmt.Errorf("range controls require a declarative range or integer enum")
		}
		control.Descriptor.Min, control.Descriptor.Max = minimum, maximum
	}
	if len(value.Enum) == 0 {
		if len(control.OptionLabels) > 0 {
			return fmt.Errorf("option labels require a declarative enum")
		}
		return nil
	}
	if control.Descriptor.Kind != "select" && control.Descriptor.Kind != "range" {
		return fmt.Errorf("enum fields require a select or range control")
	}
	if control.Descriptor.Kind == "select" {
		if value.Kind != schema.ValueString {
			return fmt.Errorf("select controls currently require a string enum")
		}
		options := make([]ControlOption, 0, len(value.Enum))
		seenLabels := map[string]bool{}
		for _, raw := range value.Enum {
			var candidate string
			if err := json.Unmarshal(raw, &candidate); err != nil {
				return fmt.Errorf("select enum contains a non-string value")
			}
			label := humanizeOption(candidate)
			if declared, ok := control.OptionLabels[candidate]; ok {
				label = strings.TrimSpace(declared)
			}
			if label == "" {
				return fmt.Errorf("option %q has an empty label", candidate)
			}
			seenLabels[candidate] = true
			options = append(options, ControlOption{Label: label, Value: candidate})
		}
		for candidate := range control.OptionLabels {
			if !seenLabels[candidate] {
				return fmt.Errorf("option label %q does not match the field enum", candidate)
			}
		}
		control.Descriptor.Options = options
	}
	return nil
}

func integerBounds(value schema.ValueSchema) (int, int, bool) {
	if value.Minimum != nil && value.Maximum != nil {
		return int(*value.Minimum), int(*value.Maximum), true
	}
	if len(value.Enum) == 0 {
		return 0, 0, false
	}
	values := make([]int, 0, len(value.Enum))
	for _, raw := range value.Enum {
		var candidate int
		if err := json.Unmarshal(raw, &candidate); err != nil {
			return 0, 0, false
		}
		values = append(values, candidate)
	}
	sort.Ints(values)
	return values[0], values[len(values)-1], true
}

func humanizeOption(value string) string {
	if value == "" {
		return "None"
	}
	var out strings.Builder
	var previous rune
	for index, char := range value {
		switch {
		case char == '_' || char == '-':
			out.WriteRune(' ')
		case unicode.IsUpper(char) && index > 0 && previous != ' ':
			out.WriteRune(' ')
			out.WriteRune(unicode.ToLower(char))
		default:
			out.WriteRune(char)
		}
		previous = char
	}
	result := strings.TrimSpace(out.String())
	if result == "" {
		return value
	}
	runes := []rune(result)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func exportControlSchema[T any](control Control[T], descriptor ControlDescriptor) schema.ControlSchema {
	out := schema.ControlSchema{
		ID: control.ID, ConfigPath: control.ConfigPath,
		Trait: descriptor.Trait, Kind: descriptor.Kind, Label: descriptor.Label, Property: descriptor.Property,
		ValueSchema: schema.CloneValue(control.ValueSchema), Default: append(json.RawMessage(nil), descriptor.Value...),
		Step: descriptor.Step,
	}
	if len(control.Suggestions) > 0 {
		out.Suggestions = exportControlOptions(control.Suggestions)
	} else {
		out.Options = exportControlOptions(descriptor.Options)
	}
	return out
}

func exportControlOptions(options []ControlOption) []schema.ControlOptionSchema {
	out := make([]schema.ControlOptionSchema, 0, len(options))
	for _, option := range options {
		raw, _ := json.Marshal(option.Value)
		out = append(out, schema.ControlOptionSchema{Label: option.Label, Value: raw})
	}
	return out
}
