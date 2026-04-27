package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOllamaBaseURL        = "http://127.0.0.1:11434"
	defaultChatModel            = "qwen2.5:3b-instruct"
	defaultEmbeddingModel       = "qwen3-embedding:4b"
	defaultQdrantBaseURL        = "http://127.0.0.1:6333"
	defaultQdrantPrefix         = "living-card-v1"
	defaultMemoryDBPath         = "data/state/memory.db"
	defaultCardsDir             = "data/cards"
	defaultWebAddr              = "127.0.0.1:8090"
	defaultRequestTimeoutSecond = 45
)

type Config struct {
	OllamaBaseURL          string
	OllamaChatModel        string
	OllamaEmbeddingModel   string
	QdrantBaseURL          string
	QdrantAPIKey           string
	QdrantCollectionPrefix string
	MemoryDBPath           string
	CardsDir               string
	WebAddr                string
	RequestTimeout         time.Duration
}

func Load() (Config, error) {
	requestTimeoutSeconds, err := readIntEnv("REQUEST_TIMEOUT_SECONDS", defaultRequestTimeoutSecond)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		OllamaBaseURL:          strings.TrimRight(readEnvOrDefault("OLLAMA_BASE_URL", defaultOllamaBaseURL), "/"),
		OllamaChatModel:        strings.TrimSpace(readEnvOrDefault("OLLAMA_CHAT_MODEL", defaultChatModel)),
		OllamaEmbeddingModel:   strings.TrimSpace(readEnvOrDefault("OLLAMA_EMBEDDING_MODEL", defaultEmbeddingModel)),
		QdrantBaseURL:          strings.TrimRight(readEnvOrDefault("QDRANT_BASE_URL", defaultQdrantBaseURL), "/"),
		QdrantAPIKey:           strings.TrimSpace(os.Getenv("QDRANT_API_KEY")),
		QdrantCollectionPrefix: strings.TrimSpace(readEnvOrDefault("QDRANT_COLLECTION_PREFIX", defaultQdrantPrefix)),
		MemoryDBPath:           filepath.Clean(readEnvOrDefault("MEMORY_DB_PATH", defaultMemoryDBPath)),
		CardsDir:               filepath.Clean(readEnvOrDefault("CARDS_DIR", defaultCardsDir)),
		WebAddr:                strings.TrimSpace(readEnvOrDefault("WEB_ADDR", defaultWebAddr)),
		RequestTimeout:         time.Duration(requestTimeoutSeconds) * time.Second,
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validate(cfg Config) error {
	if err := validateBaseURL("OLLAMA_BASE_URL", cfg.OllamaBaseURL); err != nil {
		return err
	}
	if err := validateBaseURL("QDRANT_BASE_URL", cfg.QdrantBaseURL); err != nil {
		return err
	}
	if cfg.OllamaChatModel == "" {
		return fmt.Errorf("OLLAMA_CHAT_MODEL cannot be empty")
	}
	if cfg.OllamaEmbeddingModel == "" {
		return fmt.Errorf("OLLAMA_EMBEDDING_MODEL cannot be empty")
	}
	if cfg.QdrantCollectionPrefix == "" {
		return fmt.Errorf("QDRANT_COLLECTION_PREFIX cannot be empty")
	}
	if cfg.MemoryDBPath == "" || cfg.MemoryDBPath == "." {
		return fmt.Errorf("MEMORY_DB_PATH cannot be empty")
	}
	if cfg.CardsDir == "" || cfg.CardsDir == "." {
		return fmt.Errorf("CARDS_DIR cannot be empty")
	}
	if cfg.WebAddr == "" {
		return fmt.Errorf("WEB_ADDR cannot be empty")
	}
	if cfg.RequestTimeout <= 0 {
		return fmt.Errorf("REQUEST_TIMEOUT_SECONDS must be > 0")
	}
	return nil
}

func validateBaseURL(name, raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("%s invalid: %w", name, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must include scheme and host", name)
	}
	return nil
}

func readEnvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func readIntEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return parsed, nil
}
