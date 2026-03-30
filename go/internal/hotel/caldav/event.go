package caldav

import (
	"fmt"
	"strconv"
	"strings"
)

// Event is the structured representation of a VEVENT component.
type Event struct {
	UID          string   `json:"uid"`
	Summary      string   `json:"summary"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status,omitempty"`
	Location     string   `json:"location,omitempty"`
	Geo          string   `json:"geo,omitempty"`
	DtStart      string   `json:"dtstart,omitempty"`
	DtEnd        string   `json:"dtend,omitempty"`
	Duration     string   `json:"duration,omitempty"`
	Organizer    string   `json:"organizer,omitempty"`
	Attendees    []string `json:"attendees,omitempty"`
	Categories   []string `json:"categories,omitempty"`
	RRule        string   `json:"rrule,omitempty"`
	RecurrenceID string   `json:"recurrence_id,omitempty"`
	Sequence     int      `json:"sequence,omitempty"`
	Transp       string   `json:"transp,omitempty"`
	Created      string   `json:"created,omitempty"`
	LastModified string   `json:"last_modified,omitempty"`

	// Derived fields
	Alarms            []Alarm `json:"alarms,omitempty"`
	HasDescription    bool    `json:"has_description"`
	DescriptionTokens int     `json:"description_tokens"`

	// Server metadata
	Href string `json:"href,omitempty"`
	ETag string `json:"etag,omitempty"`
}

// EventMetadata is the lightweight tier-1 view of an event.
type EventMetadata struct {
	UID               string   `json:"uid"`
	Summary           string   `json:"summary"`
	Status            string   `json:"status,omitempty"`
	DtStart           string   `json:"dtstart,omitempty"`
	DtEnd             string   `json:"dtend,omitempty"`
	Location          string   `json:"location,omitempty"`
	Categories        []string `json:"categories,omitempty"`
	RRule             string   `json:"rrule,omitempty"`
	HasDescription    bool     `json:"has_description"`
	DescriptionTokens int      `json:"description_tokens"`
}

// ToMetadata converts a full Event to its lightweight metadata representation.
func (e *Event) ToMetadata() EventMetadata {
	return EventMetadata{
		UID:               e.UID,
		Summary:           e.Summary,
		Status:            e.Status,
		DtStart:           e.DtStart,
		DtEnd:             e.DtEnd,
		Location:          e.Location,
		Categories:        e.Categories,
		RRule:             e.RRule,
		HasDescription:    e.HasDescription,
		DescriptionTokens: e.DescriptionTokens,
	}
}

// EventWithMeta pairs a parsed Event with its raw iCalendar text.
type EventWithMeta struct {
	Event Event
	Raw   string
}

// EventListResult holds the results of listing events, including any parse
// errors for individual events that could not be parsed.
type EventListResult struct {
	Events      []EventWithMeta
	ParseErrors []string
}

// ParseVEVENT parses a raw iCalendar string and extracts the first VEVENT as an Event.
func ParseVEVENT(raw string) (*Event, error) {
	lines := unfoldLines(raw)

	inVEVENT := false
	inVALARM := false
	e := &Event{}
	var currentAlarm *Alarm

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if trimmed == "BEGIN:VEVENT" {
			inVEVENT = true
			continue
		}
		if trimmed == "END:VEVENT" {
			inVEVENT = false
			break
		}
		if !inVEVENT {
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
				e.Alarms = append(e.Alarms, *currentAlarm)
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
			e.UID = value
		case "SUMMARY":
			e.Summary = value
		case "DESCRIPTION":
			e.Description = value
		case "STATUS":
			e.Status = value
		case "LOCATION":
			e.Location = value
		case "GEO":
			e.Geo = value
		case "DTSTART":
			e.DtStart = value
		case "DTEND":
			e.DtEnd = value
		case "DURATION":
			e.Duration = value
		case "ORGANIZER":
			e.Organizer = value
		case "ATTENDEE":
			e.Attendees = append(e.Attendees, value)
		case "CATEGORIES":
			cats := strings.Split(value, ",")
			for i := range cats {
				cats[i] = strings.TrimSpace(cats[i])
			}
			e.Categories = cats
		case "RRULE":
			e.RRule = value
		case "RECURRENCE-ID":
			e.RecurrenceID = value
		case "TRANSP":
			e.Transp = value
		case "CREATED":
			e.Created = value
		case "LAST-MODIFIED":
			e.LastModified = value
		case "SEQUENCE":
			if n, err := strconv.Atoi(value); err == nil {
				e.Sequence = n
			}
		}
	}

	if e.UID == "" {
		return nil, fmt.Errorf("VEVENT missing UID")
	}

	e.HasDescription = e.Description != ""
	e.DescriptionTokens = len(e.Description) / 4

	return e, nil
}

// EventToIcal serializes an Event to a full VCALENDAR string.
func EventToIcal(e *Event) string {
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//amarbel-llc//caldav-mcp//EN\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")

	writeIcalProp(&b, "UID", e.UID)
	writeIcalProp(&b, "DTSTAMP", formatNow())
	writeIcalProp(&b, "SUMMARY", e.Summary)

	if e.Description != "" {
		writeIcalProp(&b, "DESCRIPTION", e.Description)
	}
	if e.Status != "" {
		writeIcalProp(&b, "STATUS", e.Status)
	}
	if e.DtStart != "" {
		writeIcalProp(&b, "DTSTART", e.DtStart)
	}
	if e.DtEnd != "" {
		writeIcalProp(&b, "DTEND", e.DtEnd)
	}
	if e.Duration != "" {
		writeIcalProp(&b, "DURATION", e.Duration)
	}
	if e.Location != "" {
		writeIcalProp(&b, "LOCATION", e.Location)
	}
	if e.Geo != "" {
		writeIcalProp(&b, "GEO", e.Geo)
	}
	if len(e.Categories) > 0 {
		writeIcalProp(&b, "CATEGORIES", strings.Join(e.Categories, ","))
	}
	if e.RRule != "" {
		writeIcalProp(&b, "RRULE", e.RRule)
	}
	if e.RecurrenceID != "" {
		writeIcalProp(&b, "RECURRENCE-ID", e.RecurrenceID)
	}
	if e.Transp != "" {
		writeIcalProp(&b, "TRANSP", e.Transp)
	}
	if e.Organizer != "" {
		writeIcalProp(&b, "ORGANIZER", e.Organizer)
	}
	for _, attendee := range e.Attendees {
		writeIcalProp(&b, "ATTENDEE", attendee)
	}
	if e.Sequence > 0 {
		writeIcalProp(&b, "SEQUENCE", strconv.Itoa(e.Sequence))
	}

	for _, alarm := range e.Alarms {
		b.WriteString("BEGIN:VALARM\r\n")
		writeIcalProp(&b, "TRIGGER", alarm.Trigger)
		writeIcalProp(&b, "ACTION", alarm.Action)
		if alarm.Description != "" {
			writeIcalProp(&b, "DESCRIPTION", alarm.Description)
		}
		b.WriteString("END:VALARM\r\n")
	}

	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return b.String()
}
