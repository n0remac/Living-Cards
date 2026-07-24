package border

import (
	"testing"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestNormalizeGeneratedDoesNotRepairBorderDimensions(t *testing.T) {
	t.Parallel()

	generated := schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Clamp",
		Config: Config{
			BorderWidthPX:  100,
			BorderRadiusPX: -4,
			BorderColor:    "#ffffff",
		},
	}
	NormalizeGenerated(&generated)
	if generated.Config.BorderWidthPX != 100 || generated.Config.BorderRadiusPX != -4 {
		t.Fatalf("config = %#v", generated.Config)
	}
	if issues := ValidateGenerated(generated); len(issues) == 0 {
		t.Fatal("ValidateGenerated() issues = nil, want invalid dimensions")
	}
}

func TestRandomGeneratedBorderValidates(t *testing.T) {
	t.Parallel()

	for _, seed := range []int64{1, 2, 3, 4, 5, 6} {
		generated := RandomGenerated(seed, 3)
		NormalizeGenerated(&generated)
		if generated.ComponentKind != Kind {
			t.Fatalf("componentKind = %q, want %q", generated.ComponentKind, Kind)
		}
		if issues := ValidateGenerated(generated); len(issues) != 0 {
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
	if issues := ValidateGenerated(generated); len(issues) != 0 {
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
	issues := ValidateGenerated(generated)
	paths := issuePaths(issues)
	for _, path := range []string{"config.border_width_px", "config.border_radius_px", "config.border_color", "config.css"} {
		if !paths[path] {
			t.Fatalf("issues missing %s: %#v", path, issues)
		}
	}
}

func issuePaths(issues []schema.Issue) map[string]bool {
	paths := make(map[string]bool, len(issues))
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	return paths
}
