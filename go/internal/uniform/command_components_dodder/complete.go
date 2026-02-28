package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/sku_fmt"
	pkg_query "code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
)

type Complete struct {
	ObjectMetadata
	Query
}

func (cmd Complete) GetFlagValueMetadataTags(
	metadata objects.MetadataMutable,
) interfaces.FlagValue {
	return command.FlagValueCompleter{
		FlagValue: cmd.ObjectMetadata.GetFlagValueMetadataTags(metadata),
		FuncCompleter: func(
			req command.Request,
			envLocal env_local.Env,
			commandLine command.CommandLineInput,
		) {
			local := LocalWorkingCopy{}.MakeLocalWorkingCopy(req)

			cmd.CompleteObjects(
				req,
				local,
				pkg_query.BuilderOptionDefaultGenres(genres.Tag),
			)
		},
	}
}

func (cmd Complete) GetFlagValueStringTags(
	value *values.String,
) interfaces.FlagValue {
	return command.FlagValueCompleter{
		FlagValue: value,
		FuncCompleter: func(
			req command.Request,
			envLocal env_local.Env,
			commandLine command.CommandLineInput,
		) {
			local := LocalWorkingCopy{}.MakeLocalWorkingCopy(req)

			cmd.CompleteObjects(
				req,
				local,
				pkg_query.BuilderOptionDefaultGenres(genres.Tag),
			)
		},
	}
}

func (cmd Complete) GetFlagValueMetadataType(
	metadata objects.MetadataMutable,
) interfaces.FlagValue {
	return command.FlagValueCompleter{
		FlagValue: cmd.ObjectMetadata.GetFlagValueMetadataType(metadata),
		FuncCompleter: func(
			req command.Request,
			envLocal env_local.Env,
			commandLine command.CommandLineInput,
		) {
			local := LocalWorkingCopy{}.MakeLocalWorkingCopy(req)

			cmd.CompleteObjects(
				req,
				local,
				pkg_query.BuilderOptionDefaultGenres(genres.Type),
			)
		},
	}
}

func (cmd Complete) SetFlagsProto(
	proto *sku.Proto,
	flagSet interfaces.CLIFlagDefinitions, descriptionUsage string,
	tagUsage string,
	typeUsage string,
) {
	proto.Metadata.SetFlagSetDescription(
		flagSet,
		descriptionUsage,
	)

	flagSet.Var(
		cmd.GetFlagValueMetadataTags(&proto.Metadata),
		"tags",
		tagUsage,
	)

	flagSet.Var(
		cmd.GetFlagValueMetadataType(&proto.Metadata),
		"type",
		typeUsage,
	)
}

func (cmd Complete) CompleteObjectsIncludingWorkspace(
	req command.Request,
	local *local_working_copy.Repo,
	queryBuilderOptions pkg_query.BuilderOption,
	args ...string,
) {
	sku_fmt.OutputCliCompletions(
		local.GetEnvRepo().Env,
		local.GetAbbr().GetSeenIds(),
	)
	// printerCompletions := sku_fmt.MakePrinterComplete(local)

	// query := cmd.MakeQueryIncludingWorkspace(
	// 	req,
	// 	queryBuilderOptions,
	// 	local,
	// 	args,
	// )

	// if err := local.GetStore().QueryTransacted(
	// 	query,
	// 	printerCompletions.PrintOne,
	// ); err != nil {
	// 	local.Cancel(err)
	// }
}

func (cmd Complete) CompleteObjects(
	req command.Request,
	local *local_working_copy.Repo,
	queryBuilderOptions pkg_query.BuilderOption,
	args ...string,
) {
	sku_fmt.OutputCliCompletions(
		local.GetEnvRepo().Env,
		local.GetAbbr().GetSeenIds(),
	)
	// printerCompletions := sku_fmt.MakePrinterComplete(local)

	// query := cmd.MakeQuery(
	// 	req,
	// 	queryBuilderOptions,
	// 	local,
	// 	args,
	// )

	// if err := local.GetStore().QueryTransacted(
	// 	query,
	// 	printerCompletions.PrintOne,
	// ); err != nil {
	// 	local.Cancel(err)
	// }
}
