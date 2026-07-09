package game

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/border"
	cardcomponent "github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/slider"
	"github.com/n0remac/Living-Card/internal/design"
)

func TestSessionStartsWithEmptyLibraryAndScriptedDeck(t *testing.T) {
	t.Parallel()

	snapshot := mustResult(t, NewSession().Snapshot)
	if len(snapshot.Library) != 0 {
		t.Fatalf("library = %#v, want empty", snapshot.Library)
	}
	if len(snapshot.WorldDeck) < 5 {
		t.Fatalf("world deck length = %d, want scripted deck", len(snapshot.WorldDeck))
	}
	if snapshot.ActiveWorldCard.ID != "rusted-cell-door" {
		t.Fatalf("active card = %#v, want rusted-cell-door", snapshot.ActiveWorldCard)
	}
	if snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want locked door", snapshot.SolvedFlags)
	}
	if !documentContains(snapshot.ActiveWorldCard.Document, "LOCKED") {
		t.Fatalf("active door document should be locked: %#v", snapshot.ActiveWorldCard.Document)
	}
}

func TestEmbeddedSeededWorldDeckLoadsAndValidates(t *testing.T) {
	t.Parallel()

	definition, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		t.Fatalf("LoadEmbeddedSeededWorldDeck() error = %v", err)
	}
	if definition.ID != SeededWorldDeckDefinition {
		t.Fatalf("deck id = %q, want %q", definition.ID, SeededWorldDeckDefinition)
	}
	session, err := NewSessionFromDeck(definition)
	if err != nil {
		t.Fatalf("NewSessionFromDeck() error = %v", err)
	}
	snapshot := mustResult(t, session.Snapshot)
	key := findCard(snapshot.WorldDeck, "bent-iron-key")
	if key == nil {
		t.Fatal("world deck missing key")
	}
	if !documentContains(key.Document, "BENT IRON KEY") {
		t.Fatalf("key document should come from deck data: %#v", key.Document)
	}
}

func TestEmbeddedDeckRegistryLoadsPuzzlePacks(t *testing.T) {
	t.Parallel()

	seeded := mustLoadEmbeddedDeck(t, SeededWorldDeckDefinition)
	if err := ValidateDeckDefinition(seeded); err != nil {
		t.Fatalf("ValidateDeckDefinition(seed) error = %v", err)
	}
	fuseRoom := mustLoadEmbeddedDeck(t, FuseRoomDeckDefinition)
	if err := ValidateDeckPackDefinition(fuseRoom, definitionsByID(seeded.Cards)); err != nil {
		t.Fatalf("ValidateDeckPackDefinition(fuse_room) error = %v", err)
	}
	generatorRoom := mustLoadEmbeddedDeck(t, GeneratorDeckDefinition)
	combined := definitionsByID(seeded.Cards)
	for cardID, card := range definitionsByID(fuseRoom.Cards) {
		combined[cardID] = card
	}
	if err := ValidateDeckPackDefinition(generatorRoom, combined); err != nil {
		t.Fatalf("ValidateDeckPackDefinition(generator_room) error = %v", err)
	}
}

func TestSessionCycleWrapsPreviousAndNext(t *testing.T) {
	t.Parallel()

	session := NewSession()
	previous := mustResult(t, func() (Snapshot, error) {
		return session.Cycle("previous")
	})
	if previous.ActiveIndex != len(previous.WorldDeck)-1 || previous.ActiveWorldCard.ID != "sleeping-switch" {
		t.Fatalf("previous snapshot = %#v", previous)
	}
	next := mustResult(t, func() (Snapshot, error) {
		return session.Cycle("next")
	})
	if next.ActiveIndex != 0 || next.ActiveWorldCard.ID != "rusted-cell-door" {
		t.Fatalf("next snapshot = %#v", next)
	}
}

