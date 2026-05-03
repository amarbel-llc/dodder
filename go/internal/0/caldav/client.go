package caldav

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const requestTimeout = 30 * time.Second

// Client is a CalDAV HTTP client that supports PROPFIND, REPORT, PUT, DELETE,
// and MKCALENDAR operations.
type Client struct {
	cfg  *Config
	http *http.Client
}

// Calendar represents a CalDAV calendar collection.
type Calendar struct {
	Href           string   `json:"href"`
	DisplayName    string   `json:"display_name"`
	Color          string   `json:"color,omitempty"`
	Description    string   `json:"description,omitempty"`
	ComponentTypes []string `json:"component_types,omitempty"`
}

// NewClient creates a CalDAV client from the given configuration.
func NewClient(cfg *Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (c *Client) do(method, url, body string, depth int) (*http.Response, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	if depth >= 0 {
		req.Header.Set("Depth", fmt.Sprintf("%d", depth))
	}
	return c.http.Do(req)
}

// --- XML types for CalDAV responses ---

type multistatusResponse struct {
	XMLName   xml.Name      `xml:"DAV: multistatus"`
	Responses []davResponse `xml:"DAV: response"`
}

type davResponse struct {
	Href     string        `xml:"DAV: href"`
	PropStat []davPropStat `xml:"DAV: propstat"`
}

type davPropStat struct {
	Prop   davProp `xml:"DAV: prop"`
	Status string  `xml:"DAV: status"`
}

type davProp struct {
	DisplayName      string           `xml:"DAV: displayname"`
	CalendarColor    string           `xml:"http://apple.com/ns/ical/ calendar-color"`
	CalendarDesc     string           `xml:"urn:ietf:params:xml:ns:caldav calendar-description"`
	SupportedCalComp *calCompSet      `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set"`
	CalendarData     string           `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
	GetETag          string           `xml:"DAV: getetag"`
	ResourceType     *davResourceType `xml:"DAV: resourcetype"`
}

type davResourceType struct {
	Calendar *struct{} `xml:"urn:ietf:params:xml:ns:caldav calendar"`
}

type calCompSet struct {
	Comps []calComp `xml:"urn:ietf:params:xml:ns:caldav comp"`
}

type calComp struct {
	Name string `xml:"name,attr"`
}

// ListCalendars performs a PROPFIND to discover all calendar collections.
func (c *Client) ListCalendars() ([]Calendar, error) {
	body := `<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:" xmlns:cs="urn:ietf:params:xml:ns:caldav" xmlns:ic="http://apple.com/ns/ical/">
  <d:prop>
    <d:displayname />
    <d:resourcetype />
    <ic:calendar-color />
    <cs:calendar-description />
    <cs:supported-calendar-component-set />
  </d:prop>
</d:propfind>`

	resp, err := c.do("PROPFIND", c.cfg.URL, body, 1)
	if err != nil {
		return nil, fmt.Errorf("PROPFIND: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var ms multistatusResponse
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("parsing multistatus: %w", err)
	}

	var calendars []Calendar
	for _, r := range ms.Responses {
		for _, ps := range r.PropStat {
			if !strings.Contains(ps.Status, "200") {
				continue
			}
			if ps.Prop.ResourceType == nil || ps.Prop.ResourceType.Calendar == nil {
				continue
			}

			cal := Calendar{
				Href:        r.Href,
				DisplayName: ps.Prop.DisplayName,
				Color:       ps.Prop.CalendarColor,
				Description: ps.Prop.CalendarDesc,
			}
			if ps.Prop.SupportedCalComp != nil {
				for _, comp := range ps.Prop.SupportedCalComp.Comps {
					cal.ComponentTypes = append(cal.ComponentTypes, comp.Name)
				}
			}
			calendars = append(calendars, cal)
		}
	}
	return calendars, nil
}

// ListTasks performs a REPORT calendar-query to list all VTODOs in a calendar.
// Parse errors for individual tasks are collected in TaskListResult.ParseErrors
// rather than causing the entire listing to fail.
//
// All tasks are returned regardless of STATUS. Server-side filtering via
// prop-filter is deferred until incremental sync is implemented — the
// haustoria's status-tags mapping requires completed tasks during checkin.
// https://github.com/amarbel-llc/dodder/issues/87
func (c *Client) ListTasks(calendarHref string) (*TaskListResult, error) {
	body := `<?xml version="1.0" encoding="utf-8" ?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag />
    <c:calendar-data />
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VTODO" />
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`

	url := c.resolveHref(calendarHref)
	resp, err := c.do("REPORT", url, body, 1)
	if err != nil {
		return nil, fmt.Errorf("REPORT: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var ms multistatusResponse
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("parsing multistatus: %w", err)
	}

	result := &TaskListResult{}
	for _, r := range ms.Responses {
		for _, ps := range r.PropStat {
			if !strings.Contains(ps.Status, "200") || ps.Prop.CalendarData == "" {
				continue
			}
			task, err := parseIcalString(ps.Prop.CalendarData)
			if err != nil {
				result.ParseErrors = append(result.ParseErrors,
					fmt.Sprintf("%s: %v", r.Href, err))
				continue
			}
			task.Href = r.Href
			task.ETag = ps.Prop.GetETag
			result.Tasks = append(result.Tasks, TaskWithMeta{
				Task: *task,
				Raw:  ps.Prop.CalendarData,
			})
		}
	}
	return result, nil
}

// ListEvents performs a REPORT calendar-query to list all VEVENTs in a calendar.
// Parse errors for individual events are collected in EventListResult.ParseErrors
// rather than causing the entire listing to fail.
func (c *Client) ListEvents(calendarHref string) (*EventListResult, error) {
	body := `<?xml version="1.0" encoding="utf-8" ?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag />
    <c:calendar-data />
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT" />
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`

	url := c.resolveHref(calendarHref)
	resp, err := c.do("REPORT", url, body, 1)
	if err != nil {
		return nil, fmt.Errorf("REPORT: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var ms multistatusResponse
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("parsing multistatus: %w", err)
	}

	result := &EventListResult{}
	for _, r := range ms.Responses {
		for _, ps := range r.PropStat {
			if !strings.Contains(ps.Status, "200") || ps.Prop.CalendarData == "" {
				continue
			}
			event, err := ParseVEVENT(ps.Prop.CalendarData)
			if err != nil {
				result.ParseErrors = append(result.ParseErrors,
					fmt.Sprintf("%s: %v", r.Href, err))
				continue
			}
			event.Href = r.Href
			event.ETag = ps.Prop.GetETag
			result.Events = append(result.Events, EventWithMeta{
				Event: *event,
				Raw:   ps.Prop.CalendarData,
			})
		}
	}
	return result, nil
}

// TaskWithMeta pairs a parsed Task with its raw iCalendar text.
type TaskWithMeta struct {
	Task Task
	Raw  string
}

// TaskListResult holds the results of listing tasks, including any parse errors
// for individual tasks that could not be parsed.
type TaskListResult struct {
	Tasks       []TaskWithMeta
	ParseErrors []string
}

// GetTask fetches a single VTODO by its href.
func (c *Client) GetTask(taskHref string) (*TaskWithMeta, error) {
	url := c.resolveHref(taskHref)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", taskHref, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	raw := string(data)
	task, err := parseIcalString(raw)
	if err != nil {
		return nil, err
	}
	task.Href = taskHref
	task.ETag = resp.Header.Get("ETag")

	return &TaskWithMeta{Task: *task, Raw: raw}, nil
}

// PutTask creates or updates a VTODO at the given href.
func (c *Client) PutTask(taskHref, icalData, etag string) error {
	url := c.resolveHref(taskHref)
	req, err := http.NewRequest("PUT", url, strings.NewReader(icalData))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	if etag != "" {
		req.Header.Set("If-Match", etag)
	} else {
		req.Header.Set("If-None-Match", "*")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PUT: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT %s: status %d: %s", taskHref, resp.StatusCode, string(body))
	}
	return nil
}

// DeleteTask removes a VTODO by href.
func (c *Client) DeleteTask(taskHref, etag string) error {
	url := c.resolveHref(taskHref)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	if etag != "" {
		req.Header.Set("If-Match", etag)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE %s: status %d", taskHref, resp.StatusCode)
	}
	return nil
}

// MkCalendar creates a new calendar collection via MKCALENDAR.
func (c *Client) MkCalendar(href, displayName, description string) error {
	body := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" ?>
<c:mkcalendar xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:set>
    <d:prop>
      <d:displayname>%s</d:displayname>
      <c:calendar-description>%s</c:calendar-description>
      <c:supported-calendar-component-set>
        <c:comp name="VTODO" />
      </c:supported-calendar-component-set>
    </d:prop>
  </d:set>
</c:mkcalendar>`, xmlEscape(displayName), xmlEscape(description))

	url := c.resolveHref(href)
	resp, err := c.do("MKCALENDAR", url, body, -1)
	if err != nil {
		return fmt.Errorf("MKCALENDAR: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("MKCALENDAR %s: status %d: %s", href, resp.StatusCode, string(respBody))
	}
	return nil
}

// FindTaskByUID searches all calendars for a task with the given UID.
// Returns the task with metadata and the calendar href it was found in.
func (c *Client) FindTaskByUID(uid string) (*TaskWithMeta, string, error) {
	calendars, err := c.ListCalendars()
	if err != nil {
		return nil, "", err
	}

	for _, cal := range calendars {
		result, err := c.ListTasks(cal.Href)
		if err != nil {
			continue
		}
		for _, t := range result.Tasks {
			if t.Task.UID == uid {
				return &t, cal.Href, nil
			}
		}
	}
	return nil, "", fmt.Errorf("task with UID %q not found", uid)
}

// GetEvent fetches a single VEVENT by its href.
func (c *Client) GetEvent(eventHref string) (*EventWithMeta, error) {
	url := c.resolveHref(eventHref)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", eventHref, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	raw := string(data)
	event, err := ParseVEVENT(raw)
	if err != nil {
		return nil, err
	}
	event.Href = eventHref
	event.ETag = resp.Header.Get("ETag")

	return &EventWithMeta{Event: *event, Raw: raw}, nil
}

// PutEvent creates or updates a VEVENT at the given href.
func (c *Client) PutEvent(eventHref, icalData, etag string) error {
	url := c.resolveHref(eventHref)
	req, err := http.NewRequest("PUT", url, strings.NewReader(icalData))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	req.Header.Set("Content-Type", "text/calendar; charset=utf-8")
	if etag != "" {
		req.Header.Set("If-Match", etag)
	} else {
		req.Header.Set("If-None-Match", "*")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("PUT: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT %s: status %d: %s", eventHref, resp.StatusCode, string(body))
	}
	return nil
}

// DeleteEvent removes a VEVENT by href.
func (c *Client) DeleteEvent(eventHref, etag string) error {
	url := c.resolveHref(eventHref)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	if etag != "" {
		req.Header.Set("If-Match", etag)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE %s: status %d", eventHref, resp.StatusCode)
	}
	return nil
}

// FindEventByUID searches all calendars for an event with the given UID.
// Returns the event with metadata and the calendar href it was found in.
func (c *Client) FindEventByUID(uid string) (*EventWithMeta, string, error) {
	calendars, err := c.ListCalendars()
	if err != nil {
		return nil, "", err
	}

	for _, cal := range calendars {
		result, err := c.ListEvents(cal.Href)
		if err != nil {
			continue
		}
		for _, e := range result.Events {
			if e.Event.UID == uid {
				return &e, cal.Href, nil
			}
		}
	}
	return nil, "", fmt.Errorf("event with UID %q not found", uid)
}

func (c *Client) resolveHref(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	// href is a path — resolve against the base URL
	base := strings.TrimRight(c.cfg.URL, "/")
	// If the href starts with /, use just the scheme+host
	if strings.HasPrefix(href, "/") {
		// Extract scheme+host from base URL
		idx := strings.Index(base, "://")
		if idx >= 0 {
			rest := base[idx+3:]
			slashIdx := strings.Index(rest, "/")
			if slashIdx >= 0 {
				return base[:idx+3+slashIdx] + href
			}
		}
		return base + href
	}
	return base + "/" + href
}

func parseIcalString(data string) (*Task, error) {
	return ParseVTODO(data)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
