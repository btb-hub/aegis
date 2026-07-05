package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendAlertListConditionWithoutWhere(t *testing.T) {
	from := alertListQuery{sql: "FROM alerts "}
	extended := appendAlertListCondition(from, "labels ? $1")
	require.Equal(t, "FROM alerts  WHERE labels ? $1", extended.sql)
}

func TestAppendAlertListConditionWithWhere(t *testing.T) {
	from := alertListQuery{sql: "FROM alerts WHERE severity = $1", args: []any{"critical"}}
	extended := appendAlertListCondition(from, "labels ? $2")
	require.Equal(t, "FROM alerts WHERE severity = $1 AND labels ? $2", extended.sql)
	require.Equal(t, []any{"critical"}, extended.args)
}
