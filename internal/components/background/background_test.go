package background

import (
	"testing"

	"github.com/n0remac/Living-Card/internal/design"
)

func TestValidateGeneratedAcceptsSafeBackgroundCSS(t *testing.T) {
	t.Parallel()

	generated := design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Safe",
		Config: Config{
			BackgroundColor: "rgba(15,23,42,0.9)",
			CSS:             "background: linear-gradient(135deg, #111827, rgba(14,165,233,0.24)); box-shadow: inset 0 0 30px rgba(255,255,255,0.08);",
		},
	}
	if issues := ValidateGenerated(generated); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestRandomGeneratedBackgroundValidates(t *testing.T) {
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

func TestValidateGeneratedRejectsUnsafeBackgroundCSS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		css  string
	}{
		{name: "url", css: "background: url(https://example.com/bg.png);"},
		{name: "selector", css: ".card { background: red; }"},
		{name: "unsupported", css: "position: fixed;"},
		{name: "html", css: "<b>bad</b>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			issues := ValidateGenerated(design.GeneratedConfig[Config]{
				ComponentKind: Kind,
				Description:   "Bad",
				Config: Config{
					BackgroundColor: "#111827",
					CSS:             tt.css,
				},
			})
			if len(issues) == 0 || issues[0].Path != "config.css" {
				t.Fatalf("issues = %#v", issues)
			}
		})
	}
}

func TestValidateGeneratedRejectsInvalidBackgroundColor(t *testing.T) {
	t.Parallel()

	issues := ValidateGenerated(design.GeneratedConfig[Config]{
		ComponentKind: Kind,
		Description:   "Bad",
		Config: Config{
			BackgroundColor: "not-a-color",
			CSS:             "",
		},
	})
	if len(issues) != 1 || issues[0].Path != "config.background_color" {
		t.Fatalf("issues = %#v", issues)
	}
}
