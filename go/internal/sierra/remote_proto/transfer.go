package remote_proto

import (
	"bytes"
	"io"
	"time"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/oscar/store"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	mad_blob_io "github.com/amarbel-llc/madder/go/pkgs/blob_io"
	"github.com/amarbel-llc/madder/go/pkgs/blob_stores"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

// sendClosure is the sender half of a transfer (the server for fetch, the
// client for push). It resolves the want's query against src, expands the
// transitive closure with the type system's edge explorer, then streams the
// closure to the peer: blobs first (so the receiver imports against blobs
// already on disk), then the object batch, then a completion ack.
func sendClosure(
	env env_ui.Env,
	s *session,
	src repo.Repo,
	envAdder interfaces.EnvVarsAdder,
	want control,
) (err error) {
	var queryGroup *queries.Query

	if queryGroup, err = src.MakeExternalQueryGroup(
		nil,
		sku.ExternalQueryOptions{},
		want.Query,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var list *sku.HeapTransacted

	if list, err = src.MakeInventoryList(queryGroup); err != nil {
		err = errors.Wrap(err)
		return err
	}

	explorer := store.MakeEdgeExplorer(
		src.GetObjectStore(),
		src.GetBlobStore(),
		envAdder,
	)

	var edges sku.Edges

	if edges, err = expandEdges(list, src.GetObjectStore(), explorer); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if len(edges.Skipped) > 0 {
		err = errors.Errorf(
			"edge traversal had %d failures: %s",
			len(edges.Skipped),
			edges.Skipped[0],
		)
		return err
	}

	if !want.ExcludeBlobs {
		if err = sendBlobs(env, s, src.GetBlobStore(), list, edges); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = sendObjects(env, s, src, list); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writeControl(TypeAck, control{Status: StatusComplete}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// sendBlobs streams every blob in the closure: each object's own blob plus
// every blob reference discovered by expand-edges, deduplicated. v1 sends
// the whole closure rather than negotiating haves; the receiver's
// content-addressed store deduplicates blobs it already holds.
func sendBlobs(
	env env_ui.Env,
	s *session,
	blobStore blob_stores.BlobStoreInitialized,
	list *sku.HeapTransacted,
	edges sku.Edges,
) (err error) {
	seen := make(map[string]struct{})

	send := func(digest mad_domain_interfaces.MarklId) (err error) {
		if digest == nil || digest.IsNull() {
			return err
		}

		key := digest.String()

		if _, ok := seen[key]; ok {
			return err
		}

		seen[key] = struct{}{}

		var reader mad_domain_interfaces.BlobReader

		if reader, err = blobStore.MakeBlobReader(digest); err != nil {
			// A blob that is referenced but absent locally is skipped
			// rather than failing the whole transfer, matching
			// CopyBlobIfNecessary's tolerance on the local pull path.
			if mad_blob_io.IsErrBlobMissing(err) {
				return nil
			}

			err = errors.Wrapf(err, "opening blob %s", key)
			return err
		}

		defer errors.DeferredCloser(&err, reader)

		var data []byte

		if data, err = io.ReadAll(reader); err != nil {
			err = errors.Wrapf(err, "reading blob %s", key)
			return err
		}

		if err = s.writeBlob(key, data); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	for object := range list.All() {
		errors.ContextContinueOrPanic(env)

		if err = send(object.GetBlobDigest()); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	for i := range edges.Blobs {
		errors.ContextContinueOrPanic(env)

		if err = send(&edges.Blobs[i]); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

// sendObjects encodes the closure as one inventory_list hyphence stream and
// writes it as a single objects frame. It uses the *typed* writer so the
// stream carries its `! inventory_list-vN` metadata line, which the
// receiver's AllDecodedObjectsFromStream reads to select the decoder (the
// same typed-stream contract the HTTP backend's POST /inventory_lists uses).
func sendObjects(
	env env_ui.Env,
	s *session,
	src repo.Repo,
	list *sku.HeapTransacted,
) (err error) {
	listType := ids.GetOrPanic(
		src.GetImmutableConfigPublic().GetInventoryListTypeId(),
	).TypeStruct

	buffer := bytes.NewBuffer(nil)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(buffer)
	defer repoolBufferedWriter()

	if _, err = src.GetInventoryListCoderCloset().WriteTypedBlobToWriter(
		env,
		listType,
		quiter.MakeSeqErrorFromSeq(list.All()),
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writeObjects(buffer.Bytes()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// receiveClosure is the receiver half of a transfer (the client for fetch,
// the server for push). It reads frames until a completion ack: blob frames
// are written to the local store (and digest-verified), the objects frame is
// imported through the ordinary importer.
func receiveClosure(
	env env_ui.Env,
	s *session,
	dst *local_working_copy.Repo,
	want control,
	storeOptions sku.StoreOptions,
) (err error) {
	blobStore := dst.GetBlobStore()

	importerOptions := repo.ImporterOptions{
		CheckedOutPrinter:   dst.PrinterCheckedOutConflictsForRemoteTransfers(),
		AllowMergeConflicts: want.AllowMergeConflicts,
		BlobCopierDelegate: sku.MakeBlobCopierDelegate(
			dst.GetEnv().GetUI(),
			false,
		),
	}

	for {
		errors.ContextContinueOrPanic(env)

		kind, length, frameErr := readFrameHeader(s.reader)
		if frameErr != nil {
			if errors.IsEOF(frameErr) {
				err = errors.Errorf("stream closed before completion ack")
				return err
			}

			err = errors.Wrap(frameErr)
			return err
		}

		switch kind {
		case frameKindControl:
			var payload []byte

			if payload, err = readFramePayload(s.reader, length); err != nil {
				err = errors.Wrap(err)
				return err
			}

			var typeString string
			var msg control

			if typeString, msg, err = decodeControl(payload); err != nil {
				err = errors.Wrap(err)
				return err
			}

			switch typeString {
			case "!" + TypeBlobHeader:
				if err = recvBlob(env, s, blobStore, msg); err != nil {
					err = errors.Wrap(err)
					return err
				}

			case "!" + TypeAck:
				if msg.Status == StatusComplete {
					return err
				}

			case "!" + TypeError:
				err = errors.Errorf("remote error: %s", msg.Message)
				return err

			default:
				err = errors.Errorf("unexpected control frame %q", typeString)
				return err
			}

		case frameKindObjects:
			var payload []byte

			if payload, err = readFramePayload(s.reader, length); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if err = importObjects(dst, payload, importerOptions, storeOptions); err != nil {
				err = errors.Wrap(err)
				return err
			}

		default:
			err = errors.Errorf("unexpected frame kind %d", kind)
			return err
		}
	}
}

// recvBlob reads the blob frame announced by a blob_header and writes it to
// the local store, verifying the content digest against the header.
func recvBlob(
	env env_ui.Env,
	s *session,
	blobStore blob_stores.BlobStoreInitialized,
	header control,
) (err error) {
	kind, length, frameErr := readFrameHeader(s.reader)
	if frameErr != nil {
		err = errors.Wrapf(frameErr, "reading blob frame for %s", header.BlobId)
		return err
	}

	if kind != frameKindBlob {
		err = errors.Errorf(
			"expected blob frame after header for %s, got kind %d",
			header.BlobId,
			kind,
		)
		return err
	}

	var writer mad_domain_interfaces.BlobWriter

	if writer, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var expected mad_domain_interfaces.MarklId

	if header.BlobId != "" {
		var expectedId markl.Id

		if err = expectedId.Set(header.BlobId); err != nil {
			err = errors.Wrap(err)
			return err
		}

		expected = &expectedId
	}

	copyResult := blob_stores.CopyReaderToWriter(
		env,
		writer,
		io.LimitReader(s.reader, int64(length)),
		expected,
		nil,
		func(time.Time) {
			ui.Err().Printf("receiving blob %s...", header.BlobId)
		},
		3*time.Second,
	)

	if err = copyResult.GetError(); err != nil {
		err = errors.Wrapf(err, "writing blob %s", header.BlobId)
		return err
	}

	return err
}

// importObjects decodes the inventory_list payload and imports it through
// the ordinary importer, the same path the HTTP server uses for a received
// inventory list.
func importObjects(
	dst *local_working_copy.Repo,
	payload []byte,
	importerOptions repo.ImporterOptions,
	storeOptions sku.StoreOptions,
) (err error) {
	importer := dst.MakeImporter(importerOptions, storeOptions)

	seq := dst.GetInventoryListCoderCloset().AllDecodedObjectsFromStream(
		bytes.NewReader(payload),
		nil,
	)

	if err = dst.ImportSeq(seq, importer); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
