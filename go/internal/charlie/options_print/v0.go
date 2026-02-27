package options_print

type V0 struct {
	Abbreviations *abbreviationsV0 `toml:"abbreviations"`
	boxV0

	PrintMatchedDormant *bool `toml:"print-matched-archiviert"`
	PrintShas           *bool `toml:"print-shas"`
	PrintFlush          *bool `toml:"print-flush"`
	PrintUnchanged      *bool `toml:"print-unchanged"`
	PrintColors         *bool `toml:"print-colors"`
	PrintInventoryLists *bool `toml:"print-bestandsaufnahme"`
}

type abbreviationsV0 struct {
	ZettelIds *bool `toml:"hinweisen"`
	Shas      *bool `toml:"shas"`
}

type boxV0 struct {
	PrintIncludeDescription *bool `toml:"print-include-description"`
	PrintTime               *bool `toml:"print-time"`
	PrintTagsAlways         *bool `toml:"print-etiketten-always"`
	PrintEmptyShas          *bool `toml:"print-empty-shas"`
	PrintIncludeTypes       *bool `toml:"print-include-typen"`
	PrintTai                *bool `toml:"-"`
	DescriptionInBox        *bool `toml:"-"`
	ExcludeFields           *bool `toml:"-"`
}

func (blob V0) GetAbbreviations() *OverlayAbbreviations {
	if blob.Abbreviations == nil {
		return nil
	} else {
		return &OverlayAbbreviations{
			ZettelIds: blob.Abbreviations.ZettelIds,
			MarklIds:  blob.Abbreviations.Shas,
		}
	}
}

func (blob V0) GetPrintOptionsOverlay() Overlay {
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
