package cardimages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var nonSlugCharacter = regexp.MustCompile(`[^a-z0-9]+`)

type Manifest struct {
	Description string    `json:"description"`
	Selection   Selection `json:"selection"`
	Prompt      string    `json:"prompt"`
	DeckID      string    `json:"deck_id,omitempty"`
	CardID      string    `json:"card_id,omitempty"`
	CardName    string    `json:"card_name,omitempty"`
	Seed        int64     `json:"seed"`
	Index       int       `json:"index"`
	Model       string    `json:"model"`
	Size        string    `json:"size"`
	Quality     string    `json:"quality"`
	Format      string    `json:"format"`
	ImageFile   string    `json:"image_file"`
	RequestID   string    `json:"request_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func AssetBaseName(prompt Prompt, seed int64, index int) string {
	parts := []string{
		slug(prompt.Selection.Technology),
		slug(prompt.Selection.NaturePrimary),
		fmt.Sprintf("%d", seed),
		fmt.Sprintf("%02d", index),
	}
	return strings.Join(parts, "-")
}

func EnsureAvailable(imagePath, manifestPath string, force bool) error {
	if force {
		return nil
	}
	for _, path := range []string{imagePath, manifestPath} {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use -force to overwrite it", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect %s: %w", path, err)
		}
	}
	return nil
}

func WriteAsset(imagePath, manifestPath string, image []byte, manifest Manifest) error {
	if len(image) == 0 {
		return fmt.Errorf("generated image is empty")
	}
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		return fmt.Errorf("create image directory: %w", err)
	}
	if filepath.Dir(manifestPath) != filepath.Dir(imagePath) {
		if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
			return fmt.Errorf("create manifest directory: %w", err)
		}
	}

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode image manifest: %w", err)
	}
	manifestJSON = append(manifestJSON, '\n')

	imageTemp, err := writeTemp(filepath.Dir(imagePath), image)
	if err != nil {
		return fmt.Errorf("stage image: %w", err)
	}
	defer os.Remove(imageTemp)
	manifestTemp, err := writeTemp(filepath.Dir(manifestPath), manifestJSON)
	if err != nil {
		return fmt.Errorf("stage manifest: %w", err)
	}
	defer os.Remove(manifestTemp)

	if err := os.Rename(imageTemp, imagePath); err != nil {
		return fmt.Errorf("save image: %w", err)
	}
	if err := os.Rename(manifestTemp, manifestPath); err != nil {
		return fmt.Errorf("save manifest (image was saved at %s): %w", imagePath, err)
	}
	return nil
}

func writeTemp(directory string, contents []byte) (string, error) {
	file, err := os.CreateTemp(directory, ".cardimage-*")
	if err != nil {
		return "", err
	}
	path := file.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return "", err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	ok = true
	return path, nil
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(nonSlugCharacter.ReplaceAllString(value, "-"), "-")
	if value == "" {
		return "image"
	}
	return value
}
