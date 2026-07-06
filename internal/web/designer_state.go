package web

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/design"
)

type designerState struct {
	mu          sync.Mutex
	document    cardcomponent.Document
	gameState   GameState
	library     []cardcomponent.LibraryItem
	lastApplied *cardcomponent.LibraryItem
}

func newDesignerState() *designerState {
	return &designerState{
		document:  cardcomponent.DefaultDocument(),
		gameState: initialGameState(),
		library:   seededLibrary(),
	}
}

func seededLibrary() []cardcomponent.LibraryItem {
	items := append([]cardcomponent.LibraryItem{}, background.Presets()...)
	items = append(items, border.Presets()...)
	items = append(items, textarea.Presets()...)
	return cloneLibrary(items)
}

func (s *designerState) snapshot() (cardcomponent.Document, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return cloneValue(s.document), cloneLibrary(s.library)
}

func (s *designerState) interactiveSnapshot() (cardcomponent.Document, GameState, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gameState = normalizeGameState(s.gameState)
	s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
	s.gameState = syncGameStateWithDocument(s.gameState, s.document)
	return cloneValue(s.document), cloneValue(s.gameState), cloneLibrary(s.library)
}

func (s *designerState) reset() (cardcomponent.Document, []cardcomponent.LibraryItem) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.document = cardcomponent.DefaultDocument()
	s.gameState = initialGameState()
	s.lastApplied = nil
	return cloneValue(s.document), cloneLibrary(s.library)
}

func (s *designerState) apply(raw json.RawMessage, componentID string) (cardcomponent.Document, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	document := cloneValue(s.document)
	normalized, item, err := applyGeneratedConfigToDocumentForComponent(raw, &document, componentID)
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	s.document = document
	s.gameState = syncGameStateWithDocument(s.gameState, s.document)
	s.lastApplied = &item
	return cloneValue(s.document), normalized, nil
}

func (s *designerState) tap(target, zone string, x, y float64) (tapResult, error) {
	componentID, trait := canonicalTapComponent(target, zone)
	return s.interact(componentID, trait, interactionShortTap, x, y)
}

func (s *designerState) applyColorControl(target string, request colorControlRequest) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gameState = normalizeGameState(s.gameState)
	target = canonicalTapTarget(target, target)
	if !isKnownTapTarget(target) {
		return tapResult{}, fmt.Errorf("target %q is not available", target)
	}
	componentID, trait := canonicalTapComponent(target, target)
	if !componentUnlocked(s.gameState, componentID) || !modeUnlocked(s.gameState, target, editModeSimpleControls) {
		return tapResult{
			document:  cloneValue(s.document),
			gameState: cloneValue(s.gameState),
			library:   cloneLibrary(s.library),
			events: []CardEvent{{
				Type:          "invalidAction",
				ComponentKind: target,
				Message:       "Color controls unlock at level 5.",
			}},
		}, nil
	}

	raw, err := colorGeneratedConfig(s.document, target, request)
	if err != nil {
		return tapResult{}, err
	}
	before := documentSignature(s.document)
	document := cloneValue(s.document)
	normalized, item, err := applyGeneratedConfigToDocument(raw, &document)
	if err != nil {
		return tapResult{}, err
	}
	s.document = document
	s.lastApplied = &item
	var events []CardEvent
	if documentSignature(s.document) != before {
		var xpEvents []CardEvent
		s.gameState, xpEvents = advanceInteraction(s.gameState, componentID, xpPerInteraction)
		events = append(events, CardEvent{Type: "configApplied", ComponentKind: target, ComponentID: componentID, Trait: trait})
		events = append(events, xpEvents...)
	}

	return tapResult{
		document:      cloneValue(s.document),
		gameState:     cloneValue(s.gameState),
		appliedConfig: normalized,
		library:       cloneLibrary(s.library),
		events:        events,
		overlay:       buildOverlay(s.document, s.gameState, componentID),
	}, nil
}

func (s *designerState) interact(componentID, trait, interaction string, x, y float64) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = x
	_ = y
	s.gameState = normalizeGameState(s.gameState)
	s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
	s.gameState = syncGameStateWithDocument(s.gameState, s.document)
	componentID = strings.TrimSpace(componentID)
	trait = strings.TrimSpace(trait)
	if componentID == "" {
		componentID = componentCardRoot
	}
	if !componentExistsInDocument(s.document, componentID) {
		return tapResult{}, fmt.Errorf("component %q is not available", componentID)
	}
	if !componentUnlocked(s.gameState, componentID) {
		return s.invalidComponentResult(componentID, componentID+" is locked."), nil
	}
	progress := s.gameState.ComponentProgress[componentID]
	if !traitUnlocked(progress, trait) {
		return s.invalidComponentResult(componentID, trait+" is locked."), nil
	}
	switch strings.TrimSpace(interaction) {
	case interactionLongPress:
		return s.longPressComponent(componentID)
	case "", interactionShortTap:
		return s.shortTapComponent(componentID, trait)
	default:
		return tapResult{}, fmt.Errorf("interaction %q is not supported", interaction)
	}
}

