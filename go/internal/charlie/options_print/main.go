package options_print

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/equals"
	"code.linenisgreat.com/dodder/go/internal/bravo/values"
)

type (
	OverlayAbbreviations struct {
		ZettelIds *bool
		MarklIds  *bool
	}

	OverlayBox struct {
		PrintIncludeDescription *bool
		PrintTime               *bool
		PrintTagsAlways         *bool
		PrintEmptyMarklIds      *bool
		PrintIncludeTypes       *bool
		PrintTai                *bool
		DescriptionInBox        *bool
		ExcludeFields           *bool
	}

	Overlay struct {
		Abbreviations *OverlayAbbreviations
		OverlayBox

		PrintMatchedDormant *bool
		PrintBlobDigests    *bool
		PrintFlush          *bool
		PrintUnchanged      *bool
		PrintColors         *bool
		PrintInventoryLists *bool
		Newlines            *bool
	}

	Options struct {
		AbbreviateZettelIds        bool
		AbbreviateMarklIds         bool
		BoxPrintIncludeDescription bool
		BoxPrintTime               bool
		BoxPrintTagsAlways         bool
		BoxPrintEmptyBlobIds       bool
		BoxPrintIncludeTypes       bool
		BoxPrintTai                bool
		BoxDescriptionInBox        bool
		BoxExcludeFields           bool
		PrintMatchedDormant        bool
		PrintBlobDigests           bool
		PrintFlush                 bool
		PrintUnchanged             bool
		PrintColors                bool
		PrintInventoryLists        bool
		Newlines                   bool
	}

	OverlayGetter interface {
		GetPrintOptionsOverlay() Overlay
	}

	OptionGetter interface {
		GetPrintOptions() Options
	}
)

var (
	_ OverlayGetter = Overlay{}
	_ OverlayGetter = V0{}
	_ OverlayGetter = V1{}
	_ OverlayGetter = V2{}
)

func Default() Options {
	return Options{
		AbbreviateZettelIds:        true,
		AbbreviateMarklIds:         true,
		BoxPrintIncludeTypes:       true,
		BoxPrintIncludeDescription: true,
		BoxPrintTime:               true,
		BoxPrintTagsAlways:         true,
		BoxPrintEmptyBlobIds:       false,
		PrintMatchedDormant:        false,
		PrintBlobDigests:           true,
		PrintFlush:                 true,
		PrintUnchanged:             true,
		PrintColors:                true,
		PrintInventoryLists:        true,
	}
}

func DefaultOverlay() V2 {
	config := Default()

	return V2{
		Abbreviations: &abbreviationsV2{
			ZettelIds: &config.AbbreviateZettelIds,
			MarklIds:  &config.AbbreviateMarklIds,
		},
		PrintBlobDigests:        &config.PrintBlobDigests,
		PrintColors:             &config.PrintColors,
		PrintEmptyBlobDigests:   &config.BoxPrintEmptyBlobIds,
		PrintFlush:              &config.PrintFlush,
		PrintIncludeDescription: &config.BoxPrintIncludeDescription,
		PrintIncludeTypes:       &config.BoxPrintIncludeTypes,
		PrintInventoryLists:     &config.PrintInventoryLists,
		PrintMatchedDormant:     &config.PrintMatchedDormant,
		PrintTagsAlways:         &config.BoxPrintTagsAlways,
		PrintTime:               &config.BoxPrintTime,
		PrintUnchanged:          &config.PrintUnchanged,
	}
}

func (options Options) WithPrintBlobDigests(v bool) Options {
	options.PrintBlobDigests = v
	return options
}

func (options Options) WithDescriptionInBox(v bool) Options {
	options.BoxDescriptionInBox = v
	return options
}

func (options Options) WithPrintTai(v bool) Options {
	options.BoxPrintTai = v
	return options
}

func (options Options) WithExcludeFields(v bool) Options {
	options.BoxExcludeFields = v
	return options
}

func (options Options) WithPrintTime(v bool) Options {
	options.BoxPrintTime = v
	return options
}

func (options Options) UsePrintTime() bool {
	return options.BoxPrintTime
}

func (options Options) UsePrintTags() bool {
	return options.BoxPrintTagsAlways
}

func MakeDefaultConfig(overlays ...OverlayGetter) Options {
	return MakeConfig(Default(), overlays...)
}

