package ollama

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestClientEmbedDetailed(t *testing.T) {
	t.Parallel()

	client := NewClientWithHTTPClient("http://ollama.test", &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/api/embed" {
				t.Fatalf("req.URL.Path = %q, want /api/embed", req.URL.Path)
			}
			return jsonResponse(http.StatusOK, `{"embeddings":[[0.1,0.2,0.3]]}`), nil
		}),
	})
	result, err := client.EmbedDetailed(context.Background(), EmbeddingRequest{
		Model: "nomic-embed-text",
		Input: "hello world",
	})
	if err != nil {
		t.Fatalf("EmbedDetailed() error = %v", err)
	}
	if len(result.Vector) != 3 {
		t.Fatalf("len(result.Vector) = %d, want 3", len(result.Vector))
	}
	if result.Request.Model != "nomic-embed-text" {
		t.Fatalf("result.Request.Model = %q", result.Request.Model)
	}
	if result.RawResponse == "" {
		t.Fatalf("result.RawResponse is empty")
	}
}

func TestClientEmbedDetailedReturnsStatusError(t *testing.T) {
	t.Parallel()

	client := NewClientWithHTTPClient("http://ollama.test", &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusBadRequest, "bad request"), nil
		}),
	})
	if _, err := client.EmbedDetailed(context.Background(), EmbeddingRequest{Model: "embed", Input: "hello"}); err == nil {
		t.Fatal("EmbedDetailed() error = nil, want status error")
	}
}

func TestClientEmbedDetailedRejectsMalformedResponse(t *testing.T) {
	t.Parallel()

	client := NewClientWithHTTPClient("http://ollama.test", &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"embeddings":"oops"}`), nil
		}),
	})
	if _, err := client.EmbedDetailed(context.Background(), EmbeddingRequest{Model: "embed", Input: "hello"}); err == nil {
		t.Fatal("EmbedDetailed() error = nil, want decode error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
