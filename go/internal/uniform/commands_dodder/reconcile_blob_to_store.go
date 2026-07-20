package commands_dodder

import (
	"io"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

func init() {
	// Stopgap command for the blob-store resolution divergence tracked as
	// dodder#359 (a store name like "default" can resolve to two different
	// physical stores depending on which code path resolves it). Copies
	// one blob, by digest, from wherever the unrestricted ancestor/XDG
	// read discovers it into an EXPLICITLY-addressed destination store (no
	// walk-up involved on the write side at all). Remove once the
	// underlying divergence is resolved properly — see
	// docs/rfcs/0007-anchored-identity-and-resolution.md.
	utility.AddCmd("reconcile-blob-to-store", &ReconcileBlobToStore{})
}

type ReconcileBlobToStore struct {
	command_components_dodder.EnvRepo
	command_components_dodder.BlobStore
}

var _ command.CommandWithArgs = (*ReconcileBlobToStore)(nil)

func (cmd ReconcileBlobToStore) GetDescription() command.Description {
	return command.Description{
		Short: "TEMPORARY workaround (dodder#359): copy one blob by digest into an explicitly-addressed destination store",
	}
}

func (cmd *ReconcileBlobToStore) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "digest",
				Description: "markl digest of the blob to copy",
				Required:    true,
			},
			{
				Name:        "dest-base-path",
				Description: "base path of the destination blob store",
				Required:    true,
			},
			{
				Name:        "dest-config-path",
				Description: "path to the destination blob store's config file",
				Required:    true,
			},
		},
	}}
}

func (cmd ReconcileBlobToStore) Run(req command.Request) {
	digestArg := req.PopArg("digest")
	destBasePath := req.PopArg("dest-base-path")
	destConfigPath := req.PopArg("dest-config-path")
	req.AssertNoMoreArgs()

	env := cmd.MakeEnvRepo(req, false)

	var digest markl.Id

	if err := digest.Set(digestArg); err != nil {
		env.Cancel(errors.Wrapf(err, "invalid digest: %q", digestArg))
		return
	}

	sourceReader, err := env.GetLocalReadBlobStore().MakeBlobReader(&digest)
	if err != nil {
		env.Cancel(errors.Wrapf(err, "reading source blob %q", digestArg))
		return
	}

	defer errors.DeferredCloser(&err, sourceReader)

	body, err := io.ReadAll(sourceReader)
	if err != nil {
		env.Cancel(errors.Wrap(err))
		return
	}

	destStore := cmd.MakeBlobStoreFromIdOrConfigPath(
		env.GetEnvBlobStore(),
		destBasePath,
		destConfigPath,
	)

	var writer mad_domain_interfaces.BlobWriter

	if writer, err = destStore.MakeBlobWriter(nil); err != nil {
		env.Cancel(errors.Wrap(err))
		return
	}

	defer errors.DeferredCloser(&err, writer)

	if _, err = writer.Write(body); err != nil {
		env.Cancel(errors.Wrap(err))
		return
	}

	newDigest := writer.GetMarklId()

	if !markl.Equals(&digest, newDigest) {
		env.Cancel(errors.Errorf(
			"digest mismatch after copy: source %s, written %s",
			digest.String(),
			newDigest.String(),
		))
		return
	}

	env.GetUI().Printf("copied blob %s into %s", digest.String(), destBasePath)
}
