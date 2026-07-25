package button

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
)

func TestNormalizeAndValidateConfig(t *testing.T) {
	t.Parallel()

	part := NormalizeConfig(Config{FormID: " form ", Label: " Submit ", X: -3, Y: 140, Width: 2})
	if part.FormID != "form" || part.Label != "Submit" || part.X != -3 || part.Y != 140 || part.Width != 2 {
		t.Fatalf("part = %#v", part)
	}
	if issues := configIssues(part); len(issues) == 0 {
		t.Fatal("configIssues() issues = nil, want invalid ranges")
	}
	if issues := configIssues(Config{FormID: "bad id", Label: "Submit", Width: 44}); len(issues) == 0 {
		t.Fatal("configIssues() issues = nil, want invalid form id")
	}
}

func configIssues(config Config) []schema.Issue {
	raw, _ := json.Marshal(config)
	_, issues := Definition().CanonicalizeConfig(card.RawConfig{Present: true, Value: raw})
	return issues
}

func TestRenderLayerTargetsPrefixedExternalForm(t *testing.T) {
	t.Parallel()

	body := RenderLayerWithContext("archive-submit-button", Config{FormID: "archive-login", Label: "Unlock"}, card.RenderContext{DOMIDPrefix: "game-world-archive"}).Render()
	for _, marker := range []string{
		`id="game-world-archive-archive-submit-button-layer"`,
		`form="game-world-archive-archive-login"`,
		`type="submit"`,
		`data-component-kind="button"`,
		`data-form-id="archive-login"`,
		`Unlock`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}
