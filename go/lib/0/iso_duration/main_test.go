//go:build test

package iso_duration

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestAdvanceDate(t1 *testing.T) {
	t := ui.MakeT(t1)

	tests := []struct {
		name     string
		date     string
		duration string
		want     string
	}{
		{"one day", "2026-07-01", "P1D", "2026-07-02"},
		{"one week advances seven days", "2026-07-01", "P1W", "2026-07-08"},
		{"two weeks", "2026-07-01", "P2W", "2026-07-15"},
		{"one month", "2026-07-01", "P1M", "2026-08-01"},
		{"three months", "2026-07-01", "P3M", "2026-10-01"},
		{"one year", "2026-07-01", "P1Y", "2027-07-01"},
		// Jan 31 + 1 month overflows non-leap Feb (28 days) and clamps to the
		// last valid day of February rather than spilling into March.
		{"month-end overflow clamps to Feb 28", "2026-01-31", "P1M", "2026-02-28"},
		// Jan 31 + 1 month in a leap year clamps to Feb 29.
		{"month-end overflow clamps to leap Feb 29", "2024-01-31", "P1M", "2024-02-29"},
		// Jan 31 + 1 year needs no clamp: 31 is valid in January.
		{"year advance no clamp", "2026-01-31", "P1Y", "2027-01-31"},
		// Jan 30 also overflows non-leap Feb and clamps to Feb 28.
		{"day thirty clamps to Feb 28", "2026-01-30", "P1M", "2026-02-28"},
		// Mar 31 + 1 month clamps to Apr 30 (April has 30 days).
		{"march-end clamps to Apr 30", "2026-03-31", "P1M", "2026-04-30"},
		// Feb 28 + 1 month anchors on the 28th, valid in March, no clamp.
		{"feb 28 anchors on 28th in march", "2026-02-28", "P1M", "2026-03-28"},
		// Weekly/daily advances stay exact even off a month-end date.
		{"week advance exact off month-end", "2026-01-31", "P1W", "2026-02-07"},
		{"day advance exact off month-end", "2026-01-31", "P1D", "2026-02-01"},
		// crossing a year boundary by month arithmetic.
		{"month arithmetic crosses year", "2026-12-15", "P1M", "2027-01-15"},
		// combined month + week component.
		{"combined month and weeks", "2026-07-01", "P1M2W", "2026-08-15"},
		// Month clamp happens BEFORE the exact week/day part: Jan 31 clamps to
		// Feb 28, then +7 days lands on Mar 7.
		{"month clamp then week add", "2026-01-31", "P1M1W", "2026-03-07"},
		// leap-year Feb 29 + 1 year clamps to Feb 28 (2025 is non-leap).
		{"leap day plus year clamps to Feb 28", "2024-02-29", "P1Y", "2025-02-28"},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			got, err := AdvanceDate(tt.date, tt.duration)
			t.AssertNoError(err)
			t.AssertEqualStrings(tt.want, got)
		})
	}
}

func TestAdvanceDateInvalidDuration(t1 *testing.T) {
	t := ui.MakeT(t1)

	tests := []struct {
		name     string
		date     string
		duration string
	}{
		{"empty duration", "2026-07-01", ""},
		{"missing P prefix", "2026-07-01", "1D"},
		{"P alone", "2026-07-01", "P"},
		{"number without unit", "2026-07-01", "P1"},
		{"unsupported designator", "2026-07-01", "P1X"},
		{"unit without number", "2026-07-01", "PD"},
		{"time component unsupported", "2026-07-01", "PT1H"},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			_, err := AdvanceDate(tt.date, tt.duration)
			t.AssertError(err)
		})
	}
}

func TestAdvanceDateInvalidDate(t1 *testing.T) {
	t := ui.MakeT(t1)

	tests := []struct {
		name string
		date string
	}{
		{"empty date", ""},
		{"ical datetime not date-only", "20260415T120000Z"},
		{"garbage", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			_, err := AdvanceDate(tt.date, "P1D")
			t.AssertError(err)
		})
	}
}

func TestParseDuration(t1 *testing.T) {
	t := ui.MakeT(t1)

	got, err := ParseDuration("P1Y2M3W4D")
	t.AssertNoError(err)
	t.AssertTrue(got.Years == 1, "years")
	t.AssertTrue(got.Months == 2, "months")
	t.AssertTrue(got.Weeks == 3, "weeks")
	t.AssertTrue(got.Days == 4, "days")
}
