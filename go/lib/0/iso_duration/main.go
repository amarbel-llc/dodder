// Package iso_duration parses the date-only subset of ISO-8601 durations
// (PnY nM nW nD) and advances a YYYY-MM-DD date by such a duration. It exists
// so the commit-time on_commit_fields lua hooks can recur an actionable
// object's `due` date: gopher-lua ships no date library, so AdvanceDate is
// registered as a host function in the hook VM (see tag_blobs.MakeLuaSelfApplyV1).
package iso_duration

import (
	"fmt"
	"strconv"
	"time"
)

// DateFormat is the layout of the `due` field this package reads and writes:
// a date-only YYYY-MM-DD string.
const DateFormat = "2006-01-02"

// Duration is the parsed date-only subset of an ISO-8601 duration. Time
// components (hours/minutes/seconds) are intentionally unsupported.
type Duration struct {
	Years  int
	Months int
	Weeks  int
	Days   int
}

// ParseDuration parses the date-only subset of an ISO-8601 duration string of
// the form P[nY][nM][nW][nD] (e.g. "P1D", "P2W", "P1M", "P1Y", "P1M2W"). It
// errors on the empty string, a string not beginning with 'P', a "P" with no
// components, an unrecognised designator (including the time 'T' separator and
// H/M/S time units), or a number with no trailing unit.
func ParseDuration(s string) (duration Duration, err error) {
	if len(s) < 2 || s[0] != 'P' {
		err = fmt.Errorf(
			"invalid ISO-8601 duration %q: must be a non-empty string beginning with 'P'",
			s,
		)
		return duration, err
	}

	rest := s[1:]
	i := 0
	componentCount := 0

	for i < len(rest) {
		start := i

		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}

		if i == start {
			err = fmt.Errorf(
				"invalid ISO-8601 duration %q: expected a number before unit %q",
				s,
				string(rest[i]),
			)
			return duration, err
		}

		var n int

		if n, err = strconv.Atoi(rest[start:i]); err != nil {
			err = fmt.Errorf("invalid ISO-8601 duration %q: %w", s, err)
			return duration, err
		}

		if i >= len(rest) {
			err = fmt.Errorf(
				"invalid ISO-8601 duration %q: number %d has no trailing unit",
				s,
				n,
			)
			return duration, err
		}

		unit := rest[i]
		i++

		switch unit {
		case 'Y':
			duration.Years += n
		case 'M':
			duration.Months += n
		case 'W':
			duration.Weeks += n
		case 'D':
			duration.Days += n
		default:
			err = fmt.Errorf(
				"invalid ISO-8601 duration %q: unsupported designator %q (only Y/M/W/D are handled)",
				s,
				string(unit),
			)
			return duration, err
		}

		componentCount++
	}

	if componentCount == 0 {
		err = fmt.Errorf(
			"invalid ISO-8601 duration %q: no components",
			s,
		)
		return duration, err
	}

	return duration, err
}

// AdvanceDate advances dateStr (in DateFormat, YYYY-MM-DD) by the ISO-8601
// duration in durationStr (the PnY nM nW nD subset), returning the advanced
// date in the same format. The year+month part is applied first and clamps an
// overflowing day-of-month to the target month's last valid day rather than
// spilling forward (e.g. Jan 31 + P1M yields Feb 28, or Feb 29 in a leap year).
// The week+day part is exact and applied after the clamp. It errors on an
// unparseable date or duration.
func AdvanceDate(dateStr, durationStr string) (advanced string, err error) {
	var t time.Time

	if t, err = time.Parse(DateFormat, dateStr); err != nil {
		err = fmt.Errorf("invalid date %q (want YYYY-MM-DD): %w", dateStr, err)
		return advanced, err
	}

	var duration Duration

	if duration, err = ParseDuration(durationStr); err != nil {
		return advanced, err
	}

	// Compute the target year+month by normalizing the raw month index. time.Date
	// carries any out-of-range month into the year, so a December + P2M lands in
	// the following February.
	totalMonths := (int(t.Month()) - 1) +
		duration.Years*12 + duration.Months
	targetYear := t.Year() + totalMonths/12
	targetMonth := time.Month(totalMonths%12 + 1)

	if targetMonth < time.January {
		targetMonth += 12
		targetYear--
	}

	clampedDay := t.Day()

	if last := lastDayOfMonth(targetYear, targetMonth); clampedDay > last {
		clampedDay = last
	}

	advanced = time.Date(
		targetYear,
		targetMonth,
		clampedDay,
		0, 0, 0, 0,
		time.UTC,
	).AddDate(
		0,
		0,
		duration.Weeks*7+duration.Days,
	).Format(DateFormat)

	return advanced, err
}

// lastDayOfMonth returns the number of days in the given month by asking for day
// zero of the following month, which time.Date normalizes to the last day of
// the requested month.
func lastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
