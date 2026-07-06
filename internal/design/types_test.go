package design

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/ollama"
)

type testConfig struct {
	Foo string `json:"foo"`
}

func TestDecodeGeneratedInvalidJSONReturnsPathIssue(t *testing.T) {
	t.Parallel()

	_, issues := DecodeGeneratedConfig[testConfig](`{"componentKind":"test"`)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v", issues)
	}
	if issues[0].Path != "$" || issues[0].Code != "invalid_json" {
		t.Fatalf("issue = %#v", issues[0])
	}
}

func TestServiceRepairsInvalidOutputOnce(t *testing.T) {
	t.Parallel()

	invalid := `{"componentKind":"test","description":"Bad","config":{"foo":"bad"}}`
	client := &testChatClient{responses: []string{
		invalid,
		`{"componentKind":"test","description":"Good","config":{"foo":"ok"}}`,
	}}
	service := NewService(client, "test-model", testSpec())

	response, err := service.Generate(context.Background(), GenerateRequest{
		Instruction: "make it useful",
		OldCode:     `{"componentKind":"test","config":{"foo":"old"}}`,
		ComponentID: "component-1",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if response.Description != "Good" || response.Config.Foo != "ok" {
		t.Fatalf("response = %#v", response)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(client.calls))
	}
	repairPrompt := joinedConfigMessages(client.calls[1])
	for _, marker := range []string{
		"make it useful",
		`{"componentKind":"test","config":{"foo":"old"}}`,
		"component-1",
		invalid,
		`"path": "config.foo"`,
		`"code": "invalid_value"`,
		`"componentKind":"test"`,
		`"foo":"ok"`,
		"Preserve valid fields",
	} {
		if !strings.Contains(repairPrompt, marker) {
			t.Fatalf("repair prompt missing %q:\n%s", marker, repairPrompt)
		}
	}
}

func TestServiceFailedRepairReturnsRepairRawOutputAndIssues(t *testing.T) {
	t.Parallel()

	client := &testChatClient{responses: []string{
		`{"componentKind":"test","description":"Bad","config":{"foo":"bad"}}`,
		`{"componentKind":"test","description":"Still bad","config":{"foo":"bad"}}`,
		`{"componentKind":"test","description":"Would be ignored","config":{"foo":"ok"}}`,
	}}
	service := NewService(client, "test-model", testSpec())

	_, err := service.Generate(context.Background(), GenerateRequest{Instruction: "make it useful"})
	if !errors.Is(err, ErrInvalidModelOutput) {
		t.Fatalf("Generate() error = %v, want ErrInvalidModelOutput", err)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %d, want exactly one repair call", len(client.calls))
	}
	raw, ok := RawModelOutput(err)
	if !ok || !strings.Contains(raw, "Still bad") {
		t.Fatalf("RawModelOutput() = %q, %v", raw, ok)
	}
	issues := Issues(err)
	if len(issues) != 1 || issues[0].Path != "config.foo" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestServiceDoesNotRepairEmptyInstruction(t *testing.T) {
	t.Parallel()

	client := &testChatClient{responses: []string{
		`{"componentKind":"test","description":"Good","config":{"foo":"ok"}}`,
	}}
	service := NewService(client, "test-model", testSpec())

	_, err := service.Generate(context.Background(), GenerateRequest{Instruction: " "})
	if !errors.Is(err, ErrEmptyInstruction) {
		t.Fatalf("Generate() error = %v, want ErrEmptyInstruction", err)
	}
	if len(client.calls) != 0 {
		t.Fatalf("calls = %d, want 0", len(client.calls))
	}
}

func testSpec() Spec[testConfig] {
	return Spec[testConfig]{
		ComponentKind: "test",
		SystemPrompt:  "Generate a test design.",
		Example:       `{"componentKind":"test","description":"Example","config":{"foo":"ok"}}`,
		Normalize: func(generated *GeneratedConfig[testConfig]) {
			generated.ComponentKind = strings.TrimSpace(generated.ComponentKind)
			generated.Description = strings.TrimSpace(generated.Description)
			generated.Config.Foo = strings.TrimSpace(generated.Config.Foo)
		},
		Validate: func(generated GeneratedConfig[testConfig]) []Issue {
			if generated.Config.Foo != "ok" {
				return []Issue{{
					Path:    "config.foo",
					Code:    "invalid_value",
					Message: "foo must be ok",
					Actual:  generated.Config.Foo,
					Allowed: []string{"ok"},
				}}
			}
			return nil
		},
	}
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

func joinedConfigMessages(messages []ollama.ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		parts = append(parts, message.Content)
	}
	return strings.Join(parts, "\n")
}