func (s *designerState) shortTapComponent(componentID, trait string) (tapResult, error) {
	progress := s.gameState.ComponentProgress[componentID]
	previousSelected := s.gameState.SelectedComponentID
	s.gameState.SelectedComponentID = componentID
	if !progress.RandomTapEnabled {
		events := []CardEvent{{
			Type:        "componentSelected",
			ComponentID: componentID,
		}}
		if previousSelected == componentID {
			events = nil
		}
		return tapResult{
			document:  cloneValue(s.document),
			gameState: cloneValue(normalizeGameState(s.gameState)),
			library:   cloneLibrary(s.library),
			events:    events,
			overlay:   buildOverlay(s.document, s.gameState, componentID),
		}, nil
	}
	return s.randomizeComponentLocked(componentID, trait, "unlockedTraits")
}

func (s *designerState) longPressComponent(componentID string) (tapResult, error) {
	progress := s.gameState.ComponentProgress[componentID]
	s.gameState.SelectedComponentID = componentID
	var events []CardEvent
	var xpEvents []CardEvent
	s.gameState, xpEvents = advanceInteraction(s.gameState, componentID, xpPerInteraction)
	events = append(events, xpEvents...)
	progress = s.gameState.ComponentProgress[componentID]
	if !progress.OverlayUnlocked {
		events = append(events, CardEvent{
			Type:        "invalidAction",
			ComponentID: componentID,
			Message:     "Overlay unlocks at component level 3.",
		})
		return tapResult{
			document:  cloneValue(s.document),
			gameState: cloneValue(s.gameState),
			library:   cloneLibrary(s.library),
			events:    events,
		}, nil
	}
	if !progress.OverlayOpened {
		progress.OverlayOpened = true
		s.gameState.ComponentProgress[componentID] = progress
		s.gameState, xpEvents = advanceInteraction(s.gameState, componentID, xpPerInteraction)
		progress = s.gameState.ComponentProgress[componentID]
		progress.OverlayOpened = true
		s.gameState.ComponentProgress[componentID] = progress
		s.gameState = normalizeGameState(s.gameState)
		events = append(events, xpEvents...)
	}
	events = append(events, CardEvent{
		Type:        "overlayOpened",
		ComponentID: componentID,
	})
	return tapResult{
		document:  cloneValue(s.document),
		gameState: cloneValue(s.gameState),
		library:   cloneLibrary(s.library),
		events:    events,
		overlay:   buildOverlay(s.document, s.gameState, componentID),
	}, nil
}

func (s *designerState) randomizeComponent(componentID, trait, scope string) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gameState = normalizeGameState(s.gameState)
	s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
	s.gameState = syncGameStateWithDocument(s.gameState, s.document)
	if !componentExistsInDocument(s.document, componentID) {
		return tapResult{}, fmt.Errorf("component %q is not available", componentID)
	}
	if !componentUnlocked(s.gameState, componentID) {
		return s.invalidComponentResult(componentID, componentID+" is locked."), nil
	}
	progress := s.gameState.ComponentProgress[componentID]
	if !traitUnlocked(progress, trait) {
		return s.invalidComponentResult(componentID, trait+" is locked."), nil
	}
	return s.randomizeComponentLocked(componentID, trait, scope)
}

func (s *designerState) randomizeComponentLocked(componentID, trait, scope string) (tapResult, error) {
	s.gameState.SelectedComponentID = componentID
	normalized, item, eventComponentKind, appliedTrait, err := s.applyRandomMutation(componentID, trait, scope)
	if err != nil {
		return tapResult{}, err
	}
	var events []CardEvent
	if item.ID != "" {
		s.lastApplied = &item
	}
	events = append(events, CardEvent{
		Type:          "configApplied",
		ComponentKind: eventComponentKind,
		ComponentID:   componentID,
		Trait:         appliedTrait,
	})
	var xpEvents []CardEvent
	s.gameState, xpEvents = advanceInteraction(s.gameState, componentID, xpPerInteraction)
	s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
	events = append(events, xpEvents...)
	return tapResult{
		document:      cloneValue(s.document),
		gameState:     cloneValue(s.gameState),
		appliedConfig: normalized,
		library:       cloneLibrary(s.library),
		events:        events,
		overlay:       buildOverlay(s.document, s.gameState, componentID),
	}, nil
}

