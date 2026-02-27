package options_print

type V1 struct {
	Abbreviations *abbreviationsV1 `toml:"abbreviations"`
	boxV1

	PrintMatchedDormant *bool `toml:"print-matched-dormant"`
	PrintShas           *bool `toml:"print-shas"`
	PrintFlush          *bool `toml:"print-flush"`
	PrintUnchanged      *bool `toml:"print-unchanged"`
	PrintColors         *bool `toml:"print-colors"`
	PrintInventoryLists *bool `toml:"print-inventory_lists"`
}

type abbreviationsV1 struct {
	ZettelIds *bool `toml:"zettel-ids"`
	MarklIds  *bool `toml:"shas"`
}

type boxV1 struct {
	PrintIncludeDescription *bool `toml:"print-include-description"`
	PrintTime               *bool `toml:"print-time"`
	PrintTagsAlways         *bool `toml:"print-etiketten-always"`
	PrintEmptyShas          *bool `toml:"print-empty-shas"`
	PrintIncludeTypes       *bool `toml:"print-include-typen"`
	PrintTai                *bool `toml:"-"`
	DescriptionInBox        *bool `toml:"-"`
	ExcludeFields           *bool `toml:"-"`
}

func (blob V1) GetAbbreviations() *OverlayAbbreviations {
	if blob.Abbreviations == nil {
		return nil
	} else {
		return (*OverlayAbbreviations)(blob.Abbreviations)
	}
}

func (blob V1) GetPrintOptionsOverlay() Overlay {
	return Overlay{
		Abbreviations: blob.GetAbbreviations(),
		OverlayBox: OverlayBox{
			PrintIncludeDescription: blob.PrintIncludeDescription,
			PrintTime:               blob.PrintTime,
			PrintTagsAlways:         blob.PrintTagsAlways,
			PrintEmptyMarklIds:      blob.PrintEmptyShas,
			PrintIncludeTypes:       blob.PrintIncludeTypes,
			PrintTai:                blob.PrintTai,
			DescriptionInBox:        blob.DescriptionInBox,
			ExcludeFields:           blob.ExcludeFields,
		},
		PrintMatchedDormant: blob.PrintMatchedDormant,
		PrintBlobDigests:    blob.PrintShas,
		PrintFlush:          blob.PrintFlush,
		PrintUnchanged:      blob.PrintUnchanged,
		PrintColors:         blob.PrintColors,
		PrintInventoryLists: blob.PrintInventoryLists,
	}
}
