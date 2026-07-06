package design

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/n0remac/Living-Card/internal/ollama"
)

type ChatClient interface {
	Chat(ctx context.Context, model string, messages []ollama.ChatMessage) (string, error)
}

type GeneratedConfig[T any] struct {
	ComponentKind string `json:"componentKind"`
	Description   string `json:"description"`
	Config        T      `json:"config"`
}

type Issue struct {
	Path    string   `json:"path"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Actual  any      `json:"actual,omitempty"`
	Allowed []string `json:"allowed,omitempty"`
}

type Spec[T any] struct {
	ComponentKind string
	SystemPrompt  string
	Example       string
	Normalize     func(*GeneratedConfig[T])
	Validate      func(GeneratedConfig[T]) []Issue
}

type Service[T any] struct {
	client ChatClient
	model  string
	spec   Spec[T]
}

type GenerateRequest struct {
	Instruction string
	OldCode     string
	ComponentID string
}

func NewService[T any](client ChatClient, model string, spec Spec[T]) *Service[T] {
	return &Service[T]{
		client: client,
		model:  strings.TrimSpace(model),
		spec:   spec,
	}
}

func (s *Service[T]) Generate(ctx context.Context, request GenerateRequest) (GeneratedConfig[T], error) {
	request.Instruction = strings.TrimSpace(request.Instruction)
	if request.Instruction == "" {
		return GeneratedConfig[T]{}, ErrEmptyInstruction
	}
	if s == nil || s.client == nil {
		return GeneratedConfig[T]{}, fmt.Errorf("design service client is not initialized")
	}
	raw, err := s.client.Chat(ctx, s.model, []ollama.ChatMessage{
		{Role: "system", Content: s.spec.SystemPrompt},
		{Role: "user", Content: generateUserPrompt(request)},
	})
	if err != nil {
		return GeneratedConfig[T]{}, err
	}
	generated, err := s.decodeNormalizeValidate(raw)
	if err == nil {
		return generated, nil
	}
	if !errors.Is(err, ErrInvalidModelOutput) {
		return GeneratedConfig[T]{}, err
	}
	repaired, repairErr := s.repair(ctx, request, raw, Issues(err))
	if repairErr == nil {
		return repaired, nil
	}
	if errors.Is(repairErr, ErrInvalidModelOutput) {
		return GeneratedConfig[T]{}, repairErr
	}
	return GeneratedConfig[T]{}, err
}

func (s *Service[T]) repair(ctx context.Context, request GenerateRequest, invalidRaw string, issues []Issue) (GeneratedConfig[T], error) {
	raw, err := s.client.Chat(ctx, s.model, []ollama.ChatMessage{
		{Role: "system", Content: repairSystemPrompt(s.spec)},
		{Role: "user", Content: repairUserPrompt(request, invalidRaw, issues)},
	})
	if err != nil {
		return GeneratedConfig[T]{}, err
	}
	return s.decodeNormalizeValidate(raw)
}

func (s *Service[T]) decodeNormalizeValidate(raw string) (GeneratedConfig[T], error) {
	return DecodeNormalizeValidateConfig(raw, s.spec)
}

func DecodeNormalizeValidateConfig[T any](raw string, spec Spec[T]) (GeneratedConfig[T], error) {
	generated, issues := DecodeGeneratedConfig[T](raw)
	if len(issues) > 0 {
		return GeneratedConfig[T]{}, NewInvalidModelOutputError(raw, issues, ErrInvalidModelOutput)
	}
	if spec.Normalize != nil {
		spec.Normalize(&generated)
	}
	issues = append(issues, validateEnvelope(generated, spec.ComponentKind)...)
	if spec.Validate != nil {
		issues = append(issues, spec.Validate(generated)...)
	}
	if len(issues) > 0 {
		return GeneratedConfig[T]{}, NewInvalidModelOutputError(raw, issues, ErrInvalidModelOutput)
	}
	return generated, nil
}

func DecodeGeneratedConfig[T any](raw string) (GeneratedConfig[T], []Issue) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return GeneratedConfig[T]{}, []Issue{{
			Path:    "$",
			Code:    "empty_response",
			Message: "model response is empty",
		}}
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var generated GeneratedConfig[T]
	if err := decoder.Decode(&generated); err != nil {
		return GeneratedConfig[T]{}, []Issue{{
			Path:    "$",
			Code:    "invalid_json",
			Message: "response must be one strict JSON object: " + err.Error(),
		}}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return GeneratedConfig[T]{}, []Issue{{
			Path:    "$",
			Code:    "multiple_json_values",
			Message: "response must contain one JSON object",
		}}
	}
	return generated, nil
}

func validateEnvelope[T any](generated GeneratedConfig[T], componentKind string) []Issue {
	var issues []Issue
	if strings.TrimSpace(generated.ComponentKind) != componentKind {
		issues = append(issues, Issue{
			Path:    "componentKind",
			Code:    "invalid_component_kind",
			Message: "componentKind must be " + componentKind,
			Actual:  generated.ComponentKind,
			Allowed: []string{componentKind},
		})
	}
	if strings.TrimSpace(generated.Description) == "" {
		issues = append(issues, Issue{
			Path:    "description",
			Code:    "required",
			Message: "description is required",
		})
	}
	return issues
}

func generateUserPrompt(request GenerateRequest) string {
	parts := []string{
		"User instruction:\n" + strings.TrimSpace(request.Instruction),
	}
	if componentID := strings.TrimSpace(request.ComponentID); componentID != "" {
		parts = append(parts, "Component ID:\n"+componentID)
	}
	if oldCode := strings.TrimSpace(request.OldCode); oldCode != "" {
		parts = append(parts, "Existing config JSON to update:\n"+oldCode)
	}
	return strings.Join(parts, "\n\n")
}

func repairSystemPrompt[T any](spec Spec[T]) string {
	return strings.Join([]string{
		"You repair invalid generated component configs for componentKind " + spec.ComponentKind + ".",
		spec.SystemPrompt,
		"Return JSON only. Do not include markdown, prose, HTML, or JavaScript.",
		"Preserve valid fields from the invalid response.",
		"Change only fields listed in Issues unless another field must change to keep the config coherent.",
		"Return the full corrected JSON object.",
		"Working example:\n" + spec.Example,
	}, "\n")
}

func repairUserPrompt(request GenerateRequest, invalidRaw string, issues []Issue) string {
	issuesJSON, _ := json.MarshalIndent(issues, "", "  ")
	parts := []string{
		"Original user instruction:\n" + strings.TrimSpace(request.Instruction),
	}
	if componentID := strings.TrimSpace(request.ComponentID); componentID != "" {
		parts = append(parts, "Component ID:\n"+componentID)
	}
	if oldCode := strings.TrimSpace(request.OldCode); oldCode != "" {
		parts = append(parts, "Existing config JSON being updated:\n"+oldCode)
	}
	parts = append(parts,
		"Issues:\n"+string(issuesJSON),
		"Invalid raw model response:\n"+strings.TrimSpace(invalidRaw),
		"Regenerate a valid replacement JSON object for the same componentKind.",
	)
	return strings.Join(parts, "\n\n")
}

type outputError struct {
	raw    string
	issues []Issue
	err    error
}

func NewInvalidModelOutputError(raw string, issues []Issue, err error) error {
	if err == nil {
		err = ErrInvalidModelOutput
	}
	return &outputError{raw: raw, issues: append([]Issue(nil), issues...), err: err}
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

func Issues(err error) []Issue {
	var outputErr *outputError
	if errors.As(err, &outputErr) {
		return append([]Issue(nil), outputErr.issues...)
	}
	return nil
}

type designError string

func (e designError) Error() string { return string(e) }

const (
	ErrEmptyInstruction   designError = "instruction cannot be empty"
	ErrInvalidModelOutput designError = "invalid generated component config"
)