func TestSessionCollectsKeyIntoLibrary(t *testing.T) {
	t.Parallel()

	snapshot := mustResult(t, func() (Snapshot, error) {
		return NewSession().Collect("bent-iron-key")
	})
	if len(snapshot.Library) != 1 || snapshot.Library[0].ID != "bent-iron-key" {
		t.Fatalf("library = %#v, want collected key", snapshot.Library)
	}
	key := findCard(snapshot.WorldDeck, "bent-iron-key")
	if key == nil {
		t.Fatal("world deck missing key")
	}
	if key.Collectible || !key.Collected {
		t.Fatalf("world key = %#v, want collected and non-collectible", *key)
	}
}

func TestSessionRejectsDecoyCollection(t *testing.T) {
	t.Parallel()

	if _, err := NewSession().Collect("inventory-label"); err == nil {
		t.Fatal("Collect() decoy error = nil, want error")
	}
}

func TestSessionWrongCardUseDoesNotUnlockDoor(t *testing.T) {
	t.Parallel()

	session := NewSession()
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "faded-photograph")
	})
	if snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want locked door", snapshot.SolvedFlags)
	}
	if !strings.Contains(snapshot.Message, "Nothing on this card responds") {
		t.Fatalf("message = %q, want wrong-card message", snapshot.Message)
	}
}

func TestSessionStartsWithoutPuzzlePackCards(t *testing.T) {
	t.Parallel()

	snapshot := mustResult(t, NewSession().Snapshot)
	if findCard(snapshot.WorldDeck, "glass-fuse") != nil {
		t.Fatalf("glass fuse should not be present at session start: %#v", snapshot.WorldDeck)
	}
	if findCard(snapshot.WorldDeck, "generator-panel") != nil {
		t.Fatalf("generator panel should not be present at session start: %#v", snapshot.WorldDeck)
	}
}

func TestSessionKeyUnlocksDoorWithEffect(t *testing.T) {
	t.Parallel()

	session := NewSession()
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	if !snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want door unlocked", snapshot.SolvedFlags)
	}
	if len(snapshot.Library) != 1 || snapshot.Library[0].ID != "bent-iron-key" {
		t.Fatalf("library = %#v, key should remain visible", snapshot.Library)
	}
	door := findCard(snapshot.WorldDeck, "rusted-cell-door")
	if door == nil {
		t.Fatal("world deck missing door")
	}
	if locked, _ := door.State["locked"].(bool); locked {
		t.Fatalf("door state = %#v, want unlocked", door.State)
	}
	if hasTag(*door, "locked") {
		t.Fatalf("door tags = %#v, want locked tag removed", door.Tags)
	}
	if !documentContains(door.Document, "OPEN") {
		t.Fatalf("door document did not switch to open variant: %#v", door.Document)
	}
	if snapshot.ActiveWorldCard.ID != "fuse-box-note" {
		t.Fatalf("active card = %q, want first fuse room card", snapshot.ActiveWorldCard.ID)
	}
	if findCard(snapshot.WorldDeck, "glass-fuse") == nil {
		t.Fatal("door unlock should load fuse room cards")
	}
}

func TestSessionFusePowersSwitchAndLoadsGeneratorRoom(t *testing.T) {
	t.Parallel()

	session := NewSession()
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("glass-fuse")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("glass-fuse", "sleeping-switch")
	})
	if !snapshot.SolvedFlags["switchPowered"] {
		t.Fatalf("solved flags = %#v, want switchPowered", snapshot.SolvedFlags)
	}
	if snapshot.ActiveWorldCard.ID != "generator-panel" {
		t.Fatalf("active card = %q, want generator-panel", snapshot.ActiveWorldCard.ID)
	}
	switchCard := findCard(snapshot.WorldDeck, "sleeping-switch")
	if switchCard == nil {
		t.Fatal("world deck missing switch")
	}
	if powered, _ := switchCard.State["powered"].(bool); !powered {
		t.Fatalf("switch state = %#v, want powered", switchCard.State)
	}
	if hasTag(*switchCard, "decoy") {
		t.Fatalf("switch tags = %#v, want decoy tag removed", switchCard.Tags)
	}
	if !documentContains(switchCard.Document, "SWITCH ONLINE") {
		t.Fatalf("switch document did not switch to powered variant: %#v", switchCard.Document)
	}
	if findCard(snapshot.WorldDeck, "numbered-gauge") == nil {
		t.Fatal("generator room cards were not loaded")
	}
}

