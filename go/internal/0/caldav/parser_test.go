//go:build test

package caldav

import (
	"strings"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestUnescapeText(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"escaped newline lowercase", `line1\nline2`, "line1\nline2"},
		{"escaped newline uppercase", `line1\Nline2`, "line1\nline2"},
		{"escaped comma", `one\,two`, "one,two"},
		{"escaped semicolon", `one\;two`, "one;two"},
		{"escaped backslash", `path\\file`, `path\file`},
		{"multiple escapes", `hello\, world\; foo\\bar\nbaz`, "hello, world; foo\\bar\nbaz"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			got := unescapeText(tt.input)
			t.AssertEqualStrings(tt.want, got)
		})
	}
}

func TestEscapeText(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"newline", "line1\nline2", `line1\nline2`},
		{"comma", "one,two", `one\,two`},
		{"semicolon", "one;two", `one\;two`},
		{"backslash", `path\file`, `path\\file`},
		{"multiple special chars", "hello, world; foo\\bar\nbaz", `hello\, world\; foo\\bar\nbaz`},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			got := escapeText(tt.input)
			t.AssertEqualStrings(tt.want, got)
		})
	}
}

func TestEscapeUnescapeRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	inputs := []string{
		"simple text",
		"text with, commas; and semicolons",
		"multi\nline\ntext",
		`backslash \ in text`,
		"all special: \\, ;, comma, \n newline",
		"",
	}

	for _, input := range inputs {
		escaped := escapeText(input)
		unescaped := unescapeText(escaped)
		t.AssertEqualStrings(input, unescaped)
	}
}

func TestUnfoldLines(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			"no folding",
			"SUMMARY:hello\r\nUID:123",
			[]string{"SUMMARY:hello", "UID:123"},
		},
		{
			"space continuation",
			"DESCRIPTION:long\r\n  text here",
			[]string{"DESCRIPTION:long text here"},
		},
		{
			"tab continuation",
			"DESCRIPTION:long\r\n\t text here",
			[]string{"DESCRIPTION:long text here"},
		},
		{
			"multiple continuations",
			"DESCRIPTION:a\r\n b\r\n c",
			[]string{"DESCRIPTION:abc"},
		},
		{
			"bare LF",
			"SUMMARY:hello\nUID:123",
			[]string{"SUMMARY:hello", "UID:123"},
		},
		{
			"bare CR",
			"SUMMARY:hello\rUID:123",
			[]string{"SUMMARY:hello", "UID:123"},
		},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			got := unfoldLines(tt.input)
			// Filter empty trailing lines
			var filtered []string
			for _, l := range got {
				if l != "" {
					filtered = append(filtered, l)
				}
			}
			if len(filtered) != len(tt.want) {
				t.Fatalf("unfoldLines(%q) got %d lines, want %d: %v", tt.input, len(filtered), len(tt.want), filtered)
			}
			for i := range filtered {
				t.AssertEqualStrings(tt.want[i], filtered[i])
			}
		})
	}
}

