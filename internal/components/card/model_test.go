package card_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	"github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

func TestDefaultDocumentShape(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	if document.CardID != card.DefaultCardID || document.Name != "Empty Card" {
		t.Fatalf("document = %#v", document)
	}
	if document.Root.ID != card.DefaultRootID || document.Root.ComponentKind != card.Kind {
		t.Fatalf("root = %#v", document.Root)
	}
	if len(document.Root.Children) != 3 {
		t.Fatalf("children = %#v", document.Root.Children)
	}
	for index, expected := range []string{background.Kind, border.Kind, textarea.Kind} {
		if document.Root.Children[index].ComponentKind != expected {
			t.Fatalf("child %d type = %q, want %q", index, document.Root.Children[index].ComponentKind, expected)
		}
	}
}

func TestRenderDocumentComposesShellAndTextareaLayer(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	node, err := card.RenderDocument(document, testRegistry())
	if err != nil {
		t.Fatalf("RenderDocument() error = %v", err)
	}
	body := node.Render()
	for _, marker := range []string{
		`id="draft-card-preview"`,
		`data-component-id="card-root"`,
		`data-component-kind="card"`,
		`background-color: #111827`,
		`border-radius: 24px`,
		`padding: 24px`,
		`data-component-kind="textarea"`,
		`Start designing this card.`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}

func TestRenderDocumentRendersShapeLayer(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	document.Root.Children = append(document.Root.Children, card.Node{
		ID:            card.DefaultShapeID,
		ComponentKind: shape.Kind,
		Config: mustRaw(t, shape.Config{
			Shape:           "triangle",
			X:               20,
			Y:               24,
			Width:           32,
			Height:          28,
			BackgroundColor: "#38bdf8",
			BorderColor:     "#111827",
			BorderWidthPX:   2,
		}),
	})

	node, err := card.RenderDocument(document, testRegistry())
	if err != nil {
		t.Fatalf("RenderDocument() error = %v", err)
	}
	body := node.Render()
	for _, marker := range []string{
		`data-component-id="shape-1"`,
		`data-component-kind="shape"`,
		`<svg`,
		`points="50,8 92,88 8,88"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}

func TestRenderDocumentSupportsMultipleLayerComponentsAndCustomID(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	document.CardID = "fixture-card"
	document.Root.Children = append(document.Root.Children,
		card.Node{
			ID:            "textarea-extra",
			ComponentKind: textarea.Kind,
			Config: mustRaw(t, textarea.Config{
				Content:    "Second text layer",
				FontFamily: "system",
				FontSizePX: 16,
				FontWeight: 500,
				FontStyle:  "normal",
				Color:      "#f8fafc",
				Align:      "center",
				Position:   "center",
			}),
		},
		card.Node{
			ID:            "shape-extra",
			ComponentKind: shape.Kind,
			Config: mustRaw(t, shape.Config{
				Shape:           "diamond",
				X:               64,
				Y:               36,
				Width:           18,
				Height:          18,
				BackgroundColor: "#38bdf8",
				BorderColor:     "#111827",
				BorderWidthPX:   1,
			}),
		},
		card.Node{
			ID:            "image-extra",
			ComponentKind: imagecomponent.Kind,
			Config:        mustRaw(t, imagecomponent.DefaultConfig()),
		},
		card.Node{
			ID:            "slider-extra",
			ComponentKind: slider.Kind,
			Config: mustRaw(t, slider.Config{
				Label: "Output",
				Min:   0,
				Max:   100,
				Step:  1,
				Value: 73,
			}),
		},
	)

	node, err := card.RenderDocumentWithID(document, testRegistry(), "game-card-fixture")
	if err != nil {
		t.Fatalf("RenderDocumentWithID() error = %v", err)
	}
	body := node.Render()
	for _, marker := range []string{
		`id="game-card-fixture"`,
		`data-card-id="fixture-card"`,
		`data-component-id="textarea-extra"`,
		`Second text layer`,
		`data-component-id="shape-extra"`,
		`data-component-id="image-extra"`,
		`data-component-kind="image"`,
		`data-component-id="slider-extra"`,
		`data-component-kind="slider"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
	if strings.Contains(body, `id="draft-card-preview"`) {
		t.Fatalf("custom render should not include default preview id:\n%s", body)
	}
}

func TestRenderDocumentScopesLayerIDsAndPreservesComponentIDs(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	first, err := card.RenderDocumentWithOptions(document, testRegistry(), card.RenderOptions{
		ElementID:   "world-card",
		DOMIDPrefix: "world-card",
	})
	if err != nil {
		t.Fatalf("RenderDocumentWithOptions() first error = %v", err)
	}
	second, err := card.RenderDocumentWithOptions(document, testRegistry(), card.RenderOptions{
		ElementID:   "library-card",
		DOMIDPrefix: "library-card",
	})
	if err != nil {
		t.Fatalf("RenderDocumentWithOptions() second error = %v", err)
	}
	body := first.Render() + second.Render()
	for _, marker := range []string{
		`id="world-card"`,
		`id="world-card-textarea-main-layer"`,
		`id="library-card"`,
		`id="library-card-textarea-main-layer"`,
		`data-component-id="textarea-main"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
	if strings.Contains(body, `id="textarea-main-layer"`) {
		t.Fatalf("scoped render should not include unscoped layer id:\n%s", body)
	}
}

func TestRenderDocumentLaterShellContributionsWin(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	document.Root.Children[0].Config = mustRaw(t, background.Config{
		BackgroundColor: "#111827",
		CSS:             "box-shadow: 0 0 10px red;",
	})
	document.Root.Children[1].Config = mustRaw(t, border.Config{
		BorderWidthPX:  1,
		BorderRadiusPX: 24,
		BorderColor:    "#ffffff",
		CSS:            "box-shadow: 0 0 10px blue;",
	})

	node, err := card.RenderDocument(document, testRegistry())
	if err != nil {
		t.Fatalf("RenderDocument() error = %v", err)
	}
	body := node.Render()
	if !strings.Contains(body, "box-shadow: 0 0 10px blue") {
		t.Fatalf("render missing border shadow:\n%s", body)
	}
	if strings.Contains(body, "box-shadow: 0 0 10px red") {
		t.Fatalf("background shadow should be overwritten:\n%s", body)
	}
}

func testRegistry() *card.Registry {
	return card.MustNewRegistry(
		background.Definition(),
		border.Definition(),
		textarea.Definition(),
		shape.Definition(),
		imagecomponent.Definition(),
		slider.Definition(),
	)
}

func mustRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}
