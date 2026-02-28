package options_print

type V2 struct {
	Abbreviations *abbreviationsV2 `toml:"abbreviations"`

	DescriptionInBox        *bool `toml:"-"`
	ExcludeFields           *bool `toml:"-"`
	PrintBlobDigests        *bool `toml:"print-blob_digests"`
	PrintColors             *bool `toml:"print-colors"`
	PrintEmptyBlobDigests   *bool `toml:"print-empty-blob_digests"`
	PrintFlush              *bool `toml:"print-flush"`
	PrintIncludeDescription *bool `toml:"print-include-description"`
	PrintIncludeTypes       *bool `toml:"print-include-types"`
	PrintInventoryLists     *bool `toml:"print-inventory_lists"`
	PrintMatchedDormant     *bool `toml:"print-matched-dormant"`
	PrintTagsAlways         *bool `toml:"print-tags-always"`
	PrintTai                *bool `toml:"-"`
	PrintTime               *bool `toml:"print-time"`
	PrintUnchanged          *bool `toml:"print-unchanged"`
}

type abbreviationsV2 struct {
	ZettelIds *bool `toml:"zettel_ids"`
	MarklIds  *bool `toml:"merkle_ids"`
}

func (blob V2) GetAbbreviations() *OverlayAbbreviations {
	if blob.Abbreviations == nil {
		return nil
	} else {
		return (*OverlayAbbreviations)(blob.Abbreviations)
	}
}

func (blob V2) GetPrintOptionsOverlay() Overlay {
	return Overlay{
		Abbreviations: blob.GetAbbreviations(),
		OverlayBox: OverlayBox{
			DescriptionInBox:        blob.DescriptionInBox,
			ExcludeFields:           blob.ExcludeFields,
			PrintEmptyMarklIds:      blob.PrintEmptyBlobDigests,
			PrintIncludeDescription: blob.PrintIncludeDescription,
			PrintIncludeTypes:       blob.PrintIncludeTypes,
			PrintTagsAlways:         blob.PrintTagsAlways,
			PrintTai:                blob.PrintTai,
			PrintTime:               blob.PrintTime,
		},
		PrintBlobDigests:    blob.PrintBlobDigests,
		PrintColors:         blob.PrintColors,
		PrintFlush:          blob.PrintFlush,
		PrintInventoryLists: blob.PrintInventoryLists,
		PrintMatchedDormant: blob.PrintMatchedDormant,
		PrintUnchanged:      blob.PrintUnchanged,
	}
}
