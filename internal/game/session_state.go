package game

import (
	"encoding/json"
	"fmt"
)

type sessionState struct {
	cardDefinitions          map[CardDefinitionID]CardDefinition
	loadedDecks              map[string]bool
	rules                    []RuleDefinition
	instances                map[CardInstanceID]CardInstance
	zones                    ZoneState
	activeSceneCardID        CardInstanceID
	activeEditingComponentID string
	editSession              *editSessionState
	encounter                *EncounterState
	solvedFlags              map[string]bool
	lastMessage              string
}

func (s *Session) stateLocked() sessionState {
	return sessionState{
		cardDefinitions:          s.cardDefinitions,
		loadedDecks:              s.loadedDecks,
		rules:                    s.rules,
		instances:                s.instances,
		zones:                    s.zones,
		activeSceneCardID:        s.activeSceneCardID,
		activeEditingComponentID: s.activeEditingComponentID,
		editSession:              s.editSession,
		encounter:                s.encounter,
		solvedFlags:              s.solvedFlags,
		lastMessage:              s.lastMessage,
	}
}

func cloneSessionState(state sessionState) (sessionState, error) {
	cardDefinitions, err := cloneSessionValue(state.cardDefinitions)
	if err != nil {
		return sessionState{}, err
	}
	loadedDecks, err := cloneSessionValue(state.loadedDecks)
	if err != nil {
		return sessionState{}, err
	}
	rules, err := cloneSessionValue(state.rules)
	if err != nil {
		return sessionState{}, err
	}
	instances, err := cloneSessionValue(state.instances)
	if err != nil {
		return sessionState{}, err
	}
	zones, err := cloneSessionValue(state.zones)
	if err != nil {
		return sessionState{}, err
	}
	editSession, err := cloneSessionValue(state.editSession)
	if err != nil {
		return sessionState{}, err
	}
	encounter, err := cloneSessionValue(state.encounter)
	if err != nil {
		return sessionState{}, err
	}
	solvedFlags, err := cloneSessionValue(state.solvedFlags)
	if err != nil {
		return sessionState{}, err
	}
	return sessionState{
		cardDefinitions:          cardDefinitions,
		loadedDecks:              loadedDecks,
		rules:                    rules,
		instances:                instances,
		zones:                    zones,
		activeSceneCardID:        state.activeSceneCardID,
		activeEditingComponentID: state.activeEditingComponentID,
		editSession:              editSession,
		encounter:                encounter,
		solvedFlags:              solvedFlags,
		lastMessage:              state.lastMessage,
	}, nil
}

func cloneSessionValue[T any](value T) (T, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("clone session state: %w", err)
	}
	var cloned T
	if err := json.Unmarshal(raw, &cloned); err != nil {
		var zero T
		return zero, fmt.Errorf("clone session state: %w", err)
	}
	return cloned, nil
}

func (s *Session) restoreStateLocked(state sessionState) {
	s.cardDefinitions = state.cardDefinitions
	s.loadedDecks = state.loadedDecks
	s.rules = state.rules
	s.instances = state.instances
	s.zones = state.zones
	s.activeSceneCardID = state.activeSceneCardID
	s.activeEditingComponentID = state.activeEditingComponentID
	s.editSession = state.editSession
	s.encounter = state.encounter
	s.solvedFlags = state.solvedFlags
	s.lastMessage = state.lastMessage
}
