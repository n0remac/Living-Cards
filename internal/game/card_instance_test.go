package game

import (
	"reflect"
	"testing"

	"github.com/n0remac/Living-Card/internal/components/card"
	"github.com/n0remac/Living-Card/internal/components/catalog"
)

func TestLegacyDeckMaterializesDeterministicInstanceIDsAndReset(t *testing.T) {
	definition := DeckDefinition{
		ID: "legacy-instances", Name: "Legacy Instances", InitialActiveCardID: "anchor",
		Cards: []CardDefinition{
			{ID: "anchor", Name: "Anchor", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("anchor")}},
			{ID: "item", Name: "Item", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("item")}},
		},
	}
	session, err := NewSessionFromDeck(catalog.MustNew(), definition)
	if err != nil {
		t.Fatal(err)
	}
	initial := mustViewResult(t, session)
	if got := cardIDs(initial.Snapshot.WorldDeck); !reflect.DeepEqual(got, []string{"anchor", "item"}) {
		t.Fatalf("legacy instance ids = %v", got)
	}
	for _, instanceID := range []CardInstanceID{"anchor", "item"} {
		instance := session.instances[instanceID]
		if instance.InstanceID != instanceID || instance.DefinitionID != CardDefinitionID(instanceID) ||
			instance.Document.CardID != string(instanceID) {
			t.Fatalf("legacy instance %q = %#v", instanceID, instance)
		}
	}

	mustExecuteResult(t, session, CollectCardCommand{CardID: "item"})
	reset := mustExecuteResult(t, session, ResetCommand{})
	if got := cardIDs(reset.Snapshot.WorldDeck); !reflect.DeepEqual(got, []string{"anchor", "item"}) {
		t.Fatalf("reset instance ids = %v", got)
	}
	if hasEventType(reset.Events, EventCardInstantiated) {
		t.Fatalf("reset emitted cardInstantiated: %#v", reset.Events)
	}
}

