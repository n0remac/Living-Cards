package main

import (
	"log"
	"net/http"

	"github.com/n0remac/Living-Card/internal/config"
	"github.com/n0remac/Living-Card/internal/ollama"
	"github.com/n0remac/Living-Card/internal/web"
	"github.com/n0remac/Living-Card/internal/webbuild"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.DevMode {
		if err := webbuild.BuildFrontend(); err != nil {
			return err
		}
	}

	ollamaClient := ollama.NewClient(cfg.OllamaBaseURL, cfg.RequestTimeout)

	mux := http.NewServeMux()
	web.Register(mux, web.Dependencies{
		Patch:      ollamaClient,
		PatchModel: cfg.OllamaChatModel,
	})

	log.Printf("living card server listening on http://%s", cfg.WebAddr)
	return http.ListenAndServe(cfg.WebAddr, mux)
}