func (s *designerState) applyControlChange(componentID, trait, control string, value json.RawMessage) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.gameState = normalizeGameState(s.gameState)
	s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
	s.gameState = syncGameStateWithDocument(s.gameState, s.document)
	componentID = strings.TrimSpace(componentID)
	trait = strings.TrimSpace(trait)
	control = strings.TrimSpace(control)
	if componentID == "" {
		componentID = componentCardRoot
	}
	if !componentExistsInDocument(s.document, componentID) {
		return tapResult{}, fmt.Errorf("component %q is not available", componentID)
	}
	if !componentUnlocked(s.gameState, componentID) {
		return s.invalidComponentResult(componentID, componentID+" is locked."), nil
	}
	progress := s.gameState.ComponentProgress[componentID]
	if !controlUnlocked(progress, control) {
		return s.invalidComponentResult(componentID, "Control is locked."), nil
	}
	if control == "preventRandomizing" {
		next, err := readBoolValue(value)
		if err != nil {
			return tapResult{}, err
		}
		events := []CardEvent{}
		if progress.PreventRandomizing != next {
			progress.PreventRandomizing = next
			s.gameState.ComponentProgress[componentID] = progress
			var xpEvents []CardEvent
			s.gameState, xpEvents = advanceInteraction(s.gameState, componentID, xpPerInteraction)
			progress = s.gameState.ComponentProgress[componentID]
			progress.PreventRandomizing = next
			s.gameState.ComponentProgress[componentID] = progress
			s.gameState = normalizeGameState(s.gameState)
			events = append(events, CardEvent{
				Type:        "controlChanged",
				ComponentID: componentID,
				Control:     control,
			})
			events = append(events, xpEvents...)
		}
		return tapResult{
			document:  cloneValue(s.document),
			gameState: cloneValue(s.gameState),
			library:   cloneLibrary(s.library),
			events:    events,
			overlay:   buildOverlay(s.document, s.gameState, componentID),
		}, nil
	}
	before := documentSignature(s.document)
	normalized, item, target, err := s.applyControlMutation(componentID, trait, control, value)
	if err != nil {
		return tapResult{}, err
	}
	var events []CardEvent
	if documentSignature(s.document) != before {
		if item.ID != "" {
			s.lastApplied = &item
		}
		events = append(events, CardEvent{
			Type:          "configApplied",
			ComponentKind: target,
			ComponentID:   componentID,
			Trait:         trait,
			Control:       control,
		})
		var xpEvents []CardEvent
		s.gameState, xpEvents = advanceInteraction(s.gameState, componentID, xpPerInteraction)
		s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
		events = append(events, xpEvents...)
	}
	return tapResult{
		document:      cloneValue(s.document),
		gameState:     cloneValue(s.gameState),
		appliedConfig: normalized,
		library:       cloneLibrary(s.library),
		events:        events,
		overlay:       buildOverlay(s.document, s.gameState, componentID),
	}, nil
}

func (s *designerState) addComponent(componentKind string, rawConfig json.RawMessage) (tapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	componentKind = strings.TrimSpace(componentKind)
	if componentKind != componentKindTextarea && componentKind != componentKindShape && componentKind != componentKindImage {
		return tapResult{}, fmt.Errorf("componentKind must be textarea, shape, or image")
	}
	configRaw, err := defaultOrValidatedComponentConfig(componentKind, rawConfig)
	if err != nil {
		return tapResult{}, err
	}
	s.gameState = normalizeGameState(s.gameState)
	s.document = ensureUnlockedDocumentComponents(s.document, s.gameState)
	componentID := nextComponentID(s.document, componentKind)
	s.document.Root.Children = append(s.document.Root.Children, cardcomponent.Node{
		ID:            componentID,
		ComponentKind: componentKind,
		Config:        configRaw,
	})
	s.gameState = syncGameStateWithDocument(s.gameState, s.document)
	s.gameState.SelectedComponentID = componentID
	return tapResult{
		document:  cloneValue(s.document),
		gameState: cloneValue(s.gameState),
		library:   cloneLibrary(s.library),
		events: []CardEvent{{
			Type:          "componentAdded",
			ComponentID:   componentID,
			ComponentKind: componentKind,
			Message:       componentLabel(componentKind) + " added",
		}},
		overlay: buildOverlay(s.document, s.gameState, componentID),
	}, nil
}

