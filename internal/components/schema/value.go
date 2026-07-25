package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxSafeInteger = int64(9007199254740991)

type validationMode int

const (
	structureMode validationMode = iota
	canonicalMode
)

func ValidateJSONStructure(raw json.RawMessage, value ValueSchema, path string) []Issue {
	return validateJSON(raw, value, path, structureMode, true)
}

func ValidateCanonicalJSON(raw json.RawMessage, value ValueSchema, path string) []Issue {
	return validateJSON(raw, value, path, canonicalMode, false)
}

func validateJSON(raw json.RawMessage, value ValueSchema, path string, mode validationMode, partialObject bool) []Issue {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return []Issue{{Path: path, Code: "invalid_json", Message: "value is required"}}
	}
	if bytes.Equal(trimmed, []byte("null")) {
		if value.Nullable {
			return nil
		}
		return []Issue{{Path: path, Code: "null_value", Message: "value must not be null"}}
	}
	switch value.Kind {
	case ValueString:
		var decoded string
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return typeIssue(path, "string")
		}
		if mode == canonicalMode {
			return validateString(decoded, value, path)
		}
	case ValueBoolean:
		var decoded bool
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return typeIssue(path, "boolean")
		}
		if mode == canonicalMode {
			return validateEnum(trimmed, value, path)
		}
	case ValueInteger:
		var decoded int64
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return typeIssue(path, "integer")
		}
		if decoded < -maxSafeInteger || decoded > maxSafeInteger {
			return []Issue{{Path: path, Code: "unsafe_integer", Message: "integer must be representable exactly by JavaScript", Actual: decoded}}
		}
		if mode == canonicalMode {
			return validateNumber(float64(decoded), trimmed, value, path)
		}
	case ValueNumber:
		var decoded float64
		if err := json.Unmarshal(trimmed, &decoded); err != nil || math.IsInf(decoded, 0) || math.IsNaN(decoded) {
			return typeIssue(path, "finite number")
		}
		if mode == canonicalMode {
			return validateNumber(decoded, trimmed, value, path)
		}
	case ValueObject:
		var decoded map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return typeIssue(path, "object")
		}
		fields := make(map[string]FieldSchema, len(value.Fields))
		for _, field := range value.Fields {
			fields[field.JSONName] = field
		}
		var issues []Issue
		for name, fieldRaw := range decoded {
			field, ok := fields[name]
			if !ok {
				issues = append(issues, Issue{Path: joinPath(path, name), Code: "unknown_field", Message: "field is not allowed"})
				continue
			}
			issues = append(issues, validateJSON(fieldRaw, field.Schema, joinPath(path, name), mode, mode == structureMode)...)
		}
		if mode == canonicalMode && !partialObject {
			for _, field := range value.Fields {
				if field.Required {
					if _, ok := decoded[field.JSONName]; !ok {
						issues = append(issues, Issue{Path: joinPath(path, field.JSONName), Code: "required", Message: "field is required"})
					}
				}
			}
		}
		return issues
	case ValueArray:
		var decoded []json.RawMessage
		if err := json.Unmarshal(trimmed, &decoded); err != nil {
			return typeIssue(path, "array")
		}
		if value.Items == nil {
			return []Issue{{Path: path, Code: "invalid_schema", Message: "array item schema is missing"}}
		}
		var issues []Issue
		for index, item := range decoded {
			issues = append(issues, validateJSON(item, *value.Items, fmt.Sprintf("%s[%d]", path, index), mode, false)...)
		}
		return issues
	default:
		return []Issue{{Path: path, Code: "invalid_schema", Message: fmt.Sprintf("unsupported schema kind %q", value.Kind)}}
	}
	return nil
}

func validateString(decoded string, value ValueSchema, path string) []Issue {
	var issues []Issue
	length := utf8.RuneCountInString(decoded)
	if value.MinLength != nil && length < *value.MinLength {
		issues = append(issues, Issue{Path: path, Code: "too_short", Message: fmt.Sprintf("value must contain at least %d characters", *value.MinLength), Actual: length})
	}
	if value.MaxLength != nil && length > *value.MaxLength {
		issues = append(issues, Issue{Path: path, Code: "too_long", Message: fmt.Sprintf("value must contain at most %d characters", *value.MaxLength), Actual: length})
	}
	switch value.Format {
	case FormatNone:
	case FormatCSSColor:
		if !IsAllowedColor(decoded) {
			issues = append(issues, Issue{Path: path, Code: "invalid_color", Message: "value must be a hex, rgb, rgba, hsl, or hsla color", Actual: decoded})
		}
	case FormatOptionalCSSColor:
		if decoded != "" && !IsAllowedColor(decoded) {
			issues = append(issues, Issue{Path: path, Code: "invalid_color", Message: "value must be empty or a hex, rgb, rgba, hsl, or hsla color", Actual: decoded})
		}
	case FormatSafeToken:
		if !IsSafeToken(decoded) {
			issues = append(issues, Issue{Path: path, Code: "invalid_token", Message: "value must contain only letters, numbers, hyphens, or underscores", Actual: decoded})
		}
	default:
		issues = append(issues, Issue{Path: path, Code: "invalid_schema", Message: fmt.Sprintf("unsupported string format %q", value.Format)})
	}
	raw, _ := json.Marshal(decoded)
	issues = append(issues, validateEnum(raw, value, path)...)
	return issues
}

func validateNumber(decoded float64, raw json.RawMessage, value ValueSchema, path string) []Issue {
	var issues []Issue
	if value.Minimum != nil && decoded < *value.Minimum {
		issues = append(issues, Issue{Path: path, Code: "out_of_range", Message: fmt.Sprintf("value must be at least %v", *value.Minimum), Actual: decoded})
	}
	if value.Maximum != nil && decoded > *value.Maximum {
		issues = append(issues, Issue{Path: path, Code: "out_of_range", Message: fmt.Sprintf("value must be at most %v", *value.Maximum), Actual: decoded})
	}
	issues = append(issues, validateEnum(raw, value, path)...)
	return issues
}

func validateEnum(raw json.RawMessage, value ValueSchema, path string) []Issue {
	if len(value.Enum) == 0 {
		return nil
	}
	compact := compactJSON(raw)
	allowed := make([]string, 0, len(value.Enum))
	for _, candidate := range value.Enum {
		canonical := compactJSON(candidate)
		allowed = append(allowed, string(canonical))
		if bytes.Equal(compact, canonical) {
			return nil
		}
	}
	return []Issue{{Path: path, Code: "invalid_option", Message: "value is not an allowed option", Actual: jsonValue(raw), Allowed: allowed}}
}

func IsSafeToken(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func typeIssue(path, expected string) []Issue {
	return []Issue{{Path: path, Code: "invalid_type", Message: "value must be a " + expected}}
}

func joinPath(parent, child string) string {
	if parent == "" || parent == "$" {
		if parent == "$" {
			return "$." + child
		}
		return child
	}
	return parent + "." + child
}

func compactJSON(raw json.RawMessage) json.RawMessage {
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		return append(json.RawMessage(nil), raw...)
	}
	return out.Bytes()
}

func jsonValue(raw json.RawMessage) any {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return string(raw)
	}
	return value
}
