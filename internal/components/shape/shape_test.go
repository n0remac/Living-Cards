package shape

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
)

func TestValidateGeneratedAcceptsSafeShapeConfig(t *testing.T) {
	t.Parallel()

	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Safe shape",
		Config: Config{
			Shape:           "diamond",
			X:               24,
			Y:               32,
			Width:           40,
			Height:          28,
			Rotation:        12,
			BackgroundColor: "#38bdf8",
			BorderColor:     "rgba(15,23,42,0.8)",
			BorderWidthPX:   2,
			Shadow:          "0 10px 24px rgba(14,165,233,0.22)",
		},
	}
	if issues := ValidateGenerated(generated); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateGeneratedRejectsInvalidShapeConfig(t *testing.T) {
	t.Parallel()

	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Invalid shape",
		Config: Config{
			Shape:           "hexagon",
			X:               -1,
			Y:               101,
			Width:           4,
			Height:          120,
			BackgroundColor: "url(https://example.test)",
			BorderColor:     "clear",
			BorderWidthPX:   99,
			Shadow:          "0 0 1px red",
		},
	}
	issues := ValidateGenerated(generated)
	paths := map[string]bool{}
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	for _, path := range []string{
		"config.shape",
		"config.x",
		"config.y",
		"config.width",
		"config.height",
		"config.background_color",
		"config.border_color",
		"config.border_width_px",
		"config.shadow",
	} {
		if !paths[path] {
			t.Fatalf("issues missing %s: %#v", path, issues)
		}
	}
}

func TestRenderLayerIncludesShapeDataAttributesAndSVG(t *testing.T) {
	t.Parallel()

	node := RenderLayer("shape-1", Config{
		Shape:           "star",
		X:               10,
		Y:               20,
		Width:           30,
		Height:          30,
		BackgroundColor: "#f43f5e",
		BorderColor:     "#f8fafc",
		BorderWidthPX:   2,
	})
	body := node.Render()
	for _, marker := range []string{
		`data-component-id="shape-1"`,
		`data-component-kind="shape"`,
		`left: 10%`,
		`top: 20%`,
		`<svg`,
		`points="50,6 61,36`,
		`fill="#f43f5e"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}

func TestRenderLayerWithContextScopesShapeID(t *testing.T) {
	t.Parallel()

	body := RenderLayerWithContext("shape-1", DefaultConfig(), card.RenderContext{DOMIDPrefix: "game-world-card"}).Render()
	for _, marker := range []string{
		`id="game-world-card-shape-1-layer"`,
		`data-component-id="shape-1"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}
