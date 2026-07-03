package web

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/n0remac/Living-Card/internal/components/background"
	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

const (
	editModeRandom = "random"
	xpPerTap       = 1
	xpPerLevel     = 5
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
	return state, events
}

func tapSeed(state GameState, target string) int64 {
	var targetOffset int64
	for _, char := range target {
		targetOffset += int64(char)
	}
	return time.Now().UnixNano() + int64(state.TapCount+1)*7919 + targetOffset
}
