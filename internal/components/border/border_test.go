package border

import (
	"encoding/json"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestNormalizeGeneratedDoesNotRepairBorderDimensions(t *testing.T) {
	t.Parallel()

	config := NormalizeConfig(Config{
		BorderWidthPX:  100,
		BorderRadiusPX: -4,
		BorderColor:    "#ffffff",
	})
	if config.BorderWidthPX != 100 || config.BorderRadiusPX != -4 {
		t.Fatalf("config = %#v", config)
	}
	if issues := configIssues(config); len(issues) == 0 {
		t.Fatal("configIssues() issues = nil, want invalid dimensions")
	}
}

func TestRandomGeneratedBorderValidates(t *testing.T) {
	t.Parallel()

	for _, seed := range []int64{1, 2, 3, 4, 5, 6} {
		generated := RandomGenerated(seed, 3)
		if generated.ComponentKind != Kind {
			t.Fatalf("componentKind = %q, want %q", generated.ComponentKind, Kind)
		}
		if issues := configIssues(generated.Config); len(issues) != 0 {
			t.Fatalf("seed %d issues = %#v", seed, issues)
		}
	}
}

func TestValidateGeneratedAcceptsSafeBorderCSS(t *testing.T) {
	t.Parallel()

	generated := schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Safe",
		Config: Config{
			BorderWidthPX:  2,
			BorderRadiusPX: 18,
			BorderColor:    "hsla(190, 90%, 70%, 0.7)",
			BorderStyle:    "solid",
			CSS:            "border: 2px solid rgba(103, 232, 249, 0.7); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
		},
	}
	if issues := configIssues(generated.Config); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateGeneratedRejectsInvalidBorderFields(t *testing.T) {
	t.Parallel()

	generated := schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Bad",
		Config: Config{
			BorderWidthPX:  -1,
			BorderRadiusPX: 90,
			BorderColor:    "blueish",
			BorderStyle:    "solid",
			CSS:            "background: red;",
		},
	}
	issues := configIssues(generated.Config)
	paths := issuePaths(issues)
	for _, path := range []string{"config.border_width_px", "config.border_radius_px", "config.border_color", "config.css"} {
		if !paths[path] {
			t.Fatalf("issues missing %s: %#v", path, issues)
		}
	}
}

func configIssues(config Config) []schema.Issue {
	raw, _ := json.Marshal(config)
	_, issues := Definition().CanonicalizeConfig(card.RawConfig{Present: true, Value: raw})
	return issues
}

func issuePaths(issues []schema.Issue) map[string]bool {
	paths := make(map[string]bool, len(issues))
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	return paths
}