func TestSessionSliderControllerWrongValueDoesNotPowerGenerator(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.SaveController(BlankControllerCardID, controllerDraftDocument(t, 72))
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard(RegulatorControllerCardID, "generator-panel")
	})
	if snapshot.SolvedFlags[GeneratorPoweredFlag] {
		t.Fatalf("solved flags = %#v, want generator unpowered", snapshot.SolvedFlags)
	}
	if !strings.Contains(snapshot.Message, "regulator value is wrong") {
		t.Fatalf("message = %q, want wrong value message", snapshot.Message)
	}
	generator := findCard(snapshot.WorldDeck, "generator-panel")
	if generator == nil {
		t.Fatal("world deck missing generator")
	}
	if powered, _ := generator.State["powered"].(bool); powered {
		t.Fatalf("generator state = %#v, want unpowered", generator.State)
	}
	if !documentContains(generator.Document, "SLEEPING GENERATOR") {
		t.Fatalf("generator document should remain inactive: %#v", generator.Document)
	}
}

func TestSessionSliderControllerPowersGenerator(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	saved := mustResult(t, func() (Snapshot, error) {
		return session.SaveController(BlankControllerCardID, controllerDraftDocument(t, 73))
	})
	if findCard(saved.Library, RegulatorControllerCardID) == nil {
		t.Fatalf("library = %#v, want saved regulator controller", saved.Library)
	}
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard(RegulatorControllerCardID, "generator-panel")
	})
	if !snapshot.SolvedFlags[GeneratorPoweredFlag] {
		t.Fatalf("solved flags = %#v, want generator powered", snapshot.SolvedFlags)
	}
	generator := findCard(snapshot.WorldDeck, "generator-panel")
	if generator == nil {
		t.Fatal("world deck missing generator")
	}
	if powered, _ := generator.State["powered"].(bool); !powered {
		t.Fatalf("generator state = %#v, want powered", generator.State)
	}
	if hasTag(*generator, "inactive") {
		t.Fatalf("generator tags = %#v, want inactive tag removed", generator.Tags)
	}
	if !documentContains(generator.Document, "GENERATOR ONLINE") {
		t.Fatalf("generator document did not switch to active variant: %#v", generator.Document)
	}
}

