package command_components_dodder

import (
	"io"
	"os"
	"path/filepath"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/0/remote_connection_types"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_blobs"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/india/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_http"
	"code.linenisgreat.com/dodder/go/internal/sierra/remote_proto"
	"code.linenisgreat.com/dodder/go/lib/bravo/cli"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/values"
)

type Remote struct {
	RemoteRepoBlobs

	InventoryLists
	LocalWorkingCopy

	DirectPath           string
	parentIsHomeRepo     bool
	RemoteConnectionType remote_connection_types.Type
}

var _ interfaces.CommandComponentWriter = (*Remote)(nil)

func (cmd *Remote) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.StringVar(&cmd.DirectPath, "direct", "", "path to a local dodder repository for direct transfer without a stored remote")

	cli.FlagSetVarWithCompletion(
		flagSet,
		&cmd.RemoteConnectionType,
		"remote-connection-type",
	)
}

func (cmd Remote) IsDirectTransfer() bool {
	return cmd.DirectPath != ""
}

// IsWebSocketProtocol reports whether the remote should be reached over the
// drtp websocket transport (sierra/remote_proto, RFC 0004) rather than the
// legacy remote_http backend.
func (cmd Remote) IsWebSocketProtocol() bool {
	return cmd.RemoteConnectionType == remote_connection_types.TypeUrlWebsocket
}

// MakeProtoConnectionFromObject resolves a stored url remote object to a
// drtp websocket connection and a client bound to local. Used by pull/push
// when -remote-connection-type=url-websocket is set.
func (cmd Remote) MakeProtoConnectionFromObject(
	req command.Request,
	local *local_working_copy.Repo,
	object *sku.Transacted,
) (conn io.ReadWriteCloser, client *remote_proto.Client) {
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

	uriBlob, ok := blob.(repo_blobs.BlobUri)
	if !ok {
		errors.ContextCancelWithErrorf(
			req,
			"the websocket protocol requires a url remote, got %T",
			blob,
		)
	}

	uri := uriBlob.GetUri()
	url := uri.GetUrl()

	var err error

	if conn, err = remote_proto.DialWebSocket(
		req,
		remote_proto.WebSocketURL(url.String()),
	); err != nil {
		req.Cancel(err)
	}

	return conn, remote_proto.MakeClient(local)
}

func (cmd *Remote) ResolveImplicitDirectPath(
	local *local_working_copy.Repo,
) {
	if cmd.IsDirectTransfer() {
		return
	}

	parentPath := local.GetEnvWorkspace().GetParentPath()
	if parentPath != "" {
		cmd.DirectPath = parentPath
		return
	}

	// Check if this is a V1 workspace (repo-backed) with empty ParentPath,
	// meaning the parent is the home XDG repo.
	wsConfig := local.GetEnvWorkspace().GetWorkspaceConfig()
	if _, ok := wsConfig.(workspace_config_blobs.ConfigWithParentPath); ok {
		cmd.parentIsHomeRepo = true
	}
}

func (cmd Remote) IsHomeRepoParent() bool {
	return cmd.parentIsHomeRepo
}

func (cmd Remote) MakeHomeRepoRemote(
	req command.Request,
) repo.Repo {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	home, err := os.UserHomeDir()
	if err != nil {
		req.Cancel(err)
	}

	ownDir := env_dir.MakeWithHomeAndInitialize(
		req,
		dodder_env.XDGUtilityName,
		home,
		config.Debug,
	)

	madderDir := env_dir.MakeWithHomeAndInitialize(
		req,
		XDGUtilityNameMadder,
		home,
		config.Debug,
	)

	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{},
	)

	return local_working_copy.Make(
		env_local.Make(envUI, ownDir),
		env_local.Make(envUI, madderDir),
		local_working_copy.OptionsEmpty,
	)
}

func (cmd Remote) MakeDirectRemoteFromPath(
	req command.Request,
	local *local_working_copy.Repo,
) repo.Repo {
	absPath := cmd.DirectPath

	if !filepath.IsAbs(absPath) {
		var err error

		if absPath, err = filepath.Abs(absPath); err != nil {
			req.Cancel(err)
		}
	}

	blob := &repo_blobs.TomlLocalOverridePathV0{
		OverridePath: absPath,
	}

	return cmd.MakeRemoteFromBlob(req, local, blob)
}

// returns a ready-to-use repo.Repo and an associated *sku.Transacted that can
// be persisted
func (cmd Remote) MakeRemoteAndObject(
	req command.Request,
	local *local_working_copy.Repo,
) (remote repo.Repo, remoteObject *sku.Transacted) {
	remoteEnvRepo := cmd.MakeEnvRepo(req, false)
	remoteTypedRepoBlobStore := typed_blob_store.MakeRepoStore(remoteEnvRepo)

	remoteObject, _ = sku.GetTransactedPool().GetWithRepool() //repool:owned

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

	// A websocket remote is registered Trust-On-First-Use: serve-proto
	// speaks drtp over /drtp, not the remote_http REST surface, so there is
	// no /config-immutable to fetch the public key from at add time. The
	// server attests per-session during the transfer instead. remote-add
	// ignores the returned remote, so leaving it nil here is safe.
	if !cmd.IsWebSocketProtocol() {
		remote = cmd.MakeRemoteFromBlobAndSetPublicKey(req, local, blob)
	}

	var blobId mad_domain_interfaces.MarklId

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
		ownXDG := blob.MakeXDG(req.Utility.GetName())

		ownDir := env_dir.MakeWithXDG(
			req,
			config.Debug,
			ownXDG,
		)

		madderDir := env_dir.MakeWithXDG(
			req,
			config.Debug,
			ownXDG.CloneWithUtilityName(XDGUtilityNameMadder),
		)

		envUI := env_ui.Make(
			req,
			config,
			config.Debug,
			env.GetOptions(),
		)

		remote = local_working_copy.Make(
			env_local.Make(envUI, ownDir),
			env_local.Make(envUI, madderDir),
			local_working_copy.OptionsEmpty,
		)

	case repo_blobs.BlobOverridePath:
		ownDir := env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
			req,
			blob.GetOverridePath(),
			req.Utility.GetName(),
			config.Debug,
		)

		madderDir := env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
			req,
			blob.GetOverridePath(),
			XDGUtilityNameMadder,
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
			env_local.Make(envUI, ownDir),
			env_local.Make(envUI, madderDir),
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
	pubkey mad_domain_interfaces.MarklId,
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
	httpRoundTripper.HashFormat = repo.GetBlobStore().GetDefaultHashType()

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

	// Dial the URI's host:port directly and run requests through
	// RoundTripperBufioWrappedSigner so the nonce-injection /
	// trailer-verification path matches what stdio and unix-socket
	// transports already do. The previous shape used
	// RoundTripperHost + DefaultRoundTripper which bypassed sig-auth
	// entirely (#170).
	httpRoundTripper := &remote_http.RoundTripperBufioHost{}

	if err := httpRoundTripper.Initialize(
		uri,
		repo.GetBlobStore().GetDefaultHashType(),
	); err != nil {
		env.Cancel(err)
	}

	remoteHTTP = remote_http.MakeClient(
		envRepo,
		httpRoundTripper,
		repo,
		cmd.MakeInventoryListCoderCloset(envRepo),
	)

	return remoteHTTP
}