func MakeConfig(base Options, overlays ...OverlayGetter) Options {
	for _, overlayGetter := range overlays {
		overlay := overlayGetter.GetPrintOptionsOverlay()
		if abbreviations := overlay.Abbreviations; abbreviations != nil {
			equals.SetIfValueNotNil(
				&base.AbbreviateZettelIds,
				abbreviations.ZettelIds,
			)
			equals.SetIfValueNotNil(
				&base.AbbreviateMarklIds,
				abbreviations.MarklIds,
			)
		}

		box := overlay.OverlayBox
		equals.SetIfValueNotNil(&base.BoxDescriptionInBox, box.DescriptionInBox)
		equals.SetIfValueNotNil(&base.BoxPrintTime, box.PrintTime)
		equals.SetIfValueNotNil(&base.BoxPrintTagsAlways, box.PrintTagsAlways)
		equals.SetIfValueNotNil(
			&base.BoxPrintEmptyBlobIds,
			box.PrintEmptyMarklIds,
		)
		equals.SetIfValueNotNil(
			&base.BoxPrintIncludeTypes,
			box.PrintIncludeTypes,
		)
		equals.SetIfValueNotNil(&base.BoxPrintTai, box.PrintTai)
		equals.SetIfValueNotNil(
			&base.BoxPrintIncludeDescription,
			box.PrintIncludeDescription,
		)
		equals.SetIfValueNotNil(&base.BoxExcludeFields, box.ExcludeFields)

		equals.SetIfValueNotNil(
			&base.PrintMatchedDormant,
			overlay.PrintMatchedDormant,
		)
		equals.SetIfValueNotNil(
			&base.PrintBlobDigests,
			overlay.PrintBlobDigests,
		)
		equals.SetIfValueNotNil(&base.PrintFlush, overlay.PrintFlush)
		equals.SetIfValueNotNil(&base.PrintUnchanged, overlay.PrintUnchanged)
		equals.SetIfValueNotNil(&base.PrintColors, overlay.PrintColors)
		equals.SetIfValueNotNil(
			&base.PrintInventoryLists,
			overlay.PrintInventoryLists,
		)
		equals.SetIfValueNotNil(&base.Newlines, overlay.Newlines)
	}
	return base
}

func (overlay Overlay) GetPrintOptionsOverlay() Overlay {
	return overlay
}

func makeFlagSetFuncBoolVar(valuePtr **bool) func(value string) (err error) {
	return func(value string) (err error) {
		var boolValue values.Bool

		if err = boolValue.Set(value); err != nil {
			return err
		}

		booll := boolValue.Bool()

		*valuePtr = &booll

		return err
	}
}

func (overlay *Overlay) AddToFlags(flagSet interfaces.CLIFlagDefinitions) {
	flagSet.Func(
		"print-types",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintIncludeTypes),
	)

	// TODO-P4 combine below three options
	flagSet.Func(
		"abbreviate-shas",
		"",
		func(value string) (err error) {
			if overlay.Abbreviations == nil {
				overlay.Abbreviations = &OverlayAbbreviations{}
			}

			return makeFlagSetFuncBoolVar(
				&overlay.Abbreviations.MarklIds,
			)(
				value,
			)
		},
	)

	flagSet.Func(
		"abbreviate-zettel-ids",
		"",
		func(value string) (err error) {
			if overlay.Abbreviations == nil {
				overlay.Abbreviations = &OverlayAbbreviations{}
			}

			return makeFlagSetFuncBoolVar(
				&overlay.Abbreviations.ZettelIds,
			)(
				value,
			)
		},
	)

	flagSet.Func(
		"print-description",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintIncludeDescription),
	)

	flagSet.Func(
		"print-time",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintTime),
	)

	flagSet.Func(
		"print-tags",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintTagsAlways),
	)

	flagSet.Func(
		"print-empty-shas",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintEmptyMarklIds),
	)

	flagSet.Func(
		"print-matched-dormant",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintMatchedDormant),
	)

	flagSet.Func(
		"print-shas",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintBlobDigests),
	)

	flagSet.Func(
		"print-flush",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintFlush),
	)

	flagSet.Func(
		"print-unchanged",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintUnchanged),
	)

	flagSet.Func(
		"print-colors",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintColors),
	)

	flagSet.Func(
		"print-inventory_list",
		"",
		makeFlagSetFuncBoolVar(&overlay.PrintInventoryLists),
	)

	flagSet.Func(
		"boxed-description",
		"",
		makeFlagSetFuncBoolVar(&overlay.DescriptionInBox),
	)

	flagSet.Func(
		"zittish-newlines",
		"add extra newlines to zittish to improve readability",
		makeFlagSetFuncBoolVar(&overlay.Newlines),
	)
}
