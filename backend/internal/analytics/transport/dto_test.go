package transport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeRange_PlainDatesAreBangladeshDays(t *testing.T) {
	q := GetAnalyticsQuery{StartAt: "2026-10-01", EndAt: "2026-10-11"}
	tr, err := q.ParseTimeRange()
	require.NoError(t, err)
	// 1 Oct 00:00 in Dhaka (UTC+6) is 30 Sep 18:00 UTC
	assert.Equal(t, time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC), tr.Start)
	// the end date includes the whole day: 11 Oct 23:59:59.999... in Dhaka = 11 Oct 17:59:59 UTC
	assert.Equal(t, 2026, tr.End.Year())
	assert.Equal(t, time.Date(2026, 10, 11, 17, 59, 59, 999999999, time.UTC), tr.End)
}

func TestParseTimeRange_StillAcceptsRFC3339AndStartEnd(t *testing.T) {
	q := GetAnalyticsQuery{Start: "2026-10-01T00:00:00Z", End: "2026-10-02T00:00:00Z"}
	tr, err := q.ParseTimeRange()
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), tr.Start)
	assert.Equal(t, time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), tr.End)
}

func TestParseTimeRange_RejectsGarbage(t *testing.T) {
	q := GetAnalyticsQuery{Start: "yesterday"}
	_, err := q.ParseTimeRange()
	require.Error(t, err)
}
