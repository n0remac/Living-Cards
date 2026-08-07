package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/n0remac/Living-Card/internal/cardimages"
	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
)

type options struct {
	count      int
	seed       int64
	dryRun     bool
	outputDir  string
	model      string
	quality    string
	size       string
	format     string
	technology string
	nature     string
	force      bool
	gameCards  bool
}

type plannedAsset struct {
	prompt       cardimages.Prompt
	baseName     string
	deckID       string
	cardID       string
	cardName     string
	imagePath    string
	manifestPath string
}

type deckFile struct {
	ID    string     `json:"id"`
	Cards []deckCard `json:"cards"`
}

type deckCard struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Kind      string                     `json:"kind"`
	Tags      []string                   `json:"tags"`
	Documents map[string]json.RawMessage `json:"documents"`
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cardimages:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, arguments []string) error {
	flags := flag.NewFlagSet("cardimages", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var options options
	flags.IntVar(&options.count, "count", 1, "number of images to generate")
	flags.Int64Var(&options.seed, "seed", 0, "prompt seed; zero chooses and prints a time-based seed")
	flags.BoolVar(&options.dryRun, "dry-run", false, "print prompts without calling OpenAI or writing files")
	flags.StringVar(&options.outputDir, "out", "web/assets/card-backgrounds", "output directory")
	flags.StringVar(&options.model, "model", "gpt-image-2", "OpenAI image model")
	flags.StringVar(&options.quality, "quality", "medium", "image quality: low, medium, or high")
	flags.StringVar(&options.size, "size", "960x1344", "output dimensions")
	flags.StringVar(&options.format, "format", "webp", "output format: png, jpeg, or webp")
	flags.StringVar(&options.technology, "tech", "", "override the selected technology element")
	flags.StringVar(&options.nature, "nature", "", "override one or two comma-separated nature elements")
	flags.BoolVar(&options.force, "force", false, "overwrite existing image and manifest files")
	flags.BoolVar(&options.gameCards, "game-cards", false, "generate one image for every card in internal/game/decks")
	if err := flags.Parse(arguments); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if options.seed == 0 {
		options.seed = time.Now().UnixNano()
	}
	jobs, err := buildJobs(options)
	if err != nil {
		return err
	}
	generateOptions := cardimages.GenerateOptions{
		Model: options.model, Size: options.size, Quality: options.quality, Format: options.format,
	}
	if err := validateCLIOptions(generateOptions, options.outputDir); err != nil {
		return err
	}

	fmt.Printf("seed: %d\n", options.seed)
	for index, job := range jobs {
		fmt.Printf("\n[%d/%d] %s\n%s\n", index+1, len(jobs), job.baseName, job.prompt.Text)
	}
	if options.dryRun {
		return nil
	}

	assets := make([]plannedAsset, 0, len(jobs))
	for _, job := range jobs {
		asset := job
		asset.imagePath = filepath.Join(options.outputDir, asset.baseName+"."+options.format)
		asset.manifestPath = filepath.Join(options.outputDir, asset.baseName+".json")
		if err := cardimages.EnsureAvailable(asset.imagePath, asset.manifestPath, options.force); err != nil {
			return err
		}
		assets = append(assets, asset)
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required unless -dry-run is used")
	}
	client := cardimages.Client{APIKey: apiKey}
	for index, asset := range assets {
		fmt.Printf("\ngenerating %d/%d: %s\n", index+1, len(assets), asset.imagePath)
		request := generateOptions
		request.Prompt = asset.prompt.Text
		result, err := client.Generate(ctx, request)
		if err != nil {
			return fmt.Errorf("generate %s: %w", asset.baseName, err)
		}
		manifest := cardimages.Manifest{
			Description: asset.prompt.Description,
			Selection:   asset.prompt.Selection,
			Prompt:      asset.prompt.Text,
			DeckID:      asset.deckID,
			CardID:      asset.cardID,
			CardName:    asset.cardName,
			Seed:        options.seed,
			Index:       index + 1,
			Model:       options.model,
			Size:        options.size,
			Quality:     options.quality,
			Format:      options.format,
			ImageFile:   filepath.Base(asset.imagePath),
			RequestID:   result.RequestID,
			CreatedAt:   time.Now().UTC(),
		}
		if err := cardimages.WriteAsset(asset.imagePath, asset.manifestPath, result.Image, manifest); err != nil {
			return err
		}
		fmt.Printf("saved %s and %s\n", asset.imagePath, asset.manifestPath)
	}
	return nil
}

func buildJobs(options options) ([]plannedAsset, error) {
	if options.gameCards {
		if options.count != 1 || strings.TrimSpace(options.technology) != "" || strings.TrimSpace(options.nature) != "" {
			return nil, fmt.Errorf("-count, -tech, and -nature cannot be used with -game-cards")
		}
		root, err := projectRoot()
		if err != nil {
			return nil, err
		}
		return loadGameCardJobs(root, options.seed)
	}

	nature := strings.Split(options.nature, ",")
	if strings.TrimSpace(options.nature) == "" {
		nature = nil
	}
	prompts, err := cardimages.BuildPrompts(cardimages.PromptOptions{
		Count:              options.count,
		Seed:               options.seed,
		TechnologyOverride: options.technology,
		NatureOverrides:    nature,
	})
	if err != nil {
		return nil, err
	}
	jobs := make([]plannedAsset, 0, len(prompts))
	for index, prompt := range prompts {
		jobs = append(jobs, plannedAsset{
			prompt:   prompt,
			baseName: cardimages.AssetBaseName(prompt, options.seed, index+1),
		})
	}
	return jobs, nil
}

func loadGameCardJobs(root string, seed int64) ([]plannedAsset, error) {
	paths, err := filepath.Glob(filepath.Join(root, "internal", "game", "decks", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("find game decks: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no game decks found")
	}
	registry := catalog.MustNew()
	jobs := make([]plannedAsset, 0)
	seenCardIDs := make(map[string]string)
	seenDeckIDs := make(map[string]string)
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var deck deckFile
		if err := json.Unmarshal(raw, &deck); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		deck.ID = strings.TrimSpace(deck.ID)
		if deck.ID == "" {
			return nil, fmt.Errorf("%s has no deck id", path)
		}
		if previousPath, exists := seenDeckIDs[deck.ID]; exists {
			return nil, fmt.Errorf("deck id %q appears in %s and %s", deck.ID, previousPath, path)
		}
		seenDeckIDs[deck.ID] = path
		for _, definition := range deck.Cards {
			if previousDeck, exists := seenCardIDs[definition.ID]; exists {
				return nil, fmt.Errorf("card id %q appears in decks %q and %q", definition.ID, previousDeck, deck.ID)
			}
			seenCardIDs[definition.ID] = deck.ID
			documents, err := decodeCardDocuments(registry, definition)
			if err != nil {
				return nil, fmt.Errorf("deck %q card %q: %w", deck.ID, definition.ID, err)
			}
			prompt, err := cardimages.BuildCardPrompt(cardimages.CardPromptInput{
				ID:      definition.ID,
				Name:    definition.Name,
				Kind:    definition.Kind,
				Setting: deckSetting(deck.ID),
				Tags:    definition.Tags,
				Context: cardText(documents),
			}, seed)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, plannedAsset{
				prompt: prompt, baseName: definition.ID, deckID: deck.ID, cardID: definition.ID, cardName: definition.Name,
			})
		}
	}
	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].deckID == jobs[j].deckID {
			return jobs[i].cardID < jobs[j].cardID
		}
		return jobs[i].deckID < jobs[j].deckID
	})
	return jobs, nil
}

