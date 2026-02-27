package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/markl"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
)

func init() {
	utility.AddCmd("find-missing", &FindMissing{})
}

type FindMissing struct {
	command_components_dodder.LocalWorkingCopy
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
