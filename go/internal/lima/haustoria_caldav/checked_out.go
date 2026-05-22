package haustoria_caldav

import (
	"code.linenisgreat.com/dodder/go/internal/0/caldav"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/store_workspace"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// Compile-time interface checks for the optional store_workspace methods
// that the dodder commit / merge / organize paths probe via type
// assertion. The papa store calls these whenever
// `options.MergeCheckedOut` is set during a commit (e.g. from `dodder
// organize`), so a haustoria workspace MUST satisfy them or organize
// fails with "store does not support operation '**env_workspace.Store'".
var (
	_ store_workspace.ReadCheckedOutFromTransacted = &Store{}
	_ store_workspace.UpdateCheckoutFromCheckedOut = &Store{}
	_ store_workspace.Merge                        = &Store{}
)

// ReadCheckedOutFromTransacted returns the live external state of an
// internal dodder object by fetching the corresponding VTODO from CalDAV.
// Used by the merge path during commit to compare the internal version
// against the current remote state.
//
// Lookup strategy: the internal object's ExternalObjectId is the VTODO
// UID (set during the original CheckinHaustoria run). The calendar is
// resolved by matching the object's type against each CalendarMapping's
// TypeId. The .ics resource is fetched once via HTTP GET; the response
// is rebuilt into a TOML blob via buildTaskTomlBlob and projected into
// a fresh `*sku.CheckedOut`.
//
// TODO #103: cache CalDAV resources via ETag/LastModified to avoid an
// HTTP round-trip per organize commit. The current implementation hits
// the network for every call.
func (s *Store) ReadCheckedOutFromTransacted(
	object *sku.Transacted,
) (checkedOut *sku.CheckedOut, err error) {
	eoid := object.GetExternalObjectId()
	if eoid.IsEmpty() {
		return nil, errors.MakeErrNotFound(object.GetObjectId())
	}

	cal := s.calendarForType(object.GetMetadata().GetType().String())
	if cal == nil {
		return nil, errors.MakeErrNotFound(object.GetObjectId())
	}

	href := cal.URL + eoid.String() + ".ics"

	meta, err := s.client.GetTask(href)
	if err != nil {
		return nil, errors.Wrapf(err,
			"fetch CalDAV task %s for organize merge", eoid.String(),
		)
	}

	checkedOut, _ = sku.GetCheckedOutPool().GetWithRepool() //repool:owned

	// internal slot: the dodder-side object as it exists in the store
	sku.TransactedResetter.ResetWith(checkedOut.GetSku(), object)

	// external slot: the freshly-built remote state, sharing the same
	// object id but with a blob digest derived from the just-fetched
	// VTODO. The reader script will project status / priority / due into
	// the index when the daughter is committed.
	external := checkedOut.GetSkuExternal()
	sku.TransactedResetter.ResetWith(external, object)

	if err = external.GetMetadataMutable().GetDescriptionMutable().Set(
		meta.Task.Summary,
	); err != nil {
		return nil, errors.Wrap(err)
	}

	if cal.TypeId != "" {
		if err = external.GetMetadataMutable().GetTypeMutable().SetType(
			cal.TypeId,
		); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	for _, tagStr := range cal.Tags {
		if addErr := external.GetMetadataMutable().AddTagString(tagStr); addErr != nil {
			continue
		}
	}

	for _, tagStr := range meta.Task.Categories {
		if addErr := external.GetMetadataMutable().AddTagString(tagStr); addErr != nil {
			continue
		}
	}

	blob := buildTaskTomlBlob(&meta.Task)
	if err = s.writeBlob(external, blob); err != nil {
		return nil, errors.Wrapf(err,
			"rewrite blob for %s during merge fetch", eoid.String(),
		)
	}

	if err = external.GetExternalObjectIdMutable().SetWithGenre(
		meta.Task.UID,
		genres.Zettel,
	); err != nil {
		return nil, errors.Wrap(err)
	}

	checkedOut.SetState(checked_out_state.CheckedOut)
	return checkedOut, nil
}

// UpdateCheckoutFromCheckedOut pushes the resolved external state of a
// CheckedOut pair back to CalDAV via PUT. Called by the merging fast
// path after the daughter has been merged with the live remote state.
//
// The external slot's blob is the TOML projection of the merged field
// values; we parse it back to a Task struct and re-emit as iCal.
func (s *Store) UpdateCheckoutFromCheckedOut(
	options checkout_options.OptionsWithoutMode,
	object sku.SkuType,
) (err error) {
	external := object.GetSkuExternal()

	// The merging fast path in papa/store/merging.go:48 calls
	// ResetWithExceptFields(right=external, left=daughter), which
	// overwrites external.ExternalObjectId with the daughter's empty
	// value (organize doesn't carry the binding). Fall back to the
	// internal slot, which still has the original CalDAV UID.
	eoid := external.GetExternalObjectId()
	if eoid.IsEmpty() {
		eoid = object.GetSku().GetExternalObjectId()
	}
	if eoid.IsEmpty() {
		return errors.ErrorWithStackf(
			"cannot push external update without ExternalObjectId for %s",
			external.GetObjectId(),
		)
	}

	typeId := external.GetMetadata().GetType().String()

	cal := s.calendarForType(typeId)
	if cal == nil {
		return errors.ErrorWithStackf(
			"no calendar mapping for type %s when pushing organize update",
			typeId,
		)
	}

	// Pull the TOML blob via the metadata's blob digest. After the merging
	// fast path's ResetWithExceptFields, the external slot's blob digest
	// is the daughter's freshly-written digest from tryWriteFields. Read
	// directly through GetMetadata() to avoid an Id.IsNull panic on a
	// not-fully-typed digest copy.
	blobDigest := external.GetMetadata().GetBlobDigest()

	var blobBytes []byte
	if len(blobDigest.GetBytes()) > 0 {
		if blobBytes, err = s.readBlob(external.GetSku()); err != nil {
			return errors.Wrapf(err,
				"read blob for %s during organize push", eoid.String(),
			)
		}
	}

	values := parseTaskTomlBlob(blobBytes)

	var tags []string
	for tag := range external.GetMetadata().GetTags().All() {
		tags = append(tags, tag.String())
	}

	task := caldav.Task{
		UID:         eoid.String(),
		Summary:     external.GetMetadata().GetDescription().String(),
		Description: values.Notes,
		Categories:  tags,
		Status:      mapFieldValueToVTODOStatus(values.Status),
		Priority:    mapFieldValueToVTODOPriority(values.Priority),
		Due:         values.Due,
	}

	ical := caldav.TaskToIcal(&task)
	href := cal.URL + task.UID + ".ics"

	if err = s.client.PutTask(href, ical, ""); err != nil {
		return errors.Wrapf(err,
			"PUT VTODO %s to CalDAV during organize push", task.UID,
		)
	}

	return nil
}

// Merge resolves a 3-way merge conflict between the local edit, the
// fetched remote state, and the common ancestor. The current
// implementation does NOT attempt structural merging — it punts to
// last-write-wins by accepting the local edit (the daughter being
// committed) and letting the subsequent UpdateCheckoutFromCheckedOut
// push it to CalDAV. Conflict resolution belongs to the broader
// remote_transfer redesign tracked in #19.
//
// In practice the merging fast path in papa/store/merging.go avoids
// this method when the fetched remote equals the dodder-side mother,
// which is the common case for organize-driven edits between sync
// cycles.
func (s *Store) Merge(conflicted sku.Conflicted) (err error) {
	// Accept the local edit verbatim. The caller has already populated
	// conflicted.Local with the daughter being committed; we leave the
	// CheckedOut pair as-is so the post-merge UpdateCheckoutFromCheckedOut
	// pushes the local state to CalDAV (last-write-wins toward dodder).
	return nil
}
