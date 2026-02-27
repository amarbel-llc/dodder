package queries

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/delta/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/hotel/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

// TODO consider moving this whole file into its own package

type BuilderOption interface {
	Apply(*Builder) *Builder
}

type builderOptions []BuilderOption

// nil options are permitted, they are just skipped during application
func BuilderOptions(options ...BuilderOption) builderOptions {
	return builderOptions(options)
}

func (options builderOptions) Apply(builder *Builder) *Builder {
	for _, option := range options {
		if option == nil {
			continue
		}

		builder = option.Apply(builder)
	}

	return builder
}

type BuilderOptionWorkspaceConfigEnv interface {
	env_ui.Env
	GetWorkspaceConfig() workspace_config_blobs.Config
}

func BuilderOptionWorkspace(
	env BuilderOptionWorkspaceConfigEnv,
) BuilderOption {
	cliConfig := env.GetCLIConfig()
	var workspaceConfig workspace_config_blobs.Config

	if env != nil {
		workspaceConfig = env.GetWorkspaceConfig()
	}

	_, isTemporaryWorkspace := workspaceConfig.(workspace_config_blobs.ConfigTemporary)

	var builder builderOptionWorkspace

	var ignoreWorkspace bool
	if repoConfig, ok := cliConfig.(domain_interfaces.RepoCLIConfigProvider); ok {
		ignoreWorkspace = repoConfig.GetIgnoreWorkspace()
	}

	if isTemporaryWorkspace {
		builder.workspaceConfig = workspaceConfig
	} else if !ignoreWorkspace {
		builder.workspaceConfig = workspaceConfig
	}

	return builder
}

type builderOptionWorkspace struct {
	workspaceConfig workspace_config_blobs.Config
}

func (options builderOptionWorkspace) Apply(builder *Builder) *Builder {
	if options.workspaceConfig == nil {
		return builder
	}

	builder.workspaceEnabled = true

	type WithQueryGroup = workspace_config_blobs.ConfigWithDefaultQueryString

	if withQueryGroup, ok := options.workspaceConfig.(WithQueryGroup); ok {
		builder.defaultQuery = withQueryGroup.GetDefaultQueryString()
	}

	return builder
}

type options struct {
	defaultGenres  ids.Genre
	defaultSigil   ids.Sigil
	permittedSigil ids.Sigil
}

func BuilderOptionDefaultSigil(sigils ...ids.Sigil) builderOptionDefaultSigil {
	return builderOptionDefaultSigil(ids.MakeSigil(sigils...))
}

type builderOptionDefaultSigil ids.Sigil

func (option builderOptionDefaultSigil) Apply(builder *Builder) *Builder {
	builder.options.defaultSigil = ids.Sigil(option)
	return builder
}

type builderOptionDefaultGenre ids.Genre

func BuilderOptionDefaultGenres(
	genres ...genres.Genre,
) builderOptionDefaultGenre {
	return builderOptionDefaultGenre(ids.MakeGenre(genres...))
}

func (options builderOptionDefaultGenre) Apply(builder *Builder) *Builder {
	builder.options.defaultGenres = ids.Genre(options)
	return builder
}

type builderOptionPermittedSigil ids.Sigil

func BuilderOptionPermittedSigil(sigil ids.Sigil) builderOptionPermittedSigil {
	return builderOptionPermittedSigil(sigil)
}

func (option builderOptionPermittedSigil) Apply(builder *Builder) *Builder {
	builder.options.permittedSigil.Add(ids.Sigil(option))
	return builder
}

type builderOptionRequireNonEmptyQuery struct{}

func BuilderOptionRequireNonEmptyQuery() builderOptionRequireNonEmptyQuery {
	return builderOptionRequireNonEmptyQuery{}
}

func (option builderOptionRequireNonEmptyQuery) Apply(builder *Builder) *Builder {
	builder.requireNonEmptyQuery = true
	return builder
}

type builderOptionHidden struct {
	hidden sku.Query
}

func BuilderOptionHidden(hidden sku.Query) builderOptionHidden {
	return builderOptionHidden{hidden: hidden}
}

func (option builderOptionHidden) Apply(builder *Builder) *Builder {
	builder.hidden = option.hidden
	return builder
}

type builderOptionDoNotMatchEmpty struct{}

func BuilderOptionDoNotMatchEmpty() builderOptionDoNotMatchEmpty {
	return builderOptionDoNotMatchEmpty{}
}

func (option builderOptionDoNotMatchEmpty) Apply(builder *Builder) *Builder {
	builder.doNotMatchEmpty = true
	return builder
}

type builderOptionDebug struct{}

func BuilderOptionDebug() builderOptionDebug {
	return builderOptionDebug{}
}

func (option builderOptionDebug) Apply(builder *Builder) *Builder {
	builder.debug = true
	return builder
}

type builderOptionRepoId ids.RepoId

func BuilderOptionRepoId(repoId ids.RepoId) builderOptionRepoId {
	return builderOptionRepoId(repoId)
}

func (option builderOptionRepoId) Apply(builder *Builder) *Builder {
	builder.repoId = ids.RepoId(option)
	return builder
}

type builderOptionFileExtensions struct {
	fileExtensions file_extensions.Config
}

func BuilderOptionFileExtensions(config file_extensions.Config) builderOptionFileExtensions {
	return builderOptionFileExtensions{fileExtensions: config}
}

func (option builderOptionFileExtensions) Apply(builder *Builder) *Builder {
	builder.fileExtensions = option.fileExtensions
	return builder
}

type builderOptionExpanders struct {
	expanders ids.Abbr
}

func BuilderOptionExpanders(expanders ids.Abbr) builderOptionExpanders {
	return builderOptionExpanders{expanders: expanders}
}

func (option builderOptionExpanders) Apply(builder *Builder) *Builder {
	builder.expanders = option.expanders
	return builder
}
