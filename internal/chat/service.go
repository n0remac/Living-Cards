package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/n0remac/Living-Card/internal/cards"
	"github.com/n0remac/Living-Card/internal/memory"
	"github.com/n0remac/Living-Card/internal/ollama"
	"github.com/n0remac/Living-Card/internal/profile"
)

type CardCatalog interface {
	Get(cardID string) (cards.Card, bool)
}

type MemoryStore interface {
	Search(ctx context.Context, userID, cardID, query string, topK int) ([]memory.SearchResult, error)
}

type UserProfileReader interface {
	Summary(ctx context.Context, userID string) (string, error)
}

type PostChatProcessor interface {
	Enqueue(job PostChatJob)
}

type Config struct {
	Cards          CardCatalog
	Memory         MemoryStore
	Profile        UserProfileReader
	Processor      PostChatProcessor
	Ollama         ChatClient
	ChatModel      string
	RequestTimeout time.Duration
	TopK           int
}

type ChatClient interface {
	Chat(ctx context.Context, model string, messages []ollama.ChatMessage) (string, error)
}

type Service struct {
	cards          CardCatalog
	memory         MemoryStore
	profile        UserProfileReader
	processor      PostChatProcessor
	ollama         ChatClient
	chatModel      string
	requestTimeout time.Duration
	topK           int
}

type Request struct {
	UserID  string `json:"user_id"`
	CardID  string `json:"card_id"`
	Message string `json:"message"`
}

type Result struct {
	UserID            string                `json:"user_id"`
	Card              cards.Card            `json:"card"`
	AssistantResponse string                `json:"assistant_response"`
	RetrievedMemories []memory.SearchResult `json:"retrieved_memories"`
	StoredMemory      memory.Memory         `json:"stored_memory"`
}

func NewService(cfg Config) *Service {
	topK := cfg.TopK
	if topK <= 0 {
		topK = 3
	}
	return &Service{
		cards:          cfg.Cards,
		memory:         cfg.Memory,
		profile:        cfg.Profile,
		processor:      cfg.Processor,
		ollama:         cfg.Ollama,
		chatModel:      strings.TrimSpace(cfg.ChatModel),
		requestTimeout: cfg.RequestTimeout,
		topK:           topK,
	}
}

func (s *Service) Chat(ctx context.Context, request Request) (Result, error) {
	if s == nil || s.cards == nil || s.memory == nil || s.ollama == nil {
		return Result{}, fmt.Errorf("chat service is not initialized")
	}
	cardID := strings.TrimSpace(request.CardID)
	userID := profile.NormalizeUserID(request.UserID)
	message := strings.TrimSpace(request.Message)
	if cardID == "" {
		return Result{}, fmt.Errorf("card_id cannot be empty")
	}
	if message == "" {
		return Result{}, fmt.Errorf("message cannot be empty")
	}
	card, ok := s.cards.Get(cardID)
	if !ok {
		return Result{}, ErrCardNotFound
	}

	callCtx := ctx
	if callCtx == nil {
		callCtx = context.Background()
	}
	if _, ok := callCtx.Deadline(); !ok && s.requestTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(callCtx, s.requestTimeout)
		defer cancel()
	}

	userProfile := ""
	if s.profile != nil {
		var err error
		userProfile, err = s.profile.Summary(callCtx, userID)
		if err != nil {
			return Result{}, err
		}
	}

	retrieved, err := s.memory.Search(callCtx, userID, cardID, message, s.topK)
	if err != nil {
		return Result{}, err
	}
	systemPrompt, userPrompt := BuildPrompt(card, message, userProfile, retrieved)
	reply, err := s.ollama.Chat(callCtx, s.chatModel, []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return Result{}, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return Result{}, fmt.Errorf("ollama returned an empty response")
	}

	if s.processor != nil {
		s.processor.Enqueue(PostChatJob{
			Card:              card,
			UserID:            userID,
			UserInput:         message,
			AssistantResponse: reply,
		})
	}
	return Result{
		UserID:            userID,
		Card:              card,
		AssistantResponse: reply,
		RetrievedMemories: retrieved,
	}, nil
}

func generateSummary(ctx context.Context, client ChatClient, model string, card cards.Card, userInput, response string) (string, error) {
	systemPrompt := "Summarize the interaction for future semantic retrieval. Return one concise sentence. Do not add labels."
	userPrompt := fmt.Sprintf(
		"Card: %s\nUser input: %s\nCard response: %s\nWrite a single-sentence summary.",
		card.Name,
		strings.TrimSpace(userInput),
		strings.TrimSpace(response),
	)
	summary, err := client.Chat(ctx, model, []ollama.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return "", err
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "", fmt.Errorf("ollama returned an empty summary")
	}
	return summary, nil
}

func BuildPrompt(card cards.Card, userInput string, userProfile string, retrieved []memory.SearchResult) (string, string) {
	domainLine := ""
	if len(card.Domain) > 0 {
		domainLine = "- Domain: " + strings.Join(card.Domain, ", ") + "\n"
	}

	styleRules := ""
	for _, rule := range card.Personality.StyleRules {
		styleRules += "- " + rule + "\n"
	}

	knowledgeScope := ""
	if card.Constraints.KnowledgeScope != "" {
		knowledgeScope = "\nKnowledge scope:\n- " + card.Constraints.KnowledgeScope + "\n"
	}

	systemPrompt := "You are " + card.Name + ".\n\nIdentity:\n" +
		"- Archetype: " + fallback(card.Archetype, "persistent card entity") + "\n" +
		domainLine +
		"\nPersonality rules:\n" +
		"- Tone: " + fallback(card.Personality.Tone, "consistent and stylized") + "\n" +
		styleRules +
		"- Keep responses concise.\n" +
		"- Stay in character. Respond in a normal conversational tone. Be inspired by your character but don't over fit to it." +
		knowledgeScope +
		"\nDo not behave like a generic assistant. Respond as the same persistent entity every time. Avoid common phrases, like 'How may I help you'\n" +
		"- Use known user profile context naturally when relevant.\n" +
		"- Do not mention the profile unless it directly helps the response.\n" +
		"- Do not invent facts about the user."

	knownUser := strings.TrimSpace(userProfile)
	if knownUser == "" {
		knownUser = "- none"
	}

	relevantMemories := "- none\n"
	if len(retrieved) > 0 {
		relevantMemories = ""
		for _, item := range retrieved {
			relevantMemories += "- " + item.Memory.Summary + "\n"
		}
	}

	userPrompt := "Known user profile:\n" +
		knownUser +
		"\n\nRelevant memories:\n" +
		relevantMemories +
		"\nUser says:\n" +
		strings.TrimSpace(userInput) +
		"\n\nRespond in character."

	return systemPrompt, userPrompt
}

func fallback(value, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return value
}
