package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/collections_ptr"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd(
		"checkin-blob",
		&CheckinBlob{
			NewTags: collections_ptr.MakeFlagCommas[ids.TagStruct](
				collections_ptr.SetterPolicyAppend,
			),
		})
}

type CheckinBlob struct {
	command_components_dodder.LocalWorkingCopy

	Delete  bool
	NewTags collections_ptr.Flag[ids.TagStruct, *ids.TagStruct]
}

var _ interfaces.CommandComponentWriter = (*CheckinBlob)(nil)

func (cmd *CheckinBlob) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	flagDefinitions.BoolVar(
		&cmd.Delete,
		"delete",
		false,
		"the checked-out file",
	)
	flagDefinitions.Var(
		cmd.NewTags,
		"new-tags",
		"comma-separated tags (will replace existing tags)",
	)
}

func (cmd CheckinBlob) Run(req command.Request) {
	args := req.PopArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	if len(args)%2 != 0 {
		errors.ContextCancelWithErrorf(
			req,
			"arguments must come in pairs of zettel id and blob path or sha",
		)
	}

	pairs := make([]externalBlobPair, len(args)/2)

	// transform args into pairs of object id's and filepaths or shas
	for idx, pair := range pairs {
		// TODO switch to using object ID instead to allow
		zettelIdString := args[idx*2]
		filepathOrSha := args[(idx*2)+1]

		if err := pair.SetArgs(
			zettelIdString,
			filepathOrSha,
			localWorkingCopy.GetEnvRepo(),
		); err != nil {
			req.Cancel(err)
		}

		pairs[idx] = pair
	}

	for idx, pair := range pairs {
		// iterate through pairs and read current zettel
		{
			var err error

			if pairs[idx].object, err = localWorkingCopy.GetStore().ReadTransactedFromObjectId(
				pair.ZettelId,
			); err != nil {
				req.Cancel(err)
			}
		}

		object := pairs[idx].object

		if err := object.SetBlobDigest(pair.GetDigest()); err != nil {
			req.Cancel(err)
		}

		if cmd.NewTags.Len() > 0 {
			m := object.GetMetadataMutable()
			objects.SetTags(m, cmd.NewTags)
		}
	}

	req.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock))

	for _, pair := range pairs {
		if err := localWorkingCopy.GetStore().CreateOrUpdateDefaultProto(
			pair.object,
			sku.StoreOptions{
				MergeCheckedOut: true,
			},
		); err != nil {
			req.Cancel(err)
		}
	}

	req.Must(errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock))
}

type externalBlobPair struct {
	objectIdString string
	pathOrDigest   string

	*ids.ZettelId
	BlobFD     fd.FD
	BlobDigest markl.Id

	object *sku.Transacted
}

func (pair *externalBlobPair) SetArgs(
	objectIdString, pathOrDigest string,
	envRepo env_repo.Env,
) (err error) {
	pair.BlobFD.Reset()
	pair.BlobDigest.Reset()

	pair.objectIdString = objectIdString
	pair.pathOrDigest = pathOrDigest

	if pair.ZettelId, err = ids.MakeZettelId(pair.objectIdString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = pair.BlobFD.SetFromPath(
		envRepo.GetCwd(),
		pathOrDigest,
		envRepo.GetDefaultBlobStore(),
	); err != nil {
		if errors.IsNotExist(err) {
			if err = markl.SetMaybeSha256(
				&pair.BlobDigest,
				pair.pathOrDigest,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		} else {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (pair *externalBlobPair) GetDigest() domain_interfaces.MarklId {
	if !pair.BlobFD.IsEmpty() {
		return pair.BlobFD.GetDigest()
	} else {
		return pair.BlobDigest
	}
}

func (pair *externalBlobPair) PopulateBlobDigest() (err error) {
	if err = pair.object.SetBlobDigest(pair.GetDigest()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
