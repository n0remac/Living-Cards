package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/fragment"
)

const (
	editModeRandom         = "random"
	editModeSimpleControls = "simpleControls"
	simpleControlsLevel    = 5
	xpPerTap               = 1
	xpPerLevel             = 5
)

type GameState struct {
	TapCount        int                       `json:"tapCount"`
	Level           int                       `json:"level"`
	XP              int                       `json:"xp"`
	UnlockedTargets []string                  `json:"unlockedTargets"`
	UnlockedModes   []string                  `json:"unlockedModes"`
	TargetProgress  map[string]TargetProgress `json:"targetProgress"`
}

type TargetProgress struct {
	Taps          int      `json:"taps"`
	Level         int      `json:"level"`
	UnlockedModes []string `json:"unlockedModes"`
}

type CardEvent struct {
	Type    string `json:"type"`
	Target  string `json:"target,omitempty"`
	Amount  int    `json:"amount,omitempty"`
	Level   int    `json:"level,omitempty"`
	Mode    string `json:"mode,omitempty"`
	Message string `json:"message,omitempty"`
}

type tapResult struct {
	document        cardcomponent.Document
	gameState       GameState
	appliedFragment any
	library         []cardcomponent.LibraryItem
	events          []CardEvent
}

type colorControlRequest struct {
	Color          string
	SecondaryColor string
	Gradient       bool
	Angle          int
}

func initialGameState() GameState {
	return GameState{
		Level:           1,
		UnlockedTargets: []string{background.Type, border.Type},
		UnlockedModes:   []string{editModeRandom},
		TargetProgress: map[string]TargetProgress{
			background.Type: {Level: 1, UnlockedModes: []string{editModeRandom}},
			border.Type:     {Level: 1, UnlockedModes: []string{editModeRandom}},
		},
	}
}

func normalizeGameState(state GameState) GameState {
	if state.Level < 1 {
		state.Level = 1
	}
	if len(state.UnlockedTargets) == 0 {
		state.UnlockedTargets = []string{background.Type, border.Type}
	}
	if len(state.UnlockedModes) == 0 {
		state.UnlockedModes = []string{editModeRandom}
	}
	if state.TargetProgress == nil {
		state.TargetProgress = map[string]TargetProgress{}
	}
	for _, target := range state.UnlockedTargets {
		progress := state.TargetProgress[target]
		if progress.Level < 1 {
			progress.Level = 1
		}
		if len(progress.UnlockedModes) == 0 {
			progress.UnlockedModes = []string{editModeRandom}
		}
		state.TargetProgress[target] = progress
	}
	if state.Level >= simpleControlsLevel {
		state = unlockSimpleControls(state)
	}
	return state
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

func isKnownTapTarget(target string) bool {
	switch target {
	case background.Type, border.Type, cardcomponent.TypeTextarea:
		return true
	default:
		return false
	}
}

func targetUnlocked(state GameState, target string) bool {
	for _, candidate := range state.UnlockedTargets {
		if candidate == target {
			return true
		}
	}
	return false
}

func modeUnlocked(state GameState, target, mode string) bool {
	state = normalizeGameState(state)
	progress := state.TargetProgress[target]
	for _, candidate := range progress.UnlockedModes {
		if candidate == mode {
			return true
		}
	}
	return false
}

func randomGeneratedFragment(target string, seed int64, level int) (json.RawMessage, error) {
	var value any
	switch target {
	case background.Type:
		value = background.RandomGenerated(seed, level)
	case border.Type:
		value = border.RandomGenerated(seed, level)
	default:
		return nil, fmt.Errorf("target %q does not support random fragments", target)
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func advanceGameState(state GameState, target string) (GameState, []CardEvent) {
	state = normalizeGameState(state)
	oldLevel := state.Level
	hadSimpleControls := modeUnlocked(state, background.Type, editModeSimpleControls) && modeUnlocked(state, border.Type, editModeSimpleControls)
	state.TapCount++
	state.XP += xpPerTap
	state.Level = state.XP/xpPerLevel + 1

	progress := state.TargetProgress[target]
	if progress.Level < 1 {
		progress.Level = 1
	}
	if len(progress.UnlockedModes) == 0 {
		progress.UnlockedModes = []string{editModeRandom}
	}
	progress.Taps++
	progress.Level = progress.Taps/3 + 1
	state.TargetProgress[target] = progress

	events := []CardEvent{
		{Type: "fragmentApplied", Target: target},
		{Type: "xpGained", Amount: xpPerTap},
	}
	if state.Level > oldLevel {
		events = append(events, CardEvent{Type: "levelUp", Level: state.Level})
	}
	if !hadSimpleControls && state.Level >= simpleControlsLevel {
		state = unlockSimpleControls(state)
		events = append(events,
			CardEvent{Type: "modeUnlocked", Target: background.Type, Mode: editModeSimpleControls},
			CardEvent{Type: "modeUnlocked", Target: border.Type, Mode: editModeSimpleControls},
		)
	}
	return state, events
}

func tapSeed(state GameState, target string) int64 {
	var targetOffset int64
	for _, char := range target {
		targetOffset += int64(char)
	}
	return time.Now().UnixNano() + int64(state.TapCount+1)*7919 + targetOffset
}

func unlockSimpleControls(state GameState) GameState {
	state.UnlockedModes = appendStringOnce(state.UnlockedModes, editModeSimpleControls)
	for _, target := range []string{background.Type, border.Type} {
		progress := state.TargetProgress[target]
		if progress.Level < 1 {
			progress.Level = 1
		}
		progress.UnlockedModes = appendStringOnce(progress.UnlockedModes, editModeRandom)
		progress.UnlockedModes = appendStringOnce(progress.UnlockedModes, editModeSimpleControls)
		state.TargetProgress[target] = progress
	}
	return state
}

func appendStringOnce(values []string, value string) []string {
	for _, candidate := range values {
		if candidate == value {
			return values
		}
	}
	return append(values, value)
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
		node := findNodeByType(document.Root, target)
		part := background.DefaultFragment()
		if node != nil && len(node.Fragment) > 0 {
			if err := json.Unmarshal(node.Fragment, &part); err != nil {
				return nil, fmt.Errorf("decode current background fragment: %w", err)
			}
		}
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
		node := findNodeByType(document.Root, target)
		part := border.DefaultFragment()
		if node != nil && len(node.Fragment) > 0 {
			if err := json.Unmarshal(node.Fragment, &part); err != nil {
				return nil, fmt.Errorf("decode current border fragment: %w", err)
			}
		}
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
