package catalog_test

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
)

func TestDefaultCatalogOrderAndCanonicalVocabulary(t *testing.T) {
	registry := catalog.MustNew()
	want := []string{card.Kind, card.KindBackground, card.KindBorder, card.KindText, card.KindShape, card.KindImage, card.KindSlider, card.KindTextInput, card.KindButton}
	definitions := registry.Definitions()
	if len(definitions) != len(want) {
		t.Fatalf("definitions = %d, want %d", len(definitions), len(want))
	}
	for index, kind := range want {
		if definitions[index].Kind() != kind {
			t.Fatalf("definitions[%d] = %q, want %q", index, definitions[index].Kind(), kind)
		}
	}
	for _, reserved := range []string{"heading", "form", "stack", "grid", "textarea", "textinput"} {
		if _, ok := registry.Lookup(reserved); ok {
			t.Fatalf("reserved or legacy kind %q is registered", reserved)
		}
	}
}

func TestAllDeclaredGenerationMetadataPassesTheDefinitionCodec(t *testing.T) {
	registry := catalog.MustNew()
	aiKinds := map[string]bool{card.KindBackground: true, card.KindBorder: true, card.KindText: true, card.KindImage: true}
	for _, definition := range registry.Definitions() {
		generation, ok := definition.Generation()
		if !ok {
			if aiKinds[definition.Kind()] {
				t.Fatalf("%s is missing AI generation metadata", definition.Kind())
			}
			continue
		}
		if generation.SupportsAI() != aiKinds[definition.Kind()] {
			t.Fatalf("%s SupportsAI() = %v", definition.Kind(), generation.SupportsAI())
		}
		if generation.SupportsAI() {
			if _, issues := generation.CanonicalizeEnvelope(json.RawMessage(generation.Example())); len(issues) != 0 {
				t.Fatalf("%s AI example issues = %#v", definition.Kind(), issues)
			}
		}
		if generation.SupportsRandom() {
			for _, seed := range []int64{1, 2, 3, 4, 5} {
				if _, issues := generation.Random(seed, 3); len(issues) != 0 {
					t.Fatalf("%s random seed %d issues = %#v", definition.Kind(), seed, issues)
				}
			}
		}
	}
}

func TestProductionConsumersDoNotImportLeafPackages(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	leafPrefix := "github.com/n0remac/Living-Card/internal/components/"
	allowed := map[string]bool{
		leafPrefix + "card":   true,
		leafPrefix + "schema": true,
	}
	for _, packageDir := range []string{"internal/game", "internal/web"} {
		err := filepath.WalkDir(filepath.Join(root, packageDir), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				value, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					return err
				}
				if value == leafPrefix+"catalog" && filepath.Base(path) == "handlers.go" {
					continue
				}
				if strings.HasPrefix(value, leafPrefix) && !allowed[value] {
					t.Errorf("%s imports concrete component package %q", path, value)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
