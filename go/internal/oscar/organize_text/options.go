package organize_text

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/internal/bravo/flags"
	"code.linenisgreat.com/dodder/go/internal/charlie/collections_ptr"
	"code.linenisgreat.com/dodder/go/internal/charlie/options_print"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
)

type Flags struct {
	Options

	once      *sync.Once
	ExtraTags collections_ptr.Flag[ids.TagStruct, *ids.TagStruct]
}

var _ interfaces.CommandComponentWriter = (*Flags)(nil)

type Options struct {
	wasMade bool

	Config domain_interfaces.MutableConfigDryRun

	Metadata

	commentMatchers interfaces.Set[sku.Query]
	GroupingTags    ids.TagSlice
	ExtraTags       ids.TagSet
	Skus            sku.SkuTypeSet

	sku.ObjectFactory

	Abbr ids.Abbr

	UsePrefixJoints   bool
	UseRefiner        bool
	UseMetadataHeader bool
	Limit             int

	PrintOptions options_print.Options
	fmtBox       *box_format.BoxCheckedOut
}

func MakeFlags() Flags {
	return Flags{
		once: &sync.Once{},
		ExtraTags: collections_ptr.MakeFlagCommas[ids.TagStruct](
			collections_ptr.SetterPolicyAppend,
		),

		Options: Options{
			wasMade:      true,
			GroupingTags: collections_slice.MakeFromSlice[ids.TagStruct](),
			Skus:         sku.MakeSkuTypeSetMutable(),
			Metadata:     NewMetadata(ids.RepoId{}),
		},
	}
}

func MakeFlagsWithMetadata(metadata Metadata) Flags {
	if metadata.TagSet == nil {
		metadata.TagSet = ids.MakeTagSetFromSlice()
	}

	return Flags{
		once: &sync.Once{},
		ExtraTags: collections_ptr.MakeFlagCommas[ids.TagStruct](
			collections_ptr.SetterPolicyAppend,
		),

		Options: Options{
			Metadata:     metadata,
			wasMade:      true,
			GroupingTags: collections_slice.MakeFromSlice[ids.TagStruct](),
			Skus:         sku.MakeSkuTypeSetMutable(),
		},
	}
}

func (flagz *Flags) SetFlagDefinitions(flagDefs interfaces.CLIFlagDefinitions) {
	flagDefs.Func(
		"group-by",
		"tag prefixes to group objects",
		func(valueOrValues string) (err error) {
			seq := flags.SplitCommasAndTrimAndMake[ids.TagStruct](valueOrValues)

			var tag ids.TagStruct

			for tag, err = range seq {
				if err != nil {
					err = errors.Wrap(err)
					return err
				}

				flagz.GroupingTags.Append(tag)
			}

			return err
		},
	)

	flagDefs.Var(
		flagz.ExtraTags,
		"extras",
		"tags to always add to the organize text",
	)

	flagDefs.BoolVar(
		&flagz.UsePrefixJoints,
		"prefix-joints",
		true,
		"split tags around hyphens",
	)

	flagDefs.BoolVar(&flagz.UseRefiner, "refine", true, "refine the organize tree")

	flagDefs.BoolVar(
		&flagz.UseMetadataHeader,
		"metadata-header",
		true,
		"metadata header",
	)

	flagDefs.IntVar(
		&flagz.Limit,
		"limit",
		0,
		"limit the number of objects edited in organize",
	)
}

func (flagz *Flags) GetOptionsWithMetadata(
	printOptions options_print.Options,
	boxFormat *box_format.BoxCheckedOut,
	abbr ids.Abbr,
	objectFactory sku.ObjectFactory,
	metadata Metadata,
) Options {
	flagz.once.Do(
		func() {
			flagz.Options.ExtraTags = ids.CloneTagSet(flagz.ExtraTags)
		},
	)

	flagz.fmtBox = boxFormat

	objectFactory.SetDefaultsIfNecessary()

	flagz.ObjectFactory = objectFactory
	flagz.PrintOptions = printOptions
	flagz.Abbr = abbr
	flagz.Metadata = metadata

	return flagz.Options
}

func (flagz *Flags) GetOptions(
	printOptions options_print.Options,
	tagSet ids.TagSet,
	skuBoxFormat *box_format.BoxCheckedOut,
	abbr ids.Abbr, // TODO move Abbr as required arg
	objectFactory sku.ObjectFactory,
) Options {
	m := flagz.Metadata
	m.TagSet = tagSet

	if m.prototype == nil {
		panic("Metadata not initalized")
	}

	return flagz.GetOptionsWithMetadata(
		printOptions,
		skuBoxFormat,
		abbr,
		objectFactory,
		m,
	)
}

func (o Options) Make() (ot *Text, err error) {
	c := &constructor{
		Text: Text{
			Options: o,
		},
	}

	ot = &c.Text

	c.all = MakePrefixSet(0)
	c.Assignment = newAssignment(0)
	c.IsRoot = true

	if c.TagSet == nil {
		c.TagSet = ids.MakeTagSetFromSlice()
	}

	var objects Objects

	for sk := range c.Options.Skus.All() {
		objects.Add(&obj{sku: sk})
	}

	objects.Sort()

	for i, obj := range objects {
		if i != 0 && i == o.Limit {
			break
		}

		if err = c.all.AddSku(obj.sku); err != nil {
			err = errors.Wrap(err)
			return ot, err
		}
	}

	if err = c.preparePrefixSetsAndRootsAndExtras(); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	if err = c.populate(); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	c.Metadata.Type = c.Options.Type

	if err = ot.Refine(); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	ot.SortChildren()

	return ot, err
}

func (o Options) refiner() *Refiner {
	return &Refiner{
		Enabled:         o.UseRefiner,
		UsePrefixJoints: o.UsePrefixJoints,
	}
}
