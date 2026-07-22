package button

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
)

const Kind = "button"

type Config struct {
	FormID string `json:"form_id"`
	Label  string `json:"label"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
}

func DefaultConfig() Config {
	return Config{FormID: "controller-form", Label: "Submit", X: 50, Y: 72, Width: 44}
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
				return card.Contribution{}, fmt.Errorf("invalid button config at %s: %s", issues[0].Path, issues[0].Message)
			}
			return card.Contribution{Layers: []*godom.Node{RenderLayerWithContext(node.ID, part, renderContext)}}, nil
		},
	}
}

func NormalizeConfig(part Config) Config {
	defaults := DefaultConfig()
	part.FormID = strings.TrimSpace(part.FormID)
	if part.FormID == "" {
		part.FormID = defaults.FormID
	}
	part.Label = strings.TrimSpace(part.Label)
	if part.Label == "" {
		part.Label = defaults.Label
	}
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
	return part
}

func ValidateConfig(part Config) []design.Issue {
	var issues []design.Issue
	if !safeToken(part.FormID) {
		issues = append(issues, issue("config.form_id", "form_id must contain only letters, numbers, hyphens, or underscores"))
	}
	if strings.TrimSpace(part.Label) == "" {
		issues = append(issues, issue("config.label", "label is required"))
	}
	if len([]rune(part.Label)) > 80 {
		issues = append(issues, issue("config.label", "label must be at most 80 characters"))
	}
	if part.X < 0 || part.X > 100 {
		issues = append(issues, issue("config.x", "x must be between 0 and 100"))
	}
	if part.Y < 0 || part.Y > 100 {
		issues = append(issues, issue("config.y", "y must be between 0 and 100"))
	}
	if part.Width < 12 || part.Width > 100 {
		issues = append(issues, issue("config.width", "width must be between 12 and 100"))
	}
	return issues
}

func RenderLayer(componentID string, part Config) *godom.Node {
	return RenderLayerWithContext(componentID, part, card.RenderContext{})
}

func RenderLayerWithContext(componentID string, part Config, renderContext card.RenderContext) *godom.Node {
	part = NormalizeConfig(part)
	style := map[string]string{
		"left":      fmt.Sprintf("%d%%", part.X),
		"top":       fmt.Sprintf("%d%%", part.Y),
		"transform": "translate(-50%, -50%)",
		"width":     fmt.Sprintf("%d%%", part.Width),
		"z-index":   "4",
	}
	return godom.Button(
		godom.Id(renderContext.LayerID(componentID)),
		godom.Type("submit"),
		godom.Attr("form", renderContext.DOMID(part.FormID)),
		godom.Attr("data-component-id", componentID),
		godom.Attr("data-component-kind", Kind),
		godom.Attr("data-card-form-control", ""),
		godom.Attr("data-form-id", part.FormID),
		godom.Attr("style", styleString(style)),
		godom.Class("absolute rounded-xl border border-emerald-200/70 bg-emerald-500/90 px-4 py-2 text-sm font-black uppercase tracking-[0.12em] text-emerald-950 shadow-lg"),
		godom.T(part.Label),
	)
}

func safeToken(value string) bool {
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

func issue(path, message string) design.Issue {
	return design.Issue{Path: path, Code: "invalid", Message: message}
}

func styleString(styles map[string]string) string {
	keys := make([]string, 0, len(styles))
	for key := range styles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, key := range keys {
		out.WriteString(key)
		out.WriteString(": ")
		out.WriteString(styles[key])
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
