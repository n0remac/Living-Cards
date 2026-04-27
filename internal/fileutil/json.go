package fileutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ReadJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty json file: %s", path)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode json file %q: %w", path, err)
	}
	return nil
}

func WriteJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')

	file, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