func deckSetting(deckID string) string {
	switch deckID {
	case "archive_guardian":
		return "the awakened archive stacks"
	case "archive_terminal":
		return "the overgrown archive terminal chamber"
	case "fuse_room":
		return "the abandoned fuse room"
	case "generator_room":
		return "the dormant generator hall"
	case "seeded_world":
		return "the ruined cell complex"
	default:
		return ""
	}
}

func decodeCardDocuments(registry *card.Registry, definition deckCard) (map[string]card.Document, error) {
	definition.ID = strings.TrimSpace(definition.ID)
	if definition.ID == "" {
		return nil, fmt.Errorf("card id is required")
	}
	if len(definition.Documents) == 0 {
		return nil, fmt.Errorf("documents are required")
	}
	documents := make(map[string]card.Document, len(definition.Documents))
	for variant, raw := range definition.Documents {
		document, issues := registry.DecodeDocument(raw)
		if len(issues) > 0 {
			return nil, fmt.Errorf("document %q at %s: %s", variant, issues[0].Path, issues[0].Message)
		}
		if document.CardID != definition.ID {
			return nil, fmt.Errorf("document %q has card_id %q", variant, document.CardID)
		}
		documents[variant] = document
	}
	return documents, nil
}

func cardText(documents map[string]card.Document) []string {
	variants := make([]string, 0, len(documents))
	for variant := range documents {
		variants = append(variants, variant)
	}
	sort.Strings(variants)
	var result []string
	seen := map[string]bool{}
	for _, variant := range variants {
		walkCardText(documents[variant].Root, func(content string) {
			content = strings.Join(strings.Fields(content), " ")
			if content == "" || seen[content] {
				return
			}
			seen[content] = true
			result = append(result, content)
		})
	}
	return result
}

func walkCardText(node card.Node, visit func(string)) {
	if node.ComponentKind == card.KindText {
		var config struct {
			Content string `json:"content"`
		}
		if json.Unmarshal(node.Config, &config) == nil {
			visit(config.Content)
		}
	}
	for _, child := range node.Children {
		walkCardText(child, visit)
	}
}

func projectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("could not locate project root")
		}
		directory = parent
	}
}

func validateCLIOptions(options cardimages.GenerateOptions, outputDir string) error {
	options.Prompt = "validation"
	if _, err := filepath.Abs(strings.TrimSpace(outputDir)); err != nil {
		return fmt.Errorf("invalid output directory: %w", err)
	}
	if strings.TrimSpace(outputDir) == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	client := cardimages.Client{APIKey: "validation"}
	return client.Validate(options)
}
