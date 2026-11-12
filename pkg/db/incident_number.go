package db

import "fmt"

// DisplayID owns INC-NUM-AC1: the persisted, globally unique positive Number is
// shown as INC-01, INC-02, ... INC-100 without truncation. UUIDs remain the
// identity for relations, URLs and actions. A zero Number (legacy test/snapshot)
// falls back to its UUID. Evidence: incident_number_test.go.
func (i Incident) DisplayID() string {
	if i.Number <= 0 {
		return i.ID.String()
	}
	return fmt.Sprintf("INC-%02d", i.Number)
}
