package component

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/n0remac/Living-Card/internal/ollama"
)

type PatchProposalRequest struct {
	Instruction string `json:"instruction"`
}

type PatchProposalResponse struct {
	ComponentType string   `json:"component_type"`
	ContextFiles  []string `json:"context_files"`
	Proposal      string   `json:"proposal"`
}

type PatchProposalService struct {
	registry *Registry
	client   PatchClient
	model    string
	root     string
}

func NewPatchProposalService(registry *Registry, client PatchClient, model, root string) *PatchProposalService {
	return &PatchProposalService{
		registry: registry,
		client:   client,
		model:    strings.TrimSpace(model),
		root:     strings.TrimSpace(root),
	}
}

func (s *PatchProposalService) Propose(ctx context.Context, componentType, instruction string) (PatchProposalResponse, error) {
	componentType = strings.TrimSpace(componentType)
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return PatchProposalResponse{}, ErrEmptyInstruction
	}
	if s == nil || s.registry == nil {
		return PatchProposalResponse{}, fmt.Errorf("patch proposal service is not initialized")
	}
	if s.client == nil {
		return PatchProposalResponse{}, fmt.Errorf("patch proposal client is not initialized")
	}
	contextFiles, ok := s.registry.ContextFiles(componentType)
	if !ok {
		return PatchProposalResponse{}, ErrComponentTypeNotFound
	}
	contextBody, err := s.readContextFiles(contextFiles)
	if err != nil {
		return PatchProposalResponse{}, err
	}
	proposal, err := s.client.Chat(ctx, s.model, []ollama.ChatMessage{
		{Role: "system", Content: patchSystemPrompt()},
		{Role: "user", Content: patchUserPrompt(componentType, instruction, contextBody)},
	})
	if err != nil {
		return PatchProposalResponse{}, err
	}
	proposal = strings.TrimSpace(proposal)
	if proposal == "" {
		return PatchProposalResponse{}, fmt.Errorf("patch proposal was empty")
	}
	return PatchProposalResponse{
		ComponentType: componentType,
		ContextFiles:  contextFiles,
		Proposal:      proposal,
	}, nil
}

func (s *PatchProposalService) readContextFiles(contextFiles []string) (string, error) {
	root := s.root
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
	}
	var builder strings.Builder
	for _, name := range contextFiles {
		path, err := safeContextPath(root, name)
		if err != nil {
			return "", err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read context file %q: %w", name, err)
		}
		builder.WriteString("File: ")
		builder.WriteString(name)
		builder.WriteString("\n```")
		builder.WriteString(filepath.Ext(name))
		builder.WriteString("\n")
		builder.Write(contents)
		builder.WriteString("\n```\n\n")
	}
	return builder.String(), nil
}

func safeContextPath(root, name string) (string, error) {
	cleanName := filepath.Clean(strings.TrimSpace(name))
	if cleanName == "." || filepath.IsAbs(cleanName) || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || cleanName == ".." {
		return "", fmt.Errorf("invalid context file path %q", name)
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	path := filepath.Join(root, cleanName)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("resolve context file path %q: %w", name, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("context file path escapes project root: %q", name)
	}
	return path, nil
}

func patchSystemPrompt() string {
	return strings.Join([]string{
		"You generate reviewable code patch proposals for a single web component.",
		"Return only a unified diff or clearly fenced patch text.",
		"Do not include prose outside the patch.",
		"Only edit files explicitly included in the component context.",
	}, "\n")
}

func patchUserPrompt(componentType, instruction, contextBody string) string {
	return "Component type: " + componentType +
		"\n\nUser instruction:\n" + instruction +
		"\n\nComponent context files:\n" + contextBody +
		"\nReturn a patch proposal only."
}

type proposalError string

func (e proposalError) Error() string { return string(e) }

const (
	ErrEmptyInstruction      proposalError = "instruction cannot be empty"
	ErrComponentTypeNotFound proposalError = "component type not found"
)
