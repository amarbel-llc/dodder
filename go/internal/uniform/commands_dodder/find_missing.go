package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
)

func init() {
	utility.AddCmd("find-missing", &FindMissing{})
}

func (cmd FindMissing) GetDescription() command.Description {
	return command.Description{
		Short: "find blob digests missing from stores",
	}
}

type FindMissing struct {
	command_components_dodder.LocalWorkingCopy
}

var _ command.CommandWithArgs = (*FindMissing)(nil)

func (cmd *FindMissing) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "blob-digests",
			Description: "blob digests to check for in stores",
			Variadic:    true,
		}},
	}}
}

func (cmd FindMissing) Run(dep command.Request) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(dep)

	var lookupStored map[string][]string

	{
		var err error

		if lookupStored, err = localWorkingCopy.GetStore().MakeBlobDigestObjectIdsMap(); err != nil {
			dep.Cancel(err)
		}
	}

	for _, blobDigestString := range dep.PopArgs() {
		var blobDigest markl.Id

		if err := markl.SetMaybeSha256(
			&blobDigest,
			blobDigestString,
		); err != nil {
			localWorkingCopy.Cancel(err)
		}

		objectIds, ok := lookupStored[string(blobDigest.GetBytes())]

		if ok {
			localWorkingCopy.GetUI().Printf(
				"%s (checked in as %q)",
				&blobDigest,
				objectIds,
			)
		} else {
			localWorkingCopy.GetUI().Printf("%s (missing)", &blobDigest)
		}
	}
}
