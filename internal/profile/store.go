package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const (
	DefaultUserID             = "local-user"
	StatusCandidate           = "candidate"
	StatusAccepted            = "accepted"
	StatusRejected            = "rejected"
	StatusSuperseded          = "superseded"
	sqliteBusyTimeoutMillis   = 5000
	acceptedConfidenceCutoff  = 0.75
	minCandidateConfidence    = 0.35
	defaultProfileSummaryText = ""
)

var allowedFactKeys = map[string]struct{}{
	"identity.name":             {},
	"identity.pronouns":         {},
	"preferences":               {},
	"goals":                     {},
	"background":                {},
	"communication_preferences": {},
	"boundaries":                {},
	"recurring_topics":          {},
}

type Store struct {
	db *sql.DB
}

type Profile struct {
	UserID         string `json:"user_id"`
	ProfileSummary string `json:"profile_summary"`
	Facts          []Fact `json:"facts"`
}

type Fact struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	Key            string  `json:"key"`
	Value          string  `json:"value"`
	Confidence     float64 `json:"confidence"`
	Evidence       string  `json:"evidence"`
	SourceMemoryID string  `json:"source_memory_id,omitempty"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type FactUpdate struct {
	Key            string  `json:"key"`
	Value          string  `json:"value"`
	Confidence     float64 `json:"confidence"`
	Evidence       string  `json:"evidence"`
	SourceMemoryID string  `json:"source_memory_id,omitempty"`
}

var factSeq atomic.Uint64

func NewStore(path string) (*Store, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return nil, fmt.Errorf("profile db path cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create profile db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open profile db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db}
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

func (s *Store) Get(ctx context.Context, userID string) (Profile, error) {
	if s == nil || s.db == nil {
		return Profile{}, fmt.Errorf("profile store is not initialized")
	}
	userID = NormalizeUserID(userID)
	if err := s.ensureUser(ctx, userID); err != nil {
		return Profile{}, err
	}

	var summary string
	if err := s.db.QueryRowContext(ctx, `
		SELECT profile_summary
		FROM users
		WHERE id = ?
	`, userID).Scan(&summary); err != nil {
		return Profile{}, fmt.Errorf("get user profile: %w", err)
	}
	facts, err := s.listFacts(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	return Profile{
		UserID:         userID,
		ProfileSummary: summary,
		Facts:          facts,
	}, nil
}

func (s *Store) Summary(ctx context.Context, userID string) (string, error) {
	profile, err := s.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	return profile.ProfileSummary, nil
}

func (s *Store) ApplyFactUpdates(ctx context.Context, userID string, updates []FactUpdate) ([]Fact, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("profile store is not initialized")
	}
	userID = NormalizeUserID(userID)
	if err := s.ensureUser(ctx, userID); err != nil {
		return nil, err
	}

	sanitized := make([]FactUpdate, 0, len(updates))
	for _, update := range updates {
		item, ok := sanitizeFactUpdate(update)
		if !ok {
			continue
		}
		sanitized = append(sanitized, item)
	}
	if len(sanitized) == 0 {
		return nil, s.rebuildSummary(ctx, userID)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	inserted := make([]Fact, 0, len(sanitized))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin fact transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, update := range sanitized {
		status := StatusCandidate
		if update.Confidence >= acceptedConfidenceCutoff {
			status = StatusAccepted
		}
		fact := Fact{
			ID:             nextFactID(),
			UserID:         userID,
			Key:            update.Key,
			Value:          update.Value,
			Confidence:     update.Confidence,
			Evidence:       update.Evidence,
			SourceMemoryID: update.SourceMemoryID,
			Status:         status,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_facts (
				id, user_id, key, value, confidence, evidence, source_memory_id, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fact.ID, fact.UserID, fact.Key, fact.Value, fact.Confidence, fact.Evidence, nullableString(fact.SourceMemoryID), fact.Status, fact.CreatedAt, fact.UpdatedAt); err != nil {
			return nil, fmt.Errorf("insert user fact: %w", err)
		}
		inserted = append(inserted, fact)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit user facts: %w", err)
	}
	return inserted, s.rebuildSummary(ctx, userID)
}

func (s *Store) Reset(ctx context.Context, userID string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("profile store is not initialized")
	}
	userID = NormalizeUserID(userID)
	if err := s.ensureUser(ctx, userID); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM user_facts
		WHERE user_id = ? AND status IN (?, ?)
	`, userID, StatusAccepted, StatusCandidate); err != nil {
		return fmt.Errorf("reset user facts: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET profile_summary = ?, profile_version = profile_version + 1, updated_at = ?
		WHERE id = ?
	`, defaultProfileSummaryText, now, userID); err != nil {
		return fmt.Errorf("reset user profile: %w", err)
	}
	return nil
}

func ParseFactUpdates(raw string) ([]FactUpdate, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("fact extraction response cannot be empty")
	}
	var payload struct {
		Facts []FactUpdate `json:"facts"`
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode fact extraction response: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("fact extraction response must contain one JSON object")
	}
	updates := make([]FactUpdate, 0, len(payload.Facts))
	for _, fact := range payload.Facts {
		update, ok := sanitizeFactUpdate(fact)
		if !ok {
			return nil, fmt.Errorf("invalid fact update")
		}
		updates = append(updates, update)
	}
	return updates, nil
}

func NormalizeUserID(userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DefaultUserID
	}
	return userID
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
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			profile_summary TEXT NOT NULL,
			profile_version INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS user_facts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			confidence REAL NOT NULL,
			evidence TEXT NOT NULL,
			source_memory_id TEXT,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_user_facts_user_status ON user_facts(user_id, status, key);
	`)
	if err != nil {
		return fmt.Errorf("init profile schema: %w", err)
	}
	return nil
}

