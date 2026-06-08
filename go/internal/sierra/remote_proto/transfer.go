package remote_proto

import (
	"bytes"
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/oscar/store"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"github.com/DataDog/zstd"
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
	readBlobStore mad_domain_interfaces.BlobStore,
	envAdder interfaces.EnvVarsAdder,
	want control,
	compression string,
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

	// readBlobStore is the multi-store read view (FDR-0015), not just the
	// default write store, so blobs living in an ancestor/XDG read store are
	// advertised and sendable — otherwise a repo backed by such stores
	// advertises an incomplete manifest and fails the blob reads. The caller
	// supplies it because repo.Repo (the interface) does not expose
	// GetEnvRepo; the concrete repos at the call sites do.

	// Have-negotiation: announce every blob in the closure the sender holds,
	// learn which the receiver already has, and stream only the rest. The
	// manifest/have exchange always happens (with an empty manifest when
	// blobs are excluded) so the receiver's read stays in lockstep.
	var manifest []string

	if !want.ExcludeBlobs {
		manifest = gatherBlobDigests(readBlobStore, list, edges)
	}

	if err = s.writeControl(TypeManifest, control{Blobs: manifest}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var have control

	if have, err = s.readControlExpecting(TypeHave); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = sendBlobs(
		env,
		s,
		readBlobStore,
		manifest,
		have.HaveBlobs,
		compression,
	); err != nil {
		err = errors.Wrap(err)
		return err
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

// gatherBlobDigests collects, deduplicated, every blob digest in the
// closure that the sender actually holds: each object's own blob plus every
// blob reference discovered by expand-edges. Only locally-present blobs are
// advertised, so the receiver never expects a blob the sender cannot send.
func gatherBlobDigests(
	blobStore mad_domain_interfaces.BlobStore,
	list *sku.HeapTransacted,
	edges sku.Edges,
) (digests []string) {
	seen := make(map[string]struct{})

	add := func(digest mad_domain_interfaces.MarklId) {
		if digest == nil || digest.IsNull() {
			return
		}

		key := digest.String()

		if _, ok := seen[key]; ok {
			return
		}

		seen[key] = struct{}{}

		if !blobStore.HasBlob(digest) {
			return
		}

		digests = append(digests, key)
	}

	for object := range list.All() {
		add(object.GetBlobDigest())
	}

	for i := range edges.Blobs {
		add(&edges.Blobs[i])
	}

	return digests
}

// sendBlobs streams each manifested blob the receiver does not already hold
// (per the have list). The receiver's content-addressed store still
// deduplicates, but skipping known blobs here avoids the bandwidth.
func sendBlobs(
	env env_ui.Env,
	s *session,
	blobStore mad_domain_interfaces.BlobStore,
	manifest []string,
	haveBlobs []string,
	compression string,
) (err error) {
	have := make(map[string]struct{}, len(haveBlobs))

	for _, key := range haveBlobs {
		have[key] = struct{}{}
	}

	for _, key := range manifest {
		errors.ContextContinueOrPanic(env)

		if _, ok := have[key]; ok {
			continue
		}

		if err = sendOneBlob(s, blobStore, key, compression); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func sendOneBlob(
	s *session,
	blobStore mad_domain_interfaces.BlobStore,
	key string,
	compression string,
) (err error) {
	var digest markl.Id

	if err = digest.Set(key); err != nil {
		err = errors.Wrapf(err, "blob digest %q", key)
		return err
	}

	var reader mad_domain_interfaces.BlobReader

	if reader, err = blobStore.MakeBlobReader(&digest); err != nil {
		// A blob that vanished between manifest and stream is skipped
		// rather than failing the whole transfer.
		if mad_blob_io.IsErrBlobMissing(err) {
			return nil
		}

		err = errors.Wrapf(err, "opening blob %s", key)
		return err
	}

	defer errors.DeferredCloser(&err, reader)

	// Stream the blob straight from the store to the wire; writeBlob chunks
	// (and optionally compresses) it so nothing is buffered whole.
	if err = s.writeBlob(key, reader, compression); err != nil {
		err = errors.Wrap(err)
		return err
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
// the server for push). It first answers the sender's blob manifest with the
// subset it already holds (have-negotiation), then reads frames until a
// completion ack: blob frames are written to the local store (and
// digest-verified), the objects frame is imported through the ordinary
// importer.
func receiveClosure(
	env env_ui.Env,
	s *session,
	dst *local_working_copy.Repo,
	want control,
	storeOptions sku.StoreOptions,
) (err error) {
	// Read have-checks span every read store (FDR-0015) so blobs already
	// held in an ancestor/XDG store are not re-requested; writes still land
	// in the default write store.
	writeBlobStore := dst.GetBlobStore()

	if err = negotiateHave(s, dst.GetEnvRepo().GetReadBlobStore()); err != nil {
		err = errors.Wrap(err)
		return err
	}

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
				if err = recvBlob(s, writeBlobStore, msg); err != nil {
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

// negotiateHave reads the sender's blob manifest and replies with the subset
// of those digests the local store already holds, so the sender can skip
// streaming them.
func negotiateHave(
	s *session,
	blobStore mad_domain_interfaces.BlobStore,
) (err error) {
	var manifest control

	if manifest, err = s.readControlExpecting(TypeManifest); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var have []string

	for _, key := range manifest.Blobs {
		var digest markl.Id

		if setErr := digest.Set(key); setErr != nil {
			// An unparseable digest is simply not claimed as held; the
			// sender will stream it and the receive-side verify catches a
			// genuine problem.
			continue
		}

		if blobStore.HasBlob(&digest) {
			have = append(have, key)
		}
	}

	if err = s.writeControl(TypeHave, control{HaveBlobs: have}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// recvBlob streams the blob announced by a blob_header into the local store —
// a sequence of blob frames terminated by a zero-length frame, optionally
// zstd-compressed — and verifies the computed content digest against the
// header. The digest is computed over the decompressed bytes, so compression
// is transparent to content addressing. Nothing is buffered whole.
func recvBlob(
	s *session,
	blobStore blob_stores.BlobStoreInitialized,
	header control,
) (err error) {
	var writer mad_domain_interfaces.BlobWriter

	if writer, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if streamErr := streamBlobInto(s, writer, header); streamErr != nil {
		// Close to release the writer's resources; the partial blob is not
		// committed because its digest will not match anything referenced.
		_ = writer.Close()
		err = errors.Wrap(streamErr)
		return err
	}

	if err = writer.Close(); err != nil {
		err = errors.Wrapf(err, "closing blob %s", header.BlobId)
		return err
	}

	if header.BlobId != "" {
		var expected markl.Id

		if err = expected.Set(header.BlobId); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = markl.AssertEqual(&expected, writer.GetMarklId()); err != nil {
			err = errors.Wrapf(err, "blob %s digest mismatch", header.BlobId)
			return err
		}
	}

	return err
}

// streamBlobInto copies the framed (and optionally zstd-compressed) blob body
// into writer, then drains any trailing frames through the terminator so the
// session stays aligned for the next frame.
func streamBlobInto(
	s *session,
	writer mad_domain_interfaces.BlobWriter,
	header control,
) (err error) {
	frameReader := &blobFrameReader{session: s}

	var source io.Reader = frameReader

	if header.Compression == CompressionZstd {
		decoder := zstd.NewReader(frameReader)
		defer errors.DeferredCloser(&err, decoder)
		source = decoder
	}

	if _, err = io.Copy(writer, source); err != nil {
		err = errors.Wrapf(err, "writing blob %s", header.BlobId)
		return err
	}

	// A zstd decoder can stop once its stream ends, before consuming the
	// terminator frame; drain the remaining frames so the next read starts
	// at the next message.
	if _, err = io.Copy(io.Discard, frameReader); err != nil {
		err = errors.Wrapf(err, "draining blob %s", header.BlobId)
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
