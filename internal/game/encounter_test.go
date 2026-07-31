package game

import (
	"testing"

	"github.com/n0remac/Living-Card/internal/components/catalog"
)

func TestSeededEncounterSnapshotHydratesParticipantsAndActors(t *testing.T) {
	session := NewSession(catalog.MustNew())
	result, err := session.View()
	if err != nil {
		t.Fatal(err)
	}
	encounter := result.Snapshot.Encounter
	if encounter == nil {
		t.Fatal("seeded encounter is missing")
	}
	if encounter.ID != "cell-lockdown" || encounter.Phase != EncounterPhaseActive ||
		encounter.Pressure != 0 || encounter.MaxPressure != 4 {
		t.Fatalf("encounter = %#v", encounter)
	}
	if len(encounter.Participants) != 5 {
		t.Fatalf("participant count = %d", len(encounter.Participants))
	}
	door := encounter.Participants[0]
	if door.Role != EncounterRoleHostile || door.Card.ID != "rusted-cell-door" || door.Card.Actor == nil {
		t.Fatalf("door participant = %#v", door)
	}
	if alert, ok := door.Card.Actor.Tracks["alert"]; !ok || alert.Current != 1 || alert.Max != 4 {
		t.Fatalf("door alert track = %#v", door.Card.Actor.Tracks)
	}
}

func TestEncounterPressureAdvancesAfterUnansweredAction(t *testing.T) {
	session := NewSession(catalog.MustNew())
	mustExecuteResult(t, session, CollectCardCommand{CardID: "bent-iron-key"})
	result := mustExecuteResult(t, session, PlayCardCommand{
		SourceCardID: "bent-iron-key",
		TargetCardID: "sleeping-switch",
	})
	if result.Snapshot.Encounter == nil || result.Snapshot.Encounter.Pressure != 1 {
		t.Fatalf("encounter = %#v", result.Snapshot.Encounter)
	}
	if !hasEventType(result.Events, EventEncounterPressureChanged) {
		t.Fatalf("events = %#v", result.Events)
	}
}

func TestEncounterPressureCanOverwhelmPlayer(t *testing.T) {
	session := NewSession(catalog.MustNew())
	mustExecuteResult(t, session, CollectCardCommand{CardID: "bent-iron-key"})
	var result Result
	for range 4 {
		result = mustExecuteResult(t, session, PlayCardCommand{
			SourceCardID: "bent-iron-key",
			TargetCardID: "sleeping-switch",
		})
	}
	if result.Snapshot.Encounter == nil ||
		result.Snapshot.Encounter.Phase != EncounterPhaseResolved ||
		result.Snapshot.Encounter.Outcome != "overwhelmed" {
		t.Fatalf("encounter = %#v", result.Snapshot.Encounter)
	}
}

func TestUnlockingDoorResolvesEncounterAndKeepsFinalSnapshotRenderable(t *testing.T) {
	session := NewSession(catalog.MustNew())
	mustExecuteResult(t, session, CollectCardCommand{CardID: "bent-iron-key"})
	result := mustExecuteResult(t, session, PlayCardCommand{
		SourceCardID: "bent-iron-key",
		TargetCardID: "rusted-cell-door",
	})
	encounter := result.Snapshot.Encounter
	if encounter == nil || encounter.Phase != EncounterPhaseResolved || encounter.Outcome != "escaped" {
		t.Fatalf("encounter = %#v", encounter)
	}
	if !hasEventType(result.Events, EventEncounterResolved) {
		t.Fatalf("events = %#v", result.Events)
	}
	if len(encounter.Participants) != 5 {
		t.Fatalf("resolved participant count = %d", len(encounter.Participants))
	}
}

func TestPhotographPersuadesDoorWithoutCombat(t *testing.T) {
	session := NewSession(catalog.MustNew())
	mustExecuteResult(t, session, CollectCardCommand{CardID: "faded-photograph"})
	result := mustExecuteResult(t, session, PlayCardCommand{
		SourceCardID: "faded-photograph",
		TargetCardID: "rusted-cell-door",
	})
	if result.Snapshot.Encounter == nil || result.Snapshot.Encounter.Outcome != "persuaded" {
		t.Fatalf("encounter = %#v", result.Snapshot.Encounter)
	}
	door := session.instances["rusted-cell-door"]
	if door.Actor == nil || door.Actor.Disposition != DispositionFriendly {
		t.Fatalf("door actor = %#v", door.Actor)
	}
	if !stringInSlice(door.Actor.Statuses, "remembered") || stringInSlice(door.Actor.Statuses, "locked") {
		t.Fatalf("door statuses = %#v", door.Actor.Statuses)
	}
}
