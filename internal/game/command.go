package game

import "encoding/json"

type CommandKind string

const (
	CommandReset                  CommandKind = "reset"
	CommandCycleCard              CommandKind = "cycleCard"
	CommandCollectCard            CommandKind = "collectCard"
	CommandPlayCard               CommandKind = "playCard"
	CommandSubmitForm             CommandKind = "submitForm"
	CommandSelectWorldComponent   CommandKind = "selectWorldComponent"
	CommandChangeWorldComponent   CommandKind = "changeWorldComponent"
	CommandStartEditing           CommandKind = "startEditing"
	CommandInstallEditComponent   CommandKind = "installEditComponent"
	CommandSelectEditComponent    CommandKind = "selectEditComponent"
	CommandChangeEditComponent    CommandKind = "changeEditComponent"
	CommandChangeLibraryComponent CommandKind = "changeLibraryComponent"
	CommandSaveEdit               CommandKind = "saveEdit"
	CommandCancelEdit             CommandKind = "cancelEdit"
)

type Command interface {
	Kind() CommandKind
	isCommand()
}

type ResetCommand struct{}

func (ResetCommand) Kind() CommandKind { return CommandReset }
func (ResetCommand) isCommand()        {}

type CycleCardCommand struct {
	Direction string
}

func (CycleCardCommand) Kind() CommandKind { return CommandCycleCard }
func (CycleCardCommand) isCommand()        {}

type CollectCardCommand struct {
	CardID string
}

func (CollectCardCommand) Kind() CommandKind { return CommandCollectCard }
func (CollectCardCommand) isCommand()        {}

type PlayCardCommand struct {
	SourceCardID string
	TargetCardID string
}

func (PlayCardCommand) Kind() CommandKind { return CommandPlayCard }
func (PlayCardCommand) isCommand()        {}

type SubmitFormCommand struct {
	CardID string
	FormID string
	Fields map[string]string
}

func (SubmitFormCommand) Kind() CommandKind { return CommandSubmitForm }
func (SubmitFormCommand) isCommand()        {}

type SelectWorldComponentCommand struct {
	CardID        string
	ComponentID   string
	ComponentKind string
}

func (SelectWorldComponentCommand) Kind() CommandKind { return CommandSelectWorldComponent }
func (SelectWorldComponentCommand) isCommand()        {}

type ChangeWorldComponentCommand struct {
	CardID        string
	ComponentID   string
	ComponentKind string
	Control       string
	Value         json.RawMessage
}

func (ChangeWorldComponentCommand) Kind() CommandKind { return CommandChangeWorldComponent }
func (ChangeWorldComponentCommand) isCommand()        {}

type StartEditingCommand struct {
	CardID string
}

func (StartEditingCommand) Kind() CommandKind { return CommandStartEditing }
func (StartEditingCommand) isCommand()        {}

type InstallEditComponentCommand struct {
	ComponentCardID string
}

func (InstallEditComponentCommand) Kind() CommandKind { return CommandInstallEditComponent }
func (InstallEditComponentCommand) isCommand()        {}

type SelectEditComponentCommand struct {
	ComponentID   string
	ComponentKind string
}

func (SelectEditComponentCommand) Kind() CommandKind { return CommandSelectEditComponent }
func (SelectEditComponentCommand) isCommand()        {}

type ChangeEditComponentCommand struct {
	ComponentID string
	Control     string
	Value       json.RawMessage
}

func (ChangeEditComponentCommand) Kind() CommandKind { return CommandChangeEditComponent }
func (ChangeEditComponentCommand) isCommand()        {}

type ChangeLibraryComponentCommand struct {
	CardID        string
	ComponentID   string
	ComponentKind string
	Control       string
	Value         json.RawMessage
}

func (ChangeLibraryComponentCommand) Kind() CommandKind { return CommandChangeLibraryComponent }
func (ChangeLibraryComponentCommand) isCommand()        {}

type SaveEditCommand struct{}

func (SaveEditCommand) Kind() CommandKind { return CommandSaveEdit }
func (SaveEditCommand) isCommand()        {}

type CancelEditCommand struct{}

func (CancelEditCommand) Kind() CommandKind { return CommandCancelEdit }
func (CancelEditCommand) isCommand()        {}