func TestFoldAndWrite(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.Run(ui.MakeTestCaseInfo("short line not folded"), func(t *ui.T) {
		var b strings.Builder
		foldAndWrite(&b, "SUMMARY:short")
		got := b.String()
		want := "SUMMARY:short\r\n"
		t.AssertEqualStrings(want, got)
	})

	t.Run(ui.MakeTestCaseInfo("exactly 75 octets not folded"), func(t *ui.T) {
		line := "SUMMARY:" + strings.Repeat("x", 67) // 8 + 67 = 75
		var b strings.Builder
		foldAndWrite(&b, line)
		got := b.String()
		want := line + "\r\n"
		t.AssertEqualStrings(want, got)
	})

	t.Run(ui.MakeTestCaseInfo("76 octets folded"), func(t *ui.T) {
		line := "SUMMARY:" + strings.Repeat("x", 68) // 8 + 68 = 76
		var b strings.Builder
		foldAndWrite(&b, line)
		got := b.String()
		// Should fold after 75 octets
		if !strings.Contains(got, "\r\n ") {
			t.Errorf("expected fold in output, got %q", got)
		}
		// Unfolding should recover original
		unfolded := unfoldLines(got)
		var nonEmpty []string
		for _, l := range unfolded {
			if l != "" {
				nonEmpty = append(nonEmpty, l)
			}
		}
		if len(nonEmpty) != 1 || nonEmpty[0] != line {
			t.Errorf("unfold(fold(%q)) = %q", line, nonEmpty)
		}
	})

	t.Run(ui.MakeTestCaseInfo("long line multiple folds"), func(t *ui.T) {
		line := "DESCRIPTION:" + strings.Repeat("a", 200)
		var b strings.Builder
		foldAndWrite(&b, line)
		got := b.String()
		// Each physical line should be at most 75 octets (continuation lines
		// include the leading space in their count).
		for _, physical := range strings.Split(got, "\r\n") {
			if physical == "" {
				continue
			}
			if len(physical) > 75 {
				t.Errorf("physical line too long (%d octets): %q", len(physical), physical)
			}
		}
		// Round-trip
		unfolded := unfoldLines(got)
		var nonEmpty []string
		for _, l := range unfolded {
			if l != "" {
				nonEmpty = append(nonEmpty, l)
			}
		}
		if len(nonEmpty) != 1 || nonEmpty[0] != line {
			t.Errorf("unfold(fold) round-trip failed")
		}
	})
}

func TestParsePropLine(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		input     string
		wantName  string
		wantValue string
	}{
		{"SUMMARY:hello", "SUMMARY", "hello"},
		{"ATTACH;VALUE=URI:https://example.com", "ATTACH;VALUE=URI", "https://example.com"},
		{"DUE;VALUE=DATE:20260405", "DUE;VALUE=DATE", "20260405"},
		{"NOCOLON", "NOCOLON", ""},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.input), func(t *ui.T) {
			name, value := parsePropLine(tt.input)
			t.AssertEqualStrings(tt.wantName, name)
			t.AssertEqualStrings(tt.wantValue, value)
		})
	}
}

func TestWriteDateProp(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"date YYYYMMDD", "20260405", "DUE;VALUE=DATE:20260405\r\n"},
		{"date YYYY-MM-DD", "2026-04-05", "DUE;VALUE=DATE:20260405\r\n"},
		{"datetime", "20260405T120000Z", "DUE:20260405T120000Z\r\n"},
		{"datetime with tz", "20260405T120000", "DUE:20260405T120000\r\n"},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			var b strings.Builder
			writeDateProp(&b, "DUE", tt.value, "")
			got := b.String()
			t.AssertEqualStrings(tt.want, got)
		})
	}
}

func TestParseVTODOCategories(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.Run(ui.MakeTestCaseInfo("multiple CATEGORIES lines accumulated"), func(t *ui.T) {
		raw := "BEGIN:VCALENDAR\r\n" +
			"BEGIN:VTODO\r\n" +
			"UID:test-1\r\n" +
			"SUMMARY:test\r\n" +
			"CATEGORIES:work,urgent\r\n" +
			"CATEGORIES:project-x\r\n" +
			"CATEGORIES:team-a,review\r\n" +
			"END:VTODO\r\n" +
			"END:VCALENDAR\r\n"

		task, err := ParseVTODO(raw)
		t.AssertNoError(err)
		want := []string{"work", "urgent", "project-x", "team-a", "review"}
		if len(task.Categories) != len(want) {
			t.Fatalf("got %d categories %v, want %d %v",
				len(task.Categories), task.Categories, len(want), want)
		}
		for i, cat := range task.Categories {
			t.AssertEqualStrings(want[i], cat)
		}
	})
}

func TestParseVEVENTCategories(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.Run(ui.MakeTestCaseInfo("multiple CATEGORIES lines accumulated"), func(t *ui.T) {
		raw := "BEGIN:VCALENDAR\r\n" +
			"BEGIN:VEVENT\r\n" +
			"UID:event-1\r\n" +
			"SUMMARY:test event\r\n" +
			"CATEGORIES:meeting\r\n" +
			"CATEGORIES:important,recurring\r\n" +
			"END:VEVENT\r\n" +
			"END:VCALENDAR\r\n"

		event, err := ParseVEVENT(raw)
		t.AssertNoError(err)
		want := []string{"meeting", "important", "recurring"}
		if len(event.Categories) != len(want) {
			t.Fatalf("got %d categories %v, want %d %v",
				len(event.Categories), event.Categories, len(want), want)
		}
		for i, cat := range event.Categories {
			t.AssertEqualStrings(want[i], cat)
		}
	})
}

