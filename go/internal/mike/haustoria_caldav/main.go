package haustoria_caldav

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/caldav"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// Store implements both haustoria.Haustoria and store_workspace.StoreLike
// for CalDAV servers.
type Store struct {
	client       *caldav.Client
	calendarHref string
	supplies     store_workspace.Supplies
}

var (
	_ haustoria.Haustoria       = &Store{}
	_ store_workspace.StoreLike = &Store{}
)

func MakeStore(cfg *caldav.Config, calendarHref string) *Store {
	return &Store{
		client:       caldav.NewClient(cfg),
		calendarHref: calendarHref,
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
	resources, err := s.Discover()
	if err != nil {
		return errors.Wrap(err)
	}

	for _, resource := range resources {
		result, compileErr := s.Compile(haustoria.CompileRequest{
			ExternalId: resource.ExternalId,
		})
		if compileErr != nil {
			return errors.Wrapf(compileErr,
				"compile %s", resource.ExternalId,
			)
		}

		co, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned

		external := co.GetSkuExternal()
		metadata := external.GetMetadataMutable()

		if err = metadata.GetDescriptionMutable().Set(
			result.Description,
		); err != nil {
			return errors.Wrap(err)
		}

		if result.TypeId != "" {
			if err = metadata.GetTypeMutable().SetType(
				result.TypeId,
			); err != nil {
				return errors.Wrap(err)
			}
		}

		for _, tagStr := range result.Tags {
			if err = metadata.AddTagString(tagStr); err != nil {
				return errors.Wrap(err)
			}
		}

		if len(result.Blob) > 0 {
			if err = s.writeBlob(external, result.Blob); err != nil {
				return errors.Wrapf(err, "write blob for %s", resource.ExternalId)
			}
		}

		if err = external.GetExternalObjectIdMutable().SetWithGenre(
			resource.ExternalId,
			genres.Zettel,
		); err != nil {
			return errors.Wrap(err)
		}

		// Set genre on internal too — query filter checks internal genre.
		co.GetSku().GetObjectIdMutable().SetGenre(genres.Zettel)

		co.SetState(checked_out_state.Untracked)

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

	result, err := s.Decompile(haustoria.DecompileRequest{
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
	result, err := s.client.ListTasks(s.calendarHref)
	if err != nil {
		return haustoria.CompileResult{}, fmt.Errorf("compile from CalDAV: %w", err)
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
			TypeId:      "!task",
			ETag:        twm.Task.ETag,
		}, nil
	}

	return haustoria.CompileResult{}, fmt.Errorf(
		"CalDAV task not found: %s", req.ExternalId,
	)
}

// Decompile writes a dodder object to CalDAV as a VTODO.
// dodder → external
func (s *Store) Decompile(req haustoria.DecompileRequest) (haustoria.DecompileResult, error) {
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
	result, err := s.client.ListTasks(s.calendarHref)
	if err != nil {
		return nil, fmt.Errorf("discover CalDAV tasks: %w", err)
	}

	resources := make([]haustoria.ExternalResource, 0, len(result.Tasks))
	for _, twm := range result.Tasks {
		resources = append(resources, haustoria.ExternalResource{
			ExternalId:  twm.Task.UID,
			TypeId:      "!task",
			Description: twm.Task.Summary,
		})
	}

	return resources, nil
}

func (s *Store) Delete(externalId string) error {
	href := s.calendarHref + externalId + ".ics"
	return s.client.DeleteTask(href, "")
}
