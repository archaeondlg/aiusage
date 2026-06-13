package dateutil

import (
	"testing"
	"time"

	"github.com/archhaeondlg/aiusage/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // UTC RFC3339
	}{
		{"RFC3339 with Z", "2024-08-04T23:30:00.000Z", "2024-08-04T23:30:00Z"},
		{"RFC3339 with offset", "2024-08-05T08:30:00.000+09:00", "2024-08-04T23:30:00Z"},
		{"RFC3339 short", "2024-08-04T23:30:00Z", "2024-08-04T23:30:00Z"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := ParseTimestamp(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, ts.Format(time.RFC3339))
		})
	}
}

func TestFormatDate(t *testing.T) {
	ts, err := ParseTimestamp("2024-08-04T23:30:00.000Z")
	require.NoError(t, err)

	// UTC.
	assert.Equal(t, "2024-08-04", FormatDate(ts, time.UTC))

	// Tokyo (UTC+9).
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	assert.Equal(t, "2024-08-05", FormatDate(ts, tokyo))
}

func TestWeekStart(t *testing.T) {
	// Jan 3, 2024 is a Wednesday.
	date := "2024-01-03"

	// Sunday start.
	ws, err := WeekStart(date, types.WeekSunday)
	require.NoError(t, err)
	assert.Equal(t, "2023-12-31", ws)

	// Monday start.
	ws, err = WeekStart(date, types.WeekMonday)
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01", ws)
}

func TestNormalizeDateBound(t *testing.T) {
	assert.Equal(t, "20260422", NormalizeDateBound("2026-04-22"))
	assert.Equal(t, "20260422", NormalizeDateBound("20260422"))
	assert.Equal(t, "", NormalizeDateBound(""))
}

func TestBucketKey(t *testing.T) {
	assert.Equal(t, "2026-04", BucketKey("2026-04-22", types.ReportMonthly, types.WeekMonday))

	// Weekly: April 22, 2026 is a Wednesday. Monday start = April 20.
	ws, err := WeekStart("2026-04-22", types.WeekMonday)
	require.NoError(t, err)
	assert.Equal(t, ws, BucketKey("2026-04-22", types.ReportWeekly, types.WeekMonday))
}

func TestParseISODate(t *testing.T) {
	ts, err := ParseISODate("2026-01-15")
	require.NoError(t, err)
	assert.Equal(t, 2026, ts.Year())
	assert.Equal(t, time.January, ts.Month())
	assert.Equal(t, 15, ts.Day())
}

func TestFormatUTCMinute(t *testing.T) {
	ts, err := ParseTimestamp("2024-08-04T23:30:45.000Z")
	require.NoError(t, err)
	assert.Equal(t, "2024-08-04 23:30", FormatUTCMinute(ts))
}

func TestFormatRFC3339Millis(t *testing.T) {
	ts, err := ParseTimestamp("2024-08-04T23:30:00.000Z")
	require.NoError(t, err)
	assert.Equal(t, "2024-08-04T23:30:00.000Z", FormatRFC3339Millis(ts))
}
