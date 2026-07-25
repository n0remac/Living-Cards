package cardimages

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteAsset(t *testing.T) {
	directory := t.TempDir()
	imagePath := filepath.Join(directory, "nested", "card.webp")
	manifestPath := filepath.Join(directory, "nested", "card.json")
	manifest := Manifest{
		Description: "description",
		ImageFile:   "card.webp",
		CreatedAt:   time.Date(2026, 7, 25, 1, 2, 3, 0, time.UTC),
	}
	if err := WriteAsset(imagePath, manifestPath, []byte("image"), manifest); err != nil {
		t.Fatal(err)
	}
	image, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(image) != "image" {
		t.Fatalf("image = %q", image)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ImageFile != manifest.ImageFile || !decoded.CreatedAt.Equal(manifest.CreatedAt) {
		t.Fatalf("manifest = %#v", decoded)
	}
}

func TestEnsureAvailable(t *testing.T) {
	directory := t.TempDir()
	imagePath := filepath.Join(directory, "card.webp")
	manifestPath := filepath.Join(directory, "card.json")
	if err := EnsureAvailable(imagePath, manifestPath, false); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureAvailable(imagePath, manifestPath, false); err == nil {
		t.Fatal("expected existing file error")
	}
	if err := EnsureAvailable(imagePath, manifestPath, true); err != nil {
		t.Fatal(err)
	}
}

func TestAssetBaseName(t *testing.T) {
	prompt := Prompt{Selection: Selection{
		Technology:    "Cracked Pressure Gauge",
		NaturePrimary: "Pale Mycelium",
	}}
	if got, want := AssetBaseName(prompt, 42, 3), "cracked-pressure-gauge-pale-mycelium-42-03"; got != want {
		t.Fatalf("AssetBaseName() = %q, want %q", got, want)
	}
}
