//go:build test

package caldav

import (
	"strings"
	"testing"
)

func TestUnescapeText(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeText(tt.input)
			if got != tt.want {
				t.Errorf("unescapeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeText(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			got := escapeText(tt.input)
			if got != tt.want {
				t.Errorf("escapeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
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
		if unescaped != input {
			t.Errorf("round-trip failed: %q -> %q -> %q", input, escaped, unescaped)
		}
	}
}

func TestUnfoldLines(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
				if filtered[i] != tt.want[i] {
					t.Errorf("line %d: got %q, want %q", i, filtered[i], tt.want[i])
				}
			}
		})
	}
}

func TestFoldAndWrite(t *testing.T) {
	t.Run("short line not folded", func(t *testing.T) {
		var b strings.Builder
		foldAndWrite(&b, "SUMMARY:short")
		got := b.String()
		want := "SUMMARY:short\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("exactly 75 octets not folded", func(t *testing.T) {
		line := "SUMMARY:" + strings.Repeat("x", 67) // 8 + 67 = 75
		var b strings.Builder
		foldAndWrite(&b, line)
		got := b.String()
		want := line + "\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("76 octets folded", func(t *testing.T) {
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

	t.Run("long line multiple folds", func(t *testing.T) {
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

func TestParsePropLine(t *testing.T) {
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
		t.Run(tt.input, func(t *testing.T) {
			name, value := parsePropLine(tt.input)
			if name != tt.wantName || value != tt.wantValue {
				t.Errorf("parsePropLine(%q) = (%q, %q), want (%q, %q)",
					tt.input, name, value, tt.wantName, tt.wantValue)
			}
		})
	}
}

func TestWriteDateProp(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			writeDateProp(&b, "DUE", tt.value, "")
			got := b.String()
			if got != tt.want {
				t.Errorf("writeDateProp(DUE, %q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestParseVTODOCategories(t *testing.T) {
	t.Run("multiple CATEGORIES lines accumulated", func(t *testing.T) {
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
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"work", "urgent", "project-x", "team-a", "review"}
		if len(task.Categories) != len(want) {
			t.Fatalf("got %d categories %v, want %d %v",
				len(task.Categories), task.Categories, len(want), want)
		}
		for i, cat := range task.Categories {
			if cat != want[i] {
				t.Errorf("category %d: got %q, want %q", i, cat, want[i])
			}
		}
	})
}

func TestParseVEVENTCategories(t *testing.T) {
	t.Run("multiple CATEGORIES lines accumulated", func(t *testing.T) {
		raw := "BEGIN:VCALENDAR\r\n" +
			"BEGIN:VEVENT\r\n" +
			"UID:event-1\r\n" +
			"SUMMARY:test event\r\n" +
			"CATEGORIES:meeting\r\n" +
			"CATEGORIES:important,recurring\r\n" +
			"END:VEVENT\r\n" +
			"END:VCALENDAR\r\n"

		event, err := ParseVEVENT(raw)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"meeting", "important", "recurring"}
		if len(event.Categories) != len(want) {
			t.Fatalf("got %d categories %v, want %d %v",
				len(event.Categories), event.Categories, len(want), want)
		}
		for i, cat := range event.Categories {
			if cat != want[i] {
				t.Errorf("category %d: got %q, want %q", i, cat, want[i])
			}
		}
	})
}

func TestParseVTODOEscapedText(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:esc-1\r\n" +
		"SUMMARY:Meeting\\, Q2 review\r\n" +
		"DESCRIPTION:Line 1\\nLine 2\\nDone\\; next steps\r\n" +
		"LOCATION:Room 42\\, Building A\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	if err != nil {
		t.Fatal(err)
	}

	if task.Summary != "Meeting, Q2 review" {
		t.Errorf("Summary: got %q, want %q", task.Summary, "Meeting, Q2 review")
	}
	if task.Description != "Line 1\nLine 2\nDone; next steps" {
		t.Errorf("Description: got %q, want %q", task.Description, "Line 1\nLine 2\nDone; next steps")
	}
	if task.Location != "Room 42, Building A" {
		t.Errorf("Location: got %q, want %q", task.Location, "Room 42, Building A")
	}
}

func TestTaskToIcalEscaping(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Summary != task.Summary {
		t.Errorf("round-trip Summary: got %q, want %q", parsed.Summary, task.Summary)
	}
	if parsed.Description != task.Description {
		t.Errorf("round-trip Description: got %q, want %q", parsed.Description, task.Description)
	}
	if parsed.Location != task.Location {
		t.Errorf("round-trip Location: got %q, want %q", parsed.Location, task.Location)
	}
}

func TestWriteDatePropWithTZID(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			writeDateProp(&b, "DUE", tt.value, tt.tzid)
			got := b.String()
			if got != tt.want {
				t.Errorf("writeDateProp(DUE, %q, %q) = %q, want %q", tt.value, tt.tzid, got, tt.want)
			}
		})
	}
}

