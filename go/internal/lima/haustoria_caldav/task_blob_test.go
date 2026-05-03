//go:build test

package haustoria_caldav

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/caldav"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

func TestStatusValueRoundTrip(t1 *testing.T) {
	t := ui.T{T: t1}

	cases := []struct {
		vtodo string
		field string
	}{
		{vtodo: "", field: "todo"},
		{vtodo: "NEEDS-ACTION", field: "todo"},
		{vtodo: "IN-PROCESS", field: "in_progress"},
		{vtodo: "COMPLETED", field: "done"},
		{vtodo: "CANCELLED", field: "cancelled"},
	}

	for _, tc := range cases {
		got := mapVTODOStatusToFieldValue(tc.vtodo)
		if got != tc.field {
			t.Errorf("vtodo→field: input %q, expected %q, got %q", tc.vtodo, tc.field, got)
		}
	}

	// Reverse direction. Note: empty/NEEDS-ACTION both round-trip to "todo"
	// → "NEEDS-ACTION" (the canonical open-task VTODO STATUS).
	reverseCases := []struct {
		field string
		vtodo string
	}{
		{field: "todo", vtodo: "NEEDS-ACTION"},
		{field: "in_progress", vtodo: "IN-PROCESS"},
		{field: "done", vtodo: "COMPLETED"},
		{field: "cancelled", vtodo: "CANCELLED"},
	}

	for _, tc := range reverseCases {
		got := mapFieldValueToVTODOStatus(tc.field)
		if got != tc.vtodo {
			t.Errorf("field→vtodo: input %q, expected %q, got %q", tc.field, tc.vtodo, got)
		}
	}

	// Unknown values fall back to "todo" / "NEEDS-ACTION".
	if got := mapVTODOStatusToFieldValue("BOGUS"); got != "todo" {
		t.Errorf("unknown vtodo status: expected fallback %q, got %q", "todo", got)
	}

	if got := mapFieldValueToVTODOStatus("bogus"); got != "NEEDS-ACTION" {
		t.Errorf("unknown field status: expected fallback %q, got %q", "NEEDS-ACTION", got)
	}
}

func TestPriorityValueRoundTrip(t1 *testing.T) {
	t := ui.T{T: t1}

	// Canonical numeric values from the §1 priority table.
	canonicalCases := []struct {
		vtodo int
		field string
	}{
		{vtodo: 0, field: "p3"},
		{vtodo: 1, field: "p0"},
		{vtodo: 5, field: "p1"},
		{vtodo: 9, field: "p2"},
	}

	for _, tc := range canonicalCases {
		gotField := mapVTODOPriorityToFieldValue(tc.vtodo)
		if gotField != tc.field {
			t.Errorf("vtodo→field: input %d, expected %q, got %q", tc.vtodo, tc.field, gotField)
		}

		gotVtodo := mapFieldValueToVTODOPriority(tc.field)
		if gotVtodo != tc.vtodo {
			t.Errorf("field→vtodo: input %q, expected %d, got %d", tc.field, tc.vtodo, gotVtodo)
		}
	}

	// Out-of-band VTODO PRIORITY values bucket to the nearest canonical
	// field value. The boundaries are: 0=p3, 1-2=p0, 3-5=p1, 6-9=p2.
	bucketCases := []struct {
		vtodo int
		field string
	}{
		{vtodo: 2, field: "p0"},
		{vtodo: 3, field: "p1"},
		{vtodo: 4, field: "p1"},
		{vtodo: 6, field: "p2"},
		{vtodo: 7, field: "p2"},
		{vtodo: 8, field: "p2"},
	}

	for _, tc := range bucketCases {
		got := mapVTODOPriorityToFieldValue(tc.vtodo)
		if got != tc.field {
			t.Errorf("bucket: vtodo %d expected %q, got %q", tc.vtodo, tc.field, got)
		}
	}

	// Unknown field value falls back to no-priority.
	if got := mapFieldValueToVTODOPriority("bogus"); got != 0 {
		t.Errorf("unknown field priority: expected fallback 0, got %d", got)
	}
}

func TestBuildTaskTomlBlobBasic(t1 *testing.T) {
	t := ui.T{T: t1}

	task := &caldav.Task{
		Status:      "COMPLETED",
		Priority:    1,
		Due:         "20260415T120000Z",
		Description: "first line\nsecond line",
	}

	blob := buildTaskTomlBlob(task)

	expected := "status = \"done\"\n" +
		"priority = \"p0\"\n" +
		"due = \"20260415T120000Z\"\n" +
		"notes = \"first line\\nsecond line\"\n"

	if string(blob) != expected {
		t.Errorf("blob mismatch:\n--- expected ---\n%s\n--- actual ---\n%s",
			expected, string(blob))
	}
}

func TestBuildTaskTomlBlobEmptyDefaults(t1 *testing.T) {
	t := ui.T{T: t1}

	task := &caldav.Task{} // all zero values

	blob := buildTaskTomlBlob(task)

	expected := "status = \"todo\"\n" +
		"priority = \"p3\"\n" +
		"due = \"\"\n" +
		"notes = \"\"\n"

	if string(blob) != expected {
		t.Errorf("blob mismatch:\n--- expected ---\n%s\n--- actual ---\n%s",
			expected, string(blob))
	}
}

func TestParseTaskTomlBlobRoundTrip(t1 *testing.T) {
	t := ui.T{T: t1}

	task := &caldav.Task{
		Status:      "IN-PROCESS",
		Priority:    5,
		Due:         "20260420T080000Z",
		Description: "the notes\ncan span\nmultiple lines",
	}

	blob := buildTaskTomlBlob(task)
	values := parseTaskTomlBlob(blob)

	if values.Status != "in_progress" {
		t.Errorf("status: expected %q, got %q", "in_progress", values.Status)
	}

	if values.Priority != "p1" {
		t.Errorf("priority: expected %q, got %q", "p1", values.Priority)
	}

	if values.Due != "20260420T080000Z" {
		t.Errorf("due: expected %q, got %q", "20260420T080000Z", values.Due)
	}

	if values.Notes != "the notes\ncan span\nmultiple lines" {
		t.Errorf("notes: round-trip mismatch, got %q", values.Notes)
	}
}

func TestParseTaskTomlBlobIgnoresBlankAndCommentLines(t1 *testing.T) {
	t := ui.T{T: t1}

	blob := []byte("# comment\n" +
		"\n" +
		"status = \"done\"\n" +
		"  \n" +
		"# another comment\n" +
		"priority = \"p2\"\n")

	values := parseTaskTomlBlob(blob)

	if values.Status != "done" {
		t.Errorf("status: expected %q, got %q", "done", values.Status)
	}

	if values.Priority != "p2" {
		t.Errorf("priority: expected %q, got %q", "p2", values.Priority)
	}
}

func TestParseTaskTomlBlobUnknownKeysIgnored(t1 *testing.T) {
	t := ui.T{T: t1}

	blob := []byte("status = \"todo\"\n" +
		"unknown = \"some value\"\n" +
		"due = \"20260101T000000Z\"\n")

	values := parseTaskTomlBlob(blob)

	if values.Status != "todo" {
		t.Errorf("status: expected %q, got %q", "todo", values.Status)
	}

	if values.Due != "20260101T000000Z" {
		t.Errorf("due: expected %q, got %q", "20260101T000000Z", values.Due)
	}
}
