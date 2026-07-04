package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/shape"
	"github.com/n0remac/Living-Card/internal/components/textarea"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const (
	componentTypeCard     = cardcomponent.Type
	componentTypeTextarea = cardcomponent.TypeTextarea
	componentTypeShape    = cardcomponent.TypeShape

	componentCardRoot = cardcomponent.DefaultRootID
	componentTextarea = cardcomponent.DefaultTextareaID
	componentShape    = cardcomponent.DefaultShapeID

	traitBackground = "background"
	traitBorder     = "border"
	traitShadow     = "shadow"
	traitPadding    = "padding"
	traitText       = "text"
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
	UnlockedComponentTypes []string                     `json:"unlockedComponentTypes"`
	SelectedComponentID    string                       `json:"selectedComponentId,omitempty"`
	ComponentProgress      map[string]ComponentProgress `json:"componentProgress"`

	TapCount        int                       `json:"tapCount"`
	Level           int                       `json:"level"`
	XP              int                       `json:"xp"`
	UnlockedTargets []string                  `json:"unlockedTargets"`
	UnlockedModes   []string                  `json:"unlockedModes"`
	TargetProgress  map[string]TargetProgress `json:"targetProgress"`
}

type ComponentProgress struct {
	ComponentID        string   `json:"componentId"`
	ComponentType      string   `json:"componentType"`
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

type TargetProgress struct {
	Taps          int      `json:"taps"`
	Level         int      `json:"level"`
	UnlockedModes []string `json:"unlockedModes"`
}

type ComponentDescriptor struct {
	ComponentID   string   `json:"componentId"`
	ComponentType string   `json:"componentType"`
	Label         string   `json:"label"`
	Traits        []string `json:"traits"`
}

type ComponentOverlay struct {
	ComponentID      string              `json:"componentId"`
	ComponentType    string              `json:"componentType"`
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
	Target        string `json:"target,omitempty"`
	ComponentID   string `json:"componentId,omitempty"`
	ComponentType string `json:"componentType,omitempty"`
	Trait         string `json:"trait,omitempty"`
	Control       string `json:"control,omitempty"`
	Amount        int    `json:"amount,omitempty"`
	Level         int    `json:"level,omitempty"`
	Mode          string `json:"mode,omitempty"`
	Message       string `json:"message,omitempty"`
}

type tapResult struct {
	document        cardcomponent.Document
	gameState       GameState
	appliedFragment any
	library         []cardcomponent.LibraryItem
	events          []CardEvent
	overlay         *ComponentOverlay
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
		UnlockedComponentTypes: []string{componentTypeCard},
		SelectedComponentID:    componentCardRoot,
		ComponentProgress: map[string]ComponentProgress{
			componentCardRoot: {
				ComponentID:   componentCardRoot,
				ComponentType: componentTypeCard,
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
	state = ensureComponentProgress(state, componentCardRoot, componentTypeCard)
	state.UnlockedComponentTypes = appendStringOnce(state.UnlockedComponentTypes, componentTypeCard)
	if state.GlobalLevel >= 3 {
		state.UnlockedComponentTypes = appendStringOnce(state.UnlockedComponentTypes, componentTypeTextarea)
		state = ensureComponentProgress(state, componentTextarea, componentTypeTextarea)
	}
	if state.GlobalLevel >= 7 {
		state.UnlockedComponentTypes = appendStringOnce(state.UnlockedComponentTypes, componentTypeShape)
		state = ensureComponentProgress(state, componentShape, componentTypeShape)
	}
	for id, progress := range state.ComponentProgress {
		progress.ComponentID = strings.TrimSpace(progress.ComponentID)
		if progress.ComponentID == "" {
			progress.ComponentID = id
		}
		if progress.ComponentType == "" {
			progress.ComponentType = componentTypeForID(progress.ComponentID)
		}
		progress.XP = maxInt(progress.XP, 0)
		progress.Interactions = maxInt(progress.Interactions, 0)
		progress.Level = progress.XP/componentXPPerLevel + 1
		progress.RandomTapEnabled = !progress.PreventRandomizing
		progress.OverlayUnlocked = progress.Level >= overlayUnlockLevel
		progress.UnlockedTraits = unlockedTraits(progress.ComponentType, state.GlobalLevel, progress.Level)
		progress.UnlockedControls = unlockedControls(progress.ComponentType, state.GlobalLevel, progress.Level)
		state.ComponentProgress[id] = progress
	}
	if strings.TrimSpace(state.SelectedComponentID) == "" {
		state.SelectedComponentID = componentCardRoot
	}
	return syncLegacyGameState(state)
}

func ensureComponentProgress(state GameState, componentID, componentType string) GameState {
	progress := state.ComponentProgress[componentID]
	progress.ComponentID = componentID
	progress.ComponentType = componentType
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
	state.UnlockedTargets = []string{background.Type, border.Type}
	if componentTypeUnlocked(state, componentTypeTextarea) {
		state.UnlockedTargets = appendStringOnce(state.UnlockedTargets, textarea.Type)
	}
	if componentTypeUnlocked(state, componentTypeShape) {
		state.UnlockedTargets = appendStringOnce(state.UnlockedTargets, shape.Type)
	}
	state.UnlockedModes = []string{editModeRandom}
	if state.GlobalLevel >= 5 {
		state.UnlockedModes = appendStringOnce(state.UnlockedModes, editModeSimpleControls)
	}
	state.TargetProgress = map[string]TargetProgress{}
	rootProgress := state.ComponentProgress[componentCardRoot]
	rootModes := []string{editModeRandom}
	if state.GlobalLevel >= 5 {
		rootModes = appendStringOnce(rootModes, editModeSimpleControls)
	}
	state.TargetProgress[background.Type] = TargetProgress{Taps: rootProgress.Interactions, Level: rootProgress.Level, UnlockedModes: rootModes}
	state.TargetProgress[border.Type] = TargetProgress{Taps: rootProgress.Interactions, Level: rootProgress.Level, UnlockedModes: rootModes}
	if textProgress, ok := state.ComponentProgress[componentTextarea]; ok {
		modes := []string{editModeRandom}
		if len(textProgress.UnlockedControls) > 0 {
			modes = appendStringOnce(modes, editModeSimpleControls)
		}
		state.TargetProgress[textarea.Type] = TargetProgress{Taps: textProgress.Interactions, Level: textProgress.Level, UnlockedModes: modes}
	}
	if shapeProgress, ok := state.ComponentProgress[componentShape]; ok {
		modes := []string{editModeRandom}
		if len(shapeProgress.UnlockedControls) > 0 {
			modes = appendStringOnce(modes, editModeSimpleControls)
		}
		state.TargetProgress[shape.Type] = TargetProgress{Taps: shapeProgress.Interactions, Level: shapeProgress.Level, UnlockedModes: modes}
	}
	return state
}

func unlockedTraits(componentType string, globalLevel, componentLevel int) []string {
	switch componentType {
	case componentTypeCard:
		traits := []string{traitBackground, traitBorder}
		if globalLevel >= 2 {
			traits = append(traits, traitShadow)
		}
		if globalLevel >= 4 {
			traits = append(traits, traitPadding)
		}
		return traits
	case componentTypeTextarea:
		traits := []string{traitText, traitBackground, traitBorder, traitPosition}
		if globalLevel >= 6 || componentLevel >= 4 {
			traits = append(traits, traitTypography)
		}
		if globalLevel >= 4 || componentLevel >= 6 {
			traits = append(traits, traitPadding)
		}
		return traits
	case componentTypeShape:
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
	default:
		return nil
	}
}

func unlockedControls(componentType string, globalLevel, componentLevel int) []string {
	switch componentType {
	case componentTypeCard:
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
	case componentTypeTextarea:
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
	case componentTypeShape:
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
	case background.Type:
		return componentCardRoot, traitBackground
	case border.Type:
		return componentCardRoot, traitBorder
	case textarea.Type:
		return componentTextarea, traitText
	case shape.Type:
		return componentShape, traitGeometry
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
		return background.Type
	default:
		return target
	}
}

func isKnownComponentID(componentID string) bool {
	switch componentID {
	case componentCardRoot, componentTextarea, componentShape:
		return true
	default:
		return false
	}
}

func isKnownTapTarget(target string) bool {
	switch target {
	case background.Type, border.Type, textarea.Type, shape.Type, componentCardRoot:
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
		return componentTypeUnlocked(state, componentTypeTextarea)
	case componentShape:
		return componentTypeUnlocked(state, componentTypeShape)
	default:
		return false
	}
}

func componentTypeUnlocked(state GameState, componentType string) bool {
	for _, candidate := range state.UnlockedComponentTypes {
		if candidate == componentType {
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
	if control == "position" && (progress.ComponentType == componentTypeTextarea || progress.ComponentType == componentTypeShape) {
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
	case background.Type, border.Type:
		return state.GlobalLevel >= 5
	case textarea.Type:
		progress := state.ComponentProgress[componentTextarea]
		return len(progress.UnlockedControls) > 0
	case shape.Type:
		progress := state.ComponentProgress[componentShape]
		return len(progress.UnlockedControls) > 0
	default:
		return false
	}
}

func randomGeneratedFragment(target string, seed int64, level int) (json.RawMessage, error) {
	var value any
	switch target {
	case background.Type:
		value = background.RandomGenerated(seed, level)
	case border.Type:
		value = border.RandomGenerated(seed, level)
	case textarea.Type:
		value = textarea.RandomGenerated(seed, level)
	case shape.Type:
		value = shape.RandomGenerated(seed, level)
	default:
		return nil, fmt.Errorf("target %q does not support random fragments", target)
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
	oldUnlockedTypes := append([]string(nil), state.UnlockedComponentTypes...)
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
			CardEvent{Type: "modeUnlocked", Target: background.Type, Mode: editModeSimpleControls},
			CardEvent{Type: "modeUnlocked", Target: border.Type, Mode: editModeSimpleControls},
		)
	}
	if progress.Level > oldComponentLevel {
		events = append(events, CardEvent{
			Type:          "componentLevelUp",
			ComponentID:   componentID,
			ComponentType: progress.ComponentType,
			Level:         progress.Level,
		})
	}
	for _, componentType := range state.UnlockedComponentTypes {
		if !stringInSlice(oldUnlockedTypes, componentType) {
			events = append(events, CardEvent{
				Type:          "componentUnlocked",
				ComponentType: componentType,
				Message:       componentLabel(componentType) + " unlocked",
			})
			switch componentType {
			case componentTypeTextarea:
				events = append(events, CardEvent{Type: "targetUnlocked", Target: textarea.Type})
			case componentTypeShape:
				events = append(events, CardEvent{Type: "targetUnlocked", Target: shape.Type})
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
		ComponentType: componentTypeCard,
		Label:         "Card",
		Traits:        state.ComponentProgress[componentCardRoot].UnlockedTraits,
	}}
	if componentUnlocked(state, componentTextarea) && findNodeByID(document.Root, componentTextarea) != nil {
		progress := state.ComponentProgress[componentTextarea]
		out = append(out, ComponentDescriptor{
			ComponentID:   componentTextarea,
			ComponentType: componentTypeTextarea,
			Label:         "Text",
			Traits:        progress.UnlockedTraits,
		})
	}
	if componentUnlocked(state, componentShape) && findNodeByID(document.Root, componentShape) != nil {
		progress := state.ComponentProgress[componentShape]
		out = append(out, ComponentDescriptor{
			ComponentID:   componentShape,
			ComponentType: componentTypeShape,
			Label:         "Shape",
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
		ComponentType:    progress.ComponentType,
		Title:            componentTitle(progress.ComponentType),
		RandomizeEnabled: false,
	}
	switch progress.ComponentType {
	case componentTypeCard:
		overlay.Controls = append(overlay.Controls, cardControls(document, progress)...)
	case componentTypeTextarea:
		overlay.Controls = append(overlay.Controls, textareaControls(document, progress)...)
	case componentTypeShape:
		overlay.Controls = append(overlay.Controls, shapeControls(document, progress)...)
	}
	return overlay
}

func cardControls(document cardcomponent.Document, progress ComponentProgress) []ControlDescriptor {
	controls := randomLockControl(progress)
	bg := currentBackground(document)
	br := currentBorder(document)
	root := cardcomponent.DecodeRootFragment(document.Root.Fragment)
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
	part := currentTextarea(document)
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
	part := currentShape(document)
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

func currentBackground(document cardcomponent.Document) background.Fragment {
	part := background.DefaultFragment()
	if node := findNodeByType(document.Root, background.Type); node != nil && len(node.Fragment) > 0 {
		_ = json.Unmarshal(node.Fragment, &part)
	}
	return part
}

func currentBorder(document cardcomponent.Document) border.Fragment {
	part := border.DefaultFragment()
	if node := findNodeByType(document.Root, border.Type); node != nil && len(node.Fragment) > 0 {
		_ = json.Unmarshal(node.Fragment, &part)
	}
	return part
}

func currentTextarea(document cardcomponent.Document) textarea.Fragment {
	part := textarea.DefaultFragment()
	if node := findNodeByID(document.Root, componentTextarea); node != nil && len(node.Fragment) > 0 {
		_ = json.Unmarshal(node.Fragment, &part)
	}
	generated := fragment.Generated[textarea.Fragment]{Target: textarea.Type, Description: "Current textarea", Fragment: part}
	textarea.NormalizeGenerated(&generated)
	return generated.Fragment
}

func currentShape(document cardcomponent.Document) shape.Fragment {
	part := shape.DefaultFragment()
	if node := findNodeByID(document.Root, componentShape); node != nil && len(node.Fragment) > 0 {
		_ = json.Unmarshal(node.Fragment, &part)
	}
	generated := fragment.Generated[shape.Fragment]{Target: shape.Type, Description: "Current shape", Fragment: part}
	shape.NormalizeGenerated(&generated)
	return generated.Fragment
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

func componentTypeForID(componentID string) string {
	switch componentID {
	case componentTextarea:
		return componentTypeTextarea
	case componentShape:
		return componentTypeShape
	default:
		return componentTypeCard
	}
}

func componentTitle(componentType string) string {
	switch componentType {
	case componentTypeTextarea:
		return "Text"
	case componentTypeShape:
		return "Shape"
	default:
		return "Card"
	}
}

func componentLabel(componentType string) string {
	return componentTitle(componentType)
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

func colorGeneratedFragment(document cardcomponent.Document, target string, request colorControlRequest) (json.RawMessage, error) {
	color := strings.TrimSpace(request.Color)
	if !fragment.IsAllowedColor(color) {
		return nil, fmt.Errorf("color must be a hex, rgb, rgba, hsl, or hsla color")
	}
	secondaryColor := strings.TrimSpace(request.SecondaryColor)
	if request.Gradient {
		if secondaryColor == "" {
			return nil, fmt.Errorf("secondaryColor is required for gradients")
		}
		if !fragment.IsAllowedColor(secondaryColor) {
			return nil, fmt.Errorf("secondaryColor must be a hex, rgb, rgba, hsl, or hsla color")
		}
	}
	angle := normalizeGradientAngle(request.Angle)
	switch target {
	case background.Type:
		part := currentBackground(document)
		declarations := fragment.CSSDeclarations(part.CSS, background.AllowedCSS())
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
		return json.Marshal(fragment.Generated[background.Fragment]{
			Target:      background.Type,
			Description: description,
			Fragment:    part,
		})
	case border.Type:
		part := currentBorder(document)
		declarations := fragment.CSSDeclarations(part.CSS, border.AllowedCSS())
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
		return json.Marshal(fragment.Generated[border.Fragment]{
			Target:      border.Type,
			Description: description,
			Fragment:    part,
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
