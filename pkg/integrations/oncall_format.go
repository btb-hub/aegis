package integrations

import (
	"fmt"
	"time"
	_ "time/tzdata"
)

func ShiftTimeMSK(start, end time.Time, locale string) string {
	if start.IsZero() || end.IsZero() {
		return ""
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	start = start.In(loc)
	end = end.In(loc)
	label := "MSK"
	if locale == "ru" {
		label = "МСК"
	}
	format := "02 Jan 15:04"
	if locale == "ru" {
		format = "02.01 15:04"
	}
	return fmt.Sprintf("%s – %s %s", start.Format(format), end.Format(format), label)
}
func OnCallName(person OnCallPerson, locale string) string {
	name := person.DisplayName
	if shift := ShiftTimeMSK(person.StartAt, person.EndAt, locale); shift != "" {
		name += " (" + shift + ")"
	}
	return name
}
