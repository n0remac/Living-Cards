package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatResponse struct {
	Message ChatMessage `json:"message"`
}

type ChatResult struct {
	Request         ChatRequest `json:"request"`
	RequestBodyJSON string      `json:"request_body_json"`
	RawResponse     string      `json:"raw_response"`
	ResponseContent string      `json:"response_content"`
}

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Embedding  []float64   `json:"embedding"`
	Embeddings [][]float64 `json:"embeddings"`
}

type EmbeddingResult struct {
	Request         EmbeddingRequest `json:"request"`
	RequestBodyJSON string           `json:"request_body_json"`
	RawResponse     string           `json:"raw_response"`
	Vector          []float64        `json:"vector"`
}

type ModelOption struct {
	Name string `json:"name"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return NewClientWithHTTPClient(baseURL, &http.Client{Timeout: timeout})
}

func NewClientWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    httpClient,
	}
}

func (c *Client) Chat(ctx context.Context, model string, messages []ChatMessage) (string, error) {
	result, err := c.ChatDetailed(ctx, ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return "", err
	}
	return result.ResponseContent, nil
}

func (c *Client) ChatDetailed(ctx context.Context, payload ChatRequest) (ChatResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return ChatResult{}, fmt.Errorf("marshal chat payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return ChatResult{}, fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	result := ChatResult{
		Request:         payload,
		RequestBodyJSON: string(body),
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return result, fmt.Errorf("ollama request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("read chat response: %w", err)
	}
	result.RawResponse = string(respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("ollama status %d: %s", resp.StatusCode, strings.TrimSpace(result.RawResponse))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return result, fmt.Errorf("decode chat response: %w", err)
	}

	result.ResponseContent = parsed.Message.Content
	return result, nil
}

func (c *Client) Embed(ctx context.Context, model, input string) ([]float64, error) {
	result, err := c.EmbedDetailed(ctx, EmbeddingRequest{
		Model: model,
		Input: input,
	})
	if err != nil {
		return nil, err
	}
	return result.Vector, nil
}

func (c *Client) EmbedDetailed(ctx context.Context, payload EmbeddingRequest) (EmbeddingResult, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return EmbeddingResult{}, fmt.Errorf("marshal embedding payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return EmbeddingResult{}, fmt.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	result := EmbeddingResult{
		Request:         payload,
		RequestBodyJSON: string(body),
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return result, fmt.Errorf("ollama embedding request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("read embedding response: %w", err)
	}
	result.RawResponse = string(respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, fmt.Errorf("ollama status %d: %s", resp.StatusCode, strings.TrimSpace(result.RawResponse))
	}

	var parsed embeddingResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return result, fmt.Errorf("decode embedding response: %w", err)
	}

	switch {
	case len(parsed.Embedding) > 0:
		result.Vector = append([]float64(nil), parsed.Embedding...)
	case len(parsed.Embeddings) > 0 && len(parsed.Embeddings[0]) > 0:
		result.Vector = append([]float64(nil), parsed.Embeddings[0]...)
	default:
		return result, fmt.Errorf("embedding response did not include a vector")
	}
	return result, nil
}

func ListModels(ctx context.Context, command []string, timeout time.Duration) ([]ModelOption, error) {
	cmdArgs := append([]string(nil), command...)
	if len(cmdArgs) == 0 || strings.TrimSpace(cmdArgs[0]) == "" {
		cmdArgs = []string{"ollama"}
	}
	cmdArgs = append(cmdArgs, "list")

	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, ok := callCtx.Deadline(); !ok && timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(callCtx, cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("ollama list failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("ollama list failed: %w", err)
	}

	models := ParseModelList(string(output))
	if len(models) == 0 {
		return nil, fmt.Errorf("ollama list returned no models")
	}
	return models, nil
}

func ParseModelList(raw string) []ModelOption {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	out := make([]ModelOption, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		if name == "" || strings.EqualFold(name, "name") {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, ModelOption{Name: name})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}