func TestSessionEditWorkflowBuildsAndUsesSliderController(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	started := mustResult(t, func() (Snapshot, error) {
		return session.StartEdit(BlankControllerCardID)
	})
	if started.EditSession == nil || started.EditSession.TargetCardID != BlankControllerCardID || started.EditSession.DraftCard.ID != BlankControllerCardID {
		t.Fatalf("edit snapshot = %#v, want blank controller draft", started)
	}
	installed := mustResult(t, func() (Snapshot, error) {
		return session.InstallEditComponent(SliderComponentCardID)
	})
	if findCard(installed.Library, SliderComponentCardID) == nil {
		t.Fatalf("library = %#v, slider component should not be consumed before save", installed.Library)
	}
	if installed.EditSession == nil || !stringInSlice(installed.EditSession.PendingConsumedComponentIDs, SliderComponentCardID) {
		t.Fatalf("edit session = %#v, want pending slider consumption", installed.EditSession)
	}
	if !documentContains(installed.EditSession.DraftCard.Document, `"componentKind":"slider"`) {
		t.Fatalf("draft card missing slider: %#v", installed.EditSession.DraftCard)
	}
	tuned := mustResult(t, func() (Snapshot, error) {
		return session.ApplyEditControl("regulator-output-slider", "value", json.RawMessage("73"))
	})
	tuned = mustResult(t, func() (Snapshot, error) {
		return session.ApplyEditControl("regulator-output-slider", "position", json.RawMessage(`{"x":42,"y":64}`))
	})
	if tuned.EditSession == nil ||
		!documentContains(tuned.EditSession.DraftCard.Document, `"value":73`) ||
		!documentContains(tuned.EditSession.DraftCard.Document, `"x":42`) ||
		!documentContains(tuned.EditSession.DraftCard.Document, `"y":64`) {
		t.Fatalf("tuned draft card = %#v, want slider value 73 and dragged position", tuned.EditSession)
	}
	saved := mustResult(t, session.SaveEdit)
	if saved.EditSession != nil {
		t.Fatalf("edit session = %#v, want cleared after save", saved.EditSession)
	}
	if findCard(saved.Library, SliderComponentCardID) != nil {
		t.Fatalf("library = %#v, slider component should be consumed on save", saved.Library)
	}
	controller := findCard(saved.Library, BlankControllerCardID)
	if controller == nil {
		t.Fatalf("library = %#v, want edited blank controller", saved.Library)
	}
	if controller.Name != "Regulator Controller" || !hasTag(*controller, "controller") || !documentContains(controller.Document, `"componentKind":"slider"`) {
		t.Fatalf("controller = %#v, want named controller with slider", controller)
	}
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard(BlankControllerCardID, "generator-panel")
	})
	if !snapshot.SolvedFlags[GeneratorPoweredFlag] {
		t.Fatalf("solved flags = %#v, want generator powered", snapshot.SolvedFlags)
	}
	if generator := findCard(snapshot.WorldDeck, "generator-panel"); generator == nil || !documentContains(generator.Document, "GENERATOR ONLINE") {
		t.Fatalf("generator should be online: %#v", generator)
	}
}

func TestSessionSelectsEditSliderAndBorderAfterOverlayClose(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("border-component")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.StartEdit(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.InstallEditComponent(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.InstallEditComponent("border-component")
	})

	sliderSelected := mustResult(t, func() (Snapshot, error) {
		return session.SelectEditComponent("", slider.Kind)
	})
	if sliderSelected.EditSession == nil || sliderSelected.EditSession.SelectedComponentID != "regulator-output-slider" {
		t.Fatalf("selected slider = %#v, want regulator slider selected", sliderSelected.EditSession)
	}
	borderSelected := mustResult(t, func() (Snapshot, error) {
		return session.SelectEditComponent("", border.Kind)
	})
	if borderSelected.EditSession == nil || borderSelected.EditSession.SelectedComponentID != BlankControllerCardID+"-border" {
		t.Fatalf("selected border = %#v, want controller border selected", borderSelected.EditSession)
	}
}

func TestSessionLibrarySliderControlChangeRetunesSavedController(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.StartEdit(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.InstallEditComponent(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.ApplyEditControl("regulator-output-slider", "value", json.RawMessage("73"))
	})
	mustResult(t, session.SaveEdit)

	updated := mustResult(t, func() (Snapshot, error) {
		return session.ApplyLibraryComponentControl(BlankControllerCardID, "regulator-output-slider", slider.Kind, "value", json.RawMessage("72"))
	})
	controller := findCard(updated.Library, BlankControllerCardID)
	if controller == nil || !documentContains(controller.Document, `"value":72`) {
		t.Fatalf("controller = %#v, want library slider value 72", controller)
	}
}

