package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/remote_connection_types"
	"code.linenisgreat.com/dodder/go/src/bravo/values"
	"code.linenisgreat.com/dodder/go/src/charlie/cli"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/src/golf/env_ui"
	"code.linenisgreat.com/dodder/go/src/golf/repo_blobs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/mike/typed_blob_store"
	"code.linenisgreat.com/dodder/go/src/tango/repo"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/whiskey/remote_http"
)

type Remote struct {
	RemoteRepoBlobs

	InventoryLists
	LocalWorkingCopy

	RemoteConnectionType remote_connection_types.Type
}

var _ interfaces.CommandComponentWriter = (*Remote)(nil)

func (cmd *Remote) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cli.FlagSetVarWithCompletion(
		flagSet,
		&cmd.RemoteConnectionType,
		"remote-connection-type",
	)
}

// returns a ready-to-use repo.Repo and an associated *sku.Transacted that can
// be persisted
func (cmd Remote) MakeRemoteAndObject(
	req command.Request,
	local *local_working_copy.Repo,
) (remote repo.Repo, remoteObject *sku.Transacted) {
	remoteEnvRepo := cmd.MakeEnvRepo(req, false)
	remoteTypedRepoBlobStore := typed_blob_store.MakeRepoStore(remoteEnvRepo)

	remoteObject, _ = sku.GetTransactedPool().GetWithRepool()

	command.PopRequestArgToFunc(
		req,
		"remote type",
		remoteObject.GetMetadataMutable().GetTypeMutable().SetType,
	)

	blob := cmd.CreateRemoteBlob(
		req,
		local,
		remoteObject.GetMetadata().GetType(),
	)

	remote = cmd.MakeRemoteFromBlobAndSetPublicKey(req, local, blob)

	var blobId domain_interfaces.MarklId

	{
		var err error

		if blobId, _, err = remoteTypedRepoBlobStore.WriteTypedBlob(
			remoteObject.GetMetadata().GetType(),
			blob,
		); err != nil {
			req.Cancel(err)
		}
	}

	remoteObject.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(blobId)

	return remote, remoteObject
}

// returns a ready-to-use repo.Repo FROM an associated *sku.Transacted
func (cmd Remote) MakeRemote(
	req command.Request,
	repo *local_working_copy.Repo,
	object *sku.Transacted,
) (remote repo.Repo) {
	envRepo := cmd.MakeEnvRepo(req, false)
	typedRepoBlobStore := typed_blob_store.MakeRepoStore(envRepo)

	var blob repo_blobs.Blob

	{
		var err error

		if blob, _, err = typedRepoBlobStore.ReadTypedBlob(
			object.GetMetadata().GetType(),
			object.GetBlobDigest(),
		); err != nil {
			req.Cancel(err)
		}
	}

	remote = cmd.MakeRemoteFromBlob(req, repo, blob)

	return remote
}

func (cmd Remote) MakeRemoteFromBlobAndSetPublicKey(
	req command.Request,
	repo *local_working_copy.Repo,
	blob repo_blobs.BlobMutable,
) (remote repo.Repo) {
	remote = cmd.MakeRemoteFromBlob(req, repo, blob)

	remoteConfig := remote.GetImmutableConfigPublic()
	blob.SetPublicKey(remoteConfig.GetPublicKey())

	return remote
}

// returns a ready-to-use repo.Repo FROM an associated repo_blobs.Blob
func (cmd Remote) MakeRemoteFromBlob(
	req command.Request,
	repo *local_working_copy.Repo,
	blob repo_blobs.Blob,
) (remote repo.Repo) {
	env := cmd.MakeEnv(req)

	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	// TODO use cmd.RemoteConnectionType to determine connection type
	switch blob := blob.(type) {
	case repo_blobs.BlobXDG:
		envDir := env_dir.MakeWithXDG(
			req,
			config.Debug,
			blob.MakeXDG(req.Utility.GetName()),
		)

		envUI := env_ui.Make(
			req,
			config,
			config.Debug,
			env.GetOptions(),
		)

		remote = local_working_copy.Make(
			env_local.Make(envUI, envDir),
			local_working_copy.OptionsEmpty,
		)

	case repo_blobs.BlobOverridePath:
		envDir := env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
			req,
			blob.GetOverridePath(),
			req.Utility.GetName(),
			config.Debug,
		)

		envUIOptions := env.GetOptions()
		envUIOptions.UIPrintingPrefix = "remote"

		envUI := env_ui.Make(
			req,
			config,
			config.Debug,
			env.GetOptions(),
		)

		remote = local_working_copy.Make(
			env_local.Make(envUI, envDir),
			local_working_copy.OptionsEmpty,
		)
		// remote = cmd.MakeRemoteStdioLocal(
		// 	req,
		// 	env,
		// 	blob.OverridePath,
		// 	repo,
		// 	blob.GetPublicKey(),
		// )

	// case repo.RemoteTypeStdioSSH:
	// 	remote = cmd.MakeRemoteStdioSSH(
	// 		req,
	// 		env,
	// 		remoteArg,
	// 		local,
	// 	)

	// case repo.RemoteTypeSocketUnix:
	// 	remote = cmd.MakeRemoteHTTPFromXDGDotenvPath(
	// 		req,
	// 		remoteArg,
	// 		env.GetOptions(),
	// 		local,
	// 	)

	case repo_blobs.BlobUri:
		remote = cmd.MakeRemoteUrl(
			req,
			env,
			blob.GetUri(),
			repo,
		)

	default:
		errors.ContextCancelWithErrorf(req, "unsupported repo blob type: %T", blob)
	}

	return remote
}

