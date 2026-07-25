package cardassets

import "testing"

func TestBackgroundURL(t *testing.T) {
	t.Parallel()

	if got := BackgroundURL("rusted-cell-door"); got != "/assets/card-backgrounds/rusted-cell-door.webp" {
		t.Fatalf("BackgroundURL() = %q", got)
	}
	for _, assetID := range []string{
		"../rusted-cell-door",
		"Rusted-Cell-Door",
		"rusted-cell-door.webp",
		"https://example.com/card",
		"rusted cell door",
	} {
		if got := BackgroundURL(assetID); got != "" {
			t.Fatalf("BackgroundURL(%q) = %q", assetID, got)
		}
	}
}
