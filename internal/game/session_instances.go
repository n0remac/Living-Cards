package game

import (
	"fmt"
	"strings"
)

func (s *Session) instance(id CardInstanceID) (CardInstance, bool) {
	instance, ok := s.instances[id]
	return instance, ok
}

func (s *Session) updateInstance(id CardInstanceID, update func(*CardInstance) error) error {
	instance, ok := s.instances[id]
	if !ok {
		return fmt.Errorf("card instance %q does not exist", id)
	}
	if err := update(&instance); err != nil {
		return err
	}
	instance.InstanceID = id
	instance.Document.CardID = string(id)
	if instance.Actor != nil {
		if err := validateActor(*instance.Actor); err != nil {
			return fmt.Errorf("card instance %q actor: %w", id, err)
		}
	}
	s.instances[id] = instance
	return nil
}

func (s *Session) zoneIndex(zone Zone, id CardInstanceID) int {
	for index, candidate := range s.zones[zone] {
		if candidate == id {
			return index
		}
	}
	return -1
}

func (s *Session) instanceZone(id CardInstanceID) (Zone, bool) {
	var found Zone
	count := 0
	for _, zone := range []Zone{ZoneScene, ZoneLibrary, ZoneDiscard} {
		if s.zoneIndex(zone, id) >= 0 {
			found = zone
			count++
		}
	}
	return found, count == 1
}

func (s *Session) moveCard(id CardInstanceID, expectedFrom, to Zone) (CardMove, error) {
	if !validZone(expectedFrom) || !validZone(to) {
		return CardMove{}, fmt.Errorf("unsupported card movement from %q to %q", expectedFrom, to)
	}
	if expectedFrom == to {
		return CardMove{}, fmt.Errorf("card instance %q is already in zone %q", id, to)
	}
	if _, ok := s.instances[id]; !ok {
		return CardMove{}, fmt.Errorf("card instance %q does not exist", id)
	}
	from, unique := s.instanceZone(id)
	if !unique {
		return CardMove{}, fmt.Errorf("card instance %q does not belong to exactly one zone", id)
	}
	if from != expectedFrom {
		return CardMove{}, fmt.Errorf("card instance %q is in zone %q, not %q", id, from, expectedFrom)
	}
	fromIndex := s.zoneIndex(from, id)
	if from == ZoneScene && len(s.zones[ZoneScene]) == 1 {
		return CardMove{}, fmt.Errorf("cannot move the final scene card")
	}
	s.zones[from] = append(s.zones[from][:fromIndex], s.zones[from][fromIndex+1:]...)
	toIndex := len(s.zones[to])
	s.zones[to] = append(s.zones[to], id)
	if from == ZoneScene {
		s.repairActiveScene(id, fromIndex)
	}
	return CardMove{InstanceID: id, From: from, To: to, FromIndex: fromIndex, ToIndex: toIndex}, nil
}

func (s *Session) repairActiveScene(movedID CardInstanceID, previousIndex int) {
	if movedID != s.activeSceneCardID {
		return
	}
	scene := s.zones[ZoneScene]
	if previousIndex >= len(scene) {
		previousIndex = 0
	}
	s.activeSceneCardID = scene[previousIndex]
	s.activeEditingComponentID = ""
}

func (s *Session) activeSceneCard() (CardInstance, error) {
	instance, ok := s.instance(s.activeSceneCardID)
	if !ok || s.zoneIndex(ZoneScene, s.activeSceneCardID) < 0 {
		return CardInstance{}, fmt.Errorf("active scene card %q is invalid", s.activeSceneCardID)
	}
	return instance, nil
}