func TestSessionWrongControllerMountsGeneratorSliderAndMountedTunePowersGenerator(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.StartEdit(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.InstallEditComponent(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.ApplyEditControl("regulator-output-slider", "value", json.RawMessage("72"))
	})
	mustResult(t, session.SaveEdit)

	mounted := mustResult(t, func() (Snapshot, error) {
		return session.UseCard(BlankControllerCardID, "generator-panel")
	})
	generator := findCard(mounted.WorldDeck, "generator-panel")
	if generator == nil || !documentContains(generator.Document, `"componentKind":"slider"`) || !documentContains(generator.Document, `"value":72`) {
		t.Fatalf("generator after wrong controller = %#v, want mounted wrong-valued slider", generator)
	}
	if mounted.SolvedFlags[GeneratorPoweredFlag] || !strings.Contains(mounted.Message, "regulator value is wrong") {
		t.Fatalf("mounted snapshot = %#v, want unpowered failure with mounted slider", mounted)
	}

	powered := mustResult(t, func() (Snapshot, error) {
		return session.ApplyWorldComponentControl("generator-panel", "regulator-output-slider", slider.Kind, "value", json.RawMessage("73"))
	})
	generator = findCard(powered.WorldDeck, "generator-panel")
	if generator == nil ||
		!powered.SolvedFlags[GeneratorPoweredFlag] ||
		!documentContains(generator.Document, "GENERATOR ONLINE") ||
		!documentContains(generator.Document, `"componentKind":"slider"`) ||
		!documentContains(generator.Document, `"value":73`) {
		t.Fatalf("powered generator = %#v snapshot=%#v, want online generator with tuned mounted slider", generator, powered)
	}
}

func TestSessionEditCancelLeavesLibraryUnchanged(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.StartEdit(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.InstallEditComponent(SliderComponentCardID)
	})
	canceled := mustResult(t, session.CancelEdit)
	if canceled.EditSession != nil {
		t.Fatalf("edit session = %#v, want cleared after cancel", canceled.EditSession)
	}
	if findCard(canceled.Library, SliderComponentCardID) == nil {
		t.Fatalf("library = %#v, slider component should remain after cancel", canceled.Library)
	}
	controller := findCard(canceled.Library, BlankControllerCardID)
	if controller == nil || controller.Name != "Blank Controller" || documentContains(controller.Document, `"componentKind":"slider"`) {
		t.Fatalf("controller = %#v, want unchanged blank controller", controller)
	}
}

func TestSessionActiveWorldComponentControlsMoveText(t *testing.T) {
	t.Parallel()

	session := NewSession()
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.ApplyWorldComponentControl("rusted-cell-door", "door-title", "textarea", "position", json.RawMessage(`{"x":23,"y":31}`))
	})
	if snapshot.ActiveEditingComponentID != "door-title" {
		t.Fatalf("active editing component = %q, want door-title", snapshot.ActiveEditingComponentID)
	}
	if snapshot.ActiveWorldCard.ID != "rusted-cell-door" {
		t.Fatalf("active card = %q, want rusted-cell-door", snapshot.ActiveWorldCard.ID)
	}
	if !documentContains(snapshot.ActiveWorldCard.Document, `"x":23`) || !documentContains(snapshot.ActiveWorldCard.Document, `"y":31`) {
		t.Fatalf("door title was not moved in active document: %#v", snapshot.ActiveWorldCard.Document)
	}
}

func TestSessionActiveWorldBorderRequiresCollectedComponentCard(t *testing.T) {
	t.Parallel()

	session := NewSession()
	if _, err := session.SelectWorldComponent("rusted-cell-door", "", border.Kind); err == nil || !strings.Contains(err.Error(), "require finding") {
		t.Fatalf("SelectWorldComponent(border) error = %v, want missing component card error", err)
	}

	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("border-component")
	})
	selected := mustResult(t, func() (Snapshot, error) {
		return session.SelectWorldComponent("generator-panel", "", border.Kind)
	})
	if selected.ActiveEditingComponentID != "generator-panel-border" {
		t.Fatalf("active editing component = %q, want generator-panel-border", selected.ActiveEditingComponentID)
	}
	updated := mustResult(t, func() (Snapshot, error) {
		return session.ApplyWorldComponentControl("generator-panel", "generator-panel-border", border.Kind, "border_color", json.RawMessage(`"#f43f5e"`))
	})
	generator := findCard(updated.WorldDeck, "generator-panel")
	if generator == nil {
		t.Fatal("world deck missing generator-panel")
	}
	if !documentContains(generator.Document, `"border_color":"#f43f5e"`) || !documentContains(generator.Document, `border: 2px solid #f43f5e`) {
		t.Fatalf("generator border did not update: %#v", generator.Document)
	}
}

