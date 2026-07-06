package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const (
	KindWorld = "world"
	KindItem  = "item"
	KindClue  = "clue"

	DoorUnlockedFlag = "doorUnlocked"

	BlankControllerCardID     = "blank-controller"
	SliderComponentCardID     = "slider-component"
	RegulatorControllerCardID = "generator-regulator-controller"
	GeneratorPoweredFlag      = "generatorPowered"
)

type Card struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Kind        string                 `json:"kind"`
	Tags        []string               `json:"tags,omitempty"`
	Collectible bool                   `json:"collectible"`
	Collected   bool                   `json:"collected,omitempty"`
	State       map[string]any         `json:"state,omitempty"`
	Document    cardcomponent.Document `json:"document"`
}

type Snapshot struct {
	WorldDeck         []Card          `json:"worldDeck"`
	ActiveWorldCard   Card            `json:"activeWorldCard"`
	ActiveWorldCardID string          `json:"activeWorldCardId"`
	ActiveIndex       int             `json:"activeIndex"`
	Library           []Card          `json:"library"`
	SolvedFlags       map[string]bool `json:"solvedFlags"`
	Message           string          `json:"message,omitempty"`
}

type Session struct {
	mu               sync.Mutex
	deckDefinition   DeckDefinition
	cardDefinitions  map[string]CardDefinition
	documentVariants map[string]map[string]cardcomponent.Document
	loadedDecks      map[string]bool
	useRules         []UseRuleDefinition
	worldDeck        []Card
	activeIndex      int
	library          []Card
	solvedFlags      map[string]bool
	lastMessage      string
}

func NewSession() *Session {
	session, err := NewSessionFromEmbeddedDeck()
	if err != nil {
		panic(err)
	}
	return session
}

func NewSessionFromEmbeddedDeck() (*Session, error) {
	definition, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		return nil, err
	}
	return NewSessionFromDeck(definition)
}

func NewSessionFromDeck(definition DeckDefinition) (*Session, error) {
	if err := ValidateDeckDefinition(definition); err != nil {
		return nil, err
	}
	definition = cloneValue(definition)
	worldDeck, documentVariants, cardDefinitions, activeIndex, err := materializeDeck(definition)
	if err != nil {
		return nil, err
	}
	return &Session{
		deckDefinition:   definition,
		cardDefinitions:  cardDefinitions,
		documentVariants: documentVariants,
		loadedDecks:      map[string]bool{definition.ID: true},
		useRules:         cloneValue(definition.UseRules),
		worldDeck:        worldDeck,
		activeIndex:      activeIndex,
		library:          nil,
		solvedFlags:      cloneValue(definition.InitialSolvedFlags),
		lastMessage:      definition.InitialMessage,
	}, nil
}

func (s *Session) Snapshot() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.snapshotLocked()
}

func (s *Session) Reset() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next, err := NewSessionFromDeck(s.deckDefinition)
	if err != nil {
		return Snapshot{}, err
	}
	s.cardDefinitions = next.cardDefinitions
	s.documentVariants = next.documentVariants
	s.loadedDecks = next.loadedDecks
	s.useRules = next.useRules
	s.worldDeck = next.worldDeck
	s.activeIndex = next.activeIndex
	s.library = next.library
	s.solvedFlags = next.solvedFlags
	s.lastMessage = next.lastMessage
	return s.snapshotLocked()
}

func (s *Session) Cycle(direction string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.worldDeck) == 0 {
		return Snapshot{}, fmt.Errorf("world deck is empty")
	}
	switch strings.TrimSpace(direction) {
	case "previous", "prev", "back":
		s.activeIndex--
	case "", "next":
		s.activeIndex++
	default:
		return Snapshot{}, fmt.Errorf("direction must be next or previous")
	}
	if s.activeIndex < 0 {
		s.activeIndex = len(s.worldDeck) - 1
	}
	if s.activeIndex >= len(s.worldDeck) {
		s.activeIndex = 0
	}
	s.lastMessage = "The next card slides into view."
	return s.snapshotLocked()
}

func (s *Session) Collect(cardID string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cardID = strings.TrimSpace(cardID)
	if cardID == "" && len(s.worldDeck) > 0 {
		cardID = s.worldDeck[s.activeIndex].ID
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return Snapshot{}, fmt.Errorf("card %q is not in the world deck", cardID)
	}
	card := s.worldDeck[index]
	if !card.Collectible {
		return Snapshot{}, fmt.Errorf("%s cannot be collected", card.Name)
	}
	if card.Collected {
		s.lastMessage = card.Name + " is already in your library."
		return s.snapshotLocked()
	}
	card.Collected = true
	card.Collectible = false
	s.worldDeck[index] = card
	libraryCard := cloneValue(card)
	libraryCard.Collectible = false
	libraryCard.Collected = true
	s.library = append(s.library, libraryCard)
	s.lastMessage = card.Name + " moved into your library."
	return s.snapshotLocked()
}

