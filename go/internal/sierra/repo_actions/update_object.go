package repo_actions

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

type UpdateObject struct {
	*repo
}

type ObjectChanges struct {
	Description *string
	Tags        []string // nil = no change, non-nil = replace all
	Type        *string
	Blob        *string
}

func (op UpdateObject) Run(
	objectId domain_interfaces.ObjectId,
	changes ObjectChanges,
) (result *sku.Transacted, err error) {
	if err = op.Lock(); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	result, err = op.runAlreadyLocked(objectId, changes)

	if unlockErr := op.Unlock(); unlockErr != nil {
		if err == nil {
			err = errors.Wrap(unlockErr)
		}

		return result, err
	}

	return result, err
}

func (op UpdateObject) runAlreadyLocked(
	objectId domain_interfaces.ObjectId,
	changes ObjectChanges,
) (result *sku.Transacted, err error) {
	result, err = op.GetStore().ReadOneObjectId(objectId)
	if err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	metadata := result.GetMetadataMutable()

	if changes.Description != nil {
		// Reset before Set: descriptions.Description.Set is cumulative — it
		// space-joins onto any existing value so the doddish parser can build
		// a description token by token — so without the reset an edit would
		// APPEND to the object's current description instead of replacing it.
		// Mirrors the ResetTags() in the tags branch, which makes tag edits
		// replace.
		description := metadata.GetDescriptionMutable()
		description.Reset()

		if err = description.Set(*changes.Description); err != nil {
			err = errors.Wrap(err)
			return result, err
		}
	}

	if changes.Tags != nil {
		metadata.ResetTags()

		for _, tagString := range changes.Tags {
			if err = metadata.AddTagString(tagString); err != nil {
				err = errors.Wrap(err)
				return result, err
			}
		}
	}

	if changes.Type != nil {
		if err = metadata.GetTypeMutable().SetType(*changes.Type); err != nil {
			err = errors.Wrap(err)
			return result, err
		}
	}

	if changes.Blob != nil {
		if err = writeBlobContent(op.repo, result, *changes.Blob); err != nil {
			err = errors.Wrap(err)
			return result, err
		}
	}

	// MergeCheckedOut mirrors what organize does
	// (repo_actions/organize_options.go): it routes the commit through the
	// store's ReadExternalAndMergeIfNecessary, which refreshes the
	// checked-out working-directory copy when it still matches HEAD
	// (objects.EqualerSansTai) and raises a conflict-merge when the user has
	// local edits. It is a no-op when the object is not checked out.
	// Without it, an MCP `edit` of a checked-out object left the
	// working-directory copy stale even though its working copy was clean.
	storeOptions := sku.GetStoreOptionsUpdate()
	storeOptions.MergeCheckedOut = true

	if err = op.GetStore().CreateOrUpdate(
		result,
		sku.CommitOptions{StoreOptions: storeOptions},
	); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	return result, err
}

// writeBlobContent writes content to the default blob store and stamps the
// resulting digest onto object. Shared by UpdateObject (edit) and
// WriteNewZettels (new -blob). It's a free function rather than a method
// because repo is a type alias for local_working_copy.Repo (a non-local type),
// so methods can't be defined on it here.
func writeBlobContent(
	r *repo,
	object *sku.Transacted,
	content string,
) (err error) {
	blobWriter, err := r.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = strings.NewReader(content).WriteTo(blobWriter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = object.SetBlobDigest(blobWriter.GetMarklId()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
