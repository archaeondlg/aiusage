// Package dateutil provides timezone-aware date formatting and parsing utilities.
package dateutil

import (
	"fmt"
	"strings"
	"time"

	"github.com/archhaeondlg/aiusage/internal/types"
)

// Common date format constants.
const (
	ISO8601Date    = "2006-01-02"
	ISO8601Compact = "20060102"
	UTCMinute      = "2006-01-02 15:04"
	UTCSecond      = "2006-01-02 15:04:05"
	RFC3339Millis  = "2006-01-02T15:04:05.000Z07:00"
	RFC3339Z       = "2006-01-02T15:04:05.000Z"
)

// ParseTimestamp parses an ISO 8601 / RFC 3339 timestamp string.
// Supports formats with and without timezone offsets.
func ParseTimestamp(s string) (time.Time, error) {
	// Try standard RFC3339 formats first.
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse timestamp: %s", s)
}

// ParseTZ parses a timezone string, returning UTC if empty or invalid.
func ParseTZ(tz *string) *time.Location {
	if tz == nil || *tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(*tz)
	if err != nil {
		// Try IANA name normalization
		loc, err = time.LoadLocation(strings.ReplaceAll(*tz, " ", "_"))
		if err != nil {
			return time.UTC
		}
	}
	return loc
}

// FormatDate formats a timestamp as YYYY-MM-DD in the given timezone.
func FormatDate(t time.Time, tz *time.Location) string {
	if tz == nil {
		tz = time.UTC
	}
	return t.In(tz).Format(ISO8601Date)
}

// FormatUTCMinute formats a timestamp as "2006-01-02 15:04" UTC.
func FormatUTCMinute(t time.Time) string {
	return t.UTC().Format(UTCMinute)
}

// FormatUTCSecond formats a timestamp as "2006-01-02 15:04:05" UTC.
func FormatUTCSecond(t time.Time) string {
	return t.UTC().Format(UTCSecond)
}

// FormatRFC3339Millis formats as "2006-01-02T15:04:05.000Z".
func FormatRFC3339Millis(t time.Time) string {
	return t.UTC().Format(RFC3339Z)
}

// FormatNaiveDate formats a time as "2006-01-02".
func FormatNaiveDate(t time.Time) string {
	return t.Format(ISO8601Date)
}

// ParseISODate parses a "YYYY-MM-DD" date string.
func ParseISODate(s string) (time.Time, error) {
	return time.Parse(ISO8601Date, s)
}

// NormalizeDateBound strips dashes from a date string for comparison.
// "2026-04-22" → "20260422"
func NormalizeDateBound(s string) string {
	return strings.ReplaceAll(s, "-", "")
}

// WeekStart returns the start date of the week containing the given date,
// based on the configured start-of-week day.
func WeekStart(date string, start types.WeekDay) (string, error) {
	t, err := ParseISODate(date)
	if err != nil {
		return "", err
	}

	// Go's Weekday: Sunday=0, Monday=1, ..., Saturday=6
	currentDay := int(t.Weekday())
	startDay := int(start)

	// Calculate days to go back.
	// If current day is already the start day, go back 0.
	shift := (currentDay - startDay + 7) % 7
	result := t.AddDate(0, 0, -shift)
	return result.Format(ISO8601Date), nil
}

// BucketKey returns the monthly or weekly bucket key for a date.
func BucketKey(date string, kind types.ReportKind, weekStart types.WeekDay) string {
	switch kind {
	case types.ReportMonthly:
		if len(date) >= 7 {
			return date[:7]
		}
		return date
	case types.ReportWeekly:
		if ws, err := WeekStart(date, weekStart); err == nil {
			return ws
		}
		return date
	default:
		return date
	}
}

// TimeFromMillis creates a UTC time from Unix milliseconds.
func TimeFromMillis(ms int64) time.Time {
	return time.Unix(ms/1000, (ms%1000)*1_000_000).UTC()
}

// TruncateToDate extracts just the "YYYY-MM-DD" portion from an RFC 3339 string.
func TruncateToDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
