package queries

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/lua"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func MakeBuilder(
	envRepo env_repo.Env,
	typedBlobStore typed_blob_store.Stores,
	objectProbeIndex sku.ObjectProbeIndex,
	luaVMPoolBuilder *lua.VMPoolBuilder,
	workspaceStoreGetter store_workspace.StoreGetter,
) (b *Builder) {
	b = &Builder{
		envRepo:              envRepo,
		typedBlobStore:       typedBlobStore,
		objectProbeIndex:     objectProbeIndex,
		luaVMPoolBuilder:     luaVMPoolBuilder,
		workspaceStoreGetter: workspaceStoreGetter,
	}

	return b
}

type Builder struct {
	envRepo                 env_repo.Env
	typedBlobStore          typed_blob_store.Stores
	objectProbeIndex        sku.ObjectProbeIndex
	luaVMPoolBuilder        *lua.VMPoolBuilder
	pinnedObjectIds         []pinnedObjectId
	pinnedExternalObjectIds []sku.ExternalObjectId
	workspaceStoreGetter    store_workspace.StoreGetter
	repoId                  ids.RepoId
	fileExtensions          file_extensions.Config
	expanders               ids.Abbr
	hidden                  sku.Query
	doNotMatchEmpty         bool
	debug                   bool
	requireNonEmptyQuery    bool
	defaultQuery            string
	workspaceEnabled        bool

	options options
}

func (builder *Builder) makeState() *buildState {
	state := &buildState{
		options: builder.options,
		builder: builder,
	}

	if builder.luaVMPoolBuilder != nil {
		state.luaVMPoolBuilder = builder.luaVMPoolBuilder.Clone()
	}

	state.group = state.makeGroup()

	state.pinnedObjectIds = make([]pinnedObjectId, len(builder.pinnedObjectIds))
	copy(state.pinnedObjectIds, builder.pinnedObjectIds)

	state.pinnedExternalObjectIds = make(
		[]sku.ExternalObjectId,
		len(builder.pinnedExternalObjectIds),
	)

	copy(state.pinnedExternalObjectIds, builder.pinnedExternalObjectIds)

	return state
}

func (builder *Builder) WithOptions(options BuilderOption) *Builder {
	if options == nil {
		return builder
	}

	applied := options.Apply(builder)

	if applied != nil {
		return applied
	}

	return builder
}

func (builder *Builder) WithExternalLike(
	zts sku.SkuTypeSet,
) *Builder {
	for t := range zts.All() {
		if t.GetExternalObjectId().IsEmpty() {
			builder.pinnedObjectIds = append(
				builder.pinnedObjectIds,
				pinnedObjectId{
					Sigil: ids.SigilExternal,
					ObjectId: ObjectId{
						Exact:    true,
						ObjectId: t.GetObjectId(),
					},
				},
			)
		} else {
			if t.GetExternalObjectId().GetGenre() == genres.Unknown {
				panic(
					errors.BadRequestf(
						"External object ID has an empty genre: %q",
						t.GetExternalObjectId(),
					),
				)
			}

			builder.pinnedExternalObjectIds = append(
				builder.pinnedExternalObjectIds,
				t.GetExternalObjectId(),
			)
		}
	}

	return builder
}

func (builder *Builder) WithTransacted(
	zts sku.TransactedSet,
	sigil ids.Sigil,
) *Builder {
	for t := range zts.All() {
		clonedId, _ := t.GetObjectId().Clone()
		builder.pinnedObjectIds = append(
			builder.pinnedObjectIds,
			pinnedObjectId{
				Sigil: sigil,
				ObjectId: ObjectId{
					ObjectId: clonedId,
				},
			},
		)
	}

	return builder
}

func (builder *Builder) BuildQueryGroupWithRepoId(
	externalQueryOptions sku.ExternalQueryOptions,
	values ...string,
) (query *Query, err error) {
	state := builder.makeState()

	if builder.workspaceEnabled {
		ok := false

		state.workspaceStore, ok = builder.workspaceStoreGetter.GetWorkspaceStoreForQuery(
			externalQueryOptions.RepoId,
		)

		state.group.RepoId = externalQueryOptions.RepoId
		state.group.ExternalQueryOptions = externalQueryOptions

		if !ok {
			err = errors.ErrorWithStackf(
				"kasten not found: %q",
				externalQueryOptions.RepoId,
			)
			return query, err
		}
	}

	if err = builder.build(state, values...); err != nil {
		err = errors.Wrap(err)
		return query, err
	}

	query = state.group

	return query, err
}

func (builder *Builder) BuildQueryGroup(
	args ...string,
) (group *Query, err error) {
	state := builder.makeState()

	if err = builder.build(state, args...); err != nil {
		err = errors.Wrap(err)
		return group, err
	}

	group = state.group

	return group, err
}

func (builder *Builder) build(state *buildState, values ...string) (err error) {
	var latent *errors.GroupBuilder

	if err, latent = state.build(values...); err != nil {
		if !errors.Is400BadRequest(err) {
			latent.Add(errors.Wrapf(err, "Query String: %q", values))
			err = latent.GetError()
		}

		errors.Wrap(err)

		return err
	}

	if len(state.missingBlobs) > 0 {
		groupBuilder := errors.MakeGroupBuilder()

		for _, e := range state.missingBlobs {
			groupBuilder.Add(e)
		}

		err = groupBuilder.GetError()

		return err
	}

	if builder.defaultQuery == "" {
		return err
	}

	defaultQueryGroupState := state.copy()
	defaultQueryGroupState.options.defaultGenres = ids.MakeGenre(
		genres.All()...)

	if err, _ = defaultQueryGroupState.build(builder.defaultQuery); err != nil {
		err = errors.Wrap(err)
		return err
	}

	state.group.defaultQuery = defaultQueryGroupState.group

	ui.Log().Print(state.group.StringDebug())

	return err
}
