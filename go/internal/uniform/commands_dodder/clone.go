package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_transfers"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_http"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_proto"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"clone",
		&Clone{
			Genesis: command_components_dodder.Genesis{
				BigBang: env_repo.BigBang{
					ExcludeDefaultType: true,
				},
			},
		},
	)
}

type Clone struct {
	command_components_dodder.Genesis
	command_components_dodder.RemoteTransfer
	command_components_dodder.Query
	Organize bool

	Script       string
	ScriptDigest string
}

var (
	_ interfaces.CommandComponentWriter = (*Clone)(nil)
	_ command.CommandWithArgs           = (*Clone)(nil)
)

func (cmd *Clone) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{{
			Name:        "repo-id",
			Description: "location handle for the new local repository (scope via spelling: name=user, .name=cwd, //name=system)",
			Required:    true,
		}}},
		cmd.Query.GetArgGroup(),
	}
}

func (cmd Clone) GetDescription() command.Description {
	return command.Description{
		Short: "clone a remote repository",
	}
}

func (cmd *Clone) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.Genesis.SetFlagDefinitions(flagDefinitions)
	cmd.RemoteTransfer.SetFlagDefinitions(flagDefinitions)
	cmd.Query.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.BoolVar(
		&cmd.Organize,
		"organize",
		false,
		"open organize to filter which objects get cloned before pulling",
	)

	flagDefinitions.StringVar(
		&cmd.Script,
		"script",
		"",
		"path to a Lua list-in/list-out transform applied to the pulled objects before commit; the clone is born already rewritten and re-signed under its own key (direct transfer only; mutually exclusive with -script-digest)",
	)

	flagDefinitions.StringVar(
		&cmd.ScriptDigest,
		"script-digest",
		"",
		"markl id of a stored blob containing the Lua transform script (direct transfer only; mutually exclusive with -script)",
	)
}

func (cmd Clone) Run(req command.Request) {
	cmd.SetLocationFromPositionalRequired(req, "new repo id")

	// clone -script buffers the pulled objects and runs the transform pipeline
	// in place of a direct-commit import. This works for any transport whose
	// remote implements the repo.Repo buffering surface (MakeInventoryList +
	// GetBlobStore) — direct (local-path) and legacy HTTP both do. The
	// drtp/websocket transport commits inline as it streams (client.Fetch →
	// receiveClosure) and exposes no such surface, so it is rejected; buffering
	// it is a followup (dodder#396). Validate before genesis so a rejected
	// invocation leaves no half-created repo behind.
	scriptSet := cmd.Script != "" || cmd.ScriptDigest != ""

	if scriptSet && cmd.IsWebSocketProtocol() {
		req.Cancel(errors.BadRequestf(
			"clone -script is not supported over the websocket protocol; use direct or HTTP transfer (proto support: dodder#396)",
		))
		return
	}

	local := cmd.OnTheFirstDay(req)

	var remote repo.Repo
	var remoteObject *sku.Transacted
	useProto := false

	if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, local)
	} else if cmd.IsWebSocketProtocol() {
		if cmd.Organize {
			req.Cancel(
				errors.BadRequestf(
					"-organize is not supported over the websocket protocol",
				),
			)
			return
		}

		// MakeRemoteAndObject builds the remote object (carrying the url)
		// without connecting for a websocket remote, so we get the object
		// to dial and skip the remote_http client entirely.
		_, remoteObject = cmd.MakeRemoteAndObject(req, local)
		useProto = true
	} else {
		// TODO offer option to persist remote object, if supported
		remote, _ = cmd.MakeRemoteAndObject(req, local)
	}

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		req.PopArgs(),
	)

	if cmd.Organize {
		queryGroup = cmd.RunOrganizeAgainstRemote(
			req,
			local,
			remote,
			queryGroup,
			"instructions: to exclude an object from the clone, delete it entirely",
		)
	}

	var networkConfigDescriptor remote_proto.ConfigDescriptor

	if scriptSet {
		// Direct or HTTP (proto rejected above): buffer the pulled objects and
		// run the transform pipeline instead of a direct-commit import.
		if err := cmd.runScriptedClone(req, local, remote, queryGroup); err != nil {
			req.Cancel(err)
		}
	} else if useProto {
		conn, client := cmd.MakeProtoConnectionFromObject(req, local, remoteObject)

		var fetchErr error

		if networkConfigDescriptor, fetchErr = client.Fetch(
			conn,
			queryGroup.String(),
			cmd.WithPrintCopies(true),
		); fetchErr != nil {
			req.Cancel(fetchErr)
		}
	} else if err := local.PullQueryGroupFromRemote(
		remote,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		req.Cancel(err)
	}

	// Config is repo-local (FDR 0020): pull never transfers it, so a fresh
	// clone would otherwise carry only its own genesis-default config. For a
	// direct (local-path) transfer the source is opened locally, so seed the
	// clone's config log from the source's current config here. Over the drtp
	// (websocket) protocol the source instead names its config-log head in a
	// drtp-config-v1 control frame and streams the config blob in the closure
	// (RFC 0005); seed from that descriptor.
	if cmd.IsDirectTransfer() {
		cmd.seedConfigFromDirectSource(req, local, remote)
	} else if useProto {
		cmd.seedConfigFromNetworkTransfer(req, local, networkConfigDescriptor)
	} else {
		// Legacy HTTP backend: the transfer does not carry config (config is
		// not an object), so the clone fetches the source's config descriptor
		// over GET /config and the named config blob over the blob route, then
		// seeds from it (RFC 0005 §HTTP Backend Transport).
		cmd.seedConfigFromHTTPSource(req, local, remote)
	}
}