func (s *designerState) applyRandomMutation(componentID, trait, scope string) (any, cardcomponent.LibraryItem, string, string, error) {
	progress := s.gameState.ComponentProgress[componentID]
	seed := tapSeed(s.gameState, componentID)
	switch progress.ComponentKind {
	case componentKindCard:
		appliedTrait := chooseRandomTrait(progress, trait, scope, seed, []string{traitBackground, traitBorder, traitShadow, traitPadding})
		switch appliedTrait {
		case traitBackground:
			return s.applyRandomConfig("", background.Kind, appliedTrait, seed, maxInt(s.gameState.GlobalLevel, progress.Level))
		case traitBorder:
			return s.applyRandomConfig("", border.Kind, appliedTrait, seed, maxInt(s.gameState.GlobalLevel, progress.Level))
		case traitShadow:
			root := cardcomponent.DecodeRootConfig(s.document.Root.Config)
			root.Shadow = pickString(seed, root.Shadow, []string{
				"0 18px 48px rgba(15,23,42,0.28)",
				"0 28px 70px rgba(15,23,42,0.42)",
				"0 0 34px rgba(52,211,153,0.28)",
				"",
			})
			s.document.Root.Config = cardcomponent.EncodeRootConfig(root)
			return generatedRootTrait("Card shadow randomized", appliedTrait, root), cardcomponent.LibraryItem{}, cardcomponent.Kind, appliedTrait, nil
		case traitPadding:
			root := cardcomponent.DecodeRootConfig(s.document.Root.Config)
			root.PaddingPX = pickInt(seed, root.PaddingPX, []int{8, 16, 20, 24, 32, 40})
			s.document.Root.Config = cardcomponent.EncodeRootConfig(root)
			return generatedRootTrait("Card padding randomized", appliedTrait, root), cardcomponent.LibraryItem{}, cardcomponent.Kind, appliedTrait, nil
		default:
			return nil, cardcomponent.LibraryItem{}, "", "", fmt.Errorf("trait %q cannot be randomized", appliedTrait)
		}
	case componentKindTextarea:
		return s.applyRandomConfig(componentID, textarea.Kind, firstNonEmpty(trait, traitText), seed, maxInt(s.gameState.GlobalLevel, progress.Level))
	case componentKindShape:
		return s.applyRandomConfig(componentID, shape.Kind, firstNonEmpty(trait, traitGeometry), seed, maxInt(s.gameState.GlobalLevel, progress.Level))
	case componentKindImage:
		return s.applyRandomConfig(componentID, imagecomponent.Kind, firstNonEmpty(trait, traitImage), seed, maxInt(s.gameState.GlobalLevel, progress.Level))
	default:
		return nil, cardcomponent.LibraryItem{}, "", "", fmt.Errorf("component %q cannot be randomized", componentID)
	}
}

func (s *designerState) applyRandomConfig(componentID, target, appliedTrait string, seed int64, level int) (any, cardcomponent.LibraryItem, string, string, error) {
	raw, err := randomGeneratedConfig(target, seed, level)
	if err != nil {
		return nil, cardcomponent.LibraryItem{}, "", "", err
	}
	document := cloneValue(s.document)
	normalized, item, err := applyGeneratedConfigToDocumentForComponent(raw, &document, componentID)
	if err != nil {
		return nil, cardcomponent.LibraryItem{}, "", "", err
	}
	s.document = document
	return normalized, item, target, appliedTrait, nil
}

func (s *designerState) applyControlMutation(componentID, trait, control string, value json.RawMessage) (any, cardcomponent.LibraryItem, string, error) {
	progress := s.gameState.ComponentProgress[componentID]
	switch progress.ComponentKind {
	case componentKindCard:
		return s.applyCardControl(trait, control, value)
	case componentKindTextarea:
		return s.applyTextareaControl(componentID, trait, control, value)
	case componentKindShape:
		return s.applyShapeControl(componentID, trait, control, value)
	case componentKindImage:
		return s.applyImageControl(componentID, trait, control, value)
	default:
		return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("component %q does not support controls", componentID)
	}
}

