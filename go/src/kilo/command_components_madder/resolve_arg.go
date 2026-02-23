package command_components_madder

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
)

type ResolvedArg struct {
	Arg           string
	BlobReader    domain_interfaces.BlobReader
	BlobStoreId   blob_store_id.Id
	IsStoreSwitch bool
	Err           error
}

func ResolveFileOrBlobStoreId(arg string) (resolved ResolvedArg) {
	resolved.Arg = arg

	var err error

	if resolved.BlobReader, err = env_dir.NewFileReaderOrErrNotExist(
		env_dir.DefaultConfig,
		arg,
	); errors.IsNotExist(err) {
		if err = resolved.BlobStoreId.Set(arg); err != nil {
			resolved.Err = err
			return resolved
		}

		resolved.IsStoreSwitch = true
		return resolved
	} else if err != nil {
		resolved.Err = err
		return resolved
	}

	return resolved
}
