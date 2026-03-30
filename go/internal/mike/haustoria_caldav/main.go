package haustoria_caldav

import (
	"fmt"
	"time"

	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/hotel/caldav"
)

// Store implements haustoria.Haustoria for CalDAV servers.
type Store struct {
	client       *caldav.Client
	calendarHref string
}

var _ haustoria.Haustoria = &Store{}

func MakeStore(cfg *caldav.Config, calendarHref string) *Store {
	return &Store{
		client:       caldav.NewClient(cfg),
		calendarHref: calendarHref,
	}
}

func (s *Store) Compile(req haustoria.CompileRequest) (haustoria.CompileResult, error) {
	task := caldav.Task{
		UID:         req.ExternalId,
		Summary:     req.Description,
		Description: string(req.Blob),
		Categories:  req.Tags,
		Status:      "NEEDS-ACTION",
	}

	if task.UID == "" {
		task.UID = fmt.Sprintf("dodder-%d@dodder", time.Now().UnixNano())
	}

	ical := caldav.TaskToIcal(&task)
	href := s.calendarHref + task.UID + ".ics"

	err := s.client.PutTask(href, ical, req.ETag)
	if err != nil {
		return haustoria.CompileResult{}, fmt.Errorf("compile to CalDAV: %w", err)
	}

	meta, err := s.client.GetTask(href)
	if err != nil {
		return haustoria.CompileResult{
			ExternalId: task.UID,
		}, nil
	}

	return haustoria.CompileResult{
		ExternalId: task.UID,
		ETag:       meta.Task.ETag,
	}, nil
}

func (s *Store) Decompile(req haustoria.DecompileRequest) (haustoria.DecompileResult, error) {
	result, err := s.client.ListTasks(s.calendarHref)
	if err != nil {
		return haustoria.DecompileResult{}, fmt.Errorf("decompile from CalDAV: %w", err)
	}

	for _, twm := range result.Tasks {
		if twm.Task.UID != req.ExternalId {
			continue
		}

		return haustoria.DecompileResult{
			ExternalId:  twm.Task.UID,
			Description: twm.Task.Summary,
			Blob:        []byte(twm.Task.Description),
			Tags:        twm.Task.Categories,
			TypeId:      "task",
			ETag:        twm.Task.ETag,
		}, nil
	}

	return haustoria.DecompileResult{}, fmt.Errorf(
		"CalDAV task not found: %s", req.ExternalId,
	)
}

func (s *Store) Discover() ([]haustoria.ExternalResource, error) {
	result, err := s.client.ListTasks(s.calendarHref)
	if err != nil {
		return nil, fmt.Errorf("discover CalDAV tasks: %w", err)
	}

	resources := make([]haustoria.ExternalResource, 0, len(result.Tasks))
	for _, twm := range result.Tasks {
		resources = append(resources, haustoria.ExternalResource{
			ExternalId:  twm.Task.UID,
			TypeId:      "task",
			Description: twm.Task.Summary,
		})
	}

	return resources, nil
}

func (s *Store) Delete(externalId string) error {
	href := s.calendarHref + externalId + ".ics"
	return s.client.DeleteTask(href, "")
}

func (s *Store) Status() (haustoria.StatusResult, error) {
	resources, err := s.Discover()
	if err != nil {
		return haustoria.StatusResult{}, err
	}

	return haustoria.StatusResult{
		StoreType:         "caldav",
		ExternalResources: resources,
	}, nil
}
