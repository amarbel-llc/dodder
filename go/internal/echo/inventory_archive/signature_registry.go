package inventory_archive

import (
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type SignatureComputerParams struct {
	SignatureLen int
	AvgChunkSize int
	MinChunkSize int
	MaxChunkSize int
}

var signatureComputers = map[string]func(SignatureComputerParams) SignatureComputer{}

func RegisterSignatureComputer(
	name string,
	factory func(SignatureComputerParams) SignatureComputer,
) {
	signatureComputers[name] = factory
}

func SignatureComputerForName(
	name string,
	params SignatureComputerParams,
) (SignatureComputer, error) {
	if name == "" {
		return nil, nil
	}

	factory, ok := signatureComputers[name]
	if !ok {
		return nil, errors.Errorf("unknown signature computer type: %q", name)
	}

	return factory(params), nil
}

func init() {
	RegisterSignatureComputer(
		"gear-cdc-minhash",
		func(params SignatureComputerParams) SignatureComputer {
			return &GearCDCMinHashComputer{
				AvgChunkSize: params.AvgChunkSize,
				MinChunkSize: params.MinChunkSize,
				MaxChunkSize: params.MaxChunkSize,
				K:            params.SignatureLen,
			}
		},
	)
}