func (cmd *Remote) MakeRemoteHTTPFromXDGDotenvPath(
	req command.Request,
	xdgDotenvPath string,
	options env_ui.Options,
	repo *local_working_copy.Repo,
	pubkey markl.Id,
) (remoteHTTP repo.Repo) {
	envLocal := cmd.MakeEnvWithXDGLayoutAndOptions(
		req,
		xdgDotenvPath,
		options,
	)

	remote := cmd.MakeLocalWorkingCopyFromEnvLocal(envLocal)

	server := &remote_http.Server{
		EnvLocal: envLocal,
		Repo:     remote,
	}

	var httpRoundTripper remote_http.RoundTripperUnixSocket

	if err := httpRoundTripper.Initialize(
		server,
		pubkey,
	); err != nil {
		req.Cancel(err)
	}

	go func() {
		if err := server.Serve(httpRoundTripper.UnixSocket); err != nil {
			req.Cancel(err)
		}
	}()

	remoteHTTP = remote_http.MakeClient(
		envLocal,
		&httpRoundTripper,
		repo,
		cmd.MakeInventoryListCoderCloset(repo.GetEnvRepo()),
	)

	return remoteHTTP
}

func (cmd *Remote) MakeRemoteStdioSSH(
	req command.Request,
	env env_local.Env,
	arg string,
	repo *local_working_copy.Repo,
) (remoteHTTP repo.Repo) {
	envRepo := cmd.MakeEnvRepo(req, false)

	var httpRoundTripper remote_http.RoundTripperStdio

	if err := httpRoundTripper.InitializeWithSSH(
		envRepo,
		arg,
	); err != nil {
		env.Cancel(err)
	}

	remoteHTTP = remote_http.MakeClient(
		envRepo,
		&httpRoundTripper,
		repo,
		cmd.MakeInventoryListCoderCloset(envRepo),
	)

	return remoteHTTP
}

func (cmd *Remote) MakeRemoteStdioLocal(
	req command.Request,
	env env_local.Env,
	dir string,
	repo *local_working_copy.Repo,
	pubkey domain_interfaces.MarklId,
) (remoteHTTP repo.Repo) {
	envRepo := cmd.MakeEnvRepo(req, false)

	var httpRoundTripper remote_http.RoundTripperStdio

	if err := files.AssertDir(dir); err != nil {
		if files.IsErrNotDirectory(err) {
			errors.ContextCancelWithBadRequestError(req, err)
		} else {
			req.Cancel(err)
		}
	}

	httpRoundTripper.Cmd.Dir = dir

	if err := httpRoundTripper.InitializeWithLocal(
		envRepo,
		repo.GetConfig(),
		pubkey,
	); err != nil {
		env.Cancel(err)
	}

	remoteHTTP = remote_http.MakeClient(
		env,
		&httpRoundTripper,
		repo,
		cmd.MakeInventoryListCoderCloset(envRepo),
	)

	return remoteHTTP
}

func (cmd *Remote) MakeRemoteUrl(
	req command.Request,
	env env_local.Env,
	uri values.Uri,
	repo *local_working_copy.Repo,
) (remoteHTTP repo.Repo) {
	envRepo := cmd.MakeEnvRepo(req, false)

	remoteHTTP = remote_http.MakeClient(
		envRepo,
		&remote_http.RoundTripperHost{
			UrlData:      remote_http.MakeUrlDataFromUri(uri),
			RoundTripper: remote_http.DefaultRoundTripper,
		},
		repo,
		cmd.MakeInventoryListCoderCloset(envRepo),
	)

	return remoteHTTP
}
