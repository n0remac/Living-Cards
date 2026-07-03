package textarea

import (
	"testing"

	"github.com/n0remac/Living-Card/internal/fragment"
)

func TestValidateGeneratedAcceptsSafeTextareaFragment(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Safe",
		Fragment: Fragment{
			Content:    "A quiet note.",
			FontFamily: "serif",
			FontSizePX: 18,
			FontWeight: 400,
			FontStyle:  "italic",
			Color:      "rgba(226,232,240,0.92)",
			Align:      "center",
			Position:   "center",
			CSS:        "font-family: Georgia, serif; line-height: 1.5; text-align: center; text-shadow: 0 1px 8px rgba(0,0,0,0.3);",
		},
	}
	if issues := ValidateGenerated(generated); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestNormalizeGeneratedClampsFontSize(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Clamp",
		Fragment:    validFragment(),
	}
	generated.Fragment.FontSizePX = 999
	NormalizeGenerated(&generated)
	if generated.Fragment.FontSizePX != 72 {
		t.Fatalf("FontSizePX = %d, want 72", generated.Fragment.FontSizePX)
	}
}

func TestValidateGeneratedRejectsInvalidTextareaFields(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Bad",
		Fragment: Fragment{
			Content:    "",
			FontFamily: "fantasy",
			FontSizePX: 8,
			FontWeight: 900,
			FontStyle:  "oblique",
			Color:      "clear",
			Align:      "justify",
			Position:   "middle",
			CSS:        "position: fixed;",
		},
	}
	issues := ValidateGenerated(generated)
	paths := textareaIssuePaths(issues)
	for _, path := range []string{
		"fragment.content",
		"fragment.font_family",
		"fragment.font_size_px",
		"fragment.font_weight",
		"fragment.font_style",
		"fragment.color",
		"fragment.align",
		"fragment.position",
		"fragment.css",
	} {
		if !paths[path] {
			t.Fatalf("issues missing %s: %#v", path, issues)
		}
	}
}

func TestValidateGeneratedRejectsUnsupportedTextCSSProperty(t *testing.T) {
	t.Parallel()

	part := validFragment()
	part.CSS = "background: red;"
	issues := ValidateGenerated(fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Bad CSS",
		Fragment:    part,
	})
	if len(issues) != 1 || issues[0].Path != "fragment.css" || issues[0].Code != "unsupported_css_property" {
		t.Fatalf("issues = %#v", issues)
	}
}

func validFragment() Fragment {
	return Fragment{
		Content:    "Start designing this card.",
		FontFamily: "system",
		FontSizePX: 16,
		FontWeight: 400,
		FontStyle:  "normal",
		Color:      "#cbd5e1",
		Align:      "left",
		Position:   "center",
		CSS:        "",
	}
}

func textareaIssuePaths(issues []fragment.Issue) map[string]bool {
	paths := make(map[string]bool, len(issues))
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	return paths
}
