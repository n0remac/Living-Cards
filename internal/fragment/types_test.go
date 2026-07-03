package fragment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/ollama"
)

type testFragment struct {
	Foo string `json:"foo"`
}

func TestDecodeGeneratedInvalidJSONReturnsPathIssue(t *testing.T) {
	t.Parallel()

	_, issues := DecodeGenerated[testFragment](`{"target":"test"`)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v", issues)
	}
	if issues[0].Path != "$" || issues[0].Code != "invalid_json" {
		t.Fatalf("issue = %#v", issues[0])
	}
}

func TestServiceRepairsInvalidOutputOnce(t *testing.T) {
	t.Parallel()

	invalid := `{"target":"test","description":"Bad","fragment":{"foo":"bad"}}`
	client := &testChatClient{responses: []string{
		invalid,
		`{"target":"test","description":"Good","fragment":{"foo":"ok"}}`,
	}}
	service := NewService(client, "test-model", testSpec())

	response, err := service.Generate(context.Background(), GenerateRequest{
		Instruction: "make it useful",
		OldCode:     `{"target":"test","fragment":{"foo":"old"}}`,
		ComponentID: "component-1",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if response.Description != "Good" || response.Fragment.Foo != "ok" {
		t.Fatalf("response = %#v", response)
	}
	if len(client.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(client.calls))
	}
	repairPrompt := joinedFragmentMessages(client.calls[1])
	for _, marker := range []string{
		"make it useful",
		`{"target":"test","fragment":{"foo":"old"}}`,
		"component-1",
		invalid,
		`"path": "fragment.foo"`,
		`"code": "invalid_value"`,
		`"target":"test"`,
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
		`{"target":"test","description":"Bad","fragment":{"foo":"bad"}}`,
		`{"target":"test","description":"Still bad","fragment":{"foo":"bad"}}`,
		`{"target":"test","description":"Would be ignored","fragment":{"foo":"ok"}}`,
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
	if len(issues) != 1 || issues[0].Path != "fragment.foo" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestServiceDoesNotRepairEmptyInstruction(t *testing.T) {
	t.Parallel()

	client := &testChatClient{responses: []string{
		`{"target":"test","description":"Good","fragment":{"foo":"ok"}}`,
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

func testSpec() Spec[testFragment] {
	return Spec[testFragment]{
		Target:       "test",
		SystemPrompt: "Generate a test fragment.",
		Example:      `{"target":"test","description":"Example","fragment":{"foo":"ok"}}`,
		Normalize: func(generated *Generated[testFragment]) {
			generated.Target = strings.TrimSpace(generated.Target)
			generated.Description = strings.TrimSpace(generated.Description)
			generated.Fragment.Foo = strings.TrimSpace(generated.Fragment.Foo)
		},
		Validate: func(generated Generated[testFragment]) []Issue {
			if generated.Fragment.Foo != "ok" {
				return []Issue{{
					Path:    "fragment.foo",
					Code:    "invalid_value",
					Message: "foo must be ok",
					Actual:  generated.Fragment.Foo,
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

func joinedFragmentMessages(messages []ollama.ChatMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		parts = append(parts, message.Content)
	}
	return strings.Join(parts, "\n")
}