func TestSessionPoweredGeneratorDoesNotAcceptWrongRetune(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.SaveController(BlankControllerCardID, controllerDraftDocument(t, 73))
	})
	mustResult(t, func() (Snapshot, error) {
		return session.UseCard(RegulatorControllerCardID, "generator-panel")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.SaveController(BlankControllerCardID, controllerDraftDocument(t, 12))
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard(RegulatorControllerCardID, "generator-panel")
	})
	if !snapshot.SolvedFlags[GeneratorPoweredFlag] {
		t.Fatalf("solved flags = %#v, want generator to stay powered", snapshot.SolvedFlags)
	}
	if strings.Contains(snapshot.Message, "regulator value is wrong") {
		t.Fatalf("message = %q, powered generator should not run inactive puzzle failure", snapshot.Message)
	}
	generator := findCard(snapshot.WorldDeck, "generator-panel")
	if generator == nil || !documentContains(generator.Document, "GENERATOR ONLINE") {
		t.Fatalf("generator should stay online: %#v", generator)
	}
}

func TestSessionResetClearsSavedController(t *testing.T) {
	t.Parallel()

	session := NewSession()
	loadGeneratorRoom(t, session)
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(BlankControllerCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect(SliderComponentCardID)
	})
	mustResult(t, func() (Snapshot, error) {
		return session.SaveController(BlankControllerCardID, controllerDraftDocument(t, 73))
	})
	reset := mustResult(t, session.Reset)
	if findCard(reset.Library, RegulatorControllerCardID) != nil || findCard(reset.WorldDeck, "generator-panel") != nil {
		t.Fatalf("reset snapshot = %#v, want seed-only world and no saved controller", reset)
	}
}

func TestSessionLoadedDecksAreIdempotentAndResetToSeed(t *testing.T) {
	t.Parallel()

	session := NewSession()
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	doorLoaded := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	if countCard(doorLoaded.WorldDeck, "glass-fuse") != 1 {
		t.Fatalf("glass-fuse count = %d, want 1", countCard(doorLoaded.WorldDeck, "glass-fuse"))
	}
	doorReplay := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	if len(doorReplay.WorldDeck) != len(doorLoaded.WorldDeck) || countCard(doorReplay.WorldDeck, "glass-fuse") != 1 {
		t.Fatalf("door replay duplicated loaded cards: before=%d after=%d fuse=%d", len(doorLoaded.WorldDeck), len(doorReplay.WorldDeck), countCard(doorReplay.WorldDeck, "glass-fuse"))
	}
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("glass-fuse")
	})
	generatorLoaded := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("glass-fuse", "sleeping-switch")
	})
	generatorReplay := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("glass-fuse", "sleeping-switch")
	})
	if len(generatorReplay.WorldDeck) != len(generatorLoaded.WorldDeck) || countCard(generatorReplay.WorldDeck, "generator-panel") != 1 {
		t.Fatalf("switch replay duplicated generator cards: before=%d after=%d generator=%d", len(generatorLoaded.WorldDeck), len(generatorReplay.WorldDeck), countCard(generatorReplay.WorldDeck, "generator-panel"))
	}
	reset := mustResult(t, session.Reset)
	if reset.ActiveWorldCard.ID != "rusted-cell-door" {
		t.Fatalf("reset active card = %q, want rusted-cell-door", reset.ActiveWorldCard.ID)
	}
	if findCard(reset.WorldDeck, "glass-fuse") != nil || findCard(reset.WorldDeck, "generator-panel") != nil {
		t.Fatalf("reset should remove loaded pack cards: %#v", reset.WorldDeck)
	}
}

