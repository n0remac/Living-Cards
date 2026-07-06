package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	imagecomponent "github.com/n0remac/Living-Card/internal/components/image"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/design"
)

const (
	componentKindCard     = cardcomponent.Kind
	componentKindTextarea = cardcomponent.KindTextarea
	componentKindShape    = cardcomponent.KindShape
	componentKindImage    = cardcomponent.KindImage

	componentCardRoot = cardcomponent.DefaultRootID
	componentTextarea = cardcomponent.DefaultTextareaID
	componentShape    = cardcomponent.DefaultShapeID

	traitBackground = "background"
	traitBorder     = "border"
	traitShadow     = "shadow"
	traitPadding    = "padding"
	traitText       = "text"
	traitImage      = "image"
	traitTypography = "typography"
	traitFill       = "fill"
	traitGeometry   = "geometry"
	traitPosition   = "position"
	traitSize       = "size"

	interactionShortTap  = "shortTap"
	interactionLongPress = "longPress"

	editModeRandom         = "random"
	editModeSimpleControls = "simpleControls"
	xpPerInteraction       = 1
	xpPerLevel             = 5
	componentXPPerLevel    = 3
	overlayUnlockLevel     = 3
)

type GameState struct {
	TotalXP                int                          `json:"totalXp"`
	GlobalLevel            int                          `json:"globalLevel"`
	TotalInteractions      int                          `json:"totalInteractions"`
	UnlockedComponentKinds []string                     `json:"unlockedComponentKinds"`
	SelectedComponentID    string                       `json:"selectedComponentId,omitempty"`
	ComponentProgress      map[string]ComponentProgress `json:"componentProgress"`

	TapCount              int                              `json:"tapCount"`
	Level                 int                              `json:"level"`
	XP                    int                              `json:"xp"`
	UnlockedConfigKinds   []string                         `json:"unlockedConfigKinds"`
	UnlockedModes         []string                         `json:"unlockedModes"`
	ComponentKindProgress map[string]ComponentKindProgress `json:"componentKindProgress"`
}

type ComponentProgress struct {
	ComponentID        string   `json:"componentId"`
	ComponentKind      string   `json:"componentKind"`
	XP                 int      `json:"xp"`
	Level              int      `json:"level"`
	Interactions       int      `json:"interactions"`
	RandomTapEnabled   bool     `json:"randomTapEnabled"`
	PreventRandomizing bool     `json:"preventRandomizing"`
	OverlayUnlocked    bool     `json:"overlayUnlocked"`
	OverlayOpened      bool     `json:"overlayOpened"`
	UnlockedTraits     []string `json:"unlockedTraits"`
	UnlockedControls   []string `json:"unlockedControls"`
}

type ComponentKindProgress struct {
	Taps          int      `json:"taps"`
	Level         int      `json:"level"`
	UnlockedModes []string `json:"unlockedModes"`
}

type ComponentDescriptor struct {
	ComponentID   string   `json:"componentId"`
	ComponentKind string   `json:"componentKind"`
	Label         string   `json:"label"`
	Traits        []string `json:"traits"`
}

type ComponentOverlay struct {
	ComponentID      string              `json:"componentId"`
	ComponentKind    string              `json:"componentKind"`
	Title            string              `json:"title"`
	RandomizeEnabled bool                `json:"randomizeEnabled"`
	Controls         []ControlDescriptor `json:"controls"`
}

type ControlDescriptor struct {
	Trait   string          `json:"trait"`
	Control string          `json:"control"`
	Kind    string          `json:"kind"`
	Label   string          `json:"label"`
	Value   any             `json:"value,omitempty"`
	Options []ControlOption `json:"options,omitempty"`
	Min     int             `json:"min,omitempty"`
	Max     int             `json:"max,omitempty"`
	Step    int             `json:"step,omitempty"`
}

type ControlOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type CardEvent struct {
	Type          string `json:"type"`
	ComponentKind string `json:"componentKind,omitempty"`
	ComponentID   string `json:"componentId,omitempty"`
	Trait         string `json:"trait,omitempty"`
	Control       string `json:"control,omitempty"`
	Amount        int    `json:"amount,omitempty"`
	Level         int    `json:"level,omitempty"`
	Mode          string `json:"mode,omitempty"`
	Message       string `json:"message,omitempty"`
}

type tapResult struct {
	document      cardcomponent.Document
	gameState     GameState
	appliedConfig any
	library       []cardcomponent.LibraryItem
	events        []CardEvent
	overlay       *ComponentOverlay
}

type colorControlRequest struct {
	Color          string
	SecondaryColor string
	Gradient       bool
	Angle          int
}

