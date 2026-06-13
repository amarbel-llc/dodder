package config_log

// This package is an append-only, repo-local log of signed config
// states stored in env_repo.FileConfigLog(). Each entry is a single
// signed sku.Transacted (object id konfig, with the config blob's own
// type, e.g. !toml-config-v2) whose blob digest points at the config
// TOML blob in the default blob store. The stream itself is framed
// !inventory_list-v2 for coder selection, but each entry keeps its own
// config type, exactly as konfig appears as a contained object inside a
// real inventory list. Entries chain by mother signature; append order
// is the history and the last entry is the head.
//
// The blobStore field below thinly duplicates
// inventory_list_store/blob_store_v1.go (which is unexported and so
// cannot be reused directly). Consolidating the two is the
// explicitly-flagged follow-up in FDR 0020.

import (
	"io"
	"os"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/object_finalizer"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

var ErrEmpty = newPkgError("empty config log")

// Log is an append-only repo-local log of signed config states. The
// caller must hold the repo lock before calling Append; the command
// layer (Task 5) is responsible for that.
type Log struct {
	envRepo   env_repo.Env
	pathLog   string
	blobType  ids.TypeStruct
	closet    inventory_list_coders.Closet
	finalizer object_finalizer.Finalizer
}

// Make builds a config log over envRepo.FileConfigLog(), reusing the
// repo's inventory-list coder closet. blobType resolves to the repo's
// configured inventory list type (!inventory_list-v2), which is
// available from genesis config pre-bootstrap.
//
// The inventory-list blob store is fetched lazily (inside Append, the
// only method that needs it) rather than here: store_config bootstrap
// calls Make + Head before any blob store is guaranteed to be
// initialized (e.g. PermitNoDodderDirectory repos and the
// legacy-config migration command run under MakeBlobStoreEnvWithoutStores,
// where GetDefaultBlobStore panics). Head and All read only the log
// file and never touch the blob store, so deferring the lookup keeps
// bootstrap-time Make + Head safe.
func Make(envRepo env_repo.Env, closet inventory_list_coders.Closet) Log {
	blobType := ids.MustTypeStruct(
		envRepo.GetConfigPublic().Blob.GetInventoryListTypeId(),
	)

	return Log{
		envRepo:   envRepo,
		pathLog:   envRepo.FileConfigLog(),
		blobType:  blobType,
		closet:    closet,
		finalizer: object_finalizer.Make(),
	}
}

// Append writes a new signed config entry pointing at blobDigest, with
// the given configType and tai, chained (mother = current head's object
// sig) onto the existing head. blobDigest is the digest of the config
// TOML blob, which must already live in the default blob store.
// configType is the config blob's own type (e.g. !toml-config-v2); the
// entry keeps it rather than the stream framing type, so store_config
// bootstrap can decode the blob via repo_configs.Coder. The caller must
// hold the repo lock.
func (log Log) Append(
	blobDigest mad_domain_interfaces.MarklId,
	configType ids.Type,
	tai ids.Tai,
) (object *sku.Transacted, err error) {
	object, _ = sku.GetTransactedPool().GetWithRepool() //repool:owned

	if err = object.GetObjectIdMutable().SetWithId(ids.Config); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	if err = object.SetBlobDigest(blobDigest); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	// Stamp the entry with the config blob's own type. The stream is
	// framed !inventory_list-v2 for coder selection, but each entry keeps
	// its own config type, mirroring how konfig appears as a contained
	// object (type !toml-config-v2) inside a real inventory list.
	object.GetMetadataMutable().GetTypeMutable().ResetWithType(
		configType.ToType(),
	)

	object.SetTai(tai)

	{
		var head *sku.Transacted
		var repoolHead interfaces.FuncRepool

		if head, repoolHead, err = log.Head(); err != nil {
			if errors.Is(err, ErrEmpty) {
				// root entry; leave mother null
				err = nil
			} else {
				err = errors.Wrap(err)
				return object, err
			}
		} else {
			defer repoolHead()

			if err = object.SetMother(head); err != nil {
				err = errors.Wrap(err)
				return object, err
			}
		}
	}

	if err = log.writeObject(object); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}

// writeObject mirrors
// inventory_list_store.blobStoreV1.WriteInventoryListObject (minus the
// object-type overwrite, which is wrong for a config entry — see
// below): it opens the log file for append, builds a MultiWriter over
// the inventory-list blob store writer and the log file, signs the
// object, and encodes it via the coder closet.
func (log Log) writeObject(object *sku.Transacted) (err error) {
	var blobStoreWriteCloser mad_domain_interfaces.BlobWriter

	// Fetch the inventory-list blob store lazily, only here on the
	// append path (see Make's doc comment): bootstrap-time callers that
	// only Head/All must never trigger blob store discovery.
	if blobStoreWriteCloser, err = log.envRepo.GetInventoryListBlobStore().MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobStoreWriteCloser)

	// Do NOT overwrite the object's type with log.blobType here. The
	// object already carries its own config type (set in Append);
	// log.blobType (!inventory_list-v2) is used only for stream framing
	// (the type header doc and the coder selection in WriteObjectToWriter
	// below), exactly as konfig keeps its own type as a contained object
	// inside a real inventory list.
	var file *os.File

	// Unlike inventory_list_store (whose log file is pre-created at
	// genesis), the config log file does not exist until the first
	// Append, so create-on-append here. Mirrors zettel_id_log.AppendEntry.
	if file, err = files.OpenFile(
		log.pathLog,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0o666,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)
	defer errors.Deferred(&err, file.Sync)

	// The stream-level type header doc (written verbatim to the log
	// file, not the blob store) establishes the type for every
	// subsequent type-less object blob, exactly as
	// env_repo.writeInventoryListLog does for the inventory list log.
	// Write it once, when the file is newly created.
	if err = log.writeTypeHeaderIfEmpty(file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
		io.MultiWriter(blobStoreWriteCloser, file),
	)
	defer repoolBufferedWriter()

	if err = log.finalizer.FinalizeAndSignOverwrite(
		object,
		log.envRepo.GetConfigPrivate().Blob,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = log.closet.WriteObjectToWriter(
		log.blobType,
		object,
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// writeTypeHeaderIfEmpty writes the stream-level type header doc when
// file is empty (i.e. freshly created on the first append). Mirrors
// env_repo.writeInventoryListLog.
func (log Log) writeTypeHeaderIfEmpty(file *os.File) (err error) {
	var stat os.FileInfo

	if stat, err = file.Stat(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if stat.Size() > 0 {
		return err
	}

	coder := hyphence.Coder[*hyphence.TypedBlobEmpty]{
		Metadata: hyphence.TypedMetadataCoder[struct{}]{},
	}

	header := hyphence.TypedBlobEmpty{
		Type: log.blobType.ToMadder(),
	}

	if _, err = coder.EncodeTo(&header, file); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// Head returns the last (newest) entry in append order, or ErrEmpty
// when the log file does not exist yet. On success it returns a
// pool-owned head together with the repool function the caller must
// call exactly once when done with head; on any error path repoolHead
// is nil and no pooled object is outstanding.
func (log Log) Head() (
	head *sku.Transacted,
	repoolHead interfaces.FuncRepool,
	err error,
) {
	var file *os.File

	if file, err = files.OpenReadOnly(log.pathLog); err != nil {
		if errors.IsNotExist(err) {
			err = errors.Wrap(ErrEmpty)
			return head, repoolHead, err
		}

		err = errors.Wrap(err)
		return head, repoolHead, err
	}

	defer errors.ContextMustClose(log.envRepo, file)

	for object, iterErr := range log.closet.AllDecodedObjectsFromStream(
		file,
		nil,
	) {
		if iterErr != nil {
			if repoolHead != nil {
				repoolHead()
				repoolHead = nil
				head = nil
			}

			err = errors.Wrap(iterErr)
			return head, repoolHead, err
		}

		if head == nil {
			head, repoolHead = sku.GetTransactedPool().GetWithRepool() //repool:suppress ownership transfer via return
		}

		sku.TransactedResetter.ResetWith(head, object)
	}

	if head == nil {
		err = errors.Wrap(ErrEmpty)
		return head, repoolHead, err
	}

	return head, repoolHead, err
}

// All yields entries oldest->newest (file/append order). When the log
// file does not exist, the sequence is empty (no error).
func (log Log) All() sku.Seq {
	return func(yield func(*sku.Transacted, error) bool) {
		var file *os.File

		{
			var err error

			if file, err = files.OpenReadOnly(log.pathLog); err != nil {
				if errors.IsNotExist(err) {
					return
				}

				yield(nil, errors.Wrap(err))
				return
			}
		}

		defer errors.ContextMustClose(log.envRepo, file)

		for object, err := range log.closet.AllDecodedObjectsFromStream(
			file,
			nil,
		) {
			if err != nil {
				if !yield(nil, errors.Wrap(err)) {
					return
				}

				continue
			}

			if !yield(object, nil) {
				return
			}
		}
	}
}
