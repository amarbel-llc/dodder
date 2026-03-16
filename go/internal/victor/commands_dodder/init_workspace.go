package commands_dodder

import (
	"os"
	"path/filepath"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
)

func init() {
	utility.AddCmd(
		"init-workspace",
		&InitWorkspace{})
}

type InitWorkspace struct {
	command_components_dodder.Genesis
	command_components_dodder.RemoteTransfer
	command_components_dodder.Query

	complete command_components_dodder.Complete

	ExperimentalRepo  bool
	DefaultQueryGroup values.String
	Proto             sku.Proto
}

var _ interfaces.CommandComponentWriter = (*InitWorkspace)(nil)

func (cmd *InitWorkspace) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(
		&cmd.ExperimentalRepo,
		"experimental-repo",
		false,
		"create a repo-backed workspace with independent store and commit history",
	)

	cmd.Genesis.SetFlagDefinitions(flagSet)
	cmd.RemoteTransfer.SetFlagDefinitions(flagSet)
	cmd.Query.SetFlagDefinitions(flagSet)

	flagSet.Var(
		cmd.complete.GetFlagValueMetadataTags(&cmd.Proto.Metadata),
		"tags",
		"tags added for new objects in `checkin`, `new`, `organize`",
	)

	flagSet.Var(
		cmd.complete.GetFlagValueMetadataType(&cmd.Proto.Metadata),
		"type",
		"type used for new objects in `new` and `organize`",
	)

	flagSet.Var(
		cmd.complete.GetFlagValueStringTags(&cmd.DefaultQueryGroup),
		"query",
		"default query for `show`",
	)
}

func (cmd InitWorkspace) Complete(
	_ command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	searchDir := envLocal.GetCwd()

	if commandLine.InProgress != "" && files.Exists(commandLine.InProgress) {
		var err error

		if commandLine.InProgress, err = filepath.Abs(commandLine.InProgress); err != nil {
			envLocal.Cancel(err)
			return
		}

		if searchDir, err = filepath.Rel(searchDir, commandLine.InProgress); err != nil {
			envLocal.Cancel(err)
			return
		}
	}

	for dirEntry, err := range files.WalkDir(searchDir) {
		if err != nil {
			envLocal.Cancel(err)
			return
		}

		if !dirEntry.IsDir() {
			continue
		}

		if files.WalkDirIgnoreFuncHidden(dirEntry) {
			continue
		}

		envLocal.GetUI().Printf("%s/\tdirectory", dirEntry.RelPath)
	}
}

func (cmd InitWorkspace) Run(req command.Request) {
	if cmd.ExperimentalRepo {
		cmd.runExperimentalRepo(req)
		return
	}

	cmd.runLightweight(req)
}

func (cmd InitWorkspace) runLightweight(req command.Request) {
	envLocal := cmd.Genesis.MakeEnv(req)

	switch req.RemainingArgCount() {
	case 0:
		break

	case 1:
		dir := req.PopArg("dir")

		if err := envLocal.MakeDirs(dir); err != nil {
			req.Cancel(err)
			return
		}

		if err := os.Chdir(dir); err != nil {
			req.Cancel(err)
			return
		}
	}

	req.AssertNoMoreArgs()

	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	blob := &workspace_config_blobs.V0{
		Query: cmd.DefaultQueryGroup.String(),
		Defaults: repo_configs.DefaultsV1OmitEmpty{
			Type: cmd.Proto.Metadata.GetType().ToType(),
			Tags: slices.Collect(ids.ITagSeqToTagStructSeq(cmd.Proto.Metadata.AllTags())),
		},
	}

	if err := localWorkingCopy.GetEnvWorkspace().CreateWorkspace(
		blob,
	); err != nil {
		req.Cancel(err)
	}
}

func (cmd InitWorkspace) runExperimentalRepo(req command.Request) {
	if !cmd.IsDirectTransfer() {
		req.Cancel(
			errors.BadRequestf(
				"-direct <parent-path> is required with -experimental-repo",
			),
		)
		return
	}

	cmd.Genesis.BigBang.ExcludeDefaultType = true

	local := cmd.OnTheFirstDay(req, req.PopArg("workspace repo id"))

	remote := cmd.MakeDirectRemoteFromPath(req, local)

	queryArgs := req.PopArgs()

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		queryArgs,
	)

	if err := local.PullQueryGroupFromRemote(
		remote,
		queryGroup,
		cmd.WithPrintCopies(true),
	); err != nil {
		req.Cancel(err)
		return
	}

	absParentPath := cmd.DirectPath
	if !filepath.IsAbs(absParentPath) {
		var err error

		if absParentPath, err = filepath.Abs(absParentPath); err != nil {
			req.Cancel(err)
			return
		}
	}

	blob := &workspace_config_blobs.V1{
		V0: workspace_config_blobs.V0{
			Query: cmd.DefaultQueryGroup.String(),
			Defaults: repo_configs.DefaultsV1OmitEmpty{
				Type: cmd.Proto.Metadata.GetType().ToType(),
				Tags: slices.Collect(
					ids.ITagSeqToTagStructSeq(cmd.Proto.Metadata.AllTags()),
				),
			},
		},
		ParentPath: absParentPath,
	}

	if err := local.GetEnvWorkspace().CreateWorkspace(blob); err != nil {
		req.Cancel(err)
	}
}