func initialGameState() GameState {
	return normalizeGameState(GameState{
		GlobalLevel:            1,
		UnlockedComponentKinds: []string{componentKindCard},
		SelectedComponentID:    componentCardRoot,
		ComponentProgress: map[string]ComponentProgress{
			componentCardRoot: {
				ComponentID:   componentCardRoot,
				ComponentKind: componentKindCard,
				Level:         1,
			},
		},
	})
}

func normalizeGameState(state GameState) GameState {
	if state.TotalXP == 0 && state.XP > 0 {
		state.TotalXP = state.XP
	}
	if state.TotalInteractions == 0 && state.TapCount > 0 {
		state.TotalInteractions = state.TapCount
	}
	if state.TotalXP < 0 {
		state.TotalXP = 0
	}
	state.GlobalLevel = state.TotalXP/xpPerLevel + 1
	if state.GlobalLevel < 1 {
		state.GlobalLevel = 1
	}
	if state.ComponentProgress == nil {
		state.ComponentProgress = map[string]ComponentProgress{}
	}
	state = ensureComponentProgress(state, componentCardRoot, componentKindCard)
	state.UnlockedComponentKinds = appendStringOnce(state.UnlockedComponentKinds, componentKindCard)
	if state.GlobalLevel >= 3 {
		state.UnlockedComponentKinds = appendStringOnce(state.UnlockedComponentKinds, componentKindTextarea)
		state = ensureComponentProgress(state, componentTextarea, componentKindTextarea)
	}
	if state.GlobalLevel >= 7 {
		state.UnlockedComponentKinds = appendStringOnce(state.UnlockedComponentKinds, componentKindShape)
		state = ensureComponentProgress(state, componentShape, componentKindShape)
	}
	for id, progress := range state.ComponentProgress {
		progress.ComponentID = strings.TrimSpace(progress.ComponentID)
		if progress.ComponentID == "" {
			progress.ComponentID = id
		}
		if progress.ComponentKind == "" {
			progress.ComponentKind = componentKindForID(progress.ComponentID)
		}
		progress.XP = maxInt(progress.XP, 0)
		progress.Interactions = maxInt(progress.Interactions, 0)
		progress.Level = progress.XP/componentXPPerLevel + 1
		progress.RandomTapEnabled = !progress.PreventRandomizing
		progress.OverlayUnlocked = progress.Level >= overlayUnlockLevel
		progress.UnlockedTraits = unlockedTraits(progress.ComponentKind, state.GlobalLevel, progress.Level)
		progress.UnlockedControls = unlockedControls(progress.ComponentKind, state.GlobalLevel, progress.Level)
		state.ComponentProgress[id] = progress
	}
	if strings.TrimSpace(state.SelectedComponentID) == "" {
		state.SelectedComponentID = componentCardRoot
	}
	return syncLegacyGameState(state)
}

func ensureComponentProgress(state GameState, componentID, componentKind string) GameState {
	progress := state.ComponentProgress[componentID]
	progress.ComponentID = componentID
	progress.ComponentKind = componentKind
	if progress.Level < 1 {
		progress.Level = 1
	}
	state.ComponentProgress[componentID] = progress
	return state
}

func syncLegacyGameState(state GameState) GameState {
	state.XP = state.TotalXP
	state.Level = state.GlobalLevel
	state.TapCount = state.TotalInteractions
	state.UnlockedConfigKinds = []string{background.Kind, border.Kind}
	if componentKindUnlocked(state, componentKindTextarea) {
		state.UnlockedConfigKinds = appendStringOnce(state.UnlockedConfigKinds, textarea.Kind)
	}
	if componentKindUnlocked(state, componentKindShape) {
		state.UnlockedConfigKinds = appendStringOnce(state.UnlockedConfigKinds, shape.Kind)
	}
	if componentKindUnlocked(state, componentKindImage) {
		state.UnlockedConfigKinds = appendStringOnce(state.UnlockedConfigKinds, imagecomponent.Kind)
	}
	state.UnlockedModes = []string{editModeRandom}
	if state.GlobalLevel >= 5 {
		state.UnlockedModes = appendStringOnce(state.UnlockedModes, editModeSimpleControls)
	}
	state.ComponentKindProgress = map[string]ComponentKindProgress{}
	rootProgress := state.ComponentProgress[componentCardRoot]
	rootModes := []string{editModeRandom}
	if state.GlobalLevel >= 5 {
		rootModes = appendStringOnce(rootModes, editModeSimpleControls)
	}
	state.ComponentKindProgress[background.Kind] = ComponentKindProgress{Taps: rootProgress.Interactions, Level: rootProgress.Level, UnlockedModes: rootModes}
	state.ComponentKindProgress[border.Kind] = ComponentKindProgress{Taps: rootProgress.Interactions, Level: rootProgress.Level, UnlockedModes: rootModes}
	if textProgress, ok := state.ComponentProgress[componentTextarea]; ok {
		modes := []string{editModeRandom}
		if len(textProgress.UnlockedControls) > 0 {
			modes = appendStringOnce(modes, editModeSimpleControls)
		}
		state.ComponentKindProgress[textarea.Kind] = ComponentKindProgress{Taps: textProgress.Interactions, Level: textProgress.Level, UnlockedModes: modes}
	}
	if shapeProgress, ok := state.ComponentProgress[componentShape]; ok {
		modes := []string{editModeRandom}
		if len(shapeProgress.UnlockedControls) > 0 {
			modes = appendStringOnce(modes, editModeSimpleControls)
		}
		state.ComponentKindProgress[shape.Kind] = ComponentKindProgress{Taps: shapeProgress.Interactions, Level: shapeProgress.Level, UnlockedModes: modes}
	}
	for id, imageProgress := range state.ComponentProgress {
		if imageProgress.ComponentKind != componentKindImage {
			continue
		}
		modes := []string{editModeRandom}
		if len(imageProgress.UnlockedControls) > 0 {
			modes = appendStringOnce(modes, editModeSimpleControls)
		}
		state.ComponentKindProgress[id] = ComponentKindProgress{Taps: imageProgress.Interactions, Level: imageProgress.Level, UnlockedModes: modes}
	}
	return state
}