func (s *Session) UseCard(sourceCardID, targetCardID string) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceCardID = strings.TrimSpace(sourceCardID)
	targetCardID = strings.TrimSpace(targetCardID)
	source := s.libraryCard(sourceCardID)
	if source == nil {
		return Snapshot{}, fmt.Errorf("card %q is not in your library", sourceCardID)
	}
	if targetCardID == "" && len(s.worldDeck) > 0 {
		targetCardID = s.worldDeck[s.activeIndex].ID
	}
	targetIndex := s.worldCardIndex(targetCardID)
	if targetIndex < 0 {
		return Snapshot{}, fmt.Errorf("target card %q is not in the world deck", targetCardID)
	}
	target := s.worldDeck[targetIndex]
	for _, rule := range s.useRules {
		if !s.ruleBaseMatches(rule, *source, target) {
			continue
		}
		if !sourceComponentConditionsMatch(rule.SourceComponentConditions, source.Document) {
			if strings.TrimSpace(rule.FailureMessage) != "" {
				s.lastMessage = rule.FailureMessage
				return s.snapshotLocked()
			}
			continue
		}
		if err := s.applyRuleEffects(rule, target); err != nil {
			return Snapshot{}, err
		}
		return s.snapshotLocked()
	}
	s.lastMessage = "Nothing on this card responds to " + source.Name + "."
	return s.snapshotLocked()
}

func (s *Session) SaveController(templateCardID string, document cardcomponent.Document) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	templateCardID = strings.TrimSpace(templateCardID)
	if templateCardID != BlankControllerCardID {
		return Snapshot{}, fmt.Errorf("templateCardId must be %q", BlankControllerCardID)
	}
	if s.libraryCard(BlankControllerCardID) == nil {
		return Snapshot{}, fmt.Errorf("blank controller must be in your library")
	}
	if s.libraryCard(SliderComponentCardID) == nil {
		return Snapshot{}, fmt.Errorf("slider component must be in your library")
	}
	sliderPart, err := validatedSliderFromDocument(document)
	if err != nil {
		return Snapshot{}, err
	}
	controller := Card{
		ID:          RegulatorControllerCardID,
		Name:        "Regulator Controller",
		Kind:        KindItem,
		Tags:        []string{"item", "controller", "slider-controller"},
		Collectible: false,
		Collected:   true,
		State: map[string]any{
			"created":  true,
			"template": BlankControllerCardID,
		},
		Document: regulatorControllerDocument(sliderPart),
	}
	for index := range s.library {
		if s.library[index].ID == RegulatorControllerCardID {
			s.library[index] = controller
			s.lastMessage = "Regulator Controller updated in your library."
			return s.snapshotLocked()
		}
	}
	s.library = append(s.library, controller)
	s.lastMessage = "Regulator Controller saved to your library."
	return s.snapshotLocked()
}

func (s *Session) snapshotLocked() (Snapshot, error) {
	if len(s.worldDeck) == 0 {
		return Snapshot{}, fmt.Errorf("world deck is empty")
	}
	if s.activeIndex < 0 || s.activeIndex >= len(s.worldDeck) {
		s.activeIndex = 0
	}
	worldDeck := cloneCards(s.worldDeck)
	library := cloneCards(s.library)
	return Snapshot{
		WorldDeck:         worldDeck,
		ActiveWorldCard:   cloneValue(s.worldDeck[s.activeIndex]),
		ActiveWorldCardID: s.worldDeck[s.activeIndex].ID,
		ActiveIndex:       s.activeIndex,
		Library:           library,
		SolvedFlags:       cloneValue(s.solvedFlags),
		Message:           s.lastMessage,
	}, nil
}

func (s *Session) worldCardIndex(cardID string) int {
	for index, card := range s.worldDeck {
		if card.ID == cardID {
			return index
		}
	}
	return -1
}

func (s *Session) libraryCard(cardID string) *Card {
	for index := range s.library {
		if s.library[index].ID == cardID {
			return &s.library[index]
		}
	}
	return nil
}