func (s *designerState) applyCardControl(trait, control string, value json.RawMessage) (any, cardcomponent.LibraryItem, string, error) {
	switch control {
	case "backgroundColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		raw, err := colorGeneratedConfig(s.document, background.Kind, colorControlRequest{Color: color})
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		return s.applyRawConfig(raw)
	case "borderColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		raw, err := colorGeneratedConfig(s.document, border.Kind, colorControlRequest{Color: color})
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		return s.applyRawConfig(raw)
	case "borderWidthPx", "borderRadiusPx":
		part := currentBorder(s.document)
		next, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		if control == "borderWidthPx" {
			part.BorderWidthPX = next
		} else {
			part.BorderRadiusPX = next
		}
		part.CSS = fmt.Sprintf("border: %dpx solid %s; border-radius: %dpx;", part.BorderWidthPX, part.BorderColor, part.BorderRadiusPX)
		return s.applyGeneratedConfig(border.Kind, "Border control changed", part)
	case "shadowPreset":
		shadow, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		if !controlOptionValueAllowed(shadowOptions(), shadow) {
			return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("shadow preset is not allowed")
		}
		root := cardcomponent.DecodeRootConfig(s.document.Root.Config)
		root.Shadow = shadow
		s.document.Root.Config = cardcomponent.EncodeRootConfig(root)
		return generatedRootTrait("Card shadow changed", traitShadow, root), cardcomponent.LibraryItem{}, cardcomponent.Kind, nil
	case "paddingPx":
		padding, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		root := cardcomponent.DecodeRootConfig(s.document.Root.Config)
		root.PaddingPX = padding
		s.document.Root.Config = cardcomponent.EncodeRootConfig(root)
		return generatedRootTrait("Card padding changed", traitPadding, root), cardcomponent.LibraryItem{}, cardcomponent.Kind, nil
	default:
		return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("control %q is not supported for card", control)
	}
}

func (s *designerState) applyTextareaControl(componentID, trait, control string, value json.RawMessage) (any, cardcomponent.LibraryItem, string, error) {
	part := currentTextarea(s.document, componentID)
	switch control {
	case "content":
		text, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Content = text
	case "backgroundColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BackgroundColor = color
	case "borderColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderColor = color
	case "textColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Color = color
	case "fontFamily":
		font, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.FontFamily = font
	case "fontSizePx":
		size, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.FontSizePX = size
	case "fontWeight":
		weight, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.FontWeight = weight
	case "paddingPx":
		padding, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.PaddingPX = padding
	case "x":
		x, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.X = x
	case "y":
		y, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Y = y
	case "position":
		position, err := readPositionValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.X = position.X
		part.Y = position.Y
	case "borderWidthPx":
		width, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderWidthPX = width
	case "borderRadiusPx":
		radius, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderRadiusPX = radius
	default:
		return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("control %q is not supported for text", control)
	}
	return s.applyGeneratedConfigForComponent(componentID, textarea.Kind, "Text control changed", part)
}

func (s *designerState) applyShapeControl(componentID, trait, control string, value json.RawMessage) (any, cardcomponent.LibraryItem, string, error) {
	part := currentShape(s.document, componentID)
	switch control {
	case "shape":
		shapeValue, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Shape = shapeValue
	case "backgroundColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BackgroundColor = color
	case "borderColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderColor = color
	case "borderWidthPx":
		width, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderWidthPX = width
	case "width":
		width, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Width = width
	case "height":
		height, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Height = height
	case "x":
		x, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.X = x
	case "y":
		y, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Y = y
	case "position":
		position, err := readPositionValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.X = position.X
		part.Y = position.Y
	case "rotation":
		rotation, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Rotation = rotation
	case "shadowPreset":
		shadowValue, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Shadow = shadowValue
	default:
		return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("control %q is not supported for shape", control)
	}
	return s.applyGeneratedConfigForComponent(componentID, shape.Kind, "Shape control changed", part)
}

func (s *designerState) applyImageControl(componentID, trait, control string, value json.RawMessage) (any, cardcomponent.LibraryItem, string, error) {
	part := currentImage(s.document, componentID)
	switch control {
	case "src":
		src, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Src = src
	case "alt":
		alt, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Alt = alt
	case "x":
		x, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.X = x
	case "y":
		y, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Y = y
	case "position":
		position, err := readPositionValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.X = position.X
		part.Y = position.Y
	case "width":
		width, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Width = width
	case "height":
		height, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Height = height
	case "rotation":
		rotation, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.Rotation = rotation
	case "borderColor":
		color, err := readStringValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderColor = color
	case "borderWidthPx":
		width, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderWidthPX = width
	case "borderRadiusPx":
		radius, err := readIntValue(value)
		if err != nil {
			return nil, cardcomponent.LibraryItem{}, "", err
		}
		part.BorderRadiusPX = radius
	default:
		return nil, cardcomponent.LibraryItem{}, "", fmt.Errorf("control %q is not supported for image", control)
	}
	return s.applyGeneratedConfigForComponent(componentID, imagecomponent.Kind, "Image control changed", part)
}

func (s *designerState) applyGeneratedConfig(target, description string, part any) (any, cardcomponent.LibraryItem, string, error) {
	return s.applyGeneratedConfigForComponent("", target, description, part)
}