func unlockedTraits(componentKind string, globalLevel, componentLevel int) []string {
	switch componentKind {
	case componentKindCard:
		traits := []string{traitBackground, traitBorder}
		if globalLevel >= 2 {
			traits = append(traits, traitShadow)
		}
		if globalLevel >= 4 {
			traits = append(traits, traitPadding)
		}
		return traits
	case componentKindTextarea:
		traits := []string{traitText, traitBackground, traitBorder, traitPosition}
		if globalLevel >= 6 || componentLevel >= 4 {
			traits = append(traits, traitTypography)
		}
		if globalLevel >= 4 || componentLevel >= 6 {
			traits = append(traits, traitPadding)
		}
		return traits
	case componentKindShape:
		traits := []string{traitGeometry, traitFill, traitPosition}
		if globalLevel >= 8 || componentLevel >= 5 {
			traits = append(traits, traitBorder)
		}
		if globalLevel >= 10 || componentLevel >= 6 {
			traits = append(traits, traitSize)
		}
		if componentLevel >= 8 {
			traits = append(traits, traitShadow)
		}
		return traits
	case componentKindImage:
		return []string{traitImage, traitPosition, traitSize, traitBorder}
	default:
		return nil
	}
}

func unlockedControls(componentKind string, globalLevel, componentLevel int) []string {
	switch componentKind {
	case componentKindCard:
		controls := randomLockControls(componentLevel)
		if globalLevel < 5 {
			return controls
		}
		controls = append(controls, "backgroundColor", "borderColor", "borderWidthPx", "borderRadiusPx")
		if globalLevel >= 2 {
			controls = append(controls, "shadowPreset")
		}
		if globalLevel >= 4 {
			controls = append(controls, "paddingPx")
		}
		return controls
	case componentKindTextarea:
		if componentLevel < overlayUnlockLevel {
			return nil
		}
		controls := append(randomLockControls(componentLevel), "content", "backgroundColor", "borderColor", "x", "y", "position")
		if globalLevel >= 6 || componentLevel >= 4 {
			controls = append(controls, "textColor")
		}
		if globalLevel >= 6 || componentLevel >= 5 {
			controls = append(controls, "fontFamily", "fontSizePx", "fontWeight")
		}
		if globalLevel >= 4 || componentLevel >= 6 {
			controls = append(controls, "paddingPx")
		}
		if componentLevel >= 7 {
			controls = append(controls, "borderWidthPx", "borderRadiusPx")
		}
		return controls
	case componentKindShape:
		if componentLevel < overlayUnlockLevel {
			return nil
		}
		controls := append(randomLockControls(componentLevel), "shape", "backgroundColor", "x", "y", "position")
		if globalLevel >= 8 || componentLevel >= 5 {
			controls = append(controls, "borderColor", "borderWidthPx")
		}
		if globalLevel >= 10 || componentLevel >= 6 {
			controls = append(controls, "width", "height")
		}
		if globalLevel >= 10 || componentLevel >= 7 {
			controls = append(controls, "rotation")
		}
		if componentLevel >= 8 {
			controls = append(controls, "shadowPreset")
		}
		return controls
	case componentKindImage:
		if componentLevel < overlayUnlockLevel {
			return nil
		}
		return append(randomLockControls(componentLevel), "src", "alt", "x", "y", "position", "width", "height", "rotation", "borderColor", "borderWidthPx", "borderRadiusPx")
	default:
		return nil
	}
}

func randomLockControls(componentLevel int) []string {
	if componentLevel < overlayUnlockLevel {
		return nil
	}
	return []string{"preventRandomizing"}
}

