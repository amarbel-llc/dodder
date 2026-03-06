package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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

func setMultiEncryptionFlagDefinition(
	flagSet interfaces.CLIFlagDefinitions,
	keys *[]markl.Id,
) {
	flagSet.Func(
		"encryption",
		"add encryption for blobs (repeatable)",
		func(value string) (err error) {
			switch value {
			case "none":
				*keys = (*keys)[:0]
				return err
			}

			var key markl.Id

			if files.Exists(value) {
				if err = markl.SetFromPath(
					&key,
					value,
				); err != nil {
					err = errors.Wrapf(err, "Value: %q", value)
					return err
				}

				*keys = append(*keys, key)

				return err
			}

			switch value {
			case "", "generate":
				if err = key.GeneratePrivateKey(
					nil,
					markl.FormatIdAgeX25519Sec,
					markl.PurposeMadderPrivateKeyV1,
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

			default:
				if err = key.Set(value); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			*keys = append(*keys, key)

			return err
		},
	)
}
