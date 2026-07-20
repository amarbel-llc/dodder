package store_config

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/file_extensions"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_id"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	configRepo    = repo_configs.ConfigOverlay
	configGenesis = genesis_configs.ConfigPrivate
	CLI           = repo_config_cli.Config

	Config struct {
		*compiled

		configGenesis

		// TODO combine below into repo_configs.Config
		configRepo
		CLI
	}

	compiled struct {
		// TODO move to store
		lock sync.Mutex

		// TODO move to store
		changes []string

		// TODO move to store
		Sku sku.Transacted

		Tags         interfaces.SetMutable[*tag]
		ImplicitTags implicitTagMap

		// Typen
		ExtensionsToTypes map[string]string
		TypesToExtensions map[string]string
		Types             sku.TransactedMutableSet

		// inlineTypeChecker is the index-backed resolver injected post-Initialize
		// by the owning store (see StoreMutable.SetInlineTypeChecker). IsInlineType
		// delegates to it; nil before injection.
		inlineTypeChecker ids.InlineTypeChecker

		// Kasten
		Repos sku.TransactedMutableSet

		FileExtensions file_extensions.Config

		PrintOptions options_print.Options
	}
)

func (config Config) GetBlobStores() []blob_store_id.Id {
	return repo_configs.GetBlobStores(config.configRepo, nil)
}

// GetStreamIndexFixed resolves whether the fixed-size-row-with-overflow stream
// index should be used. The persisted repo config overlay supplies the default;
// the CLI flag (--stream-index-fixed) forces it on regardless of the overlay.
func (config Config) GetStreamIndexFixed() bool {
	if config.CLI.UseStreamIndexFixed() {
		return true
	}

	return repo_configs.GetStreamIndexFixed(config.configRepo, false)
}

func (config Config) GetPrintOptions() options_print.Options {
	return config.PrintOptions
}

func (compiled *compiled) GetSku() *sku.Transacted {
	return &compiled.Sku
}

func (compiled *compiled) addRepo(
	object *sku.Transacted,
) (didChange bool, err error) {
	compiled.lock.Lock()
	defer compiled.lock.Unlock()

	b, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	sku.Resetter.ResetWith(b, object)

	if didChange, err = quiter.AddOrReplaceIfGreater(
		compiled.Repos,
		b,
		sku.TransactedCompare,
	); err != nil {
		err = errors.Wrap(err)
		return didChange, err
	}

	return didChange, err
}

func (compiled *compiled) addType(
	object *sku.Transacted,
) (didChange bool, err error) {
	if err = genres.Type.AssertGenre(object); err != nil {
		err = errors.Wrap(err)
		return didChange, err
	}

	b, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	sku.Resetter.ResetWith(b, object)

	compiled.lock.Lock()
	defer compiled.lock.Unlock()

	if didChange, err = quiter.AddOrReplaceIfGreater(
		compiled.Types,
		b,
		sku.TransactedCompare,
	); err != nil {
		err = errors.Wrap(err)
		return didChange, err
	}

	return didChange, err
}

func (config Config) GetTypeStringFromExtension(t string) string {
	return config.ExtensionsToTypes[t]
}

func (config Config) GetTypeExtension(v string) string {
	return config.TypesToExtensions[v]
}

// IsInlineType delegates to the index-backed checker injected by the owning
// store (SetInlineTypeChecker). Resolution is deterministic against the
// signature-backed stream index — see (*store.Store).IsInlineType. The empty
// type renders inline (the descriptionless / typeless case). If the checker has
// not been injected yet (e.g. config-only contexts before store wiring), fall
// back to treating the type as inline so metadata-only is never forced by a
// missing dependency.
func (config Config) IsInlineType(tipe ids.Type) bool {
	if tipe.IsEmpty() {
		return true
	}

	if config.inlineTypeChecker == nil {
		return true
	}

	return config.inlineTypeChecker.IsInlineType(tipe)
}
