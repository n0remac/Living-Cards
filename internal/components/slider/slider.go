package slider

import (
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
)

const Kind = "slider"

type Config struct {
	Label       string `json:"label"`
	Min         int    `json:"min"`
	Max         int    `json:"max"`
	Step        int    `json:"step"`
	Value       int    `json:"value"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Width       int    `json:"width"`
	TrackColor  string `json:"track_color"`
	AccentColor string `json:"accent_color"`
}

func DefaultConfig() Config {
	return Config{
		Label:       "Output",
		Min:         0,
		Max:         100,
		Step:        1,
		Value:       50,
		X:           50,
		Y:           70,
		Width:       72,
		TrackColor:  "rgba(15,23,42,0.72)",
		AccentColor: "#7dd3fc",
	}
}

func Definition() card.Definition {
	return card.Definition{
		ComponentKind: Kind,
		Contribute: func(node card.Node, renderContext card.RenderContext) (card.Contribution, error) {
			part, err := card.DecodeConfig[Config](node)
			if err != nil {
				return card.Contribution{}, err
			}
			part = NormalizeConfig(part)
			if issues := ValidateConfig(part); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid slider config at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)},
			}, nil
		},
	}
}

func NormalizeConfig(part Config) Config {
	defaults := DefaultConfig()
	part.Label = strings.TrimSpace(part.Label)
	if part.Label == "" {
		part.Label = defaults.Label
	}
	part.Min = clamp(part.Min, 0, 100)
	part.Max = clamp(part.Max, 0, 100)
	if part.Max < part.Min {
		part.Max = part.Min
	}
	if part.Step <= 0 {
		part.Step = defaults.Step
	}
	part.Step = clamp(part.Step, 1, 100)
	part.Value = clamp(part.Value, part.Min, part.Max)
	if part.X == 0 {
		part.X = defaults.X
	}
	if part.Y == 0 {
		part.Y = defaults.Y
	}
	if part.Width == 0 {
		part.Width = defaults.Width
	}
	part.X = clamp(part.X, 0, 100)
	part.Y = clamp(part.Y, 0, 100)
	part.Width = clamp(part.Width, 12, 100)
	part.TrackColor = strings.TrimSpace(part.TrackColor)
	if part.TrackColor == "" {
		part.TrackColor = defaults.TrackColor
	}
	part.AccentColor = strings.TrimSpace(part.AccentColor)
	if part.AccentColor == "" {
		part.AccentColor = defaults.AccentColor
	}
	return part
}

func ValidateConfig(part Config) []design.Issue {
	return ValidateGenerated(design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Slider config",
		Config:        part,
	})
}

func NormalizeGenerated(generated *design.GeneratedConfig[Config]) {
	if generated == nil {
		return
	}
	generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Config = NormalizeConfig(generated.Config)
}

func ValidateGenerated(generated design.GeneratedConfig[Config]) []design.Issue {
	var issues []design.Issue
	if strings.TrimSpace(generated.Config.Label) == "" {
		issues = append(issues, design.Issue{
			Path:    "config.label",
			Code:    "required",
			Message: "label is required",
		})
	}
	if generated.Config.Min < 0 || generated.Config.Min > 100 {
		issues = append(issues, rangeIssue("config.min", "min", generated.Config.Min, 0, 100))
	}
	if generated.Config.Max < 0 || generated.Config.Max > 100 {
		issues = append(issues, rangeIssue("config.max", "max", generated.Config.Max, 0, 100))
	}
	if generated.Config.Max < generated.Config.Min {
		issues = append(issues, design.Issue{
			Path:    "config.max",
			Code:    "out_of_range",
			Message: "max must be greater than or equal to min",
			Actual:  generated.Config.Max,
		})
	}
	if generated.Config.Step < 1 || generated.Config.Step > 100 {
		issues = append(issues, rangeIssue("config.step", "step", generated.Config.Step, 1, 100))
	}
	if generated.Config.Value < generated.Config.Min || generated.Config.Value > generated.Config.Max {
		issues = append(issues, design.Issue{
			Path:    "config.value",
			Code:    "out_of_range",
			Message: "value must be between min and max",
			Actual:  generated.Config.Value,
		})
	}
	if generated.Config.X < 0 || generated.Config.X > 100 {
		issues = append(issues, rangeIssue("config.x", "x", generated.Config.X, 0, 100))
	}
	if generated.Config.Y < 0 || generated.Config.Y > 100 {
		issues = append(issues, rangeIssue("config.y", "y", generated.Config.Y, 0, 100))
	}
	if generated.Config.Width < 12 || generated.Config.Width > 100 {
		issues = append(issues, rangeIssue("config.width", "width", generated.Config.Width, 12, 100))
	}
	if !design.IsAllowedColor(generated.Config.TrackColor) {
		issues = append(issues, design.Issue{
			Path:    "config.track_color",
			Code:    "invalid_color",
			Message: "track_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  generated.Config.TrackColor,
		})
	}
	if !design.IsAllowedColor(generated.Config.AccentColor) {
		issues = append(issues, design.Issue{
			Path:    "config.accent_color",
			Code:    "invalid_color",
			Message: "accent_color must be a hex, rgb, rgba, hsl, or hsla color",
			Actual:  generated.Config.AccentColor,
		})
	}
	return issues
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = NormalizeConfig(part)
	style := map[string]string{
		"background":     part.TrackColor,
		"border":         "1px solid " + part.AccentColor,
		"border-radius":  "14px",
		"box-shadow":     "0 14px 34px rgba(8,47,73,0.22)",
		"color":          "#e0f2fe",
		"display":        "grid",
		"gap":            "8px",
		"left":           fmt.Sprintf("%d%%", part.X),
		"padding":        "12px",
		"pointer-events": "auto",
		"top":            fmt.Sprintf("%d%%", part.Y),
		"transform":      "translate(-50%, -50%)",
		"width":          fmt.Sprintf("%d%%", part.Width),
		"z-index":        "2",
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("style", styleString(style)),
		godom.Div(
			godom.Class("flex items-center justify-between gap-2 text-xs font-bold uppercase"),
			godom.Span(godom.T(part.Label)),
			godom.Span(godom.T(fmt.Sprintf("%d", part.Value))),
		),
		godom.Input(
			godom.Type("range"),
			godom.Attr("min", fmt.Sprintf("%d", part.Min)),
			godom.Attr("max", fmt.Sprintf("%d", part.Max)),
			godom.Attr("step", fmt.Sprintf("%d", part.Step)),
			godom.Value(fmt.Sprintf("%d", part.Value)),
			godom.Attr("aria-label", part.Label),
			godom.Attr("disabled", "disabled"),
			godom.Attr("style", "accent-color: "+part.AccentColor+";"),
			godom.Class("w-full"),
		),
	)
}

func rangeIssue(path, field string, actual, min, max int) design.Issue {
	return design.Issue{
		Path:    path,
		Code:    "out_of_range",
		Message: fmt.Sprintf("%s must be between %d and %d", field, min, max),
		Actual:  actual,
	}
}

func styleString(styles map[string]string) string {
	keys := make([]string, 0, len(styles))
	for key := range styles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		value := strings.TrimSpace(styles[key])
		if value == "" {
			continue
		}
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(value)
		out.WriteString("; ")
	}
	return strings.TrimSpace(out.String())
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
