package slider

import (
	"fmt"
	"sort"
	"strings"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const Type = "slider"

type Fragment struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Step  int    `json:"step"`
	Value int    `json:"value"`
}

func DefaultFragment() Fragment {
	return Fragment{
		Label: "Output",
		Min:   0,
		Max:   100,
		Step:  1,
		Value: 50,
	}
}

func Definition() card.Definition {
	return card.Definition{
		Type: Type,
		Contribute: func(node card.Node, renderContext card.RenderContext) (card.Contribution, error) {
			part, err := card.DecodeFragment[Fragment](node)
			if err != nil {
				return card.Contribution{}, err
			}
			part = NormalizeFragment(part)
			if issues := ValidateFragment(part); len(issues) > 0 {
				return card.Contribution{}, fmt.Errorf("invalid slider fragment at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{
				Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)},
			}, nil
		},
	}
}

func NormalizeFragment(part Fragment) Fragment {
	defaults := DefaultFragment()
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
	return part
}

func ValidateFragment(part Fragment) []fragment.Issue {
	return ValidateGenerated(fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Slider fragment",
		Fragment:    part,
	})
}

func NormalizeGenerated(generated *fragment.Generated[Fragment]) {
	if generated == nil {
		return
	}
	generated.Target = strings.TrimSpace(generated.Target)
	generated.Description = strings.TrimSpace(generated.Description)
	generated.Fragment = NormalizeFragment(generated.Fragment)
}

func ValidateGenerated(generated fragment.Generated[Fragment]) []fragment.Issue {
	var issues []fragment.Issue
	if strings.TrimSpace(generated.Fragment.Label) == "" {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.label",
			Code:    "required",
			Message: "label is required",
		})
	}
	if generated.Fragment.Min < 0 || generated.Fragment.Min > 100 {
		issues = append(issues, rangeIssue("fragment.min", "min", generated.Fragment.Min, 0, 100))
	}
	if generated.Fragment.Max < 0 || generated.Fragment.Max > 100 {
		issues = append(issues, rangeIssue("fragment.max", "max", generated.Fragment.Max, 0, 100))
	}
	if generated.Fragment.Max < generated.Fragment.Min {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.max",
			Code:    "out_of_range",
			Message: "max must be greater than or equal to min",
			Actual:  generated.Fragment.Max,
		})
	}
	if generated.Fragment.Step < 1 || generated.Fragment.Step > 100 {
		issues = append(issues, rangeIssue("fragment.step", "step", generated.Fragment.Step, 1, 100))
	}
	if generated.Fragment.Value < generated.Fragment.Min || generated.Fragment.Value > generated.Fragment.Max {
		issues = append(issues, fragment.Issue{
			Path:    "fragment.value",
			Code:    "out_of_range",
			Message: "value must be between min and max",
			Actual:  generated.Fragment.Value,
		})
	}
	return issues
}

func RenderLayer(componentID string, part Fragment) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Fragment, renderContext card.RenderContext) *godom.Node {
	part = NormalizeFragment(part)
	style := map[string]string{
		"background":     "rgba(15,23,42,0.72)",
		"border":         "1px solid rgba(125,211,252,0.42)",
		"border-radius":  "14px",
		"box-shadow":     "0 14px 34px rgba(8,47,73,0.22)",
		"color":          "#e0f2fe",
		"display":        "grid",
		"gap":            "8px",
		"left":           "50%",
		"padding":        "12px",
		"pointer-events": "auto",
		"top":            "70%",
		"transform":      "translate(-50%, -50%)",
		"width":          "72%",
		"z-index":        "2",
	}
	return godom.Div(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Class("absolute"),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-type", Type),
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
			godom.Class("w-full accent-sky-300"),
		),
	)
}

func rangeIssue(path, field string, actual, min, max int) fragment.Issue {
	return fragment.Issue{
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
