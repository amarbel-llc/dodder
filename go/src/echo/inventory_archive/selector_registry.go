package inventory_archive

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

type BaseSelectorParams struct {
	Bands       int
	RowsPerBand int
	MinBlobSize uint64
	MaxBlobSize uint64
}

var baseSelectors = map[string]func(BaseSelectorParams) BaseSelector{}

func RegisterBaseSelector(
	name string,
	factory func(BaseSelectorParams) BaseSelector,
) {
	baseSelectors[name] = factory
}

func BaseSelectorForName(
	name string,
	params BaseSelectorParams,
) (BaseSelector, error) {
	if name == "" {
		return nil, nil
	}

	factory, ok := baseSelectors[name]
	if !ok {
		return nil, errors.Errorf("unknown base selector type: %q", name)
	}

	return factory(params), nil
}

func init() {
	RegisterBaseSelector(
		"lsh-banding",
		func(params BaseSelectorParams) BaseSelector {
			return &LSHBandingSelector{
				Bands:       params.Bands,
				RowsPerBand: params.RowsPerBand,
				MinBlobSize: params.MinBlobSize,
				MaxBlobSize: params.MaxBlobSize,
			}
		},
	)
}
