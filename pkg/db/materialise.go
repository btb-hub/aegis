package db

import (
	"context"
	"time"

	"github.com/aegis/aegis/pkg/oncall"
	"github.com/google/uuid"
)

const MaterialiseHorizonDays = 90

func (s *Store) MaterialiseOnCallForTeam(ctx context.Context, teamID uuid.UUID) error {
	schedules, err := s.ListSchedulesWithLayersByTeam(ctx, teamID)
	if err != nil {
		return err
	}
	overrides, err := s.ListOverridesByTeam(ctx, teamID)
	if err != nil {
		return err
	}

	layers := weeklyLayersFromSchedules(schedules)
	oncallOverrides := make([]oncall.Override, 0, len(overrides))
	for _, override := range overrides {
		oncallOverrides = append(oncallOverrides, oncall.Override{
			UserID:  override.UserID,
			StartAt: override.StartAt,
			EndAt:   override.EndAt,
		})
	}

	from := time.Now().UTC().Truncate(time.Second)
	to := from.Add(MaterialiseHorizonDays * 24 * time.Hour)
	computed, err := oncall.Materialise(layers, oncallOverrides, from, to)
	if err != nil {
		return err
	}

	slots := make([]OnCallSlot, 0, len(computed))
	for _, slot := range computed {
		slots = append(slots, OnCallSlot{
			TeamID:  teamID,
			UserID:  slot.UserID,
			StartAt: slot.StartAt,
			EndAt:   slot.EndAt,
			Source:  slot.Source,
		})
	}
	return s.ReplaceOnCallSlots(ctx, teamID, slots)
}

func weeklyLayersFromSchedules(schedules []ScheduleWithLayers) []oncall.WeeklyLayer {
	var layers []oncall.WeeklyLayer
	for _, schedule := range schedules {
		for _, layer := range schedule.Layers {
			layers = append(layers, oncall.WeeklyLayer{
				ScheduleID:     schedule.Schedule.ID,
				Priority:       layer.Priority,
				Timezone:       schedule.Schedule.Timezone,
				HandoffWeekday: layer.HandoffWeekday,
				HandoffTime:    layer.HandoffTime,
				Participants:   layer.ParticipantUserIDs,
			})
		}
	}
	return layers
}

func (s *Store) ListTeamIDsWithSchedules(ctx context.Context) ([]uuid.UUID, error) {
	const q = `SELECT DISTINCT team_id FROM schedules ORDER BY team_id`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
