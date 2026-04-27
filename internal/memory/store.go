package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/n0remac/Living-Card/internal/embedding"
	_ "modernc.org/sqlite"
)

const sqliteBusyTimeoutMillis = 5000

type VectorIndex interface {
	CollectionName(model string) string
	UpsertDocuments(ctx context.Context, model string, docs []embedding.Document) (string, error)
	Search(ctx context.Context, model, query string, topK int, filters map[string]string) (embedding.SearchResponse, error)
}

type Memory struct {
	ID             string   `json:"id"`
	CardID         string   `json:"card_id"`
	Timestamp      string   `json:"timestamp"`
	UserInput      string   `json:"user_input"`
	CardResponse   string   `json:"card_response"`
	Summary        string   `json:"summary"`
	Tags           []string `json:"tags"`
	Importance     float64  `json:"importance"`
	CollectionName string   `json:"collection_name,omitempty"`
}

type SaveInput struct {
	CardID       string
	UserInput    string
	CardResponse string
	Summary      string
	Tags         []string
	Importance   float64
}

type SearchResult struct {
	Memory Memory  `json:"memory"`
	Rank   int     `json:"rank"`
	Score  float64 `json:"score"`
}

type Store struct {
	db             *sql.DB
	index          VectorIndex
	embeddingModel string
}

var memorySeq atomic.Uint64

func NewStore(path string, index VectorIndex, embeddingModel string) (*Store, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return nil, fmt.Errorf("memory db path cannot be empty")
	}
	if index == nil {
		return nil, fmt.Errorf("memory vector index is required")
	}
	embeddingModel = strings.TrimSpace(embeddingModel)
	if embeddingModel == "" {
		return nil, fmt.Errorf("embedding model cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create memory db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open memory db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{
		db:             db,
		index:          index,
		embeddingModel: embeddingModel,
	}
	if err := store.configure(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) SaveMemory(ctx context.Context, input SaveInput) (Memory, error) {
	if s == nil || s.db == nil || s.index == nil {
		return Memory{}, fmt.Errorf("memory store is not initialized")
	}
	memory, err := sanitizeSaveInput(input)
	if err != nil {
		return Memory{}, err
	}

	document := embedding.Document{
		DocumentID: memory.ID,
		Text:       memory.Summary,
		Payload: map[string]any{
			"card_id":    memory.CardID,
			"memory_id":  memory.ID,
			"timestamp":  memory.Timestamp,
			"importance": memory.Importance,
			"summary":    memory.Summary,
		},
	}
	collectionName, err := s.index.UpsertDocuments(ctx, s.embeddingModel, []embedding.Document{document})
	if err != nil {
		return Memory{}, err
	}
	memory.CollectionName = collectionName

	tagsJSON, err := json.Marshal(memory.Tags)
	if err != nil {
		return Memory{}, fmt.Errorf("marshal tags: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO memories (
			id, card_id, timestamp, user_input, card_response, summary, tags_json, importance, collection_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, memory.ID, memory.CardID, memory.Timestamp, memory.UserInput, memory.CardResponse, memory.Summary, string(tagsJSON), memory.Importance, memory.CollectionName); err != nil {
		return Memory{}, fmt.Errorf("insert memory: %w", err)
	}
	return memory, nil
}

func (s *Store) ListByCard(ctx context.Context, cardID string, limit int) ([]Memory, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("memory store is not initialized")
	}
	cardID = strings.TrimSpace(cardID)
	if cardID == "" {
		return nil, fmt.Errorf("card_id cannot be empty")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, card_id, timestamp, user_input, card_response, summary, tags_json, importance, collection_name
		FROM memories
		WHERE card_id = ?
		ORDER BY timestamp DESC, id DESC
		LIMIT ?
	`, cardID, limit)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	memories := make([]Memory, 0, limit)
	for rows.Next() {
		memory, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}
	return memories, rows.Err()
}

func (s *Store) Search(ctx context.Context, cardID, query string, topK int) ([]SearchResult, error) {
	if s == nil || s.index == nil {
		return nil, fmt.Errorf("memory store is not initialized")
	}
	cardID = strings.TrimSpace(cardID)
	query = strings.TrimSpace(query)
	if cardID == "" {
		return nil, fmt.Errorf("card_id cannot be empty")
	}
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if topK <= 0 {
		topK = 3
	}
	response, err := s.index.Search(ctx, s.embeddingModel, query, topK, map[string]string{"card_id": cardID})
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(response.Results))
	for _, item := range response.Results {
		memory, err := s.GetByID(ctx, item.DocumentID)
		if err != nil {
			if item.DocumentID == "" {
				continue
			}
			return nil, err
		}
		results = append(results, SearchResult{
			Memory: memory,
			Rank:   item.Rank,
			Score:  item.Score,
		})
	}
	return results, nil
}

func (s *Store) GetByID(ctx context.Context, id string) (Memory, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, card_id, timestamp, user_input, card_response, summary, tags_json, importance, collection_name
		FROM memories
		WHERE id = ?
	`, strings.TrimSpace(id))
	return scanMemory(row)
}

func (s *Store) configure() error {
	if _, err := s.db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", sqliteBusyTimeoutMillis)); err != nil {
		return fmt.Errorf("sqlite busy timeout: %w", err)
	}
	if _, err := s.db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return fmt.Errorf("sqlite journal mode: %w", err)
	}
	return nil
}