func TestParseVTODOTZIDPreservation(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:tzid-1\r\n" +
		"SUMMARY:Meeting prep\r\n" +
		"DUE;TZID=America/New_York:20260405T170000\r\n" +
		"DTSTART;TZID=Europe/Berlin:20260405T090000\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	if err != nil {
		t.Fatal(err)
	}

	if task.Due != "20260405T170000" {
		t.Errorf("Due: got %q, want %q", task.Due, "20260405T170000")
	}
	if task.DueTZID != "America/New_York" {
		t.Errorf("DueTZID: got %q, want %q", task.DueTZID, "America/New_York")
	}
	if task.DtStart != "20260405T090000" {
		t.Errorf("DtStart: got %q, want %q", task.DtStart, "20260405T090000")
	}
	if task.DtStartTZID != "Europe/Berlin" {
		t.Errorf("DtStartTZID: got %q, want %q", task.DtStartTZID, "Europe/Berlin")
	}

	// Round-trip: serialize and re-parse
	ical := TaskToIcal(task)
	if !strings.Contains(ical, "DUE;TZID=America/New_York:20260405T170000") {
		t.Errorf("TZID not preserved in serialized output:\n%s", ical)
	}
	if !strings.Contains(ical, "DTSTART;TZID=Europe/Berlin:20260405T090000") {
		t.Errorf("DTSTART TZID not preserved in serialized output:\n%s", ical)
	}

	parsed, err := ParseVTODO(ical)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.DueTZID != "America/New_York" {
		t.Errorf("round-trip DueTZID: got %q, want %q", parsed.DueTZID, "America/New_York")
	}
	if parsed.DtStartTZID != "Europe/Berlin" {
		t.Errorf("round-trip DtStartTZID: got %q, want %q", parsed.DtStartTZID, "Europe/Berlin")
	}
}

func TestParseVTODODateTimeFormats(t *testing.T) {
	t.Run("UTC datetime", func(t *testing.T) {
		raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:dt-1\r\nSUMMARY:test\r\n" +
			"DUE:20260405T120000Z\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
		task, err := ParseVTODO(raw)
		if err != nil {
			t.Fatal(err)
		}
		if task.Due != "20260405T120000Z" {
			t.Errorf("Due: got %q", task.Due)
		}
		if task.DueTZID != "" {
			t.Errorf("DueTZID should be empty for UTC, got %q", task.DueTZID)
		}
	})

	t.Run("floating datetime", func(t *testing.T) {
		raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:dt-2\r\nSUMMARY:test\r\n" +
			"DUE:20260405T120000\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
		task, err := ParseVTODO(raw)
		if err != nil {
			t.Fatal(err)
		}
		if task.Due != "20260405T120000" {
			t.Errorf("Due: got %q", task.Due)
		}
		if task.DueTZID != "" {
			t.Errorf("DueTZID should be empty for floating, got %q", task.DueTZID)
		}
	})

	t.Run("date-only VALUE=DATE", func(t *testing.T) {
		raw := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:dt-3\r\nSUMMARY:test\r\n" +
			"DUE;VALUE=DATE:20260405\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
		task, err := ParseVTODO(raw)
		if err != nil {
			t.Fatal(err)
		}
		if task.Due != "20260405" {
			t.Errorf("Due: got %q", task.Due)
		}
	})
}

func TestParseVTODORecurrenceID(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VTODO\r\n" +
		"UID:rec-1\r\n" +
		"SUMMARY:Weekly review\r\n" +
		"RRULE:FREQ=WEEKLY;BYDAY=FR\r\n" +
		"RECURRENCE-ID:20260410T090000Z\r\n" +
		"END:VTODO\r\n" +
		"END:VCALENDAR\r\n"

	task, err := ParseVTODO(raw)
	if err != nil {
		t.Fatal(err)
	}
	if task.RRule != "FREQ=WEEKLY;BYDAY=FR" {
		t.Errorf("RRule: got %q", task.RRule)
	}
	if task.RecurrenceID != "20260410T090000Z" {
		t.Errorf("RecurrenceID: got %q, want %q", task.RecurrenceID, "20260410T090000Z")
	}

	// Round-trip
	ical := TaskToIcal(task)
	if !strings.Contains(ical, "RECURRENCE-ID:20260410T090000Z") {
		t.Errorf("RecurrenceID not in serialized output:\n%s", ical)
	}
}

func TestParseVTODOMultiComponentMaster(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}
	// Should get the master, not the override
	if task.Summary != "Take out trash" {
		t.Errorf("expected master VTODO, got Summary=%q", task.Summary)
	}
	if task.RecurrenceID != "" {
		t.Errorf("master should have no RecurrenceID, got %q", task.RecurrenceID)
	}
	if task.RRule != "FREQ=WEEKLY;BYDAY=TU" {
		t.Errorf("RRule: got %q", task.RRule)
	}
}

func TestParseVEVENTTZIDPreservation(t *testing.T) {
	raw := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\n" +
		"UID:ev-tzid-1\r\n" +
		"SUMMARY:Standup\r\n" +
		"DTSTART;TZID=America/Chicago:20260405T090000\r\n" +
		"DTEND;TZID=America/Chicago:20260405T093000\r\n" +
		"END:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	event, err := ParseVEVENT(raw)
	if err != nil {
		t.Fatal(err)
	}

	if event.DtStartTZID != "America/Chicago" {
		t.Errorf("DtStartTZID: got %q", event.DtStartTZID)
	}
	if event.DtEndTZID != "America/Chicago" {
		t.Errorf("DtEndTZID: got %q", event.DtEndTZID)
	}

	// Round-trip
	ical := EventToIcal(event)
	if !strings.Contains(ical, "DTSTART;TZID=America/Chicago:20260405T090000") {
		t.Errorf("DTSTART TZID not preserved:\n%s", ical)
	}
	if !strings.Contains(ical, "DTEND;TZID=America/Chicago:20260405T093000") {
		t.Errorf("DTEND TZID not preserved:\n%s", ical)
	}
}