func (s *designerState) applyGeneratedConfigForComponent(componentID, target, description string, part any) (any, cardcomponent.LibraryItem, string, error) {
	raw, err := json.Marshal(struct {
		ComponentKind string `json:"componentKind"`
		Description   string `json:"description"`
		Config        any    `json:"config"`
	}{
		ComponentKind: target,
		Description:   description,
		Config:        part,
	})
	if err != nil {
		return nil, cardcomponent.LibraryItem{}, "", err
	}
	return s.applyRawConfigForComponent(raw, componentID)
}

func (s *designerState) applyRawConfig(raw json.RawMessage) (any, cardcomponent.LibraryItem, string, error) {
	return s.applyRawConfigForComponent(raw, "")
}

func (s *designerState) applyRawConfigForComponent(raw json.RawMessage, componentID string) (any, cardcomponent.LibraryItem, string, error) {
	var envelope struct {
		ComponentKind string `json:"componentKind"`
	}
	_ = json.Unmarshal(raw, &envelope)
	document := cloneValue(s.document)
	normalized, item, err := applyGeneratedConfigToDocumentForComponent(raw, &document, componentID)
	if err != nil {
		return nil, cardcomponent.LibraryItem{}, "", err
	}
	s.document = document
	return normalized, item, strings.TrimSpace(envelope.ComponentKind), nil
}

func (s *designerState) invalidComponentResult(componentID, message string) tapResult {
	return tapResult{
		document:  cloneValue(s.document),
		gameState: cloneValue(s.gameState),
		library:   cloneLibrary(s.library),
		events: []CardEvent{{
			Type:          "invalidAction",
			ComponentKind: fallbackConfigKindForComponent(componentID),
			ComponentID:   componentID,
			Message:       message,
		}},
	}
}

func (s *designerState) applyLibraryItem(id string) (cardcomponent.Document, any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var item cardcomponent.LibraryItem
	for _, candidate := range s.library {
		if candidate.ID == id {
			item = cloneValue(candidate)
			break
		}
	}
	if item.ID == "" {
		return cardcomponent.Document{}, nil, fmt.Errorf("library item %q was not found", id)
	}
	raw, err := generatedRawFromLibraryItem(item)
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	document := cloneValue(s.document)
	normalized, applied, err := applyGeneratedConfigToDocument(raw, &document)
	if err != nil {
		return cardcomponent.Document{}, nil, err
	}
	s.document = document
	s.lastApplied = &applied
	return cloneValue(s.document), normalized, nil
}

func (s *designerState) saveLastApplied() (cardcomponent.LibraryItem, []cardcomponent.LibraryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastApplied == nil {
		return cardcomponent.LibraryItem{}, nil, fmt.Errorf("no applied config is available to save")
	}
	item := cloneValue(*s.lastApplied)
	item.Saved = true
	if strings.TrimSpace(item.ID) == "" || strings.HasPrefix(item.ID, "applied-") {
		item.ID = "saved-" + item.ComponentKind + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if strings.TrimSpace(item.Name) == "" {
		item.Name = item.Description
	}
	for _, candidate := range s.library {
		if candidate.ComponentKind == item.ComponentKind && string(candidate.Config) == string(item.Config) {
			return cloneValue(candidate), cloneLibrary(s.library), nil
		}
	}
	s.library = append([]cardcomponent.LibraryItem{item}, s.library...)
	return cloneValue(item), cloneLibrary(s.library), nil
}

func (s *designerState) libraryForComponentKind(target string) []cardcomponent.LibraryItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	target = strings.TrimSpace(target)
	if target == "" {
		return cloneLibrary(s.library)
	}
	var out []cardcomponent.LibraryItem
	for _, item := range s.library {
		if item.ComponentKind == target {
			out = append(out, cloneValue(item))
		}
	}
	return out
}

func (s *designerState) currentConfig(target string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node := findNodeByKind(s.document.Root, target)
	if node == nil || len(node.Config) == 0 {
		return "", ""
	}
	raw, err := json.MarshalIndent(struct {
		ComponentKind string          `json:"componentKind"`
		Description   string          `json:"description"`
		Config        json.RawMessage `json:"config"`
	}{
		ComponentKind: target,
		Description:   "Current applied config",
		Config:        node.Config,
	}, "", "  ")
	if err != nil {
		return "", node.ID
	}
	return string(raw), node.ID
}

