package haustoria_caldav

import (
	"bytes"
	"fmt"
	"strconv"

	"code.linenisgreat.com/dodder/go/internal/hotel/caldav"
)

// buildTaskTomlBlob serializes a CalDAV VTODO into the TOML blob format
// expected by the !task / !chore built-in types. The blob contains four
// keys:
//
//   - status   : enum projected as a typed field via the !task fields-reader
//   - priority : enum projected as a typed field
//   - due      : string projected as a typed field
//   - notes    : free-form text — present in the blob but NOT declared as a
//     [[fields]] entry on !task. Holds the VTODO DESCRIPTION text. Promote
//     to a typed field once the round-trip story is proven.
//
// All four keys are always emitted (with empty-string defaults for due and
// notes) so the blob shape is stable across compile cycles.
func buildTaskTomlBlob(task *caldav.Task) []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "status = %s\n", quoteTomlString(mapVTODOStatusToFieldValue(task.Status)))
	fmt.Fprintf(&buf, "priority = %s\n", quoteTomlString(mapVTODOPriorityToFieldValue(task.Priority)))
	fmt.Fprintf(&buf, "due = %s\n", quoteTomlString(task.Due))
	fmt.Fprintf(&buf, "notes = %s\n", quoteTomlString(task.Description))

	return buf.Bytes()
}

// quoteTomlString emits a TOML basic string literal. strconv.Quote produces
// the same backslash-escape syntax that TOML basic strings accept (\", \\,
// \n, \r, \t, \uXXXX), with the exception of \a / \v which TOML doesn't
// support — but those don't appear in CalDAV text fields in practice.
func quoteTomlString(s string) string {
	return strconv.Quote(s)
}

// mapVTODOStatusToFieldValue maps the iCalendar VTODO STATUS property to a
// !task `status` field value per the table in
// docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md §3.
//
// Unknown values fall back to the "todo" default.
func mapVTODOStatusToFieldValue(vtodoStatus string) string {
	switch vtodoStatus {
	case "":
		return "todo"
	case "NEEDS-ACTION":
		return "todo"
	case "IN-PROCESS":
		return "in_progress"
	case "COMPLETED":
		return "done"
	case "CANCELLED":
		return "cancelled"
	default:
		return "todo"
	}
}

// mapFieldValueToVTODOStatus is the inverse of mapVTODOStatusToFieldValue,
// used by the decompile path. The "todo" field value rounds back to
// NEEDS-ACTION (the canonical "open task" VTODO STATUS).
func mapFieldValueToVTODOStatus(fieldValue string) string {
	switch fieldValue {
	case "todo":
		return "NEEDS-ACTION"
	case "in_progress":
		return "IN-PROCESS"
	case "done":
		return "COMPLETED"
	case "cancelled":
		return "CANCELLED"
	default:
		return "NEEDS-ACTION"
	}
}

// mapVTODOPriorityToFieldValue maps the numeric iCalendar VTODO PRIORITY
// property (0-9, where 0 means "no priority" and 1 is highest) to a !task
// `priority` field value per the table in §1 of the plan:
//
//	0 (or absent) → p3 (default)
//	1, 2          → p0 (highest, !!!)
//	3, 4, 5       → p1 (!!)
//	6, 7, 8, 9    → p2 (!)
//
// Out-of-range values fall back to the "p3" default.
func mapVTODOPriorityToFieldValue(vtodoPriority int) string {
	switch {
	case vtodoPriority <= 0:
		return "p3"
	case vtodoPriority <= 2:
		return "p0"
	case vtodoPriority <= 5:
		return "p1"
	case vtodoPriority <= 9:
		return "p2"
	default:
		return "p3"
	}
}

// mapFieldValueToVTODOPriority is the inverse of mapVTODOPriorityToFieldValue.
// Emits the canonical numeric value for each bucket (1, 5, 9) plus 0 for
// "no priority".
func mapFieldValueToVTODOPriority(fieldValue string) int {
	switch fieldValue {
	case "p0":
		return 1
	case "p1":
		return 5
	case "p2":
		return 9
	case "p3":
		return 0
	default:
		return 0
	}
}
