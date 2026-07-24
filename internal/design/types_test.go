package design

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
	"github.com/n0remac/Living-Card/internal/ollama"
)

type testConfig struct {
	Foo string `json:"foo"`
}

func TestServiceUsesErasedDefinitionAndRepairsOnce(t *testing.T) {
	invalid := `{"component_kind":"test","description":"Bad","config":{"foo":"bad"}}`
	client := &testChatClient{responses: []string{
		invalid,
		`{"component_kind":"test","description":" Good ","config":{"foo":" ok "}}`,
	}}
	service, err := NewService(client, "test-model", testDefinition(true))
	if err != nil {
		t.Fatal(err)
	}

	response, err := service.Generate(context.Background(), GenerateRequest{Instruction: "make it useful", OldCode: `{"component_kind":"test","config":{"foo":"old"}}`, ComponentID: "component-1"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	var envelope schema.GeneratedConfigEnvelope
	if err := json.Unmarshal(response, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.ComponentKind != "test" || envelope.Description != "Good" || string(envelope.Config) != `{"foo":"ok"}` {
		t.Fatalf("response = %s", response)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(client.calls))
	}
	repairPrompt := joinedMessages(client.calls[1])
	for _, marker := range []string{"make it useful", "component-1", invalid, `"path": "config.foo"`, `"code": "invalid_value"`, "Preserve valid fields"} {
		if !strings.Contains(repairPrompt, marker) {
			t.Fatalf("repair prompt missing %q:\n%s", marker, repairPrompt)
		}
	}
}

func TestServiceReportsFailedRepairOutput(t *testing.T) {
	client := &testChatClient{responses: []string{
		`{"component_kind":"test","description":"Bad","config":{"foo":"bad"}}`,
		`{"component_kind":"test","description":"Still bad","config":{"foo":"bad"}}`,
	}}
	service, err := NewService(client, "test-model", testDefinition(true))
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Generate(context.Background(), GenerateRequest{Instruction: "make it useful"})
	if !errors.Is(err, ErrInvalidModelOutput) || len(client.calls) != 2 {
		t.Fatalf("Generate() error = %v, calls = %d", err, len(client.calls))
	}
	raw, ok := RawModelOutput(err)
	if !ok || !strings.Contains(raw, "Still bad") {
		t.Fatalf("RawModelOutput() = %q, %v", raw, ok)
	}
	if issues := Issues(err); len(issues) != 1 || issues[0].Path != "config.foo" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestServiceRejectsEmptyInstructionAndUnsupportedDefinition(t *testing.T) {
	client := &testChatClient{responses: []string{`{"component_kind":"test","description":"Good","config":{"foo":"ok"}}`}}
	service, err := NewService(client, "test-model", testDefinition(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Generate(context.Background(), GenerateRequest{Instruction: " "}); !errors.Is(err, ErrEmptyInstruction) {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(client.calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(client.calls))
	}
	if _, err := NewService(client, "test-model", testDefinition(false)); !errors.Is(err, card.ErrUnsupportedOperation) {
		t.Fatalf("NewService() error = %v, want ErrUnsupportedOperation", err)
	}
}

func testDefinition(withAI bool) card.Definition {
	var generation *card.TypedGenerationDefinition[testConfig]
	if withAI {
		generation = &card.TypedGenerationDefinition[testConfig]{SystemPrompt: "Generate a test design.", Example: `{"component_kind":"test","description":"Example","config":{"foo":"ok"}}`}
	}
	return card.MustDefine(card.TypedDefinition[testConfig]{
		Kind: "test", Label: "Test", Structure: card.StructureLeaf,
		Default:   func() testConfig { return testConfig{Foo: "ok"} },
		Normalize: func(config testConfig) testConfig { config.Foo = strings.TrimSpace(config.Foo); return config },
		Validate: func(config testConfig) []schema.Issue {
			if config.Foo != "ok" {
				return []schema.Issue{{Path: "config.foo", Code: "invalid_value", Message: "foo must be ok", Actual: config.Foo, Allowed: []string{"ok"}}}
			}
			return nil
		},
		Render: func(card.Node, testConfig, card.RenderContext) (card.Contribution, error) {
			return card.Contribution{}, nil
		},
		Generation: generation,
	})
}

type testChatClient struct {
	responses []string
	calls     [][]ollama.ChatMessage
}

func (f *testChatClient) Chat(_ context.Context, _ string, messages []ollama.ChatMessage) (string, error) {
	f.calls = append(f.calls, append([]ollama.ChatMessage(nil), messages...))
	if len(f.responses) == 0 {
		return "", nil
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func joinedMessages(messages []ollama.ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		parts = append(parts, message.Content)
	}
	return strings.Join(parts, "\n")
}