func generatedRawFromLibraryItem(item cardcomponent.LibraryItem) (json.RawMessage, error) {
	raw, err := json.Marshal(struct {
		ComponentKind string          `json:"componentKind"`
		Description   string          `json:"description"`
		Config        json.RawMessage `json:"config"`
	}{
		ComponentKind: item.ComponentKind,
		Description:   item.Description,
		Config:        item.Config,
	})
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func defaultOrValidatedComponentConfig(componentKind string, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		switch componentKind {
		case componentKindTextarea:
			return json.Marshal(textarea.DefaultConfig())
		case componentKindShape:
			return json.Marshal(shape.DefaultConfig())
		case componentKindImage:
			return json.Marshal(imagecomponent.DefaultConfig())
		default:
			return nil, fmt.Errorf("component type %q is not supported", componentKind)
		}
	}
	switch componentKind {
	case componentKindTextarea:
		var part textarea.Config
		if err := json.Unmarshal(raw, &part); err != nil {
			return nil, fmt.Errorf("invalid textarea config")
		}
		generated := design.GeneratedConfig[textarea.Config]{ComponentKind: textarea.Kind, Description: "Added textarea", Config: part}
		textarea.NormalizeGenerated(&generated)
		if issues := textarea.ValidateGenerated(generated); len(issues) > 0 {
			return nil, fmt.Errorf("invalid textarea config at %s: %s", issues[0].Path, issues[0].Message)
		}
		return json.Marshal(generated.Config)
	case componentKindShape:
		var part shape.Config
		if err := json.Unmarshal(raw, &part); err != nil {
			return nil, fmt.Errorf("invalid shape config")
		}
		generated := design.GeneratedConfig[shape.Config]{ComponentKind: shape.Kind, Description: "Added shape", Config: part}
		shape.NormalizeGenerated(&generated)
		if issues := shape.ValidateGenerated(generated); len(issues) > 0 {
			return nil, fmt.Errorf("invalid shape config at %s: %s", issues[0].Path, issues[0].Message)
		}
		return json.Marshal(generated.Config)
	case componentKindImage:
		var part imagecomponent.Config
		if err := json.Unmarshal(raw, &part); err != nil {
			return nil, fmt.Errorf("invalid image config")
		}
		generated := design.GeneratedConfig[imagecomponent.Config]{ComponentKind: imagecomponent.Kind, Description: "Added image", Config: part}
		imagecomponent.NormalizeGenerated(&generated)
		if issues := imagecomponent.ValidateGenerated(generated); len(issues) > 0 {
			return nil, fmt.Errorf("invalid image config at %s: %s", issues[0].Path, issues[0].Message)
		}
		return json.Marshal(generated.Config)
	default:
		return nil, fmt.Errorf("component type %q is not supported", componentKind)
	}
}

func nextComponentID(document cardcomponent.Document, componentKind string) string {
	prefix := componentKind
	switch componentKind {
	case componentKindTextarea:
		prefix = "textarea"
	case componentKindShape:
		prefix = "shape"
	case componentKindImage:
		prefix = "image"
	}
	for index := 1; ; index++ {
		id := fmt.Sprintf("%s-%d", prefix, index)
		if componentKind == componentKindTextarea && index == 1 {
			id = componentTextarea
		}
		if componentKind == componentKindShape && index == 1 {
			id = componentShape
		}
		if findNodeByID(document.Root, id) == nil {
			return id
		}
	}
}

