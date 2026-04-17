package haustoria_caldav

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/caldav"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// CalendarMapping associates a CalDAV calendar URL with a dodder type and
// optional tags.
type CalendarMapping struct {
	URL    string
	TypeId string
	Tags   []string
}

// Store implements both haustoria.Haustoria and store_workspace.StoreLike
// for CalDAV servers. Supports multiple calendars with per-calendar type
// and tag mappings.
type Store struct {
	client    *caldav.Client
	calendars []CalendarMapping
	supplies  store_workspace.Supplies
}

var (
	_ haustoria.Haustoria       = &Store{}
	_ store_workspace.StoreLike = &Store{}
)

func MakeStore(cfg *caldav.Config, calendars []CalendarMapping) *Store {
	return &Store{
		client:    caldav.NewClient(cfg),
		calendars: calendars,
	}
}

// --- StoreLike interface ---

func (s *Store) Initialize(supplies store_workspace.Supplies) error {
	s.supplies = supplies
	return nil
}

func (s *Store) QueryCheckedOut(
	queryGroup *queries.Query,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	// Build a lookup of existing bindings: ExternalObjectId string → Transacted.
	// This enables detecting which CalDAV resources already have dodder objects
	// (CheckedOut) vs which are new (Untracked).
	bindings := make(map[string]*sku.Transacted)

	if err = s.supplies.ReadPrimitiveQuery(
		nil,
		func(object *sku.Transacted) error {
			if object.GetObjectId().GetGenre() != genres.Zettel {
				return nil
			}

			eoid := object.GetExternalObjectId()
			if eoid.IsEmpty() {
				return nil
			}

			cloned, _ := object.CloneTransacted() //repool:owned
			bindings[eoid.String()] = cloned
			return nil
		},
	); err != nil {
		return errors.Wrap(err)
	}

	for _, cal := range s.calendars {
		if err = s.queryCheckedOutForCalendar(cal, bindings, output); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) queryCheckedOutForCalendar(
	cal CalendarMapping,
	bindings map[string]*sku.Transacted,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	taskResult, err := s.client.ListTasks(cal.URL)
	if err != nil {
		return errors.Wrapf(err, "list CalDAV tasks from %s", cal.URL)
	}

	for _, twm := range taskResult.Tasks {
		co, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned

		external := co.GetSkuExternal()
		metadata := external.GetMetadataMutable()

		if err = metadata.GetDescriptionMutable().Set(
			twm.Task.Summary,
		); err != nil {
			return errors.Wrap(err)
		}

		if cal.TypeId != "" {
			if err = metadata.GetTypeMutable().SetType(
				cal.TypeId,
			); err != nil {
				return errors.Wrap(err)
			}
		}

		// Add per-calendar tags from the mapping.
		for _, tagStr := range cal.Tags {
			if addErr := metadata.AddTagString(tagStr); addErr != nil {
				continue
			}
		}

		// Add tags from CalDAV CATEGORIES.
		for _, tagStr := range twm.Task.Categories {
			if addErr := metadata.AddTagString(tagStr); addErr != nil {
				continue
			}
		}

		// Build the !task TOML blob with status, priority, due baked in
		// (the reader script declared on !task projects them into
		// Metadata.Index.Fields during commit). The VTODO DESCRIPTION text
		// is stored as a `notes` key in the blob — present in the file but
		// NOT declared as a [[fields]] entry on !task, so it doesn't
		// appear as a queryable field for now (#TODO promote to field
		// once the round-trip story is proven).
		blob := buildTaskTomlBlob(&twm.Task)
		if err = s.writeBlob(external, blob); err != nil {
			return errors.Wrapf(err, "write task blob for %s", twm.Task.UID)
		}

		if err = external.GetExternalObjectIdMutable().SetWithGenre(
			twm.Task.UID,
			genres.Zettel,
		); err != nil {
			return errors.Wrap(err)
		}

		// Check if this CalDAV resource has an existing dodder binding.
		if bound, ok := bindings[twm.Task.UID]; ok {
			// Copy the zettel ID from the committed object onto the
			// external side so it appears in status/show output.
			external.GetObjectIdMutable().ResetWithObjectId(bound.GetObjectId())

			sku.TransactedResetter.ResetWith(co.GetSku(), bound)

			// Copy the type lock signature from the committed object onto
			// the external side. The haustoria assigns the type from
			// workspace config (not CalDAV), so the external type lock
			// should match the internal one. Without this, the signature
			// mismatch causes a perpetual "changed" state (#111).
			boundTypeLock := bound.GetMetadata().GetTypeLock()
			external.GetMetadataMutable().GetTypeLockMutable().GetValueMutable().ResetWithMarklId(
				boundTypeLock.GetValue(),
			)

			co.SetState(checked_out_state.CheckedOut)
		} else {
			co.GetSku().GetObjectIdMutable().SetGenre(genres.Zettel)
			co.SetState(checked_out_state.Untracked)
		}

		if err = output(co); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

func (s *Store) CheckoutOne(
	options checkout_options.Options,
	transactedGetter sku.TransactedGetter,
) (checkedOut sku.SkuType, err error) {
	object := transactedGetter.GetSku()

	description := object.GetMetadata().GetDescription().String()
	typeId := object.GetMetadata().GetType().String()

	var tags []string
	for tag := range object.GetMetadata().GetTags().All() {
		tags = append(tags, tag.String())
	}

	var blob []byte
	if !object.GetBlobDigest().IsNull() {
		if blob, err = s.readBlob(object); err != nil {
			return nil, errors.Wrapf(err,
				"read blob for %s", object.GetObjectId(),
			)
		}
	}

	// Preserve the CalDAV UID across the round-trip. Without this,
	// Decompile would generate a fresh `dodder-N@dodder` UID, breaking
	// the binding between the dodder object and its remote VTODO and
	// orphaning the original .ics file on the server.
	externalId := object.GetExternalObjectId().String()

	result, err := s.Decompile(haustoria.DecompileRequest{
		ObjectId:    object.GetObjectId().String(),
		ExternalId:  externalId,
		Description: description,
		Blob:        blob,
		Tags:        tags,
		TypeId:      typeId,
	})
	if err != nil {
		return nil, errors.Wrapf(err,
			"decompile %s to CalDAV", object.GetObjectId(),
		)
	}

	co, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned

	sku.TransactedResetter.ResetWith(co.GetSku(), object)

	external := co.GetSkuExternal()
	sku.TransactedResetter.ResetWith(external, object)

	if err = external.GetExternalObjectIdMutable().SetWithGenre(
		result.ExternalId,
		genres.Zettel,
	); err != nil {
		return nil, errors.Wrap(err)
	}

	co.SetState(checked_out_state.CheckedOut)

	return co, nil
}

func (s *Store) readBlob(object *sku.Transacted) (content []byte, err error) {
	blobReader, err := s.supplies.Env.GetDefaultBlobStore().MakeBlobReader(
		object.GetBlobDigest(),
	)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	if content, err = io.ReadAll(blobReader); err != nil {
		return nil, errors.Wrap(err)
	}

	return content, nil
}

func (s *Store) ReadAllExternalItems() error {
	return nil
}

func (s *Store) Flush() error {
	return nil
}

func (s *Store) GetObjectIdsForString(
	v string,
) ([]domain_interfaces.ExternalObjectId, error) {
	return nil, nil
}

func (s *Store) writeBlob(
	object *sku.Transacted,
	content []byte,
) (err error) {
	blobWriter, err := s.supplies.Env.GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = bytes.NewReader(content).WriteTo(blobWriter); err != nil {
		return errors.Wrap(err)
	}

	if err = object.SetBlobDigest(blobWriter.GetMarklId()); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// --- Haustoria interface ---

// Compile reads a CalDAV VTODO and returns dodder-compatible fields.
// external → dodder
func (s *Store) Compile(req haustoria.CompileRequest) (haustoria.CompileResult, error) {
	for _, cal := range s.calendars {
		result, err := s.client.ListTasks(cal.URL)
		if err != nil {
			continue
		}

		for _, twm := range result.Tasks {
			if twm.Task.UID != req.ExternalId {
				continue
			}

			return haustoria.CompileResult{
				ExternalId:  twm.Task.UID,
				Description: twm.Task.Summary,
				Blob:        []byte(twm.Task.Description),
				Tags:        twm.Task.Categories,
				TypeId:      cal.TypeId,
				ETag:        twm.Task.ETag,
			}, nil
		}
	}

	return haustoria.CompileResult{}, fmt.Errorf(
		"CalDAV task not found: %s", req.ExternalId,
	)
}

// Decompile writes a dodder object to CalDAV as a VTODO.
// dodder → external
func (s *Store) Decompile(req haustoria.DecompileRequest) (haustoria.DecompileResult, error) {
	cal := s.calendarForType(req.TypeId)
	if cal == nil {
		return haustoria.DecompileResult{}, fmt.Errorf(
			"no calendar mapping for type %s", req.TypeId,
		)
	}

	// Parse the !task TOML blob (produced by buildTaskTomlBlob during the
	// compile path or by the type's fields-writer script during organize
	// edits) to recover status / priority / due / notes. The blob is the
	// canonical source of truth — we don't read from Metadata.Index.Fields
	// here so the path works whether or not the !task type with reader
	// script is committed in the repo.
	values := parseTaskTomlBlob(req.Blob)

	task := caldav.Task{
		UID:         req.ExternalId,
		Summary:     req.Description,
		Description: values.Notes,
		Categories:  req.Tags,
		Status:      mapFieldValueToVTODOStatus(values.Status),
		Priority:    mapFieldValueToVTODOPriority(values.Priority),
		Due:         values.Due,
	}

	if task.UID == "" {
		task.UID = fmt.Sprintf("dodder-%d@dodder", time.Now().UnixNano())
	}

	ical := caldav.TaskToIcal(&task)
	href := cal.URL + task.UID + ".ics"

	err := s.client.PutTask(href, ical, req.ETag)
	if err != nil {
		return haustoria.DecompileResult{}, fmt.Errorf("decompile to CalDAV: %w", err)
	}

	meta, err := s.client.GetTask(href)
	if err != nil {
		return haustoria.DecompileResult{
			ExternalId: task.UID,
		}, nil
	}

	return haustoria.DecompileResult{
		ExternalId: task.UID,
		ETag:       meta.Task.ETag,
	}, nil
}

func (s *Store) Discover() ([]haustoria.ExternalResource, error) {
	var resources []haustoria.ExternalResource

	for _, cal := range s.calendars {
		result, err := s.client.ListTasks(cal.URL)
		if err != nil {
			return nil, fmt.Errorf("discover CalDAV tasks from %s: %w", cal.URL, err)
		}

		for _, twm := range result.Tasks {
			resources = append(resources, haustoria.ExternalResource{
				ExternalId:  twm.Task.UID,
				TypeId:      cal.TypeId,
				Description: twm.Task.Summary,
			})
		}
	}

	return resources, nil
}

func (s *Store) Delete(externalId string) error {
	for _, cal := range s.calendars {
		href := cal.URL + externalId + ".ics"
		if err := s.client.DeleteTask(href, ""); err == nil {
			return nil
		}
	}

	return fmt.Errorf("CalDAV task not found for delete: %s", externalId)
}

func (s *Store) calendarForType(typeId string) *CalendarMapping {
	for i := range s.calendars {
		if s.calendars[i].TypeId == typeId {
			return &s.calendars[i]
		}
	}

	// Fall back to first calendar.
	if len(s.calendars) > 0 {
		return &s.calendars[0]
	}

	return nil
}
