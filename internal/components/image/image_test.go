package imagecomponent

import (
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/fragment"
)

func TestValidateGeneratedAcceptsSafeImageDataURL(t *testing.T) {
	t.Parallel()

	generated := fragment.Generated[Fragment]{
		Target:      Type,
		Description: "Safe image",
		Fragment:    DefaultFragment(),
	}
	NormalizeGenerated(&generated)
	if issues := ValidateGenerated(generated); len(issues) > 0 {
		t.Fatalf("ValidateGenerated() issues = %#v", issues)
	}
}

func TestValidateGeneratedRejectsUnsafeImageSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		code string
	}{
		{name: "external url", src: "https://example.test/key.png", code: "invalid_data_url"},
		{name: "svg", src: "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=", code: "invalid_mime_type"},
		{name: "invalid base64", src: "data:image/png;base64,%%%%", code: "invalid_base64"},
		{name: "unsafe marker", src: "javascript:alert(1)", code: "unsafe_value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			part := DefaultFragment()
			part.Src = test.src
			generated := fragment.Generated[Fragment]{Target: Type, Description: "Unsafe", Fragment: part}
			NormalizeGenerated(&generated)
			issues := ValidateGenerated(generated)
			if len(issues) == 0 || issues[0].Code != test.code {
				t.Fatalf("issues = %#v, want first code %q", issues, test.code)
			}
		})
	}
}

func TestRenderLayerIncludesImageAttributes(t *testing.T) {
	t.Parallel()

	body := RenderLayer("image-1", DefaultFragment()).Render()
	for _, marker := range []string{
		`id="image-1-layer"`,
		`data-component-id="image-1"`,
		`data-component-type="image"`,
		`<img`,
		`src="data:image/gif;base64`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}
