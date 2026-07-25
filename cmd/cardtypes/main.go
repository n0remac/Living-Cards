package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/n0remac/Living-Card/internal/components/catalog"
	"github.com/n0remac/Living-Card/internal/components/tsgen"
)

func main() {
	check := flag.Bool("check", false, "fail when generated TypeScript files are stale")
	flag.Parse()

	root, err := projectRoot()
	if err != nil {
		fail(err)
	}
	files, err := tsgen.Generate(catalog.MustNew().Schema())
	if err != nil {
		fail(err)
	}
	for _, relativePath := range tsgen.SortedPaths(files) {
		path := filepath.Join(root, filepath.FromSlash(relativePath))
		if *check {
			current, err := os.ReadFile(path)
			if err != nil {
				fail(fmt.Errorf("%s is missing; run go run ./cmd/cardtypes: %w", relativePath, err))
			}
			if !bytes.Equal(current, files[relativePath]) {
				fail(fmt.Errorf("%s is stale; run go run ./cmd/cardtypes", relativePath))
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fail(err)
		}
		if err := os.WriteFile(path, files[relativePath], 0o644); err != nil {
			fail(err)
		}
		fmt.Println("generated", relativePath)
	}
}

func projectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("could not locate project root")
		}
		directory = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "cardtypes:", err)
	os.Exit(1)
}
