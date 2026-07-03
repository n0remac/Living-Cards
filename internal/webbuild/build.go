package webbuild

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	entryPoint = "web/src/app.ts"
	outFile    = "web/dist/app.js"
)

func BuildFrontend() error {
	root, err := projectRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, "web", "dist"), 0o755); err != nil {
		return fmt.Errorf("create frontend dist dir: %w", err)
	}

	result := api.Build(api.BuildOptions{
		AbsWorkingDir: root,
		EntryPoints:   []string{entryPoint},
		Bundle:        true,
		Outfile:       outFile,
		Write:         true,
		Format:        api.FormatESModule,
		Platform:      api.PlatformBrowser,
		Target:        api.ES2020,
		Sourcemap:     api.SourceMapLinked,
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
	})

	if len(result.Errors) > 0 {
		return fmt.Errorf("frontend build failed: %s", formatMessages(result.Errors))
	}
	return nil
}

func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect go.mod: %w", err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate project root")
		}
		dir = parent
	}
}

func formatMessages(messages []api.Message) string {
	out := make([]string, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(message.Text)
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return strings.Join(out, "; ")
}