func materializeDeck(definition DeckDefinition) ([]Card, map[string]map[string]cardcomponent.Document, map[string]CardDefinition, int, error) {
	worldDeck := make([]Card, 0, len(definition.Cards))
	documentVariants := make(map[string]map[string]cardcomponent.Document, len(definition.Cards))
	cardDefinitions := make(map[string]CardDefinition, len(definition.Cards))
	activeIndex := -1
	for index, card := range definition.Cards {
		document, ok := card.Documents[card.InitialDocument]
		if !ok {
			return nil, nil, nil, 0, fmt.Errorf("card %q initial document variant %q does not exist", card.ID, card.InitialDocument)
		}
		cardDefinitions[card.ID] = cloneValue(card)
		documentVariants[card.ID] = cloneValue(card.Documents)
		worldDeck = append(worldDeck, Card{
			ID:          card.ID,
			Name:        card.Name,
			Kind:        card.Kind,
			Tags:        append([]string(nil), card.Tags...),
			Collectible: card.Collectible,
			State:       cloneValue(card.State),
			Document:    cloneValue(document),
		})
		if card.ID == definition.InitialActiveCardID {
			activeIndex = index
		}
	}
	if activeIndex < 0 {
		return nil, nil, nil, 0, fmt.Errorf("initial active card %q does not exist", definition.InitialActiveCardID)
	}
	return worldDeck, documentVariants, cardDefinitions, activeIndex, nil
}

func (s *Session) ruleBaseMatches(rule UseRuleDefinition, source Card, target Card) bool {
	if !cardMatches(source, rule.Source) || !cardMatches(target, rule.Target) {
		return false
	}
	for flag, value := range rule.FlagConditions {
		if s.solvedFlags[flag] != value {
			return false
		}
	}
	return true
}

