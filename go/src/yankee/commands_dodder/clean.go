package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/checked_out_state"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/oscar/organize_text"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("clean", &Clean{})
}

type Clean struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	force                    bool
	includeRecognizedBlobs   bool
	includeRecognizedZettels bool
	includeParent            bool
	organize                 bool
}

var _ interfaces.CommandComponentWriter = (*Clean)(nil)

func (c *Clean) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	c.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(f)

	f.BoolVar(
		&c.force,
		"force",
		false,
		"remove objects in working directory even if they have changes",
	)

	f.BoolVar(
		&c.includeParent,
		"include-mutter",
		false,
		"remove objects in working directory if they match their Mutter",
	)

	f.BoolVar(
		&c.includeRecognizedBlobs,
		"recognized-blobs",
		false,
		"remove blobs in working directory or args that are recognized",
	)

	f.BoolVar(
		&c.includeRecognizedZettels,
		"recognized-zettelen",
		false,
		"remove Zetteln in working directory or args that are recognized",
	)

	f.BoolVar(&c.organize, "organize", false, "")
}

func (cmd Clean) Run(req command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionHidden(nil),
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
	)

	envWorkspace := localWorkingCopy.GetEnvWorkspace()
	envWorkspace.AssertNotTemporary(req)

	if cmd.organize {
		if err := cmd.runOrganize(localWorkingCopy, queryGroup); err != nil {
			localWorkingCopy.Cancel(err)
		}

		return
	}

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Lock),
	)

	if err := localWorkingCopy.GetStore().QuerySkuType(
		queryGroup,
		func(co sku.SkuType) (err error) {
			if !cmd.shouldClean(localWorkingCopy, co, queryGroup) {
				return err
			}

			if err = localWorkingCopy.GetStore().DeleteCheckedOut(co); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	); err != nil {
		localWorkingCopy.Cancel(err)
	}

	localWorkingCopy.Must(
		errors.MakeFuncContextFromFuncErr(localWorkingCopy.Unlock),
	)
}

func (c Clean) runOrganize(
	u *local_working_copy.Repo,
	qg *queries.Query,
) (err error) {
	opOrganize := user_ops.Organize{
		Repo: u,
		Metadata: organize_text.Metadata{
			RepoId: qg.RepoId,
			OptionCommentSet: organize_text.MakeOptionCommentSet(
				nil,
				&organize_text.OptionCommentUnknown{
					Value: "instructions: to clean an object, delete it entirely",
				},
			),
		},
		DontUseQueryGroupForOrganizeMetadata: true,
	}

	ui.Log().Print(qg)

	var organizeResults organize_text.OrganizeResults

	if organizeResults, err = opOrganize.RunWithQueryGroup(
		qg,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var changes organize_text.Changes

	if changes, err = organize_text.ChangesFromResults(
		u.GetConfig().GetPrintOptions(),
		organizeResults,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	u.Must(errors.MakeFuncContextFromFuncErr(u.Lock))

	for _, el := range changes.Removed.AllSkuAndIndex() {
		if err = u.GetStore().DeleteCheckedOut(
			el,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	u.Must(errors.MakeFuncContextFromFuncErr(u.Unlock))

	return err
}

func (cmd Clean) shouldClean(
	u *local_working_copy.Repo,
	co sku.SkuType,
	qg *queries.Query,
) bool {
	if cmd.force {
		return true
	}

	state := co.GetState()

	switch state {
	case checked_out_state.CheckedOut:
		return sku.InternalAndExternalEqualsWithoutTai(co)

	case checked_out_state.Recognized:
		return !qg.ExcludeRecognized
	}

	if cmd.includeParent {
		mother, motherRepool := sku.GetTransactedPool().GetWithRepool()
		defer motherRepool()

		err := u.GetStore().GetStreamIndex().ReadOneObjectId(
			co.GetSku().GetObjectId(),
			mother,
		)

		errors.PanicIfError(err)

		if objects.EqualerSansTai.Equals(
			co.GetSkuExternal().GetSku().GetMetadata(),
			mother.GetMetadata(),
		) {
			return true
		}
	}

	return false
}
