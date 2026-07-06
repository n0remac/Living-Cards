package design

import (
	"regexp"
	"sort"
	"strings"
)

var (
	hexColorPattern      = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	functionColorPattern = regexp.MustCompile(`(?i)^(rgba?|hsla?)\([0-9.,%\s+-]+\)$`)
	forbiddenCSSMarkers  = []string{
		"<",
		">",
		"{",
		"}",
		"url(",
		"javascript:",
		"expression(",
		"@import",
		"position",
		"content",
	}
)

func IsAllowedColor(value string) bool {
	value = strings.TrimSpace(value)
	return hexColorPattern.MatchString(value) || functionColorPattern.MatchString(value)
}

func ValidateInlineCSS(path, css string, allowed map[string]struct{}) []Issue {
	css = strings.TrimSpace(css)
	if css == "" {
		return nil
	}
	lower := strings.ToLower(css)
	for _, marker := range forbiddenCSSMarkers {
		if strings.Contains(lower, marker) {
			return []Issue{{
				Path:    path,
				Code:    "unsafe_css",
				Message: "css contains forbidden marker " + marker,
				Actual:  marker,
			}}
		}
	}
	var issues []Issue
	for _, declaration := range strings.Split(css, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		property, value, ok := strings.Cut(declaration, ":")
		if !ok {
			issues = append(issues, Issue{
				Path:    path,
				Code:    "invalid_css_declaration",
				Message: "css declaration is missing a colon",
				Actual:  declaration,
			})
			continue
		}
		property = strings.ToLower(strings.TrimSpace(property))
		value = strings.TrimSpace(value)
		if property == "" || value == "" || strings.ContainsAny(property, " \t\r\n") {
			issues = append(issues, Issue{
				Path:    path,
				Code:    "invalid_css_declaration",
				Message: "css declaration is incomplete or invalid",
				Actual:  declaration,
			})
			continue
		}
		if _, ok := allowed[property]; !ok {
			issues = append(issues, Issue{
				Path:    path,
				Code:    "unsupported_css_property",
				Message: "css property is not allowed",
				Actual:  property,
				Allowed: allowedKeys(allowed),
			})
		}
	}
	return issues
}

func CSSDeclarations(css string, allowed map[string]struct{}) map[string]string {
	out := map[string]string{}
	if len(ValidateInlineCSS("css", css, allowed)) > 0 {
		return out
	}
	for _, declaration := range strings.Split(css, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}
		property, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		property = strings.ToLower(strings.TrimSpace(property))
		value = strings.TrimSpace(value)
		if _, ok := allowed[property]; !ok || value == "" {
			continue
		}
		out[property] = value
	}
	return out
}

func allowedKeys(input map[string]struct{}) []string {
	out := make([]string, 0, len(input))
	for key := range input {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
