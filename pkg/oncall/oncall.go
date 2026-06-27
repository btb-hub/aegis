package oncall

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

const (
	SourceRotation = "rotation"
	SourceOverride = "override"
)

type WeeklyLayer struct {
	ScheduleID     uuid.UUID
	Priority       int32
	Timezone       string
	HandoffWeekday int32
	HandoffTime    time.Time
	Participants   []uuid.UUID
}

type Override struct {
	UserID  uuid.UUID
	StartAt time.Time
	EndAt   time.Time
}

type Slot struct {
	UserID  uuid.UUID
	StartAt time.Time
	EndAt   time.Time
	Source  string
}

// WhoAt returns the on-call user at instant at. Overrides win over rotation.
func WhoAt(at time.Time, layers []WeeklyLayer, overrides []Override) (uuid.UUID, string, error) {
	for _, override := range overrides {
		if covers(override.StartAt, override.EndAt, at) {
			return override.UserID, SourceOverride, nil
		}
	}
	user, err := rotationUserAt(at, layers)
	if err != nil {
		return uuid.Nil, "", err
	}
	return user, SourceRotation, nil
}

// Materialise builds non-overlapping on-call slots for [from, to).
func Materialise(layers []WeeklyLayer, overrides []Override, from, to time.Time) ([]Slot, error) {
	if !from.Before(to) {
		return nil, fmt.Errorf("from must be before to")
	}
	boundaries := []time.Time{from, to}
	for _, layer := range layers {
		handoffs, err := handoffTimesInRange(layer, from, to)
		if err != nil {
			return nil, err
		}
		boundaries = append(boundaries, handoffs...)
	}
	for _, override := range overrides {
		if override.EndAt.After(from) && override.StartAt.Before(to) {
			if override.StartAt.After(from) {
				boundaries = append(boundaries, override.StartAt)
			}
			if override.EndAt.Before(to) {
				boundaries = append(boundaries, override.EndAt)
			}
		}
	}
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].Before(boundaries[j]) })
	unique := dedupeTimes(boundaries)

	var slots []Slot
	for i := 0; i < len(unique)-1; i++ {
		start := unique[i]
		end := unique[i+1]
		if !start.Before(end) {
			continue
		}
		mid := start.Add(end.Sub(start) / 2)
		user, source, err := WhoAt(mid, layers, overrides)
		if err != nil {
			return nil, err
		}
		if user == uuid.Nil {
			continue
		}
		if len(slots) > 0 {
			last := &slots[len(slots)-1]
			if last.UserID == user && last.Source == source && last.EndAt.Equal(start) {
				last.EndAt = end
				continue
			}
		}
		slots = append(slots, Slot{UserID: user, StartAt: start, EndAt: end, Source: source})
	}
	return slots, nil
}

func rotationUserAt(at time.Time, layers []WeeklyLayer) (uuid.UUID, error) {
	if len(layers) == 0 {
		return uuid.Nil, nil
	}
	winner := layers[0]
	for _, layer := range layers[1:] {
		if layer.Priority < winner.Priority {
			winner = layer
		}
	}
	loc, err := time.LoadLocation(winner.Timezone)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid timezone %q: %w", winner.Timezone, err)
	}
	handoff := lastHandoff(at.In(loc), int(winner.HandoffWeekday), winner.HandoffTime.Hour(), winner.HandoffTime.Minute(), loc)
	idx := rotationIndex(handoff, int(winner.HandoffWeekday), winner.HandoffTime.Hour(), winner.HandoffTime.Minute(), loc, len(winner.Participants))
	return winner.Participants[idx], nil
}

func lastHandoff(localAt time.Time, weekday, hour, minute int, loc *time.Location) time.Time {
	year, month, day := localAt.Date()
	currentWeekday := int(localAt.Weekday())
	daysBack := (currentWeekday - weekday + 7) % 7
	candidate := time.Date(year, month, day-daysBack, hour, minute, 0, 0, loc)
	if candidate.After(localAt) {
		candidate = candidate.AddDate(0, 0, -7)
	}
	return candidate
}

func rotationIndex(handoff time.Time, weekday, hour, minute int, loc *time.Location, participants int) int {
	if participants == 0 {
		return 0
	}
	anchor := time.Date(2020, 1, 1, hour, minute, 0, 0, loc)
	delta := (weekday - int(anchor.Weekday()) + 7) % 7
	anchor = anchor.AddDate(0, 0, delta)
	if anchor.After(handoff) {
		anchor = anchor.AddDate(0, 0, -7)
	}
	weeks := 0
	for next := anchor.AddDate(0, 0, 7); !next.After(handoff); next = next.AddDate(0, 0, 7) {
		weeks++
	}
	return weeks % participants
}

func handoffTimesInRange(layer WeeklyLayer, from, to time.Time) ([]time.Time, error) {
	loc, err := time.LoadLocation(layer.Timezone)
	if err != nil {
		return nil, err
	}
	localFrom := from.In(loc)
	cursor := lastHandoff(localFrom, int(layer.HandoffWeekday), layer.HandoffTime.Hour(), layer.HandoffTime.Minute(), loc)
	var times []time.Time
	for cursor.Before(to) {
		utc := cursor.UTC()
		if utc.After(from) && utc.Before(to) {
			times = append(times, utc)
		}
		cursor = cursor.AddDate(0, 0, 7)
	}
	return times, nil
}

func covers(start, end, at time.Time) bool {
	return !at.Before(start) && at.Before(end)
}

func dedupeTimes(times []time.Time) []time.Time {
	if len(times) == 0 {
		return nil
	}
	out := []time.Time{times[0]}
	for _, t := range times[1:] {
		if t.Equal(out[len(out)-1]) {
			continue
		}
		out = append(out, t)
	}
	return out
}
