package cardimages

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGenerate(t *testing.T) {
	image := []byte("generated-webp")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/images/generations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		var request struct {
			Model        string `json:"model"`
			Prompt       string `json:"prompt"`
			Size         string `json:"size"`
			Quality      string `json:"quality"`
			OutputFormat string `json:"output_format"`
			N            int    `json:"n"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "gpt-image-2" || request.Prompt != "test prompt" ||
			request.Size != "960x1344" || request.Quality != "medium" ||
			request.OutputFormat != "webp" || request.N != 1 {
			t.Errorf("unexpected payload: %#v", request)
		}
		w.Header().Set("X-Request-ID", "request-123")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(image)}},
		})
	}))
	defer server.Close()

	client := Client{APIKey: "test-key", Endpoint: server.URL + "/v1/images/generations"}
	result, err := client.Generate(context.Background(), GenerateOptions{
		Model: "gpt-image-2", Prompt: "test prompt", Size: "960x1344", Quality: "medium", Format: "webp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Image) != string(image) || result.RequestID != "request-123" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestClientGenerateReportsAPIErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad image request","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	client := Client{APIKey: "test-key", Endpoint: server.URL}
	_, err := client.Generate(context.Background(), GenerateOptions{
		Model: "gpt-image-2", Prompt: "test", Size: "960x1344", Quality: "low", Format: "webp",
	})
	if err == nil || !strings.Contains(err.Error(), "bad image request") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientGenerateRejectsInvalidBase64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"%%%%"}]}`))
	}))
	defer server.Close()

	client := Client{APIKey: "test-key", Endpoint: server.URL}
	_, err := client.Generate(context.Background(), GenerateOptions{
		Model: "gpt-image-2", Prompt: "test", Size: "960x1344", Quality: "low", Format: "webp",
	})
	if err == nil || !strings.Contains(err.Error(), "decode generated image") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGenerateOptions(t *testing.T) {
	valid := GenerateOptions{
		Model: "gpt-image-2", Prompt: "test", Size: "960x1344", Quality: "medium", Format: "webp",
	}
	if err := validateGenerateOptions(valid); err != nil {
		t.Fatal(err)
	}
	tests := []GenerateOptions{
		{Model: "", Prompt: "test", Size: "960x1344", Quality: "medium", Format: "webp"},
		{Model: "gpt-image-2", Prompt: "", Size: "960x1344", Quality: "medium", Format: "webp"},
		{Model: "gpt-image-2", Prompt: "test", Size: "961x1344", Quality: "medium", Format: "webp"},
		{Model: "gpt-image-2", Prompt: "test", Size: "960x1344", Quality: "ultra", Format: "webp"},
		{Model: "gpt-image-2", Prompt: "test", Size: "960x1344", Quality: "medium", Format: "gif"},
		{Model: "gpt-image-1.5", Prompt: "test", Size: "960x1344", Quality: "medium", Format: "webp"},
	}
	for _, options := range tests {
		if err := validateGenerateOptions(options); err == nil {
			t.Errorf("validateGenerateOptions(%#v) succeeded", options)
		}
	}
}