func TestExplicitInitialInstancesAreAuthoritativeAndIndependent(t *testing.T) {
	tokenDefault := sliderDocument("token", 2)
	tokenUsed := sliderDocument("token", 9)
	definition := DeckDefinition{
		ID: "explicit-instances", Name: "Explicit Instances",
		InitialActiveInstanceID: "target-one",
		InitialInstances: []CardInstanceSpec{
			{InstanceID: "target-one", DefinitionID: "target", Zone: ZoneScene},
			{
				InstanceID: "token-one", DefinitionID: "token", Zone: ZoneScene,
				Actor: &ActorState{
					Integrity:   ResourceState{Current: 2, Max: 2},
					Charge:      ResourceState{Current: 1, Max: 2},
					Disposition: DispositionFriendly,
				},
			},
			{
				InstanceID: "token-two", DefinitionID: "token", Zone: ZoneScene,
				Actor: &ActorState{
					Integrity:   ResourceState{Current: 2, Max: 2},
					Charge:      ResourceState{Current: 1, Max: 2},
					Disposition: DispositionFriendly,
				},
			},
		},
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("target")}},
			{ID: "token", Name: "Token", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": tokenDefault, "used": tokenUsed}},
			{ID: "template-only", Name: "Template", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("template-only")}},
		},
		Rules: []RuleDefinition{{
			ID:      "use-token",
			Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "token"}, CardMatcherDefinition{ID: "target"}),
			Effects: []RuleEffect{
				SetCardStateFor(SignaledInstance(RuleCardSource), "used", true),
				SetDocumentVariantFor(SignaledInstance(RuleCardSource), "used"),
				MoveCardEffect(SignaledInstance(RuleCardSource), ZoneDiscard),
			},
		}},
	}
	session, err := NewSessionFromDeck(catalog.MustNew(), definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := session.instances["template-only"]; exists {
		t.Fatal("unreferenced definition was implicitly instantiated")
	}
	if session.instances["token-one"].DefinitionID != session.instances["token-two"].DefinitionID {
		t.Fatal("duplicate instances do not share their definition")
	}
	for _, id := range []CardInstanceID{"token-one", "token-two"} {
		if session.instances[id].Document.CardID != string(id) {
			t.Fatalf("%s document card id = %q", id, session.instances[id].Document.CardID)
		}
	}

	if err := session.updateInstance("token-two", func(instance *CardInstance) error {
		instance.State = map[string]any{"editedBeforeMove": true}
		instance.Actor.Charge.Current = 0
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	mustExecuteResult(t, session, CollectCardCommand{CardID: "token-two"})
	result := mustExecuteResult(t, session, PlayCardCommand{SourceCardID: "token-two", TargetCardID: "target-one"})

	if containsCard(result.Snapshot.Library, "token-two") || session.zoneIndex(ZoneDiscard, "token-two") < 0 {
		t.Fatalf("token-two zones = %#v", session.zones)
	}
	used := session.instances["token-two"]
	untouched := session.instances["token-one"]
	if !stateBool(used.State, "used") || !stateBool(used.State, "editedBeforeMove") ||
		used.Document.CardID != "token-two" {
		t.Fatalf("signaled instance = %#v", used)
	}
	if stateBool(untouched.State, "used") || untouched.Document.CardID != "token-one" {
		t.Fatalf("other instance was mutated: %#v", untouched)
	}
	if untouched.Actor == nil || untouched.Actor.Charge.Current != 1 || used.Actor == nil || used.Actor.Charge.Current != 0 {
		t.Fatalf("actor state was shared: token-one=%#v token-two=%#v", untouched.Actor, used.Actor)
	}
	if session.cardDefinitions["token"].Documents["used"].CardID != "token" {
		t.Fatal("immutable definition document was rewritten")
	}
}

func TestZoneMovementRepairsActiveSceneAndRejectsEmptyScene(t *testing.T) {
	definition := DeckDefinition{
		ID: "active-repair", Name: "Active Repair", InitialActiveCardID: "third",
		Cards: []CardDefinition{
			{ID: "first", Name: "First", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("first")}},
			{ID: "second", Name: "Second", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("second")}},
			{ID: "third", Name: "Third", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("third")}},
		},
	}
	session, err := NewSessionFromDeck(catalog.MustNew(), definition)
	if err != nil {
		t.Fatal(err)
	}
	collected := mustExecuteResult(t, session, CollectCardCommand{CardID: "first"})
	if collected.Snapshot.ActiveWorldCardID != "third" || collected.Snapshot.ActiveIndex != 1 {
		t.Fatalf("active after preceding move = %q at %d", collected.Snapshot.ActiveWorldCardID, collected.Snapshot.ActiveIndex)
	}

	session.zones[ZoneLibrary] = append(session.zones[ZoneLibrary], "second")
	if err := session.validateZoneState(); err == nil {
		t.Fatal("duplicate zone membership validated")
	}
	session.zones[ZoneLibrary] = session.zones[ZoneLibrary][:len(session.zones[ZoneLibrary])-1]
	session.zones[ZoneScene] = session.zones[ZoneScene][1:]
	if err := session.validateZoneState(); err == nil {
		t.Fatal("missing zone membership validated")
	}
	session.zones[ZoneScene] = append([]CardInstanceID{"second"}, session.zones[ZoneScene]...)

	final := &Session{
		cardDefinitions: map[CardDefinitionID]CardDefinition{
			"only": {ID: "only", Name: "Only", Kind: KindWorld},
		},
		instances: map[CardInstanceID]CardInstance{
			"only": {InstanceID: "only", DefinitionID: "only", Document: simpleDocument("only")},
		},
		zones:             ZoneState{ZoneScene: {"only"}, ZoneLibrary: nil, ZoneDiscard: nil},
		activeSceneCardID: "only",
	}
	if _, err := final.moveCard("only", ZoneScene, ZoneDiscard); err == nil {
		t.Fatal("moving the final scene card succeeded")
	}
}

func TestDeckPackRejectsInstanceIDCollision(t *testing.T) {
	definition := DeckDefinition{
		ID: "colliding-pack", Name: "Colliding Pack",
		InitialActiveInstanceID: "shared-instance",
		InitialInstances: []CardInstanceSpec{
			{InstanceID: "shared-instance", DefinitionID: "pack-card", Zone: ZoneScene},
		},
		Cards: []CardDefinition{{
			ID: "pack-card", Name: "Pack Card", Kind: KindWorld,
			InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("pack-card")},
		}},
	}
	err := ValidateDeckPackDefinition(
		catalog.MustNew(),
		definition,
		map[CardDefinitionID]CardDefinition{
			"base-card": {ID: "base-card", Name: "Base Card", Kind: KindWorld},
		},
		map[CardInstanceID]CardDefinitionID{"shared-instance": "base-card"},
		nil,
	)
	if err == nil {
		t.Fatal("deck pack with colliding instance id validated")
	}
}