func (s *Session) validateZoneState() error {
	counts := make(map[CardInstanceID]int, len(s.instances))
	for zone, ids := range s.zones {
		if !validZone(zone) {
			return fmt.Errorf("unsupported zone %q", zone)
		}
		for _, id := range ids {
			if _, ok := s.instances[id]; !ok {
				return fmt.Errorf("zone %q references unknown card instance %q", zone, id)
			}
			counts[id]++
			if counts[id] > 1 {
				return fmt.Errorf("card instance %q belongs to more than one zone", id)
			}
		}
	}
	for id, instance := range s.instances {
		if counts[id] != 1 {
			return fmt.Errorf("card instance %q must belong to exactly one zone", id)
		}
		if id != instance.InstanceID {
			return fmt.Errorf("card instance map key %q does not match instance id %q", id, instance.InstanceID)
		}
		if _, ok := s.cardDefinitions[instance.DefinitionID]; !ok {
			return fmt.Errorf("card instance %q references missing definition %q", id, instance.DefinitionID)
		}
		if instance.Document.CardID != string(id) {
			return fmt.Errorf("card instance %q document has card_id %q", id, instance.Document.CardID)
		}
		if instance.Actor != nil {
			if err := validateActor(*instance.Actor); err != nil {
				return fmt.Errorf("card instance %q actor: %w", id, err)
			}
		}
	}
	if len(s.zones[ZoneScene]) == 0 {
		return fmt.Errorf("scene zone is empty")
	}
	if s.zoneIndex(ZoneScene, s.activeSceneCardID) < 0 {
		return fmt.Errorf("active scene card %q is not in the scene zone", s.activeSceneCardID)
	}
	if s.encounter != nil {
		if err := s.validateEncounter(*s.encounter); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) cardSnapshot(id CardInstanceID, zone Zone) (CardSnapshot, error) {
	instance, ok := s.instance(id)
	if !ok {
		return CardSnapshot{}, fmt.Errorf("card instance %q does not exist", id)
	}
	definition, ok := s.cardDefinitions[instance.DefinitionID]
	if !ok {
		return CardSnapshot{}, fmt.Errorf("card instance %q references missing definition %q", id, instance.DefinitionID)
	}
	return CardSnapshot{
		ID:          id,
		Name:        definition.Name,
		Kind:        definition.Kind,
		Tags:        append([]string(nil), instance.Tags...),
		Collectible: definition.Collectible && zone == ZoneScene,
		Collected:   zone == ZoneLibrary,
		State:       cloneValue(instance.State),
		Document:    cloneValue(instance.Document),
		Actor:       cloneValue(instance.Actor),
	}, nil
}

func (s *Session) zoneSnapshots(zone Zone) ([]CardSnapshot, error) {
	ids := s.zones[zone]
	if len(ids) == 0 {
		return nil, nil
	}
	out := make([]CardSnapshot, 0, len(ids))
	for _, id := range ids {
		card, err := s.cardSnapshot(id, zone)
		if err != nil {
			return nil, err
		}
		out = append(out, card)
	}
	return out, nil
}

func (s *Session) validateEncounter(encounter EncounterState) error {
	if strings.TrimSpace(string(encounter.ID)) == "" {
		return fmt.Errorf("encounter id is required")
	}
	if !validEncounterPhase(encounter.Phase) {
		return fmt.Errorf("encounter %q has unsupported phase %q", encounter.ID, encounter.Phase)
	}
	if len(encounter.Participants) == 0 {
		return fmt.Errorf("encounter %q requires at least one participant", encounter.ID)
	}
	if encounter.Pressure < 0 || encounter.MaxPressure < 0 ||
		(encounter.MaxPressure > 0 && encounter.Pressure > encounter.MaxPressure) {
		return fmt.Errorf("encounter %q pressure is out of range", encounter.ID)
	}
	if encounter.ReactionPressure < 0 {
		return fmt.Errorf("encounter %q reaction pressure cannot be negative", encounter.ID)
	}
	seen := map[CardInstanceID]bool{}
	for _, participant := range encounter.Participants {
		if _, ok := s.instances[participant.InstanceID]; !ok {
			return fmt.Errorf("encounter %q references unknown participant %q", encounter.ID, participant.InstanceID)
		}
		if seen[participant.InstanceID] {
			return fmt.Errorf("encounter %q repeats participant %q", encounter.ID, participant.InstanceID)
		}
		if !validEncounterRole(participant.Role) {
			return fmt.Errorf("encounter %q has unsupported role %q", encounter.ID, participant.Role)
		}
		if zone, ok := s.instanceZone(participant.InstanceID); !ok ||
			(zone == ZoneDiscard && encounter.Phase != EncounterPhaseResolved) {
			return fmt.Errorf("encounter %q participant %q must be in the scene or library", encounter.ID, participant.InstanceID)
		}
		seen[participant.InstanceID] = true
	}
	return nil
}

func (s *Session) startEncounter(encounter EncounterState, events *eventCollector) error {
	if s.encounter != nil && s.encounter.Phase != EncounterPhaseResolved {
		return fmt.Errorf("encounter %q is already active", s.encounter.ID)
	}
	if err := s.validateEncounter(encounter); err != nil {
		return err
	}
	s.encounter = &encounter
	events.emit(EventEncounterStarted, EncounterStartedPayload{EncounterID: string(encounter.ID), Phase: string(encounter.Phase)})
	return nil
}

func (s *Session) changeEncounterPhase(phase EncounterPhase, events *eventCollector) error {
	if s.encounter == nil {
		return fmt.Errorf("no encounter is active")
	}
	if !validEncounterPhase(phase) {
		return fmt.Errorf("unsupported encounter phase %q", phase)
	}
	previous := s.encounter.Phase
	s.encounter.Phase = phase
	events.emit(EventEncounterPhaseChanged, EncounterPhaseChangedPayload{
		EncounterID: string(s.encounter.ID), PreviousPhase: string(previous), Phase: string(phase),
	})
	return nil
}

func (s *Session) resolveEncounter(outcome string, events *eventCollector) error {
	if s.encounter == nil {
		return fmt.Errorf("no encounter is active")
	}
	if strings.TrimSpace(outcome) == "" {
		return fmt.Errorf("encounter outcome is required")
	}
	s.encounter.Phase = EncounterPhaseResolved
	s.encounter.Outcome = outcome
	events.emit(EventEncounterResolved, EncounterResolvedPayload{EncounterID: string(s.encounter.ID), Outcome: outcome})
	return nil
}

func (s *Session) changeActorResource(
	id CardInstanceID,
	resource string,
	current int,
	events *eventCollector,
) error {
	var previous int
	err := s.updateInstance(id, func(instance *CardInstance) error {
		if instance.Actor == nil {
			return fmt.Errorf("card instance %q has no actor state", id)
		}
		switch resource {
		case "integrity":
			previous = instance.Actor.Integrity.Current
			instance.Actor.Integrity.Current = current
		case "charge":
			previous = instance.Actor.Charge.Current
			instance.Actor.Charge.Current = current
		default:
			track, ok := instance.Actor.Tracks[resource]
			if !ok {
				return fmt.Errorf("actor %q has no track %q", id, resource)
			}
			previous = track.Current
			track.Current = current
			instance.Actor.Tracks[resource] = track
		}
		return nil
	})
	if err != nil {
		return err
	}
	events.emit(EventActorResourceChanged, ActorResourceChangedPayload{
		InstanceID: string(id), Resource: resource, Previous: previous, Current: current,
	})
	return nil
}

func actorResourceCurrent(actor ActorState, resource string) (int, bool) {
	switch resource {
	case "integrity":
		return actor.Integrity.Current, true
	case "charge":
		return actor.Charge.Current, true
	default:
		track, ok := actor.Tracks[resource]
		return track.Current, ok
	}
}

func (s *Session) changeEncounterPressure(delta int, events *eventCollector) error {
	if s.encounter == nil || s.encounter.Phase == EncounterPhaseResolved || delta == 0 {
		return nil
	}
	previous := s.encounter.Pressure
	current := previous + delta
	if current < 0 {
		current = 0
	}
	if s.encounter.MaxPressure > 0 && current > s.encounter.MaxPressure {
		current = s.encounter.MaxPressure
	}
	s.encounter.Pressure = current
	events.emit(EventEncounterPressureChanged, EncounterPressureChangedPayload{
		EncounterID: string(s.encounter.ID), Previous: previous, Current: current,
	})
	if s.encounter.MaxPressure > 0 && current >= s.encounter.MaxPressure {
		return s.resolveEncounter("overwhelmed", events)
	}
	return nil
}

func (s *Session) advanceEncounter(events *eventCollector) error {
	if s.encounter == nil || s.encounter.Phase != EncounterPhaseActive {
		return nil
	}
	return s.changeEncounterPressure(s.encounter.ReactionPressure, events)
}

func (s *Session) setActorDisposition(id CardInstanceID, disposition ActorDisposition, events *eventCollector) error {
	var previous ActorDisposition
	if err := s.updateInstance(id, func(instance *CardInstance) error {
		if instance.Actor == nil {
			return fmt.Errorf("card instance %q has no actor state", id)
		}
		if !validDisposition(disposition) {
			return fmt.Errorf("unsupported actor disposition %q", disposition)
		}
		previous = instance.Actor.Disposition
		instance.Actor.Disposition = disposition
		return nil
	}); err != nil {
		return err
	}
	events.emit(EventActorDispositionChanged, ActorDispositionChangedPayload{
		InstanceID: string(id), Previous: string(previous), Current: string(disposition),
	})
	return nil
}

func (s *Session) changeActorStatus(id CardInstanceID, status string, add bool, events *eventCollector) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return fmt.Errorf("actor status is required")
	}
	changed := false
	if err := s.updateInstance(id, func(instance *CardInstance) error {
		if instance.Actor == nil {
			return fmt.Errorf("card instance %q has no actor state", id)
		}
		index := -1
		for candidateIndex, candidate := range instance.Actor.Statuses {
			if candidate == status {
				index = candidateIndex
				break
			}
		}
		if add && index < 0 {
			instance.Actor.Statuses = append(instance.Actor.Statuses, status)
			changed = true
		}
		if !add && index >= 0 {
			instance.Actor.Statuses = append(instance.Actor.Statuses[:index], instance.Actor.Statuses[index+1:]...)
			changed = true
		}
		return nil
	}); err != nil {
		return err
	}
	if changed {
		events.emit(EventActorStatusChanged, ActorStatusChangedPayload{
			InstanceID: string(id), Status: status, Added: add,
		})
	}
	return nil
}
