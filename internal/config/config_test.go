package config

import (
	"strings"
	"testing"
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
