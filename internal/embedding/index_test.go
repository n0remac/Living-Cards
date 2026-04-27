package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/n0remac/Living-Card/internal/ollama"
)

func TestIndexSearchAndFilter(t *testing.T) {
	t.Parallel()

	qdrant := newFakeQdrantState()
	index, err := New(
		ollama.NewClientWithHTTPClient("http://ollama.test", &http.Client{
			Timeout:   2 * time.Second,
			Transport: newFakeEmbeddingOllamaTransport(),
		}),
		Config{
			QdrantBaseURL:    "http://qdrant.test",
			CollectionPrefix: "embedding-v1",
			RequestTimeout:   2 * time.Second,
			HTTPClient: &http.Client{
				Timeout:   2 * time.Second,
				Transport: qdrant,
			},
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	if _, err := index.UpsertDocuments(ctx, "nomic-embed-text", []Document{
		{DocumentID: "set-a:0", Text: "alpha", Payload: map[string]any{"message_set_id": "set-a", "message_index": 0}},
		{DocumentID: "set-a:1", Text: "beta", Payload: map[string]any{"message_set_id": "set-a", "message_index": 1}},
		{DocumentID: "set-b:0", Text: "gamma", Payload: map[string]any{"message_set_id": "set-b", "message_index": 0}},
	}); err != nil {
		t.Fatalf("UpsertDocuments() error = %v", err)
	}

	result, err := index.Search(ctx, "nomic-embed-text", "find alpha", 2, map[string]string{"message_set_id": "set-a"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.CollectionName != "embedding-v1-nomic-embed-text" {
		t.Fatalf("CollectionName = %q", result.CollectionName)
	}
	if len(result.Results) != 2 {
		t.Fatalf("len(result.Results) = %d, want 2", len(result.Results))
	}
	if result.Results[0].Text != "alpha" {
		t.Fatalf("first result = %#v, want alpha first", result.Results[0])
	}
	for _, item := range result.Results {
		if item.Text == "gamma" {
			t.Fatalf("Search() returned message from a different message set: %#v", item)
		}
	}
}

func TestIndexSearchReturnsEmptyWhenCollectionMissing(t *testing.T) {
	t.Parallel()

	index, err := New(
		ollama.NewClientWithHTTPClient("http://ollama.test", &http.Client{
			Timeout:   2 * time.Second,
			Transport: newFakeEmbeddingOllamaTransport(),
		}),
		Config{
			QdrantBaseURL:    "http://qdrant.test",
			CollectionPrefix: "embedding-v1",
			RequestTimeout:   2 * time.Second,
			HTTPClient: &http.Client{
				Timeout:   2 * time.Second,
				Transport: newFakeQdrantState(),
			},
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := index.Search(context.Background(), "nomic-embed-text", "find alpha", 3, nil)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.CollectionName != "embedding-v1-nomic-embed-text" {
		t.Fatalf("CollectionName = %q", result.CollectionName)
	}
	if len(result.Results) != 0 {
		t.Fatalf("len(result.Results) = %d, want 0 for missing collection", len(result.Results))
	}
}

func TestIndexMessagesUpsertsDeterministically(t *testing.T) {
	t.Parallel()

	qdrant := newFakeQdrantState()
	index, err := New(
		ollama.NewClientWithHTTPClient("http://ollama.test", &http.Client{
			Timeout:   2 * time.Second,
			Transport: newFakeEmbeddingOllamaTransport(),
		}),
		Config{
			QdrantBaseURL:    "http://qdrant.test",
			CollectionPrefix: "embedding-v1",
			RequestTimeout:   2 * time.Second,
			HTTPClient: &http.Client{
				Timeout:   2 * time.Second,
				Transport: qdrant,
			},
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	if _, err := index.UpsertDocuments(ctx, "nomic-embed-text", []Document{
		{DocumentID: "set-a:0", Text: "alpha", Payload: map[string]any{"message_set_id": "set-a", "message_index": 0}},
		{DocumentID: "set-a:1", Text: "beta", Payload: map[string]any{"message_set_id": "set-a", "message_index": 1}},
	}); err != nil {
		t.Fatalf("UpsertDocuments(first) error = %v", err)
	}
	if _, err := index.UpsertDocuments(ctx, "nomic-embed-text", []Document{
		{DocumentID: "set-a:0", Text: "alpha revised", Payload: map[string]any{"message_set_id": "set-a", "message_index": 0}},
		{DocumentID: "set-a:1", Text: "beta", Payload: map[string]any{"message_set_id": "set-a", "message_index": 1}},
	}); err != nil {
		t.Fatalf("UpsertDocuments(second) error = %v", err)
	}

	collection := qdrant.collection("embedding-v1-nomic-embed-text")
	if len(collection.points) != 2 {
		t.Fatalf("len(points) = %d, want 2 after deterministic upsert", len(collection.points))
	}

	result, err := index.Search(ctx, "nomic-embed-text", "find alpha", 2, map[string]string{"message_set_id": "set-a"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Results[0].Text != "alpha revised" {
		t.Fatalf("first result = %#v, want revised payload", result.Results[0])
	}
}

func TestIndexMessagesRejectsIncompatibleCollection(t *testing.T) {
	t.Parallel()

	qdrant := newFakeQdrantState()
	qdrant.collections["embedding-v1-nomic-embed-text"] = &fakeCollection{
		size:   2,
		points: make(map[uint64]fakePoint),
	}

	index, err := New(
		ollama.NewClientWithHTTPClient("http://ollama.test", &http.Client{
			Timeout:   2 * time.Second,
			Transport: newFakeEmbeddingOllamaTransport(),
		}),
		Config{
			QdrantBaseURL:    "http://qdrant.test",
			CollectionPrefix: "embedding-v1",
			RequestTimeout:   2 * time.Second,
			HTTPClient: &http.Client{
				Timeout:   2 * time.Second,
				Transport: qdrant,
			},
		},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if _, err := index.UpsertDocuments(context.Background(), "nomic-embed-text", []Document{
		{DocumentID: "set-a:0", Text: "alpha", Payload: map[string]any{"message_set_id": "set-a", "message_index": 0}},
	}); err == nil || !strings.Contains(err.Error(), "vector size") {
		t.Fatalf("UpsertDocuments() error = %v, want vector size mismatch", err)
	}
}

func newFakeEmbeddingOllamaTransport() http.RoundTripper {
	vectors := map[string][]float64{
		"alpha":         {1, 0, 0},
		"alpha revised": {0.95, 0.05, 0},
		"beta":          {0.7, 0.3, 0},
		"gamma":         {0, 1, 0},
		"find alpha":    {1, 0, 0},
		"find gamma":    {0, 1, 0},
	}
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/api/embed" {
			return jsonHTTPResponse(http.StatusNotFound, "not found"), nil
		}
		var payload struct {
			Input string `json:"input"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return jsonHTTPResponse(http.StatusBadRequest, err.Error()), nil
		}
		vector, ok := vectors[payload.Input]
		if !ok {
			return jsonHTTPResponse(http.StatusBadRequest, "unknown input"), nil
		}
		raw, _ := json.Marshal(map[string]any{"embeddings": [][]float64{vector}})
		return jsonHTTPResponse(http.StatusOK, string(raw)), nil
	})
}

type fakeQdrantState struct {
	mu          sync.Mutex
	collections map[string]*fakeCollection
}

type fakeCollection struct {
	size   int
	points map[uint64]fakePoint
}

type fakePoint struct {
	Vector  []float64
	Payload map[string]any
}

func newFakeQdrantState() *fakeQdrantState {
	return &fakeQdrantState{collections: make(map[string]*fakeCollection)}
}

func (s *fakeQdrantState) collection(name string) *fakeCollection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.collections[name]
}

func (s *fakeQdrantState) RoundTrip(req *http.Request) (*http.Response, error) {
	path := strings.TrimPrefix(req.URL.Path, "/collections/")
	switch {
	case strings.HasSuffix(path, "/points/search"):
		return s.search(req, strings.TrimSuffix(path, "/points/search"))
	case strings.HasSuffix(path, "/points"):
		return s.upsert(req, strings.TrimSuffix(path, "/points"))
	default:
		switch req.Method {
		case http.MethodGet:
			return s.getCollection(strings.Trim(path, "/"))
		case http.MethodPut:
			return s.createCollection(req, strings.Trim(path, "/"))
		default:
			return jsonHTTPResponse(http.StatusMethodNotAllowed, "method not allowed"), nil
		}
	}
}

func (s *fakeQdrantState) getCollection(name string) (*http.Response, error) {
	s.mu.Lock()
	collection, ok := s.collections[name]
	s.mu.Unlock()
	if !ok {
		return jsonHTTPResponse(http.StatusNotFound, "not found"), nil
	}
	raw, _ := json.Marshal(map[string]any{
		"result": map[string]any{
			"config": map[string]any{
				"params": map[string]any{
					"vectors": map[string]any{"size": collection.size},
				},
			},
		},
	})
	return jsonHTTPResponse(http.StatusOK, string(raw)), nil
}

func (s *fakeQdrantState) createCollection(req *http.Request, name string) (*http.Response, error) {
	var payload struct {
		Vectors struct {
			Size int `json:"size"`
		} `json:"vectors"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		return jsonHTTPResponse(http.StatusBadRequest, err.Error()), nil
	}
	s.mu.Lock()
	s.collections[name] = &fakeCollection{size: payload.Vectors.Size, points: make(map[uint64]fakePoint)}
	s.mu.Unlock()
	return jsonHTTPResponse(http.StatusOK, `{"status":"ok"}`), nil
}

func (s *fakeQdrantState) upsert(req *http.Request, name string) (*http.Response, error) {
	var payload struct {
		Points []struct {
			ID      uint64         `json:"id"`
			Vector  []float64      `json:"vector"`
			Payload map[string]any `json:"payload"`
		} `json:"points"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		return jsonHTTPResponse(http.StatusBadRequest, err.Error()), nil
	}
	s.mu.Lock()
	collection, ok := s.collections[name]
	if !ok {
		s.mu.Unlock()
		return jsonHTTPResponse(http.StatusNotFound, "not found"), nil
	}
	for _, point := range payload.Points {
		if len(point.Vector) != collection.size {
			s.mu.Unlock()
			return jsonHTTPResponse(http.StatusBadRequest, fmt.Sprintf("vector size %d does not match collection size %d", len(point.Vector), collection.size)), nil
		}
		collection.points[point.ID] = fakePoint{Vector: append([]float64(nil), point.Vector...), Payload: clonePayload(point.Payload)}
	}
	s.mu.Unlock()
	return jsonHTTPResponse(http.StatusOK, `{"status":"ok"}`), nil
}

func (s *fakeQdrantState) search(req *http.Request, name string) (*http.Response, error) {
	var payload struct {
		Vector []float64 `json:"vector"`
		Limit  int       `json:"limit"`
		Filter struct {
			Must []struct {
				Key   string `json:"key"`
				Match struct {
					Value string `json:"value"`
				} `json:"match"`
			} `json:"must"`
		} `json:"filter"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		return jsonHTTPResponse(http.StatusBadRequest, err.Error()), nil
	}
	s.mu.Lock()
	collection, ok := s.collections[name]
	s.mu.Unlock()
	if !ok {
		return jsonHTTPResponse(http.StatusNotFound, "not found"), nil
	}

	type scoredPoint struct {
		Score   float64
		Payload map[string]any
	}
	results := make([]scoredPoint, 0, len(collection.points))
	for _, point := range collection.points {
		if !matchesFilters(point.Payload, payload.Filter.Must) {
			continue
		}
		results = append(results, scoredPoint{
			Score:   cosineSimilarity(payload.Vector, point.Vector),
			Payload: clonePayload(point.Payload),
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			left, _ := results[i].Payload["document_id"].(string)
			right, _ := results[j].Payload["document_id"].(string)
			return left < right
		}
		return results[i].Score > results[j].Score
	})
	if payload.Limit > 0 && len(results) > payload.Limit {
		results = results[:payload.Limit]
	}
	response := make([]map[string]any, 0, len(results))
	for _, item := range results {
		response = append(response, map[string]any{
			"score":   item.Score,
			"payload": item.Payload,
		})
	}
	raw, _ := json.Marshal(map[string]any{"result": response})
	return jsonHTTPResponse(http.StatusOK, string(raw)), nil
}

func matchesFilters(payload map[string]any, must []struct {
	Key   string `json:"key"`
	Match struct {
		Value string `json:"value"`
	} `json:"match"`
}) bool {
	for _, filter := range must {
		value, ok := payload[filter.Key]
		if !ok || fmt.Sprintf("%v", value) != filter.Match.Value {
			return false
		}
	}
	return true
}

func cosineSimilarity(left, right []float64) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var dot float64
	var leftNorm float64
	var rightNorm float64
	for idx := range left {
		dot += left[idx] * right[idx]
		leftNorm += left[idx] * left[idx]
		rightNorm += right[idx] * right[idx]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