func TestParseVTODOEscapedText(t1 *testing.T) {
	t := ui.MakeT(t1)
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:esc-1\r\n" +
		"SUMMARY:Meeting\\, Q2 review\r\n" +
		"DESCRIPTION:Line 1\\nLine 2\\nDone\\; next steps\r\n" +
		"LOCATION:Room 42\\, Building A\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	t.AssertNoError(err)

	t.AssertEqualStrings("Meeting, Q2 review", task.Summary)
	t.AssertEqualStrings("Line 1\nLine 2\nDone; next steps", task.Description)
	t.AssertEqualStrings("Room 42, Building A", task.Location)
}

func TestTaskToIcalEscaping(t1 *testing.T) {
	t := ui.MakeT(t1)
	task := &Task{
		UID:         "esc-rt-1",
		Summary:     "Buy milk, eggs; bread",
		Description: "Line 1\nLine 2\nhas backslash \\ too",
		Location:    "Store, downtown; mall",
	}

	ical := TaskToIcal(task)

	// The serialized output should contain escaped versions
	if !strings.Contains(ical, `Buy milk\, eggs\; bread`) {
		t.Errorf("Summary not escaped in output:\n%s", ical)
	}
	if !strings.Contains(ical, `Line 1\nLine 2\nhas backslash \\ too`) {
		t.Errorf("Description not escaped in output:\n%s", ical)
	}
	if !strings.Contains(ical, `Store\, downtown\; mall`) {
		t.Errorf("Location not escaped in output:\n%s", ical)
	}

	// Round-trip: parse the serialized output back
	parsed, err := ParseVTODO(ical)
	t.AssertNoError(err)
	t.AssertEqualStrings(task.Summary, parsed.Summary)
	t.AssertEqualStrings(task.Description, parsed.Description)
	t.AssertEqualStrings(task.Location, parsed.Location)
}

func TestWriteDatePropWithTZID(t1 *testing.T) {
	t := ui.MakeT(t1)
	tests := []struct {
		name  string
		value string
		tzid  string
		want  string
	}{
		{"no tzid", "20260405T120000Z", "", "DUE:20260405T120000Z\r\n"},
		{"with tzid", "20260405T120000", "America/New_York", "DUE;TZID=America/New_York:20260405T120000\r\n"},
		{"date-only ignores tzid", "20260405", "America/New_York", "DUE;VALUE=DATE:20260405\r\n"},
	}

	for _, tt := range tests {
		t.Run(ui.MakeTestCaseInfo(tt.name), func(t *ui.T) {
			var b strings.Builder
			writeDateProp(&b, "DUE", tt.value, tt.tzid)
			got := b.String()
			t.AssertEqualStrings(tt.want, got)
		})
	}
}

func TestParseVTODOTZIDPreservation(t1 *testing.T) {
	t := ui.MakeT(t1)
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:tzid-1\r\n" +
		"SUMMARY:Meeting prep\r\n" +
		"DUE;TZID=America/New_York:20260405T170000\r\n" +
		"DTSTART;TZID=Europe/Berlin:20260405T090000\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	t.AssertNoError(err)

	t.AssertEqualStrings("20260405T170000", task.Due)
	t.AssertEqualStrings("America/New_York", task.DueTZID)
	t.AssertEqualStrings("20260405T090000", task.DtStart)
	t.AssertEqualStrings("Europe/Berlin", task.DtStartTZID)

	// Round-trip: serialize and re-parse
	ical := TaskToIcal(task)
	if !strings.Contains(ical, "DUE;TZID=America/New_York:20260405T170000") {
		t.Errorf("TZID not preserved in serialized output:\n%s", ical)
	}
	if !strings.Contains(ical, "DTSTART;TZID=Europe/Berlin:20260405T090000") {
		t.Errorf("DTSTART TZID not preserved in serialized output:\n%s", ical)
	}

	parsed, err := ParseVTODO(ical)
	t.AssertNoError(err)
	t.AssertEqualStrings("America/New_York", parsed.DueTZID)
	t.AssertEqualStrings("Europe/Berlin", parsed.DtStartTZID)
}

