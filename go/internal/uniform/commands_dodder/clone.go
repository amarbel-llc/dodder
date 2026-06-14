package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_proto"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/madder/go/pkgs/blob_stores"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
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
}

var (
	_ interfaces.CommandComponentWriter = (*Clone)(nil)
	_ command.CommandWithArgs           = (*Clone)(nil)
)

func (cmd *Clone) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{{
			Name:        "repo-id",
			Description: "identifier for the new local repository",
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
}

func (cmd Clone) Run(req command.Request) {
	local := cmd.OnTheFirstDay(req, req.PopArg("new repo id"))

	var remote repo.Repo
	var remoteObject *sku.Transacted
	useProto := false

	if cmd.IsDirectTransfer() {
		remote = cmd.MakeDirectRemoteFromPath(req, local)
	} else if cmd.IsWebSocketProtocol() {
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

	var networkConfigDescriptor remote_proto.ConfigDescriptor

	if useProto {
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
	}
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
