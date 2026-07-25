package game

import (
	"encoding/json"
	"fmt"

	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
)

type sessionState struct {
	cardDefinitions          map[string]CardDefinition
	documentVariants         map[string]map[string]cardcomponent.Document
	loadedDecks              map[string]bool
	useRules                 []UseRuleDefinition
	formSubmitRules          []FormSubmitRuleDefinition
	worldDeck                []Card
	activeIndex              int
	activeEditingComponentID string
	library                  []Card
	editSession              *EditSession
	solvedFlags              map[string]bool
	lastMessage              string
}

func (s *Session) stateLocked() sessionState {
	return sessionState{
		cardDefinitions:          s.cardDefinitions,
		documentVariants:         s.documentVariants,
		loadedDecks:              s.loadedDecks,
		useRules:                 s.useRules,
		formSubmitRules:          s.formSubmitRules,
		worldDeck:                s.worldDeck,
		activeIndex:              s.activeIndex,
		activeEditingComponentID: s.activeEditingComponentID,
		library:                  s.library,
		editSession:              s.editSession,
		solvedFlags:              s.solvedFlags,
		lastMessage:              s.lastMessage,
	}
}

func cloneSessionState(state sessionState) (sessionState, error) {
	cardDefinitions, err := cloneSessionValue(state.cardDefinitions)
	if err != nil {
		return sessionState{}, err
	}
	documentVariants, err := cloneSessionValue(state.documentVariants)
	if err != nil {
		return sessionState{}, err
	}
	loadedDecks, err := cloneSessionValue(state.loadedDecks)
	if err != nil {
		return sessionState{}, err
	}
	useRules, err := cloneSessionValue(state.useRules)
	if err != nil {
		return sessionState{}, err
	}
	formSubmitRules, err := cloneSessionValue(state.formSubmitRules)
	if err != nil {
		return sessionState{}, err
	}
	worldDeck, err := cloneSessionValue(state.worldDeck)
	if err != nil {
		return sessionState{}, err
	}
	library, err := cloneSessionValue(state.library)
	if err != nil {
		return sessionState{}, err
	}
	editSession, err := cloneSessionValue(state.editSession)
	if err != nil {
		return sessionState{}, err
	}
	solvedFlags, err := cloneSessionValue(state.solvedFlags)
	if err != nil {
		return sessionState{}, err
	}
	return sessionState{
		cardDefinitions:          cardDefinitions,
		documentVariants:         documentVariants,
		loadedDecks:              loadedDecks,
		useRules:                 useRules,
		formSubmitRules:          formSubmitRules,
		worldDeck:                worldDeck,
		activeIndex:              state.activeIndex,
		activeEditingComponentID: state.activeEditingComponentID,
		library:                  library,
		editSession:              editSession,
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
	s.documentVariants = state.documentVariants
	s.loadedDecks = state.loadedDecks
	s.useRules = state.useRules
	s.formSubmitRules = state.formSubmitRules
	s.worldDeck = state.worldDeck
	s.activeIndex = state.activeIndex
	s.activeEditingComponentID = state.activeEditingComponentID
	s.library = state.library
	s.editSession = state.editSession
	s.solvedFlags = state.solvedFlags
	s.lastMessage = state.lastMessage
}
