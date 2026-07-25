package cardimages

import (
	"fmt"
	"math/rand"
	"strings"
)

const MaxBatchSize = 20

type PromptOptions struct {
	Count              int
	Seed               int64
	TechnologyOverride string
	NatureOverrides    []string
}

type Selection struct {
	Technology      string `json:"technology"`
	NaturePrimary   string `json:"nature_primary"`
	NatureSecondary string `json:"nature_secondary"`
	Location        string `json:"location"`
	Awakening       string `json:"awakening"`
	Lighting        string `json:"lighting"`
}

type Prompt struct {
	Description string    `json:"description"`
	Text        string    `json:"prompt"`
	Selection   Selection `json:"selection"`
}

func BuildPrompts(options PromptOptions) ([]Prompt, error) {
	if options.Count < 1 || options.Count > MaxBatchSize {
		return nil, fmt.Errorf("count must be between 1 and %d", MaxBatchSize)
	}
	technologyOverride := strings.TrimSpace(options.TechnologyOverride)
	natureOverrides := cleanOverrides(options.NatureOverrides)
	if len(natureOverrides) > 2 {
		return nil, fmt.Errorf("nature accepts at most two comma-separated elements")
	}

	random := rand.New(rand.NewSource(options.Seed))
	prompts := make([]Prompt, 0, options.Count)
	seen := make(map[string]struct{}, options.Count)
	for len(prompts) < options.Count {
		selection := Selection{
			Technology:      choose(random, technologyRelics),
			NaturePrimary:   choose(random, natureGrowths),
			NatureSecondary: choose(random, natureGrowths),
			Location:        choose(random, locations),
			Awakening:       choose(random, awakeningSigns),
			Lighting:        choose(random, lightingDirections),
		}
		if technologyOverride != "" {
			selection.Technology = technologyOverride
		}
		if len(natureOverrides) > 0 {
			selection.NaturePrimary = natureOverrides[0]
		}
		if len(natureOverrides) > 1 {
			selection.NatureSecondary = natureOverrides[1]
		}
		if selection.NatureSecondary == selection.NaturePrimary {
			selection.NatureSecondary = chooseDifferent(random, natureGrowths, selection.NaturePrimary)
		}

		key := strings.Join([]string{
			selection.Technology,
			selection.NaturePrimary,
			selection.NatureSecondary,
			selection.Location,
			selection.Awakening,
			selection.Lighting,
		}, "\x00")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		prompts = append(prompts, makePrompt(selection))
	}
	return prompts, nil
}

func makePrompt(selection Selection) Prompt {
	description := fmt.Sprintf(
		"A %s rests inside %s. Its ancient surfaces are being reclaimed by %s and %s. A sign of reawakening is visible: %s.",
		selection.Technology,
		selection.Location,
		selection.NaturePrimary,
		selection.NatureSecondary,
		selection.Awakening,
	)
	text := fmt.Sprintf(`Scene:
%s

Story:
An advanced civilization vanished long ago. Nature has reclaimed its technology, but the machinery is quietly beginning to awaken.

Composition:
Vertical 5:7 collectible-card background artwork. One clear focal object, layered depth, a readable silhouette, and quieter negative space near the upper and lower edges for card text. The scene should fill the canvas.

Visual direction:
Grounded fantasy archaeology, believable aged metal and ceramic, damp organic textures, %s, restrained green, copper, and amber palette, atmospheric light, detailed painterly realism.

Constraints:
Background artwork only. No words, lettering, numbers, logos, watermark, interface, decorative card border, frame, modern consumer electronics, or visible people.`, description, selection.Lighting)
	return Prompt{Description: description, Text: text, Selection: selection}
}

func cleanOverrides(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func choose(random *rand.Rand, values []string) string {
	return values[random.Intn(len(values))]
}

func chooseDifferent(random *rand.Rand, values []string, excluded string) string {
	start := random.Intn(len(values))
	for offset := range len(values) {
		candidate := values[(start+offset)%len(values)]
		if candidate != excluded {
			return candidate
		}
	}
	return values[start]
}
