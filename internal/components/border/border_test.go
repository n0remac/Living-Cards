package border

import (
	"testing"

	"github.com/n0remac/Living-Card/internal/fragment"
)

func TestNormalizeGeneratedClampsBorderDimensions(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Clamp",
		Fragment: Fragment{
			BorderWidthPX:  100,
			BorderRadiusPX: -4,
			BorderColor:    "#ffffff",
		},
	}
	NormalizeGenerated(&generated)
	if generated.Fragment.BorderWidthPX != 16 || generated.Fragment.BorderRadiusPX != 0 {
		t.Fatalf("fragment = %#v", generated.Fragment)
	}
}

func TestValidateGeneratedAcceptsSafeBorderCSS(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Safe",
		Fragment: Fragment{
			BorderWidthPX:  2,
			BorderRadiusPX: 18,
			BorderColor:    "hsla(190, 90%, 70%, 0.7)",
			CSS:            "border: 2px solid rgba(103, 232, 249, 0.7); box-shadow: 0 0 24px rgba(34, 211, 238, 0.25);",
		},
	}
	if issues := ValidateGenerated(generated); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateGeneratedRejectsInvalidBorderFields(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Bad",
		Fragment: Fragment{
			BorderWidthPX:  -1,
			BorderRadiusPX: 90,
			BorderColor:    "blueish",
			CSS:            "background: red;",
		},
	}
	issues := ValidateGenerated(generated)
	paths := issuePaths(issues)
	for _, path := range []string{"fragment.border_width_px", "fragment.border_radius_px", "fragment.border_color", "fragment.css"} {
		if !paths[path] {
			t.Fatalf("issues missing %s: %#v", path, issues)
		}
	}
}

func issuePaths(issues []fragment.Issue) map[string]bool {
	paths := make(map[string]bool, len(issues))
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	return paths
}
