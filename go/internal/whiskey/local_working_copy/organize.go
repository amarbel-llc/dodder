package local_working_copy

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/organize_text"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func (local *Repo) MakeOrganizeOptionsWithOrganizeMetadata(
	organizeFlags organize_text.Flags,
	metadata organize_text.Metadata,
) organize_text.Options {
	options := organizeFlags.GetOptions(
		local.GetConfig().GetPrintOptions(),
		nil,
		local.SkuFormatBoxCheckedOutNoColor(),
		local.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
	)

	options.Metadata = metadata

	return options
}

func (local *Repo) MakeOrganizeOptionsWithQueryGroup(
	organizeFlags organize_text.Flags,
	qg *queries.Query,
) organize_text.Options {
	return organizeFlags.GetOptions(
		local.GetConfig().GetPrintOptions(),
		queries.GetTags(qg),
		local.SkuFormatBoxCheckedOutNoColor(),
		local.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
	)
}

func (local *Repo) LockAndCommitOrganizeResults(
	results organize_text.OrganizeResults,
) (changeResults organize_text.Changes, err error) {
	if changeResults, err = organize_text.ChangesFromResults(
		local.GetConfig().GetPrintOptions(),
		results,
	); err != nil {
		err = errors.Wrap(err)
		return changeResults, err
	}

	local.Must(errors.MakeFuncContextFromFuncErr(local.Lock))

	count := changeResults.Changed.Len()

	if count > 30 {
		if !local.Confirm(
			fmt.Sprintf(
				"a large number (%d) of objects are being changed. continue to commit?",
				count,
			),
			"",
		) {
			// TODO output organize file used
			errors.ContextCancelWith499ClientClosedRequest(local)
			return changeResults, err
		}
	}

	var proto sku.Proto

	workspace := local.GetEnvWorkspace()
	workspaceType := workspace.GetDefaults().GetDefaultType()

	proto.Metadata.GetTypeMutable().ResetWithType(workspaceType)

	for _, changed := range changeResults.Changed.AllSkuAndIndex() {
		if err = local.GetStore().CreateOrUpdate(
			changed.GetSkuExternal(),
			sku.CommitOptions{
				Proto: proto,
				StoreOptions: sku.StoreOptions{
					MergeCheckedOut: true,
				},
			},
		); err != nil {
			err = errors.Wrap(err)
			return changeResults, err
		}
	}

	local.Must(errors.MakeFuncContextFromFuncErr(local.Unlock))

	return changeResults, err
}

func (local *Repo) ApplyToOrganizeOptions(oo *organize_text.Options) {
	oo.Config = local.GetConfigPtr()
	oo.Abbr = local.GetStore().GetAbbrStore().GetAbbr()

	if !local.GetConfig().IsDryRun() {
		return
	}

	oo.AddPrototypeAndOption(
		"dry-run",
		&organize_text.OptionCommentDryRun{
			MutableConfigDryRun: local.GetConfigPtr(),
		},
	)
}
