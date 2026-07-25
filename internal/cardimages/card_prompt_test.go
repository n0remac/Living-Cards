package cardimages

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildCardPromptUsesAuthoredCardDetails(t *testing.T) {
	input := CardPromptInput{
		ID:      "glass-fuse",
		Name:    "Glass Fuse",
		Kind:    "item",
		Setting: "the abandoned fuse room",
		Tags:    []string{"item", "fuse", "power"},
		Context: []string{"A thin fuse, still clear enough to carry current."},
	}
	first, err := BuildCardPrompt(input, 42)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildCardPrompt(input, 42)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same card and seed produced different prompts")
	}
	for _, required := range []string{
		"Glass Fuse",
		"the abandoned fuse room",
		"A thin fuse, still clear enough to carry current.",
		"item, fuse, power",
		"Do not render the card name",
	} {
		if !strings.Contains(first.Text, required) {
			t.Errorf("prompt is missing %q", required)
		}
	}
	if first.Selection.NaturePrimary == first.Selection.NatureSecondary {
		t.Fatal("card prompt selected the same nature growth twice")
	}
}

func TestBuildCardPromptInterpretsComponentAsHardware(t *testing.T) {
	prompt, err := BuildCardPrompt(CardPromptInput{
		ID: "slider-component", Name: "Slider Component", Kind: "item", Tags: []string{"component"},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prompt.Text, "removable piece of ancient physical control hardware") {
		t.Fatalf("component interpretation missing from prompt: %s", prompt.Text)
	}
}

func TestBuildCardPromptRejectsMissingIdentity(t *testing.T) {
	for _, input := range []CardPromptInput{
		{Name: "Missing ID"},
		{ID: "missing-name"},
	} {
		if _, err := BuildCardPrompt(input, 1); err == nil {
			t.Fatalf("BuildCardPrompt(%#v) succeeded", input)
		}
	}
}
