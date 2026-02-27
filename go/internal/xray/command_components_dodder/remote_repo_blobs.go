package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/repo_blobs"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
)

type RemoteRepoBlobs struct {
	EnvRepo
}

var _ interfaces.CommandComponentWriter = (*RemoteRepoBlobs)(nil)

func (cmd *RemoteRepoBlobs) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
}

// Returns a repo_blobs.BlobMutable that can be used to create a
// repo.Repo. The blob's public key SHOULD be set before writing it to the
// store.
func (cmd RemoteRepoBlobs) CreateRemoteBlob(
	req command.Request,
	local *local_working_copy.Repo,
	remoteType ids.Type,
) (blob repo_blobs.BlobMutable) {
	remoteEnvRepo := cmd.MakeEnvRepo(req, false)

	switch remoteType.ToType() {
	default:
		errors.ContextCancelWithBadRequestf(
			req,
			"unsupported remote type: %q",
			remoteType,
		)

	case ids.GetOrPanic(ids.TypeTomlRepoLocalOverridePath).TypeStruct:
		xdgOverridePath := req.PopArg("xdg-path-override")

		blob = &repo_blobs.TomlLocalOverridePathV0{
			OverridePath: xdgOverridePath,
		}

	case ids.GetOrPanic(ids.TypeTomlRepoUri).TypeStruct:
		url := req.PopArg("url")

		var typedBlob repo_blobs.TomlUriV0

		if err := typedBlob.Uri.Set(url); err != nil {
			errors.ContextCancelWithBadRequestf(req, "invalid url: %s", err)
		}

		blob = &typedBlob

	case ids.GetOrPanic(ids.TypeTomlRepoLocalOverridePath).TypeStruct:
		path := req.PopArg("path")

		blob = &repo_blobs.TomlLocalOverridePathV0{
			OverridePath: remoteEnvRepo.AbsFromCwdOrSame(path),
		}
	}

	return blob
}
