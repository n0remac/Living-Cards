package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/memory"
	"github.com/n0remac/Living-Card/internal/ollama"
	"github.com/n0remac/Living-Card/internal/profile"
)

type PostChatJob struct {
	Card              cards.Card
	UserID            string
	UserInput         string
	AssistantResponse string
}

type MemoryWriter interface {
	SaveMemory(ctx context.Context, input memory.SaveInput) (memory.Memory, error)
}

type FactStore interface {
	ApplyFactUpdates(ctx context.Context, userID string, updates []profile.FactUpdate) ([]profile.Fact, error)
}

type ProcessorConfig struct {
	Memory         MemoryWriter
	Profile        FactStore
	Ollama         ChatClient
	ChatModel      string
	RequestTimeout time.Duration
	QueueSize      int
	Logger         func(format string, args ...any)
}

type BackgroundProcessor struct {
	memory         MemoryWriter
	profile        FactStore
	ollama         ChatClient
	chatModel      string
	requestTimeout time.Duration
	jobs           chan PostChatJob
	logger         func(format string, args ...any)
	wg             sync.WaitGroup
	closeOnce      sync.Once
}

func NewBackgroundProcessor(cfg ProcessorConfig) *BackgroundProcessor {
	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 32
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Printf
	}
	processor := &BackgroundProcessor{
		memory:         cfg.Memory,
		profile:        cfg.Profile,
		ollama:         cfg.Ollama,
		chatModel:      strings.TrimSpace(cfg.ChatModel),
		requestTimeout: cfg.RequestTimeout,
		jobs:           make(chan PostChatJob, queueSize),
		logger:         logger,
	}
	processor.wg.Add(1)
	go processor.run()
	return processor
}

func (p *BackgroundProcessor) Enqueue(job PostChatJob) {
	if p == nil {
		return
	}
	select {
	case p.jobs <- job:
	default:
		p.log("post-chat queue is full; dropping profile job for user %q card %q", job.UserID, job.Card.CardID)
	}
}

func (p *BackgroundProcessor) Close() error {
	if p == nil {
		return nil
	}
	p.closeOnce.Do(func() {
		close(p.jobs)
		p.wg.Wait()
	})
	return nil
}

func (p *BackgroundProcessor) run() {
	defer p.wg.Done()
	for job := range p.jobs {
		if err := p.process(job); err != nil {
			p.log("post-chat processing failed for user %q card %q: %v", job.UserID, job.Card.CardID, err)
		}
	}
}

func (p *BackgroundProcessor) process(job PostChatJob) error {
	if p.memory == nil || p.ollama == nil {
		return fmt.Errorf("post-chat processor is not initialized")
	}
	userID := profile.NormalizeUserID(job.UserID)
	ctx := context.Background()
	if p.requestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.requestTimeout)
		defer cancel()
	}

	summary, err := generateSummary(ctx, p.ollama, p.chatModel, job.Card, job.UserInput, job.AssistantResponse)
	if err != nil {
		return err
	}
	stored, err := p.memory.SaveMemory(ctx, memory.SaveInput{
		UserID:       userID,
		CardID:       job.Card.CardID,
		UserInput:    job.UserInput,
		CardResponse: job.AssistantResponse,
		Summary:      summary,
		Importance:   0.5,
	})
	if err != nil {
		return err
	}
	if p.profile == nil {
		return nil
	}

	updates, err := p.extractFactUpdates(ctx, job)
	if err != nil {
		return err
	}
	for idx := range updates {
		updates[idx].SourceMemoryID = stored.ID
	}
	_, err = p.profile.ApplyFactUpdates(ctx, userID, updates)
	return err
}

func (p *BackgroundProcessor) extractFactUpdates(ctx context.Context, job PostChatJob) ([]profile.FactUpdate, error) {
	systemPrompt := strings.Join([]string{
		"Extract stable user persona facts from this chat turn.",
		"Return strict JSON with this exact shape: {\"facts\":[{\"key\":\"preferences\",\"value\":\"likes concise technical explanations\",\"confidence\":0.86,\"evidence\":\"I prefer short technical explanations.\"}]}",
		"Allowed keys: identity.name, identity.pronouns, preferences, goals, background, communication_preferences, boundaries, recurring_topics.",
		"Extract only stable traits, preferences, goals, background, boundaries, or communication preferences.",
		"Do not extract one-off task details.",
		"Do not infer sensitive attributes unless explicitly stated by the user.",
		"Keep each fact atomic. Return {\"facts\":[]} when there are no stable facts.",
	}, "\n")
	userPrompt := fmt.Sprintf(
		"User input: %s\nAssistant response: %s\nReturn only JSON.",
		strings.TrimSpace(job.UserInput),
		strings.TrimSpace(job.AssistantResponse),
	)
	raw, err := p.ollama.Chat(ctx, p.chatModel, []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, err
	}
	return profile.ParseFactUpdates(raw)
}

func (p *BackgroundProcessor) log(format string, args ...any) {
	if p != nil && p.logger != nil {
		p.logger(format, args...)
	}
}