func (s *Store) ensureUser(ctx context.Context, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, created_at, updated_at, profile_summary, profile_version)
		VALUES (?, ?, ?, ?, 0)
		ON CONFLICT(id) DO NOTHING
	`, userID, now, now, defaultProfileSummaryText); err != nil {
		return fmt.Errorf("ensure user profile: %w", err)
	}
	return nil
}

func (s *Store) listFacts(ctx context.Context, userID string) ([]Fact, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, key, value, confidence, evidence, COALESCE(source_memory_id, ''), status, created_at, updated_at
		FROM user_facts
		WHERE user_id = ? AND status IN (?, ?)
		ORDER BY key ASC, created_at ASC, id ASC
	`, userID, StatusAccepted, StatusCandidate)
	if err != nil {
		return nil, fmt.Errorf("list user facts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	facts := make([]Fact, 0)
	for rows.Next() {
		var fact Fact
		if err := rows.Scan(
			&fact.ID,
			&fact.UserID,
			&fact.Key,
			&fact.Value,
			&fact.Confidence,
			&fact.Evidence,
			&fact.SourceMemoryID,
			&fact.Status,
			&fact.CreatedAt,
			&fact.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user fact: %w", err)
		}
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

func (s *Store) rebuildSummary(ctx context.Context, userID string) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, value
		FROM user_facts
		WHERE user_id = ? AND status = ?
		ORDER BY key ASC, created_at ASC, id ASC
	`, userID, StatusAccepted)
	if err != nil {
		return fmt.Errorf("list accepted facts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	lines := make([]string, 0)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("scan accepted fact: %w", err)
		}
		lines = append(lines, "- "+key+": "+value)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read accepted facts: %w", err)
	}
	sort.Strings(lines)
	summary := strings.Join(lines, "\n")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET profile_summary = ?, profile_version = profile_version + 1, updated_at = ?
		WHERE id = ?
	`, summary, now, userID); err != nil {
		return fmt.Errorf("update profile summary: %w", err)
	}
	return nil
}

func sanitizeFactUpdate(input FactUpdate) (FactUpdate, bool) {
	key := strings.TrimSpace(input.Key)
	value := strings.TrimSpace(input.Value)
	evidence := strings.TrimSpace(input.Evidence)
	if _, ok := allowedFactKeys[key]; !ok {
		return FactUpdate{}, false
	}
	if value == "" || evidence == "" {
		return FactUpdate{}, false
	}
	confidence := input.Confidence
	if confidence < minCandidateConfidence || confidence > 1 {
		return FactUpdate{}, false
	}
	return FactUpdate{
		Key:            key,
		Value:          value,
		Confidence:     confidence,
		Evidence:       evidence,
		SourceMemoryID: strings.TrimSpace(input.SourceMemoryID),
	}, true
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nextFactID() string {
	n := factSeq.Add(1)
	return "fact_" + strconv.FormatInt(time.Now().UTC().UnixMilli(), 10) + "_" + strconv.FormatUint(n, 10)
}
