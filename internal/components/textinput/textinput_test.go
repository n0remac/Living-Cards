package textinput

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
)

func TestNormalizeAndValidateConfig(t *testing.T) {
	t.Parallel()

	part := NormalizeConfig(Config{FormID: " form ", Name: " field ", Label: " Label ", InputType: " PASSWORD ", X: -4, Y: 120, Width: 4})
	if part.FormID != "form" || part.Name != "field" || part.Label != "Label" || part.InputType != "PASSWORD" || part.X != -4 || part.Y != 120 || part.Width != 4 {
		t.Fatalf("part = %#v", part)
	}
	if issues := ValidateConfig(part); len(issues) == 0 {
		t.Fatal("ValidateConfig() issues = nil, want noncanonical type and invalid ranges")
	}
	if issues := ValidateConfig(Config{FormID: "bad id", Name: "password", Label: "Password", InputType: "email", Width: 50}); len(issues) < 2 {
		t.Fatalf("issues = %#v, want invalid form id and type", issues)
	}
}

func TestRenderLayerUsesPrefixedSemanticForm(t *testing.T) {
	t.Parallel()

	body := RenderLayerWithContext("archive-password-input", Config{
		FormID:      "archive-login",
		Name:        "password",
		Label:       "Password",
		Placeholder: "Enter password",
		InputType:   "password",
	}, card.RenderContext{DOMIDPrefix: "game-world-archive"}).Render()
	for _, marker := range []string{
		`id="game-world-archive-archive-login"`,
		`data-card-form`,
		`data-form-id="archive-login"`,
		`data-component-kind="text_input"`,
		`name="password"`,
		`type="password"`,
		`for="game-world-archive-archive-password-input-input"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
	if strings.Contains(body, `value=`) {
		t.Fatalf("render must not persist a runtime value:\n%s", body)
	}
}
