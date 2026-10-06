package oncall

import (
	"fmt"
	"time"
)

// NextPublication returns the first scheduled instant strictly after now.
// A skipped wall time moves forward to the next valid minute. A repeated time
// uses its first occurrence, so the local date is published only once.
func NextPublication(now time.Time, clock, zone string) (time.Time, error) {
	wall, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("time must be HH:mm")
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.Time{}, fmt.Errorf("timezone must be a valid IANA name")
	}
	local := now.In(loc)
	for day := 0; day < 4; day++ {
		date := time.Date(local.Year(), local.Month(), local.Day()+day, 12, 0, 0, 0, time.UTC)
		anchor := time.Date(date.Year(), date.Month(), date.Day(), wall.Hour(), wall.Minute(), 0, 0, time.UTC)
		bestMinute := 24 * 60
		var best time.Time
		for t := anchor.Add(-18 * time.Hour); !t.After(anchor.Add(18 * time.Hour)); t = t.Add(time.Minute) {
			lt := t.In(loc)
			if lt.Year() != date.Year() || lt.Month() != date.Month() || lt.Day() != date.Day() {
				continue
			}
			minute := lt.Hour()*60 + lt.Minute()
			if minute < wall.Hour()*60+wall.Minute() {
				continue
			}
			if minute < bestMinute {
				bestMinute = minute
				best = t
			}
		}
		if !best.IsZero() && best.After(now) {
			return best, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot determine next publication")
}
