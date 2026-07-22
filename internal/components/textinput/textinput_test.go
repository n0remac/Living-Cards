package textinput

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
)

func TestNormalizeAndValidateConfig(t *testing.T) {
	t.Parallel()

	part := NormalizeConfig(Config{InputType: " PASSWORD ", X: -4, Y: 120, Width: 4})
	if part.FormID != "controller-form" || part.Name != "text" || part.InputType != "password" || part.X != 0 || part.Y != 100 || part.Width != 12 {
		t.Fatalf("part = %#v", part)
	}
	if issues := ValidateConfig(part); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
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
		`data-component-kind="textinput"`,
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
