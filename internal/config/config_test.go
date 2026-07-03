package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDevModeDefault(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DevMode {
		t.Fatal("DevMode = true, want false")
	}
}

func TestLoadDevModeEnabledValues(t *testing.T) {
	for _, value := range []string{"1", "true", "yes", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DEV_MODE", value)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !cfg.DevMode {
				t.Fatalf("DevMode = false for %q, want true", value)
			}
		})
	}
}

func TestLoadDevModeDisabledValues(t *testing.T) {
	for _, value := range []string{"0", "false", "no", "off"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("DEV_MODE", value)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.DevMode {
				t.Fatalf("DevMode = true for %q, want false", value)
			}
		})
	}
}

func TestLoadDevModeInvalid(t *testing.T) {
	t.Setenv("DEV_MODE", "sometimes")
	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "DEV_MODE must be a boolean") {
		t.Fatalf("Load() error = %q", err.Error())
	}
}

func TestLoadRetainedConfigFields(t *testing.T) {
	t.Setenv("OLLAMA_BASE_URL", "http://ollama.local:11434")
	t.Setenv("OLLAMA_CHAT_MODEL", "card-model")
	t.Setenv("WEB_ADDR", "127.0.0.1:9090")
	t.Setenv("REQUEST_TIMEOUT_SECONDS", "12")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OllamaBaseURL != "http://ollama.local:11434" {
		t.Fatalf("OllamaBaseURL = %q", cfg.OllamaBaseURL)
	}
	if cfg.OllamaChatModel != "card-model" {
		t.Fatalf("OllamaChatModel = %q", cfg.OllamaChatModel)
	}
	if cfg.WebAddr != "127.0.0.1:9090" {
		t.Fatalf("WebAddr = %q", cfg.WebAddr)
	}
	if cfg.RequestTimeout != 12*time.Second {
		t.Fatalf("RequestTimeout = %s", cfg.RequestTimeout)
	}
}

func TestLoadIgnoresRemovedLegacyEnvVars(t *testing.T) {
	t.Setenv("OLLAMA_EMBEDDING_MODEL", "")
	t.Setenv("QDRANT_BASE_URL", "not a url")
	t.Setenv("QDRANT_API_KEY", "legacy-key")
	t.Setenv("QDRANT_COLLECTION_PREFIX", "")
	t.Setenv("MEMORY_DB_PATH", "")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsInvalidRetainedValues(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		message string
	}{
		{name: "ollama base url", key: "OLLAMA_BASE_URL", value: "://bad", message: "OLLAMA_BASE_URL"},
		{name: "chat model", key: "OLLAMA_CHAT_MODEL", value: " ", message: "OLLAMA_CHAT_MODEL cannot be empty"},
		{name: "web addr", key: "WEB_ADDR", value: " ", message: "WEB_ADDR cannot be empty"},
		{name: "timeout", key: "REQUEST_TIMEOUT_SECONDS", value: "0", message: "REQUEST_TIMEOUT_SECONDS must be > 0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.key, test.value)
			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Load() error = %q, want containing %q", err.Error(), test.message)
			}
		})
	}
}