func cloneLibrary(items []cardcomponent.LibraryItem) []cardcomponent.LibraryItem {
	out := make([]cardcomponent.LibraryItem, len(items))
	for index, item := range items {
		out[index] = cloneValue(item)
	}
	return out
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

func findNodeByKind(node cardcomponent.Node, target string) *cardcomponent.Node {
	if node.ComponentKind == target {
		copyNode := cloneValue(node)
		return &copyNode
	}
	for index := range node.Children {
		if match := findNodeByKind(node.Children[index], target); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByID(node cardcomponent.Node, id string) *cardcomponent.Node {
	if node.ID == id {
		copyNode := cloneValue(node)
		return &copyNode
	}
	for index := range node.Children {
		if match := findNodeByID(node.Children[index], id); match != nil {
			return match
		}
	}
	return nil
}

func findNodeByIDPtr(node *cardcomponent.Node, id string) *cardcomponent.Node {
	if node == nil {
		return nil
	}
	if node.ID == id {
		return node
	}
	for index := range node.Children {
		if match := findNodeByIDPtr(&node.Children[index], id); match != nil {
			return match
		}
	}
	return nil
}

func componentExistsInDocument(document cardcomponent.Document, componentID string) bool {
	componentID = strings.TrimSpace(componentID)
	if componentID == "" {
		return false
	}
	return findNodeByID(document.Root, componentID) != nil
}

func syncGameStateWithDocument(state GameState, document cardcomponent.Document) GameState {
	state = normalizeGameState(state)
	var visit func(cardcomponent.Node)
	visit = func(node cardcomponent.Node) {
		switch node.ComponentKind {
		case componentKindCard, componentKindTextarea, componentKindShape, componentKindImage:
			progress := state.ComponentProgress[node.ID]
			progress.ComponentID = node.ID
			progress.ComponentKind = node.ComponentKind
			state.ComponentProgress[node.ID] = progress
			if node.ComponentKind == componentKindImage {
				state.UnlockedComponentKinds = appendStringOnce(state.UnlockedComponentKinds, componentKindImage)
			}
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(document.Root)
	return normalizeGameState(state)
}

func ensureUnlockedDocumentComponents(document cardcomponent.Document, state GameState) cardcomponent.Document {
	if len(document.Root.Config) == 0 {
		document.Root.Config = cardcomponent.EncodeRootConfig(cardcomponent.DefaultRootConfig())
	}
	if componentKindUnlocked(state, componentKindShape) && findNodeByID(document.Root, componentShape) == nil {
		part := shape.DefaultConfig()
		raw, err := json.Marshal(part)
		if err != nil {
			panic(err)
		}
		document.Root.Children = append(document.Root.Children, cardcomponent.Node{
			ID:            componentShape,
			ComponentKind: shape.Kind,
			Config:        raw,
		})
	}
	return document
}

func generatedRootTrait(description, trait string, part cardcomponent.RootConfig) map[string]any {
	return map[string]any{
		"componentKind": cardcomponent.Kind,
		"description":   description,
		"trait":         trait,
		"config":        part,
	}
}

func documentSignature(document cardcomponent.Document) string {
	raw, err := json.Marshal(document)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func chooseRandomTrait(progress ComponentProgress, trait, scope string, seed int64, candidates []string) string {
	trait = strings.TrimSpace(trait)
	if trait != "" {
		return trait
	}
	var available []string
	for _, candidate := range candidates {
		if traitUnlocked(progress, candidate) {
			available = append(available, candidate)
		}
	}
	if len(available) == 0 {
		return ""
	}
	if scope == "trait" && trait != "" {
		return trait
	}
	return available[rand.New(rand.NewSource(seed)).Intn(len(available))]
}

func pickString(seed int64, current string, values []string) string {
	var candidates []string
	for _, value := range values {
		if value != current {
			candidates = append(candidates, value)
		}
	}
	if len(candidates) == 0 {
		candidates = values
	}
	return candidates[rand.New(rand.NewSource(seed)).Intn(len(candidates))]
}

func pickInt(seed int64, current int, values []int) int {
	var candidates []int
	for _, value := range values {
		if value != current {
			candidates = append(candidates, value)
		}
	}
	if len(candidates) == 0 {
		candidates = values
	}
	return candidates[rand.New(rand.NewSource(seed)).Intn(len(candidates))]
}

func firstNonEmpty(left, right string) string {
	if strings.TrimSpace(left) != "" {
		return strings.TrimSpace(left)
	}
	return right
}

func readStringValue(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", fmt.Errorf("value is required")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text), nil
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return strconv.Itoa(int(number)), nil
	}
	return "", fmt.Errorf("value must be a string")
}

func readIntValue(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("value is required")
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}
	var floatNumber float64
	if err := json.Unmarshal(raw, &floatNumber); err == nil {
		return int(floatNumber), nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		value, parseErr := strconv.Atoi(strings.TrimSpace(text))
		if parseErr != nil {
			return 0, fmt.Errorf("value must be a number")
		}
		return value, nil
	}
	return 0, fmt.Errorf("value must be a number")
}

type positionValue struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func readPositionValue(raw json.RawMessage) (positionValue, error) {
	if len(raw) == 0 {
		return positionValue{}, fmt.Errorf("value is required")
	}
	var value positionValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return positionValue{}, fmt.Errorf("value must include x and y")
	}
	return value, nil
}

func readBoolValue(raw json.RawMessage) (bool, error) {
	if len(raw) == 0 {
		return false, fmt.Errorf("value is required")
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return false, fmt.Errorf("value must be a boolean")
		}
	}
	return false, fmt.Errorf("value must be a boolean")
}

func controlOptionValueAllowed(options []ControlOption, value string) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func fallbackConfigKindForComponent(componentID string) string {
	switch componentID {
	case componentTextarea:
		return textarea.Kind
	case componentShape:
		return shape.Kind
	case componentCardRoot:
		return cardcomponent.Kind
	default:
		return ""
	}
}