func TestParseVTODODateTimeFormats(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.Run(ui.MakeTestCaseInfo("UTC datetime"), func(t *ui.T) {
		raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:dt-1\r\nSUMMARY:test\r\n" +
			"DUE:20260405T120000Z\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
		task, err := ParseVTODO(raw)
		t.AssertNoError(err)
		t.AssertEqualStrings("20260405T120000Z", task.Due)
		t.AssertEqualStrings("", task.DueTZID)
	})

	t.Run(ui.MakeTestCaseInfo("floating datetime"), func(t *ui.T) {
		raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:dt-2\r\nSUMMARY:test\r\n" +
			"DUE:20260405T120000\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
		task, err := ParseVTODO(raw)
		t.AssertNoError(err)
		t.AssertEqualStrings("20260405T120000", task.Due)
		t.AssertEqualStrings("", task.DueTZID)
	})

	t.Run(ui.MakeTestCaseInfo("date-only VALUE=DATE"), func(t *ui.T) {
		raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:dt-3\r\nSUMMARY:test\r\n" +
			"DUE;VALUE=DATE:20260405\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
		task, err := ParseVTODO(raw)
		t.AssertNoError(err)
		t.AssertEqualStrings("20260405", task.Due)
	})
}

func TestParseVTODORecurrenceID(t1 *testing.T) {
	t := ui.MakeT(t1)
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:rec-1\r\n" +
		"SUMMARY:Weekly review\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=FR\r\n" +
		"RECURRENCE-ID:20260410T090000Z\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	t.AssertNoError(err)
	t.AssertEqualStrings("FREQ=WEEKLY;BYDAY=FR", task.RRule)
	t.AssertEqualStrings("20260410T090000Z", task.RecurrenceID)

	// Round-trip
	ical := TaskToIcal(task)
	if !strings.Contains(ical, "RECURRENCE-ID:20260410T090000Z") {
		t.Errorf("RecurrenceID not in serialized output:\n%s", ical)
	}
}

func TestParseVTODOMultiComponentMaster(t1 *testing.T) {
	t := ui.MakeT(t1)
	// A .ics with a master VTODO and an override — only master is extracted.
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:multi-1\r\n" +
		"SUMMARY:Take out trash\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=TU\r\n" +
		"END:VTODO\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:multi-1\r\n" +
		"SUMMARY:Take out trash (deferred)\r\n" +
		"RECURRENCE-ID:20260408T080000Z\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	t.AssertNoError(err)
	// Should get the master, not the override
	t.AssertEqualStrings("Take out trash", task.Summary)
	t.AssertEqualStrings("", task.RecurrenceID)
	t.AssertEqualStrings("FREQ=WEEKLY;BYDAY=TU", task.RRule)
}

func TestParseVEVENTTZIDPreservation(t1 *testing.T) {
	t := ui.MakeT(t1)
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:ev-tzid-1\r\n" +
		"SUMMARY:Standup\r\n" +
		"DTSTART;TZID=America/Chicago:20260405T090000\r\n" +
		"DTEND;TZID=America/Chicago:20260405T093000\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	event, err := ParseVEVENT(raw)
	t.AssertNoError(err)

	t.AssertEqualStrings("America/Chicago", event.DtStartTZID)
	t.AssertEqualStrings("America/Chicago", event.DtEndTZID)

	// Round-trip
	ical := EventToIcal(event)
	if !strings.Contains(ical, "DTSTART;TZID=America/Chicago:20260405T090000") {
		t.Errorf("DTSTART TZID not preserved:\n%s", ical)
	}
	if !strings.Contains(ical, "DTEND;TZID=America/Chicago:20260405T093000") {
		t.Errorf("DTEND TZID not preserved:\n%s", ical)
	}
}
