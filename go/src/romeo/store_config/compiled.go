package store_config

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/comments"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/values"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/charlie/options_print"
	"code.linenisgreat.com/dodder/go/src/delta/file_extensions"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/src/golf/repo_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/genesis_configs"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
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
		InlineTypes       interfaces.Set[values.String]

		// Kasten
		Repos sku.TransactedMutableSet

		FileExtensions file_extensions.Config

		PrintOptions options_print.Options
	}
)

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

	b, _ := sku.GetTransactedPool().GetWithRepool()

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

	b, _ := sku.GetTransactedPool().GetWithRepool()

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

func (config Config) IsInlineType(tipe ids.Type) (isInline bool) {
	comments.Change("fix this horrible hack")
	if tipe.IsEmpty() {
		return true
	}

	isInline = config.InlineTypes.ContainsKey(tipe.String()) ||
		ids.IsBuiltin(tipe)

	return isInline
}
