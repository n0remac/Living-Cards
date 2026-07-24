package slider

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestNormalizeConfigDoesNotRepairSliderRange(t *testing.T) {
	t.Parallel()

	part := NormalizeConfig(Config{
		Label: "  ",
		Min:   -10,
		Max:   120,
		Step:  0,
		Value: 140,
	})
	if part.Label != "" || part.Min != -10 || part.Max != 120 || part.Step != 0 || part.Value != 140 {
		t.Fatalf("part = %#v", part)
	}
	if issues := ValidateConfig(part); len(issues) == 0 {
		t.Fatal("ValidateConfig() issues = nil, want invalid slider")
	}
}

func TestValidateGeneratedRejectsInvalidSlider(t *testing.T) {
	t.Parallel()

	issues := ValidateGenerated(schema.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Invalid slider",
		Config: Config{
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

	body := RenderLayer("regulator-slider", Config{
		Label: "Output",
		Min:   0,
		Max:   100,
		Step:  1,
		Value: 73,
	}).Render()
	for _, marker := range []string{
		`data-component-id="regulator-slider"`,
		`data-component-kind="slider"`,
		`data-slider-input`,
		`data-slider-value`,
		`type="range"`,
		`value="73"`,
		`Output`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
	if strings.Contains(body, `disabled`) {
		t.Fatalf("render should keep slider input enabled:\n%s", body)
	}
}