func (s *Store) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			card_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			user_input TEXT NOT NULL,
			card_response TEXT NOT NULL,
			summary TEXT NOT NULL,
			tags_json TEXT NOT NULL,
			importance REAL NOT NULL,
			collection_name TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_memories_card_time ON memories(card_id, timestamp DESC);
	`)
	if err != nil {
		return fmt.Errorf("init memory schema: %w", err)
	}
	return nil
}

func sanitizeSaveInput(input SaveInput) (Memory, error) {
	cardID := strings.TrimSpace(input.CardID)
	userInput := strings.TrimSpace(input.UserInput)
	cardResponse := strings.TrimSpace(input.CardResponse)
	summary := strings.TrimSpace(input.Summary)
	if cardID == "" {
		return Memory{}, fmt.Errorf("card_id cannot be empty")
	}
	if userInput == "" {
		return Memory{}, fmt.Errorf("user_input cannot be empty")
	}
	if cardResponse == "" {
		return Memory{}, fmt.Errorf("card_response cannot be empty")
	}
	if summary == "" {
		return Memory{}, fmt.Errorf("summary cannot be empty")
	}
	importance := input.Importance
	if importance <= 0 {
		importance = 0.5
	}
	if importance > 1 {
		importance = 1
	}
	return Memory{
		ID:           nextMemoryID(),
		CardID:       cardID,
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		UserInput:    userInput,
		CardResponse: cardResponse,
		Summary:      summary,
		Tags:         sanitizeTags(input.Tags),
		Importance:   importance,
	}, nil
}

func sanitizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		out = append(out, tag)
	}
	return out
}

func nextMemoryID() string {
	n := memorySeq.Add(1)
	return "mem_" + strconv.FormatInt(time.Now().UTC().UnixMilli(), 10) + "_" + strconv.FormatUint(n, 10)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMemory(row scanner) (Memory, error) {
	var memory Memory
	var tagsJSON string
	if err := row.Scan(
		&memory.ID,
		&memory.CardID,
		&memory.Timestamp,
		&memory.UserInput,
		&memory.CardResponse,
		&memory.Summary,
		&tagsJSON,
		&memory.Importance,
		&memory.CollectionName,
	); err != nil {
		return Memory{}, err
	}
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &memory.Tags); err != nil {
			return Memory{}, fmt.Errorf("decode memory tags: %w", err)
		}
	}
	return memory, nil
}
