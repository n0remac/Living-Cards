package game

import (
	"errors"
	"fmt"
	"reflect"
)

type InvalidCommandError struct {
	Err error
}

func (e *InvalidCommandError) Error() string { return e.Err.Error() }
func (e *InvalidCommandError) Unwrap() error { return e.Err }

type executionError struct {
	err error
}

func (e *executionError) Error() string { return e.err.Error() }
func (e *executionError) Unwrap() error { return e.err }

func invalidCommand(err error) error {
	if err == nil {
		return nil
	}
	return &InvalidCommandError{Err: err}
}

func failExecution(err error) error {
	if err == nil {
		return nil
	}
	return &executionError{err: err}
}

func IsInvalidCommand(err error) bool {
	var target *InvalidCommandError
	return errors.As(err, &target)
}

func (s *Session) View() (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.snapshotLocked()
	if err != nil {
		return Result{}, err
	}
	return Result{Revision: s.revision, Snapshot: snapshot, Events: []Event{}}, nil
}

func (s *Session) Execute(command Command) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if command == nil {
		return Result{}, invalidCommand(fmt.Errorf("command is required"))
	}
	commandValue := reflect.ValueOf(command)
	if commandValue.Kind() == reflect.Pointer {
		if commandValue.IsNil() {
			return Result{}, invalidCommand(fmt.Errorf("command is required"))
		}
		if dereferenced, ok := commandValue.Elem().Interface().(Command); ok {
			command = dereferenced
		}
	}
	previous := s.stateLocked()
	working, err := cloneSessionState(previous)
	if err != nil {
		return Result{}, err
	}
	s.restoreStateLocked(working)
	collector := &eventCollector{}
	snapshot, err := s.executeLocked(command, collector)
	if err != nil {
		s.restoreStateLocked(previous)
		var internal *executionError
		if errors.As(err, &internal) {
			return Result{}, internal.err
		}
		return Result{}, invalidCommand(err)
	}
	if err := s.validateZoneState(); err != nil {
		s.restoreStateLocked(previous)
		return Result{}, fmt.Errorf("validate session state: %w", err)
	}
	s.revision++
	return Result{
		Revision: s.revision,
		Snapshot: snapshot,
		Events:   append([]Event(nil), collector.events...),
	}, nil
}

func (s *Session) executeLocked(command Command, events *eventCollector) (Snapshot, error) {
	switch typed := command.(type) {
	case ResetCommand:
		return s.resetLocked(events)
	case CycleCardCommand:
		return s.cycleLocked(typed.Direction, events)
	case CollectCardCommand:
		return s.collectLocked(string(typed.CardID), events)
	case PlayCardCommand:
		return s.playCardLocked(string(typed.SourceCardID), string(typed.TargetCardID), events)
	case SubmitFormCommand:
		return s.submitFormLocked(string(typed.CardID), typed.FormID, typed.Fields, events)
	case SelectWorldComponentCommand:
		return s.selectWorldComponentLocked(string(typed.CardID), typed.ComponentID, typed.ComponentKind, events)
	case ChangeWorldComponentCommand:
		return s.changeWorldComponentLocked(string(typed.CardID), typed.ComponentID, typed.ComponentKind, typed.Control, typed.Value, events)
	case StartEditingCommand:
		return s.startEditingLocked(string(typed.CardID), events)
	case InstallEditComponentCommand:
		return s.installEditComponentLocked(string(typed.ComponentCardID), events)
	case SelectEditComponentCommand:
		return s.selectEditComponentLocked(typed.ComponentID, typed.ComponentKind, events)
	case ChangeEditComponentCommand:
		return s.changeEditComponentLocked(typed.ComponentID, typed.Control, typed.Value, events)
	case ChangeLibraryComponentCommand:
		return s.changeLibraryComponentLocked(string(typed.CardID), typed.ComponentID, typed.ComponentKind, typed.Control, typed.Value, events)
	case SaveEditCommand:
		return s.saveEditLocked(events)
	case CancelEditCommand:
		return s.cancelEditLocked(events)
	default:
		return Snapshot{}, fmt.Errorf("unsupported command kind %q", command.Kind())
	}
}
