package remote_proto

import (
	"bytes"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/oscar/store"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	mad_blob_io "code.linenisgreat.com/madder/go/pkgs/blob_io"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
	"github.com/DataDog/zstd"
)

// sendClosure is the sender half of a transfer (the server for fetch, the
// client for push). It resolves the want's query against src, expands the
// transitive closure with the type system's edge explorer, then streams the
// closure to the peer: blobs first (so the receiver imports against blobs
// already on disk), then the object batch, then a completion ack.
// configDescriptor, when non-nil, is sent by a fetch server after
// have-negotiation as the RFC-0005 drtp-config-v1 control frame, and its
// named config blob is folded into the transfer's blob set even though no
// transferred object references it. It is nil on push (config is never
// pushed) and nil on a fetch whose config log is empty.
func sendClosure(
	env env_ui.Env,
	s *session,
	src repo.Repo,
	readBlobStore mad_domain_interfaces.BlobStore,
	envAdder interfaces.EnvVarsAdder,
	want control,
	compression string,
	configDescriptor *control,
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

	// #299: ship each object's full version history so the receiver's in-band
	// merge negotiator can find the common ancestor. The transfer is otherwise
	// effectively incremental (the query may resolve to latest-only), which
	// would leave the receiver unable to tell a fast-forward from a real
	// divergence. Symmetric across both directions (fetch and push); blobs are
	// still deduped by have-negotiation below, so this re-sends only historical
	// object metadata. Shared with the HTTP transport.
	if list, err = local_working_copy.ExpandListToObjectHistory(
		src,
		list,
	); err != nil {
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
		if manifest, err = gatherBlobDigests(
			readBlobStore,
			src,
			list,
			edges,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		// RFC 0005: fold the config-log head blob into the transfer's
		// blob set so it streams via the ordinary manifest/have/blob
		// mechanism, even though no transferred object references it. The
		// clone seeds its config log from it after the closure arrives.
		// Skip silently if the blob is not locally present (an incomplete
		// closure must not fail the object transfer).
		if configDescriptor != nil {
			manifest = appendConfigBlob(
				readBlobStore,
				manifest,
				configDescriptor.BlobId,
			)
		}
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

	// RFC 0005: name the config-log head after the blob carrying it has
	// streamed and before the terminal ack, so a clone client can capture
	// the descriptor and seed. Only sent on a fetch with a non-empty
	// config log (configDescriptor non-nil).
	if configDescriptor != nil {
		if err = s.writeControl(TypeConfig, *configDescriptor); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = s.writeControl(TypeAck, control{Status: StatusComplete}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// appendConfigBlob folds the config-log head blob digest into the blob
// manifest if it is present locally and not already advertised. A digest
// the sender does not hold is skipped (the closure stays self-consistent;
// the clone simply does not seed). A malformed digest is treated the same.
func appendConfigBlob(
	blobStore mad_domain_interfaces.BlobStore,
	manifest []string,
	blobId string,
) []string {
	if blobId == "" {
		return manifest
	}

	var digest markl.Id

	if err := digest.Set(blobId); err != nil {
		return manifest
	}

	if !blobStore.HasBlob(&digest) {
		return manifest
	}

	key := digest.String()

	for _, existing := range manifest {
		if existing == key {
			return manifest
		}
	}

	return append(manifest, key)
}

// gatherBlobDigests collects, deduplicated, every blob digest in the
// closure that the sender actually holds: each object's own blob, every
// blob reference discovered by expand-edges, and -- for each inventory-list
// object in the closure -- the contained objects' own blobs plus their blob
// references (#329). Expand-edges never traverses into a list's contained
// objects, so without that last expansion an inventory-list-genre query
// (what a DEFAULT clone/pull resolves to) advertises none of the contained
// objects' blobs and the receiver's import fails with an incomplete
// closure. This is the sender-side twin of the importer's reference copy
// (remote_transfer.ImportBlobIfNecessary). Only locally-present blobs are
// advertised, so the receiver never expects a blob the sender cannot send.
func gatherBlobDigests(
	blobStore mad_domain_interfaces.BlobStore,
	src repo.Repo,
	list *sku.HeapTransacted,
	edges sku.Edges,
) (digests []string, err error) {
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

	addObjectBlobs := func(object *sku.Transacted) {
		add(object.GetBlobDigest())

		metadata := object.GetMetadata()

		for refDigest := range metadata.AllBlobReferences() {
			refCopy := refDigest
			add(&refCopy)
		}
	}

	for object := range list.All() {
		addObjectBlobs(object)

		if object.GetGenre() != genres.InventoryList {
			continue
		}

		// The sender holds the whole store, so a list blob it cannot
		// decode is a hard error: silently skipping would reintroduce
		// the incomplete-closure failure this expansion exists to fix.
		containedSeq := src.GetInventoryListCoderCloset().
			StreamInventoryListBlobSkus(object)

		for contained, iterErr := range containedSeq {
			if iterErr != nil {
				err = errors.Wrapf(
					iterErr,
					"expanding inventory list %s",
					sku.String(object),
				)
				return nil, err
			}

			addObjectBlobs(contained)
		}
	}

	for i := range edges.Blobs {
		add(&edges.Blobs[i])
	}

	return digests, nil
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
// configDescriptorOut, when non-nil, receives the RFC-0005 config
// descriptor if the sender sends a drtp-config-v1 frame. A clone passes a
// non-nil pointer to capture the source's config-log head for seeding;
// every other receiver (pull, push) passes nil and the frame is discarded.
// bufferedObjectsOut, when non-nil, switches the receiver into staging mode
// (clone -script, dodder#396): the objects frame is decoded into the buffer
// for a transform to rewrite and re-sign instead of being imported inline.
// Blobs still stream into the local store, so the buffered objects' blob
// references resolve locally; a fresh clone has no history to merge, so the
// negotiator and inline import are skipped. Every non-scripted receiver
// passes nil for the ordinary import path.
func receiveClosure(
	env env_ui.Env,
	s *session,
	dst *local_working_copy.Repo,
	want control,
	storeOptions sku.StoreOptions,
	configDescriptorOut *control,
	bufferedObjectsOut *[]*sku.Transacted,
) (err error) {
	// Read have-checks span every read store (FDR-0015) so blobs already
	// held in an ancestor/XDG store are not re-requested; writes still land
	// in the default write store.
	writeBlobStore := dst.GetBlobStore()

	if err = negotiateHave(s, dst.GetEnvRepo().GetReadBlobStore()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// drtp streams the whole closure up front, blobs before objects, so a
	// blob still missing when an object imports means the closure was
	// incomplete — a hard error, not remote_http's reactive retry (there is
	// nothing to re-request; the single pass already delivered everything).
	// Wrap the standard delegate (keeping its progress output) and fail loud
	// on a missing/errored blob instead of logging it to stderr and
	// continuing, which would import an object with a dangling blob ref.
	logBlobCopy := sku.MakeBlobCopierDelegate(dst.GetEnv().GetUI(), false)

	importerOptions := repo.ImporterOptions{
		CheckedOutPrinter:   dst.PrinterCheckedOutConflictsForRemoteTransfers(),
		AllowMergeConflicts: want.AllowMergeConflicts,
		BlobCopierDelegate: func(result sku.BlobCopyResult) (err error) {
			if err = logBlobCopy(result); err != nil {
				err = errors.Wrap(err)
				return err
			}

			switch _, state := result.GetBytesWrittenAndState(); state {
			case blob_stores.CopyResultStateError,
				blob_stores.CopyResultStateNilRemoteBlobStore:
				err = errors.Errorf(
					"blob %s unavailable at import — drtp closure incomplete: %s",
					result.BlobId,
					result.GetError(),
				)
				return err
			}

			return err
		},
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

			case "!" + TypeConfig:
				// RFC 0005: capture the source's config-log head for a
				// clone to seed from. A receiver that does not want it
				// (pull, push) passes a nil out-param and discards the
				// frame. The blob itself arrives via the normal blob
				// stream (digest-verified on receipt).
				if configDescriptorOut != nil {
					*configDescriptorOut = msg
				}

			case "!" + TypeAck:
				if msg.Status == StatusComplete {
					return err
				}

			case "!" + TypeError:
				err = errors.Errorf("remote error: %s", msg.Message)
				return err

			default:
				// RFC 0004 additive/horizontal versioning: new control
				// types are additive and optional, so a receiver MUST skip
				// any control frame it does not recognize rather than abort.
				// This keeps an un-upgraded client compatible with an
				// upgraded server (e.g. RFC 0005's drtp-config-v1). Log a
				// diagnostic to stderr and continue the loop; the terminal
				// ack/error frames above still drive normal termination.
				env.GetUI().Printf(
					"skipping unrecognized control frame %q",
					typeString,
				)
			}

		case frameKindObjects:
			var payload []byte

			if payload, err = readFramePayload(s.reader, length); err != nil {
				err = errors.Wrap(err)
				return err
			}

			// Staging mode (clone -script, dodder#396): buffer the objects for a
			// transform to rewrite and re-sign instead of importing them inline.
			// The batch's blobs already streamed into the local store above, and
			// a fresh clone has no local history, so the merge negotiator and the
			// import are both skipped.
			if bufferedObjectsOut != nil {
				if err = decodeObjectsIntoBuffer(
					dst,
					payload,
					bufferedObjectsOut,
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				continue
			}

			// #299: the sender ships each object's full history in this batch
			// (sendClosure expands it), so build the in-band merge negotiator
			// from the batch before importing — the lock-step session has no
			// out-of-band way to query the sender's history. dst is the
			// receiving repo in both directions (client on fetch, server on
			// push), so this resolves the merge base symmetrically.
			negotiator := local_working_copy.MakeParentNegotiatorInBand(dst)

			if err = addObjectsToNegotiator(
				dst,
				payload,
				negotiator,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			importerOptions.ParentNegotiator = negotiator

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
	blobStore mad_domain_interfaces.BlobStore,
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

// addObjectsToNegotiator decodes the transferred object batch and records each
// version as the sender's history in the in-band merge negotiator. It decodes a
// fresh reader over the in-memory payload; importObjects decodes the same bytes
// again to import (the negotiator must be populated before that import runs).
func addObjectsToNegotiator(
	dst *local_working_copy.Repo,
	payload []byte,
	negotiator *local_working_copy.ParentNegotiatorInBand,
) (err error) {
	seq := dst.GetInventoryListCoderCloset().AllDecodedObjectsFromStream(
		bytes.NewReader(payload),
		nil,
	)

	for object, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return err
		}

		negotiator.AddRemoteObject(object)
	}

	return err
}

// decodeObjectsIntoBuffer decodes the transferred object batch into out,
// cloning each object so it outlives the decode iteration. It is the staging
// receive mode's substitute for importObjects (clone -script, dodder#396): the
// buffered objects are handed to the transform pipeline to rewrite and re-sign
// instead of being committed inline. The batch's blobs already streamed into
// the local store, so the buffered objects' blob references resolve locally.
func decodeObjectsIntoBuffer(
	dst *local_working_copy.Repo,
	payload []byte,
	out *[]*sku.Transacted,
) (err error) {
	seq := dst.GetInventoryListCoderCloset().AllDecodedObjectsFromStream(
		bytes.NewReader(payload),
		nil,
	)

	for object, iterErr := range seq {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return err
		}

		cloned, _ := object.CloneTransacted() //repool:owned
		*out = append(*out, cloned)
	}

	return err
}
