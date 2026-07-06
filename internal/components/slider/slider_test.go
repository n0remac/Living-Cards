package slider

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/fragment"
)

func TestNormalizeFragmentClampsSliderRange(t *testing.T) {
	t.Parallel()

	part := NormalizeFragment(Fragment{
		Label: "  ",
		Min:   -10,
		Max:   120,
		Step:  0,
		Value: 140,
	})
	if part.Label != "Output" || part.Min != 0 || part.Max != 100 || part.Step != 1 || part.Value != 100 {
		t.Fatalf("part = %#v", part)
	}
}

func TestValidateGeneratedRejectsInvalidSlider(t *testing.T) {
	t.Parallel()

	issues := ValidateGenerated(fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Invalid slider",
		Fragment: Fragment{
			Label: "",
			Min:   90,
			Max:   20,
			Step:  0,
			Value: 101,
		},
	})
	if len(issues) < 4 {
		t.Fatalf("issues = %#v, want multiple validation issues", issues)
	}
}

func TestRenderLayerIncludesSliderValue(t *testing.T) {
	t.Parallel()

	body := RenderLayer("regulator-slider", Fragment{
		Label: "Output",
		Min:   0,
		Max:   100,
		Step:  1,
		Value: 73,
	}).Render()
	for _, marker := range []string{
		`data-component-id="regulator-slider"`,
		`data-component-type="slider"`,
		`type="range"`,
		`value="73"`,
		`Output`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}
