package store_workspace

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/store_workspace"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/mike/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
)

type (
	Supplies struct {
		WorkspaceDir string
		sku.RepoStore
		DirCache string
		env_repo.Env
		ids.RepoId
		ids.TypeSet
		ids.Clock
		BlobStore typed_blob_store.Stores // TODO reduce this dependency
	}

	CheckoutOne interface {
		CheckoutOne(
			options checkout_options.Options,
			sz sku.TransactedGetter,
		) (cz sku.SkuType, err error)
	}

	DeleteCheckedOut interface {
		DeleteCheckedOut(el *sku.CheckedOut) (err error)
	}

	UpdateTransacted = sku.ExternalStoreUpdateTransacted

	UpdateTransactedFromBlobs interface {
		UpdateTransactedFromBlobs(sku.ExternalLike) (err error)
	}

	Open interface {
		Open(
			m checkout_mode.Mode,
			ph interfaces.FuncIter[string],
			zsc sku.SkuTypeSet,
		) (err error)
	}

	UpdateCheckoutFromCheckedOut interface {
		UpdateCheckoutFromCheckedOut(
			options checkout_options.OptionsWithoutMode,
			co sku.SkuType,
		) (err error)
	}

	ReadCheckedOutFromTransacted interface {
		ReadCheckedOutFromTransacted(
			object *sku.Transacted,
		) (checkedOut *sku.CheckedOut, err error)
	}

	Merge interface {
		Merge(conflicted sku.Conflicted) (err error)
	}

	MergeCheckedOut interface {
		MergeCheckedOut(
			co *sku.CheckedOut,
			parentNegotiator sku.ParentNegotiator,
			allowMergeConflicts bool,
		) (commitOptions sku.CommitOptions, err error)
	}

	QueryCheckedOut = queries.QueryCheckedOut

	StoreLike interface {
		Initialize(Supplies) error
		QueryCheckedOut
		errors.Flusher
		store_workspace.Store
	}
)