func canonicalTapComponent(target, zone string) (string, string) {
	target = canonicalTapTarget(target, zone)
	switch target {
	case background.Kind:
		return componentCardRoot, traitBackground
	case border.Kind:
		return componentCardRoot, traitBorder
	case textarea.Kind:
		return componentTextarea, traitText
	case shape.Kind:
		return componentShape, traitGeometry
	case imagecomponent.Kind:
		return target, traitImage
	case componentCardRoot:
		return componentCardRoot, ""
	default:
		return target, ""
	}
}

func canonicalTapTarget(target, zone string) string {
	target = strings.TrimSpace(target)
	zone = strings.TrimSpace(zone)
	if target == "" {
		target = zone
	}
	switch target {
	case "interior":
		return background.Kind
	default:
		return target
	}
}

func isKnownComponentID(componentID string) bool {
	switch componentID {
	case componentCardRoot, componentTextarea, componentShape:
		return true
	default:
		return strings.TrimSpace(componentID) != ""
	}
}

func isKnownTapTarget(target string) bool {
	switch target {
	case background.Kind, border.Kind, textarea.Kind, shape.Kind, componentCardRoot:
		return true
	default:
		return false
	}
}

func componentUnlocked(state GameState, componentID string) bool {
	state = normalizeGameState(state)
	switch componentID {
	case componentCardRoot:
		return true
	case componentTextarea:
		return componentKindUnlocked(state, componentKindTextarea)
	case componentShape:
		return componentKindUnlocked(state, componentKindShape)
	default:
		progress, ok := state.ComponentProgress[componentID]
		return ok && componentKindUnlocked(state, progress.ComponentKind)
	}
}

func componentKindUnlocked(state GameState, componentKind string) bool {
	for _, candidate := range state.UnlockedComponentKinds {
		if candidate == componentKind {
			return true
		}
	}
	return false
}

func traitUnlocked(progress ComponentProgress, trait string) bool {
	if strings.TrimSpace(trait) == "" {
		return true
	}
	for _, candidate := range progress.UnlockedTraits {
		if candidate == trait {
			return true
		}
	}
	return false
}

func controlUnlocked(progress ComponentProgress, control string) bool {
	if control == "position" && (progress.ComponentKind == componentKindTextarea || progress.ComponentKind == componentKindShape || progress.ComponentKind == componentKindImage) {
		return true
	}
	for _, candidate := range progress.UnlockedControls {
		if candidate == control {
			return true
		}
	}
	return false
}

func modeUnlocked(state GameState, target, mode string) bool {
	state = normalizeGameState(state)
	if mode != editModeSimpleControls {
		return mode == editModeRandom
	}
	switch target {
	case background.Kind, border.Kind:
		return state.GlobalLevel >= 5
	case textarea.Kind:
		progress := state.ComponentProgress[componentTextarea]
		return len(progress.UnlockedControls) > 0
	case shape.Kind:
		progress := state.ComponentProgress[componentShape]
		return len(progress.UnlockedControls) > 0
	default:
		return false
	}
}

