package cardimages

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildPromptsIsDeterministicAndUnique(t *testing.T) {
	options := PromptOptions{Count: 10, Seed: 42}
	first, err := BuildPrompts(options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildPrompts(options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same seed produced different prompts")
	}

	seen := make(map[string]bool, len(first))
	for _, prompt := range first {
		if seen[prompt.Text] {
			t.Fatalf("duplicate prompt: %q", prompt.Text)
		}
		seen[prompt.Text] = true
		for _, required := range []string{
			prompt.Selection.Technology,
			prompt.Selection.NaturePrimary,
			prompt.Selection.NatureSecondary,
			"Vertical 5:7 collectible-card background artwork",
			"No words, lettering, numbers",
		} {
			if !strings.Contains(prompt.Text, required) {
				t.Errorf("prompt is missing %q", required)
			}
		}
	}
}

func TestBuildPromptsHonorsOverrides(t *testing.T) {
	prompts, err := BuildPrompts(PromptOptions{
		Count:              2,
		Seed:               7,
		TechnologyOverride: "  copper washer  ",
		NatureOverrides:    []string{"moss", "fungi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, prompt := range prompts {
		if prompt.Selection.Technology != "copper washer" ||
			prompt.Selection.NaturePrimary != "moss" ||
			prompt.Selection.NatureSecondary != "fungi" {
			t.Fatalf("overrides were not preserved: %#v", prompt.Selection)
		}
	}
}

func TestBuildPromptsRejectsInvalidOptions(t *testing.T) {
	tests := []PromptOptions{
		{Count: 0},
		{Count: MaxBatchSize + 1},
		{Count: 1, NatureOverrides: []string{"moss", "fungi", "vines"}},
	}
	for _, options := range tests {
		if _, err := BuildPrompts(options); err == nil {
			t.Fatalf("BuildPrompts(%#v) succeeded", options)
		}
	}
}
