package textarea

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/design"
)

func TestValidateGeneratedAcceptsSafeTextareaConfig(t *testing.T) {
	t.Parallel()

	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Safe",
		Config: Config{
			Content:    "A quiet note.",
			FontFamily: "serif",
			FontSizePX: 18,
			FontWeight: 400,
			FontStyle:  "italic",
			Color:      "rgba(226,232,240,0.92)",
			Align:      "center",
			Position:   "center",
			X:          50,
			Y:          50,
			CSS:        "font-family: Georgia, serif; line-height: 1.5; text-align: center; text-shadow: 0 1px 8px rgba(0,0,0,0.3);",
		},
	}
	if issues := ValidateGenerated(generated); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestNormalizeGeneratedClampsFontSize(t *testing.T) {
	t.Parallel()

	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Clamp",
		Config:        validConfig(),
	}
	generated.Config.FontSizePX = 999
	NormalizeGenerated(&generated)
	if generated.Config.FontSizePX != 72 {
		t.Fatalf("FontSizePX = %d, want 72", generated.Config.FontSizePX)
	}
}

func TestValidateGeneratedRejectsInvalidTextareaFields(t *testing.T) {
	t.Parallel()

	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Bad",
		Config: Config{
			Content:    "",
			FontFamily: "fantasy",
			FontSizePX: 8,
			FontWeight: 900,
			FontStyle:  "oblique",
			Color:      "clear",
			Align:      "justify",
			Position:   "middle",
			X:          -1,
			Y:          101,
			CSS:        "position: fixed;",
		},
	}
	issues := ValidateGenerated(generated)
	paths := textareaIssuePaths(issues)
	for _, path := range []string{
		"config.content",
		"config.font_family",
		"config.font_size_px",
		"config.font_weight",
		"config.font_style",
		"config.color",
		"config.align",
		"config.position",
		"config.x",
		"config.y",
		"config.css",
	} {
		if !paths[path] {
			t.Fatalf("issues missing %s: %#v", path, issues)
		}
	}
}

func TestValidateGeneratedRejectsUnsupportedTextCSSProperty(t *testing.T) {
	t.Parallel()

	part := validConfig()
	part.CSS = "background: red;"
	issues := ValidateGenerated(design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Bad CSS",
		Config:        part,
	})
	if len(issues) != 1 || issues[0].Path != "config.css" || issues[0].Code != "unsupported_css_property" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestRenderLayerIncludesExtendedTextareaStyles(t *testing.T) {
	t.Parallel()

	node := RenderLayer("textarea-main", Config{
		Content:         "Styled text",
		FontFamily:      "system",
		FontSizePX:      18,
		FontWeight:      600,
		FontStyle:       "normal",
		Color:           "#111827",
		Align:           "center",
		Position:        "center",
		X:               42,
		Y:               58,
		BackgroundColor: "#f8fafc",
		BorderColor:     "#111827",
		BorderWidthPX:   2,
		BorderRadiusPX:  14,
		PaddingPX:       12,
		CSS:             "",
	})
	body := node.Render()
	for _, marker := range []string{
		`data-component-id="textarea-main"`,
		`data-component-kind="textarea"`,
		`background-color: #f8fafc`,
		`border: 2px solid #111827`,
		`border-radius: 14px`,
		`left: 42%`,
		`padding: 12px`,
		`top: 58%`,
		`Styled text`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}

func TestRenderLayerWithContextScopesTextareaID(t *testing.T) {
	t.Parallel()

	body := RenderLayerWithContext("textarea-main", validConfig(), card.RenderContext{DOMIDPrefix: "game-world-card"}).Render()
	for _, marker := range []string{
		`id="game-world-card-textarea-main-layer"`,
		`data-component-id="textarea-main"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}

func validConfig() Config {
	return Config{
		Content:    "Start designing this card.",
		FontFamily: "system",
		FontSizePX: 16,
		FontWeight: 400,
		FontStyle:  "normal",
		Color:      "#cbd5e1",
		Align:      "left",
		Position:   "center",
		X:          50,
		Y:          50,
		CSS:        "",
	}
}

func textareaIssuePaths(issues []design.Issue) map[string]bool {
	paths := make(map[string]bool, len(issues))
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	return paths
}