// runScriptedClone is the clone -script consumer of the transform pipeline
// (dodder#392): buffer the source's queried objects, run one Lua transform over
// them, and commit the result re-signed under the clone's own key — a clone
// born already rewritten (tag cleanup, fork resolution, hash migration) rather
// than pulled verbatim and corrected after.
//
// It works against the repo.Repo INTERFACE, so it serves any transport whose
// remote can buffer: direct (local-path) and legacy HTTP both implement
// MakeInventoryList (the object set) and GetBlobStore (the read source for the
// blob pre-copy). The drtp/websocket transport commits inline as it streams and
// exposes neither, so the caller rejects it (dodder#396).
func (cmd Clone) runScriptedClone(
	req command.Request,
	local *local_working_copy.Repo,
	remote repo.Repo,
	queryGroup *queries.Query,
) error {
	scriptReader, err := makeTransformScriptReader(
		local,
		cmd.Script,
		cmd.ScriptDigest,
	)
	if err != nil {
		return err
	}

	defer errors.ContextMustClose(local, scriptReader)

	list, err := remote.MakeInventoryList(queryGroup)
	if err != nil {
		return errors.Wrap(err)
	}

	var objects []*sku.Transacted

	for object := range list.All() {
		cloned, _ := object.CloneTransacted() //repool:owned
		objects = append(objects, cloned)
	}

	local.GetUI().Printf("cloning %d object(s) through transform", len(objects))

	// Copy every source blob the buffered objects reference into the clone's own
	// store BEFORE the transform, so the script reads and the commit resolves
	// blobs natively from the clone and the clone is self-contained. A pre-copy
	// rather than init-from-lists' read overlay: the source and clone can share
	// an XDG namespace, and overlaying the source's stores would put the clone's
	// own write store in the read list, which MakeReadBlobStoreWithOverlay
	// rejects. remote.GetBlobStore() is the source's own read view (direct) or
	// the read-only HTTP blob wrapper (HTTP) — either resolves every referenced
	// blob by digest.
	blobImporter := blob_transfers.MakeBlobImporter(
		local.GetEnvRepo().GetEnvBlobStore(),
		remote.GetBlobStore(),
		blob_stores.MakeBlobStoreMap(local.GetEnvRepo().GetDefaultBlobStore()),
	)

	for _, object := range objects {
		if err := copyObjectReferencedBlobs(&blobImporter, object, false); err != nil {
			return err
		}
	}

	pipeline := transformPipeline{
		repo:         local,
		scriptReader: scriptReader,
		objects:      objects,
		// A clone carries the source's full history — many (id,tai) versions per
		// id by design — so do NOT reject duplicate object ids (dodder#392).
		disallowDuplicateObjectIds: false,
		// A transformed object cannot keep its source signature, and clone
		// already re-signs everything under the clone's own key — same commit
		// path as init-from-lists.
		commit: makeReSigningCommit(local),
	}

	return pipeline.run()
}

