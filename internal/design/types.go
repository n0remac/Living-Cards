package design

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/schema"
	"github.com/n0remac/Living-Card/internal/ollama"
)

type ChatClient interface {
	Chat(context.Context, string, []ollama.ChatMessage) (string, error)
}

type GenerateRequest struct {
	Instruction string
	OldCode     string
	ComponentID string
}

type Service struct {
	client     ChatClient
	model      string
	kind       string
	generation card.GenerationDefinition
}

func NewService(client ChatClient, model string, definition card.Definition) (*Service, error) {
	generation, ok := definition.Generation()
	if !ok || strings.TrimSpace(generation.SystemPrompt()) == "" {
		return nil, card.NewUnsupportedOperationError(definition.Kind(), "AI generation")
	}
	return &Service{client: client, model: strings.TrimSpace(model), kind: definition.Kind(), generation: generation}, nil
}

func (s *Service) Generate(ctx context.Context, request GenerateRequest) (json.RawMessage, error) {
	request.Instruction = strings.TrimSpace(request.Instruction)
	if request.Instruction == "" {
		return nil, ErrEmptyInstruction
	}
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("design service client is not initialized")
	}
	raw, err := s.client.Chat(ctx, s.model, []ollama.ChatMessage{{Role: "system", Content: s.generation.SystemPrompt()}, {Role: "user", Content: generateUserPrompt(request)}})
	if err != nil {
		return nil, err
	}
	generated, err := s.decode(raw)
	if err == nil {
		return generated, nil
	}
	if !errors.Is(err, ErrInvalidModelOutput) {
		return nil, err
	}
	repaired, repairErr := s.repair(ctx, request, raw, Issues(err))
	if repairErr == nil {
		return repaired, nil
	}
	if errors.Is(repairErr, ErrInvalidModelOutput) {
		return nil, repairErr
	}
	return nil, err
}

func (s *Service) repair(ctx context.Context, request GenerateRequest, invalidRaw string, issues []schema.Issue) (json.RawMessage, error) {
	raw, err := s.client.Chat(ctx, s.model, []ollama.ChatMessage{{Role: "system", Content: s.repairSystemPrompt()}, {Role: "user", Content: repairUserPrompt(request, invalidRaw, issues)}})
	if err != nil {
		return nil, err
	}
	return s.decode(raw)
}
func (s *Service) decode(raw string) (json.RawMessage, error) {
	canonical, issues := s.generation.CanonicalizeEnvelope(json.RawMessage(strings.TrimSpace(raw)))
	if len(issues) > 0 {
		return nil, NewInvalidModelOutputError(raw, issues, ErrInvalidModelOutput)
	}
	return canonical, nil
}

func generateUserPrompt(request GenerateRequest) string {
	parts := []string{"User instruction:\n" + strings.TrimSpace(request.Instruction)}
	if id := strings.TrimSpace(request.ComponentID); id != "" {
		parts = append(parts, "Component ID:\n"+id)
	}
	if old := strings.TrimSpace(request.OldCode); old != "" {
		parts = append(parts, "Existing config JSON to update:\n"+old)
	}
	return strings.Join(parts, "\n\n")
}
func (s *Service) repairSystemPrompt() string {
	return strings.Join([]string{"You repair invalid generated component configs for component_kind " + s.kind + ".", s.generation.SystemPrompt(), "Return JSON only. Do not include markdown, prose, HTML, or JavaScript.", "Preserve valid fields from the invalid response.", "Change only fields listed in Issues unless another field must change to keep the config coherent.", "Return the full corrected JSON object.", "Working example:\n" + s.generation.Example()}, "\n")
}
func repairUserPrompt(request GenerateRequest, invalidRaw string, issues []schema.Issue) string {
	issuesJSON, _ := json.MarshalIndent(issues, "", "  ")
	parts := []string{"Original user instruction:\n" + strings.TrimSpace(request.Instruction)}
	if id := strings.TrimSpace(request.ComponentID); id != "" {
		parts = append(parts, "Component ID:\n"+id)
	}
	if old := strings.TrimSpace(request.OldCode); old != "" {
		parts = append(parts, "Existing config JSON being updated:\n"+old)
	}
	parts = append(parts, "Issues:\n"+string(issuesJSON), "Invalid raw model response:\n"+strings.TrimSpace(invalidRaw), "Regenerate a valid replacement JSON object for the same component_kind.")
	return strings.Join(parts, "\n\n")
}

type outputError struct {
	raw    string
	issues []schema.Issue
	err    error
}

func NewInvalidModelOutputError(raw string, issues []schema.Issue, err error) error {
	if err == nil {
		err = ErrInvalidModelOutput
	}
	return &outputError{raw: raw, issues: append([]schema.Issue(nil), issues...), err: err}
}
func (e *outputError) Error() string {
	if e == nil || e.err == nil {
		return ErrInvalidModelOutput.Error()
	}
	return e.err.Error()
}
func (e *outputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}
func RawModelOutput(err error) (string, bool) {
	var outputErr *outputError
	if errors.As(err, &outputErr) && strings.TrimSpace(outputErr.raw) != "" {
		return outputErr.raw, true
	}
	return "", false
}
func Issues(err error) []schema.Issue {
	var outputErr *outputError
	if errors.As(err, &outputErr) {
		return append([]schema.Issue(nil), outputErr.issues...)
	}
	return nil
}

type designError string

func (e designError) Error() string { return string(e) }

const (
	ErrEmptyInstruction   designError = "instruction cannot be empty"
	ErrInvalidModelOutput designError = "invalid generated component config"
)