func sourceComponentConditionsMatch(conditions []ComponentConditionDefinition, document cardcomponent.Document) bool {
	for _, condition := range conditions {
		switch strings.TrimSpace(condition.Type) {
		case slider.Type:
			part, ok := firstSliderFragment(document)
			if !ok || condition.ValueEquals == nil {
				return false
			}
			if part.Value != *condition.ValueEquals {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validatedSliderFromDocument(document cardcomponent.Document) (slider.Fragment, error) {
	part, ok := firstSliderFragment(document)
	if !ok {
		return slider.Fragment{}, fmt.Errorf("controller document must include a slider component")
	}
	return part, nil
}

func firstSliderFragment(document cardcomponent.Document) (slider.Fragment, bool) {
	var out slider.Fragment
	found := false
	var visit func(cardcomponent.Node)
	visit = func(node cardcomponent.Node) {
		if found {
			return
		}
		if node.Type == slider.Type {
			var part slider.Fragment
			if err := json.Unmarshal(node.Fragment, &part); err != nil {
				return
			}
			generated := fragment.Generated[slider.Fragment]{
				Target:      slider.Type,
				Description: "Controller slider",
				Fragment:    part,
			}
			slider.NormalizeGenerated(&generated)
			if issues := slider.ValidateGenerated(generated); len(issues) > 0 {
				return
			}
			out = generated.Fragment
			found = true
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document.Root)
	return out, found
}

func regulatorControllerDocument(part slider.Fragment) cardcomponent.Document {
	part = slider.NormalizeFragment(part)
	part.Label = "Output"
	return cardcomponent.Document{
		CardID: RegulatorControllerCardID,
		Name:   "Regulator Controller",
		Root: cardcomponent.Node{
			ID:       RegulatorControllerCardID + "-root",
			Type:     cardcomponent.Type,
			Fragment: cardcomponent.EncodeRootFragment(cardcomponent.RootFragment{PaddingPX: 18, Shadow: "0 24px 60px rgba(8,47,73,0.34)"}),
			Children: []cardcomponent.Node{
				{
					ID:   RegulatorControllerCardID + "-background",
					Type: background.Type,
					Fragment: mustRaw(background.Fragment{
						BackgroundColor: "#082f49",
						CSS:             "background: linear-gradient(160deg, #082f49 0%, #0f172a 100%);",
					}),
				},
				{
					ID:   RegulatorControllerCardID + "-border",
					Type: border.Type,
					Fragment: mustRaw(border.Fragment{
						BorderWidthPX:  2,
						BorderRadiusPX: 22,
						BorderColor:    "#7dd3fc",
						CSS:            "border: 2px solid #7dd3fc; border-radius: 22px;",
					}),
				},
				{
					ID:   RegulatorControllerCardID + "-title",
					Type: textarea.Type,
					Fragment: mustRaw(textarea.Fragment{
						Content:    "REGULATOR",
						FontFamily: "system",
						FontSizePX: 24,
						FontWeight: 800,
						FontStyle:  "normal",
						Color:      "#e0f2fe",
						Align:      "center",
						Position:   "center",
						X:          50,
						Y:          20,
						PaddingPX:  4,
						CSS:        "text-align: center;",
					}),
				},
				{
					ID:   RegulatorControllerCardID + "-hint",
					Type: textarea.Type,
					Fragment: mustRaw(textarea.Fragment{
						Content:    "Output setpoint",
						FontFamily: "system",
						FontSizePX: 14,
						FontWeight: 700,
						FontStyle:  "normal",
						Color:      "#bae6fd",
						Align:      "center",
						Position:   "center",
						X:          50,
						Y:          40,
						PaddingPX:  4,
						CSS:        "text-align: center;",
					}),
				},
				{
					ID:       "regulator-output-slider",
					Type:     slider.Type,
					Fragment: mustRaw(part),
				},
			},
		},
	}
}

func cardMatches(card Card, matcher CardMatcherDefinition) bool {
	if strings.TrimSpace(matcher.ID) != "" && card.ID != matcher.ID {
		return false
	}
	for _, tag := range matcher.Tags {
		if !hasTag(card, tag) {
			return false
		}
	}
	return true
}

func (s *Session) applyRuleEffects(rule UseRuleDefinition, target Card) error {
	for _, effect := range rule.Effects {
		switch effect.Type {
		case EffectSetFlag:
			if s.solvedFlags == nil {
				s.solvedFlags = map[string]bool{}
			}
			value, ok := effect.Value.(bool)
			if !ok {
				return fmt.Errorf("%s effect value must be a boolean", EffectSetFlag)
			}
			s.solvedFlags[effect.Flag] = value
		case EffectSetCardState:
			if err := s.updateEffectCard(effect, target, func(card *Card) error {
				if card.State == nil {
					card.State = map[string]any{}
				}
				card.State[effect.Key] = cloneValue(effect.Value)
				return nil
			}); err != nil {
				return err
			}
		case EffectRemoveCardTags:
			if err := s.updateEffectCard(effect, target, func(card *Card) error {
				for _, tag := range effect.Tags {
					card.Tags = removeString(card.Tags, tag)
				}
				return nil
			}); err != nil {
				return err
			}
		case EffectSetDocumentVariant:
			if err := s.updateEffectCard(effect, target, func(card *Card) error {
				variants := s.documentVariants[card.ID]
				document, ok := variants[effect.Variant]
				if !ok {
					return fmt.Errorf("card %q document variant %q does not exist", card.ID, effect.Variant)
				}
				card.Document = cloneValue(document)
				return nil
			}); err != nil {
				return err
			}
		case EffectSetMessage:
			s.lastMessage = effect.Message
		case EffectLoadDeck:
			if err := s.loadDeck(effect.DeckID); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported effect type %q", effect.Type)
		}
	}
	return nil
}

func (s *Session) loadDeck(deckID string) error {
	deckID = strings.TrimSpace(deckID)
	if s.loadedDecks[deckID] {
		return nil
	}
	definition, err := LoadEmbeddedDeck(deckID)
	if err != nil {
		return err
	}
	if err := ValidateDeckPackDefinition(definition, s.cardDefinitions); err != nil {
		return err
	}
	worldDeck, documentVariants, cardDefinitions, activeIndex, err := materializeDeck(definition)
	if err != nil {
		return err
	}
	if s.solvedFlags == nil {
		s.solvedFlags = map[string]bool{}
	}
	for flag, value := range definition.InitialSolvedFlags {
		if _, exists := s.solvedFlags[flag]; !exists {
			s.solvedFlags[flag] = value
		}
	}
	startIndex := len(s.worldDeck)
	s.worldDeck = append(s.worldDeck, worldDeck...)
	if s.documentVariants == nil {
		s.documentVariants = map[string]map[string]cardcomponent.Document{}
	}
	for cardID, documents := range documentVariants {
		s.documentVariants[cardID] = documents
	}
	if s.cardDefinitions == nil {
		s.cardDefinitions = map[string]CardDefinition{}
	}
	for cardID, card := range cardDefinitions {
		s.cardDefinitions[cardID] = card
	}
	s.useRules = append(s.useRules, cloneValue(definition.UseRules)...)
	if s.loadedDecks == nil {
		s.loadedDecks = map[string]bool{}
	}
	s.loadedDecks[definition.ID] = true
	s.activeIndex = startIndex + activeIndex
	return nil
}

func (s *Session) updateEffectCard(effect RuleEffectDefinition, target Card, update func(*Card) error) error {
	cardID := effect.CardID
	if strings.TrimSpace(cardID) == "" {
		cardID = target.ID
	}
	index := s.worldCardIndex(cardID)
	if index < 0 {
		return fmt.Errorf("effect %q references card %q outside world deck", effect.Type, cardID)
	}
	card := s.worldDeck[index]
	if err := update(&card); err != nil {
		return err
	}
	s.worldDeck[index] = card
	return nil
}

func hasTag(card Card, tag string) bool {
	for _, candidate := range card.Tags {
		if candidate == tag {
			return true
		}
	}
	return false
}

func removeString(values []string, value string) []string {
	out := values[:0]
	for _, candidate := range values {
		if candidate != value {
			out = append(out, candidate)
		}
	}
	return out
}

func cloneCards(cards []Card) []Card {
	if len(cards) == 0 {
		return nil
	}
	out := make([]Card, len(cards))
	for index, card := range cards {
		out[index] = cloneValue(card)
	}
	return out
}

func mustRaw(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func cloneValue[T any](value T) T {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}