// seedConfigFromHTTPSource seeds the clone's config log from an HTTP-backend
// source (RFC 0005 §HTTP Backend Transport). Unlike drtp, the HTTP transfer
// never carries the config blob (config is not an object), so this first
// fetches the source's config descriptor (GET /config; a 404 or unrouted
// /config means "no config offered" and is a silent no-op), then copies the
// named config blob into the clone's default blob store via the existing blob
// route. It then delegates to the shared seedConfigFromNetworkTransfer, which
// confirms the blob landed, verifies its digest, and appends the clone-signed
// entry. Failure to fetch the descriptor or blob is surfaced as a diagnostic
// and never fails the object clone (RFC 0005 §Errors).
func (cmd Clone) seedConfigFromHTTPSource(
	req command.Request,
	local *local_working_copy.Repo,
	remote repo.Repo,
) {
	descriptor, offered, err := remote_http.FetchConfigDescriptor(remote)
	if err != nil {
		local.GetUI().Printf("skipping config seed: %s", err)
		return
	}

	// No config offered (server predates RFC 0005, routes no /config, or has
	// an empty config log). Keep the genesis default silently.
	if !offered {
		return
	}

	var sourceDigest markl.Id

	if err := sourceDigest.Set(descriptor.BlobId); err != nil {
		local.GetUI().Printf(
			"skipping config seed: unparseable config blob-id %q: %s",
			descriptor.BlobId,
			err,
		)
		return
	}

	// The HTTP transfer does not fold the config blob into the closure, so
	// copy it from the source's blob store (GET /blobs/{id}) into the clone's
	// default blob store. The store is content-addressed, so the written blob
	// keeps sourceDigest; CopyBlobIfNecessary verifies the digest on copy.
	localBlobStore := local.GetEnvRepo().GetDefaultBlobStore()

	copyResult := blob_stores.CopyBlobIfNecessary(
		local.GetEnv(),
		localBlobStore,
		remote.GetBlobStore(),
		&sourceDigest,
		nil,
		localBlobStore.GetDefaultHashType(),
	)

	if err := copyResult.GetError(); err != nil {
		local.GetUI().Printf(
			"skipping config seed: copying source config blob %s: %s",
			&sourceDigest,
			err,
		)
		return
	}

	// Delegate to the shared seeder: the config blob is now present locally,
	// so it confirms presence, skips when equal to the clone's head, and
	// appends the clone-signed entry.
	cmd.seedConfigFromNetworkTransfer(
		req,
		local,
		remote_proto.ConfigDescriptor{
			BlobId:     descriptor.BlobId,
			ConfigType: descriptor.ConfigType,
			Tai:        descriptor.Tai,
		},
	)
}

// seedConfigFromNetworkTransfer appends the source repo's current config as
// a new entry on the clone's config log, signed by the clone's own key,
// from the RFC-0005 descriptor a drtp fetch server offered. The config TOML
// blob itself already arrived over the normal blob stream (digest-verified
// on receipt), so this only confirms it landed locally and records the
// entry. Failure to seed is surfaced as a diagnostic and never fails the
// object clone — a clone with default config is a valid repository (RFC
// 0005 §Errors). An absent descriptor (old server, empty config log) is a
// no-op.
func (cmd Clone) seedConfigFromNetworkTransfer(
	req command.Request,
	local *local_working_copy.Repo,
	descriptor remote_proto.ConfigDescriptor,
) {
	// Absent descriptor: the server offered no config (predates RFC 0005 or
	// has an empty config log). Keep the genesis default silently.
	if descriptor.BlobId == "" {
		return
	}

	var sourceDigest markl.Id

	if err := sourceDigest.Set(descriptor.BlobId); err != nil {
		local.GetUI().Printf(
			"skipping config seed: unparseable config blob-id %q: %s",
			descriptor.BlobId,
			err,
		)
		return
	}

	var configType ids.Type

	if parsedType, err := ids.MakeType(descriptor.ConfigType); err != nil {
		local.GetUI().Printf(
			"skipping config seed: unparseable config-type %q: %s",
			descriptor.ConfigType,
			err,
		)
		return
	} else {
		configType = parsedType.ToType()
	}

	// The config blob arrives in the transferred closure's blob set. If it
	// is not present locally the closure was incomplete; seeding cannot
	// reference a blob the clone does not hold.
	if !local.GetEnvRepo().GetReadBlobStore().HasBlob(&sourceDigest) {
		local.GetUI().Printf(
			"skipping config seed: config blob %s not present after transfer",
			&sourceDigest,
		)
		return
	}

	// Skip when the source config matches the clone's current head: the
	// genesis root already records the identical config blob.
	var cloneDigest markl.Id
	cloneDigest.ResetWithMarklId(
		local.GetConfigStore().GetConfig().GetSku().GetBlobDigest(),
	)

	if markl.Equals(&cloneDigest, &sourceDigest) {
		return
	}

	log := config_log.Make(
		local.GetEnvRepo(),
		command_components_dodder.InventoryLists{}.MakeInventoryListCoderCloset(
			local.GetEnvRepo(),
		),
	)

	lockSmith := local.GetEnvRepo().GetLockSmith()

	if err := lockSmith.Lock(); err != nil {
		local.GetUI().Printf("skipping config seed: %s", err)
		return
	}

	defer local.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	if err := log.Append(
		&sourceDigest,
		configType,
		local.GetStore().GetTai(),
	); err != nil {
		local.GetUI().Printf("skipping config seed: %s", err)
		return
	}
}