func randomGeneratedConfig(target string, seed int64, level int) (json.RawMessage, error) {
	var value any
	switch target {
	case background.Kind:
		value = background.RandomGenerated(seed, level)
	case border.Kind:
		value = border.RandomGenerated(seed, level)
	case textarea.Kind:
		value = textarea.RandomGenerated(seed, level)
	case shape.Kind:
		value = shape.RandomGenerated(seed, level)
	case imagecomponent.Kind:
		value = imagecomponent.RandomGenerated(seed, level)
	default:
		return nil, fmt.Errorf("target %q does not support random configs", target)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func advanceInteraction(state GameState, componentID string, amount int) (GameState, []CardEvent) {
	state = normalizeGameState(state)
	if amount <= 0 {
		return state, nil
	}
	oldGlobalLevel := state.GlobalLevel
	oldUnlockedTypes := append([]string(nil), state.UnlockedComponentKinds...)
	progress := state.ComponentProgress[componentID]
	oldComponentLevel := progress.Level
	progress.XP += amount
	progress.Interactions += amount
	state.TotalXP += amount
	state.TotalInteractions += amount
	state.ComponentProgress[componentID] = progress
	state = normalizeGameState(state)

	progress = state.ComponentProgress[componentID]
	events := []CardEvent{{
		Type:        "xpGained",
		ComponentID: componentID,
		Amount:      amount,
	}}
	if state.GlobalLevel > oldGlobalLevel {
		events = append(events, CardEvent{Type: "levelUp", Level: state.GlobalLevel})
	}
	if oldGlobalLevel < 5 && state.GlobalLevel >= 5 {
		events = append(events,
			CardEvent{Type: "modeUnlocked", ComponentKind: background.Kind, Mode: editModeSimpleControls},
			CardEvent{Type: "modeUnlocked", ComponentKind: border.Kind, Mode: editModeSimpleControls},
		)
	}
	if progress.Level > oldComponentLevel {
		events = append(events, CardEvent{
			Type:          "componentLevelUp",
			ComponentID:   componentID,
			ComponentKind: progress.ComponentKind,
			Level:         progress.Level,
		})
	}
	for _, componentKind := range state.UnlockedComponentKinds {
		if !stringInSlice(oldUnlockedTypes, componentKind) {
			events = append(events, CardEvent{
				Type:          "componentUnlocked",
				ComponentKind: componentKind,
				Message:       componentLabel(componentKind) + " unlocked",
			})
			switch componentKind {
			case componentKindTextarea:
				events = append(events, CardEvent{Type: "configKindUnlocked", ComponentKind: textarea.Kind})
			case componentKindShape:
				events = append(events, CardEvent{Type: "configKindUnlocked", ComponentKind: shape.Kind})
			}
		}
	}
	return state, events
}

func tapSeed(state GameState, componentID string) int64 {
	var componentOffset int64
	for _, char := range componentID {
		componentOffset += int64(char)
	}
	return time.Now().UnixNano() + int64(state.TotalInteractions+1)*7919 + componentOffset
}

func availableComponents(state GameState, document cardcomponent.Document) []ComponentDescriptor {
	state = normalizeGameState(state)
	out := []ComponentDescriptor{{
		ComponentID:   componentCardRoot,
		ComponentKind: componentKindCard,
		Label:         "Card",
		Traits:        state.ComponentProgress[componentCardRoot].UnlockedTraits,
	}}
	for _, node := range document.Root.Children {
		if node.ComponentKind != componentKindTextarea && node.ComponentKind != componentKindShape && node.ComponentKind != componentKindImage {
			continue
		}
		if !componentUnlocked(state, node.ID) {
			continue
		}
		progress := state.ComponentProgress[node.ID]
		out = append(out, ComponentDescriptor{
			ComponentID:   node.ID,
			ComponentKind: node.ComponentKind,
			Label:         componentLabel(node.ComponentKind),
			Traits:        progress.UnlockedTraits,
		})
	}
	return out
}

func buildOverlay(document cardcomponent.Document, state GameState, componentID string) *ComponentOverlay {
	state = normalizeGameState(state)
	progress, ok := state.ComponentProgress[componentID]
	if !ok || !componentUnlocked(state, componentID) || !progress.OverlayUnlocked {
		return nil
	}
	overlay := &ComponentOverlay{
		ComponentID:      componentID,
		ComponentKind:    progress.ComponentKind,
		Title:            componentTitle(progress.ComponentKind),
		RandomizeEnabled: false,
	}
	switch progress.ComponentKind {
	case componentKindCard:
		overlay.Controls = append(overlay.Controls, cardControls(document, progress)...)
	case componentKindTextarea:
		overlay.Controls = append(overlay.Controls, textareaControls(document, progress)...)
	case componentKindShape:
		overlay.Controls = append(overlay.Controls, shapeControls(document, progress)...)
	case componentKindImage:
		overlay.Controls = append(overlay.Controls, imageControls(document, progress)...)
	}
	return overlay
}

func cardControls(document cardcomponent.Document, progress ComponentProgress) []ControlDescriptor {
	controls := randomLockControl(progress)
	bg := currentBackground(document)
	br := currentBorder(document)
	root := cardcomponent.DecodeRootConfig(document.Root.Config)
	if controlUnlocked(progress, "backgroundColor") {
		controls = append(controls, colorControl(traitBackground, "backgroundColor", "Background", bg.BackgroundColor))
	}
	if controlUnlocked(progress, "borderColor") {
		controls = append(controls, colorControl(traitBorder, "borderColor", "Border", br.BorderColor))
	}
	if controlUnlocked(progress, "borderWidthPx") {
		controls = append(controls, rangeControl(traitBorder, "borderWidthPx", "Border Width", br.BorderWidthPX, 0, 16, 1))
	}
	if controlUnlocked(progress, "borderRadiusPx") {
		controls = append(controls, rangeControl(traitBorder, "borderRadiusPx", "Border Radius", br.BorderRadiusPX, 0, 64, 1))
	}
	if controlUnlocked(progress, "shadowPreset") {
		controls = append(controls, selectControl(traitShadow, "shadowPreset", "Shadow", root.Shadow, shadowOptions()))
	}
	if controlUnlocked(progress, "paddingPx") {
		controls = append(controls, rangeControl(traitPadding, "paddingPx", "Padding", root.PaddingPX, 0, 48, 1))
	}
	return controls
}

func textareaControls(document cardcomponent.Document, progress ComponentProgress) []ControlDescriptor {
	part := currentTextarea(document, progress.ComponentID)
	controls := randomLockControl(progress)
	if controlUnlocked(progress, "content") {
		controls = append(controls, ControlDescriptor{Trait: traitText, Control: "content", Kind: "text", Label: "Text", Value: part.Content})
	}
	if controlUnlocked(progress, "backgroundColor") {
		controls = append(controls, colorControl(traitBackground, "backgroundColor", "Background", part.BackgroundColor))
	}
	if controlUnlocked(progress, "borderColor") {
		controls = append(controls, colorControl(traitBorder, "borderColor", "Border", part.BorderColor))
	}
	if controlUnlocked(progress, "textColor") {
		controls = append(controls, colorControl(traitText, "textColor", "Text Color", part.Color))
	}
	if controlUnlocked(progress, "fontFamily") {
		controls = append(controls, selectControl(traitTypography, "fontFamily", "Font", part.FontFamily, []ControlOption{
			{Label: "System", Value: "system"},
			{Label: "Serif", Value: "serif"},
			{Label: "Mono", Value: "mono"},
			{Label: "Display", Value: "display"},
		}))
	}
	if controlUnlocked(progress, "fontSizePx") {
		controls = append(controls, rangeControl(traitTypography, "fontSizePx", "Font Size", part.FontSizePX, 10, 72, 1))
	}
	if controlUnlocked(progress, "fontWeight") {
		controls = append(controls, selectControl(traitTypography, "fontWeight", "Weight", fmt.Sprintf("%d", part.FontWeight), []ControlOption{
			{Label: "400", Value: "400"},
			{Label: "500", Value: "500"},
			{Label: "600", Value: "600"},
			{Label: "700", Value: "700"},
			{Label: "800", Value: "800"},
		}))
	}
	if controlUnlocked(progress, "paddingPx") {
		controls = append(controls, rangeControl(traitPadding, "paddingPx", "Padding", part.PaddingPX, 0, 32, 1))
	}
	if controlUnlocked(progress, "x") {
		controls = append(controls, rangeControl(traitPosition, "x", "X", part.X, 0, 100, 1))
	}
	if controlUnlocked(progress, "y") {
		controls = append(controls, rangeControl(traitPosition, "y", "Y", part.Y, 0, 100, 1))
	}
	if controlUnlocked(progress, "borderWidthPx") {
		controls = append(controls, rangeControl(traitBorder, "borderWidthPx", "Border Width", part.BorderWidthPX, 0, 12, 1))
	}
	if controlUnlocked(progress, "borderRadiusPx") {
		controls = append(controls, rangeControl(traitBorder, "borderRadiusPx", "Radius", part.BorderRadiusPX, 0, 40, 1))
	}
	return controls
}

func shapeControls(document cardcomponent.Document, progress ComponentProgress) []ControlDescriptor {
	part := currentShape(document, progress.ComponentID)
	controls := randomLockControl(progress)
	if controlUnlocked(progress, "shape") {
		controls = append(controls, selectControl(traitGeometry, "shape", "Shape", part.Shape, shapeOptions()))
	}
	if controlUnlocked(progress, "backgroundColor") {
		controls = append(controls, colorControl(traitFill, "backgroundColor", "Fill", part.BackgroundColor))
	}
	if controlUnlocked(progress, "borderColor") {
		controls = append(controls, colorControl(traitBorder, "borderColor", "Border", part.BorderColor))
	}
	if controlUnlocked(progress, "borderWidthPx") {
		controls = append(controls, rangeControl(traitBorder, "borderWidthPx", "Border Width", part.BorderWidthPX, 0, 10, 1))
	}
	if controlUnlocked(progress, "width") {
		controls = append(controls, rangeControl(traitSize, "width", "Width", part.Width, 8, 100, 1))
	}
	if controlUnlocked(progress, "height") {
		controls = append(controls, rangeControl(traitSize, "height", "Height", part.Height, 8, 100, 1))
	}
	if controlUnlocked(progress, "x") {
		controls = append(controls, rangeControl(traitPosition, "x", "X", part.X, 0, 100, 1))
	}
	if controlUnlocked(progress, "y") {
		controls = append(controls, rangeControl(traitPosition, "y", "Y", part.Y, 0, 100, 1))
	}
	if controlUnlocked(progress, "rotation") {
		controls = append(controls, rangeControl(traitGeometry, "rotation", "Rotation", part.Rotation, 0, 359, 1))
	}
	if controlUnlocked(progress, "shadowPreset") {
		controls = append(controls, selectControl(traitShadow, "shadowPreset", "Shadow", part.Shadow, shapeShadowOptions()))
	}
	return controls
}

func imageControls(document cardcomponent.Document, progress ComponentProgress) []ControlDescriptor {
	part := currentImage(document, progress.ComponentID)
	controls := randomLockControl(progress)
	if controlUnlocked(progress, "src") {
		controls = append(controls, ControlDescriptor{Trait: traitImage, Control: "src", Kind: "text", Label: "Image Data", Value: part.Src})
	}
	if controlUnlocked(progress, "alt") {
		controls = append(controls, ControlDescriptor{Trait: traitImage, Control: "alt", Kind: "text", Label: "Alt Text", Value: part.Alt})
	}
	if controlUnlocked(progress, "x") {
		controls = append(controls, rangeControl(traitPosition, "x", "X", part.X, 0, 100, 1))
	}
	if controlUnlocked(progress, "y") {
		controls = append(controls, rangeControl(traitPosition, "y", "Y", part.Y, 0, 100, 1))
	}
	if controlUnlocked(progress, "width") {
		controls = append(controls, rangeControl(traitSize, "width", "Width", part.Width, 8, 100, 1))
	}
	if controlUnlocked(progress, "height") {
		controls = append(controls, rangeControl(traitSize, "height", "Height", part.Height, 8, 100, 1))
	}
	if controlUnlocked(progress, "rotation") {
		controls = append(controls, rangeControl(traitImage, "rotation", "Rotation", part.Rotation, 0, 359, 1))
	}
	if controlUnlocked(progress, "borderColor") {
		controls = append(controls, colorControl(traitBorder, "borderColor", "Border", part.BorderColor))
	}
	if controlUnlocked(progress, "borderWidthPx") {
		controls = append(controls, rangeControl(traitBorder, "borderWidthPx", "Border Width", part.BorderWidthPX, 0, 12, 1))
	}
	if controlUnlocked(progress, "borderRadiusPx") {
		controls = append(controls, rangeControl(traitBorder, "borderRadiusPx", "Radius", part.BorderRadiusPX, 0, 48, 1))
	}
	return controls
}

func colorControl(trait, control, label, value string) ControlDescriptor {
	return ControlDescriptor{Trait: trait, Control: control, Kind: "color", Label: label, Value: value}
}

func rangeControl(trait, control, label string, value, min, max, step int) ControlDescriptor {
	return ControlDescriptor{Trait: trait, Control: control, Kind: "range", Label: label, Value: value, Min: min, Max: max, Step: step}
}

func selectControl(trait, control, label, value string, options []ControlOption) ControlDescriptor {
	return ControlDescriptor{Trait: trait, Control: control, Kind: "select", Label: label, Value: value, Options: options}
}

func checkboxControl(trait, control, label string, value bool) ControlDescriptor {
	return ControlDescriptor{Trait: trait, Control: control, Kind: "checkbox", Label: label, Value: value}
}

func randomLockControl(progress ComponentProgress) []ControlDescriptor {
	if !controlUnlocked(progress, "preventRandomizing") {
		return nil
	}
	return []ControlDescriptor{
		checkboxControl("", "preventRandomizing", "Prevent randomizing on tap", progress.PreventRandomizing),
	}
}

func currentBackground(document cardcomponent.Document) background.Config {
	part := background.DefaultConfig()
	if node := findNodeByKind(document.Root, background.Kind); node != nil && len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	return part
}

func currentBorder(document cardcomponent.Document) border.Config {
	part := border.DefaultConfig()
	if node := findNodeByKind(document.Root, border.Kind); node != nil && len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	return part
}

func currentTextarea(document cardcomponent.Document, componentID string) textarea.Config {
	part := textarea.DefaultConfig()
	if strings.TrimSpace(componentID) == "" {
		componentID = componentTextarea
	}
	if node := findNodeByID(document.Root, componentID); node != nil && len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[textarea.Config]{ComponentKind: textarea.Kind, Description: "Current textarea", Config: part}
	textarea.NormalizeGenerated(&generated)
	return generated.Config
}

func currentShape(document cardcomponent.Document, componentID string) shape.Config {
	part := shape.DefaultConfig()
	if strings.TrimSpace(componentID) == "" {
		componentID = componentShape
	}
	if node := findNodeByID(document.Root, componentID); node != nil && len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[shape.Config]{ComponentKind: shape.Kind, Description: "Current shape", Config: part}
	shape.NormalizeGenerated(&generated)
	return generated.Config
}

func currentImage(document cardcomponent.Document, componentID string) imagecomponent.Config {
	part := imagecomponent.DefaultConfig()
	if node := findNodeByID(document.Root, componentID); node != nil && len(node.Config) > 0 {
		_ = json.Unmarshal(node.Config, &part)
	}
	generated := design.GeneratedConfig[imagecomponent.Config]{ComponentKind: imagecomponent.Kind, Description: "Current image", Config: part}
	imagecomponent.NormalizeGenerated(&generated)
	return generated.Config
}

func shadowOptions() []ControlOption {
	return []ControlOption{
		{Label: "None", Value: ""},
		{Label: "Soft", Value: "0 18px 48px rgba(15,23,42,0.28)"},
		{Label: "Lift", Value: "0 28px 70px rgba(15,23,42,0.42)"},
		{Label: "Glow", Value: "0 0 34px rgba(52,211,153,0.28)"},
	}
}

func shapeOptions() []ControlOption {
	return []ControlOption{
		{Label: "Circle", Value: "circle"},
		{Label: "Oval", Value: "oval"},
		{Label: "Rectangle", Value: "rectangle"},
		{Label: "Rounded", Value: "roundedRectangle"},
		{Label: "Triangle", Value: "triangle"},
		{Label: "Diamond", Value: "diamond"},
		{Label: "Star", Value: "star"},
		{Label: "Blob", Value: "blob"},
	}
}

func shapeShadowOptions() []ControlOption {
	return []ControlOption{
		{Label: "None", Value: ""},
		{Label: "Soft", Value: "0 10px 24px rgba(15,23,42,0.22)"},
		{Label: "Lift", Value: "0 12px 28px rgba(15,23,42,0.28)"},
		{Label: "Rose", Value: "0 12px 26px rgba(244,63,94,0.24)"},
		{Label: "Sky", Value: "0 10px 24px rgba(14,165,233,0.22)"},
	}
}

func componentKindForID(componentID string) string {
	switch componentID {
	case componentTextarea:
		return componentKindTextarea
	case componentShape:
		return componentKindShape
	default:
		return componentKindCard
	}
}

func componentTitle(componentKind string) string {
	switch componentKind {
	case componentKindTextarea:
		return "Text"
	case componentKindShape:
		return "Shape"
	case componentKindImage:
		return "Image"
	default:
		return "Card"
	}
}

func componentLabel(componentKind string) string {
	return componentTitle(componentKind)
}

func appendStringOnce(values []string, value string) []string {
	for _, candidate := range values {
		if candidate == value {
			return values
		}
	}
	return append(values, value)
}

func stringInSlice(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func colorGeneratedConfig(document cardcomponent.Document, target string, request colorControlRequest) (json.RawMessage, error) {
	color := strings.TrimSpace(request.Color)
	if !design.IsAllowedColor(color) {
		return nil, fmt.Errorf("color must be a hex, rgb, rgba, hsl, or hsla color")
	}
	secondaryColor := strings.TrimSpace(request.SecondaryColor)
	if request.Gradient {
		if secondaryColor == "" {
			return nil, fmt.Errorf("secondaryColor is required for gradients")
		}
		if !design.IsAllowedColor(secondaryColor) {
			return nil, fmt.Errorf("secondaryColor must be a hex, rgb, rgba, hsl, or hsla color")
		}
	}
	angle := normalizeGradientAngle(request.Angle)
	switch target {
	case background.Kind:
		part := currentBackground(document)
		declarations := design.CSSDeclarations(part.CSS, background.AllowedCSS())
		part.BackgroundColor = color
		if request.Gradient {
			part.CSS = fmt.Sprintf("background: linear-gradient(%ddeg, %s 0%%, %s 100%%);", angle, color, secondaryColor)
		} else {
			part.CSS = "background: " + color + ";"
		}
		if shadow := strings.TrimSpace(declarations["box-shadow"]); shadow != "" {
			part.CSS += " box-shadow: " + shadow + ";"
		}
		description := "Background color changed"
		if request.Gradient {
			description = "Background gradient changed"
		}
		return json.Marshal(design.GeneratedConfig[background.Config]{
			ComponentKind: background.Kind,
			Description:   description,
			Config:        part,
		})
	case border.Kind:
		part := currentBorder(document)
		declarations := design.CSSDeclarations(part.CSS, border.AllowedCSS())
		part.BorderColor = color
		part.CSS = fmt.Sprintf("border: %dpx solid %s;", part.BorderWidthPX, color)
		if request.Gradient {
			part.CSS += fmt.Sprintf(" border-image: linear-gradient(%ddeg, %s 0%%, %s 100%%) 1;", angle, color, secondaryColor)
		}
		if shadow := strings.TrimSpace(declarations["box-shadow"]); shadow != "" {
			part.CSS += " box-shadow: " + shadow + ";"
		}
		description := "Border color changed"
		if request.Gradient {
			description = "Border gradient changed"
		}
		return json.Marshal(design.GeneratedConfig[border.Config]{
			ComponentKind: border.Kind,
			Description:   description,
			Config:        part,
		})
	default:
		return nil, fmt.Errorf("target %q does not support color controls", target)
	}
}

func normalizeGradientAngle(angle int) int {
	if angle == 0 {
		return 135
	}
	angle = angle % 360
	if angle < 0 {
		angle += 360
	}
	return angle
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
