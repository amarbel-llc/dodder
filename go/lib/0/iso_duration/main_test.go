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
		// Jan 31 + 1 month overflows non-leap Feb (28 days) and normalizes
		// forward to early March via time.AddDate.
		{"month-end overflow normalizes forward", "2026-01-31", "P1M", "2026-03-03"},
		// crossing a year boundary by month arithmetic.
		{"month arithmetic crosses year", "2026-12-15", "P1M", "2027-01-15"},
		// combined month + week component.
		{"combined month and weeks", "2026-07-01", "P1M2W", "2026-08-15"},
		// leap-year Feb 29 + 1 year normalizes to Mar 1 (2025 is non-leap).
		{"leap day plus year", "2024-02-29", "P1Y", "2025-03-01"},
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
