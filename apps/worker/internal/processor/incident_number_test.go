package processor

import (
	"github.com/aegis/aegis/pkg/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

// INC-NUM-CHAT: worker mapping preserves UUID and exposes its durable label.
// This pure mapper test does not establish delivery to an external provider.
func TestIncidentRefCarriesDisplayID(t *testing.T) {
	id := uuid.New()
	ref := toIncidentRef(db.Incident{ID: id, Number: 1})
	require.Equal(t, "INC-01", ref.DisplayID)
	require.Equal(t, id, ref.ID)
}