// seedConfigFromDirectSource appends the direct-transfer source repo's
// current config as a new entry on the clone's config log, signed by the
// clone's own key. The source's config TOML blob is copied into the clone's
// default blob store first (content-addressed, so the digest is preserved).
// A no-op edit (source digest equals the clone's current head) is skipped.
func (cmd Clone) seedConfigFromDirectSource(
	req command.Request,
	local *local_working_copy.Repo,
	remote repo.Repo,
) {
	// The direct branch builds remote via MakeRemoteFromBlob, which for a
	// BlobOverridePath returns a *local_working_copy.Repo with its config
	// already bootstrapped from the source's own config log.
	source, ok := remote.(*local_working_copy.Repo)
	if !ok {
		req.Cancel(
			errors.ErrorWithStackf(
				"direct transfer expected a local source repo, got %T",
				remote,
			),
		)
		return
	}

	seedConfigLogFromLocalSource(req, local, source)
}

// seedConfigLogFromLocalSource appends a locally-opened source repo's
// current config as a new entry on the destination's config log, signed
// by the destination's own key (shared by clone's direct transfer and
// init-from's copy-migration). The source's config TOML blob is copied
// into the destination's default blob store first (content-addressed, so
// the digest is preserved). A no-op edit (source digest equals the
// destination's current head) is skipped.
func seedConfigLogFromLocalSource(
	req command.Request,
	local *local_working_copy.Repo,
	source *local_working_copy.Repo,
) {
	sourceSku := source.GetConfigStore().GetConfig().GetSku()

	var sourceDigest markl.Id
	sourceDigest.ResetWithMarklId(sourceSku.GetBlobDigest())

	sourceConfigType := sourceSku.GetType().ToType()

	// Skip when the source config matches the clone's current head: the
	// genesis root entry already records the identical config blob, so a
	// second identical entry would only clutter the log.
	var cloneDigest markl.Id
	cloneDigest.ResetWithMarklId(
		local.GetConfigStore().GetConfig().GetSku().GetBlobDigest(),
	)

	if markl.Equals(&cloneDigest, &sourceDigest) {
		return
	}

	// Copy the source's config blob into the clone's default blob store. The
	// store is content-addressed, so the written blob keeps sourceDigest and
	// the config log entry below resolves against it.
	localBlobStore := local.GetEnvRepo().GetDefaultBlobStore()

	copyResult := blob_stores.CopyBlobIfNecessary(
		local.GetEnv(),
		localBlobStore,
		source.GetBlobStore(),
		&sourceDigest,
		nil,
		localBlobStore.GetDefaultHashType(),
	)

	if err := copyResult.GetError(); err != nil {
		req.Cancel(errors.Wrapf(err, "copying source config blob %s", &sourceDigest))
		return
	}

	log := config_log.Make(
		local.GetEnvRepo(),
		command_components_dodder.InventoryLists{}.MakeInventoryListCoderCloset(
			local.GetEnvRepo(),
		),
	)

	lockSmith := local.GetEnvRepo().GetLockSmith()

	if err := lockSmith.Lock(); err != nil {
		req.Cancel(err)
		return
	}

	defer local.Must(errors.MakeFuncContextFromFuncErr(lockSmith.Unlock))

	if err := log.Append(
		&sourceDigest,
		sourceConfigType,
		local.GetStore().GetTai(),
	); err != nil {
		req.Cancel(err)
		return
	}
}
