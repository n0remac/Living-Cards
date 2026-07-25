package background

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestValidateGeneratedAcceptsSafeBackgroundCSS(t *testing.T) {
	t.Parallel()

	config := Config{
		BackgroundColor: "rgba(15,23,42,0.9)",
		CSS:             "background: linear-gradient(135deg, #111827, rgba(14,165,233,0.24)); box-shadow: inset 0 0 30px rgba(255,255,255,0.08);",
	}
	if issues := configIssues(config); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestRandomGeneratedBackgroundValidates(t *testing.T) {
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

			issues := configIssues(Config{BackgroundColor: "#111827", CSS: tt.css})
			if len(issues) == 0 || issues[0].Path != "config.css" {
				t.Fatalf("issues = %#v", issues)
			}
		})
	}
}

func TestValidateGeneratedRejectsInvalidBackgroundColor(t *testing.T) {
	t.Parallel()

	issues := configIssues(Config{BackgroundColor: "not-a-color"})
	if len(issues) != 1 || issues[0].Path != "config.background_color" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestAssetIDValidation(t *testing.T) {
	t.Parallel()

	if issues := configIssues(Config{AssetID: "rusted-cell-door", BackgroundColor: "#111827"}); len(issues) != 0 {
		t.Fatalf("valid asset issues = %#v", issues)
	}
	for _, assetID := range []string{
		"../rusted-cell-door",
		"Rusted-Cell-Door",
		"rusted-cell-door.webp",
		"https://example.com/card",
		"rusted cell door",
	} {
		issues := configIssues(Config{AssetID: assetID, BackgroundColor: "#111827"})
		if len(issues) == 0 || issues[0].Path != "config.asset_id" {
			t.Fatalf("asset %q issues = %#v", assetID, issues)
		}
	}
}

func TestAssetURLRejectsUnsafeIDs(t *testing.T) {
	t.Parallel()

	if got := AssetURL("rusted-cell-door"); got != "/assets/card-backgrounds/rusted-cell-door.webp" {
		t.Fatalf("AssetURL() = %q", got)
	}
	if got := AssetURL("../rusted-cell-door"); got != "" {
		t.Fatalf("unsafe AssetURL() = %q", got)
	}
}

func TestDefinitionRendersFullBleedAssetLayer(t *testing.T) {
	t.Parallel()

	config, _ := json.Marshal(Config{AssetID: "rusted-cell-door", BackgroundColor: "#201815"})
	contribution, err := Definition().Render(card.Node{ID: "door-background", ComponentKind: Kind, Config: config}, card.RenderContext{DOMIDPrefix: "world"})
	if err != nil {
		t.Fatal(err)
	}
	if len(contribution.Layers) != 1 {
		t.Fatalf("layers = %d", len(contribution.Layers))
	}
	body := contribution.Layers[0].Render()
	for _, marker := range []string{
		`id="world-door-background-layer"`,
		`src="/assets/card-backgrounds/rusted-cell-door.webp"`,
		`alt=""`,
		`aria-hidden="true"`,
		`data-component-kind="background"`,
		`object-fit: cover`,
		`pointer-events: none`,
		`z-index: 0`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q: %s", marker, body)
		}
	}
}

func configIssues(config Config) []schema.Issue {
	raw, _ := json.Marshal(config)
	_, issues := Definition().CanonicalizeConfig(card.RawConfig{Present: true, Value: raw})
	return issues
}