func TestSessionUseRulesComeFromDeckData(t *testing.T) {
	t.Parallel()

	definition, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		t.Fatalf("LoadEmbeddedSeededWorldDeck() error = %v", err)
	}
	definition.UseRules = nil
	session, err := NewSessionFromDeck(definition)
	if err != nil {
		t.Fatalf("NewSessionFromDeck() error = %v", err)
	}
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	snapshot := mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	if snapshot.SolvedFlags[DoorUnlockedFlag] {
		t.Fatalf("solved flags = %#v, want locked door without data rule", snapshot.SolvedFlags)
	}
	door := findCard(snapshot.WorldDeck, "rusted-cell-door")
	if door == nil {
		t.Fatal("world deck missing door")
	}
	if !documentContains(door.Document, "LOCKED") {
		t.Fatalf("door document should remain locked without data rule: %#v", door.Document)
	}
}

func TestValidateDeckDefinitionRejectsInvalidFixtures(t *testing.T) {
	t.Parallel()

	base, err := LoadEmbeddedSeededWorldDeck()
	if err != nil {
		t.Fatalf("LoadEmbeddedSeededWorldDeck() error = %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*DeckDefinition)
	}{
		{
			name: "duplicate card id",
			mutate: func(definition *DeckDefinition) {
				definition.Cards = append(definition.Cards, definition.Cards[0])
			},
		},
		{
			name: "missing initial document variant",
			mutate: func(definition *DeckDefinition) {
				definition.Cards[0].InitialDocument = "missing"
			},
		},
		{
			name: "bad rule target reference",
			mutate: func(definition *DeckDefinition) {
				definition.UseRules[0].Target.ID = "missing-card"
			},
		},
		{
			name: "bad rule document variant reference",
			mutate: func(definition *DeckDefinition) {
				for index := range definition.UseRules[0].Effects {
					if definition.UseRules[0].Effects[index].EffectKind == EffectSetDocumentVariant {
						definition.UseRules[0].Effects[index].Variant = "missing"
					}
				}
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			definition := cloneValue(base)
			test.mutate(&definition)
			if err := ValidateDeckDefinition(definition); err == nil {
				t.Fatal("ValidateDeckDefinition() error = nil, want error")
			}
		})
	}
}

func TestDeckPackValidationRejectsBadPackReferences(t *testing.T) {
	t.Parallel()

	seeded := mustLoadEmbeddedDeck(t, SeededWorldDeckDefinition)
	fuseRoom := mustLoadEmbeddedDeck(t, FuseRoomDeckDefinition)
	if err := ValidateDeckPackDefinition(fuseRoom, nil); err == nil {
		t.Fatal("ValidateDeckPackDefinition() error = nil, want missing sleeping-switch reference")
	}
	duplicate := cloneValue(fuseRoom)
	duplicate.Cards[0].ID = "bent-iron-key"
	for variant, document := range duplicate.Cards[0].Documents {
		document.CardID = "bent-iron-key"
		duplicate.Cards[0].Documents[variant] = document
	}
	if err := ValidateDeckPackDefinition(duplicate, definitionsByID(seeded.Cards)); err == nil {
		t.Fatal("ValidateDeckPackDefinition() duplicate error = nil, want error")
	}
}

func TestDeckPackValidationRejectsUnsupportedComponentCondition(t *testing.T) {
	t.Parallel()

	seeded := mustLoadEmbeddedDeck(t, SeededWorldDeckDefinition)
	fuseRoom := mustLoadEmbeddedDeck(t, FuseRoomDeckDefinition)
	generatorRoom := mustLoadEmbeddedDeck(t, GeneratorDeckDefinition)
	combined := definitionsByID(seeded.Cards)
	for cardID, card := range definitionsByID(fuseRoom.Cards) {
		combined[cardID] = card
	}
	generatorRoom.UseRules[0].SourceComponentConditions[0].ComponentKind = "dial"
	if err := ValidateDeckPackDefinition(generatorRoom, combined); err == nil {
		t.Fatal("ValidateDeckPackDefinition() unsupported component condition error = nil, want error")
	}
}

func TestSessionLoadDeckEffectReturnsMissingDeckError(t *testing.T) {
	t.Parallel()

	definition := mustLoadEmbeddedDeck(t, SeededWorldDeckDefinition)
	for ruleIndex := range definition.UseRules {
		for effectIndex := range definition.UseRules[ruleIndex].Effects {
			if definition.UseRules[ruleIndex].Effects[effectIndex].EffectKind == EffectLoadDeck {
				definition.UseRules[ruleIndex].Effects[effectIndex].DeckID = "missing_pack"
			}
		}
	}
	session, err := NewSessionFromDeck(definition)
	if err != nil {
		t.Fatalf("NewSessionFromDeck() error = %v", err)
	}
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	if _, err := session.UseCard("bent-iron-key", "rusted-cell-door"); err == nil {
		t.Fatal("UseCard() missing loadDeck error = nil, want error")
	}
}

func loadGeneratorRoom(t *testing.T, session *Session) {
	t.Helper()

	mustResult(t, func() (Snapshot, error) {
		return session.Collect("bent-iron-key")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.UseCard("bent-iron-key", "rusted-cell-door")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.Collect("glass-fuse")
	})
	mustResult(t, func() (Snapshot, error) {
		return session.UseCard("glass-fuse", "sleeping-switch")
	})
}

func controllerDraftDocument(t *testing.T, value int) cardcomponent.Document {
	t.Helper()

	generated := design.GeneratedConfig[slider.Config]{
		ComponentKind: slider.Kind,
		Description:   "Draft slider",
		Config: slider.Config{
			Label: "Output",
			Min:   0,
			Max:   100,
			Step:  1,
			Value: value,
		},
	}
	slider.NormalizeGenerated(&generated)
	raw, err := json.Marshal(generated.Config)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return cardcomponent.Document{
		CardID: RegulatorControllerCardID,
		Name:   "Regulator Controller",
		Root: cardcomponent.Node{
			ID:            RegulatorControllerCardID + "-root",
			ComponentKind: cardcomponent.Kind,
			Children: []cardcomponent.Node{{
				ID:            "regulator-output-slider",
				ComponentKind: slider.Kind,
				Config:        raw,
			}},
		},
	}
}

func mustResult(t *testing.T, result func() (Snapshot, error)) Snapshot {
	t.Helper()
	snapshot, err := result()
	if err != nil {
		t.Fatalf("snapshot error = %v", err)
	}
	return snapshot
}

func findCard(cards []Card, id string) *Card {
	for index := range cards {
		if cards[index].ID == id {
			return &cards[index]
		}
	}
	return nil
}

func countCard(cards []Card, id string) int {
	count := 0
	for _, card := range cards {
		if card.ID == id {
			count++
		}
	}
	return count
}

func mustLoadEmbeddedDeck(t *testing.T, deckID string) DeckDefinition {
	t.Helper()
	definition, err := LoadEmbeddedDeck(deckID)
	if err != nil {
		t.Fatalf("LoadEmbeddedDeck(%q) error = %v", deckID, err)
	}
	return definition
}

func definitionsByID(cards []CardDefinition) map[string]CardDefinition {
	out := make(map[string]CardDefinition, len(cards))
	for _, card := range cards {
		out[card.ID] = card
	}
	return out
}

func documentContains(document any, marker string) bool {
	raw, err := json.Marshal(document)
	return err == nil && strings.Contains(string(raw), marker)
}