func TestFailedRuleMovementRollsBackInstancesZonesAndEncounter(t *testing.T) {
	definition := DeckDefinition{
		ID: "rollback-instances", Name: "Rollback Instances", InitialActiveCardID: "target",
		Cards: []CardDefinition{
			{ID: "target", Name: "Target", Kind: KindWorld, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("target")}},
			{ID: "source", Name: "Source", Kind: KindItem, Collectible: true, InitialDocument: "default", Documents: map[string]card.Document{"default": simpleDocument("source")}},
		},
		Rules: []RuleDefinition{{
			ID:      "invalid-final-move",
			Trigger: CardPlayedTrigger(CardMatcherDefinition{ID: "source"}, CardMatcherDefinition{ID: "target"}),
			Effects: []RuleEffect{
				SetCardStateFor(SignaledInstance(RuleCardSource), "partiallyMutated", true),
				MoveCardEffect(SignaledInstance(RuleCardTarget), ZoneDiscard),
			},
		}},
	}
	session, err := NewSessionFromDeck(catalog.MustNew(), definition)
	if err != nil {
		t.Fatal(err)
	}
	source := session.instances["source"]
	source.Actor = &ActorState{
		Integrity:   ResourceState{Current: 3, Max: 3},
		Charge:      ResourceState{Current: 1, Max: 2},
		Disposition: DispositionFriendly,
	}
	session.instances["source"] = source
	session.encounter = &EncounterState{
		ID: "rollback-encounter", Phase: EncounterPhaseActive,
		Participants: []EncounterParticipant{{InstanceID: "source", Role: EncounterRolePlayer}},
		Pressure:     2,
	}
	mustExecuteResult(t, session, CollectCardCommand{CardID: "source"})
	before, err := cloneSessionState(session.stateLocked())
	if err != nil {
		t.Fatal(err)
	}
	revision := session.revision

	if _, err := session.Execute(PlayCardCommand{SourceCardID: "source", TargetCardID: "target"}); err == nil {
		t.Fatal("play moving the final scene card succeeded")
	}
	after := session.stateLocked()
	if session.revision != revision || !reflect.DeepEqual(after.instances, before.instances) ||
		!reflect.DeepEqual(after.zones, before.zones) || !reflect.DeepEqual(after.encounter, before.encounter) {
		t.Fatalf("rollback mismatch:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestDynamicDeckInstantiationEventsAndEncounterEvents(t *testing.T) {
	definition, err := LoadEmbeddedDeck(catalog.MustNew(), GeneratorDeckDefinition)
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewSessionFromDeck(catalog.MustNew(), definition)
	if err != nil {
		t.Fatal(err)
	}
	events := &eventCollector{}
	if _, err := session.applyRuleEffects(
		[]RuleEffect{LoadDeckEffect(ArchiveTerminalDefinition)},
		ComponentUpdatedSignal{InstanceID: "generator-panel"},
		events,
	); err != nil {
		t.Fatal(err)
	}
	specs := initialInstanceSpecs(mustLoadDeck(t, session, ArchiveTerminalDefinition))
	if len(events.events) != len(specs)+1 {
		t.Fatalf("load events = %#v", events.events)
	}
	for index := range specs {
		if events.events[index].Type != EventCardInstantiated {
			t.Fatalf("event %d = %q", index, events.events[index].Type)
		}
	}
	if events.events[len(events.events)-1].Type != EventDeckLoaded || session.activeSceneCardID != "archive-terminal" {
		t.Fatalf("dynamic load events=%#v active=%q", events.events, session.activeSceneCardID)
	}

	actor := session.instances["archive-terminal"]
	actor.Actor = &ActorState{
		Integrity:   ResourceState{Current: 4, Max: 5},
		Charge:      ResourceState{Current: 2, Max: 3},
		Disposition: DispositionNeutral,
	}
	session.instances["archive-terminal"] = actor
	encounterEvents := &eventCollector{}
	if err := session.startEncounter(EncounterState{
		ID: "archive-pressure", Phase: EncounterPhaseSetup,
		Participants: []EncounterParticipant{{InstanceID: "archive-terminal", Role: EncounterRoleEnvironmental}},
	}, encounterEvents); err != nil {
		t.Fatal(err)
	}
	if err := session.changeEncounterPhase(EncounterPhaseActive, encounterEvents); err != nil {
		t.Fatal(err)
	}
	if err := session.changeActorResource("archive-terminal", "charge", 1, encounterEvents); err != nil {
		t.Fatal(err)
	}
	if err := session.resolveEncounter("stabilized", encounterEvents); err != nil {
		t.Fatal(err)
	}
	assertEventTypes(t, encounterEvents.events,
		EventEncounterStarted,
		EventEncounterPhaseChanged,
		EventActorResourceChanged,
		EventEncounterResolved,
	)
}

func mustLoadDeck(t *testing.T, session *Session, deckID string) DeckDefinition {
	t.Helper()
	definition, err := LoadEmbeddedDeck(session.registry, deckID)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
