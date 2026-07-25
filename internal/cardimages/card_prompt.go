package cardimages

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
)

type CardPromptInput struct {
	ID      string
	Name    string
	Kind    string
	Setting string
	Tags    []string
	Context []string
}

func BuildCardPrompt(input CardPromptInput, seed int64) (Prompt, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Name = strings.TrimSpace(input.Name)
	input.Kind = strings.TrimSpace(input.Kind)
	if input.ID == "" {
		return Prompt{}, fmt.Errorf("card id is required")
	}
	if input.Name == "" {
		return Prompt{}, fmt.Errorf("card %q name is required", input.ID)
	}

	random := rand.New(rand.NewSource(cardSeed(seed, input.ID)))
	primary := choose(random, natureGrowths)
	secondary := chooseDifferent(random, natureGrowths, primary)
	location := strings.TrimSpace(input.Setting)
	if location == "" {
		location = choose(random, locations)
	}
	selection := Selection{
		Technology:      input.Name,
		NaturePrimary:   primary,
		NatureSecondary: secondary,
		Location:        location,
		Awakening:       choose(random, awakeningSigns),
		Lighting:        choose(random, lightingDirections),
	}
	description := fmt.Sprintf(
		"The %s is the clear focal subject in %s. Organic growths—%s and %s—have spread across it. A sign of reawakening is visible: %s.",
		input.Name,
		selection.Location,
		selection.NaturePrimary,
		selection.NatureSecondary,
		selection.Awakening,
	)
	context := cleanContext(input.Context)
	contextText := "No additional authored context."
	if len(context) > 0 {
		contextText = strings.Join(context, " ")
	}
	tags := cleanOverrides(input.Tags)
	tagText := "none"
	if len(tags) > 0 {
		tagText = strings.Join(tags, ", ")
	}

	text := fmt.Sprintf(`Card:
%s

Scene:
%s

Authored game context (use as narrative reference; do not render these words):
%s

Card role and tags:
%s. Tags: %s.

Interpretation:
%s

Story:
An advanced civilization vanished long ago. Nature has reclaimed its technology, but the machinery is quietly beginning to awaken.

Composition:
Vertical 5:7 collectible-card background artwork. Make the named card subject immediately recognizable, with one clear focal point, layered depth, a readable silhouette, and quieter negative space near the upper and lower edges for overlaid card text. Fill the canvas.

Visual direction:
Grounded fantasy archaeology, believable aged metal, glass, paper, stone, and ceramic as appropriate, damp organic textures, %s, restrained green, copper, and amber palette, atmospheric light, detailed painterly realism. Let the authored context influence small environmental storytelling details.

Constraints:
Background artwork only. Do not render the card name, authored context, tags, passwords, numbers, or any other readable text. No letters, logos, watermark, decorative card border, frame, overlay interface, modern consumer electronics, or visible people. Physical controls and ancient machine displays are allowed, but they must have no readable symbols.`, input.Name, description, contextText, input.Kind, tagText, cardInterpretation(input), selection.Lighting)

	return Prompt{Description: description, Text: text, Selection: selection}, nil
}

func cardInterpretation(input CardPromptInput) string {
	for _, tag := range input.Tags {
		switch strings.ToLower(strings.TrimSpace(tag)) {
		case "component":
			return "Interpret this component card as a removable piece of ancient physical control hardware, not as software or a modern user interface."
		case "image":
			return "Treat the subject as a small, weathered physical photograph discovered among the ruins, with its erased image suggested by faded chemistry and damaged paper."
		}
	}
	switch input.Kind {
	case "world":
		return "Treat the subject as a persistent part of the environment. Scale it naturally for what it is: architecture should feel imposing, while machinery or found objects should remain physically believable."
	case "item":
		return "Treat the subject as a tangible artifact that could be found, carried, and fitted into an ancient machine."
	case "clue":
		return "Treat the subject as a physical clue discovered in the abandoned complex, with its meaning suggested through wear, placement, and surrounding objects rather than readable writing."
	default:
		return "Treat the subject as a tangible part of the abandoned fantasy civilization."
	}
}

func cleanContext(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.Join(strings.Fields(value), " ")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func cardSeed(seed int64, cardID string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(cardID))
	return int64(hash.Sum64() ^ uint64(seed))
}
