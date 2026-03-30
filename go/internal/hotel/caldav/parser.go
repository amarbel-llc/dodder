package caldav

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Task is the structured representation of a VTODO component, covering all
// fields that tasks.org uses.
type Task struct {
	UID             string   `json:"uid"`
	Summary         string   `json:"summary"`
	Description     string   `json:"description,omitempty"`
	Status          string   `json:"status,omitempty"`
	Priority        int      `json:"priority,omitempty"`
	Due             string   `json:"due,omitempty"`
	DtStart         string   `json:"dtstart,omitempty"`
	Completed       string   `json:"completed,omitempty"`
	Created         string   `json:"created,omitempty"`
	LastModified    string   `json:"last_modified,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	PercentComplete int      `json:"percent_complete,omitempty"`
	ParentUID       string   `json:"parent_uid,omitempty"`
	RRule           string   `json:"rrule,omitempty"`
	Location        string   `json:"location,omitempty"`
	Geo             string   `json:"geo,omitempty"`
	SortOrder          int      `json:"sort_order,omitempty"`
	Sequence           int      `json:"sequence,omitempty"`
	Attachments        []string `json:"attachments,omitempty"`
	StructuredLocation string   `json:"structured_location,omitempty"`

	// Derived fields
	SubtaskUIDs       []string `json:"subtask_uids,omitempty"`
	Alarms            []Alarm  `json:"alarms,omitempty"`
	HasDescription    bool     `json:"has_description"`
	DescriptionTokens int      `json:"description_tokens"`

	// Server metadata
	Href string `json:"href,omitempty"`
	ETag string `json:"etag,omitempty"`
}

// Alarm represents a VALARM component.
type Alarm struct {
	Trigger     string `json:"trigger"`
	Action      string `json:"action"`
	Description string `json:"description,omitempty"`
}

// TaskMetadata is the lightweight tier-1 view of a task.
type TaskMetadata struct {
	UID               string   `json:"uid"`
	Summary           string   `json:"summary"`
	Status            string   `json:"status,omitempty"`
	Priority          int      `json:"priority,omitempty"`
	Due               string   `json:"due,omitempty"`
	Categories        []string `json:"categories,omitempty"`
	RRule             string   `json:"rrule,omitempty"`
	HasDescription    bool     `json:"has_description"`
	DescriptionTokens int      `json:"description_tokens"`
	ParentUID         string   `json:"parent_uid,omitempty"`
	AttachmentCount   int      `json:"attachment_count,omitempty"`
	Location          string   `json:"location,omitempty"`
}

// ToMetadata converts a full Task to its lightweight metadata representation.
func (t *Task) ToMetadata() TaskMetadata {
	return TaskMetadata{
		UID:               t.UID,
		Summary:           t.Summary,
		Status:            t.Status,
		Priority:          t.Priority,
		Due:               t.Due,
		Categories:        t.Categories,
		RRule:             t.RRule,
		HasDescription:    t.HasDescription,
		DescriptionTokens: t.DescriptionTokens,
		ParentUID:         t.ParentUID,
		AttachmentCount:   len(t.Attachments),
		Location:          t.Location,
	}
}

// ParseVTODO parses a raw iCalendar string and extracts the first VTODO as a Task.
func ParseVTODO(raw string) (*Task, error) {
	lines := unfoldLines(raw)

	inVTODO := false
	inVALARM := false
	t := &Task{}
	var currentAlarm *Alarm

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if trimmed == "BEGIN:VTODO" {
			inVTODO = true
			continue
		}
		if trimmed == "END:VTODO" {
			inVTODO = false
			break
		}
		if !inVTODO {
			continue
		}

		if trimmed == "BEGIN:VALARM" {
			inVALARM = true
			currentAlarm = &Alarm{}
			continue
		}
		if trimmed == "END:VALARM" {
			inVALARM = false
			if currentAlarm != nil {
				t.Alarms = append(t.Alarms, *currentAlarm)
				currentAlarm = nil
			}
			continue
		}

		name, value := parsePropLine(trimmed)

		if inVALARM && currentAlarm != nil {
			switch propName(name) {
			case "TRIGGER":
				currentAlarm.Trigger = value
			case "ACTION":
				currentAlarm.Action = value
			case "DESCRIPTION":
				currentAlarm.Description = value
			}
			continue
		}

		switch propName(name) {
		case "UID":
			t.UID = value
		case "SUMMARY":
			t.Summary = value
		case "DESCRIPTION":
			t.Description = value
		case "STATUS":
			t.Status = value
		case "PRIORITY":
			if n, err := strconv.Atoi(value); err == nil {
				t.Priority = n
			}
		case "DUE":
			t.Due = value
		case "DTSTART":
			t.DtStart = value
		case "COMPLETED":
			t.Completed = value
		case "CREATED":
			t.Created = value
		case "LAST-MODIFIED":
			t.LastModified = value
		case "CATEGORIES":
			cats := strings.Split(value, ",")
			for i := range cats {
				cats[i] = strings.TrimSpace(cats[i])
			}
			t.Categories = cats
		case "PERCENT-COMPLETE":
			if n, err := strconv.Atoi(value); err == nil {
				t.PercentComplete = n
			}
		case "RELATED-TO":
			relType := paramValue(name, "RELTYPE")
			if relType == "" || relType == "PARENT" {
				t.ParentUID = value
			}
		case "RRULE":
			t.RRule = value
		case "LOCATION":
			t.Location = value
		case "GEO":
			t.Geo = value
		case "X-APPLE-SORT-ORDER":
			if n, err := strconv.Atoi(value); err == nil {
				t.SortOrder = n
			}
		case "SEQUENCE":
			if n, err := strconv.Atoi(value); err == nil {
				t.Sequence = n
			}
		case "ATTACH":
			t.Attachments = append(t.Attachments, value)
		case "X-APPLE-STRUCTURED-LOCATION":
			t.StructuredLocation = value
		}
	}

	if t.UID == "" {
		return nil, fmt.Errorf("VTODO missing UID")
	}

	t.HasDescription = t.Description != ""
	t.DescriptionTokens = len(t.Description) / 4

	return t, nil
}

// unfoldLines handles iCalendar line folding (RFC 5545 §3.1):
// continuation lines start with a space or tab.
func unfoldLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	rawLines := strings.Split(raw, "\n")

	var result []string
	for _, line := range rawLines {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if len(result) > 0 {
				result[len(result)-1] += line[1:]
			}
		} else {
			result = append(result, line)
		}
	}
	return result
}

// parsePropLine splits "NAME;PARAM=VAL:value" into the full name part and value.
func parsePropLine(line string) (name, value string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return line, ""
	}
	return line[:idx], line[idx+1:]
}

// propName extracts just the property name (before any parameters).
func propName(nameWithParams string) string {
	idx := strings.Index(nameWithParams, ";")
	if idx < 0 {
		return nameWithParams
	}
	return nameWithParams[:idx]
}

// paramValue extracts a named parameter value from the name portion.
// e.g., paramValue("RELATED-TO;RELTYPE=PARENT", "RELTYPE") returns "PARENT".
func paramValue(nameWithParams, paramName string) string {
	parts := strings.Split(nameWithParams, ";")
	for _, part := range parts[1:] {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.EqualFold(kv[0], paramName) {
			return kv[1]
		}
	}
	return ""
}

// TaskToIcal serializes a Task to a full VCALENDAR string.
func TaskToIcal(t *Task) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//amarbel-llc//caldav-mcp//EN\r\n")
	b.WriteString("BEGIN:VTODO\r\n")

	writeIcalProp(&b, "UID", t.UID)
	writeIcalProp(&b, "DTSTAMP", formatNow())
	writeIcalProp(&b, "SUMMARY", t.Summary)

	if t.Description != "" {
		writeIcalProp(&b, "DESCRIPTION", t.Description)
	}
	if t.Status != "" {
		writeIcalProp(&b, "STATUS", t.Status)
	}
	if t.Priority > 0 {
		writeIcalProp(&b, "PRIORITY", strconv.Itoa(t.Priority))
	}
	if t.Due != "" {
		writeDateProp(&b, "DUE", t.Due)
	}
	if t.DtStart != "" {
		writeDateProp(&b, "DTSTART", t.DtStart)
	}
	if t.Completed != "" {
		writeIcalProp(&b, "COMPLETED", t.Completed)
	}
	if t.PercentComplete > 0 {
		writeIcalProp(&b, "PERCENT-COMPLETE", strconv.Itoa(t.PercentComplete))
	}
	if len(t.Categories) > 0 {
		writeIcalProp(&b, "CATEGORIES", strings.Join(t.Categories, ","))
	}
	if t.ParentUID != "" {
		b.WriteString("RELATED-TO;RELTYPE=PARENT:" + t.ParentUID + "\r\n")
	}
	if t.RRule != "" {
		writeIcalProp(&b, "RRULE", t.RRule)
	}
	if t.Location != "" {
		writeIcalProp(&b, "LOCATION", t.Location)
	}
	if t.Geo != "" {
		writeIcalProp(&b, "GEO", t.Geo)
	}
	if t.SortOrder != 0 {
		writeIcalProp(&b, "X-APPLE-SORT-ORDER", strconv.Itoa(t.SortOrder))
	}
	if t.Sequence > 0 {
		writeIcalProp(&b, "SEQUENCE", strconv.Itoa(t.Sequence))
	}

	for _, alarm := range t.Alarms {
		b.WriteString("BEGIN:VALARM\r\n")
		writeIcalProp(&b, "TRIGGER", alarm.Trigger)
		writeIcalProp(&b, "ACTION", alarm.Action)
		if alarm.Description != "" {
			writeIcalProp(&b, "DESCRIPTION", alarm.Description)
		}
		b.WriteString("END:VALARM\r\n")
	}

	b.WriteString("END:VTODO\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}

func writeIcalProp(b *strings.Builder, name, value string) {
	b.WriteString(name + ":" + value + "\r\n")
}

func writeDateProp(b *strings.Builder, name, value string) {
	// If it looks like a date-only value (YYYYMMDD or YYYY-MM-DD), use VALUE=DATE
	if len(value) == 8 || len(value) == 10 {
		normalized := strings.ReplaceAll(value, "-", "")
		if len(normalized) == 8 {
			b.WriteString(name + ";VALUE=DATE:" + normalized + "\r\n")
			return
		}
	}
	writeIcalProp(b, name, value)
}

func formatNow() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

// CapDescription returns the description capped at maxLen characters.
func CapDescription(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	return desc[:maxLen] + "..."
}
