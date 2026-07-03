package webbuild

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFrontend(t *testing.T) {
	if err := BuildFrontend(); err != nil {
		t.Fatalf("BuildFrontend() error = %v", err)
	}
	root, err := projectRoot()
	if err != nil {
		t.Fatalf("projectRoot() error = %v", err)
	}
	for _, name := range []string{"app.js", "app.js.map"} {
		path := filepath.Join(root, "web", "dist", name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("os.Stat(%q) error = %v", path, err)
		}
	}
}
