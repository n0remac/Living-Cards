package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/chat"
	"github.com/n0remac/Living-Card/internal/config"
	"github.com/n0remac/Living-Card/internal/embedding"
	"github.com/n0remac/Living-Card/internal/memory"
	"github.com/n0remac/Living-Card/internal/ollama"
	"github.com/n0remac/Living-Card/internal/web"
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

	cardStore, err := cards.NewStore(cfg.CardsDir)
	if err != nil {
		return fmt.Errorf("load cards: %w", err)
	}

	ollamaClient := ollama.NewClient(cfg.OllamaBaseURL, cfg.RequestTimeout)
	index, err := embedding.New(ollamaClient, embedding.Config{
		QdrantBaseURL:    cfg.QdrantBaseURL,
		QdrantAPIKey:     cfg.QdrantAPIKey,
		CollectionPrefix: cfg.QdrantCollectionPrefix,
		RequestTimeout:   cfg.RequestTimeout,
	})
	if err != nil {
		return fmt.Errorf("init embedding index: %w", err)
	}

	memoryStore, err := memory.NewStore(cfg.MemoryDBPath, index, cfg.OllamaEmbeddingModel)
	if err != nil {
		return fmt.Errorf("init memory store: %w", err)
	}
	defer func() {
		_ = memoryStore.Close()
	}()

	service := chat.NewService(chat.Config{
		Cards:          cardStore,
		Memory:         memoryStore,
		Ollama:         ollamaClient,
		ChatModel:      cfg.OllamaChatModel,
		RequestTimeout: cfg.RequestTimeout,
		TopK:           3,
	})

	mux := http.NewServeMux()
	web.Register(mux, web.Dependencies{
		Cards:  cardStore,
		Memory: memoryStore,
		Chat:   service,
	})

	log.Printf("living card server listening on http://%s", cfg.WebAddr)
	return http.ListenAndServe(cfg.WebAddr, mux)
}
