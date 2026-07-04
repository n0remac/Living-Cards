package card_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
)

func TestDefaultDocumentShape(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	if document.CardID != card.DefaultCardID || document.Name != "Empty Card" {
		t.Fatalf("document = %#v", document)
	}
	if document.Root.ID != card.DefaultRootID || document.Root.Type != card.Type {
		t.Fatalf("root = %#v", document.Root)
	}
	if len(document.Root.Children) != 3 {
		t.Fatalf("children = %#v", document.Root.Children)
	}
	for index, expected := range []string{background.Type, border.Type, textarea.Type} {
		if document.Root.Children[index].Type != expected {
			t.Fatalf("child %d type = %q, want %q", index, document.Root.Children[index].Type, expected)
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
		`data-component-type="card"`,
		`background-color: #111827`,
		`border-radius: 24px`,
		`padding: 24px`,
		`data-component-type="textarea"`,
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
		ID:   card.DefaultShapeID,
		Type: shape.Type,
		Fragment: mustRaw(t, shape.Fragment{
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
		`data-component-type="shape"`,
		`<svg`,
		`points="50,8 92,88 8,88"`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("render missing %q:\n%s", marker, body)
		}
	}
}

func TestRenderDocumentLaterShellContributionsWin(t *testing.T) {
	t.Parallel()

	document := card.DefaultDocument()
	document.Root.Children[0].Fragment = mustRaw(t, background.Fragment{
		BackgroundColor: "#111827",
		CSS:             "box-shadow: 0 0 10px red;",
	})
	document.Root.Children[1].Fragment = mustRaw(t, border.Fragment{
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
