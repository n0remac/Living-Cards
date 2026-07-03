package component

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	godom "github.com/n0remac/GoDom/html"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/ollama"
)

func TestRegistryContextFilesAreCloned(t *testing.T) {
	t.Parallel()

	registry := MustNewRegistry(Definition{
		Type:         "example",
		ContextFiles: []string{"view.go", "client.ts"},
	})
	files, ok := registry.ContextFiles("example")
	if !ok {
		t.Fatal("ContextFiles() ok = false, want true")
	}
	files[0] = "changed"

	files, ok = registry.ContextFiles("example")
	if !ok {
		t.Fatal("ContextFiles() ok = false, want true")
	}
	if files[0] != "view.go" {
		t.Fatalf("ContextFiles()[0] = %q, want view.go", files[0])
	}
}

func TestRegistryDispatchesComponentAction(t *testing.T) {
	t.Parallel()

	handled := false
	registry := MustNewRegistry(Definition{
		Type: "example",
		Render: func(cards.ComponentInstance) *godom.Node {
			return nil
		},
		HandleAction: func(w http.ResponseWriter, _ *http.Request, _ Dependencies, _ cards.Card, instance cards.ComponentInstance, action string) bool {
			if instance.ID != "main" || action != "send" {
				return false
			}
			handled = true
			w.WriteHeader(http.StatusAccepted)
			return true
		},
	})
	card := cards.Card{
		CardID: "card",
		Components: []cards.ComponentInstance{{
			ID:   "main",
			Type: "example",
		}},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/cards/card/components/main/actions/send", nil)
	recorder := httptest.NewRecorder()

	if !registry.HandleCardResource(recorder, request, Dependencies{}, card, []string{"card", "components", "main", "actions", "send"}) {
		t.Fatal("HandleCardResource() = false, want true")
	}
	if !handled {
		t.Fatal("handler was not called")
	}
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", recorder.Code)
	}
}

func TestRegistryRendersComponentInstancesInOrderWithMetadata(t *testing.T) {
	t.Parallel()

	registry := MustNewRegistry(
		Definition{
			Type:              "first",
			ClientInitializer: "initFirst",
			Render: func(cards.ComponentInstance) *godom.Node {
				return godom.Div(godom.Id("first-view"))
			},
		},
		Definition{
			Type:              "second",
			ClientInitializer: "initSecond",
			Render: func(cards.ComponentInstance) *godom.Node {
				return godom.Div(godom.Id("second-view"))
			},
		},
	)
	node, err := registry.RenderCard(cards.Card{
		CardID: "card",
		Components: []cards.ComponentInstance{
			{ID: "a", Type: "first"},
			{ID: "b", Type: "second"},
		},
	})
	if err != nil {
		t.Fatalf("RenderCard() error = %v", err)
	}
	html := node.Render()
	firstIdx := strings.Index(html, `data-component-id="a"`)
	secondIdx := strings.Index(html, `data-component-id="b"`)
	if firstIdx < 0 || secondIdx < 0 || firstIdx > secondIdx {
		t.Fatalf("components were not rendered in order: %s", html)
	}
	for _, marker := range []string{
		`data-component-type="first"`,
		`data-client-initializer="initFirst"`,
		`id="first-view"`,
		`id="second-view"`,
	} {
		if !strings.Contains(html, marker) {
			t.Fatalf("rendered html missing %s: %s", marker, html)
		}
	}
}

func TestPatchProposalServiceReadsOnlyContextFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "component/view.go", "package component\n")
	writeTestFile(t, root, "component/client.ts", "export const client = true;\n")
	writeTestFile(t, root, "component/secret.txt", "do not include\n")

	client := &fakePatchClient{response: "diff --git a/component/view.go b/component/view.go"}
	service := NewPatchProposalService(MustNewRegistry(Definition{
		Type:         "example",
		ContextFiles: []string{"component/view.go", "component/client.ts"},
	}), client, "test-model", root)

	response, err := service.Propose(context.Background(), "example", "change the title")
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}
	if response.ComponentType != "example" || response.Proposal == "" {
		t.Fatalf("response = %#v", response)
	}
	if len(response.ContextFiles) != 2 {
		t.Fatalf("ContextFiles = %#v", response.ContextFiles)
	}
	prompt := client.lastPrompt()
	if !strings.Contains(prompt, "component/view.go") || !strings.Contains(prompt, "component/client.ts") {
		t.Fatalf("prompt missing context files: %s", prompt)
	}
	if strings.Contains(prompt, "secret.txt") || strings.Contains(prompt, "do not include") {
		t.Fatalf("prompt included unlisted file: %s", prompt)
	}
}

func TestPatchProposalServiceRejectsUnknownComponentType(t *testing.T) {
	t.Parallel()

	service := NewPatchProposalService(MustNewRegistry(), &fakePatchClient{}, "test-model", t.TempDir())
	if _, err := service.Propose(context.Background(), "missing", "change it"); err != ErrComponentTypeNotFound {
		t.Fatalf("Propose() error = %v, want ErrComponentTypeNotFound", err)
	}
}

func TestPatchProposalServiceRejectsEmptyInstruction(t *testing.T) {
	t.Parallel()

	service := NewPatchProposalService(MustNewRegistry(Definition{Type: "example"}), &fakePatchClient{}, "test-model", t.TempDir())
	if _, err := service.Propose(context.Background(), "example", " "); err != ErrEmptyInstruction {
		t.Fatalf("Propose() error = %v, want ErrEmptyInstruction", err)
	}
}

func writeTestFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

type fakePatchClient struct {
	response string
	messages []ollama.ChatMessage
}

func (f *fakePatchClient) Chat(_ context.Context, _ string, messages []ollama.ChatMessage) (string, error) {
	f.messages = append([]ollama.ChatMessage(nil), messages...)
	if f.response == "" {
		return "diff --git a/file b/file", nil
	}
	return f.response, nil
}

func (f *fakePatchClient) lastPrompt() string {
	if len(f.messages) == 0 {
		return ""
	}
	return f.messages[len(f.messages)-1].Content
}
