package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

func setEncryptionFlagDefinition(
	flagSet interfaces.CLIFlagDefinitions,
	encryption *markl.Id,
) {
	flagSet.Func(
		"encryption",
		"add encryption for blobs",
		func(value string) (err error) {
			if files.Exists(value) {
				if err = markl.SetFromPath(
					encryption,
					value,
				); err != nil {
					err = errors.Wrapf(err, "Value: %q", value)
					return err
				}

				return err
			}

			switch value {
			case "none":
				// no-op

			case "", "generate":
				if err = encryption.GeneratePrivateKey(
					nil,
					markl.FormatIdAgeX25519Sec,
					markl.PurposeMadderPrivateKeyV1,
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

			default:
				if err = encryption.Set(value); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			return err
		},
	)
}
